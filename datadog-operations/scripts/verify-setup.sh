#!/bin/bash
set -e

# Datadog Setup Verification Script
# Checks if Datadog agent is reachable and configuration is valid

echo "Datadog Setup Verification" >&2
echo "=============================" >&2
echo "" >&2

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to check HTTP endpoint
check_endpoint() {
    local url=$1
    local name=$2

    if command_exists curl; then
        if curl -s -f -m 5 "$url" >/dev/null 2>&1; then
            echo "[OK] $name is reachable" >&2
            return 0
        else
            echo "[FAIL] $name is not reachable" >&2
            return 1
        fi
    else
        echo "[WARN] curl not found, cannot check $name" >&2
        return 2
    fi
}

# Initialize results
results=()
warnings=()
errors=()

# Check environment variables
echo "Checking environment variables..." >&2
if [ -n "$DD_API_KEY" ]; then
    echo "[OK] DD_API_KEY is set" >&2
    results+=("DD_API_KEY: configured")
else
    echo "[FAIL] DD_API_KEY is not set" >&2
    errors+=("DD_API_KEY: missing")
fi

if [ -n "$DD_SITE" ]; then
    echo "[OK] DD_SITE is set ($DD_SITE)" >&2
    results+=("DD_SITE: $DD_SITE")
else
    echo "[WARN] DD_SITE not set, defaulting to datadoghq.com" >&2
    warnings+=("DD_SITE: not set (using default)")
    DD_SITE="datadoghq.com"
fi

if [ -n "$DD_SERVICE" ]; then
    echo "[OK] DD_SERVICE is set ($DD_SERVICE)" >&2
    results+=("DD_SERVICE: $DD_SERVICE")
else
    echo "[WARN] DD_SERVICE not set" >&2
    warnings+=("DD_SERVICE: not set")
fi

if [ -n "$DD_ENV" ]; then
    echo "[OK] DD_ENV is set ($DD_ENV)" >&2
    results+=("DD_ENV: $DD_ENV")
else
    echo "[WARN] DD_ENV not set" >&2
    warnings+=("DD_ENV: not set")
fi

echo "" >&2

# Check Datadog Agent connectivity
echo "Checking Datadog Agent connectivity..." >&2
DD_AGENT_HOST=${DD_AGENT_HOST:-localhost}
DD_TRACE_AGENT_PORT=${DD_TRACE_AGENT_PORT:-8126}
DD_DOGSTATSD_PORT=${DD_DOGSTATSD_PORT:-8125}

echo "Agent host: $DD_AGENT_HOST" >&2
echo "Trace port: $DD_TRACE_AGENT_PORT" >&2
echo "DogStatsD port: $DD_DOGSTATSD_PORT" >&2
echo "" >&2

# Check APM endpoint
if check_endpoint "http://$DD_AGENT_HOST:$DD_TRACE_AGENT_PORT/info" "Datadog APM agent"; then
    results+=("APM_AGENT: reachable at $DD_AGENT_HOST:$DD_TRACE_AGENT_PORT")
else
    errors+=("APM_AGENT: unreachable at $DD_AGENT_HOST:$DD_TRACE_AGENT_PORT")
fi

# Check if netcat is available for UDP check
if command_exists nc; then
    if echo "test" | nc -u -w 1 "$DD_AGENT_HOST" "$DD_DOGSTATSD_PORT" 2>/dev/null; then
        echo "[OK] DogStatsD port is accessible" >&2
        results+=("DOGSTATSD: accessible at $DD_AGENT_HOST:$DD_DOGSTATSD_PORT")
    else
        echo "[FAIL] DogStatsD port is not accessible" >&2
        errors+=("DOGSTATSD: inaccessible at $DD_AGENT_HOST:$DD_DOGSTATSD_PORT")
    fi
else
    echo "[WARN] netcat not found, cannot check DogStatsD port" >&2
    warnings+=("DOGSTATSD: cannot verify (netcat not installed)")
fi

echo "" >&2

# Check language-specific tracer installations
echo "Checking for installed Datadog tracers..." >&2

# Node.js
if [ -f "package.json" ]; then
    if grep -q "dd-trace" package.json 2>/dev/null; then
        echo "[OK] Node.js tracer (dd-trace) found in package.json" >&2
        results+=("TRACER_NODEJS: installed")
    else
        echo "[WARN] Node.js project detected but dd-trace not in package.json" >&2
        warnings+=("TRACER_NODEJS: not installed")
    fi
fi

# Python
if [ -f "requirements.txt" ]; then
    if grep -q "ddtrace" requirements.txt 2>/dev/null; then
        echo "[OK] Python tracer (ddtrace) found in requirements.txt" >&2
        results+=("TRACER_PYTHON: installed")
    else
        echo "[WARN] Python project detected but ddtrace not in requirements.txt" >&2
        warnings+=("TRACER_PYTHON: not installed")
    fi
fi

# .NET
if [ -f "*.csproj" ] || ls *.csproj >/dev/null 2>&1; then
    if grep -q "Datadog.Trace" *.csproj 2>/dev/null; then
        echo "[OK] .NET tracer found in project file" >&2
        results+=("TRACER_DOTNET: installed")
    else
        echo "[WARN] .NET project detected but Datadog.Trace not in project file" >&2
        warnings+=("TRACER_DOTNET: not installed")
    fi
fi

echo "" >&2

# Check Docker/Container environment
if [ -f "/.dockerenv" ] || [ -n "$KUBERNETES_SERVICE_HOST" ]; then
    echo "Container environment detected" >&2
    if [ -n "$DD_AGENT_HOST" ]; then
        echo "[OK] DD_AGENT_HOST is configured for container" >&2
        results+=("CONTAINER_CONFIG: DD_AGENT_HOST set")
    else
        echo "[FAIL] DD_AGENT_HOST should be set in container environments" >&2
        errors+=("CONTAINER_CONFIG: DD_AGENT_HOST not set")
    fi
fi

echo "" >&2

# Summary
echo "=============================" >&2
echo "Summary" >&2
echo "=============================" >&2
echo "Results: ${#results[@]}" >&2
echo "Warnings: ${#warnings[@]}" >&2
echo "Errors: ${#errors[@]}" >&2
echo "" >&2

# Output JSON result
cat <<EOF
{
  "status": "$([ ${#errors[@]} -eq 0 ] && echo "ok" || echo "error")",
  "results": $(printf '%s\n' "${results[@]}" | jq -R . | jq -s .),
  "warnings": $(printf '%s\n' "${warnings[@]}" | jq -R . | jq -s .),
  "errors": $(printf '%s\n' "${errors[@]}" | jq -R . | jq -s .),
  "agent_host": "$DD_AGENT_HOST",
  "site": "$DD_SITE",
  "service": "${DD_SERVICE:-not_set}",
  "env": "${DD_ENV:-not_set}"
}
EOF

# Exit with error if there are errors
[ ${#errors[@]} -eq 0 ] && exit 0 || exit 1
