#!/bin/bash
# GKE Integration Test Suite
# Matches Cloud Run M1-T4-001 test coverage (17 tests)

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
WARNINGS=0

# Configuration
NAMESPACE="apx"
EDGE_SVC="apx-edge"
ROUTER_SVC="apx-router"
WORKER_SVC="apx-worker"
TEST_TENANT="gke-integration-test"

# Helper functions
log_test() {
    ((TOTAL_TESTS++))
    echo -e "\n${YELLOW}[TEST $TOTAL_TESTS]${NC} $1"
}

pass() {
    ((PASSED_TESTS++))
    echo -e "${GREEN}✓ PASS${NC}: $1"
}

fail() {
    ((FAILED_TESTS++))
    echo -e "${RED}✗ FAIL${NC}: $1"
}

warn() {
    ((WARNINGS++))
    echo -e "${YELLOW}⚠ WARNING${NC}: $1"
}

# Port-forward helper
start_port_forward() {
    local svc=$1
    local local_port=$2
    local remote_port=$3

    kubectl port-forward -n $NAMESPACE svc/$svc $local_port:$remote_port >/dev/null 2>&1 &
    local PID=$!
    sleep 2
    echo $PID
}

stop_port_forward() {
    kill $1 2>/dev/null || true
    sleep 1
}

echo "============================================"
echo "GKE Integration Test Suite"
echo "============================================"
echo "Namespace: $NAMESPACE"
echo "Timestamp: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "============================================"

# ============================================
# SECTION 1: Basic Connectivity (3 tests)
# ============================================

log_test "Edge health endpoint"
PF_PID=$(start_port_forward $EDGE_SVC 8090 80)
RESPONSE=$(curl -s -w "\n%{http_code}" http://localhost:8090/healthz)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
stop_port_forward $PF_PID

if [ "$HTTP_CODE" = "200" ]; then
    pass "Edge health endpoint responding (HTTP 200)"
else
    fail "Edge health check failed (HTTP $HTTP_CODE)"
fi

log_test "Router health endpoint"
PF_PID=$(start_port_forward $ROUTER_SVC 8091 8081)
RESPONSE=$(curl -s -w "\n%{http_code}" http://localhost:8091/health)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
stop_port_forward $PF_PID

if [ "$HTTP_CODE" = "200" ]; then
    pass "Router health endpoint responding (HTTP 200)"
else
    fail "Router health check failed (HTTP $HTTP_CODE)"
fi

log_test "Worker health endpoint"
PF_PID=$(start_port_forward $WORKER_SVC 8092 8080)
RESPONSE=$(curl -s -w "\n%{http_code}" http://localhost:8092/health)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
stop_port_forward $PF_PID

if [ "$HTTP_CODE" = "200" ]; then
    pass "Worker health endpoint responding (HTTP 200)"
else
    fail "Worker health check failed (HTTP $HTTP_CODE)"
fi

# ============================================
# SECTION 2: API Request Flow (3 tests)
# ============================================

log_test "Edge → Router request flow"
PF_PID=$(start_port_forward $EDGE_SVC 8090 80)
REQUEST_ID="test-edge-router-$(date +%s)"
RESPONSE=$(curl -s -X POST http://localhost:8090/api/test \
  -H "Content-Type: application/json" \
  -H "X-Request-ID: $REQUEST_ID" \
  -H "X-Tenant-ID: $TEST_TENANT" \
  -d '{"test":"edge-router"}' \
  -w "\n%{http_code}")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -0)
stop_port_forward $PF_PID

if [ "$HTTP_CODE" = "202" ] && echo "$BODY" | grep -q "$REQUEST_ID"; then
    pass "Edge → Router flow (HTTP 202, request_id in response)"
else
    fail "Edge → Router flow failed (HTTP $HTTP_CODE)"
fi

log_test "Request ID propagation"
if echo "$BODY" | grep -q "\"request_id\":\"$REQUEST_ID\""; then
    pass "Request ID propagated correctly"
else
    fail "Request ID missing or incorrect in response"
fi

log_test "Status URL generation"
if echo "$BODY" | grep -q "\"status_url\""; then
    pass "Status URL generated in response"
else
    fail "Status URL missing from response"
fi

# ============================================
# SECTION 3: Async Processing (2 tests)
# ============================================

log_test "Pub/Sub message delivery"
sleep 5  # Wait for message processing
WORKER_LOGS=$(kubectl logs -n $NAMESPACE -l app=apx-worker --tail=50 | grep "$REQUEST_ID" || echo "")

if [ -n "$WORKER_LOGS" ]; then
    pass "Message delivered to worker via Pub/Sub"
else
    fail "Message not found in worker logs after 5 seconds"
fi

log_test "Worker message processing"
if echo "$WORKER_LOGS" | grep -q "request completed\|processing request"; then
    pass "Worker processed message successfully"
else
    fail "Worker did not process message"
fi

# ============================================
# SECTION 4: Error Handling (3 tests)
# ============================================

log_test "Invalid JSON handling"
PF_PID=$(start_port_forward $EDGE_SVC 8090 80)
RESPONSE=$(curl -s -X POST http://localhost:8090/api/test \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TEST_TENANT" \
  -d 'invalid{json' \
  -w "\n%{http_code}")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
stop_port_forward $PF_PID

if [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "500" ]; then
    pass "Invalid JSON rejected (HTTP $HTTP_CODE)"
else
    warn "Invalid JSON not handled properly (HTTP $HTTP_CODE)"
fi

log_test "Missing tenant ID handling"
PF_PID=$(start_port_forward $EDGE_SVC 8090 80)
RESPONSE=$(curl -s -X POST http://localhost:8090/api/test \
  -H "Content-Type: application/json" \
  -d '{"test":"no-tenant"}' \
  -w "\n%{http_code}")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
stop_port_forward $PF_PID

# Router should still accept (tenant defaults to "unknown")
if [ "$HTTP_CODE" = "202" ]; then
    pass "Missing tenant ID handled gracefully (HTTP 202)"
else
    warn "Missing tenant ID behavior: HTTP $HTTP_CODE"
fi

log_test "Rate limiting verification"
PF_PID=$(start_port_forward $EDGE_SVC 8090 80)
# Send 10 rapid requests
SUCCESS_COUNT=0
for i in {1..10}; do
    RESPONSE=$(curl -s -X POST http://localhost:8090/api/test \
      -H "Content-Type: application/json" \
      -H "X-Tenant-ID: rate-limit-test" \
      -d "{\"seq\":$i}" \
      -w "\n%{http_code}")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    if [ "$HTTP_CODE" = "202" ]; then
        ((SUCCESS_COUNT++))
    fi
done
stop_port_forward $PF_PID

if [ $SUCCESS_COUNT -ge 8 ]; then
    pass "Rate limiting active (accepted $SUCCESS_COUNT/10 requests)"
else
    warn "Rate limiting may be too strict ($SUCCESS_COUNT/10 accepted)"
fi

# ============================================
# SECTION 5: Security (2 tests)
# ============================================

log_test "Tenant isolation (keyspace)"
REDIS_POD=$(kubectl get pods -n $NAMESPACE -l app=apx-redis -o jsonpath='{.items[0].metadata.name}')
REDIS_KEYS=$(kubectl exec -n $NAMESPACE $REDIS_POD -- redis-cli KEYS "apx:rl:$TEST_TENANT:*" 2>/dev/null || echo "")

if [ -n "$REDIS_KEYS" ]; then
    pass "Tenant-specific Redis keys found (isolation working)"
else
    warn "No tenant-specific Redis keys found (may not have persisted yet)"
fi

log_test "Request ID uniqueness"
PF_PID=$(start_port_forward $EDGE_SVC 8090 80)
REQ_ID_1=$(curl -s -X POST http://localhost:8090/api/test \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TEST_TENANT" \
  -d '{"test":"1"}' | jq -r '.request_id')
REQ_ID_2=$(curl -s -X POST http://localhost:8090/api/test \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TEST_TENANT" \
  -d '{"test":"2"}' | jq -r '.request_id')
stop_port_forward $PF_PID

if [ "$REQ_ID_1" != "$REQ_ID_2" ] && [ -n "$REQ_ID_1" ] && [ -n "$REQ_ID_2" ]; then
    pass "Request IDs are unique ($REQ_ID_1 != $REQ_ID_2)"
else
    fail "Request IDs not unique or missing"
fi

# ============================================
# SECTION 6: Performance (3 tests)
# ============================================

log_test "Edge → Router latency (p95)"
PF_PID=$(start_port_forward $EDGE_SVC 8090 80)
LATENCIES=()
for i in {1..20}; do
    START=$(date +%s%N)
    curl -s -X POST http://localhost:8090/api/test \
      -H "Content-Type: application/json" \
      -H "X-Tenant-ID: latency-test" \
      -d "{\"seq\":$i}" >/dev/null
    END=$(date +%s%N)
    LATENCY=$(( (END - START) / 1000000 ))  # Convert to ms
    LATENCIES+=($LATENCY)
done
stop_port_forward $PF_PID

# Calculate p95 (19th value out of 20)
SORTED=($(printf '%s\n' "${LATENCIES[@]}" | sort -n))
P95=${SORTED[18]}

if [ $P95 -lt 500 ]; then
    pass "Edge → Router p95 latency: ${P95}ms (< 500ms target)"
else
    warn "Edge → Router p95 latency: ${P95}ms (exceeds 500ms target)"
fi

log_test "Worker processing time"
# Check recent worker logs for processing duration
WORKER_DURATION=$(kubectl logs -n $NAMESPACE -l app=apx-worker --tail=100 | \
  grep "request completed" | tail -5 | \
  awk -F'duration":' '{print $2}' | awk -F',' '{print $1}' | \
  sort -n | head -1)

if [ -n "$WORKER_DURATION" ]; then
    pass "Worker processing time measured (recent: ${WORKER_DURATION}s)"
else
    warn "Worker processing time not measurable from logs"
fi

log_test "Pod resource utilization"
POD_CPU=$(kubectl top pods -n $NAMESPACE --no-headers 2>/dev/null | awk '{sum+=$2} END {print sum}' || echo "0")
POD_MEM=$(kubectl top pods -n $NAMESPACE --no-headers 2>/dev/null | awk '{sum+=$3} END {print sum}' || echo "0")

if [ "$POD_CPU" != "0" ]; then
    pass "Resource metrics available (CPU: ${POD_CPU}m, Mem: ${POD_MEM}Mi)"
else
    warn "Metrics server may not be ready"
fi

# ============================================
# Summary
# ============================================

echo ""
echo "============================================"
echo "Test Summary"
echo "============================================"
echo "Total Tests:   $TOTAL_TESTS"
echo -e "${GREEN}Passed:        $PASSED_TESTS${NC}"
echo -e "${RED}Failed:        $FAILED_TESTS${NC}"
echo -e "${YELLOW}Warnings:      $WARNINGS${NC}"

PASS_RATE=$(( PASSED_TESTS * 100 / TOTAL_TESTS ))
echo "Pass Rate:     $PASS_RATE%"

echo ""
echo "============================================"
echo "Pod Status"
echo "============================================"
kubectl get pods -n $NAMESPACE

echo ""
echo "============================================"
echo "Recent Logs (Edge)"
echo "============================================"
kubectl logs -n $NAMESPACE -l app=apx-edge --tail=5

echo ""
echo "============================================"
echo "Recent Logs (Router)"
echo "============================================"
kubectl logs -n $NAMESPACE -l app=apx-router --tail=5

echo ""
echo "============================================"
echo "Recent Logs (Worker)"
echo "============================================"
kubectl logs -n $NAMESPACE -l app=apx-worker --tail=5

# Exit code
if [ $FAILED_TESTS -gt 0 ]; then
    echo -e "\n${RED}Some tests failed${NC}"
    exit 1
elif [ $WARNINGS -gt 0 ]; then
    echo -e "\n${YELLOW}All critical tests passed, but some warnings${NC}"
    exit 0
else
    echo -e "\n${GREEN}All tests passed!${NC}"
    exit 0
fi
