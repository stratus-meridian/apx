# APX Implementation Plan
**Next-Gen API Management Platform - Execution Roadmap**

**Last Updated:** 2025-11-11
**Status:** Foundation Complete, M1 Ready to Begin

---

## Executive Summary

This implementation plan transforms the OpenAI-style serving architecture into a production-ready, AI-native API management platform. All critical gaps and "future regrets" identified in the architecture review are baked into the foundation.

**Key Differentiators:**
- Ultra-thin edge (Envoy + WASM) vs. heavy policy gateways
- Async-by-default for AI workloads (streaming, long-running)
- Config-as-artifacts with N/N-1 versioning for safe rollouts
- Agent-assisted operations with human guardrails
- Observability within budget (sampled logs, aggregated analytics)
- Multi-region + data residency from day 1

---

## Phase 0: Foundation ✅ COMPLETE

**Timeline:** Week 0-1
**Status:** ✅ Scaffolded

### Deliverables (Complete)

#### 1. Repository Structure
```
/apx
  ├── edge/              ✅ Envoy config, WASM filters, Dockerfile
  ├── router/            ✅ Go service structure, middleware, policy store
  ├── workers/           ✅ Directory structure
  ├── control/           ✅ Directory structure
  ├── agents/            ✅ Directory structure
  ├── portal/            ✅ Directory structure
  ├── infra/             ✅ Directory structure
  ├── configs/           ✅ CRD schemas (Product, Route, PolicyBundle)
  ├── observability/     ✅ Directory structure
  ├── tools/             ✅ Directory structure
  └── docs/              ✅ ADRs, principles, runbooks
```

#### 2. Configuration Schemas ✅
- [Product CRD](../configs/crds/product.schema.yaml) - Multi-tenant plans, quotas, isolation
- [Route CRD](../configs/crds/route.schema.yaml) - Routing rules, backends, canary
- [PolicyBundle CRD](../configs/crds/policybundle.schema.yaml) - Auth, authz, transforms, observability

#### 3. Sample Configurations ✅
- [Payment API Example](../configs/samples/payments-api.yaml) - Complete product with 3 tiers

#### 4. Core Documentation ✅
- [First Principles](./PRINCIPLES.md) - Non-negotiable design tenets
- [ADR 001: Monorepo Structure](./adrs/001-monorepo-structure.md)
- [README](../README.md) - Quick start, architecture overview

#### 5. Development Environment ✅
- [docker-compose.yml](../docker-compose.yml) - Local stack (Envoy, Router, Redis, Firestore, Pub/Sub, OTEL, Prometheus, Grafana)
- [Makefile](../Makefile) - Dev commands (`make up`, `make test`, `make deploy`)
- [.env.example](../.env.example) - Configuration template

#### 6. Edge Gateway Scaffold ✅
- Envoy configuration with JWT fast-path, rate limiting, OTEL tracing
- Dockerfile for Cloud Run deployment
- WASM filter placeholders (tenant extraction, PII redaction)

#### 7. Router Service Scaffold ✅
- Go service structure with middleware chain
- Policy store (Firestore-backed)
- Tenant context propagation
- Request ID generation

### Acceptance Criteria ✅
- [x] Repository structure matches monorepo design
- [x] All CRD schemas validate against spec
- [x] docker-compose starts without errors
- [x] README provides clear quick-start path
- [x] First Principles document signed off by architecture team

---

## Phase 1 (Milestone 1): Edge + Router + Async + Observability

**Timeline:** Weeks 1-4 (4 weeks)
**Goal:** Ultra-thin edge → async queue → worker → streaming response with full observability

### Week 1: Core Runtime

#### Edge Gateway
- [x] Envoy configuration (JWT, rate limiting, CORS)
- [ ] Deploy to Cloud Run with Cloud Armor
- [ ] Integrate with OTEL collector
- [ ] Health checks + readiness probes
- [ ] Load test: 1k rps → p99 overhead <20ms

#### Router Service
- [x] Go service with middleware chain
- [ ] Route matching (path, host, method)
- [ ] Policy bundle loading from Firestore
- [ ] Policy version tagging (`x-apx-policy-version: 1.2.0`)
- [ ] Pub/Sub message publishing with tenant attributes
- [ ] Health/ready endpoints

#### Infrastructure
- [ ] Terraform: VPC, subnets, Cloud NAT
- [ ] Terraform: Firestore database
- [ ] Terraform: Pub/Sub topics (per region)
- [ ] Terraform: Cloud KMS keys (CMEK for queues)
- [ ] Terraform: Service accounts (least privilege)

### Week 2: Async Queue + Workers

#### Pub/Sub Setup
- [ ] Topic per region (us-payments, eu-payments)
- [ ] Message schema: request_id, tenant_id, policy_version, residency
- [ ] Ordering keys for tenant fairness
- [ ] Dead-letter queue for failed messages
- [ ] Encryption with CMEK

#### Worker Pool (CPU - Cloud Run)
- [ ] Go worker service: consume Pub/Sub, process, respond
- [ ] Streaming response aggregator (SSE or gRPC)
- [ ] Timeout handling (30s default, configurable per route)
- [ ] Retry logic with exponential backoff
- [ ] Graceful shutdown (drain in-flight requests)

#### Integration
- [ ] Edge → Router → Pub/Sub → Worker end-to-end flow
- [ ] Test: POST request → 202 Accepted → poll /status/{request_id}
- [ ] Test: Streaming response via SSE

### Week 3: Observability Foundation

#### OpenTelemetry
- [ ] OTEL SDK in Edge (Envoy tracer)
- [ ] OTEL SDK in Router (Go)
- [ ] OTEL SDK in Workers
- [ ] Trace context propagation (W3C Trace Context)
- [ ] Baggage propagation (tenant_id, policy_version)

#### Metrics
- [ ] Prometheus metrics: request_count, request_duration, error_rate
- [ ] Low-cardinality labels: tenant_tier, route_pattern, status_code
- [ ] Export to Cloud Monitoring
- [ ] SLO dashboards: availability, p99 latency, error budget

#### Logging
- [ ] Structured JSON logs (zap in Go)
- [ ] Sampling: 1% at edge, 100% for errors
- [ ] Log correlation via request_id
- [ ] Export to Cloud Logging with budget caps

#### Tracing
- [ ] Tail-based sampling (errors + p99 outliers)
- [ ] Cloud Trace integration
- [ ] Span attributes: tenant_id, policy_version, route

### Week 4: Testing + Polish

#### Testing
- [ ] Unit tests: router middleware, policy store
- [ ] Integration tests: Edge → Router → Worker
- [ ] Load tests: 1k rps sustained, 5k rps burst
- [ ] Chaos tests: kill workers, delay queues
- [ ] Security tests: JWT tampering, replay attacks

#### Acceptance Criteria
- [ ] **Edge latency**: p99 ≤ 20ms @ 1k rps
- [ ] **No request without request_id**: 100% coverage
- [ ] **BigQuery daily cost**: ≤ $15/day at test load (100k req/day)
- [ ] **Observability**: Traces viewable in Cloud Trace within 30s
- [ ] **Async flow**: 202 response → poll /status → 200 with result
- [ ] **Regional isolation**: US requests stay in us-central1

### Risk Mitigations
- **Risk:** Pub/Sub latency >100ms
  - **Mitigation:** Use regional topics, avoid cross-region
- **Risk:** Worker cold starts >5s
  - **Mitigation:** Min instances = 1 for Cloud Run
- **Risk:** OTEL overhead >10ms p99
  - **Mitigation:** Async export, batch spans

---

## Phase 2 (Milestone 2): Policies + Versioning + Rollouts

**Timeline:** Weeks 5-8 (4 weeks)
**Goal:** Safe policy rollouts with canary, N/N-1 support, instant rollback

### Week 5: Policy Compiler

#### OPA Integration
- [ ] OPA binary in control plane service
- [ ] Rego policy validation
- [ ] Compile Rego → WASM bundles
- [ ] Hash artifacts (SHA256)
- [ ] Upload to GCS artifact store (gs://apx-artifacts/policies/)

#### Compiler Service
- [ ] Go service: accept YAML → compile → store artifact
- [ ] API: POST /compile (accepts PolicyBundle YAML)
- [ ] Versioning: semver (1.2.0) + compatibility flag (backward/breaking)
- [ ] Artifact format: {name}@{version}@{hash}.{wasm|json}

#### GitOps Integration
- [ ] Cloud Build trigger on config repo push
- [ ] Validate YAML schema
- [ ] Compile to artifacts
- [ ] Store in GCS + Firestore (metadata)

### Week 6: N/N-1 Policy Support

#### Router Enhancement
- [ ] Policy version selection: read `x-apx-policy-version` header
- [ ] Load multiple policy versions concurrently (N and N-1)
- [ ] Default to latest if version not specified
- [ ] Reject requests with unsupported policy versions (HTTP 409)

#### Worker Enhancement
- [ ] Workers accept policy_version from Pub/Sub message
- [ ] Load policy artifact from GCS or Firestore cache
- [ ] Apply policy to request (auth, quotas, transforms)
- [ ] Support N and N-1 versions for 24h rollout window

#### Testing
- [ ] Test: Deploy policy v1.2.0, all requests use it
- [ ] Test: Deploy policy v1.3.0, in-flight v1.2.0 requests continue
- [ ] Test: Gradual rollout (10% v1.3.0, 90% v1.2.0)

### Week 7: Canary Rollouts

#### Canary Logic
- [ ] Router: read Route.canary.weight (1-100%)
- [ ] Use consistent hashing (tenant_id) for stickiness
- [ ] Tag requests: `x-apx-canary: true|false`
- [ ] Metrics split: canary vs. stable

#### Rollback Automation
- [ ] Monitor error rate per policy version
- [ ] If canary error_rate > stable + 5%: auto-rollback
- [ ] Rollback = set canary weight to 0%, purge from cache
- [ ] Alert to Slack/PagerDuty

#### CLI Tools
- [ ] `apx rollout --canary 5% policy@1.3.0`
- [ ] `apx rollout --increase 25%`
- [ ] `apx rollback policy@1.3.0`
- [ ] `apx status policy@1.3.0` (shows traffic split, error rate)

### Week 8: Testing + Acceptance

#### Testing
- [ ] Test: Canary 5% → 100% over 1 hour
- [ ] Test: Introduce breaking policy → auto-rollback
- [ ] Test: In-flight requests complete with old policy
- [ ] Test: Rollback completes in <2 minutes

#### Acceptance Criteria
- [ ] **Rollback time**: ≤ 2 minutes, zero dropped requests
- [ ] **In-flight safety**: Requests carry policy_version, unaffected by rollout
- [ ] **Canary accuracy**: Traffic split within ±2% of target
- [ ] **Auto-rollback**: Triggered within 60s of error spike

---

## Phase 3 (Milestone 3): Rate Limiting + Cost Controls

**Timeline:** Weeks 9-12 (4 weeks)
**Goal:** Accurate distributed rate limiting, budget-aware observability

### Week 9: Redis Rate Limiting

#### Redis Setup
- [ ] Cloud Memorystore (Redis) in each region
- [ ] VPC peering for private access
- [ ] HA configuration (2 replicas)

#### Rate Limit Service
- [ ] Go service: implements token bucket or sliding window
- [ ] gRPC API: CheckRateLimit(tenant_id, key, limit, window)
- [ ] Redis operations: INCR + EXPIRE (sliding window)
- [ ] Hierarchical limits: per-key, per-tenant, per-tier

#### Integration
- [ ] Envoy calls ratelimit service (gRPC)
- [ ] Router checks Redis before queuing message
- [ ] Return HTTP 429 + Retry-After header
- [ ] Metrics: rate_limit_exceeded (by tenant_tier)

### Week 10: Queue Fairness

#### Pub/Sub Enhancements
- [ ] Ordering keys = tenant_id (ensures per-tenant FIFO)
- [ ] Flow control settings: max 100 outstanding messages per worker
- [ ] Backpressure: if queue depth >1000, return 503 at edge

#### Per-Tenant Concurrency Caps
- [ ] Track active requests per tenant in Redis
- [ ] INCR on request start, DECR on complete
- [ ] Reject if tenant exceeds concurrency limit (from Product plan)

### Week 11: Cost Controls

#### BigQuery Optimization
- [ ] Partition by date (daily)
- [ ] Cluster by tenant_id
- [ ] Materialize hourly aggregates (Cloud Function or Dataflow)
- [ ] Set table expiration: raw logs 7 days, aggregates 2 years

#### Logging Budget
- [ ] Sampling: 1% success, 100% errors
- [ ] Tail-based tracing: errors + p99 outliers only
- [ ] Budget alerts: if observability cost >7% of infra, alarm
- [ ] Dashboard: cost per tenant, cost per route

#### Metrics Cardinality
- [ ] Audit metric labels: remove tenant_id, keep tenant_tier
- [ ] Use exemplars (link metrics → traces for high-cardinality data)
- [ ] Limit to 50k time series max

### Week 12: Testing + Acceptance

#### Testing
- [ ] Test: 10k rps burst → rate limit enforced within ±3%
- [ ] Test: Free-tier user exceeds quota → 429 response
- [ ] Test: Premium user never throttled incorrectly
- [ ] Test: BigQuery scans use partitions (query cost <$0.01)

#### Acceptance Criteria
- [ ] **Rate limit accuracy**: ±3% @ p95 under 10k rps
- [ ] **BigQuery cost**: ≤ $5/day for 1M requests/day
- [ ] **Observability spend**: ≤ 10% of total infra cost

---

## Phase 4 (Milestone 4): Agents + Portal

**Timeline:** Weeks 13-16 (4 weeks)
**Goal:** Agent-assisted operations, developer portal, monetization

### Week 13: Agent Orchestrator

#### Orchestrator Service
- [ ] Go service: central inbox for agent intents
- [ ] Pub/Sub topic: agent-intents
- [ ] Deduplication (hash of intent)
- [ ] Rate limiting (max 1 action per agent per 5min)
- [ ] Sequencing (Builder → Validator → Applier)

#### Validators
- [ ] Schema validator: check YAML against CRD schema
- [ ] Policy linter: check Rego syntax
- [ ] Dry-run: apply policy to staging, check metrics
- [ ] Blast-radius: prevent changes affecting >50 routes

### Week 14: Builder Agent v0

#### Agent Logic (Vertex AI + Gemini)
- [ ] Prompt template: NL intent → OpenAPI + PolicyBundle YAML
- [ ] Vertex AI Agent: orchestrate generation + validation
- [ ] Context: existing Product/Route/PolicyBundle configs
- [ ] Output: PR to config repo (GitHub)

#### Integration
- [ ] Web UI: input box "Create payments API with OAuth2"
- [ ] Agent generates configs
- [ ] Create PR in GitHub
- [ ] Human reviews + merges (or auto-merge in dev)

### Week 15: Portal v1 (Next.js)

#### Features
- [ ] Firebase Auth (SSO)
- [ ] API key management (generate, revoke, rotate)
- [ ] Usage dashboard (requests, latency, errors, cost)
- [ ] Live console (tail logs for tenant)
- [ ] OpenAPI docs (generated from configs)
- [ ] Billing (Stripe integration)

#### Backend API
- [ ] Go service: /keys, /usage, /billing
- [ ] Read from BigQuery (aggregates)
- [ ] Firestore: API key storage (hashed)

### Week 16: Monetization

#### Stripe Integration
- [ ] Plans: free, pro, enterprise
- [ ] Usage-based billing (cost per 1k requests)
- [ ] Overage charges
- [ ] Webhooks: subscription.created, invoice.paid

#### Quota Enforcement
- [ ] Check Stripe subscription status before admitting request
- [ ] If subscription expired: HTTP 402 Payment Required
- [ ] Prepaid credits (atomic decrement in Redis)

#### Acceptance Criteria
- [ ] **Builder Agent**: NL → config in <2 minutes (dev auto-merge)
- [ ] **Portal**: Generate API key, make request, see usage in dashboard
- [ ] **Billing**: Stripe subscription → quota enforced → invoice generated

---

## Phase 5 (Milestone 5): Multi-Region + Residency

**Timeline:** Weeks 17-24 (8 weeks)
**Goal:** EU/US regions, data residency enforcement, WebSocket gateway

### Week 17-18: Multi-Region Infrastructure

#### GCP Setup
- [ ] VPCs in us-central1, eu-west1
- [ ] Pub/Sub topics per region
- [ ] BigQuery datasets per region (us-payments, eu-payments)
- [ ] Cloud KMS keys per region

#### Global Load Balancer
- [ ] External HTTPS LB with geo-routing
- [ ] Backend services per region (Cloud Run edge)
- [ ] Cloud Armor policies (WAF, DDoS)

### Week 19-20: Residency Enforcement

#### Product Config
- [ ] `residency: US|EU|ASIA|GLOBAL` in Product spec
- [ ] LB routes based on residency flag + geo hint
- [ ] Pub/Sub: enforce topic = region for residency=EU tenants
- [ ] BigQuery: write to regional dataset only

#### Validation
- [ ] Test: EU tenant request → routed to eu-west1 → data in EU BigQuery
- [ ] Test: US tenant request → us-central1 only
- [ ] Test: Attempt to route EU data to US → rejected (HTTP 451)

### Week 21-22: WebSocket Gateway (GKE)

#### Why GKE?
- Cloud Run has 60min timeout; agents may need hours
- WebSocket requires long-lived connections

#### Setup
- [ ] GKE Autopilot cluster per region
- [ ] WebSocket gateway (Go) deployed as Deployment
- [ ] Horizontal Pod Autoscaler (HPA) based on connection count
- [ ] Resume tokens: if connection drops, client reconnects with token

#### Integration
- [ ] Client connects to wss://api.example.com/ws
- [ ] Gateway subscribes to Pub/Sub for tenant's messages
- [ ] Stream responses as JSON lines
- [ ] Timeout: 1 hour, then send resume token

### Week 23: Optimizer Agent v0

#### Agent Logic
- [ ] Monitor: p99 latency, error budget, cost per request
- [ ] Actions:
  - Scale worker pools (min instances)
  - Adjust concurrency limits
  - Enable caching for hot routes
  - Flip canary (rollback if degraded)
- [ ] Constraints: can't change quotas >50%, can't auto-apply to prod

#### Integration
- [ ] Cloud Function triggered every 5 minutes
- [ ] Read metrics from Cloud Monitoring
- [ ] Publish intent to orchestrator
- [ ] Orchestrator applies (with approval gate for prod)

### Week 24: Testing + Acceptance

#### Testing
- [ ] Test: 20× traffic spike → pre-warm pools, graceful degradation
- [ ] Test: EU customer → data never leaves EU
- [ ] Test: WebSocket session lasts 30min, no drops
- [ ] Test: Optimizer auto-scales workers during spike

#### Acceptance Criteria
- [ ] **Regional failover**: RTO ≤ 10min (control plane), data plane continues
- [ ] **Residency enforcement**: 100% of EU tenant data in eu-west1
- [ ] **WebSocket stability**: 99.9% uninterrupted sessions <5min
- [ ] **Optimizer**: Auto-scales correctly, no false positives

---

## Post-Milestone 5: Future Roadmap

### Q2 2026
- [ ] Security Agent (auto-apply firewall rules, detect anomalies)
- [ ] Docs/SDK Agent (auto-generate SDKs, test collections)
- [ ] Multi-cloud (AWS, Azure backends)
- [ ] GraphQL support (in addition to REST)

### Q3 2026
- [ ] Advanced caching (semantic cache for AI responses)
- [ ] Model router (cost/latency-aware model selection)
- [ ] FedRAMP compliance (dedicated infra, FIPS 140-2)

### Q4 2026
- [ ] Real-time analytics (sub-second dashboards)
- [ ] Predictive scaling (ML-based traffic forecasting)
- [ ] Monetization Agent (auto-tune pricing based on elasticity)

---

## Success Metrics (Platform-Wide)

| Metric | M1 Target | M5 Target | Post-M5 Target |
|--------|-----------|-----------|----------------|
| **Availability (regional)** | 99.9% | 99.95% | 99.99% |
| **Edge latency (p99)** | <25ms | <20ms | <15ms |
| **Rate limit accuracy** | ±5% | ±3% | ±1% |
| **Observability cost** | <15% | <10% | <7% |
| **Rollback time** | <5min | <2min | <1min |
| **Agent accuracy** | N/A | 80% | 95% |

---

## Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Pub/Sub latency spikes | Medium | High | Regional topics, monitoring, fallback to direct worker calls |
| Agent hallucinations break prod | Low | Critical | Human-in-loop for prod, validators, dry-run in staging |
| Cost overrun (logs/BQ) | Medium | High | Sampling, partitioning, budget alerts |
| Cold start cascade | Medium | Medium | Predictive scaling, min instances, circuit breakers |
| Multi-tenant isolation breach | Low | Critical | Dedicated pools for enterprise, PII redaction, audit logs |
| GCP region outage | Low | High | Multi-region from M5, LB failover, stateless design |

---

## Team Structure (Recommended)

### Core Team (6-8 engineers)
- **Platform Lead** (1): Architecture, roadmap, stakeholder management
- **Infrastructure** (2): GCP, Terraform, K8s, networking
- **Backend** (2): Go services (router, workers, control plane)
- **Agents** (1): Vertex AI, LangChain, agent orchestration
- **Frontend** (1): Next.js portal, observability dashboards
- **SRE** (1): Monitoring, on-call, incident response

### Extended Team (Part-Time)
- **Security**: IAM, compliance, pentesting
- **Product**: Requirements, design partners, GTM
- **DevRel**: Docs, SDKs, developer experience

---

## Dependencies (External)

1. **GCP Project Setup**
   - Project ID, billing account
   - APIs enabled: Compute, GKE, Cloud Run, Pub/Sub, Firestore, Cloud Build, Secret Manager, KMS
   - Service accounts with IAM roles

2. **GitHub Repo**
   - Monorepo created
   - Branch protection rules
   - GitHub Actions runners

3. **Stripe Account**
   - API keys (test + live)
   - Products/prices configured
   - Webhook endpoints

4. **Design Partners**
   - 2-3 early adopters for M1-M3 testing
   - Feedback loop established

---

## Decision Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2025-11-11 | Monorepo structure | Atomic changes, agent context, shared tooling |
| 2025-11-11 | Go for router/workers | Performance, concurrency, GCP SDK maturity |
| 2025-11-11 | Envoy (not Nginx) | WASM filters, OTEL native, Istio compatibility |
| 2025-11-11 | Redis for rate limiting | Distributed, atomic ops, <1ms latency |
| 2025-11-11 | Firestore for policy cache | Real-time sync, multi-region, serverless |
| 2025-11-11 | BigQuery for analytics | Cost-effective at scale, SQL, BI tool integrations |

---

## Next Steps (Immediate)

1. **Infrastructure Setup** (Week 1)
   - [ ] Create GCP projects (dev, staging, prod)
   - [ ] Set up Terraform backend (GCS bucket for state)
   - [ ] Create service accounts + IAM bindings
   - [ ] Enable required APIs

2. **Local Development** (Week 1)
   - [ ] Run `make init` to set up local environment
   - [ ] Run `make up` to start docker-compose stack
   - [ ] Test edge health check: `curl http://localhost:8080/health`
   - [ ] Test router health check: `curl http://localhost:8081/health`

3. **First Integration Test** (Week 1)
   - [ ] Deploy sample policy to Firestore emulator
   - [ ] Send request through edge → router
   - [ ] Verify request tagged with policy_version
   - [ ] Verify trace visible in local OTEL collector

4. **Design Partner Kickoff** (Week 2)
   - [ ] Identify 2-3 early adopters
   - [ ] Onboard to dev environment
   - [ ] Weekly feedback sessions

---

## Appendix: Key Architectural Decisions

See [ADRs directory](./adrs/) for full details.

- [ADR 001: Monorepo Structure](./adrs/001-monorepo-structure.md)
- ADR 002: Async-by-Default (TODO)
- ADR 003: Policy Versioning (TODO)
- ADR 004: Multi-Region Data Residency (TODO)
- ADR 005: Agent Guardrails (TODO)

---

**End of Implementation Plan**

**Maintained by:** Platform Architecture Team
**Review Cadence:** Bi-weekly
**Feedback:** #apx-platform on Slack
