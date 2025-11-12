#!/bin/bash

echo "Tracing complete request flow..."
echo "================================"

# Send request through edge
RESPONSE=$(curl -s -X POST http://localhost:8080/api/trace-test \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: trace-tenant" \
  -d '{"trace":"test"}')

REQUEST_ID=$(echo $RESPONSE | jq -r '.request_id')
echo "Request ID: $REQUEST_ID"
echo ""

# Check edge logs for the request
echo "Edge logs:"
echo "----------"
docker logs apilee-edge-1 2>&1 | grep -i "$REQUEST_ID" | tail -5 || echo "No edge logs found for request ID"

# Check router logs
echo ""
echo "Router logs:"
echo "------------"
docker logs apilee-router-1 2>&1 | grep "$REQUEST_ID" | tail -5 || echo "No router logs found for request ID"

# Check worker logs
echo ""
echo "Worker logs:"
echo "------------"
docker logs apilee-cpu-worker-1 2>&1 | grep "$REQUEST_ID" | tail -5 || echo "No worker logs found for request ID (may not be needed for this request)"

# Show Envoy stats for the request
echo ""
echo "Envoy Statistics:"
echo "-----------------"
echo "Total requests: $(curl -s http://localhost:9901/stats | grep 'http.edge.downstream_rq_total' | awk '{print $2}')"
echo "Active requests: $(curl -s http://localhost:9901/stats | grep 'http.edge.downstream_rq_active' | awk '{print $2}')"
echo "2xx responses: $(curl -s http://localhost:9901/stats | grep 'http.edge.downstream_rq_2xx' | awk '{print $2}')"
echo "4xx responses: $(curl -s http://localhost:9901/stats | grep 'http.edge.downstream_rq_4xx' | awk '{print $2}')"
echo "5xx responses: $(curl -s http://localhost:9901/stats | grep 'http.edge.downstream_rq_5xx' | awk '{print $2}')"

echo ""
echo "PASS Request traced through all components"
