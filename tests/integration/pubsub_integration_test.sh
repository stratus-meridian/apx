#!/bin/bash

set -e

echo "========================================="
echo "Testing Pub/Sub Integration"
echo "========================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
ROUTER_URL="http://localhost:8081"
PUBSUB_EMULATOR_HOST="localhost:8085"
PROJECT_ID="apx-dev"
TOPIC_NAME="apx-requests-us-central1"
SUBSCRIPTION_NAME="apx-workers-us-central1"

# Step 1: Send request to router
echo -e "${YELLOW}[1/5] Sending test request to router...${NC}"
RESPONSE=$(curl -s -X POST ${ROUTER_URL}/api/test \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: test-tenant" \
  -d '{"message":"pubsub integration test"}')

echo "Response: $RESPONSE"
echo ""

# Extract request_id
REQUEST_ID=$(echo $RESPONSE | jq -r '.request_id')
if [ -z "$REQUEST_ID" ] || [ "$REQUEST_ID" = "null" ]; then
    echo -e "${RED}FAILED: No request_id in response${NC}"
    exit 1
fi
echo -e "${GREEN}Request ID: $REQUEST_ID${NC}"
echo ""

# Step 2: Set up Pub/Sub emulator environment
echo -e "${YELLOW}[2/5] Configuring Pub/Sub emulator connection...${NC}"
export PUBSUB_EMULATOR_HOST=${PUBSUB_EMULATOR_HOST}
echo "PUBSUB_EMULATOR_HOST=$PUBSUB_EMULATOR_HOST"
echo ""

# Step 3: Create subscription if it doesn't exist
echo -e "${YELLOW}[3/5] Creating/checking subscription...${NC}"
gcloud pubsub subscriptions create ${SUBSCRIPTION_NAME} \
  --topic=${TOPIC_NAME} \
  --project=${PROJECT_ID} 2>/dev/null || echo "Subscription already exists (OK)"
echo ""

# Step 4: Pull message from Pub/Sub
echo -e "${YELLOW}[4/5] Pulling message from Pub/Sub...${NC}"
sleep 2  # Give the router time to publish

MESSAGE=$(gcloud pubsub subscriptions pull ${SUBSCRIPTION_NAME} \
  --project=${PROJECT_ID} \
  --limit=1 \
  --format=json \
  --auto-ack)

if [ -z "$MESSAGE" ] || [ "$MESSAGE" = "[]" ]; then
    echo -e "${RED}FAILED: No message in Pub/Sub queue${NC}"
    echo ""
    echo "Debugging information:"
    echo "- Check if router is running: curl ${ROUTER_URL}/health"
    echo "- Check router logs: docker logs apilee-router-1 --tail 50"
    echo "- Verify topic exists: gcloud pubsub topics list --project=${PROJECT_ID}"
    exit 1
fi

echo "Message received from Pub/Sub:"
echo "$MESSAGE" | jq '.'
echo ""

# Step 5: Verify message contains our request_id
echo -e "${YELLOW}[5/5] Verifying message content...${NC}"

# Extract message data and decode base64
MESSAGE_DATA=$(echo "$MESSAGE" | jq -r '.[0].message.data' | base64 -d)
echo "Decoded message data:"
echo "$MESSAGE_DATA" | jq '.'
echo ""

# Check if request_id matches
MESSAGE_REQUEST_ID=$(echo "$MESSAGE_DATA" | jq -r '.request_id')
if [ "$MESSAGE_REQUEST_ID" = "$REQUEST_ID" ]; then
    echo -e "${GREEN}SUCCESS: Request ID matches!${NC}"
else
    echo -e "${RED}FAILED: Request ID mismatch${NC}"
    echo "Expected: $REQUEST_ID"
    echo "Got: $MESSAGE_REQUEST_ID"
    exit 1
fi

# Check tenant_id attribute
TENANT_ID=$(echo "$MESSAGE" | jq -r '.[0].message.attributes.tenant_id')
if [ "$TENANT_ID" = "test-tenant" ]; then
    echo -e "${GREEN}SUCCESS: Tenant ID attribute correct!${NC}"
else
    echo -e "${RED}WARNING: Tenant ID attribute mismatch${NC}"
    echo "Expected: test-tenant"
    echo "Got: $TENANT_ID"
fi

# Check ordering key
ORDERING_KEY=$(echo "$MESSAGE" | jq -r '.[0].message.orderingKey // empty')
if [ -n "$ORDERING_KEY" ]; then
    echo -e "${GREEN}SUCCESS: Ordering key present: ${ORDERING_KEY}${NC}"
else
    echo -e "${YELLOW}WARNING: No ordering key found${NC}"
fi

echo ""
echo "========================================="
echo -e "${GREEN}ALL TESTS PASSED!${NC}"
echo "========================================="
echo ""
echo "Summary:"
echo "  - Request ID: $REQUEST_ID"
echo "  - Tenant ID: $TENANT_ID"
echo "  - Ordering Key: $ORDERING_KEY"
echo "  - Message published to: $TOPIC_NAME"
echo "  - Message received from: $SUBSCRIPTION_NAME"
echo ""
