#!/bin/bash
set -e

# CI/CD Verification Script
# Validates all GitHub Actions workflows and CI/CD configuration

echo "================================================================================"
echo "                 CI/CD Configuration Verification"
echo "================================================================================"
echo ""

ERRORS=0

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

print_status() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✅ $2${NC}"
    else
        echo -e "${RED}❌ $2${NC}"
        ERRORS=$((ERRORS + 1))
    fi
}

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    echo -e "${RED}Error: Must be run from project root${NC}"
    exit 1
fi

echo "1. Checking GitHub Actions workflow files..."
echo "   ---------------------------------------"

WORKFLOWS=(
    ".github/workflows/ci.yml"
    ".github/workflows/build.yml"
    ".github/workflows/release.yml"
    ".github/workflows/coverage.yml"
    ".github/workflows/docker.yml"
    ".github/workflows/validate.yml"
)

for workflow in "${WORKFLOWS[@]}"; do
    if [ -f "$workflow" ]; then
        print_status 0 "Found: $workflow"
    else
        print_status 1 "Missing: $workflow"
    fi
done

echo ""
echo "2. Checking supporting files..."
echo "   ---------------------------"

SUPPORT_FILES=(
    "Dockerfile"
    ".dockerignore"
    ".golangci.yml"
    "Makefile"
    "CONTRIBUTING.md"
    ".github/dependabot.yml"
    ".github/CODEOWNERS"
)

for file in "${SUPPORT_FILES[@]}"; do
    if [ -f "$file" ]; then
        print_status 0 "Found: $file"
    else
        print_status 1 "Missing: $file"
    fi
done

echo ""
echo "3. Checking documentation..."
echo "   ------------------------"

DOCS=(
    ".github/workflows/README.md"
    ".github/CI-CD-GUIDE.md"
    ".github/ARCHITECTURE.md"
    "CI-CD-IMPLEMENTATION.md"
)

for doc in "${DOCS[@]}"; do
    if [ -f "$doc" ]; then
        print_status 0 "Found: $doc"
    else
        print_status 1 "Missing: $doc"
    fi
done

echo ""
echo "4. Checking issue templates..."
echo "   -------------------------"

TEMPLATES=(
    ".github/ISSUE_TEMPLATE/bug_report.md"
    ".github/ISSUE_TEMPLATE/feature_request.md"
    ".github/ISSUE_TEMPLATE/config.yml"
    ".github/pull_request_template.md"
)

for template in "${TEMPLATES[@]}"; do
    if [ -f "$template" ]; then
        print_status 0 "Found: $template"
    else
        print_status 1 "Missing: $template"
    fi
done

echo ""
echo "5. Validating Makefile targets..."
echo "   -----------------------------"

MAKE_TARGETS=(
    "build"
    "test"
    "lint"
    "fmt"
    "clean"
    "ci"
    "docker"
)

for target in "${MAKE_TARGETS[@]}"; do
    if make -n "$target" &>/dev/null; then
        print_status 0 "Makefile target: $target"
    else
        print_status 1 "Makefile target: $target"
    fi
done

echo ""
echo "6. Checking Go configuration..."
echo "   --------------------------"

# Check if go.mod exists
if [ -f "go.mod" ]; then
    print_status 0 "go.mod exists"
else
    print_status 1 "go.mod missing"
fi

# Check Go version
if command -v go &> /dev/null; then
    GO_VERSION=$(go version | awk '{print $3}')
    print_status 0 "Go installed: $GO_VERSION"
else
    print_status 1 "Go not installed"
fi

# Verify Go modules
if go mod verify &>/dev/null; then
    print_status 0 "Go modules verified"
else
    print_status 1 "Go module verification failed"
fi

echo ""
echo "7. Checking Docker configuration..."
echo "   -------------------------------"

if [ -f "Dockerfile" ]; then
    # Check for multi-stage build
    if grep -q "FROM.*AS builder" Dockerfile; then
        print_status 0 "Dockerfile uses multi-stage build"
    else
        print_status 1 "Dockerfile missing multi-stage build"
    fi

    # Check for FROM scratch
    if grep -q "FROM scratch" Dockerfile; then
        print_status 0 "Dockerfile uses minimal base image"
    else
        print_status 1 "Dockerfile not using FROM scratch"
    fi
fi

echo ""
echo "8. Security checks..."
echo "   ----------------"

# Check workflows for command injection patterns
UNSAFE_PATTERNS=0
for workflow in "${WORKFLOWS[@]}"; do
    if [ -f "$workflow" ]; then
        # Look for unsafe patterns like ${{ github.event.issue.title }} in run: commands
        # Exclude safe uses in action inputs (like metadata-action)
        while IFS= read -r line; do
            # Skip if this is in a 'with:' block (action inputs are safe)
            if echo "$line" | grep -E '^\s+(with|env):' &>/dev/null; then
                continue
            fi
            # Check for github.event in run commands
            if echo "$line" | grep -E '^\s+run:.*\$\{\{.*github\.event\.' &>/dev/null; then
                echo -e "${YELLOW}⚠️  Warning: Potential unsafe input in $workflow${NC}"
                echo "    Line: $line"
                UNSAFE_PATTERNS=$((UNSAFE_PATTERNS + 1))
            fi
        done < "$workflow"
    fi
done

if [ $UNSAFE_PATTERNS -eq 0 ]; then
    print_status 0 "No unsafe input patterns detected"
else
    print_status 1 "Found $UNSAFE_PATTERNS potential unsafe patterns in run commands"
fi

# Check for pinned action versions
UNPINNED=0
for workflow in "${WORKFLOWS[@]}"; do
    if [ -f "$workflow" ]; then
        if grep -E 'uses:.*@(main|master)' "$workflow" &>/dev/null; then
            UNPINNED=$((UNPINNED + 1))
        fi
    fi
done

if [ $UNPINNED -eq 0 ]; then
    print_status 0 "All actions use pinned versions"
else
    print_status 1 "Found $UNPINNED workflows with unpinned actions"
fi

echo ""
echo "9. Testing build..."
echo "   --------------"

# Try to build
if go build -o /tmp/dd-verify ./cmd &>/dev/null; then
    print_status 0 "Project builds successfully"
    rm -f /tmp/dd-verify
else
    print_status 1 "Build failed"
fi

# Test with version flags
if go build -ldflags="-X main.version=test -X main.commit=abc123 -X main.buildDate=2026-01-21" -o /tmp/dd-verify ./cmd &>/dev/null; then
    # Check if version info is injected
    if /tmp/dd-verify version | grep -q "test"; then
        print_status 0 "Version injection working"
    else
        print_status 1 "Version injection not working"
    fi
    rm -f /tmp/dd-verify
else
    print_status 1 "Build with version flags failed"
fi

echo ""
echo "10. Summary..."
echo "    --------"

echo ""
echo "Total files created: $(find .github -type f | wc -l | tr -d ' ')"
echo "Workflow files: ${#WORKFLOWS[@]}"
echo "Documentation files: ${#DOCS[@]}"
echo ""

if [ $ERRORS -eq 0 ]; then
    echo -e "${GREEN}================================================================================"
    echo "                 ✅ All Checks Passed!"
    echo "================================================================================${NC}"
    echo ""
    echo "Next steps:"
    echo "  1. Review the workflows in .github/workflows/"
    echo "  2. Customize CODEOWNERS team names"
    echo "  3. Configure Codecov token (optional)"
    echo "  4. Push to GitHub to trigger workflows"
    echo "  5. Create a release tag: git tag v0.1.0 && git push origin v0.1.0"
    echo ""
    exit 0
else
    echo -e "${RED}================================================================================"
    echo "                 ❌ $ERRORS Checks Failed"
    echo "================================================================================${NC}"
    echo ""
    echo "Please review the errors above and fix them before proceeding."
    echo ""
    exit 1
fi
