# Getting Started with APX

**Welcome to APX** - the next-generation API management platform that's AI-native, agentic, and built for GCP.

## What You Have Now

Your APX monorepo is fully scaffolded with:

✅ **Complete architecture** (OpenAI-style serving with enterprise governance)
✅ **All critical gaps addressed** (multi-tenancy, streaming, cost controls, etc.)
✅ **Production-ready schemas** (Product, Route, PolicyBundle CRDs)
✅ **Edge gateway** (Envoy + WASM filters)
✅ **Router service** (Go with middleware, policy store)
✅ **Local development stack** (Docker Compose with emulators)
✅ **Comprehensive documentation** (principles, ADRs, implementation plan)

## Quick Start (5 Minutes)

### 1. Initialize Local Environment

```bash
cd /Users/agentsy/APILEE

# Copy environment template
cp .env.example .env

# Edit .env with your settings (or use defaults for local dev)
# At minimum, set: GCP_PROJECT_ID=your-project-id

# Initialize
make init
```

### 2. Start All Services

```bash
# Start entire stack (Edge, Router, Redis, Firestore, Pub/Sub emulators, OTEL, Prometheus, Grafana)
make up

# Check status
make status

# View logs
make logs
```

### 3. Test the Stack

```bash
# Health check edge gateway
curl http://localhost:8080/health

# Health check router
curl http://localhost:8081/health

# Test a full request (once policies are loaded)
curl -X POST http://localhost:8080/v1/payments \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test-token" \
  -d '{"amount": 100}'
```

### 4. View Observability

```bash
# Open Prometheus (metrics)
make metrics
# Visit http://localhost:9090

# Open Grafana (dashboards)
make dashboards
# Visit http://localhost:3000 (admin/admin)
```

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                     Global HTTPS Load Balancer              │
│                         (GCP Cloud LB)                       │
└────────────────────────────┬────────────────────────────────┘
                             │
                    ┌────────▼─────────┐
                    │   Cloud Armor    │ (WAF, DDoS)
                    └────────┬─────────┘
                             │
        ┌────────────────────▼────────────────────┐
        │     Edge Gateway (Cloud Run + Envoy)    │
        │  • TLS termination                      │
        │  • JWT fast-path                        │
        │  • Coarse rate limiting                 │
        │  • Request ID generation                │
        │  • WASM micro-transforms                │
        └────────────────────┬────────────────────┘
                             │
        ┌────────────────────▼────────────────────┐
        │       Router Service (Go)               │
        │  • Route matching                       │
        │  • Policy version tagging               │
        │  • Tenant context propagation           │
        │  • Feature/canary routing               │
        └────────────────────┬────────────────────┘
                             │
        ┌────────────────────▼────────────────────┐
        │    Async Queue (Pub/Sub)                │
        │  • Priority lanes                       │
        │  • Per-tenant ordering                  │
        │  • Backpressure                         │
        │  • CMEK encryption                      │
        └────────┬───────────────────────┬─────────┘
                 │                       │
    ┌────────────▼────────┐   ┌─────────▼──────────┐
    │  CPU Worker Pool    │   │  GPU Worker Pool   │
    │  (Cloud Run)        │   │  (GKE Autopilot)   │
    │  • Stateless        │   │  • A100/L4 GPUs    │
    │  • Auto-scaling     │   │  • Model serving   │
    │  • Per-pool SLOs    │   │  • Agentic work    │
    └────────────┬────────┘   └─────────┬──────────┘
                 │                      │
                 └──────────┬───────────┘
                            │
        ┌───────────────────▼───────────────────┐
        │     Streaming Aggregator              │
        │  • SSE/WebSocket                      │
        │  • Resume tokens                      │
        │  • Timeout handling                   │
        └───────────────────┬───────────────────┘
                            │
        ┌───────────────────▼───────────────────┐
        │   Observability Bus                   │
        │  • OTEL (traces, metrics, logs)       │
        │  • Cloud Monitoring/Logging           │
        │  • BigQuery (analytics)               │
        │  • Prometheus + Grafana (local)       │
        └───────────────────────────────────────┘
```

## Key Components

### 1. Edge Gateway ([edge/](edge/))
- **Tech:** Envoy Proxy on Cloud Run
- **Config:** [envoy.yaml](edge/envoy/envoy.yaml)
- **Responsibilities:** TLS, JWT, rate limiting, request IDs
- **SLO:** p99 latency ≤ 20ms

### 2. Router Service ([router/](router/))
- **Tech:** Go 1.22+
- **Entry Point:** [main.go](router/cmd/router/main.go)
- **Responsibilities:** Route matching, policy selection, tenant context
- **SLO:** p99 latency ≤ 10ms

### 3. Policy Store ([router/internal/policy/](router/internal/policy/))
- **Backend:** Firestore (production) or local cache (dev)
- **Format:** Compiled PolicyBundle artifacts
- **Refresh:** Every 30 seconds

### 4. Configuration Schemas ([configs/crds/](configs/crds/))
- **Product:** Multi-tenant plans, quotas, isolation
- **Route:** Path matching, backend pools, canary
- **PolicyBundle:** Auth, authz, transforms, observability

## Documentation Map

### Start Here
1. [README.md](README.md) - Architecture overview, quick start
2. [GETTING_STARTED.md](GETTING_STARTED.md) - This file

### Understand the Design
3. [PRINCIPLES.md](docs/PRINCIPLES.md) - Non-negotiable design tenets
4. [GAPS_AND_REGRETS.md](docs/GAPS_AND_REGRETS.md) - How we prevented future regrets
5. [IMPLEMENTATION_PLAN.md](docs/IMPLEMENTATION_PLAN.md) - 6-month execution roadmap

### Deep Dives
6. [ADRs](docs/adrs/) - Architecture Decision Records
7. [CRD Schemas](configs/crds/) - Configuration format specs
8. [Sample Configs](configs/samples/) - Real-world examples

## Next Steps

### Immediate (This Week)

1. **Set up GCP infrastructure**
   ```bash
   cd infra/terraform
   terraform init
   terraform plan
   terraform apply
   ```

2. **Deploy sample policy**
   ```bash
   # Load sample payment API config to local Firestore
   # (TODO: Script this in M1)
   ```

3. **Run integration test**
   ```bash
   # Send request through full stack
   # Verify it appears in traces
   ```

### Milestone 1 (Weeks 1-4)

See [IMPLEMENTATION_PLAN.md](docs/IMPLEMENTATION_PLAN.md#phase-1-milestone-1-edge--router--async--observability) for detailed tasks.

**Goal:** Ultra-thin edge → async queue → worker → streaming response with observability

**Key Deliverables:**
- Edge + Router deployed to GCP
- Pub/Sub queue with tenant attributes
- CPU worker pool (Cloud Run)
- OTEL integration (traces, metrics, logs)

**Acceptance:**
- p99 edge overhead ≤ 20ms @ 1k rps
- 100% of requests have request_id
- BigQuery cost ≤ $15/day at test load

### Design Partners

**Target:** 2-3 early adopters by Week 2

**Ideal profiles:**
- Migrating from Apigee/Kong
- AI/ML API workloads (LLMs, embeddings)
- Multi-region requirements
- Need governance + flexibility

**Deliverables:**
- Weekly feedback sessions
- Custom configs for their use cases
- Co-design portal features

## Development Workflow

### Day-to-Day

```bash
# Start local stack
make up

# Make changes to router
cd router
go run cmd/router/main.go

# Run tests
make test-router

# View logs
make logs

# Stop stack
make down
```

### Adding a New Feature

1. **Update schemas** (if needed)
   - Edit [configs/crds/](configs/crds/)
   - Update sample configs

2. **Implement in router/workers**
   - Add middleware or handler
   - Write tests

3. **Update documentation**
   - Add ADR if architectural
   - Update implementation plan

4. **Deploy to dev**
   ```bash
   make deploy-dev
   ```

### Policy Changes

```bash
# Compile policies from YAML
make compile-policies

# Apply to dev environment
make apply-policies

# Rollout to production with canary
./tools/cli/apx rollout --canary 5% policy@1.3.0
```

## Troubleshooting

### Edge not responding
```bash
# Check Envoy logs
docker-compose logs edge

# Check Envoy admin
curl http://localhost:9901/stats
```

### Router can't connect to Firestore
```bash
# Verify Firestore emulator is running
docker-compose ps firestore

# Check router logs for connection errors
docker-compose logs router | grep firestore
```

### Pub/Sub messages not flowing
```bash
# Check Pub/Sub emulator
docker-compose ps pubsub

# Verify topic exists (in production)
gcloud pubsub topics list
```

### High latency
```bash
# Check traces in Grafana/Prometheus
make dashboards

# Query specific request_id
curl http://localhost:8081/debug/trace/{request_id}
```

## Project Structure Reference

```
/apx
├── README.md                  # Project overview
├── GETTING_STARTED.md         # This file
├── Makefile                   # Dev commands
├── docker-compose.yml         # Local stack
├── .env.example               # Config template
│
├── edge/                      # Edge gateway
│   ├── Dockerfile
│   ├── envoy/
│   │   └── envoy.yaml        # Envoy config
│   └── wasm-filters/         # WASM modules
│       └── README.md
│
├── router/                    # Router service (Go)
│   ├── go.mod
│   ├── cmd/router/main.go    # Entry point
│   ├── internal/
│   │   ├── config/           # Configuration
│   │   ├── middleware/       # HTTP middleware
│   │   ├── policy/           # Policy store
│   │   └── routes/           # Route matching
│   └── pkg/                  # Shared packages
│
├── workers/                   # Worker pools
│   ├── cpu-pool/             # Cloud Run workers
│   ├── gpu-pool/             # GKE GPU workers
│   └── examples/
│
├── control/                   # Control plane
│   ├── compiler/             # Policy compiler (OPA→WASM)
│   ├── artifact-service/     # Artifact storage
│   └── api/                  # Control plane API
│
├── agents/                    # Agentic layer
│   ├── orchestrator/         # Central coordinator
│   ├── builder/              # Config generator
│   ├── optimizer/            # SLO/cost optimizer
│   ├── security/             # Security scanner
│   └── validators/           # Action validators
│
├── portal/                    # Next.js developer portal
│   ├── app/
│   ├── components/
│   └── lib/
│
├── infra/                     # Infrastructure as code
│   ├── terraform/            # GCP resources
│   ├── kms/                  # Encryption keys
│   ├── iam/                  # Service accounts
│   └── cicd/                 # CI/CD pipelines
│
├── configs/                   # Configuration
│   ├── crds/                 # Schema definitions
│   │   ├── product.schema.yaml
│   │   ├── route.schema.yaml
│   │   └── policybundle.schema.yaml
│   ├── samples/              # Example configs
│   │   └── payments-api.yaml
│   └── environments/         # Env-specific configs
│
├── observability/             # Monitoring & logging
│   ├── otel/                 # OpenTelemetry config
│   ├── dashboards/           # Grafana dashboards
│   ├── budgets/              # Cost controls
│   └── alerts/               # Alert rules
│
├── tools/                     # Developer tools
│   ├── cli/                  # apx CLI
│   ├── load-testing/         # Load test scripts
│   └── replay/               # Request replay
│
└── docs/                      # Documentation
    ├── PRINCIPLES.md          # Design principles
    ├── GAPS_AND_REGRETS.md   # Risk mitigation
    ├── IMPLEMENTATION_PLAN.md # Roadmap
    ├── adrs/                 # Architecture decisions
    ├── runbooks/             # Operational guides
    └── blueprints/           # Integration patterns
```

## Community & Support

### Internal Resources
- **Slack:** #apx-platform
- **Weekly sync:** Tuesdays 10am PT
- **Office hours:** Thursdays 2-3pm PT

### External (Post-Launch)
- **Docs:** https://docs.apx.dev
- **GitHub Issues:** Report bugs, request features
- **Discord:** Community support

## Success Metrics

Track these to know you're on the right path:

### Milestone 1 (Weeks 1-4)
- [ ] Edge p99 latency ≤ 20ms @ 1k rps
- [ ] 100% requests have request_id
- [ ] Traces visible in Cloud Trace
- [ ] BigQuery cost ≤ $15/day at test load

### Milestone 2 (Weeks 5-8)
- [ ] Policy rollback ≤ 2 minutes
- [ ] Canary traffic split accurate within ±2%
- [ ] Zero dropped requests during rollout

### Milestone 3 (Weeks 9-12)
- [ ] Rate limit accuracy ±3% @ p95
- [ ] Observability cost ≤ 10% of infra
- [ ] BigQuery queries use partitions

### Milestone 4 (Weeks 13-16)
- [ ] Builder Agent: NL → config in <2 minutes
- [ ] Portal: Generate key → make request → see usage
- [ ] Stripe: Subscription → quota enforced

### Milestone 5 (Weeks 17-24)
- [ ] EU tenant data stays in EU (100%)
- [ ] WebSocket sessions stable >5min (99.9%)
- [ ] Regional failover RTO ≤ 10min

## Questions?

1. **How do I add a new API product?**
   - Create a Product YAML in [configs/samples/](configs/samples/)
   - Define plans, quotas, isolation
   - Compile and apply: `make apply-policies`

2. **How do I change rate limits?**
   - Edit the Product YAML `rateLimit` section
   - Deploy with canary: `apx rollout --canary 5%`
   - Monitor error rate, rollback if needed

3. **How do I debug a failed request?**
   - Get the `request_id` from the user
   - Query Cloud Trace: `apx trace {request_id}`
   - View span attributes (tenant, policy, route)
   - Query BigQuery for context

4. **How do I add a new worker pool?**
   - Create a new directory in [workers/](workers/)
   - Define Dockerfile and GKE/Cloud Run config
   - Update router to route to new pool
   - Deploy: `make deploy-dev`

5. **How do I contribute?**
   - See [CONTRIBUTING.md](docs/CONTRIBUTING.md) (TODO: Create)
   - PR checklist: tests pass, docs updated, ADR if needed

---

**Ready to build the future of API management?**

Run `make up` and let's go! 🚀

---

**Last Updated:** 2025-11-11
**Maintained by:** Platform Architecture Team
