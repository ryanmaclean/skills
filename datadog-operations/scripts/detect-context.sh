#!/bin/bash
# Detect service context - thin wrapper around Python

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PYTHON_SCRIPT="$SCRIPT_DIR/../python/detect_context.py"

# Check if Python is available
if ! command -v python3 &> /dev/null; then
    echo "[ERROR] python3 not found - install Python 3.7+ to use this skill" >&2
    exit 1
fi

# Run Python script
exec python3 "$PYTHON_SCRIPT" "$@"
