# APX - Next-Gen API Management Platform

**AI-Native, Agentic, GCP-First API Gateway & Management Platform**


## Architecture Principles

1. **Tenant-first design**: Isolation level explicit in config, enforced at every hop
2. **Async-by-default**: Streaming & long-running work never blocks the edge
3. **Config-as-artifacts**: Versioned, canaried, rollbackable; requests carry policy version
4. **AI ≠ root**: Agents propose; guardrails validate; humans gate production
5. **Observability within budget**: Metrics (cheap) for SLOs, logs/traces (sampled) for forensics
6. **Multi-region from day 1**: Region affinity in config; residency respected everywhere

## Stack Overview

```
[Global HTTPS LB]
    ↓
[Cloud Armor] → [Edge (Cloud Run + Envoy + WASM)]
    ↓
[Router (Go)] ← reads compiled policy artifacts
    ↓
[Pub/Sub Queues] + [Redis (Memorystore)]
    ↓
[Worker Pools] (Cloud Run CPU + GKE Autopilot GPU)
    ↓
[Streaming Aggregator]
    ↓
[OTEL → Cloud Monitoring/Logging → BigQuery]
```

## Repository Structure

```
/apx
  /edge           - Envoy config, WASM filters, Docker images
  /router         - Go routing service (policy selection, request tagging)
  /workers        - CPU/GPU worker pools, examples
  /control        - Policy compiler (OPA→WASM), artifact service, control API
  /agents         - Orchestrator, Builder, Optimizer, Security, Validators
  /portal         - Next.js AgentHub (keys, usage, docs, console)
  /infra          - Terraform, IAM, networks, CI/CD, KMS
  /configs        - CRDs (Product, Route, PolicyBundle, SLO), samples, envs
  /observability  - OTEL config, dashboards, budgets, alerts
  /tools          - CLI (apx), load testing, replay tools
  /docs           - ADRs, runbooks, blueprints
```

## Quick Start

### Prerequisites
- Go 1.22+
- Node.js 20+
- Docker & Docker Compose
- GCP Project with appropriate APIs enabled
- Terraform 1.6+

### Local Development

```bash
# Initialize local environment
./tools/cli/apx init --local

# Start local stack (emulators for Pub/Sub, Firestore, Redis)
docker-compose up -d

# Compile and apply sample config
./tools/cli/apx compile configs/samples/payments-api.yaml
./tools/cli/apx apply --env dev

# Run edge gateway locally
cd edge && make run

# Run router
cd router && go run cmd/router/main.go

# Test
curl -H "Authorization: Bearer test-token" http://localhost:8080/v1/payments/health
```

### Deploy to GCP

```bash
# Set up infrastructure (first time only)
cd infra/terraform && terraform init && terraform apply

# Deploy edge
./tools/cli/apx deploy edge --env staging

# Deploy router
./tools/cli/apx deploy router --env staging

# Canary rollout policy change
./tools/cli/apx rollout --canary 5% configs/samples/payments-api-v2.yaml

# Rollback if needed
./tools/cli/apx rollback --route pay-v1
```

## Key Components

### Edge Gateway
- **Tech**: Cloud Run + Envoy + WASM filters
- **Responsibilities**: TLS termination, JWT fast-path, coarse rate limiting, correlation IDs
- **SLO**: p99 overhead ≤ 20ms @ 1k rps

### Router
- **Tech**: Go service
- **Responsibilities**: Policy selection, feature/canary flags, request tagging (policy_version)
- **SLO**: p99 latency ≤ 10ms

### Policy Compiler
- **Tech**: OPA (Rego) → WASM
- **Output**: Immutable artifacts (semver + SHA256 hash)
- **Versioning**: N/N-1 support for in-flight requests

### Worker Pools
- **CPU Pool**: Cloud Run (auto-scaling 0-1000)
- **GPU Pool**: GKE Autopilot (A100/L4)
- **Per-pool**: SLOs, surge caps, graceful degradation

### Agents
- **Orchestrator**: Central intent inbox, deduplication, rate limiting, sequencing
- **Builder**: NL → OpenAPI + policy YAML → PR
- **Optimizer**: SLO/cost guardian (auto-scale, cache, batch)
- **Security**: Config scanner, traffic analyzer, mitigation applier
- **Validators**: Schema, lint, dry-run, blast-radius checks

## Configuration Examples

### Product Definition
```yaml
apiVersion: apx/v1
kind: Product
metadata:
  name: payments
  regionAffinity: us
spec:
  plans:
    - name: pro
      rateLimit: { rps: 200, burst: 400 }
      residency: US
      isolation: namespace
```

### Route + Policy Bundle
```yaml
apiVersion: apx/v1
kind: Route
metadata:
  name: pay-v1
spec:
  match:
    host: api.example.com
    path: /v1/payments/**
    methods: [POST, GET]
  backend:
    pool: payments-cpu
    timeoutMs: 30000
  policyBundleRef: pb-pay-v1@1.2.0
---
apiVersion: apx/v1
kind: PolicyBundle
metadata:
  name: pb-pay-v1
  version: 1.2.0
  compat: backward
spec:
  auth:
    jwt:
      jwksUri: https://idp/.well-known/jwks.json
      aud: payments
  quotas:
    perTenant: { window: 60s, limit: 10000 }
  transforms:
    - wasm: redact-pii@sha256:abc123...
  observability:
    sampleRate: 0.02
    piiSafe: true
```

## Milestones

- **M0 (Week 0-1)**: Foundations - ADRs, IAM, CMEK, CI skeleton ✅
- **M1 (Weeks 1-4)**: Edge + Router + Pub/Sub + Worker + OTEL
- **M2 (Weeks 5-8)**: Policy versioning, canary rollouts, N/N-1 support
- **M3 (Weeks 9-12)**: Rate limiting (Redis), cost controls, BQ aggregates
- **M4 (Weeks 13-16)**: Agents v0 + Portal v1 + Stripe
- **M5 (Weeks 17-24)**: Multi-region, residency enforcement, WebSocket gateway

## SLOs

| Metric | Target |
|--------|--------|
| Availability (regional) | 99.95% |
| Availability (multi-region) | 99.99% |
| Edge latency (p99) | ≤ 25ms @ 2k rps |
| Streaming continuity | 99.9% uninterrupted < 5min |
| Rate limit accuracy | ±3% @ p95 |
| Observability spend | ≤ 10% of infra cost |
| Rollback time | ≤ 2 minutes |

## Contributing

See [CONTRIBUTING.md](docs/CONTRIBUTING.md)

## 🌍 Open Core Model

APX uses an **open-core model** to balance community adoption with sustainable business:

### Open Source (Apache 2.0)
The **runtime engine** is fully open-source:
- ✅ Edge gateway (Envoy + WASM)
- ✅ Router service (Go)
- ✅ Worker pools
- ✅ Configuration system (CRDs)
- ✅ CLI tools
- ✅ Tests & documentation

### Enterprise (Proprietary)
**AI agents** and advanced features are available commercially:
- 🔒 AI agents (Builder, Optimizer, Security)
- 🔒 Advanced analytics & ML models
- 🔒 Enterprise portal features
- 🔒 24/7 support & SLA guarantees

**Why this split?** The runtime should be public infrastructure - inspectable, auditable, and community-owned. The intelligence layer is our competitive advantage and enables sustainable development.

### For Developers
```bash
# Open source - full runtime capabilities
git clone https://github.com/apx-platform/apx.git
cd apx
make up
```

### For Enterprise
```bash
# Open source + enterprise features
git clone https://github.com/apx-platform/apx.git
cd apx
git clone git@github.com:apx-platform/apx-private.git .private
make up  # Now with AI agents
```

## License

- **Runtime (this repo)**: Apache 2.0
- **Agents & Enterprise**: Proprietary - All Rights Reserved
- See [LICENSE](LICENSE) for details

## Support

- Internal docs: https://docs.apx.internal
- Runbooks: [docs/runbooks/](docs/runbooks/)
- ADRs: [docs/adrs/](docs/adrs/)
