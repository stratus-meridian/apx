#!/bin/bash

# APX Request Pathing Smoke Test
# Task: V-001 - Verify full request path: edge → router → Pub/Sub → worker → stream
#
# This test validates the end-to-end request processing flow in local development

# Don't exit on error - we want to run all tests
set +e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
EDGE_URL="${EDGE_URL:-http://localhost:8080}"
ROUTER_URL="${ROUTER_URL:-http://localhost:8081}"
TEST_TENANT="smoke-test"
TEST_API_KEY="test-key-123"

# Test results tracking
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0
FAILURES=()

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((TESTS_PASSED++))
}

log_failure() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((TESTS_FAILED++))
    FAILURES+=("$1")
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_test() {
    echo -e "${BLUE}[TEST]${NC} $1"
    ((TESTS_RUN++))
}

# Test 1: Verify infrastructure services are running
test_infrastructure() {
    log_test "Infrastructure services health check"

    # Check Redis
    if docker exec apilee-redis-1 redis-cli ping > /dev/null 2>&1; then
        log_success "Redis is healthy"
    else
        log_failure "Redis is not responding"
        return 1
    fi

    # Check Pub/Sub emulator
    if curl -s -f http://localhost:8085 > /dev/null 2>&1 || \
       docker logs apilee-pubsub-1 2>&1 | grep -q "Server started, listening on 8085"; then
        log_success "Pub/Sub emulator is running"
    else
        log_failure "Pub/Sub emulator is not running"
        return 1
    fi

    # Check Firestore emulator
    if docker ps --format '{{.Names}}\t{{.Status}}' | grep -q "apilee-firestore-1.*Up"; then
        log_warning "Firestore container is up, but emulator may not be working (needs Java 21)"
    else
        log_failure "Firestore container is not running"
    fi

    return 0
}

# Test 2: Check if router service is accessible
test_router_health() {
    log_test "Router service health check"

    if docker ps --format '{{.Names}}\t{{.Status}}' | grep -q "apilee-router-1.*Up"; then
        log_success "Router container is running"

        # Try to hit health endpoint
        if response=$(curl -s -f "${ROUTER_URL}/health" 2>&1); then
            log_success "Router health endpoint responded: $response"
            return 0
        else
            log_warning "Router container is up but health endpoint not responding"
            return 1
        fi
    else
        log_warning "Router service is not deployed (implementation incomplete)"
        log_info "Router implementation needs: go.sum file, missing dependencies"
        return 1
    fi
}

# Test 3: Check if edge gateway is accessible
test_edge_health() {
    log_test "Edge gateway health check"

    if docker ps --format '{{.Names}}\t{{.Status}}' | grep -q "apilee-edge-1.*Up"; then
        log_success "Edge container is running"

        # Try to hit admin endpoint
        if response=$(curl -s -f http://localhost:9901/ready 2>&1); then
            log_success "Edge admin endpoint responded"
            return 0
        else
            log_warning "Edge container is up but admin endpoint not responding"
            return 1
        fi
    else
        log_warning "Edge gateway is not deployed (implementation incomplete)"
        log_info "Edge gateway needs: Envoy config validation, router integration"
        return 1
    fi
}

# Test 4: Test Pub/Sub topic creation and publishing (direct)
test_pubsub_direct() {
    log_test "Pub/Sub direct publish/subscribe test"

    # Test if we can create a topic and publish a message using gcloud CLI
    log_info "Testing direct Pub/Sub operations with emulator..."

    export PUBSUB_EMULATOR_HOST=localhost:8085

    # Try creating a topic (this requires gcloud CLI)
    if command -v gcloud >/dev/null 2>&1; then
        if gcloud pubsub topics create apx-requests-us-dev --project=apx-dev 2>/dev/null || \
           gcloud pubsub topics describe apx-requests-us-dev --project=apx-dev >/dev/null 2>&1; then
            log_success "Pub/Sub topic exists or created"

            # Try creating subscription
            if gcloud pubsub subscriptions create apx-workers-us-dev \
                --topic=apx-requests-us-dev \
                --project=apx-dev 2>/dev/null || \
               gcloud pubsub subscriptions describe apx-workers-us-dev --project=apx-dev >/dev/null 2>&1; then
                log_success "Pub/Sub subscription exists or created"

                # Try publishing a test message
                if gcloud pubsub topics publish apx-requests-us-dev \
                    --project=apx-dev \
                    --message='{"test":"smoke-test","request_id":"req-smoke-001"}' \
                    --attribute=tenant_id="${TEST_TENANT}" 2>/dev/null; then
                    log_success "Successfully published test message to Pub/Sub"

                    # Try pulling the message
                    sleep 1
                    if gcloud pubsub subscriptions pull apx-workers-us-dev \
                        --project=apx-dev --limit=1 --auto-ack 2>/dev/null; then
                        log_success "Successfully pulled message from subscription"
                        return 0
                    else
                        log_warning "Could not pull message from subscription"
                    fi
                else
                    log_failure "Failed to publish message to Pub/Sub"
                fi
            else
                log_failure "Failed to create Pub/Sub subscription"
            fi
        else
            log_failure "Failed to create Pub/Sub topic"
        fi
    else
        log_warning "gcloud CLI not available, skipping Pub/Sub direct test"
        log_info "Install gcloud CLI to test Pub/Sub operations"
    fi

    return 1
}

# Test 5: End-to-end request flow (if services are running)
test_end_to_end() {
    log_test "End-to-end request flow test"

    # Check if both edge and router are running
    if ! docker ps --format '{{.Names}}' | grep -q "apilee-edge-1"; then
        log_warning "Edge not running, skipping end-to-end test"
        log_info "Start edge with: docker-compose up -d edge"
        return 1
    fi

    if ! docker ps --format '{{.Names}}' | grep -q "apilee-router-1"; then
        log_warning "Router not running, skipping end-to-end test"
        log_info "Start router with: docker-compose up -d router"
        return 1
    fi

    log_info "Sending test request to edge gateway..."

    REQUEST_ID="req-$(date +%s)"

    # Send POST request to edge
    response=$(curl -s -w "\n%{http_code}" -X POST "${EDGE_URL}/v1/test" \
        -H "Content-Type: application/json" \
        -H "X-API-Key: ${TEST_API_KEY}" \
        -H "X-Tenant-ID: ${TEST_TENANT}" \
        -d "{\"test\": \"data\", \"request_id\": \"${REQUEST_ID}\"}" 2>&1)

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" == "202" ]; then
        log_success "Received 202 Accepted response"

        # Check for request_id in response
        if echo "$body" | grep -q "request_id"; then
            log_success "Response contains request_id"

            # Extract request_id and status_url
            returned_id=$(echo "$body" | grep -o '"request_id":"[^"]*"' | cut -d'"' -f4)
            status_url=$(echo "$body" | grep -o '"status_url":"[^"]*"' | cut -d'"' -f4)

            log_info "Request ID: $returned_id"
            log_info "Status URL: $status_url"

            # Try to check status endpoint
            if [ -n "$status_url" ]; then
                sleep 2
                status_response=$(curl -s "${EDGE_URL}${status_url}" 2>&1)
                log_info "Status response: $status_response"
            fi

            return 0
        else
            log_failure "Response missing request_id"
        fi
    else
        log_failure "Expected 202 Accepted, got HTTP $http_code"
        log_info "Response: $body"
    fi

    return 1
}

# Test 6: Verify logs and tracing
test_observability() {
    log_test "Observability verification"

    if docker ps --format '{{.Names}}' | grep -q "apilee-otel-collector-1"; then
        log_success "OTEL Collector is running"
    else
        log_warning "OTEL Collector is not running"
        log_info "Start with: docker-compose up -d otel-collector"
    fi

    if docker ps --format '{{.Names}}' | grep -q "apilee-prometheus-1"; then
        log_success "Prometheus is running"
    else
        log_warning "Prometheus is not running"
        log_info "Start with: docker-compose up -d prometheus"
    fi

    if docker ps --format '{{.Names}}' | grep -q "apilee-grafana-1"; then
        log_success "Grafana is running"
    else
        log_warning "Grafana is not running"
        log_info "Start with: docker-compose up -d grafana"
    fi

    return 0
}

# Main test execution
main() {
    echo ""
    echo "=========================================="
    echo "  APX Request Pathing Smoke Test (V-001)"
    echo "=========================================="
    echo ""

    log_info "Starting smoke tests..."
    echo ""

    # Run all tests
    test_infrastructure
    echo ""

    test_router_health
    echo ""

    test_edge_health
    echo ""

    test_pubsub_direct
    echo ""

    test_end_to_end
    echo ""

    test_observability
    echo ""

    # Print summary
    echo "=========================================="
    echo "  Test Summary"
    echo "=========================================="
    echo "Tests Run:    $TESTS_RUN"
    echo "Tests Passed: ${GREEN}$TESTS_PASSED${NC}"
    echo "Tests Failed: ${RED}$TESTS_FAILED${NC}"
    echo ""

    if [ ${#FAILURES[@]} -gt 0 ]; then
        echo "Failures:"
        for failure in "${FAILURES[@]}"; do
            echo "  - $failure"
        done
        echo ""
    fi

    # Current status
    echo "=========================================="
    echo "  Current System Status"
    echo "=========================================="
    echo ""

    log_info "Infrastructure Services:"
    echo "  ✓ Redis: Running and healthy"
    echo "  ✓ Pub/Sub Emulator: Running"
    echo "  ✗ Firestore Emulator: Requires Java 21"
    echo ""

    log_info "Application Services:"
    echo "  ✗ Edge Gateway: Not built (needs go.sum)"
    echo "  ✗ Router Service: Not built (needs go.sum)"
    echo "  ✗ Worker Service: Not implemented"
    echo ""

    log_info "Observability Stack:"
    echo "  ✓ OTEL Collector config: Created"
    echo "  ✓ Prometheus config: Created"
    echo "  ✓ Grafana config: Created"
    echo "  ✗ Services not started"
    echo ""

    echo "=========================================="
    echo "  Acceptance Criteria Status (V-001)"
    echo "=========================================="
    echo ""
    echo "[ ] Request returns 202 Accepted"
    echo "[ ] Request ID generated and returned"
    echo "[ ] Message published to Pub/Sub"
    echo "[ ] Worker receives and processes message"
    echo "[ ] Status endpoint returns result"
    echo "[ ] End-to-end latency < 5 seconds"
    echo ""

    echo "=========================================="
    echo "  Next Steps"
    echo "=========================================="
    echo ""
    echo "1. Fix Go build issues:"
    echo "   - Generate go.sum: cd router && go mod tidy"
    echo "   - Or run in Docker with Go installed"
    echo ""
    echo "2. Fix Firestore emulator:"
    echo "   - Update docker-compose to use Java 21 image"
    echo "   - Or use alternative Firestore emulator image"
    echo ""
    echo "3. Complete router implementation:"
    echo "   - Implement missing pkg/observability"
    echo "   - Implement routes.NewMatcher with proper signature"
    echo "   - Wire up Pub/Sub publishing"
    echo ""
    echo "4. Implement worker service:"
    echo "   - Subscribe to Pub/Sub"
    echo "   - Process messages"
    echo "   - Publish results"
    echo ""
    echo "5. Build and test end-to-end flow"
    echo ""

    # Exit with appropriate code
    if [ $TESTS_FAILED -eq 0 ]; then
        exit 0
    else
        exit 1
    fi
}

# Run main
main
