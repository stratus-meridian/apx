#!/bin/bash
# Quick GKE Integration Test - Mac compatible

set -e

NAMESPACE="apx"
PASSED=0
FAILED=0
TEST_NUM=0

test_endpoint() {
    ((TEST_NUM++))
    local name=$1
    local svc=$2
    local port=$3
    local path=$4
    
    echo "[$TEST_NUM] Testing $name..."
    
    kubectl port-forward -n $NAMESPACE svc/$svc 9000:$port >/dev/null 2>&1 &
    local PID=$!
    sleep 2
    
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:9000$path)
    
    kill $PID 2>/dev/null
    sleep 1
    
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "202" ]; then
        echo "✓ PASS: $name (HTTP $HTTP_CODE)"
        ((PASSED++))
    else
        echo "✗ FAIL: $name (HTTP $HTTP_CODE)"
        ((FAILED++))
    fi
}

test_e2e_flow() {
    ((TEST_NUM++))
    echo "[$TEST_NUM] Testing Edge → Router → Pub/Sub → Worker..."
    
    kubectl port-forward -n $NAMESPACE svc/apx-edge 9000:80 >/dev/null 2>&1 &
    local PID=$!
    sleep 2
    
    REQUEST_ID="test-$(date +%s)"
    RESPONSE_FILE="/tmp/gke_test_$$"
    
    curl -s -X POST http://localhost:9000/api/test \
      -H "Content-Type: application/json" \
      -H "X-Request-ID: $REQUEST_ID" \
      -H "X-Tenant-ID: test" \
      -d '{"test":"e2e"}' > $RESPONSE_FILE
    
    kill $PID 2>/dev/null
    
    if grep -q "\"request_id\":\"$REQUEST_ID\"" $RESPONSE_FILE; then
        echo "✓ PASS: Request accepted with ID"
        ((PASSED++))
        
        # Check worker logs
        sleep 5
        if kubectl logs -n $NAMESPACE -l app=apx-worker --tail=50 | grep -q "$REQUEST_ID"; then
            ((TEST_NUM++))
            echo "[$TEST_NUM] Worker processing..."
            echo "✓ PASS: Worker processed message"
            ((PASSED++))
        else
            ((TEST_NUM++))
            echo "✗ FAIL: Worker did not process"
            ((FAILED++))
        fi
    else
        echo "✗ FAIL: Request not accepted properly"
        ((FAILED++))
    fi
    
    rm -f $RESPONSE_FILE
}

echo "============================================"
echo "GKE Quick Integration Test"
echo "============================================"

# Health checks
test_endpoint "Edge Health" "apx-edge" "80" "/healthz"
test_endpoint "Router Health" "apx-router" "8081" "/health"
test_endpoint "Worker Health" "apx-worker" "8080" "/health"

# E2E flow
test_e2e_flow

echo ""
echo "============================================"
echo "Results: $PASSED passed, $FAILED failed"
echo "============================================"

kubectl get pods -n $NAMESPACE

if [ $FAILED -gt 0 ]; then
    exit 1
fi
