#!/bin/bash

echo "================================================"
echo "Phase 2 Cloud Run Deployment Verification"
echo "================================================"
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

PASSED=0
FAILED=0

# Test 1: GCS Bucket Access
echo "Test 1: GCS Bucket Access..."
if gsutil ls gs://apx-build-478003-apx-artifacts > /dev/null 2>&1; then
  echo -e "${GREEN}✓ PASS${NC} - GCS bucket accessible"
  PASSED=$((PASSED+1))
else
  echo -e "${RED}✗ FAIL${NC} - GCS bucket not accessible"
  FAILED=$((FAILED+1))
fi

# Test 2: GCS Bucket Versioning
echo "Test 2: GCS Bucket Versioning..."
VERSIONING=$(gsutil versioning get gs://apx-build-478003-apx-artifacts | grep Enabled)
if [ -n "$VERSIONING" ]; then
  echo -e "${GREEN}✓ PASS${NC} - GCS bucket versioning enabled"
  PASSED=$((PASSED+1))
else
  echo -e "${RED}✗ FAIL${NC} - GCS bucket versioning not enabled"
  FAILED=$((FAILED+1))
fi

# Test 3: Firestore Database
echo "Test 3: Firestore Database..."
if gcloud firestore databases list --project=apx-build-478003 | grep -q "(default)"; then
  echo -e "${GREEN}✓ PASS${NC} - Firestore database accessible"
  PASSED=$((PASSED+1))
else
  echo -e "${RED}✗ FAIL${NC} - Firestore database not accessible"
  FAILED=$((FAILED+1))
fi

# Test 4: Cloud Build Trigger
echo "Test 4: Cloud Build Trigger..."
if gcloud builds triggers list --project=apx-build-478003 | grep -q "apx-policy-compiler"; then
  echo -e "${GREEN}✓ PASS${NC} - Cloud Build trigger exists"
  PASSED=$((PASSED+1))
else
  echo -e "${RED}✗ FAIL${NC} - Cloud Build trigger not found"
  FAILED=$((FAILED+1))
fi

# Test 5: Monitor Service
echo "Test 5: Monitor Service..."
MONITOR_STATUS=$(gcloud run services describe apx-monitor-dev --region=us-central1 --project=apx-build-478003 --format="value(status.conditions[0].status)" 2>&1)
if [ "$MONITOR_STATUS" == "True" ]; then
  echo -e "${GREEN}✓ PASS${NC} - Monitor service deployed and ready"
  PASSED=$((PASSED+1))
else
  echo -e "${RED}✗ FAIL${NC} - Monitor service not ready: $MONITOR_STATUS"
  FAILED=$((FAILED+1))
fi

# Test 6: Router Phase 2 Configuration
echo "Test 6: Router Phase 2 Configuration..."
ROUTER_ENV=$(gcloud run services describe apx-router-dev --region=us-central1 --project=apx-build-478003 --format="yaml" 2>&1 | grep -E "ENABLE_POLICY_VERSIONING|ENABLE_CANARY|GCS_ARTIFACTS_BUCKET")
if echo "$ROUTER_ENV" | grep -q "ENABLE_POLICY_VERSIONING"; then
  echo -e "${GREEN}✓ PASS${NC} - Router has Phase 2 environment variables"
  PASSED=$((PASSED+1))
else
  echo -e "${RED}✗ FAIL${NC} - Router missing Phase 2 environment variables"
  FAILED=$((FAILED+1))
fi

# Test 7: Workers Phase 2 Configuration
echo "Test 7: Workers Phase 2 Configuration..."
WORKER_ENV=$(gcloud run services describe apx-worker-cpu-dev --region=us-central1 --project=apx-build-478003 --format="yaml" 2>&1 | grep -E "GCS_ARTIFACTS_BUCKET|POLICY_CACHE_TTL|ENABLE_POLICY_CACHE")
if echo "$WORKER_ENV" | grep -q "GCS_ARTIFACTS_BUCKET"; then
  echo -e "${GREEN}✓ PASS${NC} - Workers have Phase 2 environment variables"
  PASSED=$((PASSED+1))
else
  echo -e "${RED}✗ FAIL${NC} - Workers missing Phase 2 environment variables"
  FAILED=$((FAILED+1))
fi

# Test 8: Service Accounts
echo "Test 8: Phase 2 Service Accounts..."
SA_COUNT=$(gcloud iam service-accounts list --filter="email:apx-compiler@ OR email:apx-cloudbuild@ OR email:apx-monitor@" --format="value(email)" | wc -l)
if [ "$SA_COUNT" -eq 3 ]; then
  echo -e "${GREEN}✓ PASS${NC} - All Phase 2 service accounts exist"
  PASSED=$((PASSED+1))
else
  echo -e "${RED}✗ FAIL${NC} - Missing Phase 2 service accounts (found $SA_COUNT, expected 3)"
  FAILED=$((FAILED+1))
fi

echo ""
echo "================================================"
echo "Verification Summary"
echo "================================================"
echo -e "Passed: ${GREEN}${PASSED}${NC}"
echo -e "Failed: ${RED}${FAILED}${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
  echo -e "${GREEN}✓ All tests passed!${NC}"
  exit 0
else
  echo -e "${RED}✗ Some tests failed${NC}"
  exit 1
fi
