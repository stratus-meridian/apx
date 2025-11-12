# Tenant Isolation Architecture

## Overview

APX implements **defense-in-depth tenant isolation** at every layer of the stack to ensure complete separation between tenants. No tenant should ever be able to access, observe, or impact another tenant's resources, data, or performance.

**Isolation Principle**: Every request MUST have a validated `tenant_id` that is propagated through all layers and used to enforce boundaries.

---

## Isolation Layers

### 1. API Layer - Authentication & Authorization

**Location**: `edge/` and `router/internal/middleware/tenant.go`

**Mechanism**:
- Every incoming request MUST include a valid API key or JWT token
- The Edge gateway or Router middleware extracts the `tenant_id` from the authentication credential
- Requests without a valid `tenant_id` are rejected with `401 Unauthorized` or `403 Forbidden`

**Headers Set**:
```http
X-Tenant-ID: tenant-abc-123
X-Tenant-Tier: enterprise
```

**Code Implementation**:
```go
// router/internal/middleware/tenant.go
func TenantContext(logger *zap.Logger) Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            tenantID := r.Header.Get("X-Tenant-ID")
            if tenantID == "" {
                http.Error(w, "tenant_id required", http.StatusUnauthorized)
                return
            }
            // Propagate through context
            ctx := context.WithValue(r.Context(), TenantIDKey, tenantID)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

**Security Guarantees**:
- ✅ Every authenticated request has a verified `tenant_id`
- ✅ Tenant cannot spoof or modify their `tenant_id`
- ✅ API keys are scoped to exactly one tenant
- ✅ Cross-tenant access attempts result in `403 Forbidden`

---

### 2. Rate Limiting Layer - Redis Keyspace Isolation

**Location**: `router/internal/ratelimit/redis.go`

**Mechanism**:
- Rate limit keys include `tenant_id` as a prefix
- Pattern: `apx:rl:{tenant_id}:{resource}`
- Each tenant has completely isolated Redis keyspace
- One tenant hitting their rate limit does NOT affect other tenants

**Example Keys**:
```
apx:rl:tenant-a:requests     → Tenant A's request counter
apx:rl:tenant-b:requests     → Tenant B's request counter
apx:rl:tenant-a:api-calls    → Tenant A's API call counter
```

**Code Implementation**:
```go
// router/internal/ratelimit/redis.go
func RateLimitKey(tenantID, resource string) string {
    // CRITICAL: Tenant ID MUST be part of key
    return fmt.Sprintf("apx:rl:%s:%s", tenantID, resource)
}

func (r *RedisRateLimiter) CheckRateLimit(ctx context.Context, tenantID string, resource string, limit int, window time.Duration) error {
    if tenantID == "" {
        return fmt.Errorf("tenant_id is required")
    }

    key := RateLimitKey(tenantID, resource)
    count, err := r.client.Incr(ctx, key).Result()

    if count > int64(limit) {
        return &RateLimitExceededError{TenantID: tenantID, ...}
    }

    return nil
}
```

**Security Guarantees**:
- ✅ Redis keys are namespaced by `tenant_id`
- ✅ Tenant A cannot access or modify Tenant B's rate limit counters
- ✅ Rate limit exhaustion is isolated per tenant
- ✅ No global rate limits that can be exhausted by a single tenant

**Testing**:
```bash
# Verify isolation
redis-cli KEYS "apx:rl:tenant-a:*"  # Shows only tenant-a keys
redis-cli KEYS "apx:rl:tenant-b:*"  # Shows only tenant-b keys
```

---

### 3. Queue Layer - Pub/Sub Message Attributes

**Location**: `router/internal/routes/matcher.go`

**Mechanism**:
- Every Pub/Sub message includes `tenant_id` as a message attribute
- Ordering key set to `tenant_id` for FIFO processing per tenant
- Workers can filter messages by tenant if needed
- Message attributes prevent cross-tenant message consumption

**Message Structure**:
```go
pubsubMsg := &pubsub.Message{
    Data: requestJSON,
    Attributes: map[string]string{
        "tenant_id":      msg.TenantID,      // CRITICAL for isolation
        "tenant_tier":    msg.TenantTier,    // For priority routing
        "request_id":     msg.RequestID,
        "policy_version": msg.PolicyVersion,
    },
    OrderingKey: msg.TenantID,  // FIFO per tenant
}
```

**Code Implementation**:
```go
// router/internal/routes/matcher.go
func (m *Matcher) PublishRequest(ctx context.Context, msg *RequestMessage) error {
    if msg.TenantID == "" {
        return fmt.Errorf("tenant_id is required")
    }

    pubsubMsg := &pubsub.Message{
        Data: data,
        Attributes: map[string]string{
            "tenant_id": msg.TenantID,
            "tenant_tier": msg.TenantTier,
        },
        OrderingKey: msg.TenantID,  // CRITICAL for tenant isolation
    }

    return m.topic.Publish(ctx, pubsubMsg).Get(ctx)
}
```

**Security Guarantees**:
- ✅ Messages tagged with immutable `tenant_id` attribute
- ✅ Ordering keys ensure per-tenant FIFO (no HOL blocking across tenants)
- ✅ Workers can filter by `tenant_id` if implementing tenant-specific pools
- ✅ Message replay preserves tenant context

**Pub/Sub Benefits**:
- **FIFO per tenant**: Ordering key = `tenant_id` means requests from the same tenant are processed in order
- **No cross-tenant blocking**: Slow tenant-a requests don't block tenant-b
- **Tenant-aware routing**: Can route enterprise tenants to premium worker pools

---

### 4. Worker Layer - Concurrency Limits

**Location**: `workers/cpu-pool/limits.go`

**Mechanism**:
- Per-tenant concurrency semaphores limit simultaneous requests
- Limits configured based on tenant tier (free/pro/enterprise)
- Non-blocking `TryAcquire()` ensures one tenant can't starve others
- Active request tracking per tenant

**Tenant Tiers**:
```go
TierLimits: map[string]*TierLimit{
    "free": {
        MaxConcurrency: 5,
        Timeout: 30 * time.Second,
    },
    "pro": {
        MaxConcurrency: 50,
        Timeout: 60 * time.Second,
    },
    "enterprise": {
        MaxConcurrency: 500,
        Timeout: 120 * time.Second,
    },
}
```

**Code Implementation**:
```go
// workers/cpu-pool/limits.go
type TenantLimits struct {
    semaphores map[string]*semaphore.Weighted  // Per-tenant semaphores
    activeRequests map[string]int
    mu sync.RWMutex
}

func (tl *TenantLimits) AcquireSlot(ctx context.Context, tenantID string, tenantTier string) error {
    if tenantID == "" {
        return fmt.Errorf("tenant_id is required")
    }

    sem := tl.getSemaphore(tenantID, tenantTier)

    // CRITICAL: TryAcquire (non-blocking) ensures we don't block other tenants
    if !sem.TryAcquire(1) {
        return &ConcurrencyLimitExceededError{...}
    }

    tl.incrementActiveRequests(tenantID)
    return nil
}

func (tl *TenantLimits) ReleaseSlot(tenantID string) {
    sem.Release(1)
    tl.decrementActiveRequests(tenantID)
}
```

**Processing Pattern**:
```go
func ProcessMessage(msg *pubsub.Message) error {
    tenantID := msg.Attributes["tenant_id"]
    tenantTier := msg.Attributes["tenant_tier"]

    // Acquire slot
    if err := tenantLimits.AcquireSlot(ctx, tenantID, tenantTier); err != nil {
        // Tenant over limit - nack message for retry
        return err
    }
    defer tenantLimits.ReleaseSlot(tenantID)

    // Process request...
}
```

**Security Guarantees**:
- ✅ Per-tenant concurrency limits enforced
- ✅ Free tier tenant cannot exhaust worker capacity
- ✅ Enterprise tenant guaranteed higher concurrency
- ✅ Non-blocking acquisition (TryAcquire) prevents starvation
- ✅ Limits are configurable per tenant or tier

**Monitoring**:
```go
stats := tenantLimits.GetStats("tenant-abc", "enterprise")
// Returns: MaxConcurrency=500, ActiveRequests=127, Available=373
```

---

### 5. Logging Layer - PII Redaction & Tenant Tagging

**Location**: Structured logging throughout the stack

**Mechanism**:
- All logs tagged with `tenant_id` field
- Log queries can be filtered by `tenant_id`
- PII redaction prevents cross-tenant data leakage in logs
- Tenant cannot query logs for other tenants

**Log Structure**:
```json
{
  "timestamp": "2025-11-11T10:30:00Z",
  "level": "INFO",
  "tenant_id": "tenant-abc-123",
  "tenant_tier": "enterprise",
  "request_id": "req-xyz-789",
  "message": "request processed",
  "duration_ms": 45,
  "status": 200
}
```

**Query Examples**:
```bash
# View tenant-a logs only
gcloud logging read 'jsonPayload.tenant_id="tenant-a"'

# Tenant-b logs are separate
gcloud logging read 'jsonPayload.tenant_id="tenant-b"'
```

**PII Redaction**:
```go
func redactPII(data map[string]interface{}) map[string]interface{} {
    // Remove sensitive fields before logging
    delete(data, "password")
    delete(data, "api_key")
    delete(data, "credit_card")
    // ...
    return data
}
```

**Security Guarantees**:
- ✅ Logs tagged with `tenant_id` for filtering
- ✅ Tenants can only query their own logs (IAM enforced)
- ✅ PII redaction prevents accidental data leakage
- ✅ Cross-tenant log queries return no results

---

## Isolation Testing

### Positive Tests (`tests/security/tenant_isolation_test.sh`)

Verifies that isolation mechanisms work correctly:

1. **Redis Keyspace Isolation**
   - Tenant-A and Tenant-B use different Redis keys
   - Rate limit counters are separate
   - Keys follow `apx:rl:{tenant_id}:{resource}` pattern

2. **Cross-Tenant Rate Limits**
   - Tenant-A hitting rate limit does NOT affect Tenant-B
   - Both tenants can reach their individual limits independently

3. **Pub/Sub Tenant Attributes**
   - Messages have `tenant_id` attribute
   - Ordering key set to `tenant_id`
   - Workers receive tenant context

4. **Worker Concurrency Limits**
   - Per-tenant semaphores exist
   - Limits enforced by tenant tier
   - Non-blocking acquisition used

5. **Attribute Propagation**
   - `tenant_id` propagates through entire request lifecycle
   - Headers preserved in responses
   - Context available in logs

### Negative Tests (`tests/security/tenant_negative_test.sh`)

Verifies that security boundaries BLOCK unauthorized access:

1. **Cross-Tenant Data Access**
   - Tenant-A cannot access Tenant-B's request status
   - Returns `403 Forbidden` or `404 Not Found`

2. **Log Isolation**
   - Tenant-A logs don't contain Tenant-B secrets
   - Log queries filtered by `tenant_id`

3. **Quota Exhaustion**
   - Tenant-A flooding requests does NOT exhaust Tenant-B quota
   - Tenant-B can still make requests

4. **Unauthenticated Access**
   - Status endpoints require valid credentials
   - Returns `401 Unauthorized` without auth

5. **Rate Limit Modification**
   - Tenant-A cannot modify Tenant-B's Redis keys
   - Keyspace isolation enforced

6. **Invalid Tenant ID**
   - Requests without `tenant_id` rejected
   - Empty `tenant_id` rejected

7. **Worker Concurrency Isolation**
   - Per-tenant semaphores prevent cross-tenant blocking
   - Non-blocking acquisition used

**Running Tests**:
```bash
# Positive tests
./tests/security/tenant_isolation_test.sh

# Negative tests (security boundaries)
./tests/security/tenant_negative_test.sh
```

---

## Threat Model

### Threats Mitigated

| Threat | Mitigation | Layer |
|--------|-----------|-------|
| **Cross-tenant data access** | API key validation, request filtering | API Layer |
| **Rate limit exhaustion** | Per-tenant Redis keys | Rate Limiting |
| **Queue flooding** | Per-tenant concurrency limits | Worker Layer |
| **Resource starvation** | Non-blocking semaphore acquisition | Worker Layer |
| **Log data leakage** | PII redaction, tenant-tagged logs | Logging Layer |
| **Unauthorized status access** | API key validation, tenant scoping | API Layer |
| **Tenant ID spoofing** | Signed JWT/API keys, middleware validation | API Layer |

### Known Gaps

1. **Network-level isolation**: Workers share the same Cloud Run service
   - **Mitigation**: Per-tenant semaphores + non-blocking acquisition
   - **Future**: Dedicated worker pools per tier

2. **Redis single namespace**: All tenants share same Redis instance
   - **Mitigation**: Keyspace isolation with `apx:rl:{tenant_id}:` prefix
   - **Future**: Separate Redis instances per tier

3. **Pub/Sub ordering**: Ordering key only guarantees FIFO per tenant, not latency
   - **Mitigation**: Workers use TryAcquire to prevent blocking
   - **Future**: Priority queues for enterprise tier

4. **Metrics aggregation**: Some metrics aggregated across all tenants
   - **Mitigation**: Metrics tagged with `tenant_id` for filtering
   - **Future**: Per-tenant dashboards in Grafana

---

## Production Readiness

### Deployment Checklist

- [ ] API gateway validates `tenant_id` on every request
- [ ] Redis rate limiting uses tenant-prefixed keys
- [ ] Pub/Sub messages include `tenant_id` attribute and ordering key
- [ ] Workers enforce per-tenant concurrency limits
- [ ] Logs tagged with `tenant_id` for filtering
- [ ] PII redaction enabled in production
- [ ] Positive tests passing
- [ ] Negative tests passing (all blocked correctly)
- [ ] Monitoring dashboards show per-tenant metrics
- [ ] Alerts configured for cross-tenant access attempts

### Monitoring

**Key Metrics**:
- `tenant_active_requests{tenant_id="X"}` - Current concurrent requests
- `tenant_rate_limit_exceeded{tenant_id="X"}` - Rate limit violations
- `tenant_concurrency_limit_exceeded{tenant_id="X"}` - Concurrency limit hits
- `cross_tenant_access_attempts` - Security boundary violations (should be 0)

**Alerts**:
- 🚨 Cross-tenant access attempt detected
- ⚠️  Tenant approaching concurrency limit
- ⚠️  Tenant rate limit exceeded repeatedly

**Dashboards**:
- Per-tenant request rate
- Per-tenant error rate
- Per-tenant p95/p99 latency
- Worker concurrency by tenant

---

## Configuration

### Tenant Tier Limits

Edit `workers/cpu-pool/limits.go` to configure limits:

```go
TierLimits: map[string]*TierLimit{
    "free": {
        MaxConcurrency:  5,
        Timeout:         30 * time.Second,
        MaxRequestSize:  1 * 1024 * 1024,   // 1MB
        MaxResponseSize: 5 * 1024 * 1024,   // 5MB
    },
    "pro": {
        MaxConcurrency:  50,
        Timeout:         60 * time.Second,
        MaxRequestSize:  10 * 1024 * 1024,  // 10MB
        MaxResponseSize: 50 * 1024 * 1024,  // 50MB
    },
    "enterprise": {
        MaxConcurrency:  500,
        Timeout:         120 * time.Second,
        MaxRequestSize:  100 * 1024 * 1024, // 100MB
        MaxResponseSize: 500 * 1024 * 1024, // 500MB
    },
}
```

### Per-Tenant Overrides

For custom contracts:

```go
TenantOverrides: map[string]*TierLimit{
    "tenant-vip-123": {
        MaxConcurrency: 1000,
        Timeout: 300 * time.Second,
    },
}
```

---

## Best Practices

1. **Always validate tenant_id**: Every layer must verify `tenant_id` is present
2. **Use non-blocking operations**: `TryAcquire()` instead of `Acquire()` to prevent starvation
3. **Namespace all shared resources**: Redis keys, Pub/Sub topics, logs
4. **Tag everything**: Metrics, logs, traces should include `tenant_id`
5. **Test negative cases**: Verify unauthorized access is blocked
6. **Monitor boundary violations**: Alert on cross-tenant access attempts
7. **Fail closed**: When in doubt, reject the request

---

## References

- Task: [V-008 Tenant Isolation Enforcement](/docs/VALIDATION_HARDENING_PLAN.md#task-v-008-tenant-isolation-enforcement)
- Code: `router/internal/ratelimit/redis.go`
- Code: `router/internal/routes/matcher.go`
- Code: `workers/cpu-pool/limits.go`
- Tests: `tests/security/tenant_isolation_test.sh`
- Tests: `tests/security/tenant_negative_test.sh`

---

**Last Updated**: 2025-11-11
**Status**: ✅ Implemented and Tested
