# APX Policy Management Guide

**Version:** 1.0
**Date:** 2025-11-12
**Status:** Production Ready

---

## Overview

This guide covers the complete policy management workflow in the APX Portal, integrated with the Phase 2 Policy Engine Backend.

## Table of Contents

1. [Policy Editor](#policy-editor)
2. [Policy Validation](#policy-validation)
3. [Policy Compilation](#policy-compilation)
4. [Version Management](#version-management)
5. [Deployment Workflows](#deployment-workflows)
6. [Canary Deployments](#canary-deployments)
7. [Monitoring & Rollback](#monitoring--rollback)
8. [GitOps Integration](#gitops-integration)
9. [Testing Policies](#testing-policies)
10. [Troubleshooting](#troubleshooting)

---

## Policy Editor

### Creating a New Policy

1. Navigate to **Policies** → **Create New Policy**
2. Enter policy details:
   - **Name**: Unique identifier (e.g., `rate-limiting`)
   - **Version**: Semantic version (e.g., `1.0.0`)
   - **Description**: Human-readable description
3. Write your Rego policy in the editor
4. The editor provides:
   - Syntax highlighting
   - Auto-completion
   - Real-time validation
   - Error highlighting

### Policy Structure

All APX policies should follow this structure:

```rego
package apx.{policy-name}

import future.keywords.if
import future.keywords.in

# Default deny
default allow := false

# Your policy rules
allow if {
  input.request.method == "GET"
  input.request.path == "/api/v1/users"
}
```

### Best Practices

- Always use the `apx.*` namespace
- Import future keywords for modern syntax
- Define a default `allow` or `deny` rule
- Use descriptive rule names
- Add comments to explain complex logic

---

## Policy Validation

### Real-Time Validation

The policy editor provides real-time validation using the OPA Runtime API:

```typescript
import { validateRegoPolicy } from '@/lib/policies/rego-validator'

const result = await validateRegoPolicy(policyCode)

if (result.valid) {
  console.log('Policy is valid!')
} else {
  console.error('Validation errors:', result.errors)
}
```

### Validation Features

- **Syntax checking**: Catches Rego syntax errors
- **Package validation**: Ensures proper namespace usage
- **Import validation**: Checks for missing or incorrect imports
- **Rule validation**: Validates rule structure and logic
- **APX conventions**: Enforces APX-specific requirements

### Common Validation Errors

| Error | Cause | Solution |
|-------|-------|----------|
| Missing package declaration | No `package` statement | Add `package apx.{name}` at top |
| Unmatched braces | `{` without `}` | Check brace pairing |
| Missing future.keywords | Using `if` without import | Add `import future.keywords.if` |
| Invalid rule syntax | Malformed rule | Check Rego syntax guide |

---

## Policy Compilation

### Compiling to WASM

Policies are compiled to WebAssembly for efficient execution:

```typescript
import { compilePolicy } from '@/lib/policies/policy-compiler'

const result = await compilePolicy(policyCode, {
  name: 'rate-limiting',
  version: '1.0.0',
  optimize: true
})

if (result.success) {
  console.log('WASM bundle:', result.bundle)
  console.log('Artifact URL:', result.metadata?.artifact_url)
}
```

### Compilation Process

1. **Validate**: Policy is validated before compilation
2. **Compile**: OPA compiles Rego to WASM bytecode
3. **Optimize**: Optimization passes reduce bundle size
4. **Upload**: Bundle uploaded to GCS
5. **Metadata**: Bundle hash and metadata stored

### Bundle Metadata

Compiled bundles include:

- **Hash**: SHA-256 hash of WASM bundle
- **Size**: Bundle size in bytes
- **Entrypoints**: Available policy entrypoints
- **Dependencies**: Imported modules
- **Compiler version**: OPA compiler version used

---

## Version Management

### N/N-1 Version Pattern

APX supports running two versions simultaneously:

- **N (Current)**: Latest active version
- **N-1 (Previous)**: Previous active version for rollback

### Creating a Version

```typescript
import { createPolicyVersion } from '@/lib/policies/version-manager'

const versionId = await createPolicyVersion({
  id: 'rate-limiting@1.0.0',
  name: 'rate-limiting',
  version: '1.0.0',
  content: policyCode,
  status: 'draft',
  createdAt: new Date(),
  createdBy: 'user@example.com',
  description: 'Rate limiting for API endpoints'
})
```

### Version Lifecycle

```
draft → active → deprecated → rolled_back
```

- **draft**: Initial state, not deployed
- **active**: Currently deployed and serving traffic
- **deprecated**: Older version, kept for rollback
- **rolled_back**: Version that was rolled back due to issues

### Version Comparison

Compare two versions to see changes:

```typescript
import { compareVersionContent } from '@/lib/policies/version-manager'

const comparison = compareVersionContent(currentVersion, previousVersion)

console.log('Lines added:', comparison.summary.linesAdded)
console.log('Lines removed:', comparison.summary.linesRemoved)
console.log('Lines modified:', comparison.summary.linesModified)
```

---

## Deployment Workflows

### Quick Deployment

Deploy a policy directly to production:

```typescript
import { quickDeployPolicy } from '@/lib/policies/deployment-workflow'

const result = await quickDeployPolicy(
  'rate-limiting',
  '1.0.0',
  policyCode,
  'user@example.com'
)

if (result.success) {
  console.log('Deployed version:', result.versionId)
  console.log('Steps completed:', result.steps.length)
}
```

### Deployment Steps

1. **Validate Policy**: OPA validation
2. **Compile to WASM**: Generate WASM bundle
3. **Create Version**: Store in Firestore
4. **Sync to Router**: Deploy to router instances
5. **Mark Active**: Set version status to active

### Canary Deployment

Deploy with gradual traffic rollout:

```typescript
import { canaryDeployPolicy } from '@/lib/policies/deployment-workflow'

const result = await canaryDeployPolicy(
  'rate-limiting',
  '1.1.0',
  policyCode,
  10, // 10% canary traffic
  'user@example.com',
  ['us-east1', 'europe-west1'] // Optional: specific regions
)
```

---

## Canary Deployments

### Creating a Canary

Use the Canary Deployment Dialog:

1. Click **Deploy Canary** on a policy version
2. Set traffic percentage (0-100%)
3. Select target regions (optional)
4. Click **Deploy Canary**

### Traffic Split Configuration

- **0-10%**: Safe for initial testing
- **10-25%**: Early validation
- **25-50%**: Wider rollout
- **50-75%**: Majority rollout
- **75-100%**: Near-complete rollout

### Monitoring Canary Health

The Canary Monitoring Dashboard shows:

- **Real-time metrics**: Error rates, latency, request volume
- **Comparison charts**: Stable vs Canary performance
- **Traffic distribution**: Current traffic split
- **Health status**: Healthy, Warning, Critical
- **Auto-refresh**: Updates every 10 seconds

### Promoting a Canary

Once validated, promote to 100%:

1. Monitor canary metrics for at least 30 minutes
2. Verify error rate is below threshold
3. Click **Promote to 100%**
4. Traffic gradually shifts to canary version

---

## Monitoring & Rollback

### Auto-Rollback Configuration

Configure automatic rollback thresholds:

```typescript
{
  "policy_name": "rate-limiting",
  "error_rate_threshold": 5.0,      // 5% error rate
  "check_interval_seconds": 300,    // Check every 5 minutes
  "min_requests": 100,              // Min 100 requests before evaluating
  "rollback_action": "disable"      // Set canary to 0%
}
```

### Rollback Actions

- **disable**: Set canary traffic to 0%
- **rollback**: Full rollback to stable version

### Manual Rollback

Trigger manual rollback:

1. Navigate to Canary Monitoring
2. Click **Rollback** button
3. Provide rollback reason
4. Confirm rollback

### Rollback Process

1. Traffic immediately redirected to stable version
2. Canary version marked as `rolled_back`
3. Rollback event logged
4. Notification sent to operators

---

## GitOps Integration

### Connecting a Repository

1. Navigate to **Policies** → **GitOps Settings**
2. Click **Connect Repository**
3. Select provider (GitHub or GitLab)
4. Authorize the application
5. Select repository and branch
6. Configure policies path (default: `policies/`)

### GitOps Workflow

```
1. Create PR with policy changes
   ↓
2. Automated validation runs
   ↓
3. Review and approval
   ↓
4. Merge to main branch
   ↓
5. Webhook triggers deployment
   ↓
6. Policy compiled and deployed
```

### Webhook Configuration

Webhooks are automatically configured for:

- **push**: Deployment on merge to main
- **pull_request**: Validation on PR creation
- **pull_request.synchronize**: Re-validation on PR updates

### Policy Files

Store policies in your repository:

```
policies/
  ├── rate-limiting.rego
  ├── quota-enforcement.rego
  ├── security-policies.rego
  └── tests/
      ├── rate-limiting.test.yaml
      └── quota-enforcement.test.yaml
```

---

## Testing Policies

### Creating Test Scenarios

```typescript
import { runTestScenario } from '@/lib/policies/test-runner'

const scenario = {
  id: 'test-1',
  name: 'Allow GET requests',
  input: {
    request: {
      method: 'GET',
      path: '/api/v1/users'
    }
  },
  expectedResult: {
    allow: true
  }
}

const result = await runTestScenario(policyCode, scenario, { trace: true })

if (result.passed) {
  console.log('Test passed!')
} else {
  console.log('Expected:', result.expected)
  console.log('Actual:', result.actual)
}
```

### Test Suites

Organize tests into suites:

```typescript
import { runTestSuite, createTestSuite } from '@/lib/policies/test-runner'

const suite = createTestSuite(
  'Rate Limiting Tests',
  'rate-limiting',
  '1.0.0',
  [scenario1, scenario2, scenario3]
)

const results = await runTestSuite(policyCode, suite, { parallel: true })

console.log(`${results.passed}/${results.totalTests} tests passed`)
console.log(`Pass rate: ${results.summary.passRate}%`)
```

### Test Coverage

Aim for:

- **Happy path**: Normal, successful requests
- **Edge cases**: Boundary conditions
- **Error cases**: Invalid inputs, missing data
- **Security**: Authorization failures
- **Performance**: High-volume scenarios

---

## Troubleshooting

### Common Issues

#### Policy Won't Validate

**Symptoms**: Validation fails with syntax errors

**Solutions**:
1. Check package declaration: `package apx.{name}`
2. Verify all braces are balanced
3. Import future.keywords if using modern syntax
4. Check for typos in rule names

#### Compilation Fails

**Symptoms**: WASM compilation errors

**Solutions**:
1. Ensure policy validates successfully first
2. Check for unsupported OPA features
3. Verify all imports are available
4. Review compilation error messages

#### Deployment Stuck

**Symptoms**: Deployment doesn't complete

**Solutions**:
1. Check router sync status
2. Verify GCS bucket permissions
3. Check Firestore connectivity
4. Review deployment logs

#### Canary Not Rolling Out

**Symptoms**: Canary stays at 0% traffic

**Solutions**:
1. Verify canary configuration was created
2. Check router is receiving canary config
3. Review canary status endpoint
4. Check for version conflicts

#### Auto-Rollback Not Triggering

**Symptoms**: High errors but no rollback

**Solutions**:
1. Verify auto-rollback is enabled
2. Check error rate threshold configuration
3. Ensure minimum requests threshold is met
4. Review monitor service logs

### Debug Checklist

- [ ] Policy validates successfully
- [ ] Compilation produces WASM bundle
- [ ] Version created in Firestore
- [ ] Router received sync command
- [ ] Traffic routing configured
- [ ] Monitoring data flowing
- [ ] Auto-rollback thresholds set

### Getting Help

- **Documentation**: Check this guide and API docs
- **Logs**: Review policy deployment logs
- **Status**: Check `/api/health` endpoint
- **Support**: Contact platform team

---

## API Reference

### Policy Validation

```
POST /api/v1/policies/validate
Content-Type: application/json

{
  "policy": "package apx.test...",
  "query": "data",
  "input": {}
}
```

### Policy Compilation

```
POST /api/v1/policies/compile
Content-Type: application/json

{
  "name": "rate-limiting",
  "version": "1.0.0",
  "policy": "package apx.rate_limiting...",
  "options": { "optimize": true }
}
```

### Version Management

```
POST /api/v1/policies/versions
GET /api/v1/policies/versions/{name}
GET /api/v1/policies/versions/{name}/{version}
GET /api/v1/policies/versions/{name}/latest
PATCH /api/v1/policies/versions/{id}
```

### Canary Deployment

```
POST /api/v1/canary/configs
GET /api/v1/canary/configs/{policy_name}
PATCH /api/v1/canary/configs/{policy_name}
POST /api/v1/canary/configs/{policy_name}/promote
POST /api/v1/canary/configs/{policy_name}/rollback
```

### Monitoring

```
GET /api/v1/monitor/status/{policy_name}
GET /api/v1/monitor/metrics/{policy_name}
POST /api/v1/monitor/thresholds
GET /api/v1/monitor/thresholds/{policy_name}
```

### GitOps

```
POST /api/v1/gitops/repositories
POST /api/v1/gitops/webhooks
POST /api/v1/gitops/pull-requests
POST /api/v1/gitops/sync/{repository_id}
```

---

## Appendix

### Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| Ctrl+S | Save policy |
| Ctrl+Shift+V | Validate policy |
| Ctrl+Shift+C | Compile policy |
| Ctrl+Shift+D | Deploy policy |
| Ctrl+/ | Toggle comment |

### Version Naming

Follow semantic versioning:

- **Major (1.x.x)**: Breaking changes
- **Minor (x.1.x)**: New features, backward compatible
- **Patch (x.x.1)**: Bug fixes, backward compatible

### Support

- **Documentation**: `/docs/portal/`
- **API Docs**: `/docs/api/`
- **Issues**: GitHub Issues
- **Email**: platform-team@example.com

---

**Document Version:** 1.0
**Last Updated:** 2025-11-12
**Maintained By:** APX Platform Team
