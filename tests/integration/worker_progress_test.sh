#!/bin/bash

set -e

echo "====================================="
echo "Worker Progress Updates Test"
echo "====================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Configuration
ROUTER_URL="${ROUTER_URL:-http://localhost:8081}"
MAX_WAIT=15

echo "Testing real-time progress updates..."
echo ""

# Send request
RESPONSE=$(curl -s -X POST "$ROUTER_URL/api/test" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: progress-test" \
  -d '{"message":"progress tracking test"}')

REQUEST_ID=$(echo "$RESPONSE" | jq -r '.request_id')
STATUS_URL=$(echo "$RESPONSE" | jq -r '.status_url')

echo -e "${BLUE}Request ID:${NC} $REQUEST_ID"
echo ""

# Track progress
ELAPSED=0
LAST_PROGRESS=-1
PROGRESS_UPDATES=0

echo "Progress Timeline:"
echo "----------------"

while [ $ELAPSED -lt $MAX_WAIT ]; do
    STATUS_RESPONSE=$(curl -s "$STATUS_URL" 2>/dev/null || echo '{}')
    CURRENT_STATUS=$(echo "$STATUS_RESPONSE" | jq -r '.status // "unknown"')
    CURRENT_PROGRESS=$(echo "$STATUS_RESPONSE" | jq -r '.progress // 0')

    # Print progress updates
    if [ "$CURRENT_PROGRESS" != "$LAST_PROGRESS" ]; then
        TIMESTAMP=$(date +"%H:%M:%S.%3N")

        # Create progress bar
        BAR_LENGTH=20
        FILLED=$((CURRENT_PROGRESS * BAR_LENGTH / 100))
        BAR=$(printf "%${FILLED}s" | tr ' ' '█')
        BAR="${BAR}$(printf "%$((BAR_LENGTH - FILLED))s" | tr ' ' '░')"

        echo "  [$TIMESTAMP] $BAR ${CURRENT_PROGRESS}% ($CURRENT_STATUS)"

        LAST_PROGRESS=$CURRENT_PROGRESS
        PROGRESS_UPDATES=$((PROGRESS_UPDATES + 1))
    fi

    # Check if complete
    if [ "$CURRENT_STATUS" = "complete" ]; then
        echo ""
        echo -e "${GREEN}✓ Request completed${NC}"
        echo ""
        echo "Progress Updates Captured: $PROGRESS_UPDATES"

        if [ "$PROGRESS_UPDATES" -ge 2 ]; then
            echo -e "${GREEN}PASSED: Multiple progress updates captured (worker processes quickly!)${NC}"
            exit 0
        else
            echo -e "${RED}FAILED: Expected >= 2 progress updates, got $PROGRESS_UPDATES${NC}"
            exit 1
        fi
    fi

    sleep 1
    ELAPSED=$((ELAPSED + 1))
done

echo ""
echo -e "${RED}FAILED: Timeout waiting for completion${NC}"
exit 1
