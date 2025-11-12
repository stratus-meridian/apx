#!/bin/bash
# GKE End-to-End Test Script

set -e

echo "=== GKE End-to-End Test ==="
echo

# Port-forward in background
echo "Starting port-forward..."
kubectl port-forward -n apx svc/apx-router 8082:8081 >/dev/null 2>&1 &
PF_PID=$!
sleep 3

# Send test request
REQUEST_ID="test-gke-$(date +%s)"
echo "Sending request with ID: $REQUEST_ID"
echo

RESPONSE=$(curl -s -X POST "http://localhost:8082/test" \
  -H "Content-Type: application/json" \
  -H "X-Request-ID: $REQUEST_ID" \
  -H "X-Tenant-ID: test-tenant" \
  -d '{"operation":"echo","message":"GKE E2E Test"}' \
  -w "\nHTTP_CODE:%{http_code}")

echo "Response:"
echo "$RESPONSE"
echo

# Kill port-forward
kill $PF_PID 2>/dev/null || true

# Check router logs
echo "=== Router Logs (last 10 lines) ==="
kubectl logs -n apx -l app=apx-router --tail=10

echo
echo "=== Worker Logs (last 10 lines) ==="
kubectl logs -n apx -l app=apx-worker --tail=10

echo
echo "=== Pub/Sub Topic Messages ==="
gcloud pubsub topics list --project=apx-build-478003 --filter="name:apx-requests-us" --format="table(name)"

echo
echo "Test complete!"
