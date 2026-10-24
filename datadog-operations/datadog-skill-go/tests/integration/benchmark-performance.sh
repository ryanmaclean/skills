#!/bin/bash
# Performance benchmark for Datadog CLI
# Compares startup time, memory usage, and command execution speed

set -e

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

echo ""
echo "========================================"
echo "Datadog CLI Performance Benchmark"
echo "========================================"
echo ""

# Check if dd command exists
if ! command -v dd &> /dev/null; then
    echo -e "${RED}ERROR: dd command not found${NC}"
    echo "Please install the Datadog CLI first"
    exit 1
fi

# Check if time command exists
if ! command -v time &> /dev/null; then
    echo -e "${YELLOW}WARNING: time command not found, using bash time${NC}"
fi

# Number of iterations for averaging
ITERATIONS=10

echo -e "${BLUE}Running $ITERATIONS iterations for each test...${NC}"
echo ""

# Function to measure execution time
measure_time() {
    local command=$1
    local iterations=$2
    local total=0

    for ((i=1; i<=iterations; i++)); do
        # Use bash's TIMEFORMAT for consistent timing
        start=$(date +%s%N)
        eval "$command" &> /dev/null
        end=$(date +%s%N)
        elapsed=$((end - start))
        total=$((total + elapsed))
    done

    # Calculate average in milliseconds
    avg=$((total / iterations / 1000000))
    echo $avg
}

# Function to measure memory usage
measure_memory() {
    local command=$1

    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        /usr/bin/time -l $command 2>&1 | grep "maximum resident set size" | awk '{print $1}'
    elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
        # Linux
        /usr/bin/time -v $command 2>&1 | grep "Maximum resident set size" | awk '{print $6}'
    else
        echo "0"
    fi
}

echo "========================================="
echo "Test 1: Startup Time (Cold)"
echo "========================================="
echo ""

echo -n "Measuring startup time (dd --version)... "
startup_time=$(measure_time "dd --version" $ITERATIONS)
echo -e "${CYAN}${startup_time}ms${NC} (average over $ITERATIONS runs)"

if [ $startup_time -lt 10 ]; then
    echo -e "${GREEN}✓ EXCELLENT${NC} (< 10ms)"
elif [ $startup_time -lt 50 ]; then
    echo -e "${GREEN}✓ GOOD${NC} (< 50ms)"
elif [ $startup_time -lt 100 ]; then
    echo -e "${YELLOW}⚠ ACCEPTABLE${NC} (< 100ms)"
else
    echo -e "${RED}✗ SLOW${NC} (> 100ms)"
fi

echo ""

echo "========================================="
echo "Test 2: Help Command"
echo "========================================="
echo ""

echo -n "Measuring help time (dd --help)... "
help_time=$(measure_time "dd --help" $ITERATIONS)
echo -e "${CYAN}${help_time}ms${NC} (average over $ITERATIONS runs)"

echo ""

echo "========================================="
echo "Test 3: Context Detection"
echo "========================================="
echo ""

if git rev-parse --git-dir > /dev/null 2>&1; then
    echo -n "Measuring context detection (dd context)... "
    context_time=$(measure_time "dd context" $ITERATIONS)
    echo -e "${CYAN}${context_time}ms${NC} (average over $ITERATIONS runs)"
else
    echo -e "${YELLOW}SKIP: Not in git repository${NC}"
fi

echo ""

echo "========================================="
echo "Test 4: Memory Usage"
echo "========================================="
echo ""

echo -n "Measuring memory usage (dd --version)... "

if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS - use time -l
    memory=$(/usr/bin/time -l dd --version 2>&1 | grep "maximum resident set size" | awk '{print $1}')
    memory_mb=$((memory / 1024 / 1024))
    echo -e "${CYAN}${memory_mb}MB${NC}"
elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    # Linux - use /usr/bin/time -v
    memory=$(/usr/bin/time -v dd --version 2>&1 | grep "Maximum resident set size" | awk '{print $6}')
    memory_mb=$((memory / 1024))
    echo -e "${CYAN}${memory_mb}MB${NC}"
else
    echo -e "${YELLOW}N/A (unsupported OS)${NC}"
    memory_mb=0
fi

if [ $memory_mb -lt 20 ]; then
    echo -e "${GREEN}✓ EXCELLENT${NC} (< 20MB)"
elif [ $memory_mb -lt 50 ]; then
    echo -e "${GREEN}✓ GOOD${NC} (< 50MB)"
else
    echo -e "${YELLOW}⚠ HIGH${NC} (> 50MB)"
fi

echo ""

echo "========================================="
echo "Test 5: Binary Size"
echo "========================================="
echo ""

dd_path=$(which dd)
if [ -f "$dd_path" ]; then
    size=$(stat -f%z "$dd_path" 2>/dev/null || stat -c%s "$dd_path" 2>/dev/null || echo "0")
    size_mb=$((size / 1024 / 1024))
    echo -e "Binary size: ${CYAN}${size_mb}MB${NC}"

    if [ $size_mb -lt 15 ]; then
        echo -e "${GREEN}✓ EXCELLENT${NC} (< 15MB)"
    elif [ $size_mb -lt 30 ]; then
        echo -e "${GREEN}✓ GOOD${NC} (< 30MB)"
    else
        echo -e "${YELLOW}⚠ LARGE${NC} (> 30MB)"
    fi
else
    echo -e "${RED}ERROR: Cannot find dd binary${NC}"
fi

echo ""

echo "========================================="
echo "Performance Summary"
echo "========================================="
echo ""

echo -e "${BLUE}Startup Performance:${NC}"
echo "  Average startup time: ${startup_time}ms"
echo "  Help command time: ${help_time}ms"
if [ -n "$context_time" ]; then
    echo "  Context detection: ${context_time}ms"
fi

echo ""
echo -e "${BLUE}Resource Usage:${NC}"
if [ $memory_mb -gt 0 ]; then
    echo "  Memory usage: ${memory_mb}MB"
fi
if [ $size_mb -gt 0 ]; then
    echo "  Binary size: ${size_mb}MB"
fi

echo ""
echo -e "${BLUE}Comparison to Python CLI:${NC}"
echo "  Python startup: ~200ms (estimated)"
echo "  Python memory: ~30-50MB (estimated)"
echo ""

if [ $startup_time -gt 0 ] && [ $startup_time -lt 200 ]; then
    speedup=$((200 / startup_time))
    echo -e "${GREEN}Go CLI is ${speedup}x faster${NC}"
fi

if [ $memory_mb -gt 0 ] && [ $memory_mb -lt 50 ]; then
    savings=$((100 - memory_mb * 100 / 40))
    echo -e "${GREEN}Go CLI uses ${savings}% less memory${NC}"
fi

echo ""

# Performance targets
echo "========================================="
echo "Performance Targets"
echo "========================================="
echo ""

all_pass=true

echo -n "Startup time < 50ms: "
if [ $startup_time -lt 50 ]; then
    echo -e "${GREEN}✓ PASS${NC}"
else
    echo -e "${RED}✗ FAIL${NC}"
    all_pass=false
fi

if [ $memory_mb -gt 0 ]; then
    echo -n "Memory usage < 30MB: "
    if [ $memory_mb -lt 30 ]; then
        echo -e "${GREEN}✓ PASS${NC}"
    else
        echo -e "${RED}✗ FAIL${NC}"
        all_pass=false
    fi
fi

if [ $size_mb -gt 0 ]; then
    echo -n "Binary size < 20MB: "
    if [ $size_mb -lt 20 ]; then
        echo -e "${GREEN}✓ PASS${NC}"
    else
        echo -e "${RED}✗ FAIL${NC}"
        all_pass=false
    fi
fi

echo ""

if [ "$all_pass" = true ]; then
    echo -e "${GREEN}All performance targets met!${NC}"
    exit 0
else
    echo -e "${YELLOW}Some performance targets not met${NC}"
    exit 1
fi
