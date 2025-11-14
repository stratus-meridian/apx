# APX Router - Open Core Edition

**A high-performance API gateway with policy-based routing, rate limiting, and multi-tenancy support.**

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go)](https://go.dev/)

> **Note**: This is the **open-core edition** of APX Router. The full **commercial platform** includes:
> - Advanced quota management with billing integration (Stripe)
> - Enterprise-grade multi-tenant isolation (Firestore + Redis)
> - Real-time usage analytics and billing (BigQuery)
> - Control plane APIs for configuration management
> - Customer portal for self-service management
> - SLA guarantees and enterprise support
>
> For commercial features, visit [https://apilee.io](https://apilee.io)

---

## Features (Open Core)

- ✅ **Sync & Async Routing**: Direct HTTP proxy or Pub/Sub-based async processing
- ✅ **Policy-Based Configuration**: Kubernetes-style CRDs (Product, Route, PolicyBundle)
- ✅ **Rate Limiting**: Per-tenant token bucket rate limiting (in-memory)
- ✅ **Multi-Tenancy**: Tenant context resolution and isolation
- ✅ **Observability**: OpenTelemetry tracing, Prometheus metrics, structured logging (Zap)
- ✅ **Health Checks**: `/health` and `/ready` endpoints
- ✅ **Status Tracking**: Request status tracking for async requests (requires Redis)
- ✅ **Middleware Chain**: Composable middleware for extensibility

---

## Quick Start

### Prerequisites

- **Go 1.23+**
- **Redis** (optional, for status storage and distributed rate limiting)
- **Google Cloud Pub/Sub** (optional, for async routing)

### Run Locally (Minimal Setup)

```bash
# Clone the repository
git clone https://github.com/stratus-meridian/apx-router-open-core.git
cd apx-router-open-core

# Build the router
go build -o router ./cmd/router

# Run with a simple sync route
export ROUTES_CONFIG="/v1/hello/**=http://localhost:9000:sync"
./router

# Test (in another terminal)
curl http://localhost:8080/v1/hello/world
```

### Run with Docker

```bash
# Build Docker image
docker build -t apx-router:latest .

# Run container
docker run -p 8080:8080 \
  -e ROUTES_CONFIG="/api/**=http://host.docker.internal:9000:sync" \
  apx-router:latest
```

### Complete Example (with Backend)

See [examples/hello-world/](examples/hello-world/) for a complete working example with:
- Simple backend server (Go)
- Router configuration
- Sample requests

```bash
cd examples/hello-world
./run.sh
```

---

## Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `PORT` | HTTP server port | `8080` | No |
| `ROUTES_CONFIG` | Route definitions (see format below) | - | No |
| `REDIS_ADDR` | Redis address for status storage | - | No |
| `PUBSUB_TOPIC` | Pub/Sub topic for async routing | - | No |
| `PROJECT_ID` | Google Cloud project ID | - | No |
| `PUBLIC_URL` | Public base URL for status/stream URLs | `http://localhost:8080` | No |
| `ENVIRONMENT` | Environment name (dev/staging/prod) | `development` | No |
| `LOG_LEVEL` | Log level (debug/info/warn/error) | `info` | No |

### Route Configuration Format

The `ROUTES_CONFIG` environment variable defines routing rules:

```
ROUTES_CONFIG="path1=backend1:mode1,path2=backend2:mode2"
```

**Format**: `path=backend:mode`

- **path**: URL path pattern (supports wildcards: `/api/**`)
- **backend**: Backend URL (e.g., `http://localhost:9000`)
- **mode**: `sync` (direct proxy) or `async` (Pub/Sub)

**Examples**:

```bash
# Single sync route
ROUTES_CONFIG="/v1/chat/**=http://localhost:9000:sync"

# Multiple routes
ROUTES_CONFIG="/v1/chat/**=http://localhost:9000:sync,/v1/completions/**=http://localhost:9001:async"

# Async routing (requires Pub/Sub)
ROUTES_CONFIG="/v1/long-running/**=http://localhost:9000:async"
```

---

## CRD Schemas

The router uses Kubernetes-style **Custom Resource Definitions (CRDs)** for declarative configuration:

### Product CRD

Defines an API product with plans, quotas, isolation, and regional affinity.

**File**: [configs/crds/product.schema.yaml](configs/crds/product.schema.yaml)

```yaml
apiVersion: apx/v1
kind: Product
metadata:
  name: chat-api
  regionAffinity: [us, eu]
spec:
  plans:
    - name: free
      tier: free
      quotas:
        requests_per_minute: 10
        monthly_requests: 10000
    - name: pro
      tier: pro
      quotas:
        requests_per_minute: 100
        monthly_requests: 1000000
```

### Route CRD

Defines routing rules with path matching, backend selection, and transformations.

**File**: [configs/crds/route.schema.yaml](configs/crds/route.schema.yaml)

```yaml
apiVersion: apx/v1
kind: Route
metadata:
  name: chat-route
spec:
  path: /v1/chat/**
  backend: http://chat-service:8080
  mode: sync
  timeout: 30s
```

### PolicyBundle CRD

Defines policy versioning and canary rollouts (for WASM-based policies).

**File**: [configs/crds/policybundle.schema.yaml](configs/crds/policybundle.schema.yaml)

```yaml
apiVersion: apx/v1
kind: PolicyBundle
metadata:
  name: production-policies
spec:
  version: v1.2.0
  policies:
    - name: rate-limit
      wasm_url: gs://apx-artifacts/policies/rate-limit-v1.2.0.wasm
```

### Tier Schema

Defines subscription tiers with quotas and limits.

**File**: [configs/schemas/tier-schema.json](configs/schemas/tier-schema.json)

```json
{
  "tiers": {
    "free": {
      "requests_per_minute": 10,
      "monthly_requests": 10000,
      "burst_limit": 20,
      "max_policies": 5
    },
    "pro": {
      "requests_per_minute": 100,
      "monthly_requests": 1000000,
      "burst_limit": 200,
      "max_policies": 50
    }
  }
}
```

---

## Architecture

```
                    ┌─────────────────────────────┐
                    │      APX Router             │
                    │   (Open-Core Edition)       │
                    └──────────┬──────────────────┘
                               │
                ┌──────────────┼──────────────┐
                │              │              │
           ┌────▼────┐   ┌────▼────┐   ┌────▼────┐
           │  Sync   │   │  Async  │   │ Policy  │
           │  Proxy  │   │ Pub/Sub │   │  Store  │
           └─────────┘   └─────────┘   └─────────┘
```

### Middleware Chain

Requests flow through the following middleware layers:

1. **RequestID** - Generate unique request ID for tracing
2. **TenantContext** - Resolve tenant from API key (demo: always default tenant)
3. **RateLimit** - Enforce per-tenant rate limits (in-memory token bucket)
4. **PolicyVersionTag** - Add policy version to response headers
5. **UsageTracker** - Log usage events (demo: stdout only)
6. **Metrics** - Record Prometheus metrics
7. **Logging** - Structured logging with Zap
8. **Tracing** - Distributed tracing with OpenTelemetry

### Commercial Platform (Closed-Source)

The full platform adds:

```
┌──────────────────────────────────────────────────────┐
│                 Control Plane APIs                   │
│  (Tenant, Product, Policy, API Key Management)       │
└────────────┬─────────────────────────────────────────┘
             │
┌────────────▼─────────────────────────────────────────┐
│            Customer Portal (Next.js + Tailwind)      │
│  (Self-service: API keys, usage, billing, analytics) │
└────────────┬─────────────────────────────────────────┘
             │
┌────────────▼─────────────────────────────────────────┐
│     Billing Integration (Stripe + Usage Metering)    │
│           Analytics (BigQuery + Firestore)           │
└──────────────────────────────────────────────────────┘
```

---

## API Endpoints

### Health Check

```bash
GET /health
```

Returns health status of the router and its dependencies.

**Response**:
```json
{
  "status": "healthy",
  "edition": "open-core",
  "components": {
    "policy_store": "ready",
    "pubsub": "connected",
    "observability": "enabled"
  }
}
```

### Readiness Check

```bash
GET /ready
```

Returns readiness status (used by Kubernetes readiness probes).

### Metrics

```bash
GET /metrics
```

Prometheus metrics endpoint.

### Status Check (Async Requests)

```bash
GET /status/{request_id}
```

Check the status of an async request (requires Redis).

**Response**:
```json
{
  "request_id": "req_abc123",
  "status": "completed",
  "result": {
    "status_code": 200,
    "body": "..."
  }
}
```

---

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./internal/middleware
```

### Building

```bash
# Build for current platform
go build -o router ./cmd/router

# Build for Linux (deployment)
GOOS=linux GOARCH=amd64 go build -o router-linux-amd64 ./cmd/router

# Build with optimizations
go build -ldflags="-s -w" -o router ./cmd/router
```

### Linting

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run
```

---

## Production Deployment

### Kubernetes

See [examples/kubernetes/](examples/kubernetes/) for sample manifests.

```bash
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
kubectl apply -f ingress.yaml
```

### Google Cloud Run

```bash
# Build and push Docker image
gcloud builds submit --tag gcr.io/PROJECT_ID/apx-router

# Deploy to Cloud Run
gcloud run deploy apx-router \
  --image gcr.io/PROJECT_ID/apx-router \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated
```

### Environment Variables for Production

```bash
# Minimum required for production
export PORT=8080
export ENVIRONMENT=production
export LOG_LEVEL=info
export ROUTES_CONFIG="/api/**=http://backend:8080:sync"

# Optional: Enable async routing
export PROJECT_ID=my-gcp-project
export PUBSUB_TOPIC=apx-async-requests
export REDIS_ADDR=redis:6379
export PUBLIC_URL=https://api.example.com

# Optional: Enable observability
export OTEL_EXPORTER_OTLP_ENDPOINT=https://otel-collector:4317
```

---

## Differences: Open-Core vs Commercial

| Feature | Open-Core | Commercial |
|---------|-----------|------------|
| **Routing** | Sync + Async (Pub/Sub) | ✅ Same |
| **Rate Limiting** | In-memory token bucket | ✅ Redis-based, distributed |
| **Tenant Resolution** | Demo (header-based) | ✅ Firestore, secure API keys |
| **Usage Tracking** | Logs to stdout | ✅ BigQuery, real-time analytics |
| **Quota Enforcement** | ❌ Not included | ✅ Monthly quotas + 402 responses |
| **Billing Integration** | ❌ Not included | ✅ Stripe metering + invoicing |
| **Control Plane APIs** | ❌ Not included | ✅ Full REST/gRPC APIs |
| **Customer Portal** | ❌ Not included | ✅ Next.js self-service portal |
| **Policy Management** | Static (no-op) | ✅ WASM-based, versioned, canary rollouts |
| **Multi-Region** | Single instance | ✅ Global load balancing |
| **SLA & Support** | Community | ✅ 99.9% uptime + enterprise support |

---

## Contributing

This is the open-core edition. Contributions are welcome!

### How to Contribute

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Guidelines

- Write tests for new features
- Follow Go best practices and conventions
- Update documentation for user-facing changes
- Keep the open-core focused on routing; commercial features stay closed

---

## License

Apache 2.0 - see [LICENSE](LICENSE)

---

## Links

- **Website**: [https://apilee.io](https://apilee.io)
- **Documentation**: [https://docs.apilee.io](https://docs.apilee.io)
- **Commercial Platform**: [Contact Sales](https://apilee.io/contact)
- **Issues**: [GitHub Issues](https://github.com/stratus-meridian/apx-router-open-core/issues)

---

## Acknowledgments

Built with:
- [Go](https://go.dev/)
- [Gorilla Mux](https://github.com/gorilla/mux)
- [OpenTelemetry](https://opentelemetry.io/)
- [Prometheus](https://prometheus.io/)
- [Redis](https://redis.io/)
- [Google Cloud Pub/Sub](https://cloud.google.com/pubsub)
