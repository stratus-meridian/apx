#!/bin/bash
# ============================================================================
# APX CLI Build Script
# ============================================================================
# Builds the APX CLI tool for multiple platforms
#
# Usage:
#   ./scripts/build-cli.sh
#
# Outputs to: bin/
# ============================================================================

set -e  # Exit on error

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

# Directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
CLI_DIR="${ROOT_DIR}/.private/cli"
BIN_DIR="${ROOT_DIR}/bin"

echo -e "${BLUE}============================================================================${NC}"
echo -e "${BLUE}  APX CLI Build${NC}"
echo -e "${BLUE}============================================================================${NC}"
echo ""

# Check if CLI directory exists
if [ ! -d "$CLI_DIR" ]; then
  echo -e "${RED}Error: CLI directory not found at $CLI_DIR${NC}"
  exit 1
fi

# Create bin directory
mkdir -p "${BIN_DIR}"

echo -e "${BLUE}Building APX CLI for multiple platforms...${NC}"
echo ""

cd "${CLI_DIR}"

# Get version from git or default
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS="-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}"

# Linux AMD64
echo -e "  → ${BLUE}linux/amd64${NC}"
GOOS=linux GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o "${BIN_DIR}/apx-linux-amd64" main.go
echo -e "    ${GREEN}✓ Built${NC}"

# macOS AMD64
echo -e "  → ${BLUE}darwin/amd64${NC}"
GOOS=darwin GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o "${BIN_DIR}/apx-darwin-amd64" main.go
echo -e "    ${GREEN}✓ Built${NC}"

# macOS ARM64 (M1/M2)
echo -e "  → ${BLUE}darwin/arm64${NC}"
GOOS=darwin GOARCH=arm64 go build -ldflags "${LDFLAGS}" -o "${BIN_DIR}/apx-darwin-arm64" main.go
echo -e "    ${GREEN}✓ Built${NC}"

# Windows
echo -e "  → ${BLUE}windows/amd64${NC}"
GOOS=windows GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o "${BIN_DIR}/apx-windows-amd64.exe" main.go
echo -e "    ${GREEN}✓ Built${NC}"

# Local install (platform-specific)
echo -e "  → ${BLUE}local ($(uname -s)/$(uname -m))${NC}"
go build -ldflags "${LDFLAGS}" -o "${BIN_DIR}/apx" main.go
echo -e "    ${GREEN}✓ Built${NC}"

echo ""
echo -e "${GREEN}============================================================================${NC}"
echo -e "${GREEN}  ✅ CLI Built Successfully${NC}"
echo -e "${GREEN}============================================================================${NC}"
echo ""
echo "Binaries:"
ls -lh "${BIN_DIR}"/apx* | awk '{print "  " $9 " (" $5 ")"}'
echo ""
echo "To install locally:"
echo -e "  ${BLUE}sudo cp ${BIN_DIR}/apx /usr/local/bin/${NC}"
echo ""
echo "To test:"
echo -e "  ${BLUE}${BIN_DIR}/apx --version${NC}"
echo ""
