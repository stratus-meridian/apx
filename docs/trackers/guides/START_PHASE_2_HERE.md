# START HERE: Phase 2 Agent Onboarding

**Welcome to Phase 2 of the APX Platform implementation!**

This document is your starting point. Read it first, then follow the links to get started.

---

## 🎯 Quick Status

**Where We Are:**
- ✅ Phase 0: Foundation (COMPLETE)
- ✅ Phase 1: Backend Infrastructure (COMPLETE)
- ✅ Portal M1: Core Portal (COMPLETE)
- ✅ Portal M2: Analytics & Observability (COMPLETE)
- ⏳ **Phase 2: Policy Engine (NEXT - YOU ARE HERE!)**

**Overall Progress:** 46/100 tasks (46%) ✅

---

## 📚 Required Reading (30 minutes)

### 1. Progress Tracker (10 min) - START HERE
**File:** `/Users/agentsy/APILEE/APX_PROJECT_TRACKER.yaml`

**What it is:** The single source of truth for all tasks

**What to look for:**
- Your assigned task
- Task dependencies
- Acceptance criteria
- Current status

**Action:** Open this file and find the `backend_phase_2_policy_engine` section.

---

### 2. Calibration Summary (10 min)
**File:** `/Users/agentsy/APILEE/PHASE_2_CALIBRATION_SUMMARY.md`

**What it is:** High-level overview of Phase 2

**What to look for:**
- What's been completed
- Phase 2 task breakdown (16 tasks)
- Week-by-week plan
- Success criteria

**Action:** Read to understand the big picture.

---

### 3. Agent Instructions (10 min)
**File:** `/Users/agentsy/APILEE/.private/docs/PHASE_2_AGENT_INSTRUCTIONS.md`

**What it is:** Your step-by-step execution guide

**What to look for:**
- How to claim a task
- Code examples
- Testing requirements
- Common issues and solutions

**Action:** Bookmark this - you'll reference it constantly.

---

## 🚀 5-Minute Quick Start

If you just want to dive in:

```bash
# 1. Navigate to project
cd /Users/agentsy/APILEE

# 2. Check current status
cat APX_PROJECT_TRACKER.yaml | grep -A 5 "status: \"NOT_STARTED\""

# 3. Look at first available task
cat APX_PROJECT_TRACKER.yaml | grep -A 30 "M2-T1-001"

# 4. Read the agent instructions
cat .private/docs/PHASE_2_AGENT_INSTRUCTIONS.md | less

# 5. Claim your task (update tracker)
vim APX_PROJECT_TRACKER.yaml
# Change M2-T1-001 status to IN_PROGRESS
# Add your agent ID

# 6. Commit
git add APX_PROJECT_TRACKER.yaml
git commit -m "[M2-T1-001] Claiming task"
git push

# 7. Start coding!
```

---

## 📋 Phase 2 Overview

### Timeline: 4 Weeks (16 Tasks)

```
Week 5: Policy Compiler (4 tasks)
  ├─ M2-T1-001: OPA Integration
  ├─ M2-T1-002: Policy Compiler
  ├─ M2-T1-003: GCS Artifacts
  └─ M2-T1-004: GitOps

Week 6: Version Support (3 tasks)
  ├─ M2-T2-001: Router Version Selection
  ├─ M2-T2-002: Worker N/N-1 Support
  └─ M2-T2-003: Firestore Schema

Week 7: Canary Rollouts (3 tasks)
  ├─ M2-T3-001: Traffic Splitting
  ├─ M2-T3-002: Auto-Rollback
  └─ M2-T3-003: CLI Tools

Week 8: Testing (4 tasks)
  ├─ M2-T4-001: Canary Rollout Test
  ├─ M2-T4-002: Auto-Rollback Test
  ├─ M2-T4-003: Version Coverage Test
  └─ M2-T4-004: E2E Acceptance
```

---

## 👥 Agent Types & Assignments

### Recommended Agent Specializations

**agent-backend-1:**
- Focus: Router enhancements
- Tasks: M2-T2-001, M2-T3-001, M2-T3-003
- Skills: Go, HTTP middleware, CLI tools

**agent-backend-2:**
- Focus: Policy engine core
- Tasks: M2-T1-001, M2-T1-002, M2-T2-002, M2-T3-002
- Skills: Go, OPA, WASM, monitoring

**agent-infrastructure-1:**
- Focus: Cloud infrastructure
- Tasks: M2-T1-003, M2-T1-004, M2-T2-003
- Skills: Terraform, GCP, Firestore, Cloud Build

**agent-testing-1:**
- Focus: Integration testing
- Tasks: M2-T4-001, M2-T4-002, M2-T4-003, M2-T4-004
- Skills: Bash scripting, integration testing

---

## 🎯 Your First Task: M2-T1-001

**Task:** OPA Integration Setup
**Priority:** P0 (Critical)
**Estimated Time:** 4 hours
**Dependencies:** None
**Assigned To:** agent-backend-2 (recommended)

**What you'll do:**
1. Install OPA SDK for Go
2. Create policy evaluation engine
3. Write tests
4. Verify WASM support

**Files you'll create:**
- `control/pkg/opa/engine.go`
- `control/pkg/opa/engine_test.go`
- `control/pkg/opa/wasm_test.go`

**How to start:**
1. Read the full task definition in `APX_PROJECT_TRACKER.yaml`
2. Follow step-by-step guide in `PHASE_2_AGENT_INSTRUCTIONS.md`
3. Claim the task by updating the tracker
4. Start coding!

---

## 📊 Success Criteria

### Task-Level Success
- ✅ All acceptance criteria met
- ✅ Tests passing (>80% coverage)
- ✅ Code formatted and linted
- ✅ Documentation updated
- ✅ Tracker updated

### Phase-Level Success
- ✅ All 16 tasks complete
- ✅ GitOps pipeline working
- ✅ Canary rollout functional
- ✅ Auto-rollback working (<2 min)
- ✅ Integration tests 100% passing

---

## 🔧 Prerequisites

### Environment Setup

**Verify you have access:**
```bash
# GCP project
gcloud config get-value project
# Should be: apx-build-478003

# Deployed services
gcloud run services list --region=us-central1
# Should see: apx-edge, apx-router, apx-workers

# Firestore
gcloud firestore databases describe --database=(default)
# Should be: ACTIVE

# Pub/Sub
gcloud pubsub topics list
# Should see: apx-requests-us-dev
```

**If any checks fail:** Ask coordinator for access

---

## 📁 Key Files & Directories

### Documentation
```
/Users/agentsy/APILEE/
├── APX_PROJECT_TRACKER.yaml              ← Main tracker
├── PHASE_2_CALIBRATION_SUMMARY.md        ← Phase 2 overview
├── APX_ROADMAP_VISUAL.md                 ← Visual roadmap
├── START_PHASE_2_HERE.md                 ← This file
└── .private/docs/
    ├── PHASE_2_AGENT_INSTRUCTIONS.md     ← Your guide
    ├── AGENT_EXECUTION_PLAN.md           ← Original plan
    └── PRINCIPLES.md                     ← Design principles
```

### Code Directories
```
/Users/agentsy/APILEE/
├── control/                               ← NEW: Policy compiler
│   ├── cmd/compiler/                     ← Compiler service
│   ├── internal/compiler/                ← Compiler logic
│   └── pkg/opa/                          ← OPA engine
├── router/                                ← Enhance for versioning
│   └── internal/middleware/              ← Add canary logic
├── workers/                               ← Enhance for N/N-1
│   └── internal/policy/                  ← Policy cache
├── cli/                                   ← NEW: CLI tools
│   └── cmd/                              ← rollout, rollback, status
└── infra/terraform/                       ← Infrastructure
    ├── gcs_artifacts.tf                  ← NEW
    └── cloud_build.tf                    ← NEW
```

---

## 🆘 Getting Help

### When You're Blocked

1. **Update tracker:**
   ```yaml
   status: "BLOCKED"
   blocker_type: "TECHNICAL"
   blocker_description: "Detailed description..."
   ```

2. **Check documentation:**
   - PHASE_2_AGENT_INSTRUCTIONS.md (common issues section)
   - PRINCIPLES.md (design decisions)

3. **Escalate:**
   - Create GitHub issue with details
   - Tag coordinator
   - Include what you tried

### Common Issues

**Issue:** Can't push to Git
**Solution:** Run `git pull --rebase` first

**Issue:** GCP permission denied
**Solution:** Ask coordinator for role assignment

**Issue:** OPA compilation errors
**Solution:** Check Rego syntax with `opa check policy.rego`

---

## 🎊 Let's Get Started!

### Your Action Items:

1. ✅ Read this document (you're doing it!)
2. [ ] Read APX_PROJECT_TRACKER.yaml
3. [ ] Read PHASE_2_CALIBRATION_SUMMARY.md
4. [ ] Read PHASE_2_AGENT_INSTRUCTIONS.md
5. [ ] Verify environment access
6. [ ] Claim your first task
7. [ ] Start building!

### What Success Looks Like:

- **By end of Week 5:** Policy compiler working
- **By end of Week 6:** N/N-1 versioning working
- **By end of Week 7:** Canary rollouts working
- **By end of Week 8:** All tests passing

**Total:** Policy engine complete, APX transformed into intelligent API platform!

---

## 📝 Daily Routine

### Start of Day:
1. Pull latest: `git pull`
2. Check tracker for updates
3. Read daily logs from other agents
4. Plan your work for the day

### During Work:
1. Follow PHASE_2_AGENT_INSTRUCTIONS.md
2. Update tracker as you progress
3. Commit frequently
4. Test continuously

### End of Day:
1. Update tracker with progress
2. Add daily log entry
3. Push all work
4. Plan tomorrow

---

## 🚀 Ready to Start?

### Next Step:

```bash
cd /Users/agentsy/APILEE
cat APX_PROJECT_TRACKER.yaml | grep -A 50 "M2-T1-001"
```

### Then:

Open `.private/docs/PHASE_2_AGENT_INSTRUCTIONS.md` and follow the guide!

---

**Good luck, agent! We're counting on you to build the future of API management! 💪**

**Questions?** Update the tracker with your questions and tag the coordinator.

**Stuck?** See "Getting Help" section above.

**Celebrating?** Update daily logs with your wins! 🎉

---

*Last Updated: 2025-11-12*
*Phase: 2 (Policy Engine)*
*Status: Ready to Start*
