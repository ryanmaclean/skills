#!/usr/bin/env python3
"""
Smart health check - comprehensive service health analysis.
Auto-detects service and checks APM, logs, errors, and SLOs.
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
from conversational_output import format_health_report, format_quick_summary


def main():
    parser = argparse.ArgumentParser(
        description="Smart health check for your service"
    )
    parser.add_argument(
        "--service",
        help="Service name (auto-detected if not provided)"
    )
    parser.add_argument(
        "--duration",
        type=int,
        default=1,
        help="How many hours to look back (default: 1)"
    )
    parser.add_argument(
        "--since-deploy",
        action="store_true",
        help="Check since last deploy (from git history)"
    )
    parser.add_argument(
        "--json",
        action="store_true",
        help="Output as JSON instead of formatted text"
    )
    parser.add_argument(
        "--summary",
        action="store_true",
        help="Ultra-short summary (one line)"
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

        # Analyze health
        analyzer = HealthAnalyzer(client)
        report = analyzer.analyze(
            context=context,
            duration_hours=args.duration,
            check_since_deploy=args.since_deploy
        )

        # Output
        if args.summary:
            print(format_quick_summary(report))
        elif args.json:
            print(json.dumps(report.to_dict(), indent=2))
        else:
            print(format_health_report(report))

        # Exit with error if critical issues
        sys.exit(0 if report.status != "critical" else 1)

    except KeyError as e:
        print(f"Error: Missing environment variable - {e}", file=sys.stderr)
        print("Set DD_API_KEY and DD_APP_KEY", file=sys.stderr)
        sys.exit(1)
    except Exception as e:
        print(f"Error analyzing health: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
