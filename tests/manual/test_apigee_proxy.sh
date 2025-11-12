#!/bin/bash

# Test script for Apigee Mock Target Proxy via APX
# Tests: https://mocktarget.apigee.net/ → APX Router → Pub/Sub → Workers → Response

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo "======================================"
echo "APX Apigee Mock Target Proxy Test"
echo "======================================"
echo ""

# Configuration
ROUTER_URL="${ROUTER_URL:-http://localhost:8081}"
BACKEND_URL="https://mocktarget.apigee.net"

echo "Configuration:"
echo "  Router URL: $ROUTER_URL"
echo "  Backend URL: $BACKEND_URL"
echo ""

# Function to test endpoint
test_endpoint() {
    local method=$1
    local path=$2
    local description=$3

    echo -e "${YELLOW}Test: $description${NC}"
    echo "  Method: $method"
    echo "  Path: $path"

    # Send request to APX router
    RESPONSE=$(curl -s -w "\n%{http_code}" -X $method \
        "$ROUTER_URL$path" \
        -H "Content-Type: application/json" \
        -H "X-Tenant-ID: test-tenant" \
        -H "X-Request-ID: test-$(date +%s)")

    HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
    BODY=$(echo "$RESPONSE" | head -n-1)

    echo "  Status: $HTTP_CODE"
    echo "  Response: $BODY" | head -c 200
    echo "..."

    if [ "$HTTP_CODE" == "202" ]; then
        echo -e "  ${GREEN}✓ Request accepted (async processing)${NC}"

        # Extract request_id from response
        REQUEST_ID=$(echo "$BODY" | grep -o '"request_id":"[^"]*"' | cut -d'"' -f4)

        if [ -n "$REQUEST_ID" ]; then
            echo "  Request ID: $REQUEST_ID"

            # Poll status endpoint
            echo "  Polling status..."
            for i in {1..10}; do
                sleep 1
                STATUS_RESPONSE=$(curl -s "$ROUTER_URL/status/$REQUEST_ID")
                STATUS=$(echo "$STATUS_RESPONSE" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)

                echo "    [$i/10] Status: $STATUS"

                if [ "$STATUS" == "completed" ] || [ "$STATUS" == "failed" ]; then
                    echo "$STATUS_RESPONSE" | jq . 2>/dev/null || echo "$STATUS_RESPONSE"
                    break
                fi
            done
        fi
    elif [ "$HTTP_CODE" == "200" ]; then
        echo -e "  ${GREEN}✓ Success (synchronous response)${NC}"
    else
        echo -e "  ${RED}✗ Unexpected status code${NC}"
    fi

    echo ""
}

# Test 1: Direct backend test (verify mocktarget works)
echo "======================================"
echo "1. Testing Apigee Mock Target directly"
echo "======================================"
echo ""

DIRECT_RESPONSE=$(curl -s -w "\n%{http_code}" "$BACKEND_URL/")
DIRECT_CODE=$(echo "$DIRECT_RESPONSE" | tail -n1)
DIRECT_BODY=$(echo "$DIRECT_RESPONSE" | head -n-1)

echo "Direct Request to: $BACKEND_URL/"
echo "Status: $DIRECT_CODE"
echo "Response:"
echo "$DIRECT_BODY" | head -c 500
echo "..."
echo ""

if [ "$DIRECT_CODE" == "200" ]; then
    echo -e "${GREEN}✓ Backend is accessible${NC}"
else
    echo -e "${RED}✗ Backend returned $DIRECT_CODE${NC}"
fi

echo ""
echo "======================================"
echo "2. Testing via APX Router"
echo "======================================"
echo ""

# Test 2: Root path
test_endpoint "GET" "/mock/" "Get root path"

# Test 3: JSON endpoint
test_endpoint "GET" "/mock/json" "Get JSON response"

# Test 4: XML endpoint
test_endpoint "GET" "/mock/xml" "Get XML response"

# Test 5: User endpoint
test_endpoint "GET" "/mock/user" "Get user data"

# Test 6: POST request
test_endpoint "POST" "/mock/echo" "POST echo request"

# Test 7: Status codes
test_endpoint "GET" "/mock/statuscode/404" "Get 404 status"
test_endpoint "GET" "/mock/statuscode/500" "Get 500 status"

echo "======================================"
echo "Test Summary"
echo "======================================"
echo ""
echo "All tests completed!"
echo ""
echo "To test other endpoints from mocktarget.apigee.net:"
echo "  curl $ROUTER_URL/mock/help"
echo "  curl $ROUTER_URL/mock/json"
echo "  curl $ROUTER_URL/mock/xml"
echo "  curl $ROUTER_URL/mock/user"
echo ""
echo "Check APX Router logs:"
echo "  docker logs apx-router  # If using Docker"
echo "  kubectl logs -l app=apx-router  # If using K8s"
echo ""
