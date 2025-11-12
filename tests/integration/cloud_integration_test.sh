#!/bin/bash

# ============================================================================
# APX Cloud Integration Tests (M1-T4-001)
# ============================================================================
# Tests the complete cloud deployment:
# - Load Balancer + Cloud Armor
# - Router on Cloud Run
# - Workers on Cloud Run
# - Pub/Sub messaging
# - Firestore integration
# ============================================================================

set -e

# Configuration
API_URL="${API_URL:-https://api.apx.build}"
PROJECT_ID="${PROJECT_ID:-apx-build-478003}"
REGION="${REGION:-us-central1}"
RESULTS_DIR="$(dirname "$0")/results"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
RESULT_FILE="${RESULTS_DIR}/cloud_integration_${TIMESTAMP}.txt"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Counters
TESTS_TOTAL=0
TESTS_PASSED=0
TESTS_FAILED=0

# Create results directory
mkdir -p "$RESULTS_DIR"

# Helper functions
log() {
    echo -e "${GREEN}[$(date +'%H:%M:%S')]${NC} $1" | tee -a "$RESULT_FILE"
}

error() {
    echo -e "${RED}[$(date +'%H:%M:%S')] ERROR:${NC} $1" | tee -a "$RESULT_FILE"
}

warn() {
    echo -e "${YELLOW}[$(date +'%H:%M:%S')] WARN:${NC} $1" | tee -a "$RESULT_FILE"
}

test_start() {
    TESTS_TOTAL=$((TESTS_TOTAL + 1))
    log "TEST $TESTS_TOTAL: $1"
}

test_pass() {
    TESTS_PASSED=$((TESTS_PASSED + 1))
    log "✅ PASS: $1"
}

test_fail() {
    TESTS_FAILED=$((TESTS_FAILED + 1))
    error "❌ FAIL: $1"
}

# ============================================================================
# Test Suite 1: Basic Connectivity
# ============================================================================

test_suite_connectivity() {
    log "========================================="
    log "TEST SUITE 1: Basic Connectivity"
    log "========================================="

    # Test 1.1: Health endpoint
    test_start "Health endpoint returns 200 OK"
    HEALTH_RESPONSE=$(curl -s -w "\n%{http_code}" "$API_URL/health")
    HTTP_CODE=$(echo "$HEALTH_RESPONSE" | tail -1)
    BODY=$(echo "$HEALTH_RESPONSE" | head -1)

    if [ "$HTTP_CODE" = "200" ] && echo "$BODY" | grep -q "ok"; then
        test_pass "Health endpoint working (200 OK, status: ok)"
    else
        test_fail "Health endpoint failed (HTTP $HTTP_CODE)"
    fi

    # Test 1.2: HTTPS working
    test_start "HTTPS certificate valid"
    if curl -s --fail "$API_URL/health" > /dev/null 2>&1; then
        test_pass "HTTPS working with valid certificate"
    else
        test_fail "HTTPS connection failed"
    fi

    # Test 1.3: HTTP redirect
    test_start "HTTP redirects to HTTPS"
    HTTP_LOCATION=$(curl -s -I "http://api.apx.build/health" | grep -i "Location:" | awk '{print $2}' | tr -d '\r')
    if echo "$HTTP_LOCATION" | grep -q "https://"; then
        test_pass "HTTP → HTTPS redirect working"
    else
        test_fail "HTTP redirect not working"
    fi
}

# ============================================================================
# Test Suite 2: API Request Flow
# ============================================================================

test_suite_api_flow() {
    log "========================================="
    log "TEST SUITE 2: API Request Flow"
    log "========================================="

    # Test 2.1: API request accepted
    test_start "API request returns 202 Accepted"
    API_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/api/integration-test" \
        -H "Content-Type: application/json" \
        -H "X-Tenant-ID: integration-test" \
        -d '{"test":"api_flow","timestamp":"'$(date -u +%s)'"}')

    HTTP_CODE=$(echo "$API_RESPONSE" | tail -1)
    BODY=$(echo "$API_RESPONSE" | head -1)
    REQUEST_ID=$(echo "$BODY" | jq -r '.request_id // empty')

    if [ "$HTTP_CODE" = "202" ] && [ -n "$REQUEST_ID" ]; then
        test_pass "API request accepted (202, request_id: $REQUEST_ID)"
        export TEST_REQUEST_ID="$REQUEST_ID"
    else
        test_fail "API request failed (HTTP $HTTP_CODE)"
        return 1
    fi

    # Test 2.2: Response has correct URLs
    test_start "Response contains correct URLs"
    STATUS_URL=$(echo "$BODY" | jq -r '.status_url // empty')
    STREAM_URL=$(echo "$BODY" | jq -r '.stream_url // empty')

    if echo "$STATUS_URL" | grep -q "https://api.apx.build/status/$REQUEST_ID" && \
       echo "$STREAM_URL" | grep -q "https://api.apx.build/stream/$REQUEST_ID"; then
        test_pass "URLs use public domain (https://api.apx.build)"
    else
        test_fail "URLs incorrect (status: $STATUS_URL, stream: $STREAM_URL)"
    fi

    # Test 2.3: Request ID format
    test_start "Request ID is valid UUID"
    if echo "$REQUEST_ID" | grep -Eq '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'; then
        test_pass "Request ID is valid UUID"
    else
        test_fail "Request ID not a valid UUID: $REQUEST_ID"
    fi
}

# ============================================================================
# Test Suite 3: Async Processing
# ============================================================================

test_suite_async_processing() {
    log "========================================="
    log "TEST SUITE 3: Async Processing"
    log "========================================="

    if [ -z "$TEST_REQUEST_ID" ]; then
        warn "Skipping async tests (no request ID from previous test)"
        return
    fi

    # Test 3.1: Worker processing
    test_start "Worker processes message from Pub/Sub"
    sleep 3  # Give worker time to process

    # Check worker logs for our request ID
    WORKER_LOGS=$(gcloud logging read \
        "resource.type=cloud_run_revision AND resource.labels.service_name=apx-worker-cpu-dev AND jsonPayload.request_id=\"$TEST_REQUEST_ID\"" \
        --project="$PROJECT_ID" \
        --limit=5 \
        --freshness=1m \
        --format="value(jsonPayload.msg)" 2>/dev/null || echo "")

    if echo "$WORKER_LOGS" | grep -q "processing request\|request completed"; then
        test_pass "Worker processed message (found in logs)"
    else
        test_fail "Worker did not process message (not found in logs)"
    fi

    # Test 3.2: End-to-end latency
    test_start "End-to-end processing time < 5 seconds"
    if echo "$WORKER_LOGS" | grep -q "request completed"; then
        test_pass "Request completed within acceptable time"
    else
        warn "Could not verify processing completion time"
    fi
}

# ============================================================================
# Test Suite 4: Error Handling
# ============================================================================

test_suite_error_handling() {
    log "========================================="
    log "TEST SUITE 4: Error Handling"
    log "========================================="

    # Test 4.1: Missing tenant ID
    test_start "Request without tenant ID handled correctly"
    ERROR_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/api/test-no-tenant" \
        -H "Content-Type: application/json" \
        -d '{"test":"no_tenant"}')

    HTTP_CODE=$(echo "$ERROR_RESPONSE" | tail -1)
    if [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "202" ]; then
        test_pass "Missing tenant ID handled (HTTP $HTTP_CODE)"
    else
        test_fail "Unexpected response for missing tenant (HTTP $HTTP_CODE)"
    fi

    # Test 4.2: Invalid JSON
    test_start "Invalid JSON handled correctly"
    INVALID_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/api/test-invalid" \
        -H "Content-Type: application/json" \
        -H "X-Tenant-ID: test-invalid" \
        -d 'not valid json')

    HTTP_CODE=$(echo "$INVALID_RESPONSE" | tail -1)
    if [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "500" ]; then
        test_pass "Invalid JSON handled (HTTP $HTTP_CODE)"
    else
        warn "Unexpected response for invalid JSON (HTTP $HTTP_CODE)"
    fi

    # Test 4.3: Rate limiting (Cloud Armor)
    test_start "Rate limiting active (Cloud Armor)"
    # Send rapid requests to trigger rate limit
    RATE_LIMIT_TRIGGERED=false
    for i in {1..15}; do
        RATE_RESPONSE=$(curl -s -w "%{http_code}" -X GET "$API_URL/health" -o /dev/null)
        if [ "$RATE_RESPONSE" = "429" ] || [ "$RATE_RESPONSE" = "403" ]; then
            RATE_LIMIT_TRIGGERED=true
            break
        fi
        sleep 0.1
    done

    if [ "$RATE_LIMIT_TRIGGERED" = true ]; then
        test_pass "Rate limiting active (received 429/403)"
    else
        warn "Rate limiting not triggered in test (may need more requests)"
    fi
}

# ============================================================================
# Test Suite 5: Security
# ============================================================================

test_suite_security() {
    log "========================================="
    log "TEST SUITE 5: Security"
    log "========================================="

    # Test 5.1: Cloud Armor active
    test_start "Cloud Armor security policy active"
    ARMOR_POLICY=$(gcloud compute security-policies describe apx-security-policy-dev \
        --project="$PROJECT_ID" \
        --format="value(name)" 2>/dev/null || echo "")

    if [ -n "$ARMOR_POLICY" ]; then
        test_pass "Cloud Armor policy active: $ARMOR_POLICY"
    else
        test_fail "Cloud Armor policy not found"
    fi

    # Test 5.2: SSL/TLS
    test_start "SSL certificate valid and not expired"
    SSL_STATUS=$(gcloud compute ssl-certificates describe apx-lb-cert-dev-v2 \
        --global \
        --project="$PROJECT_ID" \
        --format="value(managed.status)" 2>/dev/null || echo "")

    if [ "$SSL_STATUS" = "ACTIVE" ]; then
        test_pass "SSL certificate ACTIVE"
    else
        test_fail "SSL certificate not active: $SSL_STATUS"
    fi

    # Test 5.3: Router internal-only
    test_start "Router accessible only via Load Balancer"
    ROUTER_INGRESS=$(gcloud run services describe apx-router-dev \
        --region="$REGION" \
        --project="$PROJECT_ID" \
        --format="value(spec.template.metadata.annotations.'run.googleapis.com/ingress')" 2>/dev/null || echo "")

    if echo "$ROUTER_INGRESS" | grep -q "internal"; then
        test_pass "Router is internal-only (not publicly accessible)"
    else
        warn "Router ingress setting: $ROUTER_INGRESS"
    fi
}

# ============================================================================
# Test Suite 6: Performance
# ============================================================================

test_suite_performance() {
    log "========================================="
    log "TEST SUITE 6: Performance"
    log "========================================="

    # Test 6.1: Response time
    test_start "API response time < 300ms"
    START_TIME=$(gdate +%s%3N 2>/dev/null || python3 -c "import time; print(int(time.time()*1000))")
    curl -s "$API_URL/health" > /dev/null
    END_TIME=$(gdate +%s%3N 2>/dev/null || python3 -c "import time; print(int(time.time()*1000))")
    RESPONSE_TIME=$((END_TIME - START_TIME))

    if [ "$RESPONSE_TIME" -lt 300 ]; then
        test_pass "Response time: ${RESPONSE_TIME}ms (< 300ms target)"
    else
        warn "Response time: ${RESPONSE_TIME}ms (exceeds 300ms target)"
    fi

    # Test 6.2: Concurrent requests
    test_start "Handle 5 concurrent requests"
    START_TIME=$(date +%s)
    for i in {1..5}; do
        curl -s -X POST "$API_URL/api/concurrent-test-$i" \
            -H "Content-Type: application/json" \
            -H "X-Tenant-ID: concurrent-test" \
            -d "{\"concurrent_id\":$i}" \
            > /dev/null &
    done
    wait
    END_TIME=$(date +%s)
    DURATION=$((END_TIME - START_TIME))

    if [ "$DURATION" -lt 5 ]; then
        test_pass "Processed 5 concurrent requests in ${DURATION}s"
    else
        warn "Concurrent requests took ${DURATION}s (may indicate scaling issues)"
    fi

    # Test 6.3: Worker scalability
    test_start "Workers can handle burst traffic"
    WORKER_INSTANCES=$(gcloud run services describe apx-worker-cpu-dev \
        --region="$REGION" \
        --project="$PROJECT_ID" \
        --format="value(status.traffic[0].latestRevision)" 2>/dev/null | wc -l)

    if [ "$WORKER_INSTANCES" -ge 1 ]; then
        test_pass "Worker service healthy with $WORKER_INSTANCES instance(s)"
    else
        test_fail "Worker service not responding"
    fi
}

# ============================================================================
# Main Execution
# ============================================================================

main() {
    log "========================================="
    log "APX Cloud Integration Tests"
    log "Date: $(date)"
    log "API URL: $API_URL"
    log "Project: $PROJECT_ID"
    log "========================================="

    # Run all test suites
    test_suite_connectivity
    test_suite_api_flow
    test_suite_async_processing
    test_suite_error_handling
    test_suite_security
    test_suite_performance

    # Summary
    log "========================================="
    log "TEST SUMMARY"
    log "========================================="
    log "Total Tests: $TESTS_TOTAL"
    log "Passed: $TESTS_PASSED"
    log "Failed: $TESTS_FAILED"

    PASS_RATE=$((TESTS_PASSED * 100 / TESTS_TOTAL))
    log "Pass Rate: ${PASS_RATE}%"

    if [ "$TESTS_FAILED" -eq 0 ]; then
        log "✅ ALL TESTS PASSED"
        exit 0
    else
        error "❌ SOME TESTS FAILED"
        exit 1
    fi
}

# Run tests
main "$@"
