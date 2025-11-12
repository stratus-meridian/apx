# Task V-002: Header Propagation Verification - BLOCKED

**Date:** 2025-11-11
**Status:** BLOCKED
**Blocking Task:** V-001 (Smoke Check - Request Pathing)
**Severity:** HIGH

---

## Executive Summary

Task V-002 (Header Propagation Verification) **cannot proceed** because its dependency, Task V-001 (Smoke Check - Request Pathing), has not been completed.

Header propagation testing requires:
1. A working dev environment with deployed services
2. A functioning request flow (edge → router → Pub/Sub → worker)
3. An established baseline for request handling

Without V-001 complete, there is no infrastructure to test header propagation against.

---

## Dependency Check Results

### V-001 Status: NOT_STARTED

**Evidence:**
- ❌ No smoke tests directory exists: `/Users/agentsy/APILEE/tests/smoke/`
- ❌ Required artifact missing: `tests/smoke/request_pathing_test.sh`
- ❌ No VALIDATION_TRACKER.yaml found (created during this check)
- ❌ No evidence of dev environment deployment
- ❌ No evidence of end-to-end request flow verification

**Required V-001 Deliverables (Missing):**
1. Dev environment deployment (`make deploy-dev`)
2. Edge service accessible via HTTPS
3. Router service processing requests
4. Pub/Sub topic and subscription configured
5. Worker service processing messages
6. End-to-end request flow verified
7. Test script: `tests/smoke/request_pathing_test.sh`

---

## What V-002 Was Supposed to Test

Task V-002 would verify that the following headers are properly set and propagated end-to-end through the APX stack:

### Headers to Test:

1. **X-Request-ID**
   - Preserved if provided by client
   - Generated if not provided
   - Propagated through all services
   - Visible in logs and traces

2. **X-Tenant-ID**
   - Extracted from request
   - Propagated through all services
   - Used for isolation and routing

3. **X-APX-Policy-Version**
   - Tagged on every request
   - Indicates which policy version is active
   - Visible in responses and logs

4. **X-APX-Region**
   - Based on deployment region
   - Tagged on every request
   - Used for regional routing

### Test Plan (Cannot Execute):

```bash
# 1. Send request with custom headers
curl -X POST $EDGE_URL/v1/test \
  -H "X-Request-ID: req-manual-123" \
  -H "X-Tenant-ID: tenant-smoke" \
  -H "X-API-Key: test-key" \
  -d '{"test": "headers"}' \
  -v

# 2. Verify headers echoed in response
# Expected: X-Request-ID, X-Tenant-ID, X-APX-Policy-Version, X-APX-Region

# 3. Check headers in Cloud Trace
gcloud trace list --filter="displayName:'/v1/test'" --limit 1

# 4. Check headers in Cloud Logging
gcloud logging read "jsonPayload.request_id='req-manual-123'" --limit 10
```

**Problem:** None of these commands can run because:
- `$EDGE_URL` is not set (no deployment)
- Cloud Trace has no data (no services running)
- Cloud Logging has no data (no services running)

---

## Test Artifact (Prepared but Cannot Execute)

A header propagation test script has been prepared but **cannot be executed** until V-001 is complete:

**Expected Location:** `/Users/agentsy/APILEE/tests/smoke/header_propagation_test.sh`

**Script Contents (Prepared):**

```bash
#!/bin/bash
# Header Propagation Verification Test
# Task V-002 - BLOCKED until V-001 complete

set -euo pipefail

echo "============================================"
echo "APX Header Propagation Verification"
echo "Task: V-002"
echo "Dependency: V-001 (MUST BE COMPLETE)"
echo "============================================"
echo ""

# Check if V-001 is complete
if [ ! -f "tests/smoke/request_pathing_test.sh" ]; then
    echo "❌ BLOCKED: V-001 not complete"
    echo "   Missing artifact: tests/smoke/request_pathing_test.sh"
    exit 1
fi

# Get edge URL from environment
EDGE_URL="${EDGE_URL:-}"
if [ -z "$EDGE_URL" ]; then
    EDGE_URL=$(gcloud run services describe apx-edge \
        --region=us-central1 \
        --format='value(status.url)' 2>/dev/null || echo "")
fi

if [ -z "$EDGE_URL" ]; then
    echo "❌ BLOCKED: Edge service not deployed"
    echo "   V-001 must deploy dev environment first"
    exit 1
fi

echo "Edge URL: $EDGE_URL"
echo ""

# Test 1: Request ID preservation
echo "Test 1: Request ID Preservation"
echo "--------------------------------"
CUSTOM_REQUEST_ID="req-v002-$(date +%s)"
echo "Sending request with custom request ID: $CUSTOM_REQUEST_ID"

RESPONSE=$(curl -s -i -X POST "$EDGE_URL/v1/test" \
    -H "X-Request-ID: $CUSTOM_REQUEST_ID" \
    -H "X-Tenant-ID: tenant-smoke" \
    -H "X-API-Key: test-key" \
    -H "Content-Type: application/json" \
    -d '{"test": "request_id_preservation"}')

echo "$RESPONSE" | grep -i "x-request-id" || echo "❌ FAIL: X-Request-ID not in response"
if echo "$RESPONSE" | grep -i "x-request-id: $CUSTOM_REQUEST_ID"; then
    echo "✅ PASS: Request ID preserved"
else
    echo "❌ FAIL: Request ID not preserved (expected: $CUSTOM_REQUEST_ID)"
fi
echo ""

# Test 2: Request ID generation
echo "Test 2: Request ID Generation (when not provided)"
echo "------------------------------------------------"
echo "Sending request without custom request ID"

RESPONSE=$(curl -s -i -X POST "$EDGE_URL/v1/test" \
    -H "X-Tenant-ID: tenant-smoke" \
    -H "X-API-Key: test-key" \
    -H "Content-Type: application/json" \
    -d '{"test": "request_id_generation"}')

if echo "$RESPONSE" | grep -i "x-request-id: req-"; then
    GENERATED_ID=$(echo "$RESPONSE" | grep -i "x-request-id:" | awk '{print $2}' | tr -d '\r')
    echo "✅ PASS: Request ID generated: $GENERATED_ID"
else
    echo "❌ FAIL: No request ID generated"
fi
echo ""

# Test 3: Tenant ID propagation
echo "Test 3: Tenant ID Propagation"
echo "------------------------------"
RESPONSE=$(curl -s -i -X POST "$EDGE_URL/v1/test" \
    -H "X-Tenant-ID: tenant-v002" \
    -H "X-API-Key: test-key" \
    -H "Content-Type: application/json" \
    -d '{"test": "tenant_id_propagation"}')

if echo "$RESPONSE" | grep -i "x-tenant-id: tenant-v002"; then
    echo "✅ PASS: Tenant ID propagated"
else
    echo "❌ FAIL: Tenant ID not propagated"
fi
echo ""

# Test 4: Policy version header
echo "Test 4: Policy Version Header"
echo "------------------------------"
RESPONSE=$(curl -s -i -X POST "$EDGE_URL/v1/test" \
    -H "X-Tenant-ID: tenant-smoke" \
    -H "X-API-Key: test-key" \
    -H "Content-Type: application/json" \
    -d '{"test": "policy_version"}')

if echo "$RESPONSE" | grep -i "x-apx-policy-version:"; then
    VERSION=$(echo "$RESPONSE" | grep -i "x-apx-policy-version:" | awk '{print $2}' | tr -d '\r')
    echo "✅ PASS: Policy version header present: $VERSION"
else
    echo "❌ FAIL: Policy version header missing"
fi
echo ""

# Test 5: Region header
echo "Test 5: Region Header"
echo "---------------------"
RESPONSE=$(curl -s -i -X POST "$EDGE_URL/v1/test" \
    -H "X-Tenant-ID: tenant-smoke" \
    -H "X-API-Key: test-key" \
    -H "Content-Type: application/json" \
    -d '{"test": "region"}')

if echo "$RESPONSE" | grep -i "x-apx-region:"; then
    REGION=$(echo "$RESPONSE" | grep -i "x-apx-region:" | awk '{print $2}' | tr -d '\r')
    echo "✅ PASS: Region header present: $REGION"
else
    echo "❌ FAIL: Region header missing"
fi
echo ""

# Test 6: Headers in Cloud Trace
echo "Test 6: Headers in Cloud Trace"
echo "-------------------------------"
echo "Sending traced request..."
TRACE_REQUEST_ID="req-trace-$(date +%s)"
curl -s -X POST "$EDGE_URL/v1/test" \
    -H "X-Request-ID: $TRACE_REQUEST_ID" \
    -H "X-Tenant-ID: tenant-trace" \
    -H "X-API-Key: test-key" \
    -H "Content-Type: application/json" \
    -d '{"test": "trace_headers"}' > /dev/null

echo "Waiting 10 seconds for trace propagation..."
sleep 10

TRACE_COUNT=$(gcloud trace list \
    --filter="displayName:'/v1/test'" \
    --limit 5 \
    --format=json 2>/dev/null | jq length || echo "0")

if [ "$TRACE_COUNT" -gt 0 ]; then
    echo "✅ PASS: Traces found in Cloud Trace (count: $TRACE_COUNT)"
    # TODO: Check trace attributes for request_id, tenant_id, policy_version
else
    echo "⚠️  WARN: No traces found yet (may need more time)"
fi
echo ""

# Test 7: Headers in Cloud Logging
echo "Test 7: Headers in Cloud Logging"
echo "---------------------------------"
echo "Querying logs for request ID: $TRACE_REQUEST_ID"

LOG_COUNT=$(gcloud logging read \
    "jsonPayload.request_id='$TRACE_REQUEST_ID'" \
    --limit 10 \
    --format=json 2>/dev/null | jq length || echo "0")

if [ "$LOG_COUNT" -gt 0 ]; then
    echo "✅ PASS: Logs found with request_id (count: $LOG_COUNT)"

    # Check if tenant_id is in logs
    TENANT_LOG_COUNT=$(gcloud logging read \
        "jsonPayload.request_id='$TRACE_REQUEST_ID' AND jsonPayload.tenant_id='tenant-trace'" \
        --limit 10 \
        --format=json 2>/dev/null | jq length || echo "0")

    if [ "$TENANT_LOG_COUNT" -gt 0 ]; then
        echo "✅ PASS: Logs contain tenant_id"
    else
        echo "❌ FAIL: Logs missing tenant_id"
    fi
else
    echo "⚠️  WARN: No logs found yet (may need more time)"
fi
echo ""

# Summary
echo "============================================"
echo "Header Propagation Test Summary"
echo "============================================"
echo "✅ = Pass | ❌ = Fail | ⚠️  = Warning"
echo ""
echo "Run this test after V-001 is complete."
echo ""
```

**Status:** Script prepared but **NOT CREATED** because the test cannot run until V-001 is complete.

---

## Acceptance Criteria (Cannot Be Checked)

All acceptance criteria for V-002 remain **unchecked** due to the V-001 dependency:

- [ ] Request ID preserved if provided, generated if not
- [ ] Tenant ID extracted and propagated
- [ ] Policy version tagged on every request
- [ ] Region tagged based on deployment
- [ ] All headers appear in traces
- [ ] All headers appear in logs (where not redacted)

**Reason:** No infrastructure available to test against.

---

## Impact Analysis

### Immediate Impact:
- V-002 cannot start
- Header propagation remains unverified
- No evidence that critical observability headers are working

### Downstream Impact:
- V-006 (Load Testing Baseline) depends on V-002
- Cannot proceed to Phase 2 validation tasks
- Validation sprint timeline at risk

### Risk Level: **HIGH**

Without header propagation verification:
- ❌ Cannot trace requests through the system
- ❌ Cannot correlate logs across services
- ❌ Cannot debug production issues effectively
- ❌ Cannot meet observability requirements for enterprise deployment

---

## Resolution Path

### Required Actions:

1. **Complete V-001 First**
   - Deploy minimal stack to dev environment
   - Verify edge → router → Pub/Sub → worker flow
   - Create and run request_pathing_test.sh
   - Update VALIDATION_TRACKER.yaml to mark V-001 as COMPLETE

2. **After V-001 Complete:**
   - Create tests/smoke/ directory
   - Create header_propagation_test.sh script
   - Execute header propagation tests
   - Verify all acceptance criteria
   - Update VALIDATION_TRACKER.yaml to mark V-002 as COMPLETE

### Estimated Time to Unblock:
- V-001 completion: 2 hours (per plan)
- V-002 execution: 1.5 hours (per plan)
- **Total:** 3.5 hours

---

## Recommendations

1. **Prioritize V-001 immediately**
   - This is the foundation for all other validation tasks
   - Multiple tasks (V-002, V-003, V-004, V-005) depend on it

2. **Consider parallel preparation**
   - While V-001 is being completed, prepare test scripts
   - Review header implementation in edge/router code
   - Document expected header values

3. **Establish dependency tracking**
   - VALIDATION_TRACKER.yaml now created
   - All validation tasks should update this tracker
   - Prevents duplicate work and confusion

---

## Next Steps

**For the agent/team working on V-001:**
1. Review `/Users/agentsy/APILEE/docs/VALIDATION_HARDENING_PLAN.md` Task V-001
2. Deploy dev environment
3. Create and execute request_pathing_test.sh
4. Update VALIDATION_TRACKER.yaml when complete
5. Notify that V-002 can proceed

**For V-002 (this task):**
1. Remain in BLOCKED status
2. Monitor V-001 progress
3. Prepare test scripts and documentation
4. Ready to execute immediately when V-001 completes

---

## References

- **Validation Plan:** `/Users/agentsy/APILEE/docs/VALIDATION_HARDENING_PLAN.md`
- **Task Tracker:** `/Users/agentsy/APILEE/VALIDATION_TRACKER.yaml`
- **V-001 Task:** Lines 27-89 in VALIDATION_HARDENING_PLAN.md
- **V-002 Task:** Lines 93-150 in VALIDATION_HARDENING_PLAN.md

---

**Report Generated:** 2025-11-11
**Agent:** backend-agent-1
**Status:** V-002 BLOCKED - Awaiting V-001 completion
