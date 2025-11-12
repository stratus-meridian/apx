# AI Agent Git Workflow Guide

**For AI Agents Working on APX Platform**

---

## 🎯 Overview

APX uses a **dual-repository structure**:
- **Public Repo** (`/Users/agentsy/APILEE`) → Runtime code (Apache 2.0)
- **Private Repo** (`/Users/agentsy/APILEE/.private`) → Proprietary code

**As an AI agent, you MUST commit to the correct repo based on what you're working on.**

---

## 📋 Quick Decision Tree

```
What did you modify?
│
├─ Files in edge/, router/, workers/, configs/, tests/ → PUBLIC REPO
├─ Files in docs/ (architecture, principles, ADRs) → PUBLIC REPO
├─ Files in .private/agents/ → PRIVATE REPO
├─ Files in .private/control/ → PRIVATE REPO
├─ Files in .private/portal/ → PRIVATE REPO
├─ Files in .private/docs/ → PRIVATE REPO
├─ TASK_TRACKER.yaml → PRIVATE REPO (.private/TASK_TRACKER.yaml)
└─ Not sure? → Read "Decision Matrix" below
```

---

## 📊 Decision Matrix: Which Repo?

| File/Folder | Repo | Reason |
|-------------|------|--------|
| `edge/` | **Public** | Envoy gateway - open source |
| `router/` | **Public** | Go routing service - open source |
| `workers/cpu-pool/` | **Public** | Worker implementation - open source |
| `configs/` | **Public** | CRD schemas - open source |
| `tools/cli/` | **Public** | CLI tool - open source |
| `tests/` | **Public** | All tests - open source |
| `observability/` | **Public** | OTEL configs - open source |
| `docs/PRINCIPLES.md` | **Public** | Architecture docs - open source |
| `docs/adrs/` | **Public** | Architecture decisions - open source |
| `README.md` | **Public** | Main documentation - open source |
| `Makefile` | **Public** | Build system - open source |
| `docker-compose.yml` | **Public** | Dev environment - open source |
| `.private/agents/` | **Private** | AI agents - proprietary |
| `.private/control/` | **Private** | Policy compiler - proprietary |
| `.private/portal/` | **Private** | Next.js app - proprietary |
| `.private/infra/` | **Private** | Terraform configs - sensitive |
| `.private/docs/` | **Private** | Internal docs - proprietary |
| `.private/TASK_TRACKER.yaml` | **Private** | Development tracker - internal |

---

## 🔧 Git Workflow for AI Agents

### Step 1: Check What You Modified

Before committing, always check what you changed:

```bash
# Check main repo
cd /Users/agentsy/APILEE
git status

# Check private repo
cd /Users/agentsy/APILEE/.private
git status
```

### Step 2: Determine Which Repo

**If you see changes in main repo (`/Users/agentsy/APILEE`):**
→ These are PUBLIC changes

**If you see changes in `.private/`:**
→ These are PRIVATE changes

**If you see changes in BOTH:**
→ You'll need TWO separate commits (one per repo)

---

## 📝 Commit & Push: Public Repo

### When: You modified edge/, router/, workers/, configs/, tests/, docs/

```bash
# Navigate to main repo
cd /Users/agentsy/APILEE

# Check what changed
git status
git diff

# Stage your changes
git add <files-you-modified>

# Example: If you modified router middleware
git add router/internal/middleware/new_feature.go

# Or add multiple files
git add router/internal/middleware/ router/pkg/

# Commit with clear message
git commit -m "[Component] Brief description

Detailed explanation:
- What changed
- Why it changed
- Impact

Fixes: #123 (if applicable)
"

# Push to GitHub
git push origin master
```

### Commit Message Format (Public Repo)

```
[Component] One-line summary (50 chars max)

Longer description explaining:
- What: What was changed
- Why: Why the change was necessary
- How: How the change works (if complex)

Examples:
- Added new rate limiting algorithm
- Improved p99 latency by 15ms
- Fixed memory leak in worker pool

Related: #issue-number (if applicable)
```

**Examples:**

```bash
# Example 1: Bug fix
git commit -m "[Router] Fix memory leak in request ID middleware

The middleware was not releasing memory after request completion,
causing gradual memory growth over time.

Changes:
- Added defer cleanup in middleware chain
- Implemented request context cancellation
- Added memory profiling test

Fixes: #42
"

# Example 2: New feature
git commit -m "[Edge] Add JWT validation caching

Implement LRU cache for JWT validation to reduce latency.

Changes:
- Added cache layer with 5-minute TTL
- Reduced JWT validation latency from 15ms to 2ms
- Added cache hit/miss metrics

Impact: p99 latency reduced by 13ms
"

# Example 3: Tests
git commit -m "[Tests] Add integration tests for async pattern

Comprehensive tests for 202 → poll → 200 flow.

Coverage:
- Happy path: request → queue → worker → response
- Timeout handling
- Error scenarios
- Load testing (1k rps)
"

# Example 4: Documentation
git commit -m "[Docs] Update architecture principles

Clarified multi-tenancy isolation model based on team feedback.

Changes:
- Added isolation level examples
- Clarified tenant context propagation
- Updated diagrams
"
```

---

## 🔒 Commit & Push: Private Repo

### When: You modified .private/agents/, .private/control/, .private/portal/, .private/docs/

```bash
# Navigate to private repo
cd /Users/agentsy/APILEE/.private

# Check what changed
git status
git diff

# Stage your changes
git add <files-you-modified>

# Example: If you modified builder agent
git add agents/builder/

# Commit with clear message
git commit -m "[Agents] Brief description

Detailed explanation of changes.
"

# Push to GitHub
git push origin main
```

### Commit Message Format (Private Repo)

```
[Component] One-line summary

Detailed description:
- Implementation details
- Agent capabilities added
- Internal dependencies

Task: Reference to TASK_TRACKER.yaml if applicable
```

**Examples:**

```bash
# Example 1: Agent implementation
git commit -m "[Agents] Implement builder agent NL→config generation

First working version of builder agent that converts natural
language to OpenAPI + policy YAML.

Implementation:
- NL parsing with GPT-4
- OpenAPI schema generation
- PolicyBundle creation
- PR generation

Capabilities:
- Handles payment APIs, authentication APIs
- Generates complete Product + Route + PolicyBundle
- Validates schemas before PR

Next: Add validation agent integration

Task: M4-T1-002 from TASK_TRACKER.yaml
"

# Example 2: Control plane
git commit -m "[Control] Add OPA→WASM compiler

Implement policy compiler that converts Rego to WASM artifacts.

Features:
- OPA compilation pipeline
- Artifact signing with cosign
- Version tagging (semver)
- SHA256 hash generation

Output: Immutable artifacts stored in GCS

Task: M2-T1-001
"

# Example 3: Portal
git commit -m "[Portal] Add API key management UI

Implement key generation, rotation, and usage tracking.

Components:
- Key generation modal
- Key list with search/filter
- Usage charts (Recharts)
- Rotation workflow

Integration:
- Connects to /api/keys endpoint
- Real-time usage via WebSocket

Task: M4-T3-002
"

# Example 4: Documentation updates
git commit -m "[Docs] Update agent execution plan

Updated task dependencies for M4 based on M3 completion.

Changes:
- Added new tasks for agent orchestration
- Updated acceptance criteria
- Refined time estimates
- Added integration test tasks

File: AGENT_EXECUTION_PLAN.md
"
```

---

## 🔄 Handling Changes in BOTH Repos

If you modified files in BOTH public and private repos, commit to each separately:

### Workflow:

```bash
# 1. Commit to PUBLIC repo first
cd /Users/agentsy/APILEE
git status
git add <public-files>
git commit -m "[Public changes message]"
git push origin master

# 2. Then commit to PRIVATE repo
cd .private
git status
git add <private-files>
git commit -m "[Private changes message]"
git push origin main

# 3. Verify both
cd ..
git log --oneline -1
cd .private
git log --oneline -1
cd ..
```

### Example: You added a feature that touches both

**Scenario:** Added agent interface (public) + agent implementation (private)

```bash
# Public repo: Interface definition
cd /Users/agentsy/APILEE
git add router/pkg/agents/interface.go
git commit -m "[Router] Add agent plugin interface

Define interface for external agent plugins to integrate with router.

Interface:
- ProcessRequest(ctx, req) (resp, error)
- ValidateConfig(config) error
- Name() string

This allows community to build custom agents that integrate
with the router's request processing pipeline.

See: docs/AGENT_SDK.md for usage
"
git push origin master

# Private repo: Implementation
cd .private
git add agents/builder/plugin.go
git commit -m "[Agents] Implement builder agent router plugin

Builder agent now integrates with router via plugin interface.

Implementation:
- Implements router.AgentPlugin interface
- Processes NL requests via /v1/agent/build
- Validates OpenAPI schemas
- Returns generated configs

Requires: router >= v0.3.0 (interface added)
"
git push origin main
```

---

## ⚠️ Safety Checks Before Committing

### Pre-Commit Checklist

**ALWAYS run these before committing:**

```bash
# 1. Check for secrets
git diff | grep -iE "(password|secret|api_key|token|private_key)"
# ❌ If found: Remove secrets, use environment variables

# 2. Verify you're in the right repo
pwd
# Should be /Users/agentsy/APILEE (public)
# or /Users/agentsy/APILEE/.private (private)

# 3. Check what you're committing
git diff --staged

# 4. Verify no private code in public repo
cd /Users/agentsy/APILEE
git status | grep -E "(agents/.*/.*.go|control/|portal/|.private/)"
# ❌ If found: You're about to commit private code to public repo!

# 5. Check .gitignore is working
git status | grep ".private/"
# Should show NOTHING (means .private/ is properly ignored)
```

### Common Mistakes to Avoid

❌ **DON'T:**
```bash
# Don't add everything blindly
git add .
git commit -m "updates"
git push

# Don't commit to wrong repo
cd /Users/agentsy/APILEE
git add .private/agents/  # ❌ This won't work but don't try!

# Don't commit secrets
git add .env  # ❌ Has secrets
git commit -m "Add config"

# Don't use vague messages
git commit -m "fix"
git commit -m "WIP"
git commit -m "updates"
```

✅ **DO:**
```bash
# Be specific about what you're committing
git add router/internal/middleware/logging.go
git commit -m "[Router] Improve logging format"
git push origin master

# Check before committing
git status
git diff --staged

# Use descriptive messages
git commit -m "[Edge] Fix JWT validation memory leak

The JWT cache was not evicting expired entries..."
```

---

## 🎯 Task-Based Commit Workflow

When you complete a task from TASK_TRACKER.yaml:

### Step 1: Complete the task

```bash
# Do your work...
# Write code, tests, docs
```

### Step 2: Update TASK_TRACKER.yaml

```bash
cd /Users/agentsy/APILEE/.private

# Edit TASK_TRACKER.yaml
# - Set task status: COMPLETE
# - Set completed_at timestamp
# - Check all acceptance_criteria: true
# - Add artifacts
# - Add notes
```

### Step 3: Commit to appropriate repo(s)

**If task was public code (e.g., M1-T1-001 Edge Gateway):**

```bash
# Commit public changes
cd /Users/agentsy/APILEE
git add edge/
git commit -m "[M1-T1-001] Build and Deploy Edge Gateway

Implemented complete edge gateway with Envoy + WASM filters.

Components:
- Envoy config with JWT fast-path
- Request ID generation
- Rate limiting (coarse)
- Health check endpoint
- Dockerfile for Cloud Run

Acceptance:
- Docker image builds successfully
- Health check returns 200 OK
- Request IDs generated on all requests
- p99 latency <20ms

Task: M1-T1-001 from TASK_TRACKER.yaml
Status: COMPLETE
"
git push origin master

# Commit TASK_TRACKER.yaml update
cd .private
git add TASK_TRACKER.yaml
git commit -m "[Task] Complete M1-T1-001 - Edge Gateway

Updated task tracker:
- Status: COMPLETE
- All acceptance criteria: ✓
- Completed at: 2025-11-12T10:30:00Z
- Artifacts: edge/Dockerfile, edge/envoy/envoy.yaml
"
git push origin main
```

**If task was private code (e.g., M4-T1-002 Builder Agent):**

```bash
# Commit private changes AND tracker
cd /Users/agentsy/APILEE/.private
git add agents/builder/ TASK_TRACKER.yaml
git commit -m "[M4-T1-002] Implement Builder Agent

Builder agent now converts NL to OpenAPI + policy configs.

Implementation:
- NL parsing with LLM
- Schema generation
- Policy creation
- PR generation

Acceptance:
- Handles 5 API types (payments, auth, webhooks, data, ML)
- Generates valid OpenAPI 3.0
- Creates PolicyBundle YAML
- Opens PR with generated files
- All unit tests passing

Task: M4-T1-002 from TASK_TRACKER.yaml
Status: COMPLETE
"
git push origin main
```

---

## 📋 Real-World Scenarios

### Scenario 1: You Fixed a Bug in Router

```bash
cd /Users/agentsy/APILEE

# Check what you changed
git status
# Shows: modified: router/internal/middleware/tenant.go

git diff router/internal/middleware/tenant.go
# Review changes

# Add and commit
git add router/internal/middleware/tenant.go
git commit -m "[Router] Fix tenant context not propagating to workers

The tenant middleware was not adding tenant_id to OTEL baggage,
causing workers to receive requests without tenant context.

Fix:
- Added baggage.Set(ctx, \"tenant_id\", tenant.ID)
- Added test to verify baggage propagation

Impact: Fixes tenant isolation issue reported in #67
"

# Push
git push origin master
```

### Scenario 2: You Implemented Agent Orchestrator

```bash
cd /Users/agentsy/APILEE/.private

# Check what you changed
git status
# Shows: 
#   new file: agents/orchestrator/main.go
#   new file: agents/orchestrator/intent.go
#   new file: agents/orchestrator/dedup.go
#   modified: TASK_TRACKER.yaml

# Add all agent files
git add agents/orchestrator/

# Add tracker separately (good practice)
git add TASK_TRACKER.yaml

# Commit
git commit -m "[M4-T1-001] Implement Agent Orchestrator

Central coordinator for all agent intents.

Features:
- Intent inbox with deduplication
- Rate limiting per agent type
- Sequencing and priority
- Conflict detection

Architecture:
- Subscribes to apx-agent-intents topic
- Uses Redis for deduplication (24h window)
- Publishes to agent-specific topics
- Tracks intent status in Firestore

Integration:
- Builder agent receives intents via orchestrator
- Optimizer agent subscribes to optimizer-intents
- Security agent on security-intents

Task: M4-T1-001 from TASK_TRACKER.yaml
Status: COMPLETE

Files:
- agents/orchestrator/main.go (entry point)
- agents/orchestrator/intent.go (intent types)
- agents/orchestrator/dedup.go (deduplication logic)
"

# Push
git push origin main
```

### Scenario 3: You Added Tests (Public Repo)

```bash
cd /Users/agentsy/APILEE

# Check changes
git status
# Shows:
#   new file: tests/integration/policy_versioning_test.sh
#   modified: tests/integration/README.md

# Add files
git add tests/integration/

# Commit
git commit -m "[Tests] Add integration tests for policy versioning

Comprehensive tests for N/N-1 policy support and canary rollouts.

Test coverage:
- Policy version tagging
- N/N-1 simultaneous support
- Canary traffic splitting (1%, 5%, 25%, 50%, 100%)
- Auto-rollback on error rate >5%
- In-flight request handling

Results:
- All 15 test scenarios passing
- Tested at 1k rps sustained load
- Verified rollback completes <2 minutes

Related: M2 acceptance criteria
"

# Push
git push origin master
```

### Scenario 4: You Updated Documentation (Both Repos)

**Public docs (architecture):**
```bash
cd /Users/agentsy/APILEE
git add docs/PRINCIPLES.md
git commit -m "[Docs] Clarify async-by-default principle

Added examples and failure modes to make principle more concrete.

Changes:
- Added code examples for 202 Accepted pattern
- Documented WebSocket gateway for >5min sessions
- Explained why synchronous breaks with Cloud Run
- Added diagram for async flow

Requested by: Community issue #89
"
git push origin master
```

**Private docs (agent implementation):**
```bash
cd /Users/agentsy/APILEE/.private
git add docs/AGENT_EXECUTION_PLAN.md
git commit -m "[Docs] Update agent execution plan for M5

Revised M5 task breakdown based on M4 learnings.

Changes:
- Split optimizer agent into 2 phases
- Added multi-region considerations
- Updated time estimates (more realistic)
- Added integration testing tasks

Total M5 tasks: 25 → 32 (added 7)
Estimated completion: +2 weeks
"
git push origin main
```

---

## 🚨 Emergency: Committed to Wrong Repo

### If you committed private code to public repo:

```bash
cd /Users/agentsy/APILEE

# DON'T PUSH YET!

# Check what's in the commit
git log -1 --stat

# If it contains private code, undo the commit
git reset --soft HEAD~1
# This undoes commit but keeps your changes

# Move private files to .private/
mv agents/builder/* .private/agents/builder/

# Re-commit public files only
git add <only-public-files>
git commit -m "[Corrected commit message]"

# Commit private files to private repo
cd .private
git add agents/builder/
git commit -m "[Private changes message]"
git push origin main
```

### If you already pushed to GitHub:

```bash
# ⚠️ CRITICAL: Act fast!

cd /Users/agentsy/APILEE

# Remove sensitive commit from history
git reset --hard HEAD~1

# Force push (dangerous but necessary)
git push --force origin master

# Verify sensitive data is gone on GitHub
# Visit: https://github.com/stratus-meridian/apx

# Re-commit correctly
# ... follow normal workflow
```

**Then immediately notify the human owner!**

---

## 📊 Daily Summary for Human

At end of your session, provide a summary:

```markdown
## Today's Git Activity

### Public Repo (apx)
**Commits:** 3
1. [Router] Fix tenant context propagation (abc123)
2. [Tests] Add integration tests for async pattern (def456)
3. [Docs] Update architecture principles (ghi789)

**Files Changed:** 8 files
- router/internal/middleware/tenant.go
- tests/integration/async_pattern_test.sh
- docs/PRINCIPLES.md

**Pushed:** ✅ All commits pushed to master

### Private Repo (apx-private)
**Commits:** 2
1. [Agents] Implement orchestrator deduplication (jkl012)
2. [Task] Complete M4-T1-001 - Orchestrator (mno345)

**Files Changed:** 4 files
- agents/orchestrator/dedup.go
- agents/orchestrator/tests/
- TASK_TRACKER.yaml

**Pushed:** ✅ All commits pushed to main

### Status
- ✅ All changes committed
- ✅ All changes pushed
- ✅ No uncommitted files
- ✅ No secrets committed
```

---

## 🔍 Quick Reference Commands

```bash
# Check status of both repos
cd /Users/agentsy/APILEE && git status
cd /Users/agentsy/APILEE/.private && git status

# See what changed (before committing)
git diff

# See what will be committed
git diff --staged

# View recent commits
git log --oneline -10

# Check remote URLs
git remote -v

# Pull latest changes
cd /Users/agentsy/APILEE && git pull
cd /Users/agentsy/APILEE/.private && git pull

# Check for secrets before committing
git diff | grep -iE "(password|secret|api_key|token)"

# Verify .private/ is ignored
cd /Users/agentsy/APILEE && git status | grep ".private/"
# Should show nothing

# See all tracked files
git ls-files

# Undo last commit (keep changes)
git reset --soft HEAD~1

# Undo last commit (discard changes)
git reset --hard HEAD~1

# Amend last commit message
git commit --amend -m "New message"
```

---

## ✅ Commit Checklist for AI Agents

Before every commit:

- [ ] `git status` - Verified what I'm committing
- [ ] `git diff` - Reviewed all changes
- [ ] `pwd` - Confirmed I'm in correct repo
- [ ] Checked for secrets in diff
- [ ] Commit message follows format
- [ ] Commit message references task (if applicable)
- [ ] .private/ is not in public repo git status
- [ ] If TASK_TRACKER.yaml updated, committed to private repo
- [ ] Ran tests (if code changes)
- [ ] Updated documentation (if needed)
- [ ] Ready to push

---

## 📞 Questions?

**Not sure which repo?**
- If in `docs/` and it's architecture/principles → Public
- If in `docs/` and it's agent implementation → Private
- When in doubt, ask: "Is this something the community should see?"

**Commit message unclear?**
- Start with component: [Edge], [Router], [Tests], [Docs], [Agents]
- Explain what and why
- Reference issue/task if applicable

**Made a mistake?**
- Don't panic
- Use `git reset` to undo (before push)
- If pushed, use `git revert` or force push (if needed)
- Notify human if sensitive data committed

---

**Last Updated:** 2025-11-12
**For:** AI Agents working on APX Platform
**Repos:** 
- Public: https://github.com/stratus-meridian/apx
- Private: https://github.com/stratus-meridian/apx-private
