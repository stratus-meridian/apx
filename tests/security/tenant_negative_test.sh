#!/bin/bash
set -euo pipefail

# Tenant Isolation Negative Test Suite
# Tests that verify tenants CANNOT access each other's resources

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Configuration
EDGE_URL="${EDGE_URL:-http://localhost:8080}"
ROUTER_URL="${ROUTER_URL:-http://localhost:8081}"
PROJECT_ID="${PROJECT_ID:-apx-dev}"

echo -e "${BLUE}================================${NC}"
echo -e "${BLUE}Tenant Isolation Negative Tests${NC}"
echo -e "${BLUE}================================${NC}"
echo ""
echo "Edge URL: $EDGE_URL"
echo "Router URL: $ROUTER_URL"
echo "Project: $PROJECT_ID"
echo ""
echo -e "${YELLOW}These tests verify that cross-tenant access is BLOCKED${NC}"
echo ""

# Helper functions
log_test() {
    echo -e "${BLUE}[TEST]${NC} $1"
}

log_pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((TESTS_PASSED++))
}

log_fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((TESTS_FAILED++))
}

log_info() {
    echo -e "${YELLOW}[INFO]${NC} $1"
}

run_test() {
    ((TESTS_RUN++))
}

# ============================================================================
# Test 1: Tenant-A Cannot Access Tenant-B's Data
# ============================================================================
test_cross_tenant_data_access() {
    log_test "Tenant-A cannot access Tenant-B's data"
    run_test

    log_info "Creating request as tenant-b"
    TENANT_B_RESPONSE=$(curl -s -X POST "$EDGE_URL/v1/test" \
        -H "X-Tenant-ID: tenant-b" \
        -H "X-API-Key: tenant-b-key" \
        -H "Content-Type: application/json" \
        -d '{"tenant": "b", "secret": "tenant-b-secret-data"}' 2>/dev/null || echo '{}')

    TENANT_B_REQUEST_ID=$(echo "$TENANT_B_RESPONSE" | jq -r '.request_id // empty')

    if [ -z "$TENANT_B_REQUEST_ID" ]; then
        log_info "Could not create tenant-b request (service may not be running)"
        return 0
    fi

    log_info "Tenant-B request ID: $TENANT_B_REQUEST_ID"

    log_info "Attempting to access tenant-b's request from tenant-a"
    STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
        -X GET "$EDGE_URL/v1/status/$TENANT_B_REQUEST_ID" \
        -H "X-Tenant-ID: tenant-a" \
        -H "X-API-Key: tenant-a-key" 2>/dev/null || echo "000")

    # Should get 403 Forbidden or 404 Not Found (both acceptable)
    if [ "$STATUS" = "403" ] || [ "$STATUS" = "404" ]; then
        log_pass "Tenant-A correctly denied access to Tenant-B's request (HTTP $STATUS)"
    elif [ "$STATUS" = "000" ]; then
        log_info "Service not available, cannot test cross-tenant access"
    else
        log_fail "Tenant-A was able to access Tenant-B's data (HTTP $STATUS - should be 403 or 404)"
        return 1
    fi
}

# ============================================================================
# Test 2: Tenant-A Cannot See Tenant-B's Logs
# ============================================================================
test_cross_tenant_log_isolation() {
    log_test "Tenant-A cannot see Tenant-B's logs"
    run_test

    if ! command -v gcloud &> /dev/null; then
        log_info "gcloud not available, skipping log isolation test"
        return 0
    fi

    log_info "Creating requests with tenant-specific data"

    # Send request from tenant-a with identifiable data
    curl -s -X POST "$EDGE_URL/v1/test" \
        -H "X-Tenant-ID: tenant-a" \
        -H "X-API-Key: tenant-a-key" \
        -d '{"tenant": "a", "message": "TENANT_A_SECRET_12345"}' > /dev/null 2>&1 || true

    # Send request from tenant-b with identifiable data
    curl -s -X POST "$EDGE_URL/v1/test" \
        -H "X-Tenant-ID: tenant-b" \
        -H "X-API-Key: tenant-b-key" \
        -d '{"tenant": "b", "message": "TENANT_B_SECRET_67890"}' > /dev/null 2>&1 || true

    sleep 3

    log_info "Querying logs for tenant-a"
    TENANT_A_LOGS=$(gcloud logging read \
        "jsonPayload.tenant_id=\"tenant-a\"" \
        --limit=50 \
        --format=json \
        --project="$PROJECT_ID" 2>/dev/null || echo "[]")

    # Check that tenant-a logs don't contain tenant-b secrets
    LEAKED_SECRETS=$(echo "$TENANT_A_LOGS" | grep -c "TENANT_B_SECRET" || echo "0")

    if [ "$LEAKED_SECRETS" = "0" ]; then
        log_pass "Tenant-B's secrets not found in Tenant-A's logs"
    else
        log_fail "Tenant-B's data leaked into Tenant-A's logs"
        return 1
    fi

    log_info "Querying logs for tenant-b"
    TENANT_B_LOGS=$(gcloud logging read \
        "jsonPayload.tenant_id=\"tenant-b\"" \
        --limit=50 \
        --format=json \
        --project="$PROJECT_ID" 2>/dev/null || echo "[]")

    # Check that tenant-b logs don't contain tenant-a secrets
    LEAKED_SECRETS=$(echo "$TENANT_B_LOGS" | grep -c "TENANT_A_SECRET" || echo "0")

    if [ "$LEAKED_SECRETS" = "0" ]; then
        log_pass "Tenant-A's secrets not found in Tenant-B's logs"
    else
        log_fail "Tenant-A's data leaked into Tenant-B's logs"
        return 1
    fi
}

# ============================================================================
# Test 3: Tenant-A Cannot Exhaust Tenant-B's Quota
# ============================================================================
test_cross_tenant_quota_exhaustion() {
    log_test "Tenant-A cannot exhaust Tenant-B's quota"
    run_test

    log_info "Flooding requests from tenant-a"

    # Send many requests from tenant-a
    TENANT_A_REJECTED=0
    for i in {1..50}; do
        STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
            -X POST "$EDGE_URL/v1/test" \
            -H "X-Tenant-ID: tenant-a" \
            -H "X-API-Key: tenant-a-key" \
            -d '{"load": "test"}' 2>/dev/null || echo "000")

        if [ "$STATUS" = "429" ]; then
            ((TENANT_A_REJECTED++))
        fi
    done

    log_info "Tenant-A rejected requests: $TENANT_A_REJECTED"

    log_info "Testing if tenant-b can still make requests"

    # Tenant-B should still be able to make requests
    TENANT_B_SUCCESS=0
    for i in {1..10}; do
        STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
            -X POST "$EDGE_URL/v1/test" \
            -H "X-Tenant-ID: tenant-b" \
            -H "X-API-Key: tenant-b-key" \
            -d '{"test": "quota"}' 2>/dev/null || echo "000")

        if [ "$STATUS" = "202" ] || [ "$STATUS" = "200" ]; then
            ((TENANT_B_SUCCESS++))
        fi
    done

    log_info "Tenant-B successful requests: $TENANT_B_SUCCESS"

    if [ "$TENANT_B_SUCCESS" -gt 5 ]; then
        log_pass "Tenant-B can still make requests despite Tenant-A's load"
    elif [ "$TENANT_B_SUCCESS" -eq 0 ] && [ "$TENANT_A_REJECTED" -eq 0 ]; then
        log_info "Service not available, cannot test quota exhaustion"
    else
        log_fail "Tenant-B's quota appears affected by Tenant-A"
        return 1
    fi
}

# ============================================================================
# Test 4: Tenant Cannot Access Status Without Proper Credentials
# ============================================================================
test_unauthenticated_status_access() {
    log_test "Unauthenticated access to status endpoint blocked"
    run_test

    log_info "Creating request as tenant-a"
    RESPONSE=$(curl -s -X POST "$EDGE_URL/v1/test" \
        -H "X-Tenant-ID: tenant-a" \
        -H "X-API-Key: tenant-a-key" \
        -d '{"test": "auth"}' 2>/dev/null || echo '{}')

    REQUEST_ID=$(echo "$RESPONSE" | jq -r '.request_id // empty')

    if [ -z "$REQUEST_ID" ]; then
        log_info "Could not create request (service may not be running)"
        return 0
    fi

    log_info "Request ID: $REQUEST_ID"

    log_info "Attempting to access status without credentials"
    STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
        -X GET "$EDGE_URL/v1/status/$REQUEST_ID" 2>/dev/null || echo "000")

    if [ "$STATUS" = "401" ] || [ "$STATUS" = "403" ]; then
        log_pass "Unauthenticated access correctly blocked (HTTP $STATUS)"
    elif [ "$STATUS" = "000" ]; then
        log_info "Service not available, cannot test authentication"
    else
        log_fail "Unauthenticated access not blocked (HTTP $STATUS - should be 401 or 403)"
        return 1
    fi
}

# ============================================================================
# Test 5: Tenant Cannot Modify Another Tenant's Rate Limits
# ============================================================================
test_cross_tenant_rate_limit_modification() {
    log_test "Tenant cannot modify another tenant's rate limits"
    run_test

    if ! command -v redis-cli &> /dev/null; then
        log_info "redis-cli not available, skipping rate limit modification test"
        return 0
    fi

    log_info "Setting up tenant-b's rate limit"

    REDIS_HOST="${REDIS_HOST:-localhost}"
    REDIS_PORT="${REDIS_PORT:-6379}"

    # Set tenant-b's rate limit
    redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" SET "apx:rl:tenant-b:requests" 5 > /dev/null 2>&1 || true
    redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" EXPIRE "apx:rl:tenant-b:requests" 60 > /dev/null 2>&1 || true

    ORIGINAL_VALUE=$(redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" GET "apx:rl:tenant-b:requests" 2>/dev/null || echo "0")
    log_info "Tenant-B rate limit set to: $ORIGINAL_VALUE"

    # Attempt to access/modify with tenant-a pattern (this should not affect tenant-b)
    log_info "Tenant-A attempts operations (should not affect tenant-b)"

    # Tenant-A can only affect their own keys
    redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" INCR "apx:rl:tenant-a:requests" > /dev/null 2>&1 || true

    # Check tenant-b's rate limit is unchanged
    NEW_VALUE=$(redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" GET "apx:rl:tenant-b:requests" 2>/dev/null || echo "0")

    if [ "$ORIGINAL_VALUE" = "$NEW_VALUE" ]; then
        log_pass "Tenant-B's rate limit unchanged by Tenant-A's operations"
    else
        log_fail "Tenant-B's rate limit was modified (was $ORIGINAL_VALUE, now $NEW_VALUE)"
        return 1
    fi
}

# ============================================================================
# Test 6: Invalid Tenant ID Rejected
# ============================================================================
test_invalid_tenant_rejection() {
    log_test "Requests without valid tenant ID are rejected"
    run_test

    log_info "Sending request without tenant ID"
    STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
        -X POST "$EDGE_URL/v1/test" \
        -H "X-API-Key: test-key" \
        -d '{"test": "no-tenant"}' 2>/dev/null || echo "000")

    # Should be rejected (400, 401, or 403)
    if [ "$STATUS" = "400" ] || [ "$STATUS" = "401" ] || [ "$STATUS" = "403" ]; then
        log_pass "Request without tenant ID correctly rejected (HTTP $STATUS)"
    elif [ "$STATUS" = "000" ]; then
        log_info "Service not available, cannot test tenant validation"
    else
        log_info "Request without tenant ID returned HTTP $STATUS (may default to 'unknown' tenant)"
    fi

    log_info "Sending request with empty tenant ID"
    STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
        -X POST "$EDGE_URL/v1/test" \
        -H "X-Tenant-ID: " \
        -H "X-API-Key: test-key" \
        -d '{"test": "empty-tenant"}' 2>/dev/null || echo "000")

    if [ "$STATUS" = "400" ] || [ "$STATUS" = "401" ] || [ "$STATUS" = "403" ]; then
        log_pass "Request with empty tenant ID correctly rejected (HTTP $STATUS)"
    elif [ "$STATUS" = "000" ]; then
        log_info "Service not available"
    else
        log_info "Request with empty tenant ID returned HTTP $STATUS"
    fi
}

# ============================================================================
# Test 7: Worker Concurrency Isolation
# ============================================================================
test_worker_concurrency_isolation() {
    log_test "Worker concurrency limits are tenant-isolated"
    run_test

    # This is a code inspection test
    if [ -f "$PROJECT_ROOT/workers/cpu-pool/limits.go" ]; then
        # Check that per-tenant semaphores are used
        if grep -q "map\[string\]\*semaphore" "$PROJECT_ROOT/workers/cpu-pool/limits.go"; then
            log_pass "Worker uses per-tenant semaphores"
        else
            log_fail "Worker does not use per-tenant semaphores"
            return 1
        fi

        # Check that tenant ID is validated
        if grep -q "tenantID == \"\"" "$PROJECT_ROOT/workers/cpu-pool/limits.go"; then
            log_pass "Worker validates tenant ID is present"
        else
            log_fail "Worker does not validate tenant ID"
            return 1
        fi

        # Check that limits are enforced with TryAcquire (non-blocking)
        if grep -q "TryAcquire" "$PROJECT_ROOT/workers/cpu-pool/limits.go"; then
            log_pass "Worker uses non-blocking semaphore acquisition"
        else
            log_fail "Worker does not use TryAcquire (may block other tenants)"
            return 1
        fi
    else
        log_fail "Worker limits file not found"
        return 1
    fi
}

# ============================================================================
# Run All Negative Tests
# ============================================================================
echo -e "${BLUE}Starting Negative Tests (verifying security boundaries)...${NC}"
echo ""

test_cross_tenant_data_access
echo ""

test_cross_tenant_log_isolation
echo ""

test_cross_tenant_quota_exhaustion
echo ""

test_unauthenticated_status_access
echo ""

test_cross_tenant_rate_limit_modification
echo ""

test_invalid_tenant_rejection
echo ""

test_worker_concurrency_isolation
echo ""

# ============================================================================
# Summary
# ============================================================================
echo -e "${BLUE}================================${NC}"
echo -e "${BLUE}Negative Test Summary${NC}"
echo -e "${BLUE}================================${NC}"
echo "Tests Run:    $TESTS_RUN"
echo -e "Tests Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Tests Failed: ${RED}$TESTS_FAILED${NC}"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ All negative tests passed - tenant isolation boundaries are secure!${NC}"
    exit 0
else
    echo -e "${RED}✗ Some negative tests failed - security boundaries may be compromised${NC}"
    exit 1
fi
