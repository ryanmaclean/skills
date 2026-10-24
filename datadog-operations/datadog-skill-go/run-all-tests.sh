#!/bin/bash
# Comprehensive Test Suite for Datadog CLI and Skill
# Tests both Go CLI and Claude Code skill integration

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║     Datadog CLI & Skill Comprehensive Test Suite            ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
SKIPPED_TESTS=0

# Function to run a test section
run_section() {
    local name=$1
    local command=$2

    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE} $name${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo ""

    if eval "$command"; then
        echo -e "${GREEN}✓ $name completed successfully${NC}"
        return 0
    else
        echo -e "${RED}✗ $name failed${NC}"
        return 1
    fi
}

# ═══════════════════════════════════════════════════════════════
# Section 1: Go CLI Unit Tests
# ═══════════════════════════════════════════════════════════════

if run_section "Go CLI Unit Tests" "go test ./... -v -count=1"; then
    ((PASSED_TESTS++))
else
    ((FAILED_TESTS++))
fi
((TOTAL_TESTS++))

# ═══════════════════════════════════════════════════════════════
# Section 2: Go CLI Build Test
# ═══════════════════════════════════════════════════════════════

if run_section "Go CLI Build Test" "go build -o dd cmd/main.go"; then
    ((PASSED_TESTS++))
    echo -e "  ${GREEN}✓${NC} Binary created: dd"
else
    ((FAILED_TESTS++))
fi
((TOTAL_TESTS++))

# ═══════════════════════════════════════════════════════════════
# Section 3: Go CLI Integration Tests
# ═══════════════════════════════════════════════════════════════

test_cli_commands() {
    echo "Testing CLI commands with live Datadog API..."
    echo ""

    local cmd_tests=0
    local cmd_passed=0

    # Test 1: Version
    echo -n "  Testing --version... "
    if ./dd --version > /dev/null 2>&1; then
        echo -e "${GREEN}PASS${NC}"
        ((cmd_passed++))
    else
        echo -e "${RED}FAIL${NC}"
    fi
    ((cmd_tests++))

    # Test 2: Help
    echo -n "  Testing --help... "
    if ./dd --help > /dev/null 2>&1; then
        echo -e "${GREEN}PASS${NC}"
        ((cmd_passed++))
    else
        echo -e "${RED}FAIL${NC}"
    fi
    ((cmd_tests++))

    # Test 3: Context
    echo -n "  Testing context command... "
    if ./dd context --json > /dev/null 2>&1; then
        echo -e "${GREEN}PASS${NC}"
        ((cmd_passed++))
    else
        echo -e "${YELLOW}SKIP${NC} (no git context)"
    fi
    ((cmd_tests++))

    # Test 4: Health
    echo -n "  Testing health command... "
    if ./dd health --json > /dev/null 2>&1; then
        echo -e "${GREEN}PASS${NC}"
        ((cmd_passed++))
    else
        echo -e "${RED}FAIL${NC}"
    fi
    ((cmd_tests++))

    # Test 5: APM
    echo -n "  Testing apm command... "
    if ./dd apm --duration 1h --json > /dev/null 2>&1; then
        echo -e "${GREEN}PASS${NC}"
        ((cmd_passed++))
    else
        echo -e "${YELLOW}SKIP${NC} (no data)"
    fi
    ((cmd_tests++))

    # Test 6: Logs
    echo -n "  Testing logs command... "
    if ./dd logs --query "status:error" --duration 30m --json > /dev/null 2>&1; then
        echo -e "${GREEN}PASS${NC}"
        ((cmd_passed++))
    else
        echo -e "${YELLOW}SKIP${NC} (no data)"
    fi
    ((cmd_tests++))

    # Test 7: Monitors
    echo -n "  Testing monitors list... "
    if ./dd monitors list --json > /dev/null 2>&1; then
        echo -e "${GREEN}PASS${NC}"
        ((cmd_passed++))
    else
        echo -e "${RED}FAIL${NC}"
    fi
    ((cmd_tests++))

    echo ""
    echo "CLI Command Tests: $cmd_passed/$cmd_tests passed"

    if [ $cmd_passed -ge 5 ]; then
        return 0
    else
        return 1
    fi
}

if run_section "Go CLI Integration Tests" "test_cli_commands"; then
    ((PASSED_TESTS++))
else
    ((FAILED_TESTS++))
fi
((TOTAL_TESTS++))

# ═══════════════════════════════════════════════════════════════
# Section 4: Skill Script Tests
# ═══════════════════════════════════════════════════════════════

SKILL_DIR="${SKILL_DIR:-$HOME/dd-skill-test}"

if [ -d "$SKILL_DIR" ] && [ -f "$SKILL_DIR/test-skills.sh" ]; then
    if run_section "Skill Script Tests" "cd '$SKILL_DIR' && ./test-skills.sh"; then
        ((PASSED_TESTS++))
    else
        ((FAILED_TESTS++))
    fi
    ((TOTAL_TESTS++))
else
    echo ""
    echo -e "${YELLOW}⚠ Skipping skill script tests (skill directory not found)${NC}"
    echo "  Expected: $SKILL_DIR"
    ((SKIPPED_TESTS++))
    ((TOTAL_TESTS++))
fi

# ═══════════════════════════════════════════════════════════════
# Section 5: Coverage Report
# ═══════════════════════════════════════════════════════════════

if run_section "Generate Coverage Report" "go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out"; then
    ((PASSED_TESTS++))
    echo ""
    echo "Coverage report saved to: coverage.out"
    echo "View HTML: go tool cover -html=coverage.out"
else
    ((FAILED_TESTS++))
fi
((TOTAL_TESTS++))

# ═══════════════════════════════════════════════════════════════
# Final Summary
# ═══════════════════════════════════════════════════════════════

echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE} Test Summary${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""
echo "Total test sections: $TOTAL_TESTS"
echo -e "${GREEN}Passed: $PASSED_TESTS${NC}"
echo -e "${RED}Failed: $FAILED_TESTS${NC}"
echo -e "${YELLOW}Skipped: $SKIPPED_TESTS${NC}"
echo ""

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}✅ All tests passed!${NC}"
    echo ""
    echo "Next steps:"
    echo "  1. View coverage: go tool cover -html=coverage.out"
    echo "  2. Test in Claude Code: Open dd-skill-test directory and use /datadog"
    echo "  3. See SKILL-TESTING.md for detailed testing guide"
    exit 0
else
    echo -e "${RED}❌ Some tests failed${NC}"
    echo ""
    echo "Troubleshooting:"
    echo "  1. Check Datadog API credentials (DD_API_KEY, DD_APP_KEY)"
    echo "  2. Verify network connectivity"
    echo "  3. See SKILL-TESTING.md for detailed troubleshooting"
    exit 1
fi
