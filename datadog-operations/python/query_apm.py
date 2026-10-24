#!/usr/bin/env python3
"""
Query Datadog APM for performance analysis.
Finds slow endpoints, errors, and performance bottlenecks.
"""

import sys
import json
import argparse
from pathlib import Path
from datetime import datetime, timedelta
from typing import Optional

# Add lib to path
sys.path.insert(0, str(Path(__file__).parent / "lib"))

from dd_observability import init_observability, finalize_observability
from datadog_client import create_client
from context_detector import detect_context


def parse_duration(duration: str) -> timedelta:
    """Parse duration string like '1h', '24h', '7d'"""
    if duration.endswith('h'):
        return timedelta(hours=int(duration[:-1]))
    elif duration.endswith('d'):
        return timedelta(days=int(duration[:-1]))
    else:
        raise ValueError(f"Invalid duration: {duration}. Use format like '1h', '24h', '7d'")


def main():
    obs = init_observability("query-apm")

    parser = argparse.ArgumentParser(
        description="Query Datadog APM for performance analysis"
    )
    parser.add_argument(
        "--service",
        help="Service name (auto-detected if not provided)"
    )
    parser.add_argument(
        "--duration",
        default="1h",
        help="Time range: 1h, 24h, 7d (default: 1h)"
    )
    parser.add_argument(
        "--status",
        choices=["error", "ok", "all"],
        help="Filter by status (default: all)"
    )
    parser.add_argument(
        "--limit",
        type=int,
        default=20,
        help="Max endpoints to return (default: 20)"
    )
    parser.add_argument(
        "--json",
        action="store_true",
        help="Output as JSON"
    )

    args = parser.parse_args()

    try:
        # Detect service if not provided
        with obs.span("detect_context"):
            if not args.service:
                context = detect_context()
                service = context.service_name
                if not service:
                    obs.log_error("Could not detect service name")
                    print("Error: Could not detect service name", file=sys.stderr)
                    print("Specify with --service or run in a git repository", file=sys.stderr)
                    finalize_observability(1)
                    sys.exit(1)
                obs.log_info(f"Auto-detected service: {service}")
            else:
                service = args.service

        obs.log_info(f"Querying APM for service: {service}")

        # Parse duration
        with obs.span("parse_duration"):
            duration = parse_duration(args.duration)
            to_time = datetime.now()
            from_time = to_time - duration

        # Create Datadog client
        with obs.span("create_client"):
            client = create_client()

        # Query APM
        with obs.span("query_apm", tags={"service": service, "duration": args.duration}):
            start = datetime.now()

            data = client.query_apm_traces(
                service=service,
                from_time=from_time,
                to_time=to_time,
                status=args.status if args.status != "all" else None,
                limit=args.limit
            )

            api_duration = (datetime.now() - start).total_seconds() * 1000
            obs.record_api_call("/api/v2/spans/analytics/aggregate", "POST", 200, api_duration)

        # Parse results
        with obs.span("parse_results"):
            # Handle API v2 response format: .data is a list of items with .attributes
            items = []
            if "data" in data:
                if isinstance(data["data"], list):
                    # v2 format: data is array of items with .attributes.by and .attributes.compute
                    items = data["data"]
                elif isinstance(data["data"], dict):
                    # Fallback: check for buckets in older format
                    buckets = data["data"].get("attributes", {}).get("buckets", [])
                    if not buckets:
                        buckets = data["data"].get("buckets", [])
                    items = buckets

            if not items:
                obs.log_warning("No trace data found")
                obs.record_result("endpoints", 0)

                output = {
                    "status": "no_data",
                    "service": service,
                    "duration": args.duration,
                    "endpoints": []
                }

                if args.json:
                    print(json.dumps(output, indent=2))
                else:
                    print(f"ℹ️ No trace data found for service: {service}")

                finalize_observability(0)
                sys.exit(0)

            endpoints = []

            for item in items:
                # v2 format: item.attributes.by and item.attributes.compute
                if "attributes" in item:
                    attrs = item["attributes"]
                    resource_name = attrs.get("by", {}).get("resource_name", "unknown")
                    computes = attrs.get("compute", {})
                else:
                    # Fallback for bucket format
                    resource_name = item.get("by", {}).get("resource_name", "unknown")
                    computes = item.get("computes", {})

                # API format: c0=count, c1=avg duration, c2=max duration, c3=min duration
                request_count = computes.get("c0", 0)
                avg_ns = computes.get("c1", 0)
                max_ns = computes.get("c2", 0)
                min_ns = computes.get("c3", 0)

                endpoint = {
                    "resource_name": resource_name,
                    "request_count": request_count,
                    "avg_ms": int(avg_ns / 1_000_000) if avg_ns else 0,
                    "max_ms": int(max_ns / 1_000_000) if max_ns else 0,
                    "min_ms": int(min_ns / 1_000_000) if min_ns else 0
                }
                endpoints.append(endpoint)

        # Calculate statistics
        with obs.span("calculate_stats"):
            total_endpoints = len(endpoints)
            total_requests = sum(e["request_count"] for e in endpoints)
            avg_latency = int(sum(e["avg_ms"] for e in endpoints) / total_endpoints) if total_endpoints > 0 else 0
            slow_endpoints = [e for e in endpoints if e["avg_ms"] > 500]

            obs.record_result("endpoints", total_endpoints)
            obs.record_result("requests", total_requests)
            obs.record_result("slow_endpoints", len(slow_endpoints))
            obs.gauge("apm.avg_latency_ms", avg_latency)

        # Output
        if args.json:
            output = {
                "status": "ok",
                "service": service,
                "duration": args.duration,
                "summary": {
                    "total_endpoints": total_endpoints,
                    "total_requests": total_requests,
                    "avg_latency_ms": avg_latency,
                    "slow_endpoints_count": len(slow_endpoints)
                },
                "endpoints": endpoints
            }
            print(json.dumps(output, indent=2))
        else:
            # Conversational output
            print(f"📊 APM Analysis: {service}")
            print(f"Duration: {args.duration}")
            print()
            print(f"✓ {total_endpoints} endpoints analyzed")
            print(f"✓ {total_requests:,} requests")
            print(f"✓ Average latency: {avg_latency}ms")

            if slow_endpoints:
                print()
                print(f"⚠️ {len(slow_endpoints)} slow endpoints (avg > 500ms):")
                for endpoint in slow_endpoints[:5]:
                    print(f"  • {endpoint['resource_name']}")
                    print(f"    Avg: {endpoint['avg_ms']}ms | {endpoint['request_count']:,} requests")
            else:
                print()
                print("✅ All endpoints performing well")

        obs.log_info(f"Query completed: {total_endpoints} endpoints")
        finalize_observability(0)
        sys.exit(0)

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
