#!/bin/bash
# Quick test to verify monitoring integration works correctly

echo "Testing Monitoring Integration"
echo "=============================="
echo ""

# Test 1: Scripts work without monitoring
echo "Test 1: Running scripts without monitoring enabled..."
export DD_MONITORING_ENABLED=false

./query-apm.sh --help > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✓ query-apm.sh works without monitoring"
else
    echo "✗ query-apm.sh failed without monitoring"
fi

./search-logs.sh --help > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✓ search-logs.sh works without monitoring"
else
    echo "✗ search-logs.sh failed without monitoring"
fi

./query-slos.sh --help > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✓ query-slos.sh works without monitoring"
else
    echo "✗ query-slos.sh failed without monitoring"
fi

./query-watchdog.sh --help > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✓ query-watchdog.sh works without monitoring"
else
    echo "✗ query-watchdog.sh failed without monitoring (help doesn't exist, but script runs)"
fi

./query-metrics.sh --help > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✓ query-metrics.sh works without monitoring"
else
    echo "✗ query-metrics.sh failed without monitoring (help doesn't exist, but script runs)"
fi

./query-security-signals.sh --help > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✓ query-security-signals.sh works without monitoring"
else
    echo "✗ query-security-signals.sh failed without monitoring (help doesn't exist, but script runs)"
fi

echo ""

# Test 2: Library can be sourced
echo "Test 2: Checking monitoring library..."
if [ -f "lib/datadog-monitoring.sh" ]; then
    echo "✓ Monitoring library exists"
    source lib/datadog-monitoring.sh 2>/dev/null
    if [ $? -eq 0 ]; then
        echo "✓ Monitoring library can be sourced"
    else
        echo "✗ Monitoring library has syntax errors"
    fi
else
    echo "✗ Monitoring library not found"
fi

echo ""

# Test 3: Check for monitoring hooks in scripts
echo "Test 3: Verifying monitoring hooks are present..."

grep -q "init_monitoring" query-apm.sh && echo "✓ query-apm.sh has init_monitoring"
grep -q "start_operation" query-apm.sh && echo "✓ query-apm.sh has start_operation"
grep -q "send_metric" query-apm.sh && echo "✓ query-apm.sh has send_metric"

grep -q "init_monitoring" search-logs.sh && echo "✓ search-logs.sh has init_monitoring"
grep -q "start_operation" search-logs.sh && echo "✓ search-logs.sh has start_operation"
grep -q "send_metric" search-logs.sh && echo "✓ search-logs.sh has send_metric"

grep -q "init_monitoring" query-slos.sh && echo "✓ query-slos.sh has init_monitoring"
grep -q "start_operation" query-slos.sh && echo "✓ query-slos.sh has start_operation"
grep -q "send_metric" query-slos.sh && echo "✓ query-slos.sh has send_metric"

echo ""
echo "Test 4: Checking documentation..."
[ -f "../MONITORING_INTEGRATION.md" ] && echo "✓ MONITORING_INTEGRATION.md exists"
[ -f "../MONITORING_CHANGES_SUMMARY.md" ] && echo "✓ MONITORING_CHANGES_SUMMARY.md exists"
[ -f "MONITORING_QUICKSTART.md" ] && echo "✓ MONITORING_QUICKSTART.md exists"

echo ""
echo "=============================="
echo "Monitoring integration test complete!"
echo ""
echo "To enable monitoring, set:"
echo "  export DD_MONITORING_ENABLED=true"
echo "  export DD_API_KEY=your_api_key"
