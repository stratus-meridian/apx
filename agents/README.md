# APX Agents (Enterprise Feature)

AI agents for autonomous API management are available as part of APX Enterprise.

## Available Agents (Enterprise)

### 🤖 Orchestrator
Central coordinator for agent intents, deduplication, and sequencing.

### 🏗️ Builder Agent
Natural language → OpenAPI + policy YAML → Pull Request

**Example:**
```
"Create a payment API with rate limiting for enterprise customers"
→ Generates complete Product, Route, and PolicyBundle configs
→ Opens PR for review
```

### ⚡ Optimizer Agent
SLO/cost guardian that auto-scales, optimizes cache, and adjusts batch sizes.

**Capabilities:**
- Predictive scaling based on traffic patterns
- Cost optimization recommendations
- Auto-tuning of worker pool sizes
- Cache warming strategies

### 🔒 Security Agent
Configuration scanner, traffic analyzer, and automatic mitigation.

**Features:**
- Detects policy violations
- Identifies suspicious traffic patterns
- Suggests security improvements
- Auto-applies rate limits on attacks

### ✅ Validators
Schema validation, linting, dry-run testing, and blast-radius checks before deployment.

---

## For Open Source Users

The **agent interface definitions and SDK** will be released in Q2 2025, allowing you to:
- Build custom agents using your own AI models
- Integrate with existing automation tools
- Contribute agents to the community marketplace

**Coming soon:**
- Agent SDK (Apache 2.0)
- Example stub agents
- Agent marketplace

---

## Enterprise Features

Full AI agents with production-grade models are available through:
- **APX Cloud**: Managed platform with agents included
- **APX Enterprise License**: Self-hosted with agent runtime
- **Custom Agents**: Build proprietary agents for your use case

📧 Contact: enterprise@apx.dev

---

## Agent Architecture

Agents operate under the **"AI ≠ Root"** principle:
- Agents **propose** changes, never directly modify production
- Validators check every action (schema, lint, dry-run, blast-radius)
- Humans gate production changes via PR approval
- All agent actions are auditable and reversible

See [PRINCIPLES.md](../docs/PRINCIPLES.md) for more details.
