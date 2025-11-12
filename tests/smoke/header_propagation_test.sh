#!/bin/bash

# V-002: Header Propagation Verification Test
# Tests that headers are correctly propagated through the request chain

set -e

ROUTER_URL="${ROUTER_URL:-http://localhost:8081}"
TEST_PATH="/api/test"
PASS_COUNT=0
FAIL_COUNT=0

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "============================================"
echo "V-002: Header Propagation Verification Test"
echo "============================================"
echo ""
echo "Router URL: $ROUTER_URL"
echo "Test Path: $TEST_PATH"
echo ""

# Test helper functions
pass() {
    echo -e "${GREEN}✓ PASS${NC}: $1"
    ((PASS_COUNT++))
}

fail() {
    echo -e "${RED}✗ FAIL${NC}: $1"
    ((FAIL_COUNT++))
}

info() {
    echo -e "${BLUE}ℹ INFO${NC}: $1"
}

# Check if router is healthy
echo "Step 1: Verifying router health..."
if curl -s -f "$ROUTER_URL/health" > /dev/null 2>&1; then
    pass "Router is healthy"
else
    fail "Router health check failed"
    echo "Cannot proceed with tests. Exiting."
    exit 1
fi
echo ""

# Test 1: Request ID - Provided header should be preserved
echo "Test 1: Request ID Preservation (with provided header)"
echo "-------------------------------------------------------"
CUSTOM_REQUEST_ID="test-request-$(date +%s)"
RESPONSE=$(curl -s -i -X GET "$ROUTER_URL$TEST_PATH" \
    -H "X-Request-ID: $CUSTOM_REQUEST_ID")

# Check if the same request ID is returned
if echo "$RESPONSE" | grep -q "X-Request-ID: $CUSTOM_REQUEST_ID"; then
    pass "Request ID preserved: $CUSTOM_REQUEST_ID"
else
    fail "Request ID not preserved"
    info "Response headers:"
    echo "$RESPONSE" | grep -i "x-request-id" || echo "No X-Request-ID header found"
fi
echo ""

# Test 2: Request ID - Generated if not provided
echo "Test 2: Request ID Generation (without provided header)"
echo "--------------------------------------------------------"
RESPONSE=$(curl -s -i -X GET "$ROUTER_URL$TEST_PATH")

# Check if a request ID is generated
GENERATED_ID=$(echo "$RESPONSE" | grep -i "x-request-id:" | awk '{print $2}' | tr -d '\r\n')
if [ -n "$GENERATED_ID" ]; then
    pass "Request ID generated: $GENERATED_ID"
else
    fail "Request ID not generated when missing"
fi
echo ""

# Test 3: Tenant ID - Extraction and propagation
echo "Test 3: Tenant ID Extraction and Propagation"
echo "---------------------------------------------"
TENANT_ID="tenant-12345"
RESPONSE=$(curl -s -i -X GET "$ROUTER_URL$TEST_PATH" \
    -H "X-Tenant-ID: $TENANT_ID")

# Check if tenant ID is returned
if echo "$RESPONSE" | grep -q "X-Tenant-ID: $TENANT_ID"; then
    pass "Tenant ID propagated: $TENANT_ID"
else
    fail "Tenant ID not propagated"
    info "Response headers:"
    echo "$RESPONSE" | grep -i "x-tenant-id" || echo "No X-Tenant-ID header found"
fi
echo ""

# Test 4: Tenant ID - Default when not provided
echo "Test 4: Tenant ID Default (without provided header)"
echo "----------------------------------------------------"
RESPONSE=$(curl -s -i -X GET "$ROUTER_URL$TEST_PATH")

# Check if default tenant ID is set
if echo "$RESPONSE" | grep -q "X-Tenant-ID: unknown"; then
    pass "Default tenant ID set: unknown"
else
    fail "Default tenant ID not set"
    info "Response headers:"
    echo "$RESPONSE" | grep -i "x-tenant-id" || echo "No X-Tenant-ID header found"
fi
echo ""

# Test 5: Policy Version - Tagged on every request
echo "Test 5: Policy Version Tagged on Every Request"
echo "-----------------------------------------------"
RESPONSE=$(curl -s -i -X GET "$ROUTER_URL$TEST_PATH")

# Check if policy version header exists
POLICY_VERSION=$(echo "$RESPONSE" | grep -i "x-policy-version:" | awk '{print $2}' | tr -d '\r\n')
if [ -n "$POLICY_VERSION" ]; then
    pass "Policy version tagged: $POLICY_VERSION"
else
    fail "Policy version not tagged"
fi
echo ""

# Test 6: Region - Tagged based on deployment
echo "Test 6: Region Tagged Based on Deployment"
echo "------------------------------------------"
RESPONSE=$(curl -s -i -X GET "$ROUTER_URL$TEST_PATH")

# Check if region header exists
REGION=$(echo "$RESPONSE" | grep -i "x-region:" | awk '{print $2}' | tr -d '\r\n')
if [ -n "$REGION" ]; then
    pass "Region tagged: $REGION"
else
    fail "Region not tagged"
fi
echo ""

# Test 7: Tenant Tier - Propagation
echo "Test 7: Tenant Tier Propagation"
echo "--------------------------------"
TENANT_TIER="premium"
RESPONSE=$(curl -s -i -X GET "$ROUTER_URL$TEST_PATH" \
    -H "X-Tenant-ID: tenant-premium" \
    -H "X-Tenant-Tier: $TENANT_TIER")

# Check if tenant tier is returned
if echo "$RESPONSE" | grep -q "X-Tenant-Tier: $TENANT_TIER"; then
    pass "Tenant tier propagated: $TENANT_TIER"
else
    fail "Tenant tier not propagated"
    info "Response headers:"
    echo "$RESPONSE" | grep -i "x-tenant-tier" || echo "No X-Tenant-Tier header found"
fi
echo ""

# Test 8: All headers together
echo "Test 8: All Headers Propagated Together"
echo "----------------------------------------"
TEST_REQUEST_ID="all-headers-test-$(date +%s)"
TEST_TENANT_ID="tenant-all-test"
TEST_TENANT_TIER="enterprise"

RESPONSE=$(curl -s -i -X GET "$ROUTER_URL$TEST_PATH" \
    -H "X-Request-ID: $TEST_REQUEST_ID" \
    -H "X-Tenant-ID: $TEST_TENANT_ID" \
    -H "X-Tenant-Tier: $TEST_TENANT_TIER")

ALL_PRESENT=true
if ! echo "$RESPONSE" | grep -q "X-Request-ID: $TEST_REQUEST_ID"; then
    ALL_PRESENT=false
    info "Missing X-Request-ID"
fi
if ! echo "$RESPONSE" | grep -q "X-Tenant-ID: $TEST_TENANT_ID"; then
    ALL_PRESENT=false
    info "Missing X-Tenant-ID"
fi
if ! echo "$RESPONSE" | grep -q "X-Tenant-Tier: $TEST_TENANT_TIER"; then
    ALL_PRESENT=false
    info "Missing X-Tenant-Tier"
fi
if ! echo "$RESPONSE" | grep -qi "x-policy-version:"; then
    ALL_PRESENT=false
    info "Missing X-Policy-Version"
fi
if ! echo "$RESPONSE" | grep -qi "x-region:"; then
    ALL_PRESENT=false
    info "Missing X-Region"
fi

if [ "$ALL_PRESENT" = true ]; then
    pass "All headers present in response"
else
    fail "Not all headers present in response"
fi
echo ""

# Test 9: Verify headers appear in router logs
echo "Test 9: Headers Appear in Router Logs"
echo "--------------------------------------"
LOG_TEST_ID="log-test-$(date +%s)"

# Send request with unique ID to track in logs
curl -s -X GET "$ROUTER_URL$TEST_PATH" \
    -H "X-Request-ID: $LOG_TEST_ID" \
    -H "X-Tenant-ID: log-test-tenant" > /dev/null 2>&1

# Wait a moment for logs to be written
sleep 1

# Check docker logs for the request ID
ROUTER_CONTAINER=$(docker ps --format "{{.Names}}" | grep router | head -1)
if [ -z "$ROUTER_CONTAINER" ]; then
    fail "Router container not found"
    info "Skipping log verification tests"
elif docker logs "$ROUTER_CONTAINER" 2>&1 | grep -q "$LOG_TEST_ID"; then
    pass "Request ID found in router logs"

    # Check for other headers in logs
    LOG_LINE=$(docker logs "$ROUTER_CONTAINER" 2>&1 | grep "$LOG_TEST_ID" | tail -1)
    if echo "$LOG_LINE" | grep -q "tenant_id"; then
        pass "Tenant ID found in router logs"
    else
        fail "Tenant ID not found in router logs"
    fi

    if echo "$LOG_LINE" | grep -q "policy_version"; then
        pass "Policy version found in router logs"
    else
        fail "Policy version not found in router logs"
    fi

    if echo "$LOG_LINE" | grep -q "region"; then
        pass "Region found in router logs"
    else
        fail "Region not found in router logs"
    fi
else
    fail "Request ID not found in router logs"
    info "Router container: $ROUTER_CONTAINER"
fi
echo ""

# Summary
echo "============================================"
echo "Test Summary"
echo "============================================"
echo -e "${GREEN}Passed: $PASS_COUNT${NC}"
echo -e "${RED}Failed: $FAIL_COUNT${NC}"
echo "Total: $((PASS_COUNT + FAIL_COUNT))"
echo ""

if [ $FAIL_COUNT -eq 0 ]; then
    echo -e "${GREEN}✓ All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}✗ Some tests failed${NC}"
    exit 1
fi
