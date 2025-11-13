# Multi-Tenant Architecture Plan for APX Platform

**Status:** Planning Phase (Post-Phase 2)
**Target:** Phase 3+ Implementation
**Last Updated:** 2025-11-13

---

## Executive Summary

APX will implement a three-tier multi-tenant SaaS model with progressive isolation levels:
- **Free Tier:** Shared infrastructure (multi-tenant pool)
- **Pro Tier:** Soft isolation (namespaces)
- **Enterprise Tier:** Dedicated runtime (isolated Cloud Run services or GKE namespaces)

This enables cost-efficient scaling while providing upgrade paths for security-conscious and high-volume customers.

---

## 1. Business Tiering Model

| Tier | Target | Key Limits | Infra Mode | Cost |
|------|--------|------------|------------|------|
| **Free / Developer** | Hobbyists, test integrations | 10K reqs/mo, 100 RPS max, no custom domain | Shared multi-tenant pool | $0 |
| **Pro / Team** | Startups & small SaaS | 1M reqs/mo, custom domain, basic SLA | Soft-isolated tenants (namespaces) | $49-499/mo |
| **Enterprise / Dedicated** | Large orgs, compliance needs | Unlimited, custom SLO/SLA, mTLS, residency, 99.9% uptime | Dedicated tenant runtime | Custom |

### Why This Model?
- **Cost control:** One set of services for thousands of free users
- **Fast onboarding:** Instant signup, no infrastructure provisioning
- **Upgrade path:** Simple Terraform variable switch
- **Security:** Tenant-aware policies with minimal blast radius
- **SLO clarity:** Free = best effort, paid = formal SLOs

---

## 2. Logical Multi-Tenancy Model

### Isolation Mechanisms

| Layer | Technique |
|-------|-----------|
| **Auth / Tenant ID** | Every request carries `X-Tenant-ID`, validated via signed JWT with `tenant_id` claim |
| **Rate Limiting** | Per-tenant Redis bucket (namespace key: `tenant:{id}:rate`) |
| **Queueing** | Pub/Sub attributes include `tenant_id`; workers enforce quota |
| **Config / Policy** | Per-tenant policy bundles; free tenants cannot upload WASM or custom transforms |
| **Logging / Analytics** | `tenant_id` label on all OTEL traces; BigQuery partitioned by `tenant_tier` |
| **Billing / Ledger** | Shared ledger table keyed by `tenant_id` |
| **Data Residency** | Optional region flag per tenant (for future regional pools) |

---

## 3. Infrastructure Layout

### Shared Pool (Free Tier)
```
Cloud Load Balancer
   ↓
Shared Cloud Run Service (Edge/Router)
   ↓
Pub/Sub topics (partitioned by tenant_tier)
   ↓
Shared Worker Pool (CPU workers)
   ↓
Firestore / BigQuery / Redis (tenant_id scoped)
```

### Paid Tiers
- **Pro:** Own namespace (isolated quotas + billing)
- **Enterprise:** Dedicated Cloud Run service or GKE pool
- **Migration:** Simple Terraform variable switch (`tenant.isolation = dedicated`)

---

## 4. Data Schema

### 4.1 Firestore Tenant Document

**Collection:** `tenants/{tenant_id}`

```yaml
tenant_id: string                # "acme-prod"
org_name: string
tier: "free" | "pro" | "enterprise"
isolation_mode: "shared" | "namespace" | "dedicated"
region_affinity: "us" | "eu" | "ap"
residency: "US" | "EU" | "APAC" | "GLOBAL"
status: "active" | "suspended"
created_at: timestamp
updated_at: timestamp

quotas:
  monthly_requests: int          # e.g., 10,000 for free
  rps: int                       # e.g., 25 (free), 200 (pro)
  burst: int
  async_jobs_daily: int

billing:
  plan_id: "free" | "pro-1m" | "custom-xyz"
  balance_usd: number
  credits_usd: number
  currency: "USD" | "USDC"
  overage_allowed: bool
  x402_enabled: bool             # HTTP 402 Payment Required

keys[]:
  key_id: string
  prefix: "apx_live_" | "apx_test_"
  scopes: ["product:payments", "env:dev"]
  ttl: timestamp | null
  status: "active" | "revoked"
  last_used_at: timestamp
```

### 4.2 Redis Keyspace

```
# Per-tenant rate limits
rl:tenant:{tenant_id}:rps            -> counter / ttl=1s
rl:tenant:{tenant_id}:burst          -> token-bucket state

# Per-tenant monthly quota
quota:tenant:{tenant_id}:month:{YYYYMM} -> consumed_count

# Idempotency for POST
idem:{tenant_id}:{hash(request)}     -> seen timestamp (ttl=24h)

# Abuse detection / temp bans
ban:{tenant_id}                      -> ttl if active
```

### 4.3 Pub/Sub Topics

```yaml
topics:
  - apx-jobs-free
  - apx-jobs-pro
  - apx-jobs-ent

message.attributes:
  tenant_id: "acme-prod"
  tier: "free" | "pro" | "enterprise"
  residency: "EU"
  policy_version: "1.3.0"
```

### 4.4 OTEL Labels (Low Cardinality)

```yaml
# Use these in metrics (low cardinality)
tenant_tier: "free" | "pro" | "enterprise"
route_pattern: "/v1/payments/*"     # normalized
region: "us" | "eu" | "ap"

# Avoid tenant_id in metrics (high cardinality)
# Use tenant_id only in traces and logs
```

---

## 5. Terraform Design

### 5.1 Variables

```hcl
# infra/variables.tf
variable "tenant_id"      { type = string }
variable "tenant_tier"    { type = string }  # free | pro | enterprise
variable "isolation_mode" { type = string }  # shared | namespace | dedicated
variable "region"         { type = string }
variable "runtime"        { type = string }  # cloudrun | gke

variable "custom_limits" {
  type = object({
    rps    = number
    burst  = number
    month  = number
  })
  default = null
}
```

### 5.2 Runtime Dispatcher

```hcl
# infra/main.tf
locals {
  # Enterprise + dedicated → GKE; else Cloud Run
  effective_runtime = var.isolation_mode == "dedicated" ? "gke" : var.runtime
}

module "apx_runtime" {
  source         = "./modules/runtime"
  tenant_id      = var.tenant_id
  tenant_tier    = var.tenant_tier
  isolation_mode = var.isolation_mode
  region         = var.region
  runtime        = local.effective_runtime
  custom_limits  = var.custom_limits
}
```

### 5.3 Isolation Mapping

| Tier | Isolation Mode | Cloud Run | GKE |
|------|----------------|-----------|-----|
| Free | `shared` | One service, tenant-aware | N/A |
| Pro | `namespace` | Separate service/revision | Namespace + HPA |
| Enterprise | `dedicated` | Dedicated service + NEG + custom domain | Dedicated namespace + nodepool |

### 5.4 Example tfvars

**Free Tier:**
```hcl
tenant_id      = "demo-free"
tenant_tier    = "free"
isolation_mode = "shared"
runtime        = "cloudrun"
region         = "us-central1"
```

**Pro Tier:**
```hcl
tenant_id      = "acme-pro"
tenant_tier    = "pro"
isolation_mode = "namespace"
runtime        = "cloudrun"
region         = "us-central1"
```

**Enterprise Tier:**
```hcl
tenant_id      = "bigcorp-ent"
tenant_tier    = "enterprise"
isolation_mode = "dedicated"  # Auto-selects GKE
runtime        = "cloudrun"   # Ignored by dispatcher
region         = "us-central1"
```

---

## 6. Implementation Phases

### Phase 3.1: Multi-Tenant Foundation (Week 9-10)
- [ ] Tenant schema in Firestore
- [ ] Tenant CRUD API endpoints
- [ ] Router middleware for tenant context
- [ ] Redis rate limiting per tenant
- [ ] Monthly quota tracking

### Phase 3.2: Tier Enforcement (Week 11)
- [ ] Tier-based quota enforcement
- [ ] Pub/Sub topic routing by tier
- [ ] Analytics rollups by tier
- [ ] Portal: Tenant admin page

### Phase 3.3: Isolation & Upgrade Path (Week 12)
- [ ] Terraform runtime dispatcher
- [ ] Pro tier namespace isolation
- [ ] Enterprise dedicated deployment
- [ ] Upgrade wizard in portal

### Phase 3.4: Monetization (Week 13-14)
- [ ] HTTP 402 Payment Required flow
- [ ] Stripe integration
- [ ] Usage-based billing
- [ ] Balance management API

---

## 7. Router Middleware (Pseudo-code)

```go
func TenantMiddleware(rl *RateLimiter, tenants TenantStore) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        fn := func(w http.ResponseWriter, r *http.Request) {
            // 1. Extract API key from Authorization header
            apiKey := extractBearerToken(r)

            // 2. Lookup tenant
            tenant, err := tenants.LookupByKey(ctx, apiKey)
            if err != nil {
                http.Error(w, "invalid api key", http.StatusUnauthorized)
                return
            }

            // 3. Check rate limit (per-tenant)
            ok := rl.allowRPS(ctx, tenant.ID, tenant.Quotas.RPS)
            if !ok {
                http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
                return
            }

            // 4. Check monthly quota
            total := tenants.IncMonthly(ctx, tenant.ID, yyyymm, 1)
            if total > tenant.Quotas.MonthlyRequests {
                http.Error(w, "monthly quota exceeded", http.StatusTooManyRequests)
                return
            }

            // 5. Check balance (if monetization enabled)
            if tenant.Billing.X402Enabled {
                if err := ledger.Charge(ctx, tenant.ID, costPerReq); err != nil {
                    http.Error(w, "payment required", http.StatusPaymentRequired)
                    return
                }
            }

            // 6. Add tenant context to request
            ctx = context.WithValue(ctx, "tenant", tenant)
            r.Header.Set("x-apx-tenant", tenant.ID)
            r.Header.Set("x-apx-tenant-tier", tenant.Tier)

            next.ServeHTTP(w, r.WithContext(ctx))
        }
        return http.HandlerFunc(fn)
    }
}
```

---

## 8. Testing Strategy

### Unit Tests
- [ ] Tenant CRUD operations
- [ ] Rate limiter per-tenant isolation
- [ ] Quota enforcement logic
- [ ] Billing charge calculations

### Integration Tests
- [ ] Free tier: shared infra, noisy neighbor protection
- [ ] Pro tier: namespace isolation, independent quotas
- [ ] Enterprise: dedicated runtime, custom limits
- [ ] Upgrade path: free → pro → enterprise (no downtime)

### Load Tests (k6)
- [ ] 1000 free tenants, concurrent requests
- [ ] Per-tenant rate limits enforced
- [ ] No cross-tenant interference
- [ ] Upgrade to pro maintains throughput

---

## 9. Migration Plan

1. **Default all existing tenants to free/shared**
2. **Add router middleware** (log-only mode for 24h)
3. **Enable enforcement** (rate limits + quotas)
4. **Test upgrade:** One tenant to pro/namespace
5. **Test dedicated:** One tenant to enterprise/dedicated
6. **Backfill analytics** with tenant_tier tags
7. **Launch monetization** (Stripe integration)

---

## 10. Success Criteria

- [ ] Free tenants share infrastructure efficiently
- [ ] Noisy neighbor attacks fail (proven via tests)
- [ ] Upgrade to pro creates isolated service/namespace
- [ ] Enterprise dedicated runtime deployed same artifacts
- [ ] Analytics rollups visible by tier
- [ ] Portal loads tier controls in <200ms
- [ ] HTTP 402 flow works end-to-end
- [ ] Docs updated (RUNBOOK_TIERING.md)

---

## 11. Cost Analysis

### Shared Pool (Free Tier)
- **Target:** 10,000 free tenants
- **Requests:** 100M/month total (10K each)
- **Cost:** ~$150/month (Cloud Run + Redis + Firestore)
- **Per tenant:** $0.015/month

### Pro Tier (Namespace)
- **Target:** 100 pro tenants
- **Requests:** 100M/month total (1M each)
- **Cost:** ~$500/month + $2/tenant
- **Per tenant:** $7/month
- **Revenue:** $49/tenant = $4,900/month
- **Margin:** 90%

### Enterprise (Dedicated)
- **Target:** 10 enterprise tenants
- **Requests:** Variable, SLA-backed
- **Cost:** $200/tenant/month (dedicated infra)
- **Revenue:** $2,000/tenant (custom)
- **Margin:** 90%

---

## 12. References

### JSON Schema
See: `/Users/agentsy/APILEE/docs/architecture/schemas/tenant.schema.json`

### Go Middleware
See: `/Users/agentsy/APILEE/docs/architecture/examples/tenant_middleware.go`

### Terraform Examples
See: `/Users/agentsy/APILEE/docs/architecture/terraform/`

---

## 13. Next Steps

**After Phase 2 completion:**
1. Review this architecture with team
2. Prioritize Phase 3.1 tasks
3. Create detailed implementation plan
4. Assign to agent workstreams
5. Begin tenant schema implementation

---

**Document Owner:** Architecture Team
**Last Review:** 2025-11-13
**Status:** Planning (Post-Phase 2)
