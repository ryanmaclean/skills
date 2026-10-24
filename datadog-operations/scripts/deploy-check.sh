#!/bin/bash
# Deploy readiness check - thin wrapper around Python

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PYTHON_SCRIPT="$SCRIPT_DIR/../python/deploy_check.py"

# Show help
if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    cat << 'EOF'
Deploy Readiness Check

Answers the question: "Can I deploy?"

Checks:
- Error budget status
- Recent errors since last deploy
- SLO breaches
- Security signals
- Performance degradation

Usage:
  deploy-check.sh [OPTIONS]

Options:
  --service SERVICE      Service name (auto-detected from git if not provided)
  --json                 Output as JSON
  --working-dir DIR      Project directory (default: current)

Examples:
  # Check if safe to deploy (auto-detect service)
  deploy-check.sh

  # Check specific service
  deploy-check.sh --service payment-api

  # Get JSON output for automation
  deploy-check.sh --json

Exit codes:
  0 - Safe to deploy
  1 - Not safe to deploy (critical issues)

Environment:
  DD_API_KEY   - Datadog API key (required)
  DD_APP_KEY   - Datadog application key (required)
  DD_SITE      - Datadog site (default: datadoghq.com)
EOF
    exit 0
fi

# Check if Python is available
if ! command -v python3 &> /dev/null; then
    echo "[ERROR] python3 not found - install Python 3.7+ to use this skill" >&2
    exit 1
fi

# Check dependencies
if ! python3 -c "import requests" 2>/dev/null; then
    echo "[ERROR] Python dependencies not installed" >&2
    echo "[INFO] Run: pip3 install -r python/requirements.txt" >&2
    exit 1
fi

# Run Python script
exec python3 "$PYTHON_SCRIPT" "$@"
