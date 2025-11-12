# Proxying Apigee Mock Target via APX

**Backend:** https://mocktarget.apigee.net/
**Goal:** Proxy requests through APX with observability, rate limiting, and policy enforcement

---

## 🎯 What This Demonstrates

This configuration shows how to use APX to proxy an external backend (Apigee's mock target) with:

- ✅ Request/Response logging
- ✅ Distributed tracing (OTEL)
- ✅ Rate limiting
- ✅ Policy enforcement (OPA-ready)
- ✅ Metrics collection
- ✅ Async processing via Pub/Sub
- ✅ Circuit breaker protection

---

## 📁 Files Created

1. **Configuration:** `/Users/agentsy/APILEE/configs/samples/apigee-mock-proxy.yaml`
   - Product definition
   - PolicyBundle (allow-all for testing)
   - Route configuration

2. **Test Script:** `/Users/agentsy/APILEE/tests/manual/test_apigee_proxy.sh`
   - Automated testing script
   - Tests multiple endpoints

---

## 🚀 Quick Start

### Option 1: Using Local APX Stack

```bash
cd /Users/agentsy/APILEE

# 1. Start local stack (if not already running)
make up

# 2. Verify services are running
make status

# 3. Test direct backend access (verify mocktarget is up)
curl https://mocktarget.apigee.net/

# 4. Test via APX Router
curl http://localhost:8081/mock/ \
  -H "X-Tenant-ID: test-tenant" \
  -H "X-Request-ID: test-123"

# 5. Run automated tests
chmod +x tests/manual/test_apigee_proxy.sh
./tests/manual/test_apigee_proxy.sh
```

---

### Option 2: Using Cloud Run Deployment

```bash
# Set your Cloud Run router URL
export ROUTER_URL="https://apx-router-xxxxx-uc.a.run.app"

# Test via Cloud Run
curl $ROUTER_URL/mock/ \
  -H "X-Tenant-ID: test-tenant"

# Run tests against Cloud Run
ROUTER_URL=$ROUTER_URL ./tests/manual/test_apigee_proxy.sh
```

---

## 📊 Current APX Architecture (Async-First)

APX currently uses an **async-first architecture**:

```
┌──────────┐      ┌──────────┐      ┌─────────┐      ┌─────────┐
│  Client  │─────▶│  Router  │─────▶│ Pub/Sub │─────▶│ Workers │
└──────────┘      └──────────┘      └─────────┘      └─────────┘
     ▲                  │                                    │
     │                  │                                    │
     │             202 Accepted                              │
     │              + request_id                             │
     │                                                        │
     │                                                        ▼
     │                                                  ┌──────────┐
     │                                                  │ Backend  │
     │                                                  │ (Apigee) │
     │                                                  └──────────┘
     │                                                        │
     │                                                        │
     └───────────── Poll /status/{request_id} ◄──────────────┘
```

### Request Flow

1. **Client → Router:**
   - Client sends request to `/mock/*`
   - Router adds request ID, tenant context
   - Router publishes to Pub/Sub
   - **Returns 202 Accepted immediately**

2. **Router → Client:**
   ```json
   {
     "status": "accepted",
     "request_id": "req-abc123",
     "status_url": "http://localhost:8081/status/req-abc123",
     "stream_url": "http://localhost:8081/stream/req-abc123"
   }
   ```

3. **Workers → Backend:**
   - Workers pull from Pub/Sub
   - Forward request to `https://mocktarget.apigee.net`
   - Get response from backend

4. **Workers → Status Store:**
   - Workers update Redis with response
   - Status: `pending` → `processing` → `completed`

5. **Client → Status Endpoint:**
   - Client polls `/status/{request_id}`
   - Gets final result when complete

---

## 🔧 Implementation Notes

### Current State

The APX router is **async-first** by design (see router/cmd/router/main.go:174-187):

```go
r.PathPrefix("/").Handler(
    middleware.Chain(
        http.HandlerFunc(routeMatcher.Handle),  // Publishes to Pub/Sub
        middleware.RequestID(logger),
        middleware.TenantContext(logger),
        middleware.RateLimit(rateLimiter, logger),
        middleware.PolicyVersionTag(policyStore, logger),
        middleware.Metrics(),
        middleware.Logging(logger),
        middleware.Tracing(),
    ),
)
```

### What Needs to Be Implemented

To support **synchronous proxying** (optional), we would need to add:

1. **Sync/Async Mode Selection** (per route configuration)
2. **HTTP Client** in router for direct proxying
3. **Backend Pool Management** (connection pooling, retries)
4. **Streaming Response** (for large responses)

**Note:** The async-first approach is intentional and provides:
- Better scalability (decoupled frontend/backend)
- Resilience (queue acts as buffer)
- Long-running request support
- Automatic retry with Pub/Sub

---

## 🧪 Testing the Proxy

### 1. Health Check

```bash
# Verify router is running
curl http://localhost:8081/health

# Expected response:
{"status":"ok","service":"apx-router"}
```

### 2. Direct Backend Test

```bash
# Test backend is accessible
curl https://mocktarget.apigee.net/

# Should return HTML page
```

### 3. Proxy Test (Async)

```bash
# Send request via APX
curl -X GET http://localhost:8081/mock/ \
  -H "X-Tenant-ID: test-tenant" \
  -H "X-Request-ID: test-$(date +%s)"

# Expected: 202 Accepted with request_id
{
  "status": "accepted",
  "request_id": "req-1699876543-abc123",
  "status_url": "http://localhost:8081/status/req-1699876543-abc123"
}
```

### 4. Poll for Result

```bash
# Get request ID from previous response
REQUEST_ID="req-1699876543-abc123"

# Poll status endpoint
curl http://localhost:8081/status/$REQUEST_ID

# When complete:
{
  "status": "completed",
  "request_id": "req-1699876543-abc123",
  "response": {
    "status_code": 200,
    "headers": {...},
    "body": "..."
  },
  "completed_at": "2025-11-12T20:15:43Z"
}
```

---

## 📈 Observability

### Logs

```bash
# Router logs
docker logs apx-router -f

# Worker logs
docker logs apx-workers -f

# Look for:
# - Request ID propagation
# - Pub/Sub publish events
# - Backend requests
# - Response updates
```

### Traces

```bash
# View traces in Cloud Trace (if using GCP)
gcloud logging read "resource.type=cloud_run_revision" \
  --project=apx-build-478003 \
  --limit=50

# Look for trace IDs matching request IDs
```

### Metrics

```bash
# Prometheus metrics endpoint
curl http://localhost:8081/metrics

# Key metrics:
# - apx_requests_total
# - apx_request_duration_seconds
# - apx_pubsub_publishes_total
# - apx_worker_jobs_total
```

---

## 🎯 Available Apigee Mock Endpoints

Test these endpoints through APX:

```bash
# Root / Help
curl http://localhost:8081/mock/
curl http://localhost:8081/mock/help

# JSON responses
curl http://localhost:8081/mock/json
curl http://localhost:8081/mock/user

# XML responses
curl http://localhost:8081/mock/xml

# Status codes
curl http://localhost:8081/mock/statuscode/200
curl http://localhost:8081/mock/statuscode/404
curl http://localhost:8081/mock/statuscode/500

# Echo (returns request)
curl -X POST http://localhost:8081/mock/echo \
  -H "Content-Type: application/json" \
  -d '{"test": "data"}'

# Headers
curl http://localhost:8081/mock/headers

# IP address
curl http://localhost:8081/mock/ip

# Delay (simulates slow backend)
curl http://localhost:8081/mock/delay/3  # 3 second delay
```

---

## 🔍 Troubleshooting

### Issue: 502 Bad Gateway

**Cause:** Router can't reach backend

**Solution:**
```bash
# Test backend directly
curl https://mocktarget.apigee.net/

# Check router logs
docker logs apx-router -f

# Verify network connectivity
```

### Issue: 503 Service Unavailable

**Cause:** Pub/Sub or Workers not available

**Solution:**
```bash
# Check Pub/Sub topic exists
gcloud pubsub topics list

# Check workers are running
docker ps | grep apx-workers

# Check worker logs
docker logs apx-workers -f
```

### Issue: Status endpoint returns 404

**Cause:** Request not processed yet or Redis not available

**Solution:**
```bash
# Check Redis is running
docker ps | grep redis

# Test Redis connection
redis-cli ping

# Check status store
docker logs apx-router | grep "status"
```

---

## 🚀 Next Steps

### 1. Add Synchronous Mode (Optional)

If you need synchronous responses, we can add:

```yaml
# In route configuration
spec:
  match:
    path: /mock/**
  backend:
    upstream: https://mocktarget.apigee.net
    mode: sync  # NEW: sync vs async
```

Implementation would add HTTP client to router for direct proxying.

### 2. Add Custom Policies

Replace allow-all policy with real authorization:

```yaml
authorization:
  rego: |
    package apx.authz

    allow {
      input.headers["x-api-key"][_] == "secret-key"
      input.method == "GET"
    }
```

### 3. Add Rate Limiting Per Endpoint

```yaml
rateLimit:
  perEndpoint:
    "/mock/json": 100  # 100 rps
    "/mock/xml": 50    # 50 rps
    "/mock/delay/*": 10  # 10 rps
```

### 4. Add Response Transformation

```yaml
transforms:
  - type: response
    wasm: transform-response@1.0.0
    config:
      removeHeaders: ["Server", "X-Powered-By"]
      addHeaders:
        X-Proxied-By: "APX"
```

---

## 📚 Related Documentation

- **APX Architecture:** `/Users/agentsy/APILEE/.private/docs/PRINCIPLES.md`
- **Router Configuration:** `/Users/agentsy/APILEE/router/internal/config/config.go`
- **Sample Configs:** `/Users/agentsy/APILEE/configs/samples/`
- **Integration Tests:** `/Users/agentsy/APILEE/tests/integration/`

---

## ✅ Summary

**Created:**
- ✅ Product configuration (`apigee-mock`)
- ✅ PolicyBundle (allow-all for testing)
- ✅ Route configuration (proxy to mocktarget.apigee.net)
- ✅ Test script with automated tests
- ✅ This documentation

**How to Use:**
1. Start APX stack: `make up`
2. Test: `./tests/manual/test_apigee_proxy.sh`
3. Monitor: Check logs, traces, metrics

**Current Behavior:**
- Async processing via Pub/Sub
- Returns 202 Accepted immediately
- Poll `/status/{request_id}` for result
- Full observability with OTEL

**Future Enhancement:**
- Add synchronous mode if needed
- Add custom policies
- Add per-endpoint rate limits
- Add response transformations

---

**Questions?** Check the APX documentation or create an issue in the repo!
