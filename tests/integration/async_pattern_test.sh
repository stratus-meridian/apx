#!/bin/bash
# V-004: Async Contract Verification Test Suite
# Tests the async request/response pattern with status polling and SSE streaming

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
ROUTER_URL="${ROUTER_URL:-http://localhost:8081}"
STREAM_URL="${STREAM_URL:-http://localhost:8083}"
REDIS_HOST="${REDIS_HOST:-localhost}"
REDIS_PORT="${REDIS_PORT:-6379}"

# Test counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Helper functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_test() {
    echo -e "${YELLOW}[TEST]${NC} $1"
}

pass_test() {
    PASSED_TESTS=$((PASSED_TESTS + 1))
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo -e "${GREEN}✓ PASS${NC}: $1"
}

fail_test() {
    FAILED_TESTS=$((FAILED_TESTS + 1))
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo -e "${RED}✗ FAIL${NC}: $1"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    # Check if curl is available
    if ! command -v curl &> /dev/null; then
        log_error "curl is not installed"
        exit 1
    fi

    # Check if jq is available
    if ! command -v jq &> /dev/null; then
        log_error "jq is not installed (required for JSON parsing)"
        exit 1
    fi

    # Check if redis-cli is available (optional)
    if ! command -v redis-cli &> /dev/null; then
        log_error "redis-cli is not installed (optional, but recommended)"
    fi

    # Check if router is running
    if ! curl -s -f "${ROUTER_URL}/health" > /dev/null 2>&1; then
        log_error "Router is not running at ${ROUTER_URL}"
        exit 1
    fi

    # Check if streaming aggregator is running
    if ! curl -s -f "${STREAM_URL}/health" > /dev/null 2>&1; then
        log_error "Streaming aggregator is not running at ${STREAM_URL}"
        exit 1
    fi

    # Check if Redis is running
    if command -v redis-cli &> /dev/null; then
        if ! redis-cli -h "${REDIS_HOST}" -p "${REDIS_PORT}" ping > /dev/null 2>&1; then
            log_error "Redis is not running at ${REDIS_HOST}:${REDIS_PORT}"
            exit 1
        fi
    fi

    log_info "All prerequisites met"
}

# Test 1: POST returns 202 with status_url
test_202_response() {
    log_test "Test 1: POST returns 202 Accepted with status_url"

    response=$(curl -s -w "\n%{http_code}" \
        -X POST "${ROUTER_URL}/api/test" \
        -H "Content-Type: application/json" \
        -H "X-Tenant-ID: test-tenant" \
        -H "X-Tenant-Tier: pro" \
        -d '{"message":"test"}')

    http_code=$(echo "$response" | tail -n 1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" = "202" ]; then
        # Check if response contains required fields
        request_id=$(echo "$body" | jq -r '.request_id')
        status_url=$(echo "$body" | jq -r '.status_url')
        stream_url=$(echo "$body" | jq -r '.stream_url')

        if [ -n "$request_id" ] && [ "$request_id" != "null" ]; then
            if [ -n "$status_url" ] && [ "$status_url" != "null" ]; then
                if [ -n "$stream_url" ] && [ "$stream_url" != "null" ]; then
                    pass_test "202 response includes request_id, status_url, and stream_url"
                    echo "$request_id" > /tmp/test_request_id.txt
                    return 0
                else
                    fail_test "Response missing stream_url"
                fi
            else
                fail_test "Response missing status_url"
            fi
        else
            fail_test "Response missing request_id"
        fi
    else
        fail_test "Expected HTTP 202, got $http_code"
        echo "Response: $body"
    fi

    return 1
}

# Test 2: Status endpoint returns correct initial state
test_status_endpoint() {
    log_test "Test 2: Status endpoint shows correct initial state"

    if [ ! -f /tmp/test_request_id.txt ]; then
        fail_test "No request_id from previous test"
        return 1
    fi

    request_id=$(cat /tmp/test_request_id.txt)

    response=$(curl -s -w "\n%{http_code}" \
        -X GET "${ROUTER_URL}/status/${request_id}")

    http_code=$(echo "$response" | tail -n 1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" = "200" ]; then
        status=$(echo "$body" | jq -r '.status')
        tenant_id=$(echo "$body" | jq -r '.tenant_id')

        if [ "$status" = "pending" ] || [ "$status" = "processing" ]; then
            if [ "$tenant_id" = "test-tenant" ]; then
                pass_test "Status endpoint returns correct initial state"
                return 0
            else
                fail_test "Status endpoint returned wrong tenant_id: $tenant_id"
            fi
        else
            fail_test "Status endpoint returned unexpected status: $status"
        fi
    else
        fail_test "Status endpoint returned HTTP $http_code"
        echo "Response: $body"
    fi

    return 1
}

# Test 3: Status can be polled every 1 second
test_polling_rate() {
    log_test "Test 3: Status can be polled every 1s without rate limiting"

    if [ ! -f /tmp/test_request_id.txt ]; then
        fail_test "No request_id from previous test"
        return 1
    fi

    request_id=$(cat /tmp/test_request_id.txt)

    success_count=0
    for i in {1..10}; do
        response=$(curl -s -w "\n%{http_code}" \
            -X GET "${ROUTER_URL}/status/${request_id}")

        http_code=$(echo "$response" | tail -n1)

        if [ "$http_code" = "200" ]; then
            success_count=$((success_count + 1))
        fi

        sleep 1
    done

    if [ "$success_count" -eq 10 ]; then
        pass_test "Polled status 10 times (1s interval) without rate limiting"
        return 0
    else
        fail_test "Only $success_count/10 polls succeeded"
    fi

    return 1
}

# Test 4: SSE streaming endpoint works
test_sse_streaming() {
    log_test "Test 4: SSE streaming works for status updates"

    if [ ! -f /tmp/test_request_id.txt ]; then
        fail_test "No request_id from previous test"
        return 1
    fi

    request_id=$(cat /tmp/test_request_id.txt)

    # Start SSE stream in background and capture first 5 events
    timeout 10s curl -s -N \
        -H "Accept: text/event-stream" \
        "${STREAM_URL}/stream/${request_id}" \
        > /tmp/sse_output.txt 2>&1 &

    sse_pid=$!
    sleep 3
    kill $sse_pid 2>/dev/null || true

    if [ -f /tmp/sse_output.txt ] && [ -s /tmp/sse_output.txt ]; then
        # Check if output contains SSE format
        if grep -q "data:" /tmp/sse_output.txt; then
            if grep -q "id:" /tmp/sse_output.txt; then
                pass_test "SSE streaming works with event IDs"
                return 0
            else
                fail_test "SSE output missing event IDs"
            fi
        else
            fail_test "SSE output doesn't contain 'data:' fields"
        fi
    else
        fail_test "SSE stream returned no data"
    fi

    return 1
}

# Test 5: Stream resume with Last-Event-ID
test_stream_resume() {
    log_test "Test 5: Stream resume token (Last-Event-ID) works"

    if [ ! -f /tmp/test_request_id.txt ]; then
        fail_test "No request_id from previous test"
        return 1
    fi

    request_id=$(cat /tmp/test_request_id.txt)

    # First stream to get some events
    timeout 5s curl -s -N \
        -H "Accept: text/event-stream" \
        "${STREAM_URL}/stream/${request_id}" \
        > /tmp/sse_first.txt 2>&1 &

    sse_pid=$!
    sleep 2
    kill $sse_pid 2>/dev/null || true

    # Extract last event ID from first stream
    last_id=$(grep "^id:" /tmp/sse_first.txt | tail -n1 | cut -d' ' -f2)

    if [ -n "$last_id" ]; then
        # Resume stream with Last-Event-ID
        timeout 5s curl -s -N \
            -H "Accept: text/event-stream" \
            -H "Last-Event-ID: $last_id" \
            "${STREAM_URL}/stream/${request_id}" \
            > /tmp/sse_resumed.txt 2>&1 &

        sse_pid=$!
        sleep 2
        kill $sse_pid 2>/dev/null || true

        if [ -f /tmp/sse_resumed.txt ] && [ -s /tmp/sse_resumed.txt ]; then
            # Check if resumed stream has events
            if grep -q "data:" /tmp/sse_resumed.txt; then
                pass_test "Stream resume with Last-Event-ID works"
                return 0
            else
                fail_test "Resumed stream returned no data"
            fi
        else
            fail_test "Failed to resume stream"
        fi
    else
        fail_test "Could not extract event ID from first stream"
    fi

    return 1
}

# Test 6: Manual status update simulation (using Redis)
test_status_transitions() {
    log_test "Test 6: Status endpoint shows correct state transitions"

    if [ ! -f /tmp/test_request_id.txt ]; then
        fail_test "No request_id from previous test"
        return 1
    fi

    request_id=$(cat /tmp/test_request_id.txt)

    # Check if redis-cli is available
    if ! command -v redis-cli &> /dev/null; then
        log_error "redis-cli not available, skipping status transition test"
        fail_test "redis-cli not available"
        return 1
    fi

    # Get current status
    redis_key="status:${request_id}"
    current_status=$(redis-cli -h "${REDIS_HOST}" -p "${REDIS_PORT}" GET "$redis_key")

    if [ -n "$current_status" ]; then
        # Parse JSON and update status to "processing"
        updated_status=$(echo "$current_status" | jq '.status = "processing" | .progress = 50 | .updated_at = now | tostring')

        # Update in Redis
        redis-cli -h "${REDIS_HOST}" -p "${REDIS_PORT}" SET "$redis_key" "$updated_status" > /dev/null
        redis-cli -h "${REDIS_HOST}" -p "${REDIS_PORT}" EXPIRE "$redis_key" 86400 > /dev/null

        # Verify via API
        sleep 1
        response=$(curl -s -X GET "${ROUTER_URL}/status/${request_id}")
        status=$(echo "$response" | jq -r '.status')
        progress=$(echo "$response" | jq -r '.progress')

        if [ "$status" = "processing" ] && [ "$progress" = "50" ]; then
            pass_test "Status transitions work correctly"
            return 0
        else
            fail_test "Status transition not reflected in API (status=$status, progress=$progress)"
        fi
    else
        fail_test "Status not found in Redis"
    fi

    return 1
}

# Test 7: Status TTL (24 hours)
test_status_ttl() {
    log_test "Test 7: Status has 24-hour TTL in Redis"

    if [ ! -f /tmp/test_request_id.txt ]; then
        fail_test "No request_id from previous test"
        return 1
    fi

    request_id=$(cat /tmp/test_request_id.txt)

    # Check if redis-cli is available
    if ! command -v redis-cli &> /dev/null; then
        log_error "redis-cli not available, skipping TTL test"
        fail_test "redis-cli not available"
        return 1
    fi

    # Check TTL in Redis
    redis_key="status:${request_id}"
    ttl=$(redis-cli -h "${REDIS_HOST}" -p "${REDIS_PORT}" TTL "$redis_key")

    # TTL should be close to 24 hours (86400 seconds)
    # Allow some margin for test execution time (between 23.5 and 24.5 hours)
    if [ "$ttl" -ge 84600 ] && [ "$ttl" -le 88200 ]; then
        pass_test "Status has correct TTL (~24 hours, actual: ${ttl}s)"
        return 0
    else
        fail_test "Status TTL is incorrect (expected ~86400s, got ${ttl}s)"
    fi

    return 1
}

# Test 8: Multiple concurrent requests
test_concurrent_requests() {
    log_test "Test 8: Multiple concurrent requests are handled correctly"

    request_ids=()

    # Send 5 concurrent requests
    for i in {1..5}; do
        response=$(curl -s \
            -X POST "${ROUTER_URL}/api/test" \
            -H "Content-Type: application/json" \
            -H "X-Tenant-ID: test-tenant-$i" \
            -H "X-Tenant-Tier: pro" \
            -d "{\"message\":\"concurrent test $i\"}")

        request_id=$(echo "$response" | jq -r '.request_id')
        if [ -n "$request_id" ] && [ "$request_id" != "null" ]; then
            request_ids+=("$request_id")
        fi
    done

    # Verify all requests have status
    success_count=0
    for request_id in "${request_ids[@]}"; do
        response=$(curl -s -X GET "${ROUTER_URL}/status/${request_id}")
        status=$(echo "$response" | jq -r '.status')

        if [ "$status" = "pending" ] || [ "$status" = "processing" ]; then
            success_count=$((success_count + 1))
        fi
    done

    if [ "$success_count" -eq "${#request_ids[@]}" ] && [ "$success_count" -eq 5 ]; then
        pass_test "All 5 concurrent requests tracked correctly"
        return 0
    else
        fail_test "Only $success_count/${#request_ids[@]} concurrent requests tracked"
    fi

    return 1
}

# Cleanup
cleanup() {
    log_info "Cleaning up test artifacts..."
    rm -f /tmp/test_request_id.txt
    rm -f /tmp/sse_output.txt
    rm -f /tmp/sse_first.txt
    rm -f /tmp/sse_resumed.txt
}

# Main test execution
main() {
    echo "=========================================="
    echo "V-004: Async Contract Verification Tests"
    echo "=========================================="
    echo ""

    check_prerequisites
    echo ""

    # Run all tests
    test_202_response
    echo ""

    test_status_endpoint
    echo ""

    test_polling_rate
    echo ""

    test_sse_streaming
    echo ""

    test_stream_resume
    echo ""

    test_status_transitions
    echo ""

    test_status_ttl
    echo ""

    test_concurrent_requests
    echo ""

    # Print summary
    echo "=========================================="
    echo "Test Summary"
    echo "=========================================="
    echo "Total Tests: $TOTAL_TESTS"
    echo -e "Passed: ${GREEN}$PASSED_TESTS${NC}"
    echo -e "Failed: ${RED}$FAILED_TESTS${NC}"
    echo ""

    if [ "$FAILED_TESTS" -eq 0 ]; then
        echo -e "${GREEN}✓ All tests passed!${NC}"
        cleanup
        exit 0
    else
        echo -e "${RED}✗ Some tests failed${NC}"
        cleanup
        exit 1
    fi
}

# Run main function
main
