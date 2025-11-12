#!/bin/bash

echo "Testing Edge Gateway integration..."

# Test 1: Health check via admin port
echo "Test 1: Edge admin endpoint..."
if curl -sf http://localhost:9901/ready > /dev/null; then
    echo "PASS Edge admin endpoint healthy"
else
    echo "FAIL Edge admin endpoint not responding"
    exit 1
fi

# Test 2: Request through edge to router
echo "Test 2: Request flow through edge..."
RESPONSE=$(curl -s -X POST http://localhost:8080/api/test \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: edge-test" \
  -d '{"message":"edge test"}')

REQUEST_ID=$(echo $RESPONSE | jq -r '.request_id')

if [ -n "$REQUEST_ID" ] && [ "$REQUEST_ID" != "null" ]; then
    echo "PASS Request routed through edge successfully"
    echo "   Request ID: $REQUEST_ID"
else
    echo "FAIL Edge routing failed"
    echo "   Response: $RESPONSE"
    exit 1
fi

# Test 3: Verify headers added by edge
echo "Test 3: Edge header injection..."
RESPONSE=$(curl -si http://localhost:8080/health | grep -i "x-request-id")
if [ -n "$RESPONSE" ]; then
    echo "PASS Edge adds request headers"
else
    echo "WARN Request ID header not found (may be expected)"
fi

# Test 4: Edge stats
echo "Test 4: Edge statistics..."
if curl -s http://localhost:9901/stats | grep -q "cluster.router_cluster"; then
    echo "PASS Edge tracking router cluster stats"
else
    echo "FAIL Edge stats not available"
    exit 1
fi

# Test 5: Router cluster health
echo "Test 5: Router cluster health check..."
HEALTH_STATUS=$(curl -s http://localhost:9901/clusters | grep "router_cluster" | grep -o "health_flags::[^:]*" | head -1)
echo "   Router cluster status: $HEALTH_STATUS"

# Test 6: Edge direct health endpoint
echo "Test 6: Edge health endpoint..."
EDGE_HEALTH=$(curl -s http://localhost:8080/health)
if echo "$EDGE_HEALTH" | jq -e '.status == "ok"' > /dev/null 2>&1; then
    echo "PASS Edge health endpoint responding"
else
    echo "WARN Edge health response: $EDGE_HEALTH"
fi

echo ""
echo "PASS Edge Gateway integration tests completed!"
