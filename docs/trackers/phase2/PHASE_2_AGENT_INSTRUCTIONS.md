# AI Agent Instructions - Phase 2 and Beyond

**Target Audience:** AI Agents (Claude, GPT-4, Codex, or similar)
**Purpose:** Step-by-step guide for autonomous execution of Phase 2+ tasks
**Version:** 1.0
**Date:** 2025-11-12

---

## Overview

You are an AI agent continuing the APX Platform implementation. **Phase 0, Phase 1, Portal M1, and Portal M2 are complete.** You will now execute Phase 2 (Policy Engine) and beyond, following the systematic approach established in earlier phases.

**Your Mission:** Build the policy engine that transforms APX from infrastructure into an intelligent API management platform.

---

## Quick Start (5 Minutes)

### 1. Check Your Context

```bash
cd /Users/agentsy/APILEE

# Read these first:
cat APX_PROJECT_TRACKER.yaml              # Current status
cat PHASE_2_CALIBRATION_SUMMARY.md        # What's next
cat APX_ROADMAP_VISUAL.md                 # Big picture
```

### 2. Understand What's Done ✅

- ✅ **Phase 0:** Foundation complete
- ✅ **Phase 1:** Cloud Run, Router, Workers, Pub/Sub, OTEL all deployed
- ✅ **Portal M1:** Dashboard, API Console, Keys, Orgs working
- ✅ **Portal M2:** Analytics, Request Explorer, SLO, Real-Time features done
- ✅ **Testing:** 100% integration tests passing, 8.7k rps load test

### 3. Understand What's Next ⏳

**Phase 2: Policy Engine (4 weeks, 16 tasks)**

- Week 5: Policy Compiler (OPA + WASM)
- Week 6: N/N-1 Version Support
- Week 7: Canary Rollouts + Auto-Rollback
- Week 8: Testing + Acceptance

**Your First Task:** M2-T1-001 (OPA Integration Setup)

---

## Before You Start

### Required Reading (30 minutes)

1. **[APX_PROJECT_TRACKER.yaml](../../APX_PROJECT_TRACKER.yaml)** (10 min)
   - Find your assigned task
   - Check dependencies
   - Understand acceptance criteria

2. **[PHASE_2_CALIBRATION_SUMMARY.md](../../PHASE_2_CALIBRATION_SUMMARY.md)** (10 min)
   - Phase 2 overview
   - Task details
   - Success criteria

3. **[PRINCIPLES.md](./PRINCIPLES.md)** (10 min)
   - Design principles (edge-thin, async-default, etc.)
   - Why we built it this way

### Environment Verification

```bash
cd /Users/agentsy/APILEE

# Verify GCP access
gcloud config get-value project
# Should be: apx-build-478003

# Verify services are deployed
gcloud run services list --region=us-central1
# Should see: apx-edge, apx-router, apx-workers

# Verify Pub/Sub topics
gcloud pubsub topics list
# Should see: apx-requests-us-dev

# Verify Firestore
gcloud firestore databases describe --database=(default)
# Should be: ACTIVE

# Test local environment (optional)
make up
make status
```

---

## Phase 2 Task Structure

Each Phase 2 task follows this format:

```yaml
- id: "M2-T1-001"
  name: "OPA Integration Setup"
  status: "NOT_STARTED"
  priority: "P0"
  estimated_hours: 4
  dependencies: []
  assigned_to: null

  description: "Integrate Open Policy Agent (OPA) library"

  steps:
    - "Install OPA SDK for Go"
    - "Create policy evaluation service"
    - "Test policy loading and execution"
    - "Verify WASM bundle support"

  acceptance_criteria:
    - checked: false
      text: "OPA library integrated"
    - checked: false
      text: "Can load and execute Rego policies"
    # ... more criteria

  artifacts:
    - "control/pkg/opa/engine.go"
    - "control/pkg/opa/engine_test.go"
```

---

## How to Execute a Phase 2 Task

### Step 1: Claim Your Task

Open `APX_PROJECT_TRACKER.yaml` and find the task section:

```yaml
backend_phase_2_policy_engine:
  week_5_policy_compiler:
    tasks:
      - id: "M2-T1-001"
        name: "OPA Integration Setup"
        status: "NOT_STARTED"  # ← Change this
```

Update to:

```yaml
      - id: "M2-T1-001"
        name: "OPA Integration Setup"
        status: "IN_PROGRESS"
        assigned_to: "your-agent-id"  # e.g., "agent-backend-2"
        started_at: "2025-11-13T09:00:00Z"
```

Commit immediately:

```bash
git add APX_PROJECT_TRACKER.yaml
git commit -m "[M2-T1-001] Claiming OPA Integration task"
git push
```

---

### Step 2: Read the Full Task Definition

Find your task in `APX_PROJECT_TRACKER.yaml`. Read:

- **Description:** What this achieves
- **Steps:** Exact commands/actions
- **Acceptance Criteria:** How you know you're done
- **Dependencies:** What must be complete first
- **Artifacts:** Files you'll create

---

### Step 3: Execute the Task

Follow each step **exactly**. For M2-T1-001 example:

#### Step 1: Install OPA SDK for Go

```bash
cd /Users/agentsy/APILEE
mkdir -p control/pkg/opa
cd control

# Create go.mod if not exists
if [ ! -f go.mod ]; then
  go mod init github.com/apx/control
fi

# Install OPA SDK
go get github.com/open-policy-agent/opa/rego
go get github.com/open-policy-agent/opa/ast
go get github.com/open-policy-agent/opa/types

# Verify installation
go mod tidy
```

#### Step 2: Create Policy Evaluation Service

```bash
cat > pkg/opa/engine.go <<'EOF'
package opa

import (
    "context"
    "fmt"

    "github.com/open-policy-agent/opa/rego"
)

// Engine wraps OPA policy evaluation
type Engine struct {
    query rego.PreparedEvalQuery
}

// NewEngine creates a new OPA evaluation engine
func NewEngine(ctx context.Context, policy string, query string) (*Engine, error) {
    r := rego.New(
        rego.Query(query),
        rego.Module("policy.rego", policy),
    )

    prepared, err := r.PrepareForEval(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to prepare policy: %w", err)
    }

    return &Engine{query: prepared}, nil
}

// Eval evaluates the policy with given input
func (e *Engine) Eval(ctx context.Context, input interface{}) (bool, error) {
    results, err := e.query.Eval(ctx, rego.EvalInput(input))
    if err != nil {
        return false, fmt.Errorf("evaluation failed: %w", err)
    }

    if len(results) == 0 {
        return false, nil
    }

    allowed, ok := results[0].Expressions[0].Value.(bool)
    if !ok {
        return false, fmt.Errorf("unexpected result type")
    }

    return allowed, nil
}
EOF
```

#### Step 3: Write Tests

```bash
cat > pkg/opa/engine_test.go <<'EOF'
package opa

import (
    "context"
    "testing"
)

func TestEngine_Eval(t *testing.T) {
    ctx := context.Background()

    policy := `
package example

allow {
    input.method == "GET"
}
`

    engine, err := NewEngine(ctx, policy, "data.example.allow")
    if err != nil {
        t.Fatalf("NewEngine failed: %v", err)
    }

    tests := []struct {
        name     string
        input    map[string]interface{}
        expected bool
    }{
        {
            name:     "allowed GET request",
            input:    map[string]interface{}{"method": "GET"},
            expected: true,
        },
        {
            name:     "denied POST request",
            input:    map[string]interface{}{"method": "POST"},
            expected: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := engine.Eval(ctx, tt.input)
            if err != nil {
                t.Fatalf("Eval failed: %v", err)
            }
            if result != tt.expected {
                t.Errorf("expected %v, got %v", tt.expected, result)
            }
        })
    }
}
EOF

# Run tests
go test ./pkg/opa/... -v
```

#### Step 4: Verify WASM Support

```bash
cat > pkg/opa/wasm_test.go <<'EOF'
package opa

import (
    "context"
    "testing"

    "github.com/open-policy-agent/opa/compile"
)

func TestWASMCompilation(t *testing.T) {
    policy := `
package example
allow { input.user == "admin" }
`

    compiler := compile.New().
        WithTarget("wasm").
        WithEntrypoints("example/allow")

    result, err := compiler.Build(context.Background())
    if err != nil {
        t.Fatalf("WASM compilation failed: %v", err)
    }

    if result.Bundle == nil {
        t.Fatal("expected bundle, got nil")
    }

    if len(result.Bundle.WasmModules) == 0 {
        t.Fatal("no WASM modules generated")
    }

    t.Logf("WASM module size: %d bytes", len(result.Bundle.WasmModules[0].Raw))
}
EOF

go test ./pkg/opa/... -v -run TestWASM
```

---

### Step 4: Verify Acceptance Criteria

Check **every** criterion:

```bash
# ✅ Criterion 1: OPA library integrated
go list -m github.com/open-policy-agent/opa
# Should show version

# ✅ Criterion 2: Can load and execute Rego policies
go test ./pkg/opa/... -v -run TestEngine_Eval
# Should pass

# ✅ Criterion 3: WASM bundle support verified
go test ./pkg/opa/... -v -run TestWASM
# Should pass

# ✅ Criterion 4: Unit tests passing
go test ./pkg/opa/... -v
# All tests pass
```

---

### Step 5: Update Progress Tracker

Update `APX_PROJECT_TRACKER.yaml`:

```yaml
      - id: "M2-T1-001"
        name: "OPA Integration Setup"
        status: "COMPLETE"  # Changed from IN_PROGRESS
        assigned_to: "agent-backend-2"
        started_at: "2025-11-13T09:00:00Z"
        completed_at: "2025-11-13T13:00:00Z"
        duration_hours: 4

        acceptance_criteria:
          - checked: true  # All changed to true
            text: "OPA library integrated"
          - checked: true
            text: "Can load and execute Rego policies"
          - checked: true
            text: "WASM bundle support verified"
          - checked: true
            text: "Unit tests passing"

        notes:
          - "2025-11-13T09:00:00Z: Started OPA integration"
          - "2025-11-13T10:30:00Z: Engine.go created, basic tests passing"
          - "2025-11-13T12:00:00Z: WASM compilation verified"
          - "2025-11-13T13:00:00Z: All acceptance criteria met"
```

---

### Step 6: Commit Artifacts

```bash
cd /Users/agentsy/APILEE

git add control/pkg/opa/engine.go
git add control/pkg/opa/engine_test.go
git add control/pkg/opa/wasm_test.go
git add control/go.mod
git add control/go.sum
git add APX_PROJECT_TRACKER.yaml

git commit -m "[M2-T1-001] OPA Integration Setup complete

- Installed OPA SDK for Go
- Created policy evaluation engine
- Implemented Engine.Eval() method
- Added comprehensive unit tests
- Verified WASM compilation support

Artifacts:
- control/pkg/opa/engine.go
- control/pkg/opa/engine_test.go
- control/pkg/opa/wasm_test.go

Tests: 5/5 passing
All acceptance criteria met.

Closes M2-T1-001"

git push
```

---

### Step 7: Log Your Work

Update daily logs in `APX_PROJECT_TRACKER.yaml`:

```yaml
daily_logs:
  - date: "2025-11-13"
    entries:
      - timestamp: "2025-11-13T13:00:00Z"
        agent: "agent-backend-2"
        task: "M2-T1-001"
        action: "COMPLETE"
        duration_hours: 4
        summary: |
          OPA Integration Setup complete
          - OPA SDK integrated
          - Policy engine working
          - WASM support verified
          - All tests passing
        notes: "Ready for M2-T1-002 (Policy Compiler)"
```

---

### Step 8: Pick Next Task

Check task dependencies in tracker:

```yaml
      - id: "M2-T1-002"
        name: "Policy Compiler Service"
        dependencies: ["M2-T1-001"]  # ← Now satisfied!
```

If dependencies are met and you're available, claim M2-T1-002.

---

## Code Quality Standards for Phase 2

### Go Code Standards

**Package Structure:**
```
control/
├── cmd/
│   ├── compiler/       # Main compiler service
│   └── monitor/        # Canary health monitor
├── internal/
│   ├── compiler/       # Compiler logic (private)
│   ├── monitor/        # Monitor logic (private)
│   └── alerts/         # Alert notifications
└── pkg/
    ├── opa/            # OPA engine (public)
    ├── artifacts/      # GCS artifact store (public)
    └── canary/         # Canary logic (public)
```

**Naming Conventions:**
```go
// Good ✅
type Engine struct { ... }
func NewEngine(...) (*Engine, error) { ... }
func (e *Engine) Eval(...) (bool, error) { ... }

// Bad ❌
type opa_engine struct { ... }
func new_engine(...) *Engine { ... }
func EvalPolicy(...) bool { ... }  // No error return
```

**Error Handling:**
```go
// Good ✅
result, err := engine.Eval(ctx, input)
if err != nil {
    return nil, fmt.Errorf("evaluation failed: %w", err)
}

// Bad ❌
result := engine.Eval(ctx, input)
if result == nil {
    return nil, errors.New("failed")
}
```

**Testing Standards:**
```go
// Every exported function must have tests
func TestEngine_Eval(t *testing.T) {
    // Arrange
    engine := setupEngine(t)

    // Act
    result, err := engine.Eval(ctx, input)

    // Assert
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if result != expected {
        t.Errorf("expected %v, got %v", expected, result)
    }
}
```

**Test Coverage Target:** >80%

```bash
go test ./... -cover
# Should show > 80% for each package
```

---

## Terraform Standards for Phase 2

### New Infrastructure Needed

**GCS Bucket for Artifacts (M2-T1-003):**

```terraform
# infra/terraform/gcs_artifacts.tf
resource "google_storage_bucket" "artifacts" {
  name          = "${var.project_id}-apx-artifacts"
  location      = var.region
  force_destroy = false  # Protect production artifacts

  versioning {
    enabled = true
  }

  lifecycle_rule {
    condition {
      age = 90
    }
    action {
      type = "Delete"
    }
  }

  uniform_bucket_level_access = true
}

# Service account for compiler
resource "google_service_account" "compiler" {
  account_id   = "apx-compiler"
  display_name = "APX Policy Compiler"
}

resource "google_storage_bucket_iam_member" "compiler_writer" {
  bucket = google_storage_bucket.artifacts.name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.compiler.email}"
}
```

**Cloud Build Trigger (M2-T1-004):**

```terraform
# infra/terraform/cloud_build.tf
resource "google_cloudbuild_trigger" "policy_compiler" {
  name        = "apx-policy-compiler"
  description = "Compile policies on config push"

  github {
    owner = var.github_owner
    name  = var.github_repo
    push {
      branch = "^main$"
    }
  }

  included_files = ["configs/samples/**/*.yaml"]

  filename = "cloudbuild.yaml"
}
```

**Always:**
1. Run `terraform plan` first
2. Review output carefully
3. Run `terraform apply`
4. Verify resources created

---

## Integration Testing for Phase 2

### Week 8 Testing Requirements

**Canary Rollout Test (M2-T4-001):**

```bash
#!/bin/bash
# tests/integration/canary_rollout_test.sh

set -e

echo "=== Canary Rollout Integration Test ==="

# Deploy stable policy v1.2.0
echo "1. Deploying stable policy v1.2.0..."
apx deploy policy@1.2.0

# Compile new policy v1.3.0
echo "2. Compiling canary policy v1.3.0..."
apx compile configs/samples/payments-api-v1.3.yaml

# Start canary at 5%
echo "3. Starting canary at 5%..."
apx rollout --canary 5% policy@1.3.0

# Send 1000 requests
echo "4. Sending 1000 test requests..."
for i in {1..1000}; do
  curl -s http://localhost:8081/v1/test \
    -H "X-Tenant-ID: test-$((i % 100))" \
    > /dev/null
done

# Check distribution
echo "5. Checking version distribution..."
v1_2_count=$(bq query --format=csv --use_legacy_sql=false \
  "SELECT COUNT(*) FROM apx_requests WHERE policy_version='1.2.0' AND timestamp > TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 1 MINUTE)")

v1_3_count=$(bq query --format=csv --use_legacy_sql=false \
  "SELECT COUNT(*) FROM apx_requests WHERE policy_version='1.3.0' AND timestamp > TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 1 MINUTE)")

# Calculate percentage
v1_3_pct=$((v1_3_count * 100 / (v1_2_count + v1_3_count)))

echo "v1.3 traffic: ${v1_3_pct}%"

# Verify within tolerance (5% ± 2%)
if [ $v1_3_pct -lt 3 ] || [ $v1_3_pct -gt 7 ]; then
  echo "❌ FAIL: Expected ~5%, got ${v1_3_pct}%"
  exit 1
fi

echo "✅ PASS: Canary rollout working correctly"
```

**Auto-Rollback Test (M2-T4-002):**

```bash
#!/bin/bash
# tests/integration/auto_rollback_test.sh

set -e

echo "=== Auto-Rollback Integration Test ==="

# Deploy stable policy
echo "1. Deploying stable policy..."
apx deploy policy@1.2.0

# Create breaking policy (rejects all)
echo "2. Creating breaking policy..."
cat > /tmp/breaking-policy.yaml <<EOF
name: "breaking-policy"
version: "1.3.0"
compat: "breaking"
authz_rego: |
  package authz
  allow { false }  # Reject everything
EOF

apx compile /tmp/breaking-policy.yaml

# Start canary at 10%
echo "3. Starting canary at 10%..."
apx rollout --canary 10% policy@1.3.0

# Wait for monitor to detect (5 minutes)
echo "4. Waiting 5 minutes for monitor to detect..."
sleep 300

# Check canary status
echo "5. Checking canary status..."
status=$(apx status policy@1.3.0 --format=json | jq -r '.canary_percentage')

if [ "$status" != "0" ]; then
  echo "❌ FAIL: Canary not rolled back (still at ${status}%)"
  exit 1
fi

echo "✅ PASS: Auto-rollback working correctly"
```

---

## Handling Blockers

### Common Issues in Phase 2

#### Issue 1: OPA Compilation Errors

**Symptom:** WASM compilation fails

**Debug:**
```bash
# Validate Rego syntax
opa check policy.rego

# Test policy locally
opa eval -d policy.rego "data.example.allow"
```

**Solution:** Fix Rego syntax errors, ensure package names match

---

#### Issue 2: GCS Upload Fails

**Symptom:** Cannot upload artifacts to GCS

**Debug:**
```bash
# Check service account permissions
gcloud storage buckets get-iam-policy gs://apx-artifacts-$PROJECT_ID

# Test upload manually
echo "test" | gsutil cp - gs://apx-artifacts-$PROJECT_ID/test.txt
```

**Solution:** Add storage.objectAdmin role to compiler service account

---

#### Issue 3: Canary Traffic Not Splitting

**Symptom:** All traffic goes to stable version

**Debug:**
```bash
# Check Firestore canary config
gcloud firestore documents list canary_configs --project=$PROJECT_ID

# Check router logs
gcloud logging read "resource.type=cloud_run_revision AND resource.labels.service_name=apx-router" --limit 50
```

**Solution:** Verify canary config saved correctly, check hashing logic

---

### Escalation Protocol

If blocked for >2 hours:

1. **Update tracker:**
   ```yaml
   status: "BLOCKED"
   blocker_type: "TECHNICAL"
   blocked_since: "2025-11-13T14:00:00Z"
   blocker_description: "OPA compilation fails with: ..."
   ```

2. **Document what you tried:**
   ```yaml
   troubleshooting_steps:
     - "Checked Rego syntax with opa check"
     - "Tested policy locally - works"
     - "Tried in Go - fails with error X"
   ```

3. **Escalate to coordinator**

---

## Parallel Execution Guidelines

### Week 5 Parallelization

**Safe Parallel Tasks:**

- **agent-backend-2:** M2-T1-001 → M2-T1-002
- **agent-infrastructure-1:** (Wait for M2-T1-002) → M2-T1-003 → M2-T1-004

**Why:** M2-T1-003 depends on M2-T1-002 completing

### Week 6 Parallelization

**All tasks can run in parallel:**

- **agent-backend-1:** M2-T2-001 (Router)
- **agent-backend-2:** M2-T2-002 (Workers)
- **agent-infrastructure-1:** M2-T2-003 (Firestore)

**Why:** All depend on M2-T1-003 but not each other

---

## Success Metrics

### Task-Level Metrics

**Every task must achieve:**
- ✅ All acceptance criteria checked: true
- ✅ All tests passing
- ✅ Code formatted (go fmt, terraform fmt)
- ✅ No linter errors
- ✅ Documentation updated

### Phase-Level Metrics

**Phase 2 success criteria:**
- ✅ All 16 tasks complete
- ✅ Canary rollout works (5% → 100%)
- ✅ Auto-rollback works (<2 min)
- ✅ Zero dropped requests during rollback
- ✅ GitOps pipeline functional
- ✅ CLI tools working
- ✅ Integration tests 100% passing

---

## Communication

### Daily Standups (Async)

Post to `APX_PROJECT_TRACKER.yaml` daily logs:

```yaml
daily_logs:
  - date: "2025-11-13"
    entries:
      - timestamp: "2025-11-13T18:00:00Z"
        agent: "agent-backend-2"
        summary: |
          Completed: M2-T1-001 (OPA Integration)
          In Progress: M2-T1-002 (Policy Compiler - 60% done)
          Planned Tomorrow: Complete M2-T1-002, start M2-T1-003 if time
          Blockers: None
        notes: "OPA integration smooth, compiler on track"
```

### Code Reviews

Before marking COMPLETE, self-review:

**Checklist:**
- [ ] Code follows Go conventions
- [ ] Error handling present
- [ ] Tests cover edge cases
- [ ] Comments explain why, not what
- [ ] No hardcoded secrets
- [ ] Terraform validated
- [ ] All acceptance criteria met

---

## Phase 2 Week-by-Week Guide

### Week 5 Focus: Policy Compiler

**Goal:** Compile PolicyBundle YAML → WASM

**Key Agents:**
- agent-backend-2: OPA + Compiler
- agent-infrastructure-1: GCS + GitOps

**Critical Path:** M2-T1-001 → M2-T1-002 → M2-T1-003 → M2-T1-004

**Success:** Can compile YAML → WASM and store in GCS

---

### Week 6 Focus: Version Support

**Goal:** Router and Workers support N and N-1

**Key Agents:**
- agent-backend-1: Router
- agent-backend-2: Workers
- agent-infrastructure-1: Firestore

**Parallelizable:** All 3 tasks can run simultaneously

**Success:** x-apx-policy-version header works

---

### Week 7 Focus: Canary Rollouts

**Goal:** Traffic splitting + auto-rollback

**Key Agents:**
- agent-backend-1: Traffic splitting + CLI
- agent-backend-2: Auto-rollback monitor

**Critical Path:** M2-T3-001 → M2-T3-002 → M2-T3-003

**Success:** Can rollout canary, auto-rollback works

---

### Week 8 Focus: Testing

**Goal:** Comprehensive testing

**Key Agents:**
- agent-backend-1 + agent-backend-2: Pair on testing

**All Sequential:** Tests build on each other

**Success:** All acceptance tests pass

---

## Artifacts Inventory

### Files You'll Create in Phase 2

```
control/
├── cmd/
│   ├── compiler/
│   │   └── main.go                    # M2-T1-002
│   └── monitor/
│       └── main.go                    # M2-T3-002
├── internal/
│   ├── compiler/
│   │   ├── parser.go                  # M2-T1-002
│   │   └── generator.go               # M2-T1-002
│   ├── monitor/
│   │   └── canary_health.go           # M2-T3-002
│   └── alerts/
│       └── notifier.go                # M2-T3-002
└── pkg/
    ├── opa/
    │   ├── engine.go                  # M2-T1-001
    │   ├── engine_test.go             # M2-T1-001
    │   └── wasm_test.go               # M2-T1-001
    ├── artifacts/
    │   ├── store.go                   # M2-T1-003
    │   ├── store_test.go              # M2-T1-003
    │   └── hasher.go                  # M2-T1-002
    └── canary/
        ├── splitter.go                # M2-T3-001
        └── hasher.go                  # M2-T3-001

router/internal/middleware/
├── canary.go                          # M2-T3-001
└── policy_version.go (enhanced)       # M2-T2-001

workers/internal/policy/
├── cache.go                           # M2-T2-002
├── loader.go                          # M2-T2-002
└── executor.go                        # M2-T2-002

cli/
├── cmd/
│   ├── rollout.go                     # M2-T3-003
│   ├── rollback.go                    # M2-T3-003
│   └── status.go                      # M2-T3-003
└── main.go                            # M2-T3-003

infra/terraform/
├── gcs_artifacts.tf                   # M2-T1-003
├── cloud_build.tf                     # M2-T1-004
└── firestore_policies.tf              # M2-T2-003

tests/integration/
├── canary_rollout_test.sh             # M2-T4-001
├── auto_rollback_test.sh              # M2-T4-002
└── policy_version_test.sh             # M2-T4-003

cloudbuild.yaml                        # M2-T1-004

docs/
└── PHASE_2_ACCEPTANCE_REPORT.md       # M2-T4-004
```

---

## Example Full Session: M2-T1-002 (Policy Compiler)

### Agent: agent-backend-2
### Date: 2025-11-14
### Task: Build Policy Compiler Service

---

#### Step 1: Claim Task

```bash
cd /Users/agentsy/APILEE
vim APX_PROJECT_TRACKER.yaml
# Update M2-T1-002 status to IN_PROGRESS

git add APX_PROJECT_TRACKER.yaml
git commit -m "[M2-T1-002] Claiming Policy Compiler task"
git push
```

---

#### Step 2: Read Task Definition

```bash
cat APX_PROJECT_TRACKER.yaml | grep -A 30 "M2-T1-002"
```

**Task:** Build service to compile PolicyBundle YAML → WASM

**Dependencies:** M2-T1-001 (OPA Integration) ✅ Complete

**Estimated:** 8 hours

---

#### Step 3: Create Compiler Package

```bash
cd /Users/agentsy/APILEE/control
mkdir -p internal/compiler
cd internal/compiler

# Create parser.go
cat > parser.go <<'EOF'
package compiler

import (
    "fmt"
    "gopkg.in/yaml.v3"
)

// PolicyBundle represents a policy configuration
type PolicyBundle struct {
    Name        string `yaml:"name"`
    Version     string `yaml:"version"`
    Compat      string `yaml:"compat"`
    AuthzRego   string `yaml:"authz_rego"`
    Quotas      map[string]int `yaml:"quotas"`
    RateLimit   map[string]int `yaml:"rate_limit"`
}

// ParseYAML parses PolicyBundle from YAML
func ParseYAML(data []byte) (*PolicyBundle, error) {
    var bundle PolicyBundle
    if err := yaml.Unmarshal(data, &bundle); err != nil {
        return nil, fmt.Errorf("failed to parse YAML: %w", err)
    }

    // Validate required fields
    if bundle.Name == "" {
        return nil, fmt.Errorf("name is required")
    }
    if bundle.Version == "" {
        return nil, fmt.Errorf("version is required")
    }
    if bundle.AuthzRego == "" {
        return nil, fmt.Errorf("authz_rego is required")
    }

    return &bundle, nil
}
EOF

# Create generator.go
cat > generator.go <<'EOF'
package compiler

import (
    "context"
    "crypto/sha256"
    "fmt"

    "github.com/open-policy-agent/opa/compile"
)

// Generator compiles Rego to WASM
type Generator struct {
    bundle *PolicyBundle
}

// NewGenerator creates a new generator
func NewGenerator(bundle *PolicyBundle) *Generator {
    return &Generator{bundle: bundle}
}

// Compile compiles the policy to WASM
func (g *Generator) Compile(ctx context.Context) (*Artifact, error) {
    // Build OPA compiler
    compiler := compile.New().
        WithTarget("wasm").
        WithEntrypoints("authz/allow")

    // Add policy module
    module := fmt.Sprintf("package authz\n%s", g.bundle.AuthzRego)
    compiler.WithModule("policy.rego", module)

    // Compile
    result, err := compiler.Build(ctx)
    if err != nil {
        return nil, fmt.Errorf("compilation failed: %w", err)
    }

    if len(result.Bundle.WasmModules) == 0 {
        return nil, fmt.Errorf("no WASM modules generated")
    }

    wasmBytes := result.Bundle.WasmModules[0].Raw

    // Calculate hash
    hash := sha256.Sum256(wasmBytes)
    hashStr := fmt.Sprintf("%x", hash)

    return &Artifact{
        Name:    g.bundle.Name,
        Version: g.bundle.Version,
        Hash:    hashStr,
        WASM:    wasmBytes,
    }, nil
}

// Artifact represents a compiled artifact
type Artifact struct {
    Name    string
    Version string
    Hash    string
    WASM    []byte
}
EOF
```

---

#### Step 4: Create REST API

```bash
cd /Users/agentsy/APILEE/control
mkdir -p cmd/compiler
cd cmd/compiler

cat > main.go <<'EOF'
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"

    "github.com/apx/control/internal/compiler"
    "github.com/apx/control/pkg/artifacts"
)

func main() {
    store := artifacts.NewGCSStore()

    http.HandleFunc("/compile", func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
            http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
            return
        }

        // Read YAML
        data, err := io.ReadAll(r.Body)
        if err != nil {
            http.Error(w, "Failed to read body", http.StatusBadRequest)
            return
        }

        // Parse
        bundle, err := compiler.ParseYAML(data)
        if err != nil {
            http.Error(w, fmt.Sprintf("Parse error: %v", err), http.StatusBadRequest)
            return
        }

        // Compile
        gen := compiler.NewGenerator(bundle)
        artifact, err := gen.Compile(context.Background())
        if err != nil {
            http.Error(w, fmt.Sprintf("Compile error: %v", err), http.StatusInternalServerError)
            return
        }

        // Upload to GCS
        url, err := store.Upload(context.Background(), artifact)
        if err != nil {
            http.Error(w, fmt.Sprintf("Upload error: %v", err), http.StatusInternalServerError)
            return
        }

        // Return response
        response := map[string]string{
            "name":    artifact.Name,
            "version": artifact.Version,
            "hash":    artifact.Hash,
            "url":     url,
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
    })

    log.Println("Compiler listening on :8082")
    log.Fatal(http.ListenAndServe(":8082", nil))
}
EOF
```

---

#### Step 5: Write Tests

```bash
cd /Users/agentsy/APILEE/control/internal/compiler

cat > parser_test.go <<'EOF'
package compiler

import (
    "testing"
)

func TestParseYAML(t *testing.T) {
    yaml := `
name: "test-policy"
version: "1.0.0"
compat: "backward"
authz_rego: |
  allow { input.user == "admin" }
quotas:
  requests_per_day: 1000
rate_limit:
  per_second: 10
`

    bundle, err := ParseYAML([]byte(yaml))
    if err != nil {
        t.Fatalf("ParseYAML failed: %v", err)
    }

    if bundle.Name != "test-policy" {
        t.Errorf("expected name 'test-policy', got '%s'", bundle.Name)
    }

    if bundle.Version != "1.0.0" {
        t.Errorf("expected version '1.0.0', got '%s'", bundle.Version)
    }
}

func TestParseYAML_MissingName(t *testing.T) {
    yaml := `
version: "1.0.0"
authz_rego: "allow { true }"
`

    _, err := ParseYAML([]byte(yaml))
    if err == nil {
        t.Fatal("expected error for missing name")
    }
}
EOF

cat > generator_test.go <<'EOF'
package compiler

import (
    "context"
    "testing"
)

func TestGenerator_Compile(t *testing.T) {
    bundle := &PolicyBundle{
        Name:      "test-policy",
        Version:   "1.0.0",
        AuthzRego: "allow { input.user == \"admin\" }",
    }

    gen := NewGenerator(bundle)
    artifact, err := gen.Compile(context.Background())
    if err != nil {
        t.Fatalf("Compile failed: %v", err)
    }

    if artifact.Name != "test-policy" {
        t.Errorf("expected name 'test-policy', got '%s'", artifact.Name)
    }

    if len(artifact.WASM) == 0 {
        t.Error("expected WASM bytes, got empty")
    }

    if artifact.Hash == "" {
        t.Error("expected hash, got empty")
    }

    t.Logf("Compiled artifact: %d bytes, hash: %s", len(artifact.WASM), artifact.Hash)
}
EOF

# Run tests
cd /Users/agentsy/APILEE/control
go test ./internal/compiler/... -v
```

---

#### Step 6: Verify Acceptance Criteria

```bash
# ✅ Compiler accepts PolicyBundle YAML
go test ./internal/compiler/... -v -run TestParseYAML
# PASS

# ✅ Outputs WASM bundle with hash
go test ./internal/compiler/... -v -run TestGenerator_Compile
# PASS

# ✅ Semantic versioning working
# (Validated in parser tests)

# ✅ REST API functional
go run cmd/compiler/main.go &
COMPILER_PID=$!

# Test API
curl -X POST http://localhost:8082/compile \
  -H "Content-Type: application/yaml" \
  -d "name: test
version: 1.0.0
authz_rego: 'allow { true }'"

kill $COMPILER_PID
# PASS

# ✅ Integration tests passing
go test ./... -v
# All PASS
```

---

#### Step 7: Update Tracker

```yaml
      - id: "M2-T1-002"
        name: "Policy Compiler Service"
        status: "COMPLETE"
        completed_at: "2025-11-14T17:00:00Z"
        duration_hours: 8

        acceptance_criteria:
          - checked: true
            text: "Compiler accepts PolicyBundle YAML"
          - checked: true
            text: "Outputs WASM bundle with hash"
          - checked: true
            text: "Semantic versioning working"
          - checked: true
            text: "REST API functional"
          - checked: true
            text: "Integration tests passing"
```

---

#### Step 8: Commit

```bash
git add control/internal/compiler/*.go
git add control/cmd/compiler/main.go
git add APX_PROJECT_TRACKER.yaml

git commit -m "[M2-T1-002] Policy Compiler Service complete

- Implemented YAML parser with validation
- Created WASM generator using OPA compile
- Built REST API: POST /compile
- SHA256 hashing for artifacts
- Comprehensive test coverage

Artifacts:
- control/internal/compiler/parser.go
- control/internal/compiler/generator.go
- control/cmd/compiler/main.go

Tests: 8/8 passing
Coverage: 87%

Closes M2-T1-002"

git push
```

---

## Final Checklist

Before starting Phase 2 execution:

- [ ] Read all 3 agent instruction docs
- [ ] Understand current state (Phase 0, 1, Portal M1/M2 complete)
- [ ] Review Phase 2 tasks in tracker
- [ ] Verify GCP environment accessible
- [ ] Understand dependency graph
- [ ] Know where to escalate blockers
- [ ] Commit to daily log updates

---

## You're Ready! 🚀

**Next Steps:**
1. Choose your agent type (backend/infrastructure)
2. Claim first available task
3. Follow this guide step-by-step
4. Update tracker daily
5. Build amazing things!

**Remember:**
- Quality over speed
- Test everything
- Document as you go
- Ask for help when blocked
- Celebrate progress!

**Let's build the future of API management together! 💪**

---

**Document Version:** 1.0
**Last Updated:** 2025-11-12
**Maintained by:** APX Platform Team
**Questions:** Update APX_PROJECT_TRACKER.yaml or escalate to coordinator
