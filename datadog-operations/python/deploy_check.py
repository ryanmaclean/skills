#!/usr/bin/env python3
"""
Deploy readiness check - "Can I deploy?"
Checks error budget, recent errors, and SLO status.
"""

import sys
import json
import argparse
from pathlib import Path

# Add lib to path
sys.path.insert(0, str(Path(__file__).parent / "lib"))

from context_detector import detect_context
from datadog_client import create_client
from health_analyzer import HealthAnalyzer
from conversational_output import format_deploy_readiness


def main():
    parser = argparse.ArgumentParser(
        description="Check if it's safe to deploy"
    )
    parser.add_argument(
        "--service",
        help="Service name (auto-detected if not provided)"
    )
    parser.add_argument(
        "--json",
        action="store_true",
        help="Output as JSON instead of formatted text"
    )
    parser.add_argument(
        "--working-dir",
        default=".",
        help="Project directory (default: current directory)"
    )

    args = parser.parse_args()

    try:
        # Detect context if service not provided
        if not args.service:
            context = detect_context(args.working_dir)
            if not context.service_name:
                print("Error: Could not detect service name", file=sys.stderr)
                print("Specify with --service or run in a git repository", file=sys.stderr)
                sys.exit(1)
        else:
            from context_detector import ServiceContext
            context = ServiceContext()
            context.service_name = args.service

        # Create Datadog client
        client = create_client()

        # Analyze health (check since deploy)
        analyzer = HealthAnalyzer(client)
        report = analyzer.analyze(
            context=context,
            duration_hours=24,  # Look back up to 24 hours
            check_since_deploy=True  # Focus on changes since last deploy
        )

        # Output deploy readiness
        if args.json:
            output = report.to_dict()
            output["can_deploy"] = report.status in ["healthy", "degraded"]
            print(json.dumps(output, indent=2))
        else:
            print(format_deploy_readiness(report, context))

        # Exit codes:
        # 0 = safe to deploy (healthy or degraded)
        # 1 = not safe to deploy (critical issues)
        can_deploy = report.status in ["healthy", "degraded"]
        sys.exit(0 if can_deploy else 1)

    except KeyError as e:
        print(f"Error: Missing environment variable - {e}", file=sys.stderr)
        print("Set DD_API_KEY and DD_APP_KEY", file=sys.stderr)
        sys.exit(1)
    except Exception as e:
        print(f"Error checking deploy readiness: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
