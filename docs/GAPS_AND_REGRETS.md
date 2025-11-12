# Critical Gaps & Future Regrets - How We Prevented Them

**Last Updated:** 2025-11-11
**Status:** All gaps addressed in foundation

This document catalogs the "future regrets" we identified during architecture review and how each one is baked into the APX platform from day 1.

---

## 1. Multi-Tenancy Isolation ✅ ADDRESSED

### The Risk
Cross-tenant data leakage through:
- Shared GPU memory in model caches
- Queue ordering revealing other tenants' requests
- PII bleeding across tenant logs
- Billing/metering collisions

### How We Prevented It

#### In Foundation
- **Product CRD** includes explicit `isolation: shared|namespace|dedicated` per plan ([product.schema.yaml](../configs/crds/product.schema.yaml:88))
- **Tenant context** propagated via headers + OTEL baggage at every hop ([middleware/tenant.go](../router/internal/middleware/tenant.go))
- **Pub/Sub messages** tagged with `tenant_id` attribute for filtering ([IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md#week-2-async-queue--workers))

#### In M1-M2 (Weeks 1-8)
- Separate Pub/Sub topics for `isolation: namespace` tenants
- Separate worker pools for `isolation: dedicated` tenants
- CMEK (Cloud KMS) per tenant for residency enforcement
- PII redaction in logs via WASM filter + structured logging

#### Test Coverage
- Integration test: Tenant A's requests never visible in Tenant B's logs
- Load test: Concurrent requests from 100 tenants, zero cross-contamination
- Security test: Attempt to inject tenant_id header → rejected at edge

---

## 2. Streaming & Long-Running Requests ✅ ADDRESSED

### The Risk
- Cloud Run 60-min timeout, most LBs timeout at 30-60s
- GPT-4 responses taking 90+ seconds
- Agentic workflows requiring multi-step reasoning (minutes)
- Streaming responses cutting off mid-sentence

### How We Prevented It

#### In Foundation
- **Async-by-default** principle ([PRINCIPLES.md](./PRINCIPLES.md#2-async-by-default))
- **202 Accepted** response pattern designed into router ([IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md#week-2-async-queue--workers))

#### In M1 (Weeks 1-4)
- Request → Router → Pub/Sub → Worker → Callback/Stream
- SSE (Server-Sent Events) for streaming responses
- Status polling endpoint: `/status/{request_id}`

#### In M5 (Weeks 21-22)
- **WebSocket gateway on GKE** for sessions >5 minutes
- Resume tokens: if connection drops, client reconnects with last position
- Backend timeout budget: edge (10s) → router (30s) → worker (600s+)

#### Test Coverage
- Test: 2-minute inference request completes successfully
- Test: WebSocket session lasts 30 minutes without drops
- Test: Client disconnects at 50% response → resumes via token

---

## 3. Cost Explosion (Logging/BigQuery) ✅ ADDRESSED

### The Risk
- "Log everything" → $50k/month in Cloud Logging ingestion
- BigQuery scans grow linearly with request volume
- No sampling, retention policies, or partition pruning

### How We Prevented It

#### In Foundation
- **Observability within budget** principle ([PRINCIPLES.md](./PRINCIPLES.md#5-observability-within-budget))
- **Log sample rate** configured per Product plan ([product.schema.yaml](../configs/crds/product.schema.yaml:139))
- **PolicyBundle observability** spec includes `sampleRate`, `piiSafe`, `piiFields` ([policybundle.schema.yaml](../configs/crds/policybundle.schema.yaml:257))

#### In M1 (Week 3)
- Edge: 1% sampling for success requests, 100% for errors
- Structured JSON logs with low-cardinality labels
- OTEL tail-based sampling: keep errors + p99 latency outliers

#### In M3 (Week 11)
- BigQuery tables: partitioned by date, clustered by tenant_id
- Hourly aggregates materialized (Cloud Function or Dataflow)
- Raw logs expire after 7 days, aggregates after 2 years
- Budget alerts: observability spend >7% of infra → alarm

#### Cost Targets
| Phase | Daily Request Volume | BigQuery Cost | Logging Cost | Total Observability |
|-------|---------------------|---------------|--------------|---------------------|
| M1 | 100k | <$1 | <$5 | <$10 |
| M3 | 1M | <$5 | <$20 | <$30 |
| M5 | 10M | <$50 | <$100 | <$200 |

---

## 4. Policy Versioning & Rollback ✅ ADDRESSED

### The Risk
- Policy change breaks 20% of traffic (new auth rule rejects valid tokens)
- No clean rollback path
- 10k in-flight requests stuck with broken policy
- Agents auto-apply conflicting policy changes

### How We Prevented It

#### In Foundation
- **Config-as-artifacts** principle ([PRINCIPLES.md](./PRINCIPLES.md#3-config-as-artifacts))
- **PolicyBundle versioning**: semver + `compat: backward|breaking` ([policybundle.schema.yaml](../configs/crds/policybundle.schema.yaml:28))
- **Route canary** support built into schema ([route.schema.yaml](../configs/crds/route.schema.yaml:124))

#### In M2 (Weeks 5-8)
- Policy compiler: YAML → OPA/Rego → WASM → immutable artifact (SHA256 hash)
- Router tags requests with `x-apx-policy-version: 1.2.0`
- Workers support N and N-1 versions for 24h rollout window
- Auto-rollback: if canary error_rate > stable + 5% → revert in <2 minutes

#### GitOps Flow
```
PR merged → Cloud Build compiles → artifact stored in GCS
  → Firestore metadata updated → workers load new version
  → canary 1% → 5% → 25% → 100% over 2 hours
  → monitor error rate → auto-rollback if spike
```

#### Test Coverage
- Test: Deploy policy v1.3.0, in-flight v1.2.0 requests unaffected
- Test: Introduce auth bug in v1.3.0 → auto-rollback in 90 seconds
- Test: Rollback completes without dropping requests

---

## 5. Rate Limiting Accuracy ✅ ADDRESSED

### The Risk
- Distributed rate limiting is "approximate"
- Token bucket at edge vs. queue backpressure vs. worker concurrency = 3 limits that drift
- Free-tier users using 10× their quota
- Premium users incorrectly throttled during bursts

### How We Prevented It

#### In Foundation
- **Product CRD** includes `rateLimit: {rps, burst, window}` per plan ([product.schema.yaml](../configs/crds/product.schema.yaml:61))
- **PolicyBundle** specifies `rateLimit.algorithm: sliding-window` + Redis config ([policybundle.schema.yaml](../configs/crds/policybundle.schema.yaml:192))

#### In M3 (Week 9)
- **Redis (Cloud Memorystore)** with sliding window counters
- Hierarchical limits: edge (coarse, per-IP) → router (fine, per-tenant) → worker (concurrency)
- Pub/Sub ordering keys + flow control enforce tenant fairness

#### In M3 (Week 10)
- Prepaid credits model (atomic decrements in Redis) for enterprise tiers
- Concurrency tracking: INCR on request start, DECR on complete
- Backpressure: if queue depth >1000, return HTTP 503 + Retry-After

#### Accuracy Target
- M1: ±5% @ p95 under 1k rps
- M3: ±3% @ p95 under 10k rps
- Post-M5: ±1% @ p95 under 100k rps

---

## 6. Agent Safety & Hallucination Guardrails ✅ ADDRESSED

### The Risk
- Builder Agent generates invalid configs that crash the system
- Optimizer Agent makes cost-destructive decisions (scaling to 1000 instances)
- Security Agent creates holes (opening 0.0.0.0/0 in firewall)

### How We Prevented It

#### In Foundation
- **AI ≠ root** principle ([PRINCIPLES.md](./PRINCIPLES.md#4-ai--root))
- **Agent action taxonomy**: read-only → propose → apply (dev) → apply (prod, gated)

#### In M4 (Weeks 13-14)
- **Orchestrator service**: central inbox, deduplication, rate limiting, sequencing
- **Validators**: schema check, Rego lint, dry-run in staging, blast-radius check
- **Constraints**: agents can't change >50 routes, can't increase quotas >2×, can't modify prod without approval

#### Agent Actions Flow
```
Agent generates intent → Orchestrator receives → Validators run
  → Schema valid? Rego compiles? Dry-run passes? Blast-radius OK?
  → If dev: auto-apply → If prod: create PR for human review
  → Audit log: signed commit, rollback recipe
```

#### Test Coverage
- Test: Builder Agent generates invalid YAML → rejected by schema validator
- Test: Optimizer Agent proposes scaling to 10,000 instances → blocked by constraint policy
- Test: Security Agent proposes IP allowlist change → requires human approval in prod

---

## 7. Cold Start Amplification ✅ ADDRESSED

### The Risk
- GKE Autopilot adds GPU nodes in 3-5 minutes
- During that time, Pub/Sub queue fills with 10k requests
- When GPU nodes arrive, thundering herd → OOM or timeout cascade

### How We Prevented It

#### In M5 (Week 23)
- **Optimizer Agent** watches queue depth + p95 latency
- **Predictive scaling**: pre-warms pools before traffic spike (based on historical patterns)
- **Backpressure**: if GPU queue depth >threshold, edge returns HTTP 503 + Retry-After
- **Graceful degradation**: route to cheaper/faster fallback models when primary pool saturated

#### Worker Pool Strategy
- CPU pool (Cloud Run): auto-scale 0-1000, cold start <2s
- GPU pool (GKE): min instances = 1, scale up gradually (not 0→10 instantly)
- Spot/preemptible GPU for batch workloads, on-demand for latency-sensitive

#### Test Coverage
- Test: 20× traffic spike → queue depth triggers pre-warming → <1% errors
- Test: GPU pool exhausted → requests routed to CPU fallback → degraded but functional

---

## 8. Config Drift & Schema Evolution ✅ ADDRESSED

### The Risk
- YAML config format evolves: new fields, deprecations, breaking changes
- Old deployments running stale configs
- Agents generating configs in outdated schema
- 500 config files in the wild, half don't validate

### How We Prevented It

#### In Foundation
- **Strict schema versioning** in all CRDs (`apiVersion: apx/v1`)
- **PolicyBundle compatibility** flag: `compat: backward|breaking` ([policybundle.schema.yaml](../configs/crds/policybundle.schema.yaml:30))

#### In M2 (Week 5)
- Config validation in CI/CD (Cloud Build)
- Schema validator blocks merges if config invalid
- Auto-migration scripts for schema upgrades

#### In M4 (Week 14)
- Builder Agent always generates latest schema version
- Portal shows deprecation warnings: "Your config uses deprecated field X; migrate by 2025-Q2"

#### Deprecation Policy
1. Announce deprecation (release notes, portal banner)
2. Dual-support for 2 releases (N and N-1)
3. Auto-migration script provided
4. Breaking change only in major version (v2)

---

## 9. Observability Cardinality Explosion ✅ ADDRESSED

### The Risk
- High-cardinality labels (tenant_id, endpoint, model, region, version)
- Cloud Monitoring charges per time series
- 1000 tenants × 50 endpoints × 5 metrics = 250k time series = $$$

### How We Prevented It

#### In Foundation
- **Label budget** enforced in PolicyBundle observability spec ([policybundle.schema.yaml](../configs/crds/policybundle.schema.yaml:269))
- **Low-cardinality labels** only: tenant_tier (not tenant_id), route_pattern (not full_path)

#### In M1 (Week 3)
- Metrics: tenant_tier, route, status_code (low-cardinality)
- Traces: tenant_id, request_id, full_path (high-cardinality OK in traces)
- Exemplars: link metrics → traces for drill-down

#### In M3 (Week 11)
- Pre-aggregate in workers: send 1 metric per minute, not per request
- BigQuery for ad-hoc queries (cheap); Monitoring for SLOs/alerts only
- Cardinality limit: 50k time series max (alarm at 40k)

---

## 10. Agents Talking to Agents (Infinite Loop) ✅ ADDRESSED

### The Risk
- Builder Agent generates config → triggers deploy → Optimizer Agent sees new traffic
  → adjusts limits → triggers re-deploy → Security Agent sees policy change → ...

### How We Prevented It

#### In M4 (Week 13)
- **Orchestrator** publishes all agent actions to shared event log (Pub/Sub topic: `agent-intents`)
- **Deduplication**: hash of intent, reject duplicates within 5-minute window
- **Rate limits**: max 1 deploy per agent per 5 minutes
- **Explicit circuit-breakers**: if >3 deploys in 15 minutes, orchestrator pauses and alerts

#### Sequencing Rules
1. Builder proposes config → Validator checks → Applier merges
2. Optimizer observes metrics (read-only) for 5 minutes before proposing action
3. Security Agent scans configs (read-only) hourly, batches proposals

#### Test Coverage
- Test: Builder + Optimizer both propose changes → orchestrator sequences correctly
- Test: Simulate loop (A→B→A) → circuit breaker trips after 3 iterations

---

## 11. Data Residency & Compliance ✅ ADDRESSED

### The Risk
- GDPR requires EU data stay in EU (fines up to 4% of global revenue)
- China requires local cloud (GCP China ≠ GCP Global)
- HIPAA/FedRAMP for healthcare/government

### How We Prevented It

#### In Foundation
- **Multi-region from day 1** principle ([PRINCIPLES.md](./PRINCIPLES.md#6-multi-region-from-day-1))
- **Product CRD**: `regionAffinity: [us, eu]`, `residency: EU|US` per plan ([product.schema.yaml](../configs/crds/product.schema.yaml:20,75))

#### In M5 (Weeks 17-20)
- VPCs per region (us-central1, eu-west1)
- Pub/Sub topics per region (us-payments, eu-payments)
- BigQuery datasets per region (no cross-region writes)
- Cloud KMS keys per region (CMEK for encryption)

#### Enforcement
- Global LB uses geo-routing + residency flag
- Router rejects requests if tenant residency ≠ region (HTTP 451)
- Logs/traces tagged with region, never cross boundary

#### Compliance Profiles
- **HIPAA tier**: dedicated pools, encrypted queues, audit logging, BAA signed
- **FedRAMP tier**: FedRAMP-certified GCP regions, FIPS 140-2 encryption

---

## 12. Debuggability ("Nobody Can Debug Production") ✅ ADDRESSED

### The Risk
- Request failed 3 hours ago
- No trace ID, no way to reconstruct what happened
- Which policy version? Which worker pool? What was the queue state?

### How We Prevented It

#### In Foundation
- **Request ID generation** at edge (Envoy) or router ([middleware/request_id.go](../router/internal/middleware/request_id.go))
- **Request ID propagation** via headers + OTEL trace context at every hop

#### In M1 (Week 3)
- Distributed tracing: OTEL span per hop (edge → router → queue → worker)
- Span attributes: tenant_id, policy_version, route, queue_depth
- Cloud Trace integration: query by request_id

#### In M3 (Week 11)
- Snapshot debugging: on errors, capture full context (request body, policy, worker metrics) to GCS
- Replay capability: save failed requests, allow manual replay in staging

#### Debug Flow
```
User reports error → Provide request_id
  → Query Cloud Trace → View span waterfall
  → Identify failing hop (e.g., worker timeout)
  → Read span attributes (policy_version, tenant_id)
  → Query BigQuery for aggregated metrics at that time
  → Retrieve snapshot from GCS if available
  → Replay in staging → Fix → Deploy
```

---

## Summary: All Gaps Addressed

| Gap | Foundation | M1-M2 | M3-M5 | Status |
|-----|-----------|-------|-------|--------|
| Multi-tenancy isolation | CRD schema | Pub/Sub topics, CMEK | Dedicated pools | ✅ |
| Streaming timeouts | Async principle | SSE, polling | WebSocket gateway | ✅ |
| Cost explosion | Sample rates | Tail-based tracing | BQ aggregates | ✅ |
| Policy versioning | Semver + compat | N/N-1 support | Canary + auto-rollback | ✅ |
| Rate limit accuracy | Redis config | Sliding window | Prepaid credits | ✅ |
| Agent safety | AI ≠ root | Validators | Constraints + audit | ✅ |
| Cold start | N/A | N/A | Predictive scaling | ✅ |
| Config drift | Schema versioning | CI validation | Auto-migration | ✅ |
| Metrics cardinality | Label budget | Exemplars | Pre-aggregation | ✅ |
| Agent loops | N/A | N/A | Orchestrator + dedupe | ✅ |
| Data residency | Multi-region CRD | N/A | Regional enforcement | ✅ |
| Debuggability | Request IDs | OTEL tracing | Snapshots + replay | ✅ |

---

## Post-Launch: Continuous Vigilance

### Quarterly Reviews
- Re-assess each gap: are mitigations still effective?
- New gaps discovered in production?
- Cost trends: observability, rate limiting, storage

### Metrics to Watch
- **Isolation breaches**: Zero tolerance; monthly audit
- **Streaming timeout rate**: Target <0.1%
- **Observability spend**: Keep <10% of infra cost
- **Rollback frequency**: If >1/week, investigate CI/CD
- **Rate limit errors**: <1% of total requests
- **Agent false positives**: <5% of proposed actions rejected

### Escalation
If any metric exceeds threshold:
1. Alert to #apx-platform Slack
2. Create incident ticket
3. Root cause analysis (RCA) within 48h
4. Update runbook + add test coverage

---

**Maintained by:** Platform Architecture Team
**Review Cadence:** Quarterly
**Last Review:** 2025-11-11
**Next Review:** 2026-02-11
