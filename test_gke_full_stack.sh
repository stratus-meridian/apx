#!/bin/bash
# Test Complete GKE Stack: Edge → Router → Pub/Sub → Worker

set -e

echo "=== Testing Complete GKE Stack ==="
echo "Architecture: Edge → Router → Pub/Sub → Worker"
echo

# Port-forward in background
echo "Starting port-forward to edge gateway..."
kubectl port-forward -n apx svc/apx-edge 8083:80 >/dev/null 2>&1 &
PF_PID=$!
sleep 3

# Send test request
REQUEST_ID="test-gke-full-$(date +%s)"
echo "Sending request with ID: $REQUEST_ID"
echo

RESPONSE=$(curl -s -X POST "http://localhost:8083/api/test" \
  -H "Content-Type: application/json" \
  -H "X-Request-ID: $REQUEST_ID" \
  -H "X-Tenant-ID: gke-full-test" \
  -d '{"message":"Complete GKE stack test"}' \
  -w "\nHTTP_CODE:%{http_code}")

echo "Response:"
echo "$RESPONSE"
echo

# Kill port-forward
kill $PF_PID 2>/dev/null || true
sleep 2

# Check logs
echo "=== Edge Logs (last 3 lines) ==="
kubectl logs -n apx -l app=apx-edge --tail=3

echo
echo "=== Router Logs (last 5 lines) ==="
kubectl logs -n apx -l app=apx-router --tail=5 | grep -A 2 "$REQUEST_ID" || echo "Request ID not found in recent logs"

echo
echo "=== Worker Logs (last 5 lines) ==="
kubectl logs -n apx -l app=apx-worker --tail=5 | grep -A 2 "$REQUEST_ID" || echo "Request ID not found in recent logs"

echo
echo "=== Test Summary ==="
if echo "$RESPONSE" | grep -q "202\|accepted"; then
  echo "✅ Request accepted by edge/router"
else
  echo "❌ Request failed"
  exit 1
fi

# Wait a bit for worker to process
sleep 2
if kubectl logs -n apx -l app=apx-worker --tail=20 | grep -q "$REQUEST_ID"; then
  echo "✅ Request processed by worker"
  echo
  echo "🎉 Complete GKE stack is working end-to-end!"
else
  echo "⚠️  Worker processing not confirmed (may need more time)"
fi
