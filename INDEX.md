# APX Platform - Master Documentation Index

**Project:** APX (API Proxy & Execution Platform)  
**Last Updated:** 2025-11-11

---

## 🎯 Quick Navigation

### For Backend (APX Router/Edge/Workers):
- **Task Tracker:** [BACKEND_TASK_TRACKER.yaml](.private/BACKEND_TASK_TRACKER.yaml) ← Backend implementation tasks
- **Validation:** [VALIDATION_TRACKER.yaml](VALIDATION_TRACKER.yaml) ← V-001 to V-007 validation tasks
- **Implementation:** [AGENT_IMPLEMENTATION_SUMMARY.md](AGENT_IMPLEMENTATION_SUMMARY.md)

### For Frontend (Developer Portal):
- **📍 START HERE:** [PORTAL_INDEX.md](.private/PORTAL_INDEX.md) ← Complete portal documentation map
- **Task Tracker:** [PORTAL_TASK_TRACKER.yaml](.private/PORTAL_TASK_TRACKER.yaml) ← Portal implementation tasks
- **Quick Start:** [PORTAL_README.md](.private/PORTAL_README.md)

---

## 📂 Project Structure

```
/Users/agentsy/APILEE/
│
├── 🏗️ BACKEND APX PLATFORM
│   ├── .private/BACKEND_TASK_TRACKER.yaml       ← Backend tasks
│   ├── VALIDATION_TRACKER.yaml                  ← Validation sprint
│   ├── AGENT_IMPLEMENTATION_SUMMARY.md          ← Backend strategy
│   ├── router/                                  ← Go router service
│   ├── edge/                                    ← Edge proxy
│   ├── workers/                                 ← Worker execution pools
│   └── tests/                                   ← Backend tests
│
├── 🎨 FRONTEND DEVELOPER PORTAL
│   ├── .private/PORTAL_INDEX.md                 ← ⭐ Portal docs index
│   ├── .private/PORTAL_TASK_TRACKER.yaml        ← Portal tasks (ACTIVE)
│   ├── .private/PORTAL_README.md                ← Portal quick start
│   ├── .private/PORTAL_AGENT_IMPLEMENTATION_SUMMARY.md ← Portal strategy
│   ├── docs/portal/                             ← Detailed portal docs
│   │   ├── PORTAL_AI_AGENT_INSTRUCTIONS.md     
│   │   ├── PORTAL_AGENT_EXECUTION_PLAN.md      
│   │   └── PORTAL_INTEGRATION_ARCHITECTURE.md  
│   └── .private/portal/                         ← Next.js codebase
│       ├── app/                                 (Pages, API routes)
│       ├── components/                          (React components)
│       └── tests/                               (E2E, a11y tests)
│
├── 🔒 PRIVATE CODE (Proprietary)
│   ├── .private/agents/                         ← AI agent implementations
│   ├── .private/control/                        ← Policy compiler
│   └── .private/infra/                          ← Infrastructure (Terraform)
│
└── 📚 SHARED DOCUMENTATION
    ├── INDEX.md                                 ← This file
    ├── GETTING_STARTED.md                       ← Platform overview
    ├── docs/                                    ← Architecture docs
    │   ├── IMPLEMENTATION_PLAN.md              
    │   ├── PRINCIPLES.md                       
    │   └── adrs/                               (Architecture decisions)
    └── README.md                                ← Platform README
```

---

## 🤖 For AI Agents

### Working on Backend Tasks?
1. Check: [BACKEND_TASK_TRACKER.yaml](.private/BACKEND_TASK_TRACKER.yaml)
2. Read: [AGENT_IMPLEMENTATION_SUMMARY.md](AGENT_IMPLEMENTATION_SUMMARY.md)
3. Update: BACKEND_TASK_TRACKER.yaml after completion

### Working on Portal Tasks?
1. **Start here:** [PORTAL_INDEX.md](.private/PORTAL_INDEX.md)
2. Check: [PORTAL_TASK_TRACKER.yaml](.private/PORTAL_TASK_TRACKER.yaml)
3. Read: [docs/portal/PORTAL_AI_AGENT_INSTRUCTIONS.md](docs/portal/PORTAL_AI_AGENT_INSTRUCTIONS.md)
4. Execute: Follow [docs/portal/PORTAL_AGENT_EXECUTION_PLAN.md](docs/portal/PORTAL_AGENT_EXECUTION_PLAN.md)
5. Update: PORTAL_TASK_TRACKER.yaml after completion

### Working on Validation Tasks?
1. Check: [VALIDATION_TRACKER.yaml](VALIDATION_TRACKER.yaml)
2. Update: VALIDATION_TRACKER.yaml after completion

---

## 📊 Current Status (2025-11-11)

### Backend Platform
- **Status:** Core implementation complete, validation sprint in progress
- **Completed:** V-005 (Cost Controls), V-007 (Artifact Signing)
- **Blocked:** V-001 (requires deployment), V-002 (depends on V-001)

### Developer Portal
- **Status:** ✅ Milestone 0 (Foundation) COMPLETE
- **Completed:** 9/9 foundation tasks
- **Quality:** 292 tests passing, 100% accessibility, 96% performance
- **Next:** Milestone 1 (Dashboard, API Keys, Usage Data)

---

## 🔗 Quick Links

- **Backend Code:** `router/`, `edge/`, `workers/`
- **Portal Code:** `.private/portal/`
- **Private Code:** `.private/agents/`, `.private/control/`, `.private/infra/`
- **Tests:** `tests/` (backend), `.private/portal/tests/` (portal)
- **Docs:** `docs/` (architecture), `docs/portal/` (portal-specific)

---

**Maintained by:** APX Development Team  
**Questions:** Create GitHub issue or Slack #apx
