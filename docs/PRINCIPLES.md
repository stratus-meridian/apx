# APX First Principles

**Non-negotiables that guide every architectural decision**

## 1. Tenant-First Design

**What it means:**
- Isolation level (shared, namespace, dedicated) is explicit in every Product config
- Tenant context (tenant_id, isolation_tier) propagates through every hop via headers + OTEL baggage
- Queues, workers, logs, metrics, and storage respect tenant boundaries

**Why it matters:**
- Prevents cross-tenant data leakage (security nightmare)
- Enables per-tenant SLOs, quotas, and cost attribution
- Foundation for SOC2/HIPAA/FedRAMP compliance

**How we enforce:**
- ✅ Tenant ID required in every request (extracted at edge)
- ✅ OTEL baggage carries tenant_id + isolation_tier
- ✅ Pub/Sub messages tagged with tenant attributes
- ✅ Workers validate tenant context before processing
- ✅ Logs redact/anonymize cross-tenant data
- ✅ Dedicated pools for `isolation: dedicated` tenants

**Failure mode if violated:**
> "Enterprise customer discovers their prompts leaked into another tenant's logs during SOC2 audit. Platform shut down for security review."

---

## 2. Async-by-Default

**What it means:**
- Edge never waits for backend work to complete
- Pattern: request → ack → queue → worker → result (via callback/polling/stream)
- Long-running operations (>30s) use WebSocket gateway or status polling

**Why it matters:**
- Cloud Run/LB timeouts (30-60s) kill long-running requests
- AI workloads (GPT-4, agents) can take minutes
- Prevents cascading failures when workers are slow

**How we enforce:**
- ✅ Edge returns 202 Accepted + request_id immediately
- ✅ Client polls /status/{request_id} or subscribes to SSE stream
- ✅ Workers publish results to callback URLs or streaming channel
- ✅ WebSocket gateway (on GKE) for sessions >5 minutes
- ✅ Resume tokens allow clients to reconnect if stream breaks

**Failure mode if violated:**
> "Half our AI requests timeout. Users complain responses cut off mid-sentence. Customer churn accelerates."

---

## 3. Config-as-Artifacts

**What it means:**
- Every config (Product, Route, PolicyBundle) is versioned (semver)
- Compiler produces immutable artifacts (WASM + JSON) with SHA256 hash
- Requests carry the policy_version they were admitted under
- Workers support N and N-1 policy versions during rollout

**Why it matters:**
- Safe canary rollouts (1%→5%→25%→100%)
- Instant rollback without affecting in-flight requests
- Reproducible deploys (same config → same artifact → same behavior)
- Auditability (who changed what, when)

**How we enforce:**
- ✅ GitOps: configs in Git → Cloud Build → artifact store (GCS)
- ✅ Router tags requests with `x-apx-policy-version: 1.2.0`
- ✅ Workers validate policy version; reject if unsupported
- ✅ Blue/green deploys: new version canaried before old version deprecated
- ✅ Automatic rollback if error rate >5% during canary

**Failure mode if violated:**
> "Policy change deployed Friday 4pm. Broke 20% of traffic. No clean rollback path. Weekend on-call nightmare."

---

## 4. AI ≠ Root

**What it means:**
- Agents (Builder, Optimizer, Security) propose actions, never directly modify production
- Guardrails validate every agent action (schema, lint, dry-run, blast-radius)
- Humans gate production changes (auto-merge allowed in dev/staging with constraints)
- All agent actions are auditable and reversible

**Why it matters:**
- Prevents agent hallucinations from breaking production
- Avoids cost-destructive decisions (scaling to 1000 GPU instances)
- Maintains human accountability for critical changes

**How we enforce:**
- ✅ Agent taxonomy: read-only → propose → apply (dev) → apply (prod, gated)
- ✅ Orchestrator sequences agent intents; detects loops
- ✅ Validators run before any apply: schema, Rego lint, dry-run in staging
- ✅ Blast-radius checks: can't change >50 routes, can't increase quotas >2x
- ✅ Production requires PR approval (or auto-merge under strict constraints)
- ✅ Audit log: signed commits, changelogs, rollback recipes

**Failure mode if violated:**
> "Optimizer Agent decided to save costs by setting all quotas to 1 req/s. Production melted. $2M revenue impact."

---

## 5. Observability Within Budget

**What it means:**
- Metrics (cheap) for SLOs and dashboards
- Logs/traces (expensive) sampled aggressively (1-5% at edge, 100% for errors)
- Analytics (BigQuery) uses hourly aggregates, not raw scans
- High-cardinality labels (tenant_id) in traces only, not metrics

**Why it matters:**
- Cloud Logging costs scale linearly with volume ($0.50/GB ingestion)
- BigQuery scans expensive without partitioning/clustering
- Time series cardinality explosions (1000 tenants × 50 routes × 5 metrics = 250k series)

**How we enforce:**
- ✅ Edge samples logs at 1% (errors + high-value tenants = 100%)
- ✅ OTEL tail-based sampling: keep errors + p99 latency outliers
- ✅ Metrics use low-cardinality labels (tenant_tier, not tenant_id)
- ✅ BigQuery: partitioned by date, clustered by tenant_id
- ✅ Hourly aggregates materialized; raw logs expire after 7 days
- ✅ Budget alerts: observability spend >7% of infra triggers alarm

**Failure mode if violated:**
> "Observability costs $50k/month—more than compute. BigQuery scans blow through budget. CFO demands cuts."

---

## 6. Multi-Region from Day 1

**What it means:**
- Region affinity explicit in Product config (`regionAffinity: us`, `residency: EU`)
- Global LB routes to nearest region; regional LB routes within region
- Queues, storage, analytics respect residency (data never crosses boundary)
- Control plane (configs, artifacts) can be global; data plane is regional

**Why it matters:**
- GDPR requires EU data stay in EU (fines up to 4% of global revenue)
- China requires local cloud (GCP China ≠ GCP Global)
- Latency: routing US customers to EU adds 100-200ms

**How we enforce:**
- ✅ Product config: `regionAffinity: [us, eu]`, `residency: EU`
- ✅ Global LB uses geo-routing; regional LB enforces residency
- ✅ Pub/Sub topics per region (us-payments, eu-payments)
- ✅ BigQuery datasets per region; analytics pipelines respect boundaries
- ✅ CMEK (Cloud KMS) per region for encryption
- ✅ Control plane: configs replicated globally; artifacts cached per region

**Failure mode if violated:**
> "Can't enter EU market. Architecture assumes single-region. 18-month re-architecture required. Competitor wins $10M deal."

---

## How These Principles Interact

**Example: Policy Rollout with Tenant Isolation**
1. Builder Agent proposes new PolicyBundle (AI ≠ root)
2. Validators check: schema valid, dry-run passes, blast-radius OK
3. Human approves PR → Cloud Build compiles to artifact (config-as-artifact)
4. Canary: 1% of traffic in `us` region uses new policy (multi-region)
5. Router tags requests with policy_version (async-by-default: ack immediately)
6. Workers process with tenant_id in OTEL baggage (tenant-first)
7. Metrics show p99 latency + error rate per tenant_tier (observability within budget)
8. If errors spike, auto-rollback to previous artifact version (config-as-artifact)

---

## Decision Framework

When facing a trade-off, ask:

1. **Does this violate tenant isolation?** → Reject
2. **Does this block the edge?** → Make it async
3. **Can this change be rolled back safely?** → If no, version it
4. **Could an agent action break production?** → Add guardrails
5. **Will this observability change blow the budget?** → Sample or aggregate
6. **Does this assume single-region?** → Make it multi-region aware

---

## Exceptions & Overrides

Sometimes principles conflict. Priority order:

1. **Security/Compliance** (tenant isolation, residency) → never compromise
2. **Availability** (async, rollback) → critical for SLOs
3. **Cost** (observability budget) → important but not at expense of safety
4. **Velocity** (agent automation) → lowest priority; humans can always intervene

**Example:** If agent-driven rollouts are too slow for incident response, humans can override. But agent safety guardrails (schema validation, blast-radius) still apply.

---

## Living Document

These principles evolve as we learn. When proposing a change:

1. Document the decision in an ADR ([docs/adrs/](./adrs/))
2. Explain which principle is affected and why
3. Get sign-off from architecture review

**Last updated:** 2025-11-11
**Next review:** 2026-02-11
