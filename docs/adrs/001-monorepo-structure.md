# ADR 001: Monorepo Structure

**Status:** Accepted
**Date:** 2025-11-11
**Deciders:** Architecture Team
**Tags:** foundation, repo-structure

## Context

We need to organize code for a complex, multi-component platform:
- Edge gateway (Envoy + WASM)
- Router service (Go)
- Worker pools (CPU/GPU)
- Control plane (compiler, artifact service)
- Agents (orchestrator, builder, optimizer, security)
- Portal (Next.js)
- Infrastructure (Terraform)
- Shared configs and schemas

## Decision

Use a **monorepo** with clear directory boundaries per component.

### Structure

```
/apx
  /edge           - Envoy config, WASM filters, Docker images
  /router         - Go routing service
  /workers        - CPU/GPU worker pools
  /control        - Policy compiler, artifact service, control API
  /agents         - Orchestrator + specialized agents
  /portal         - Next.js AgentHub
  /infra          - Terraform, IAM, CI/CD
  /configs        - CRDs, samples, environments
  /observability  - OTEL, dashboards, budgets
  /tools          - CLI, load testing, replay
  /docs           - ADRs, runbooks, blueprints
```

### Rationale

**Pros of monorepo:**
1. **Atomic changes across components** - Router + PolicyBundle schema can change together
2. **Shared tooling** - One CI/CD pipeline, one CLI, one deployment tool
3. **Agent coordination** - Builder Agent can see entire codebase context
4. **Consistent versioning** - Single version number for entire platform release
5. **Easier local development** - `docker-compose up` starts entire stack

**Cons (and mitigations):**
1. **Large repo size** → Use sparse checkout for CI jobs
2. **Blast radius of bad commits** → Branch protection, required reviews, automated tests
3. **Build times** → Bazel or Nx for incremental builds (defer until M3)

### Alternatives Considered

**Option 1: Multi-repo (separate repo per component)**
- ❌ Harder to coordinate breaking changes across repos
- ❌ Version skew between components
- ❌ Agents need cross-repo access (complex GitHub App permissions)

**Option 2: Monorepo with microservices**
- ✅ Chosen approach
- Each directory is independently deployable
- Shared schemas live in `/configs/crds`

**Option 3: Monolith**
- ❌ Can't scale edge and workers independently
- ❌ GPU pools require GKE; edge works on Cloud Run

## Consequences

### Positive
- Simplified CI/CD (single pipeline with conditional steps)
- Agents can analyze entire codebase for impact analysis
- Shared config schemas prevent drift
- Easier onboarding (clone one repo, not ten)

### Negative
- Need discipline to keep components decoupled (no `../router` imports in workers)
- Git clone time grows (mitigate with sparse checkout)
- Requires good tooling (apx CLI) to work with individual components

### Neutral
- Each component still has independent Dockerfile and deployment
- Services communicate via APIs/Pub/Sub, not function calls

## Compliance & Security

- Each component has separate service account (least privilege)
- Secrets (API keys, KMS) never committed to repo
- Terraform state stored in GCS backend, not in repo

## Monitoring & Rollback

- Each component versions independently in Git tags (`edge/v1.2.0`, `router/v1.3.1`)
- Platform releases are tagged (`release/v0.1.0`) with component manifest
- Rollback: revert commit + re-run deploy for affected component

## References

- [Google's monorepo philosophy](https://research.google/pubs/pub45424/)
- [Nx monorepo tools](https://nx.dev)
- [Bazel for incremental builds](https://bazel.build)
