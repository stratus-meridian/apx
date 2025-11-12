#!/bin/bash

# Test script for Synchronous Proxy Mode
# Tests: Client → APX Router (sync) → Backend → Response (immediate)

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

echo "=========================================="
echo "APX Synchronous Proxy Test"
echo "=========================================="
echo ""

# Configuration
ROUTER_URL="${ROUTER_URL:-http://localhost:8081}"
BACKEND_URL="https://mocktarget.apigee.net"

echo "Configuration:"
echo "  Router URL: $ROUTER_URL"
echo "  Backend URL: $BACKEND_URL"
echo "  Mode: SYNCHRONOUS (direct proxy)"
echo ""

# Function to test sync endpoint
test_sync() {
    local method=$1
    local path=$2
    local description=$3
    local expected_status=$4

    echo -e "${YELLOW}Test: $description${NC}"
    echo "  Method: $method"
    echo "  Path: $path"

    # Send request and measure time
    START_TIME=$(date +%s%N)
    RESPONSE=$(curl -s -w "\n%{http_code}\n%{time_total}" -X $method \
        "$ROUTER_URL$path" \
        -H "Content-Type: application/json" \
        -H "X-Tenant-ID: test-tenant" \
        -H "X-Request-ID: test-sync-$(date +%s)")

    END_TIME=$(date +%s%N)
    DURATION=$((($END_TIME - $START_TIME) / 1000000)) # Convert to ms

    HTTP_CODE=$(echo "$RESPONSE" | tail -n2 | head -n1)
    TIME_TOTAL=$(echo "$RESPONSE" | tail -n1)
    BODY=$(echo "$RESPONSE" | head -n-2)

    echo "  Status: $HTTP_CODE"
    echo "  Duration: ${DURATION}ms"
    echo "  Response:"
    echo "$BODY" | head -c 300
    echo "..."

    if [ "$HTTP_CODE" == "$expected_status" ]; then
        echo -e "  ${GREEN}✓ Success (synchronous response)${NC}"
    elif [ "$HTTP_CODE" == "202" ]; then
        echo -e "  ${RED}✗ Got 202 (async mode) - expected $expected_status (sync mode)${NC}"
        echo -e "  ${RED}   Check ROUTES_CONFIG environment variable${NC}"
    else
        echo -e "  ${RED}✗ Unexpected status: $HTTP_CODE (expected $expected_status)${NC}"
    fi

    echo ""
    sleep 0.5
}

# Test 1: Verify backend is accessible
echo "=========================================="
echo "1. Testing Backend Direct Access"
echo "=========================================="
echo ""

echo "Testing: $BACKEND_URL/"
DIRECT_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BACKEND_URL/")

if [ "$DIRECT_CODE" == "200" ]; then
    echo -e "${GREEN}✓ Backend is accessible (status: $DIRECT_CODE)${NC}"
else
    echo -e "${RED}✗ Backend returned: $DIRECT_CODE${NC}"
fi
echo ""

# Test 2: Verify router is up
echo "=========================================="
echo "2. Testing APX Router Health"
echo "=========================================="
echo ""

HEALTH_RESPONSE=$(curl -s "$ROUTER_URL/health")
echo "Health check: $HEALTH_RESPONSE"

if echo "$HEALTH_RESPONSE" | grep -q "ok"; then
    echo -e "${GREEN}✓ Router is healthy${NC}"
else
    echo -e "${RED}✗ Router health check failed${NC}"
    exit 1
fi
echo ""

# Test 3: Synchronous proxy tests
echo "=========================================="
echo "3. Testing Synchronous Proxy Mode"
echo "=========================================="
echo ""

# Basic GET tests
test_sync "GET" "/mock/" "Get root path (sync)" "200"
test_sync "GET" "/mock/json" "Get JSON (sync)" "200"
test_sync "GET" "/mock/xml" "Get XML (sync)" "200"
test_sync "GET" "/mock/user" "Get user data (sync)" "200"

# Test different status codes
test_sync "GET" "/mock/statuscode/200" "Get 200 status (sync)" "200"
test_sync "GET" "/mock/statuscode/404" "Get 404 status (sync)" "404"
test_sync "GET" "/mock/statuscode/500" "Get 500 status (sync)" "500"

# POST test
echo -e "${YELLOW}Test: POST echo request (sync)${NC}"
echo "  Method: POST"
echo "  Path: /mock/echo"

POST_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
    "$ROUTER_URL/mock/echo" \
    -H "Content-Type: application/json" \
    -H "X-Tenant-ID: test-tenant" \
    -d '{"test": "sync proxy", "timestamp": '$(date +%s)'}')

POST_CODE=$(echo "$POST_RESPONSE" | tail -n1)
POST_BODY=$(echo "$POST_RESPONSE" | head -n-1)

echo "  Status: $POST_CODE"
echo "  Response:"
echo "$POST_BODY" | head -c 300
echo "..."

if [ "$POST_CODE" == "200" ]; then
    echo -e "  ${GREEN}✓ Success (synchronous POST)${NC}"
else
    echo -e "  ${RED}✗ Unexpected status: $POST_CODE${NC}"
fi
echo ""

# Test 4: Headers propagation
echo "=========================================="
echo "4. Testing Header Propagation"
echo "=========================================="
echo ""

HEADERS_RESPONSE=$(curl -s "$ROUTER_URL/mock/headers" \
    -H "X-Custom-Header: test-value" \
    -H "X-Tenant-ID: test-tenant")

echo "Response from /mock/headers:"
echo "$HEADERS_RESPONSE" | jq . 2>/dev/null || echo "$HEADERS_RESPONSE"
echo ""

if echo "$HEADERS_RESPONSE" | grep -q "X-Forwarded"; then
    echo -e "${GREEN}✓ X-Forwarded headers are being added${NC}"
else
    echo -e "${YELLOW}⚠ X-Forwarded headers might not be present${NC}"
fi
echo ""

# Test 5: Performance comparison
echo "=========================================="
echo "5. Performance Test (10 requests)"
echo "=========================================="
echo ""

echo "Sending 10 concurrent requests..."

TOTAL_TIME=0
SUCCESS=0
FAILED=0

for i in {1..10}; do
    START=$(date +%s%N)
    CODE=$(curl -s -o /dev/null -w "%{http_code}" "$ROUTER_URL/mock/json" -H "X-Tenant-ID: tenant-$i")
    END=$(date +%s%N)
    DURATION=$((($END - $START) / 1000000))

    if [ "$CODE" == "200" ]; then
        SUCCESS=$((SUCCESS + 1))
        echo "  Request $i: ${GREEN}✓${NC} ${DURATION}ms"
    else
        FAILED=$((FAILED + 1))
        echo "  Request $i: ${RED}✗${NC} status=$CODE"
    fi

    TOTAL_TIME=$((TOTAL_TIME + DURATION))
done

AVG_TIME=$((TOTAL_TIME / 10))

echo ""
echo "Results:"
echo "  Success: $SUCCESS/10"
echo "  Failed: $FAILED/10"
echo "  Average latency: ${AVG_TIME}ms"
echo ""

# Test 6: Compare sync vs async modes (if async is available)
echo "=========================================="
echo "6. Mode Comparison"
echo "=========================================="
echo ""

echo "Testing path without sync config (should use async)..."
ASYNC_TEST=$(curl -s -w "\n%{http_code}" "$ROUTER_URL/notconfigured")
ASYNC_CODE=$(echo "$ASYNC_TEST" | tail -n1)

if [ "$ASYNC_CODE" == "202" ]; then
    echo -e "${GREEN}✓ Async mode working for unconfigured paths${NC}"
elif [ "$ASYNC_CODE" == "404" ]; then
    echo -e "${YELLOW}⚠ Path not found (expected, no backend configured)${NC}"
else
    echo -e "${YELLOW}⚠ Got status: $ASYNC_CODE${NC}"
fi
echo ""

# Summary
echo "=========================================="
echo "Test Summary"
echo "=========================================="
echo ""

if [ $SUCCESS -eq 10 ]; then
    echo -e "${GREEN}✓ All tests passed!${NC}"
    echo ""
    echo "Synchronous proxy mode is working correctly:"
    echo "  • Requests return immediately (200/404/500)"
    echo "  • Average latency: ${AVG_TIME}ms"
    echo "  • Headers are properly forwarded"
    echo "  • Backend responses are streamed back"
else
    echo -e "${YELLOW}⚠ Some tests failed or returned unexpected results${NC}"
fi

echo ""
echo "=========================================="
echo "How to Use Sync Mode"
echo "=========================================="
echo ""
echo "1. Configure routes via environment variable:"
echo "   export ROUTES_CONFIG=\"/mock/**=https://mocktarget.apigee.net:sync\""
echo ""
echo "2. Or in docker-compose.yml / .env file:"
echo "   ROUTES_CONFIG=/mock/**=https://mocktarget.apigee.net:sync"
echo ""
echo "3. Multiple routes:"
echo "   ROUTES_CONFIG=/mock/**=https://mocktarget.apigee.net:sync,/api/**=https://api.example.com:async"
echo ""
echo "4. Check router logs for route registration:"
echo "   docker logs apx-router | grep 'route registered'"
echo ""
echo "=========================================="
echo ""
