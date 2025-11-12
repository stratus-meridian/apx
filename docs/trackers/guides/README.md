# Agent Guides & Instructions

**How to work with AI agents, git, sessions, and handoffs**

---

## 📖 **Key Guides**

### **🤖 Agent Instructions**
- **[AI_AGENT_INSTRUCTIONS.md](./AI_AGENT_INSTRUCTIONS.md)** (19K)
  - How AI agents work
  - Task execution patterns
  - Code quality standards
  - Communication protocols
  - Phase 0/1 specific instructions

### **🔄 Git Workflow**
- **[AGENT_GIT_GUIDE.md](./AGENT_GIT_GUIDE.md)** (20K)
  - Git workflow for agents
  - Commit message format
  - Branch strategies
  - Pull request process
  - Conflict resolution

### **📋 Session Management**
- **[SESSION_HANDOFF_CONTEXT.md](./SESSION_HANDOFF_CONTEXT.md)** (19K)
  - Complete session handoff context
  - What's been accomplished
  - Current state
  - Next steps
  - Key file locations

- **[QUICK_START_NEW_SESSION.md](./QUICK_START_NEW_SESSION.md)** (2.1K)
  - Quick start for new sessions
  - Where we are
  - What to do next
  - Commands reference

---

## 🎯 **Quick Reference**

### **Starting a New Session?**

**Read these in order:**
1. [QUICK_START_NEW_SESSION.md](./QUICK_START_NEW_SESSION.md) - 2-minute overview
2. [SESSION_HANDOFF_CONTEXT.md](./SESSION_HANDOFF_CONTEXT.md) - Full context
3. [AI_AGENT_INSTRUCTIONS.md](./AI_AGENT_INSTRUCTIONS.md) - How to work

### **Working on a Task?**

**Follow:**
1. [AI_AGENT_INSTRUCTIONS.md](./AI_AGENT_INSTRUCTIONS.md) - Execution patterns
2. [AGENT_GIT_GUIDE.md](./AGENT_GIT_GUIDE.md) - Git workflow
3. Phase-specific instructions:
   - Phase 2: [`../phase2/PHASE_2_AGENT_INSTRUCTIONS.md`](../phase2/PHASE_2_AGENT_INSTRUCTIONS.md)

---

## 📊 **Agent Workflow**

### **Daily Routine**

**Start of Day:**
1. Pull latest: `git pull`
2. Check tracker for updates
3. Read daily logs from other agents
4. Plan your work

**During Work:**
1. Follow agent instructions
2. Update tracker as you progress
3. Commit frequently
4. Test continuously

**End of Day:**
1. Update tracker with progress
2. Add daily log entry
3. Push all work
4. Plan tomorrow

---

## 🔧 **Common Tasks**

### **Claim a Task**
```bash
vim ../backend/APX_PROJECT_TRACKER.yaml
# Change status to IN_PROGRESS
# Add your agent ID
git commit -m "[M2-T1-001] Claiming task"
git push
```

### **Update Progress**
```bash
vim ../backend/APX_PROJECT_TRACKER.yaml
# Update acceptance_criteria: checked: true
# Add notes
git commit -m "[M2-T1-001] Progress update"
git push
```

### **Complete a Task**
```bash
vim ../backend/APX_PROJECT_TRACKER.yaml
# Change status to COMPLETE
# Add completion notes
git commit -m "[M2-T1-001] Task complete"
git push
```

---

## 🆘 **Getting Help**

### **When Blocked:**
1. Update tracker: `status: "BLOCKED"`
2. Add blocker description
3. Try to unblock yourself
4. If blocked >2 hours, escalate

### **Common Issues:**
- Check [AI_AGENT_INSTRUCTIONS.md](./AI_AGENT_INSTRUCTIONS.md) → Common Issues
- Check phase-specific instructions
- Search error messages in docs
- Ask coordinator if still blocked

---

## 📝 **Documentation Standards**

### **Code Quality:**
- Follow Go style guide
- 80%+ test coverage
- No linter errors
- Comments on complex logic

### **Commit Messages:**
```
[M2-T1-001] Brief description

Detailed explanation of what changed and why.

Acceptance criteria met:
- ✅ Criterion 1
- ✅ Criterion 2
```

### **Daily Logs:**
```yaml
- date: "2025-11-13"
  entries:
    - timestamp: "2025-11-13T10:00:00Z"
      agent: "agent-backend-1"
      summary: "Completed OPA integration"
      tasks_completed: ["M2-T1-001"]
      notes: "Tests passing, ready for review"
```

---

**Happy coding! 🚀**
