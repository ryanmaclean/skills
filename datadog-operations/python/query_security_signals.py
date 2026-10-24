#!/usr/bin/env python3
"""
Query Datadog Security Monitoring Signals.
Analyzes security events and attack attempts with built-in observability.
"""

import sys
import json
import argparse
from pathlib import Path
from datetime import datetime, timedelta
from typing import Optional, Dict, List
from collections import Counter

# Add lib to path
sys.path.insert(0, str(Path(__file__).parent / "lib"))

from dd_observability import init_observability, finalize_observability
from datadog_client import create_client
from context_detector import detect_context


def parse_duration(duration: str) -> timedelta:
    """Parse duration string like '1h', '24h', '7d', '30d'"""
    if duration.endswith('h'):
        return timedelta(hours=int(duration[:-1]))
    elif duration.endswith('d'):
        return timedelta(days=int(duration[:-1]))
    else:
        raise ValueError(f"Invalid duration: {duration}. Use format like '1h', '24h', '7d', '30d'")


def extract_tag_value(tags: List[str], prefix: str) -> Optional[str]:
    """Extract value from tag with given prefix"""
    for tag in tags:
        if tag.startswith(prefix):
            return tag[len(prefix):]
    return None


def main():
    obs = init_observability("query-security-signals")

    parser = argparse.ArgumentParser(
        description="Query Datadog Security Monitoring Signals"
    )
    parser.add_argument(
        "--service",
        help="Filter by service name (auto-detected if not provided)"
    )
    parser.add_argument(
        "--duration",
        default="24h",
        help="Time range: 1h, 24h, 7d, 30d (default: 24h)"
    )
    parser.add_argument(
        "--severity",
        choices=["critical", "high", "medium", "low", "info"],
        help="Filter by severity"
    )
    parser.add_argument(
        "--json",
        action="store_true",
        help="Output as JSON"
    )

    args = parser.parse_args()

    try:
        # Auto-detect service if needed
        with obs.span("detect_context"):
            service = args.service
            if not service:
                context = detect_context()
                service = context.service_name
                if service:
                    obs.log_info(f"Auto-detected service: {service}")

        # Parse duration
        with obs.span("parse_duration"):
            duration = parse_duration(args.duration)
            to_time = datetime.now()
            from_time = to_time - duration

        # Create client
        with obs.span("create_client"):
            client = create_client()

        # Query security signals
        with obs.span("query_security_signals", tags={"service": service or "all", "duration": args.duration}):
            start = datetime.now()

            data = client.get_security_signals(
                from_time=from_time,
                to_time=to_time,
                service=service
            )

            api_duration = (datetime.now() - start).total_seconds() * 1000
            obs.record_api_call("/api/v2/security_monitoring/signals/search", "POST", 200, api_duration)

        # Parse results
        with obs.span("parse_results"):
            signals = data.get('data', [])
            total_signals = len(signals)

            obs.log_info(f"Found {total_signals} security signals")
            obs.record_result("signals", total_signals)

            if total_signals == 0:
                output = {
                    "status": "ok",
                    "total_signals": 0,
                    "duration": args.duration,
                    "service": service or "all",
                    "signals": [],
                    "summary": {
                        "critical": 0,
                        "high": 0,
                        "medium": 0,
                        "low": 0,
                        "info": 0
                    },
                    "attack_types": {},
                    "affected_services": {}
                }

                if args.json:
                    print(json.dumps(output, indent=2))
                else:
                    print(f"No security signals found for the specified criteria")

                finalize_observability(0)
                sys.exit(0)

            # Count by severity
            severity_counts = Counter()
            attack_types = Counter()
            affected_services = Counter()

            for signal in signals:
                attrs = signal.get('attributes', {})
                severity = attrs.get('severity', 'unknown')
                severity_counts[severity] += 1

                # Extract tags
                tags = attrs.get('tags', [])

                # Extract attack type and rule name
                for tag in tags:
                    if tag.startswith('attack_type:') or tag.startswith('rule_name:'):
                        attack_types[tag] += 1
                    if tag.startswith('service:'):
                        affected_services[tag] += 1

        # Record metrics
        critical_count = severity_counts.get('critical', 0)
        high_count = severity_counts.get('high', 0)
        medium_count = severity_counts.get('medium', 0)
        low_count = severity_counts.get('low', 0)
        info_count = severity_counts.get('info', 0)

        obs.record_result("critical_signals", critical_count)
        obs.record_result("high_signals", high_count)
        obs.gauge("security.critical_signals", critical_count)
        obs.gauge("security.high_signals", high_count)

        # Determine overall status
        if critical_count > 0:
            overall_status = "critical"
        elif high_count > 0:
            overall_status = "warning"
        else:
            overall_status = "ok"

        # Extract recent signals
        recent_signals = []
        for signal in signals[:5]:
            attrs = signal.get('attributes', {})
            tags = attrs.get('tags', [])

            recent_signals.append({
                'id': signal.get('id'),
                'severity': attrs.get('severity'),
                'timestamp': attrs.get('timestamp'),
                'rule_name': extract_tag_value(tags, 'rule_name:'),
                'service': extract_tag_value(tags, 'service:')
            })

        # Output
        if args.json:
            output = {
                "status": overall_status,
                "total_signals": total_signals,
                "duration": args.duration,
                "service": service or "all",
                "summary": {
                    "critical": critical_count,
                    "high": high_count,
                    "medium": medium_count,
                    "low": low_count,
                    "info": info_count
                },
                "attack_types": dict(attack_types),
                "affected_services": dict(affected_services),
                "recent_signals": recent_signals
            }
            print(json.dumps(output, indent=2))
        else:
            # Conversational output
            print(f"Security Signals Analysis")
            print(f"Duration: {args.duration}")
            if service:
                print(f"Service: {service}")
            print()
            print(f"Found {total_signals:,} security signals")
            print()
            print("Severity breakdown:")
            print(f"  Critical: {critical_count}")
            print(f"  High: {high_count}")
            print(f"  Medium: {medium_count}")
            print(f"  Low: {low_count}")
            print(f"  Info: {info_count}")

            if critical_count > 0 or high_count > 0:
                print()
                print(f"ALERT: {critical_count} critical and {high_count} high-severity security signals detected")

            if attack_types:
                print()
                print("Top attack types:")
                for attack_type, count in attack_types.most_common(5):
                    print(f"  {attack_type}: {count}")

            if affected_services:
                print()
                print("Affected services:")
                for service_tag, count in affected_services.most_common(5):
                    print(f"  {service_tag}: {count}")

        obs.log_info(f"Query completed: {total_signals} signals, {critical_count} critical")
        finalize_observability(0)
        sys.exit(0)

    except ValueError as e:
        obs.log_error(f"Invalid input: {str(e)}")
        print(f"Error: {e}", file=sys.stderr)
        finalize_observability(1)
        sys.exit(1)
    except KeyError as e:
        obs.log_error(f"Missing environment variable: {e}")
        print(f"Error: Missing environment variable - {e}", file=sys.stderr)
        print("Set DD_API_KEY and DD_APP_KEY", file=sys.stderr)
        finalize_observability(1)
        sys.exit(1)
    except Exception as e:
        obs.log_error(f"Query failed: {str(e)}", error_type=type(e).__name__)
        print(f"Error: {e}", file=sys.stderr)
        finalize_observability(1)
        sys.exit(1)


if __name__ == "__main__":
    main()
