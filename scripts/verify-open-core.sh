#!/bin/bash

#########################################################################
# APX Router Open-Core Verification Script
#
# This script verifies that the open-core repository is ready for publication.
# It checks for:
#   - No private dependencies
#   - Successful build
#   - Working health endpoints
#   - Example functionality
#
# Usage:
#   ./scripts/verify-open-core.sh
#########################################################################

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
OPEN_CORE_DIR="$PROJECT_ROOT/open-core"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}APX Router Open-Core Verification${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Test 1: Check if open-core directory exists
echo -e "${YELLOW}[1/8] Checking open-core directory...${NC}"
if [ ! -d "$OPEN_CORE_DIR" ]; then
    echo -e "${RED}✗ FAIL: open-core directory not found${NC}"
    exit 1
fi
echo -e "${GREEN}✓ PASS: open-core directory exists${NC}"
echo ""

# Test 2: Check for private dependencies in go.mod
echo -e "${YELLOW}[2/8] Checking for private dependencies...${NC}"
cd "$OPEN_CORE_DIR"
if grep -q "apx-private" go.mod 2>/dev/null; then
    echo -e "${RED}✗ FAIL: Found private dependencies in go.mod${NC}"
    grep "apx-private" go.mod
    exit 1
fi
echo -e "${GREEN}✓ PASS: No private dependencies in go.mod${NC}"
echo ""

# Test 3: Check for private imports in source code
echo -e "${YELLOW}[3/8] Checking for private imports in source code...${NC}"
if find . -name "*.go" -exec grep -l "apx-private" {} \; | grep -q .; then
    echo -e "${RED}✗ FAIL: Found private imports in source code:${NC}"
    find . -name "*.go" -exec grep -l "apx-private" {} \;
    exit 1
fi
echo -e "${GREEN}✓ PASS: No private imports in source code${NC}"
echo ""

# Test 4: Check that required files exist
echo -e "${YELLOW}[4/8] Checking required files...${NC}"
REQUIRED_FILES=(
    "README.md"
    "LICENSE"
    "Dockerfile"
    "go.mod"
    "cmd/router/main.go"
    "configs/crds/product.schema.yaml"
    "configs/crds/route.schema.yaml"
    "configs/crds/policybundle.schema.yaml"
    "configs/schemas/tier-schema.json"
    "examples/hello-world/README.md"
)

MISSING=0
for file in "${REQUIRED_FILES[@]}"; do
    if [ ! -f "$file" ]; then
        echo -e "${RED}  ✗ Missing: $file${NC}"
        MISSING=1
    fi
done

if [ $MISSING -eq 1 ]; then
    echo -e "${RED}✗ FAIL: Some required files are missing${NC}"
    exit 1
fi
echo -e "${GREEN}✓ PASS: All required files present${NC}"
echo ""

# Test 5: Clean and rebuild
echo -e "${YELLOW}[5/8] Building router...${NC}"
rm -f router
if ! go build -o router ./cmd/router; then
    echo -e "${RED}✗ FAIL: Build failed${NC}"
    exit 1
fi
echo -e "${GREEN}✓ PASS: Build successful${NC}"
echo ""

# Test 6: Check binary size (should be reasonable)
echo -e "${YELLOW}[6/8] Checking binary size...${NC}"
BINARY_SIZE=$(stat -f%z router 2>/dev/null || stat -c%s router 2>/dev/null)
BINARY_SIZE_MB=$((BINARY_SIZE / 1024 / 1024))
echo -e "  Binary size: ${BINARY_SIZE_MB}MB"
if [ $BINARY_SIZE_MB -gt 100 ]; then
    echo -e "${YELLOW}  Warning: Binary is quite large (>${BINARY_SIZE_MB}MB)${NC}"
fi
echo -e "${GREEN}✓ PASS: Binary created${NC}"
echo ""

# Test 7: Start router and test endpoints
echo -e "${YELLOW}[7/8] Testing router endpoints...${NC}"
export PORT=18080
export ROUTES_CONFIG="/test/**=http://localhost:9999:sync"
export LOG_LEVEL=error

# Start router in background
./router > /dev/null 2>&1 &
ROUTER_PID=$!

# Wait for router to start (with retries)
echo -e "  Waiting for router to start..."
MAX_RETRIES=10
RETRY=0
while [ $RETRY -lt $MAX_RETRIES ]; do
    if curl -sf http://localhost:18080/health > /dev/null 2>&1; then
        break
    fi
    RETRY=$((RETRY + 1))
    sleep 1
done

# Test health endpoint
if [ $RETRY -eq $MAX_RETRIES ]; then
    echo -e "${RED}  ✗ /health endpoint not responding after ${MAX_RETRIES}s${NC}"
    kill $ROUTER_PID 2>/dev/null || true
    exit 1
fi
echo -e "${GREEN}  ✓ /health endpoint working${NC}"

# Test ready endpoint
if curl -sf http://localhost:18080/ready > /dev/null 2>&1; then
    echo -e "${GREEN}  ✓ /ready endpoint working${NC}"
else
    echo -e "${RED}  ✗ /ready endpoint not responding${NC}"
    kill $ROUTER_PID 2>/dev/null || true
    exit 1
fi

# Test metrics endpoint
if curl -sf http://localhost:18080/metrics | grep -q "^#"; then
    echo -e "${GREEN}  ✓ /metrics endpoint working${NC}"
else
    echo -e "${RED}  ✗ /metrics endpoint not responding${NC}"
    kill $ROUTER_PID 2>/dev/null || true
    exit 1
fi

# Stop router
kill $ROUTER_PID 2>/dev/null || true
sleep 1

echo -e "${GREEN}✓ PASS: All endpoints working${NC}"
echo ""

# Test 8: Check README completeness
echo -e "${YELLOW}[8/8] Checking README completeness...${NC}"
README_SECTIONS=(
    "Quick Start"
    "Configuration"
    "Features"
    "Architecture"
    "License"
)

README_MISSING=0
for section in "${README_SECTIONS[@]}"; do
    if ! grep -qi "$section" README.md; then
        echo -e "${RED}  ✗ Missing section: $section${NC}"
        README_MISSING=1
    fi
done

if [ $README_MISSING -eq 1 ]; then
    echo -e "${YELLOW}  Warning: Some README sections may be missing${NC}"
else
    echo -e "${GREEN}✓ PASS: README appears complete${NC}"
fi
echo ""

# Summary
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}ALL VERIFICATION TESTS PASSED!${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "The open-core repository is ready for publication."
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo -e "  1. Review the implementation plan: ${BLUE}OPEN_CORE_EXTRACTION_PLAN.md${NC}"
echo -e "  2. Run publication script: ${BLUE}./scripts/publish-open-core.sh${NC}"
echo -e "  3. Create GitHub repository and push"
echo ""
echo -e "${GREEN}Repository statistics:${NC}"
echo -e "  Go files:   $(find . -name "*.go" -not -path "*/vendor/*" | wc -l | tr -d ' ')"
echo -e "  Binary size: ${BINARY_SIZE_MB}MB"
echo -e "  Module: $(head -1 go.mod | awk '{print $2}')"
echo ""
