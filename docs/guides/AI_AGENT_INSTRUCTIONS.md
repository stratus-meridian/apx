# AI Agent Instructions for APX Implementation

**Target Audience:** AI Agents (Claude, GPT-4, Codex, or similar)
**Purpose:** Step-by-step guide for autonomous or semi-autonomous task execution
**Version:** 1.0

---

## Overview

You are an AI agent working on the APX Platform - a next-generation API management system. Your role is to execute tasks from the [AGENT_EXECUTION_PLAN.md](./AGENT_EXECUTION_PLAN.md) systematically and update progress in [TASK_TRACKER.yaml](../TASK_TRACKER.yaml).

---

## Before You Start

### 1. Read These Documents (Priority Order)

1. **[GETTING_STARTED.md](../GETTING_STARTED.md)** (5 min) - Understand the project
2. **[PRINCIPLES.md](./PRINCIPLES.md)** (15 min) - Learn the design principles
3. **[AGENT_EXECUTION_PLAN.md](./AGENT_EXECUTION_PLAN.md)** (30 min) - Your task blueprint
4. **[TASK_TRACKER.yaml](../TASK_TRACKER.yaml)** (5 min) - Current status

### 2. Understand Your Environment

```bash
# You are working in:
/Users/agentsy/APILEE

# Key directories:
- edge/          # Envoy gateway
- router/        # Go routing service
- workers/       # Worker pools
- control/       # Policy compiler
- agents/        # AI agents (your siblings!)
- infra/         # Terraform configs
- configs/       # YAML schemas
- docs/          # Documentation
```

### 3. Set Up Your Local Environment

```bash
cd /Users/agentsy/APILEE

# Copy environment template
cp .env.example .env

# Edit .env with GCP project ID (ask human coordinator if unknown)
# Then:
make init
make up

# Verify services are running:
make status
```

---

## How to Execute a Task

### Step 1: Claim a Task

1. Open [TASK_TRACKER.yaml](../TASK_TRACKER.yaml)
2. Find a task with:
   - `status: NOT_STARTED`
   - All `dependencies: []` are `COMPLETE`
   - `priority: P0` (highest priority first)

3. Update the task:
   ```yaml
   M1-T0-001:
     status: "IN_PROGRESS"
     assigned_to: "your-agent-id"  # e.g., "agent-backend-1" or "claude-session-abc123"
     started_at: "2025-11-11T10:00:00Z"  # Current timestamp
   ```

4. Update the agent status:
   ```yaml
   agents:
     - id: "your-agent-id"
       status: "busy"
       current_task: "M1-T0-001"
   ```

5. **Commit this change** before starting work:
   ```bash
   git add TASK_TRACKER.yaml
   git commit -m "[M1-T0-001] Claiming task"
   git push
   ```

### Step 2: Read the Task Definition

Open [AGENT_EXECUTION_PLAN.md](./AGENT_EXECUTION_PLAN.md) and find your task (e.g., `Task M1-T0-001`).

Read:
- **Context**: Why this task matters
- **Prerequisites**: What must exist before you start
- **Steps**: Exact commands to run
- **Acceptance Criteria**: How to know you're done
- **Artifacts**: What files you'll create/modify

### Step 3: Execute the Task

Follow the **Steps** section **exactly**.

**Important Rules:**

1. **Run commands as written** - Don't modify unless you understand the architecture
2. **Verify each step** - Check output before proceeding
3. **Document issues** - If a command fails, add to `notes` in TASK_TRACKER.yaml
4. **Ask for help if blocked** - Update task status to `BLOCKED` and notify coordinator

**Example Execution (M1-T0-001):**

```bash
# Step 1: Create GCP project
export PROJECT_ID="apx-dev-$(openssl rand -hex 4)"
gcloud projects create $PROJECT_ID --name="APX Development"

# Verify:
gcloud projects describe $PROJECT_ID
# ✅ If this succeeds, continue. ❌ If it fails, update notes and ask for help.

# Step 2: Link billing
gcloud billing accounts list
export BILLING_ACCOUNT="<account-id>"  # Copy from output above
gcloud billing projects link $PROJECT_ID --billing-account=$BILLING_ACCOUNT

# Step 3: Enable APIs
gcloud services enable \
  compute.googleapis.com \
  run.googleapis.com \
  # ... (rest of APIs from task definition)

# Step 4: Wait for APIs to enable
sleep 180
gcloud services list --enabled | grep compute

# Step 5: Update .env
cd /Users/agentsy/APILEE
echo "GCP_PROJECT_ID=$PROJECT_ID" >> .env
```

### Step 4: Verify Acceptance Criteria

After executing all steps, verify **every** item in the acceptance criteria:

```yaml
acceptance_criteria:
  - checked: true   # ← Update to true when verified
    text: "Project created and billing linked"
  - checked: true
    text: "All 15 APIs enabled and verified"
  # etc.
```

**Run the verification commands:**

```bash
# From task definition:
gcloud projects describe $PROJECT_ID  # ✅ Should succeed
gcloud services list --enabled        # ✅ Should show all 15 APIs
cat .env | grep GCP_PROJECT_ID        # ✅ Should show correct ID
```

### Step 5: Update TASK_TRACKER.yaml

```yaml
M1-T0-001:
  status: "COMPLETE"  # Changed from IN_PROGRESS
  completed_at: "2025-11-11T12:30:00Z"
  artifacts:
    - ".env: Updated with GCP_PROJECT_ID"
    - "infra/terraform/backend.tf: Created (in next task)"
  notes:
    - "2025-11-11T10:00:00Z: Started task"
    - "2025-11-11T10:15:00Z: APIs enabled"
    - "2025-11-11T12:30:00Z: All acceptance criteria met"
  acceptance_criteria:
    - checked: true
      text: "Project created and billing linked"
    - checked: true
      text: "All 15 APIs enabled and verified"
    - checked: true
      text: ".env file contains correct PROJECT_ID"
    - checked: true
      text: "Can run: gcloud projects describe $PROJECT_ID"

agents:
  - id: "your-agent-id"
    status: "available"  # Changed from busy
    current_task: null
    tasks_completed: 1    # Incremented

progress:
  milestone_1:
    tasks_complete: 1     # Incremented
    completion: 4%        # 1/25 = 4%
```

### Step 6: Commit and Push

```bash
git add TASK_TRACKER.yaml
git add .env  # If you created artifacts

git commit -m "[M1-T0-001] GCP Project Initialization complete

- Created project: $PROJECT_ID
- Enabled 15 APIs
- Updated .env with project ID

All acceptance criteria met."

git push
```

### Step 7: Log Your Work

Append to the daily log in TASK_TRACKER.yaml:

```yaml
daily_logs:
  - date: "2025-11-11"
    entries:
      - timestamp: "2025-11-11T12:30:00Z"
        agent: "your-agent-id"
        task: "M1-T0-001"
        action: "COMPLETE"
        duration_hours: 2.5
        notes: "GCP project initialized successfully. No issues."
```

### Step 8: Pick Next Task

If your dependencies are met, claim the next task (e.g., M1-T0-002).

Otherwise, check if other P0 tasks are available.

---

## Handling Errors

### When a Command Fails

1. **Don't panic** - Errors are expected
2. **Read the error message carefully**
3. **Check prerequisites** - Did you skip a dependency?
4. **Try the rollback** - Task definition includes rollback commands
5. **Update task status:**

```yaml
M1-T0-001:
  status: "BLOCKED"
  notes:
    - "2025-11-11T11:00:00Z: API enablement failed with: 'billing not enabled'"
    - "2025-11-11T11:05:00Z: Attempting manual billing link via console"
```

6. **Add to blockers:**

```yaml
blockers:
  - task: "M1-T0-001"
    blocker_type: "COMMAND_FAILURE"
    description: "gcloud services enable failed - billing account not linked"
    blocked_since: "2025-11-11T11:00:00Z"
    assigned_to: "human-coordinator"  # Escalate if you can't resolve
    resolution: null
```

7. **Notify coordinator** (create GitHub issue or Slack message)

### When You're Blocked by Another Task

If your task depends on `M1-T0-003` but it's not complete:

1. Check if the dependency is `IN_PROGRESS` - wait for it
2. If dependency is `NOT_STARTED` - notify coordinator (or claim it yourself if you can)
3. If dependency is `BLOCKED` - find another task or help unblock it

---

## Code Quality Standards

When writing code (Go, Python, TypeScript):

### 1. Follow Existing Patterns

```bash
# Before writing new code, read existing code:
cat router/internal/middleware/tenant.go

# Match the style:
# - Error handling: if err != nil { return err }
# - Logging: logger.Info("message", zap.String("key", value))
# - Comments: Brief, explain why not what
```

### 2. Write Tests

```go
// If you write router/internal/routes/matcher.go
// Also write router/internal/routes/matcher_test.go

func TestMatcher_Handle(t *testing.T) {
  // Arrange
  // Act
  // Assert
}
```

### 3. Run Tests Before Committing

```bash
cd router
go test ./...

# All tests must pass before marking task COMPLETE
```

### 4. Format Code

```bash
# Go
go fmt ./...

# Python
black .

# TypeScript
npm run lint:fix
```

---

## Terraform Best Practices

### 1. Always Plan First

```bash
cd infra/terraform
terraform plan

# Review changes carefully
# Look for: destroys, replacements, unexpected modifications
```

### 2. Use Targeted Applies for Safety

```bash
# Instead of:
terraform apply -auto-approve  # ❌ Risky

# Do:
terraform plan -out=plan.tfplan
terraform show plan.tfplan     # Review
terraform apply plan.tfplan    # Safe
```

### 3. Validate Before Apply

```bash
terraform validate
terraform fmt -check
```

### 4. State Management

```bash
# Never edit state files manually
# If state is corrupted, ask for help

# Safe operations:
terraform state list
terraform state show <resource>
```

---

## Communication Protocol

### Daily Standups (Async)

At the end of each work session, post to TASK_TRACKER.yaml:

```yaml
daily_logs:
  - date: "2025-11-12"
    entries:
      - timestamp: "2025-11-12T18:00:00Z"
        agent: "your-agent-id"
        summary: |
          Completed: M1-T0-001, M1-T0-002
          In Progress: M1-T0-003 (50% complete, blocked on billing approval)
          Planned Tomorrow: M1-T0-004 (if M1-T0-003 unblocked)
          Blockers: Waiting for human to approve billing account link
```

### Asking for Help

1. **Check documentation first**:
   - [GETTING_STARTED.md](../GETTING_STARTED.md)
   - [PRINCIPLES.md](./PRINCIPLES.md)
   - [GAPS_AND_REGRETS.md](./GAPS_AND_REGRETS.md)

2. **Search for similar issues**:
   ```bash
   # Search codebase
   grep -r "similar error" .

   # Check git history
   git log --all --grep="relevant keyword"
   ```

3. **Create a detailed issue**:

```markdown
## Task: M1-T0-003 Service Accounts and IAM
## Agent: agent-infrastructure-1
## Status: BLOCKED

### Problem
Terraform apply fails with:
```
Error: Error creating service account: googleapi: Error 403: Caller does not have permission 'iam.serviceAccounts.create'
```

### What I've Tried
1. Verified IAM API is enabled: ✅
2. Checked my user permissions: ❌ Don't have iam.serviceAccountAdmin role
3. Attempted workaround: N/A

### Expected Behavior
Service account should be created

### Request
Please grant iam.serviceAccountAdmin role to my GCP user account

### Impact
Blocking M1-T0-003, M1-T1-001, M1-T1-002 (dependencies)
```

---

## Advanced: Parallel Execution

Multiple agents can work simultaneously on independent tasks.

### Safe Parallelization

```yaml
# Agent 1 can work on:
M1-T0-001: GCP Project Init (no dependencies)

# Agent 2 can simultaneously work on:
M1-T1-002: Router Implementation (depends on M1-T0-003, but can prep code)

# Agent 3 can work on:
M1-T3-001: OTEL Collector Setup (depends on M1-T1-004, but can prep configs)
```

### Merge Conflict Prevention

1. **Claim tasks in different files**:
   - Agent 1: infra/terraform/iam.tf
   - Agent 2: router/internal/routes/matcher.go
   - Agent 3: observability/otel/config.yaml

2. **If editing same file**:
   - Coordinate in TASK_TRACKER.yaml notes
   - Work in separate branches
   - Merge sequentially

3. **Always pull before push**:
   ```bash
   git pull --rebase
   git push
   ```

---

## Testing Your Work

### Unit Tests

```bash
# Go
cd router
go test -v ./internal/middleware/

# Python
cd workers
pytest tests/

# TypeScript
cd portal
npm test
```

### Integration Tests

```bash
# Start local stack
make up

# Run integration tests
make test-integration

# Check logs
make logs
```

### Manual Testing

```bash
# Test edge health
curl http://localhost:8080/health

# Test router
curl http://localhost:8081/health

# Test full flow
curl -X POST http://localhost:8080/v1/test \
  -H "Content-Type: application/json" \
  -d '{"test": "data"}'
```

---

## Rollback Procedures

If you break something:

### 1. Immediate Rollback (Code)

```bash
# Revert your commit
git revert HEAD
git push

# Or reset (if not pushed yet)
git reset --hard HEAD~1
```

### 2. Terraform Rollback

```bash
cd infra/terraform

# Destroy specific resource
terraform destroy -target=google_cloud_run_service.edge

# Or rollback to previous state
terraform state pull > backup.tfstate
# Manually restore previous state (ask for help)
```

### 3. Cloud Run Rollback

```bash
# Rollback to previous revision
gcloud run services update-traffic apx-edge \
  --to-revisions=apx-edge-00001-abc=100 \
  --region=us-central1
```

### 4. Update Task Status

```yaml
M1-T1-001:
  status: "ROLLED_BACK"
  notes:
    - "2025-11-11T14:00:00Z: Deployment caused 50% error rate"
    - "2025-11-11T14:05:00Z: Rolled back to previous revision"
    - "2025-11-11T14:10:00Z: Error rate back to normal"
  rollback_reason: "Edge gateway returning 500 errors due to incorrect Envoy config"
```

---

## Metrics to Track

Update these in TASK_TRACKER.yaml after each task:

```yaml
metrics:
  tasks_per_day: 3              # Your velocity
  average_task_duration_hours: 2.5
  blockers_resolved_per_day: 1
  tests_passing: 25
  tests_total: 25
  deployments_successful: 3
  deployments_total: 3
```

---

## Success Criteria

You're doing well if:

- ✅ Tasks complete with all acceptance criteria checked
- ✅ No broken tests in main branch
- ✅ Terraform applies cleanly
- ✅ Deployments succeed on first try (or second after fixing minor issues)
- ✅ Code follows existing patterns
- ✅ Documentation updated alongside code
- ✅ Daily logs show steady progress
- ✅ Blockers are identified and escalated quickly

You need help if:

- ❌ Multiple tasks marked BLOCKED
- ❌ Tests failing for >1 hour
- ❌ Same error appearing repeatedly
- ❌ Uncertain about architectural decisions
- ❌ Terraform destroying resources unexpectedly
- ❌ No progress for >4 hours on a single task

---

## Example Full Session

**Agent:** agent-backend-1
**Date:** 2025-11-12
**Goal:** Complete M1-T1-002 (Router Implementation)

### Session Start

```bash
# 1. Pull latest
cd /Users/agentsy/APILEE
git pull

# 2. Check available tasks
cat TASK_TRACKER.yaml | grep "M1-T1-002"
# Status: NOT_STARTED, Dependencies: M1-T0-003, M1-T0-005, M1-T0-006

# 3. Verify dependencies
cat TASK_TRACKER.yaml | grep "M1-T0-003"
# Status: COMPLETE ✅

cat TASK_TRACKER.yaml | grep "M1-T0-005"
# Status: COMPLETE ✅

cat TASK_TRACKER.yaml | grep "M1-T0-006"
# Status: COMPLETE ✅

# 4. Claim task
# Edit TASK_TRACKER.yaml:
# M1-T1-002:
#   status: IN_PROGRESS
#   assigned_to: agent-backend-1
#   started_at: 2025-11-12T09:00:00Z

git add TASK_TRACKER.yaml
git commit -m "[M1-T1-002] Claiming router implementation task"
git push
```

### Execution

```bash
# 5. Read task definition
cat docs/AGENT_EXECUTION_PLAN.md | grep -A 200 "Task M1-T1-002"

# 6. Execute steps 1-4 from task definition
# (Create middleware files, implement route matcher, etc.)

# 7. Test locally
cd router
go test ./...
# ✅ All tests pass

go build -o bin/router cmd/router/main.go
# ✅ Build succeeds

# 8. Start local services
cd ..
make up

# 9. Test health check
curl http://localhost:8081/health
# ✅ {"status":"ok","service":"apx-router"}

# 10. Test request
curl -X POST http://localhost:8081/v1/test \
  -H "Content-Type: application/json" \
  -d '{"test": "data"}'
# ✅ {"status":"accepted","request_id":"..."}
```

### Completion

```bash
# 11. Update TASK_TRACKER.yaml
# M1-T1-002:
#   status: COMPLETE
#   completed_at: 2025-11-12T13:30:00Z
#   acceptance_criteria: (all checked: true)

# 12. Commit artifacts
git add router/internal/middleware/*.go
git add router/internal/routes/*.go
git add router/pkg/observability/*.go
git add TASK_TRACKER.yaml

git commit -m "[M1-T1-002] Router implementation complete

- Implemented logging, tracing, metrics, policy version middleware
- Route matcher publishes to Pub/Sub
- OTEL initialized
- All tests passing

Files:
- router/internal/middleware/logging.go
- router/internal/middleware/tracing.go
- router/internal/middleware/metrics.go
- router/internal/middleware/policy_version.go
- router/internal/routes/matcher.go
- router/pkg/observability/otel.go

Acceptance: All 6 criteria met."

git push
```

### Daily Log

```yaml
# Edit TASK_TRACKER.yaml:
daily_logs:
  - date: "2025-11-12"
    entries:
      - timestamp: "2025-11-12T13:30:00Z"
        agent: "agent-backend-1"
        task: "M1-T1-002"
        action: "COMPLETE"
        duration_hours: 4.5
        notes: "Router implementation complete. All middleware working. Tests passing."
```

### Pick Next Task

```bash
# 13. Check dependencies for M1-T1-003
cat TASK_TRACKER.yaml | grep "M1-T1-003"
# Dependencies: M1-T1-002 (just completed!) ✅

# 14. Claim M1-T1-003
# (Repeat process)
```

---

## Frequently Asked Questions

### Q: What if I don't know Go/Python/TypeScript?

**A:** Use your code generation capabilities! Read existing code first, then generate new code matching the patterns.

### Q: What if a task is too vague?

**A:** Ask the human coordinator for clarification. Update the task definition with more details.

### Q: Can I skip steps in the task definition?

**A:** No. Follow steps exactly. If you think a step is wrong, ask first.

### Q: What if I find a bug in existing code?

**A:** Create a new task for the bug fix, or fix it inline if it blocks your current task.

### Q: How do I know what priority to work on?

**A:** P0 > P1 > P2 > P3. Within same priority, work on tasks that unblock the most other tasks.

### Q: What if two agents want the same task?

**A:** First to push the "claimed" commit wins. Second agent picks another task.

---

## Resources

### Documentation
- [AGENT_EXECUTION_PLAN.md](./AGENT_EXECUTION_PLAN.md) - Your blueprint
- [TASK_TRACKER.yaml](../TASK_TRACKER.yaml) - Current status
- [PRINCIPLES.md](./PRINCIPLES.md) - Design principles
- [GAPS_AND_REGRETS.md](./GAPS_AND_REGRETS.md) - Why we built it this way

### External
- [GCP Documentation](https://cloud.google.com/docs)
- [Terraform GCP Provider](https://registry.terraform.io/providers/hashicorp/google/latest/docs)
- [Go Documentation](https://go.dev/doc/)
- [OTEL Documentation](https://opentelemetry.io/docs/)

---

## Final Checklist Before Marking Task Complete

- [ ] All steps in task definition executed
- [ ] All acceptance criteria checked: true
- [ ] All tests passing
- [ ] Code formatted (go fmt, black, prettier)
- [ ] No linter errors
- [ ] TASK_TRACKER.yaml updated
- [ ] Artifacts committed and pushed
- [ ] Daily log entry added
- [ ] Next task dependencies checked

---

**You're ready! Start with M1-T0-001 and work your way through the plan.**

**Remember: We're building the future of API management. Take your time, do it right, and ask for help when needed.**

**Good luck, agent! 🤖**

---

**Last Updated:** 2025-11-11
**Maintained by:** Platform Architecture Team
**Questions:** Create GitHub issue or ask human coordinator
