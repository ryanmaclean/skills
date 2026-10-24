#!/bin/bash
# Smart health check - thin wrapper around Python

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PYTHON_SCRIPT="$SCRIPT_DIR/../python/smart_health.py"

# Show help
if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    cat << 'EOF'
Smart Health Check

Automatically detects your service and checks comprehensive health:
- APM trace performance
- Error logs
- Security signals
- SLO status

Usage:
  smart-health.sh [OPTIONS]

Options:
  --service SERVICE      Service name (auto-detected from git if not provided)
  --duration HOURS       Hours to look back (default: 1)
  --since-deploy         Check since last git commit
  --summary              Ultra-short one-line summary
  --json                 Output as JSON
  --working-dir DIR      Project directory (default: current)

Examples:
  # Auto-detect service and check health
  smart-health.sh

  # Check specific service
  smart-health.sh --service payment-api

  # Check since last deploy
  smart-health.sh --since-deploy

  # Get quick summary
  smart-health.sh --summary

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
