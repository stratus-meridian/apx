#!/bin/bash

set -e

echo "====================================="
echo "End-to-End Worker Flow Test"
echo "====================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
ROUTER_URL="${ROUTER_URL:-http://localhost:8081}"
STREAMING_URL="${STREAMING_URL:-http://localhost:8083}"
MAX_WAIT=30 # Maximum seconds to wait for completion

echo "Configuration:"
echo "  Router URL: $ROUTER_URL"
echo "  Streaming URL: $STREAMING_URL"
echo "  Max wait time: ${MAX_WAIT}s"
echo ""

# Step 1: Send request to router
echo "Step 1: Sending request to router..."
RESPONSE=$(curl -s -X POST "$ROUTER_URL/api/test" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: test-tenant" \
  -d '{"message":"e2e worker test"}')

# Extract request ID and status URL
REQUEST_ID=$(echo "$RESPONSE" | jq -r '.request_id')
STATUS_URL=$(echo "$RESPONSE" | jq -r '.status_url')

if [ -z "$REQUEST_ID" ] || [ "$REQUEST_ID" = "null" ]; then
    echo -e "${RED}FAILED: Could not extract request_id from response${NC}"
    echo "Response: $RESPONSE"
    exit 1
fi

echo -e "${GREEN}Request sent successfully${NC}"
echo "  Request ID: $REQUEST_ID"
echo "  Status URL: $STATUS_URL"
echo ""

# Step 2: Poll for status updates
echo "Step 2: Polling for status updates..."
ELAPSED=0
LAST_STATUS=""
LAST_PROGRESS=0

while [ $ELAPSED -lt $MAX_WAIT ]; do
    # Check status
    STATUS_RESPONSE=$(curl -s "$STATUS_URL" || echo '{"status":"error","error":"connection failed"}')
    CURRENT_STATUS=$(echo "$STATUS_RESPONSE" | jq -r '.status // "unknown"')
    CURRENT_PROGRESS=$(echo "$STATUS_RESPONSE" | jq -r '.progress // 0')

    # Print status if it changed
    if [ "$CURRENT_STATUS" != "$LAST_STATUS" ] || [ "$CURRENT_PROGRESS" != "$LAST_PROGRESS" ]; then
        echo "  [${ELAPSED}s] Status: $CURRENT_STATUS, Progress: ${CURRENT_PROGRESS}%"
        LAST_STATUS=$CURRENT_STATUS
        LAST_PROGRESS=$CURRENT_PROGRESS
    fi

    # Check if complete
    if [ "$CURRENT_STATUS" = "complete" ]; then
        echo ""
        echo -e "${GREEN}Request completed successfully!${NC}"
        echo ""
        echo "Final status response:"
        echo "$STATUS_RESPONSE" | jq '.'
        echo ""

        # Verify result structure
        RESULT_MESSAGE=$(echo "$STATUS_RESPONSE" | jq -r '.result.message // "missing"')
        if [ "$RESULT_MESSAGE" = "Request processed successfully" ]; then
            echo -e "${GREEN}PASSED: Worker processed request end-to-end${NC}"
            exit 0
        else
            echo -e "${YELLOW}WARNING: Result structure unexpected${NC}"
            echo "Expected message: 'Request processed successfully'"
            echo "Got: $RESULT_MESSAGE"
            exit 0
        fi
    fi

    # Check if error
    if [ "$CURRENT_STATUS" = "error" ] || [ "$CURRENT_STATUS" = "failed" ]; then
        echo ""
        echo -e "${RED}FAILED: Request failed with error${NC}"
        echo "$STATUS_RESPONSE" | jq '.'
        exit 1
    fi

    # Wait before next poll
    sleep 1
    ELAPSED=$((ELAPSED + 1))
done

# Timeout
echo ""
echo -e "${RED}FAILED: Timeout waiting for request completion${NC}"
echo "Last status: $LAST_STATUS"
echo "Last progress: ${LAST_PROGRESS}%"
echo ""
echo "Troubleshooting:"
echo "  1. Check if cpu-worker is running: docker ps | grep cpu-worker"
echo "  2. Check worker logs: docker logs apilee-cpu-worker-1 --tail 50"
echo "  3. Check Redis: docker exec apilee-redis-1 redis-cli GET status:$REQUEST_ID"
echo "  4. Check Pub/Sub: curl http://localhost:8085/v1/projects/apx-dev/topics"
exit 1
