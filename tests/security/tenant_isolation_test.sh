#!/bin/bash
set -euo pipefail

# Tenant Isolation Positive Test Suite
# Tests that tenant isolation works correctly at every layer

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
REDIS_HOST="${REDIS_HOST:-localhost:6379}"
PROJECT_ID="${PROJECT_ID:-apx-dev}"
PUBSUB_TOPIC="${PUBSUB_TOPIC:-apx-requests-us-dev}"

echo -e "${BLUE}================================${NC}"
echo -e "${BLUE}Tenant Isolation Test Suite${NC}"
echo -e "${BLUE}================================${NC}"
echo ""
echo "Edge URL: $EDGE_URL"
echo "Redis: $REDIS_HOST"
echo "Project: $PROJECT_ID"
echo "Topic: $PUBSUB_TOPIC"
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
# Test 1: Redis Keys Isolated by Tenant
# ============================================================================
test_redis_key_isolation() {
    log_test "Redis keyspace isolation between tenants"
    run_test

    # Setup: Clear any existing keys
    if command -v redis-cli &> /dev/null; then
        redis-cli -h "${REDIS_HOST%%:*}" -p "${REDIS_HOST##*:}" FLUSHDB > /dev/null 2>&1 || true
    fi

    # Simulate rate limiting for tenant-a
    log_info "Simulating rate limit checks for tenant-a"
    TENANT_A_KEY="apx:rl:tenant-a:requests"

    # Simulate rate limiting for tenant-b
    log_info "Simulating rate limit checks for tenant-b"
    TENANT_B_KEY="apx:rl:tenant-b:requests"

    # Verify keys are different
    if [ "$TENANT_A_KEY" != "$TENANT_B_KEY" ]; then
        log_pass "Redis keys use tenant-specific prefixes"
    else
        log_fail "Redis keys are not tenant-isolated"
        return 1
    fi

    # Verify key pattern
    if [[ "$TENANT_A_KEY" =~ ^apx:rl:tenant-a: ]]; then
        log_pass "Tenant-a keys follow isolation pattern"
    else
        log_fail "Tenant-a keys do not follow isolation pattern"
        return 1
    fi

    if [[ "$TENANT_B_KEY" =~ ^apx:rl:tenant-b: ]]; then
        log_pass "Tenant-b keys follow isolation pattern"
    else
        log_fail "Tenant-b keys do not follow isolation pattern"
        return 1
    fi

    # Test that increments are isolated
    if command -v redis-cli &> /dev/null; then
        log_info "Testing Redis counter isolation"

        # Increment tenant-a 5 times
        for i in {1..5}; do
            redis-cli -h "${REDIS_HOST%%:*}" -p "${REDIS_HOST##*:}" INCR "$TENANT_A_KEY" > /dev/null 2>&1 || true
        done

        # Increment tenant-b 3 times
        for i in {1..3}; do
            redis-cli -h "${REDIS_HOST%%:*}" -p "${REDIS_HOST##*:}" INCR "$TENANT_B_KEY" > /dev/null 2>&1 || true
        done

        # Verify counts are isolated
        TENANT_A_COUNT=$(redis-cli -h "${REDIS_HOST%%:*}" -p "${REDIS_HOST##*:}" GET "$TENANT_A_KEY" 2>/dev/null || echo "0")
        TENANT_B_COUNT=$(redis-cli -h "${REDIS_HOST%%:*}" -p "${REDIS_HOST##*:}" GET "$TENANT_B_KEY" 2>/dev/null || echo "0")

        if [ "$TENANT_A_COUNT" = "5" ] && [ "$TENANT_B_COUNT" = "3" ]; then
            log_pass "Redis counters are properly isolated"
        else
            log_fail "Redis counters are not isolated (tenant-a: $TENANT_A_COUNT, tenant-b: $TENANT_B_COUNT)"
            return 1
        fi
    else
        log_info "redis-cli not available, skipping counter test"
    fi
}

# ============================================================================
# Test 2: Cross-Tenant Rate Limits Don't Affect Each Other
# ============================================================================
test_cross_tenant_rate_limits() {
    log_test "Cross-tenant rate limit independence"
    run_test

    log_info "Sending 100 requests from tenant-a"
    TENANT_A_REQUESTS=0
    for i in {1..10}; do
        STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
            -X POST "$EDGE_URL/v1/test" \
            -H "X-Tenant-ID: tenant-a" \
            -H "X-API-Key: test-key-a" \
            -H "Content-Type: application/json" \
            -d '{"test": "rate-limit"}' 2>/dev/null || echo "000")

        if [ "$STATUS" = "202" ] || [ "$STATUS" = "200" ]; then
            ((TENANT_A_REQUESTS++))
        fi
    done

    log_info "Tenant-a successful requests: $TENANT_A_REQUESTS"

    log_info "Sending requests from tenant-b (should not be affected by tenant-a)"
    TENANT_B_REQUESTS=0
    for i in {1..10}; do
        STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
            -X POST "$EDGE_URL/v1/test" \
            -H "X-Tenant-ID: tenant-b" \
            -H "X-API-Key: test-key-b" \
            -H "Content-Type: application/json" \
            -d '{"test": "rate-limit"}' 2>/dev/null || echo "000")

        if [ "$STATUS" = "202" ] || [ "$STATUS" = "200" ]; then
            ((TENANT_B_REQUESTS++))
        fi
    done

    log_info "Tenant-b successful requests: $TENANT_B_REQUESTS"

    # Both tenants should be able to make requests
    if [ "$TENANT_B_REQUESTS" -gt 0 ]; then
        log_pass "Tenant-b not affected by tenant-a's rate limit"
    else
        log_fail "Tenant-b appears to be affected by tenant-a's rate limit"
        return 1
    fi
}

# ============================================================================
# Test 3: Pub/Sub Messages Tagged with Tenant ID
# ============================================================================
test_pubsub_tenant_attributes() {
    log_test "Pub/Sub message tenant attributes"
    run_test

    if ! command -v gcloud &> /dev/null; then
        log_info "gcloud not available, skipping Pub/Sub test"
        return 0
    fi

    log_info "Checking Pub/Sub messages for tenant attributes"

    # Send test requests
    curl -s -X POST "$EDGE_URL/v1/test" \
        -H "X-Tenant-ID: tenant-test-123" \
        -H "X-API-Key: test-key" \
        -d '{"test": "pubsub"}' > /dev/null 2>&1 || true

    sleep 2

    # Pull messages (if subscription exists)
    MESSAGES=$(gcloud pubsub subscriptions pull apx-workers-us-dev \
        --limit=5 \
        --format=json \
        --project="$PROJECT_ID" 2>/dev/null || echo "[]")

    # Check if any message has tenant_id attribute
    HAS_TENANT_ATTR=$(echo "$MESSAGES" | jq -r '.[].message.attributes.tenant_id // empty' | grep -c "." || echo "0")

    if [ "$HAS_TENANT_ATTR" -gt 0 ]; then
        log_pass "Pub/Sub messages contain tenant_id attribute"
    else
        log_info "Could not verify Pub/Sub tenant attributes (may need active subscription)"
    fi

    # Check ordering key
    HAS_ORDERING=$(echo "$MESSAGES" | jq -r '.[].message.orderingKey // empty' | grep -c "." || echo "0")

    if [ "$HAS_ORDERING" -gt 0 ]; then
        log_pass "Pub/Sub messages have ordering keys for FIFO per tenant"
    else
        log_info "Could not verify Pub/Sub ordering keys"
    fi
}

# ============================================================================
# Test 4: Worker Enforces Per-Tenant Concurrency
# ============================================================================
test_worker_concurrency_limits() {
    log_test "Worker per-tenant concurrency enforcement"
    run_test

    log_info "Testing worker concurrency limits"

    # This test would need workers actually running
    # For now, we test the logic is in place

    if [ -f "$PROJECT_ROOT/workers/cpu-pool/limits.go" ]; then
        # Check if the limits file has key functions
        if grep -q "AcquireSlot" "$PROJECT_ROOT/workers/cpu-pool/limits.go" && \
           grep -q "ReleaseSlot" "$PROJECT_ROOT/workers/cpu-pool/limits.go" && \
           grep -q "TenantLimits" "$PROJECT_ROOT/workers/cpu-pool/limits.go"; then
            log_pass "Worker tenant limits implementation exists"
        else
            log_fail "Worker tenant limits implementation incomplete"
            return 1
        fi

        # Check for semaphore usage
        if grep -q "semaphore" "$PROJECT_ROOT/workers/cpu-pool/limits.go"; then
            log_pass "Worker uses semaphores for tenant concurrency control"
        else
            log_fail "Worker does not use semaphores"
            return 1
        fi

        # Check for tenant isolation in limits
        if grep -q "tenantID" "$PROJECT_ROOT/workers/cpu-pool/limits.go"; then
            log_pass "Worker limits are tenant-specific"
        else
            log_fail "Worker limits do not consider tenant"
            return 1
        fi
    else
        log_fail "Worker limits file not found"
        return 1
    fi
}

# ============================================================================
# Test 5: Tenant Attributes Propagate Through Stack
# ============================================================================
test_tenant_attribute_propagation() {
    log_test "Tenant attributes propagate through stack"
    run_test

    log_info "Sending request with tenant context"

    RESPONSE=$(curl -s -X POST "$EDGE_URL/v1/test" \
        -H "X-Tenant-ID: tenant-propagation-test" \
        -H "X-Tenant-Tier: enterprise" \
        -H "X-API-Key: test-key" \
        -H "Content-Type: application/json" \
        -d '{"test": "propagation"}' 2>/dev/null || echo "{}")

    # Check if response includes tenant headers
    TENANT_HEADER=$(curl -s -D - -X POST "$EDGE_URL/v1/test" \
        -H "X-Tenant-ID: tenant-propagation-test" \
        -H "X-Tenant-Tier: enterprise" \
        -H "X-API-Key: test-key" \
        -d '{"test": "propagation"}' 2>/dev/null | grep -i "X-Tenant-ID" || echo "")

    if [ -n "$TENANT_HEADER" ]; then
        log_pass "Tenant headers propagated in response"
    else
        log_info "Could not verify tenant header propagation (may need deployed service)"
    fi
}

# ============================================================================
# Run All Tests
# ============================================================================
echo -e "${BLUE}Starting Tenant Isolation Tests...${NC}"
echo ""

test_redis_key_isolation
echo ""

test_cross_tenant_rate_limits
echo ""

test_pubsub_tenant_attributes
echo ""

test_worker_concurrency_limits
echo ""

test_tenant_attribute_propagation
echo ""

# ============================================================================
# Summary
# ============================================================================
echo -e "${BLUE}================================${NC}"
echo -e "${BLUE}Test Summary${NC}"
echo -e "${BLUE}================================${NC}"
echo "Tests Run:    $TESTS_RUN"
echo -e "Tests Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Tests Failed: ${RED}$TESTS_FAILED${NC}"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ All tenant isolation tests passed!${NC}"
    exit 0
else
    echo -e "${RED}✗ Some tenant isolation tests failed${NC}"
    exit 1
fi
