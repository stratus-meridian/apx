# Phase 2 Integration Quick Reference

**Version:** 1.0
**Date:** 2025-11-12
**Status:** Ready for Implementation

---

## Overview

Quick reference for integrating Portal M5 Policy UI with Phase 2 Policy Engine Backend.

**Phase 2 Status:** ✅ COMPLETE (242 tests passing, 82.7% coverage)

---

## Phase 2 Backend APIs

### 1. OPA Runtime API

**Validate Policy:**
```bash
POST /api/v1/policies/validate
Content-Type: application/json
Authorization: Bearer <token>

{
  "policy": "package apx.test\n...",
  "query": "data.apx.allow",
  "input": { "request": { "method": "GET" } }
}
```

**Response:**
```json
{
  "valid": true,
  "errors": [],
  "warnings": [],
  "result": { "allow": true }
}
```

### 2. Policy Compiler API

**Compile to WASM:**
```bash
POST /api/v1/policies/compile
Content-Type: application/json

{
  "name": "rate-limiting",
  "version": "1.2.3",
  "policy": "package apx.rate_limiting\n...",
  "options": { "optimize": true }
}
```

**Response:**
```json
{
  "success": true,
  "hash": "sha256:abc123...",
  "artifact_url": "gs://bucket/policies/rate-limiting/1.2.3/abc123.wasm",
  "size_bytes": 142839,
  "metadata": {
    "compiler_version": "0.59.0",
    "entrypoints": ["apx.rate_limiting.allow"],
    "dependencies": ["future.keywords"]
  }
}
```

### 3. Version Management API

**Create Version:**
```bash
POST /api/v1/policies/versions
Content-Type: application/json

{
  "name": "rate-limiting",
  "version": "1.2.3",
  "hash": "sha256:abc123...",
  "artifact_url": "gs://...",
  "status": "active",
  "compat": "backward"
}
```

**Get Latest Version:**
```bash
GET /api/v1/policies/versions/{name}/latest
```

**List All Versions:**
```bash
GET /api/v1/policies/versions/{name}
```

### 4. Canary Deployment API

**Create Canary:**
```bash
POST /api/v1/canary/configs
Content-Type: application/json

{
  "policy_name": "rate-limiting",
  "stable_version": "1.2.3",
  "canary_version": "1.3.0",
  "canary_percentage": 10,
  "created_by": "user@example.com"
}
```

**Update Canary %:**
```bash
PATCH /api/v1/canary/configs/{policy_name}
Content-Type: application/json

{
  "canary_percentage": 25
}
```

**Promote Canary:**
```bash
POST /api/v1/canary/configs/{policy_name}/promote
```

**Rollback:**
```bash
POST /api/v1/canary/configs/{policy_name}/rollback
Content-Type: application/json

{
  "reason": "High error rate detected",
  "triggered_by": "user@example.com"
}
```

### 5. Monitor API

**Get Status:**
```bash
GET /api/v1/monitor/status/{policy_name}
```

**Response:**
```json
{
  "policy_name": "rate-limiting",
  "canary_version": "1.3.0",
  "stable_version": "1.2.3",
  "current_error_rate": 2.3,
  "threshold": 5.0,
  "requests_analyzed": 1542,
  "last_check": "2025-11-12T10:30:00Z",
  "status": "healthy",
  "rollback_triggered": false
}
```

**Configure Threshold:**
```bash
POST /api/v1/monitor/thresholds
Content-Type: application/json

{
  "policy_name": "rate-limiting",
  "error_rate_threshold": 5.0,
  "check_interval_seconds": 300,
  "min_requests": 100,
  "rollback_action": "disable"
}
```

### 6. Router Sync API

**Sync Policy:**
```bash
POST /api/v1/router/policies/sync
Content-Type: application/json

{
  "policy_name": "rate-limiting",
  "version": "1.3.0",
  "artifact_url": "gs://bucket/policies/rate-limiting/1.3.0/abc123.wasm",
  "priority": "normal"
}
```

### 7. GCS Artifacts API

**Upload Bundle:**
```bash
POST /api/v1/artifacts/upload
Content-Type: multipart/form-data

file: <binary WASM data>
name: rate-limiting
version: 1.3.0
hash: sha256:abc123...
```

**Download Bundle:**
```bash
GET /api/v1/artifacts/download/{name}/{version}
```

### 8. GitOps API

**Configure Webhook:**
```bash
POST /api/v1/gitops/webhooks
Content-Type: application/json

{
  "repository_id": "repo-123",
  "provider": "github",
  "owner": "myorg",
  "repo": "policies",
  "branch": "main",
  "events": ["push", "pull_request"]
}
```

**Trigger Deployment:**
```bash
POST /api/v1/gitops/deploy
Content-Type: application/json

{
  "repository_id": "repo-123",
  "branch": "main",
  "policy_paths": ["policies/rate-limiting.yaml"]
}
```

---

## Portal Connector Updates

### File: `lib/policies/rego-validator.ts`

**Before:**
```typescript
export async function validateRegoPolicy(policy: string) {
  // Client-side only
  return performBasicValidation(policy)
}
```

**After:**
```typescript
export async function validateRegoPolicy(policy: string) {
  const response = await fetch('/api/v1/policies/validate', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${getSessionToken()}`
    },
    body: JSON.stringify({ policy })
  })
  return await response.json()
}
```

### File: `lib/policies/policy-compiler.ts`

**Before:**
```typescript
export async function compilePolicy(policy: string) {
  // Returns mock WASM
  return { success: true, wasm: mockWasm }
}
```

**After:**
```typescript
export async function compilePolicy(policy: string, options: CompilationOptions) {
  const response = await fetch('/api/v1/policies/compile', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${getSessionToken()}`
    },
    body: JSON.stringify({
      name: options.name,
      version: options.version,
      policy,
      options: { optimize: true }
    })
  })

  const result = await response.json()

  // Download WASM from GCS
  const wasmBlob = await downloadBundle(result.artifact_url)

  return {
    success: true,
    wasm: new Uint8Array(await wasmBlob.arrayBuffer()),
    bundle: result.hash,
    metadata: result.metadata
  }
}
```

### File: `lib/policies/version-manager.ts`

**Before:**
```typescript
export async function createPolicyVersion(version: PolicyVersion) {
  // In-memory storage
  versions.push(version)
  return version.id
}
```

**After:**
```typescript
export async function createPolicyVersion(version: PolicyVersion) {
  const response = await fetch('/api/v1/policies/versions', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${getSessionToken()}`
    },
    body: JSON.stringify({
      name: version.name,
      version: version.version,
      hash: version.metadata?.hash,
      artifact_url: version.metadata?.artifact_url,
      status: version.status
    })
  })

  const result = await response.json()
  return result.id
}
```

### File: `lib/policies/test-runner.ts`

**Before:**
```typescript
export async function runTestScenario(policy: string, scenario: TestScenario) {
  // Mock evaluation
  return evaluatePolicyMock(policy, scenario.input)
}
```

**After:**
```typescript
export async function runTestScenario(policy: string, scenario: TestScenario) {
  const response = await fetch('/api/v1/policies/evaluate', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${getSessionToken()}`
    },
    body: JSON.stringify({
      policy,
      input: scenario.input,
      data: scenario.data,
      trace: true
    })
  })

  const result = await response.json()
  const passed = compareResults(result, scenario.expectedResult)

  return {
    scenarioId: scenario.id,
    scenarioName: scenario.name,
    passed,
    actual: result,
    expected: scenario.expectedResult,
    trace: result.trace,
    duration: Date.now() - startTime
  }
}
```

### File: `lib/policies/gitops-client.ts`

**Before:**
```typescript
export async function configureWebhook(repo: GitRepository, events: string[]) {
  // Mock webhook
  return { id: 'webhook-123', url: 'mock-url', active: true }
}
```

**After:**
```typescript
export async function configureWebhook(repo: GitRepository, events: string[]) {
  const response = await fetch('/api/v1/gitops/webhooks', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${getSessionToken()}`
    },
    body: JSON.stringify({
      repository_id: repo.id,
      provider: repo.provider,
      owner: repo.owner,
      repo: repo.name,
      branch: repo.branch,
      events
    })
  })

  return await response.json()
}
```

---

## End-to-End Deployment Flow

### Complete Policy Deployment

```typescript
// 1. Validate policy
const validationResult = await validateRegoPolicy(policyCode)
if (!validationResult.valid) {
  throw new Error('Validation failed')
}

// 2. Compile to WASM
const compilationResult = await compilePolicy(policyCode, {
  name: 'rate-limiting',
  version: '1.3.0',
  optimize: true
})

// 3. Create version metadata
const versionId = await createPolicyVersion({
  id: 'rate-limiting@1.3.0',
  name: 'rate-limiting',
  version: '1.3.0',
  content: policyCode,
  compiled: compilationResult.bundle,
  status: 'draft',
  createdAt: new Date(),
  createdBy: getCurrentUser(),
  metadata: {
    hash: compilationResult.metadata.hash,
    artifact_url: compilationResult.metadata.artifact_url,
    size_bytes: compilationResult.metadata.size
  }
})

// 4. Configure canary (optional)
const canaryConfig = await createCanaryConfig({
  policy_name: 'rate-limiting',
  stable_version: '1.2.3',
  canary_version: '1.3.0',
  canary_percentage: 10,
  created_by: getCurrentUser()
})

// 5. Sync to router
await syncPolicyToRouter({
  policy_name: 'rate-limiting',
  version: '1.3.0',
  artifact_url: compilationResult.metadata.artifact_url,
  priority: 'normal'
})

// 6. Mark as active
await updateVersionStatus(versionId, 'active')

// 7. Monitor deployment
const monitorStatus = await getMonitorStatus('rate-limiting')
```

---

## Monitoring & Rollback

### Real-time Monitoring Hook

```typescript
export function useCanaryMonitoring(policyName: string) {
  const [status, setStatus] = useState(null)
  const [metrics, setMetrics] = useState(null)

  useEffect(() => {
    const fetchStatus = async () => {
      const config = await getCanaryConfig(policyName)
      const monitorStatus = await getMonitorStatus(policyName)
      const metrics = await getDeploymentMetrics(policyName, config.canary_version)

      setStatus({ ...config, monitor: monitorStatus })
      setMetrics(metrics)
    }

    fetchStatus()
    const interval = setInterval(fetchStatus, 10000) // Poll every 10s

    return () => clearInterval(interval)
  }, [policyName])

  return { status, metrics }
}
```

### Manual Rollback

```typescript
// Trigger rollback
const result = await triggerRollback('rate-limiting', 'High error rate detected')

// Result: { success: true, message: 'Rollback initiated' }
```

### Auto-Rollback Configuration

```typescript
// Configure auto-rollback thresholds
await fetch('/api/v1/monitor/thresholds', {
  method: 'POST',
  body: JSON.stringify({
    policy_name: 'rate-limiting',
    error_rate_threshold: 5.0,      // 5% error rate
    check_interval_seconds: 300,    // Check every 5 minutes
    min_requests: 100,              // Min 100 requests before evaluating
    rollback_action: 'disable'      // Set canary to 0%
  })
})
```

---

## Environment Variables

### Portal .env

```bash
# Phase 2 APIs
NEXT_PUBLIC_APX_POLICY_API_URL=https://api.apx.dev/v1/policies
NEXT_PUBLIC_APX_CANARY_API_URL=https://api.apx.dev/v1/canary
NEXT_PUBLIC_APX_MONITOR_API_URL=https://api.apx.dev/v1/monitor

# GCS
GCS_ARTIFACTS_BUCKET=apx-prod-apx-artifacts
GCS_ARTIFACTS_PATH=policies/

# Firestore
FIRESTORE_POLICY_COLLECTION=policy_versions
FIRESTORE_CANARY_COLLECTION=canary_configs

# GitOps
GITHUB_APP_ID=xxx
GITHUB_APP_SECRET=xxx
GITLAB_APP_ID=xxx
GITLAB_APP_SECRET=xxx

# Cloud Build
CLOUD_BUILD_PROJECT_ID=apx-prod
CLOUD_BUILD_TRIGGER_ID=apx-policy-compiler

# Monitor
MONITOR_SERVICE_URL=https://apx-monitor-xxxxxxx.run.app
```

---

## Testing Phase 2 Integration

### Integration Test

```typescript
describe('Phase 2 Integration', () => {
  test('Complete deployment flow', async () => {
    const policy = `
      package apx.test
      import future.keywords.if
      default allow := false
      allow if {
        input.request.method == "GET"
      }
    `

    // Validate
    const valid = await validateRegoPolicy(policy)
    expect(valid.valid).toBe(true)

    // Compile
    const compiled = await compilePolicy(policy, {
      name: 'test',
      version: '1.0.0'
    })
    expect(compiled.success).toBe(true)

    // Create version
    const versionId = await createPolicyVersion({
      name: 'test',
      version: '1.0.0',
      content: policy,
      status: 'draft'
    })
    expect(versionId).toMatch(/^test@1\.0\.0/)

    // Configure canary
    const canary = await createCanaryConfig({
      policy_name: 'test',
      stable_version: '0.0.0',
      canary_version: '1.0.0',
      canary_percentage: 10
    })
    expect(canary.canary_percentage).toBe(10)

    // Sync to router
    await syncPolicyToRouter({
      policy_name: 'test',
      version: '1.0.0',
      artifact_url: compiled.metadata.artifact_url
    })

    // Verify
    const deployed = await getPolicyVersion('test', '1.0.0')
    expect(deployed.status).toBe('active')
  })
})
```

---

## Implementation Timeline

### Phase 2A: Core APIs (Week 1)
- Day 1-2: OPA validation integration
- Day 2-3: Policy compilation integration
- Day 3-4: Firestore version management
- Day 4-5: Integration testing

### Phase 2B: Canary & Monitoring (Week 2)
- Day 6-7: Canary configuration
- Day 7-8: Deployment monitoring
- Day 8-9: Auto-rollback integration
- Day 9-10: End-to-end deployment

### Phase 2C: GitOps & Testing (Week 3)
- Day 11-12: GitOps integration
- Day 12-13: Policy testing
- Day 13-14: CLI integration
- Day 14-15: Documentation

---

## Common Issues & Solutions

### Issue: OPA Validation Timeout

**Solution:**
```typescript
// Add timeout and fallback
const controller = new AbortController()
const timeoutId = setTimeout(() => controller.abort(), 5000)

try {
  const response = await fetch('/api/v1/policies/validate', {
    signal: controller.signal,
    ...
  })
} catch (error) {
  if (error.name === 'AbortError') {
    // Fallback to client-side validation
    return performBasicValidation(policy)
  }
  throw error
} finally {
  clearTimeout(timeoutId)
}
```

### Issue: WASM Compilation Fails

**Solution:**
```typescript
// Check policy syntax first
const validation = await validateRegoPolicy(policy)
if (!validation.valid) {
  return {
    success: false,
    errors: validation.errors.map(e => e.message)
  }
}

// Retry compilation once on failure
let compilationResult
try {
  compilationResult = await compilePolicy(policy, options)
} catch (error) {
  logger.warn('Compilation failed, retrying...', { error })
  await new Promise(resolve => setTimeout(resolve, 2000))
  compilationResult = await compilePolicy(policy, options)
}
```

### Issue: Canary Not Rolling Back

**Solution:**
```typescript
// Check monitor status
const status = await getMonitorStatus(policyName)
console.log('Monitor status:', status)

// Verify threshold configuration
const thresholds = await fetch(`/api/v1/monitor/thresholds/${policyName}`)
const config = await thresholds.json()
console.log('Threshold config:', config)

// Manually trigger if needed
if (status.current_error_rate > config.error_rate_threshold) {
  await triggerRollback(policyName, 'Manual rollback due to high errors')
}
```

---

## Success Criteria

### Integration Completeness
- [ ] All 8 Phase 2 APIs integrated
- [ ] All 5 portal connectors updated
- [ ] End-to-end flow working
- [ ] Canary rollout functional
- [ ] Auto-rollback working
- [ ] GitOps workflow functional
- [ ] Policy testing via OPA
- [ ] N/N-1 versioning supported

### Performance Targets
- Policy validation: < 500ms
- Policy compilation: < 5 seconds
- Version creation: < 1 second
- Canary config update: < 2 seconds
- Router sync: < 10 seconds
- Dashboard load: < 2 seconds

### Quality Targets
- Integration tests: 100% passing
- Error handling: 100% coverage
- Fallback mechanisms: All critical paths
- Documentation: 100% complete

---

## Additional Resources

### Documentation
- Full Integration Plan: `/docs/trackers/portal/PORTAL_BACKEND_INTEGRATION_PLAN.md`
- Phase 2 Completion Certificate: `/docs/trackers/phase2/PHASE2_COMPLETION_CERTIFICATE.md`
- Backend Architecture: `/.private/docs/ARCHITECTURE.md`

### Backend Code
- OPA Engine: `/control/pkg/opa/`
- Policy Compiler: `/.private/control/compiler/`
- Canary System: `/.private/control/canary/`
- Monitor Service: `/.private/control/monitor/`
- Firestore Client: `/.private/control/firestore/`

### Portal Code
- Policy Validators: `/.private/portal/lib/policies/rego-validator.ts`
- Policy Compiler: `/.private/portal/lib/policies/policy-compiler.ts`
- Version Manager: `/.private/portal/lib/policies/version-manager.ts`
- Test Runner: `/.private/portal/lib/policies/test-runner.ts`
- GitOps Client: `/.private/portal/lib/policies/gitops-client.ts`

---

**Document Version:** 1.0
**Last Updated:** 2025-11-12
**Status:** Ready for Use
