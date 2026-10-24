#!/usr/bin/env python3
"""
Detect service context from git, package.json, etc.
Auto-discovers what service you're working on.
"""

import sys
import json
import argparse
from pathlib import Path

# Add lib to path
sys.path.insert(0, str(Path(__file__).parent / "lib"))

from context_detector import detect_context
from conversational_output import format_context


def main():
    parser = argparse.ArgumentParser(
        description="Auto-detect service context from your project"
    )
    parser.add_argument(
        "--working-dir",
        default=".",
        help="Project directory (default: current directory)"
    )
    parser.add_argument(
        "--json",
        action="store_true",
        help="Output as JSON instead of formatted text"
    )
    parser.add_argument(
        "--quiet",
        action="store_true",
        help="Only output service name"
    )

    args = parser.parse_args()

    try:
        context = detect_context(args.working_dir)

        if args.quiet:
            print(context.service_name or "")
            sys.exit(0 if context.service_name else 1)

        if args.json:
            print(json.dumps(context.to_dict(), indent=2))
        else:
            print(format_context(context))

        sys.exit(0 if context.service_name else 1)

    except Exception as e:
        print(f"Error detecting context: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
