#!/usr/bin/env python3
"""
Search and analyze Datadog logs for error patterns.
Built-in observability with traces, logs, and metrics.
"""

import sys
import json
import argparse
from pathlib import Path
from datetime import datetime, timedelta
from typing import Optional, Dict, List, Any
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


def analyze_error_patterns(logs: List[Dict[str, Any]], max_patterns: int = 10) -> List[Dict[str, Any]]:
    """Group error messages by frequency and extract patterns"""
    error_messages = []

    for log in logs:
        attrs = log.get('attributes', {})
        if attrs.get('status') == 'error':
            # Try multiple message locations
            message = (
                attrs.get('message') or
                attrs.get('attributes', {}).get('message') or
                'No message'
            )
            # Truncate to first 200 chars for pattern matching
            message = str(message)[:200]

            service = attrs.get('service', 'unknown')
            host = attrs.get('host', 'unknown')

            error_messages.append({
                'message': message,
                'service': service,
                'host': host
            })

    # Group by message
    message_counts = Counter(e['message'] for e in error_messages)

    patterns = []
    for message, count in message_counts.most_common(max_patterns):
        # Find services and hosts for this message
        matching_errors = [e for e in error_messages if e['message'] == message]
        services = list(set(e['service'] for e in matching_errors))
        hosts = list(set(e['host'] for e in matching_errors))

        patterns.append({
            'message': message,
            'count': count,
            'services': services,
            'hosts': hosts
        })

    return patterns


def main():
    obs = init_observability("search-logs")

    parser = argparse.ArgumentParser(
        description="Search and analyze Datadog logs for error patterns"
    )
    parser.add_argument(
        "--query",
        help="Log search query (Datadog query syntax)"
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
        "--status",
        choices=["error", "warn", "info"],
        help="Filter by status"
    )
    parser.add_argument(
        "--limit",
        type=int,
        default=100,
        help="Number of results (default: 100, max: 1000)"
    )
    parser.add_argument(
        "--json",
        action="store_true",
        help="Output as JSON"
    )

    args = parser.parse_args()

    try:
        # Validate limit
        if args.limit > 1000:
            obs.log_error("Limit cannot exceed 1000")
            print("Error: Limit cannot exceed 1000", file=sys.stderr)
            finalize_observability(1)
            sys.exit(1)

        # Auto-detect service if needed
        with obs.span("detect_context"):
            service = args.service
            if not service and not args.query:
                context = detect_context()
                service = context.service_name
                if service:
                    obs.log_info(f"Auto-detected service: {service}")

        # Build query
        with obs.span("build_query"):
            query_parts = []

            if service:
                query_parts.append(f"service:{service}")

            if args.status:
                query_parts.append(f"status:{args.status}")

            if args.query:
                query_parts.append(f"({args.query})")

            # Default to error if no query specified
            if not query_parts:
                query_parts.append("status:error")

            search_query = " AND ".join(query_parts)
            obs.log_info(f"Search query: {search_query}")

        # Parse duration
        with obs.span("parse_duration"):
            duration = parse_duration(args.duration)
            to_time = datetime.now()
            from_time = to_time - duration

        # Create client
        with obs.span("create_client"):
            client = create_client()

        # Search logs
        with obs.span("search_logs", tags={"query": search_query, "duration": args.duration}):
            start = datetime.now()

            data = client.search_logs(
                query=search_query,
                from_time=from_time,
                to_time=to_time,
                limit=args.limit
            )

            api_duration = (datetime.now() - start).total_seconds() * 1000
            obs.record_api_call("/api/v2/logs/events/search", "POST", 200, api_duration)

        # Parse results
        with obs.span("parse_results"):
            logs = data.get('data', [])
            total_logs = len(logs)

            obs.log_info(f"Found {total_logs} log entries")
            obs.record_result("logs", total_logs)

            if total_logs == 0:
                output = {
                    "status": "ok",
                    "total_logs": 0,
                    "query": search_query,
                    "duration": args.duration,
                    "logs": [],
                    "error_patterns": [],
                    "summary": {
                        "error": 0,
                        "warn": 0,
                        "info": 0
                    },
                    "services": {},
                    "hosts": {}
                }

                if args.json:
                    print(json.dumps(output, indent=2))
                else:
                    print(f"No logs found for query: {search_query}")

                finalize_observability(0)
                sys.exit(0)

            # Count by status
            status_counts = Counter()
            service_counts = Counter()
            host_counts = Counter()
            trace_ids = []

            for log in logs:
                attrs = log.get('attributes', {})
                status = attrs.get('status', 'unknown')
                status_counts[status] += 1

                service_name = attrs.get('service', 'unknown')
                service_counts[service_name] += 1

                host_name = attrs.get('host', 'unknown')
                host_counts[host_name] += 1

                # Check for trace ID
                trace_id = attrs.get('attributes', {}).get('dd', {}).get('trace_id')
                if trace_id:
                    trace_ids.append(trace_id)

        # Analyze error patterns
        with obs.span("analyze_patterns"):
            error_patterns = analyze_error_patterns(logs)
            obs.record_result("error_patterns", len(error_patterns))

        # Calculate statistics
        error_count = status_counts.get('error', 0)
        warn_count = status_counts.get('warn', 0)
        info_count = status_counts.get('info', 0)
        trace_count = len(trace_ids)

        obs.record_result("errors", error_count)
        obs.record_result("warnings", warn_count)
        obs.gauge("logs.error_count", error_count)

        # Determine overall status
        if error_count > 0:
            overall_status = "error"
        elif warn_count > 0:
            overall_status = "warning"
        else:
            overall_status = "ok"

        # Extract recent logs for output
        recent_logs = []
        for log in logs[:10]:
            attrs = log.get('attributes', {})
            dd_attrs = attrs.get('attributes', {})
            dd_trace = dd_attrs.get('dd', {})

            recent_logs.append({
                'timestamp': attrs.get('timestamp'),
                'status': attrs.get('status', 'unknown'),
                'service': attrs.get('service', 'unknown'),
                'host': attrs.get('host', 'unknown'),
                'message': str(attrs.get('message', 'No message'))[:500],
                'trace_id': dd_trace.get('trace_id'),
                'span_id': dd_trace.get('span_id'),
                'container_id': dd_attrs.get('container_id')
            })

        # Output
        if args.json:
            output = {
                "status": overall_status,
                "total_logs": total_logs,
                "query": search_query,
                "duration": args.duration,
                "summary": {
                    "error": error_count,
                    "warn": warn_count,
                    "info": info_count
                },
                "error_patterns": error_patterns,
                "services": dict(service_counts),
                "hosts": dict(host_counts),
                "trace_ids_count": trace_count,
                "recent_logs": recent_logs
            }
            print(json.dumps(output, indent=2))
        else:
            # Conversational output
            print(f"Log Search Results")
            print(f"Query: {search_query}")
            print(f"Duration: {args.duration}")
            print()
            print(f"Found {total_logs:,} log entries")
            print()
            print("Status breakdown:")
            print(f"  Errors: {error_count}")
            print(f"  Warnings: {warn_count}")
            print(f"  Info: {info_count}")

            if error_patterns:
                print()
                print(f"Top error patterns ({len(error_patterns)}):")
                for i, pattern in enumerate(error_patterns[:5], 1):
                    print(f"  {i}. {pattern['message'][:80]}...")
                    print(f"     Count: {pattern['count']} | Services: {', '.join(pattern['services'][:3])}")

            if service_counts:
                print()
                print("Top services:")
                for service_name, count in service_counts.most_common(5):
                    print(f"  {service_name}: {count}")

            if trace_count > 0:
                print()
                print(f"Found {trace_count} logs with trace IDs")

        obs.log_info(f"Search completed: {total_logs} logs, {error_count} errors")
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
        obs.log_error(f"Search failed: {str(e)}", error_type=type(e).__name__)
        print(f"Error: {e}", file=sys.stderr)
        finalize_observability(1)
        sys.exit(1)


if __name__ == "__main__":
    main()
