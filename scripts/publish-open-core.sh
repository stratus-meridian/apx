#!/bin/bash

#########################################################################
# APX Router Open-Core Publication Script
#
# This script prepares the open-core repository for publication to GitHub.
# It copies the open-core directory to a new location and initializes
# a Git repository ready for pushing to GitHub.
#
# Usage:
#   ./scripts/publish-open-core.sh [destination_directory]
#
# If no destination is provided, creates ../apx-router-open-core/
#########################################################################

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
OPEN_CORE_DIR="$PROJECT_ROOT/open-core"

# Destination directory (default: ../apx-router-open-core)
DEST_DIR="${1:-$(dirname "$PROJECT_ROOT")/apx-router-open-core}"

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}APX Router Open-Core Publication${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# Verify open-core directory exists
if [ ! -d "$OPEN_CORE_DIR" ]; then
    echo -e "${RED}Error: open-core directory not found at $OPEN_CORE_DIR${NC}"
    exit 1
fi

# Check if destination exists
if [ -d "$DEST_DIR" ]; then
    echo -e "${YELLOW}Warning: Destination directory already exists: $DEST_DIR${NC}"
    read -p "Do you want to remove it and continue? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Aborted."
        exit 1
    fi
    rm -rf "$DEST_DIR"
fi

# Create destination directory
echo -e "${GREEN}Creating destination directory: $DEST_DIR${NC}"
mkdir -p "$DEST_DIR"

# Copy open-core files
echo -e "${GREEN}Copying open-core files...${NC}"
cp -r "$OPEN_CORE_DIR"/* "$DEST_DIR/"

# Remove build artifacts
echo -e "${GREEN}Cleaning build artifacts...${NC}"
rm -f "$DEST_DIR/router"
find "$DEST_DIR" -type f -name "*.test" -delete
find "$DEST_DIR" -type f -name "*.out" -delete

# Initialize Git repository
cd "$DEST_DIR"
echo -e "${GREEN}Initializing Git repository...${NC}"
git init

# Create .gitignore
echo -e "${GREEN}Creating .gitignore...${NC}"
cat > .gitignore <<'EOF'
# Binaries
router
*.exe
*.dll
*.so
*.dylib

# Test binaries
*.test
*.out

# Coverage files
*.cov
coverage.txt
coverage.html

# Build artifacts
dist/
build/

# IDE files
.vscode/
.idea/
*.swp
*.swo
*~

# OS files
.DS_Store
Thumbs.db

# Dependency directories
vendor/

# Go workspace file
go.work
go.work.sum

# Environment files
.env
.env.local
*.pem
*.key
EOF

# Add all files
echo -e "${GREEN}Staging files...${NC}"
git add .

# Create initial commit
echo -e "${GREEN}Creating initial commit...${NC}"
git commit -m "Initial commit: APX Router Open-Core Edition v0.1.0

This is the open-core edition of APX Router, a high-performance API gateway
with policy-based routing, rate limiting, and multi-tenancy support.

Features:
- Sync & Async routing (HTTP proxy + Pub/Sub)
- In-memory rate limiting
- Simple tenant resolution
- OpenTelemetry observability
- Prometheus metrics
- CRD-based configuration

For commercial features (billing, advanced analytics, enterprise support),
visit https://apilee.io
"

# Create tag
echo -e "${GREEN}Creating v0.1.0 tag...${NC}"
git tag -a v0.1.0 -m "Initial open-core release"

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Publication Preparation Complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "Repository location: ${YELLOW}$DEST_DIR${NC}"
echo ""
echo -e "Next steps:"
echo -e "  1. Create GitHub repository: ${YELLOW}apx-router-open-core${NC}"
echo -e "  2. Add remote:"
echo -e "     ${YELLOW}cd $DEST_DIR${NC}"
echo -e "     ${YELLOW}git remote add origin git@github.com:stratus-meridian/apx-router-open-core.git${NC}"
echo -e "  3. Push to GitHub:"
echo -e "     ${YELLOW}git push -u origin master${NC}"
echo -e "     ${YELLOW}git push --tags${NC}"
echo ""
echo -e "${GREEN}Verification checklist:${NC}"
echo -e "  ✓ Build succeeds: ${YELLOW}cd $DEST_DIR && go build ./cmd/router${NC}"
echo -e "  ✓ No private dependencies: ${YELLOW}grep -r apx-private go.mod${NC} (should be empty)"
echo -e "  ✓ README is complete"
echo -e "  ✓ LICENSE is present"
echo -e "  ✓ Examples work"
echo ""
