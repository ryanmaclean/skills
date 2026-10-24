#!/usr/bin/env python3
"""
Query Datadog Watchdog API for Anomaly Detection.
Analyzes automated anomaly detection with built-in observability.
"""

import sys
import json
import argparse
from pathlib import Path
from datetime import datetime, timedelta
from typing import Optional, Dict, List
from collections import Counter
import re

# Add lib to path
sys.path.insert(0, str(Path(__file__).parent / "lib"))

from dd_observability import init_observability, finalize_observability
from datadog_client import DatadogClient
from context_detector import detect_context


def parse_duration(duration: str) -> timedelta:
    """Parse duration string like '1h', '24h', '7d', '30d'"""
    if duration.endswith('h'):
        return timedelta(hours=int(duration[:-1]))
    elif duration.endswith('d'):
        return timedelta(days=int(duration[:-1]))
    else:
        raise ValueError(f"Invalid duration: {duration}. Use format like '1h', '24h', '7d', '30d'")


def categorize_anomaly(title: str) -> str:
    """Categorize anomaly based on title"""
    title_lower = title.lower()

    if re.search(r'latency|p99|response.?time', title_lower):
        return 'latency_spike'
    elif re.search(r'error|failure', title_lower):
        return 'error_rate_increase'
    elif re.search(r'hits|traffic|request|throughput|drop', title_lower):
        return 'traffic_drop'
    else:
        return 'other'


def extract_tag_value(tags: List[str], prefix: str) -> Optional[str]:
    """Extract value from tag with given prefix"""
    for tag in tags:
        if tag.startswith(prefix):
            return tag[len(prefix):]
    return None


def main():
    obs = init_observability("query-watchdog")

    parser = argparse.ArgumentParser(
        description="Query Datadog Watchdog for anomaly detection"
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
        "--type",
        choices=["latency", "error_rate", "traffic", "all"],
        default="all",
        help="Filter by anomaly type"
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

        # Build query
        with obs.span("build_query"):
            query_parts = ["source:watchdog"]

            if service:
                query_parts.append(f"service:{service}")

            # Add type filters
            if args.type == "latency":
                query_parts.append("(latency OR p99 OR response_time)")
            elif args.type == "error_rate":
                query_parts.append("(error OR error_rate OR errors)")
            elif args.type == "traffic":
                query_parts.append("(hits OR traffic OR request_rate OR throughput)")

            query = " ".join(query_parts)
            obs.log_info(f"Watchdog query: {query}")

        # Parse duration
        with obs.span("parse_duration"):
            duration = parse_duration(args.duration)
            to_time = datetime.now()
            from_time = to_time - duration

        # Create client
        with obs.span("create_client"):
            client = DatadogClient()

        # Query Watchdog events via Events API
        with obs.span("query_watchdog", tags={"service": service or "all", "duration": args.duration}):
            start = datetime.now()

            # Convert to ISO format for events API
            from_iso = from_time.isoformat()
            to_iso = to_time.isoformat()

            # Use Events API v2
            payload = {
                "filter": {
                    "query": query,
                    "from": from_iso,
                    "to": to_iso
                },
                "page": {
                    "limit": 100
                },
                "sort": "timestamp"
            }

            response = client._request("POST", "/api/v2/events/search", json=payload)
            data = response.json()

            api_duration = (datetime.now() - start).total_seconds() * 1000
            obs.record_api_call("/api/v2/events/search", "POST", 200, api_duration)

        # Parse results
        with obs.span("parse_results"):
            events = data.get('data', [])
            total_anomalies = len(events)

            obs.log_info(f"Found {total_anomalies} Watchdog anomalies")
            obs.record_result("anomalies", total_anomalies)

            if total_anomalies == 0:
                output = {
                    "status": "ok",
                    "total_anomalies": 0,
                    "duration": args.duration,
                    "service": service or "all",
                    "query": query,
                    "anomalies": [],
                    "summary": {
                        "latency_spikes": 0,
                        "error_rate_increases": 0,
                        "traffic_drops": 0,
                        "other": 0
                    },
                    "affected_services": {}
                }

                if args.json:
                    print(json.dumps(output, indent=2))
                else:
                    print(f"No Watchdog anomalies found for the specified criteria")

                finalize_observability(0)
                sys.exit(0)

            # Categorize anomalies
            category_counts = Counter()
            service_counts = Counter()
            anomalies = []

            for event in events:
                attrs = event.get('attributes', {})
                event_attrs = attrs.get('attributes', {})
                title = event_attrs.get('title', '')
                tags = attrs.get('tags', [])

                category = categorize_anomaly(title)
                category_counts[category] += 1

                service_tag = extract_tag_value(tags, 'service:')
                if service_tag:
                    service_counts[service_tag] += 1

                # Determine severity from priority
                priority = event_attrs.get('priority', 'normal')
                if priority == 'normal':
                    severity = 'medium'
                elif priority == 'low':
                    severity = 'low'
                else:
                    severity = 'high'

                # Extract message (first line only)
                message = event_attrs.get('message', '')
                if message:
                    message = message.split('\n')[0]

                anomalies.append({
                    'id': event.get('id'),
                    'type': event.get('type'),
                    'timestamp': attrs.get('timestamp'),
                    'title': title,
                    'message': message,
                    'tags': tags,
                    'priority': priority,
                    'service': service_tag,
                    'resource': extract_tag_value(tags, 'resource_name:'),
                    'anomaly_category': category,
                    'severity': severity
                })

        # Record metrics
        latency_count = category_counts.get('latency_spike', 0)
        error_rate_count = category_counts.get('error_rate_increase', 0)
        traffic_count = category_counts.get('traffic_drop', 0)
        other_count = category_counts.get('other', 0)

        obs.record_result("latency_spikes", latency_count)
        obs.record_result("error_rate_increases", error_rate_count)
        obs.gauge("watchdog.anomalies", total_anomalies)

        # Determine overall status
        if error_rate_count > 0:
            overall_status = "critical"
        elif latency_count > 3 or traffic_count > 3:
            overall_status = "warning"
        else:
            overall_status = "ok"

        # Output
        if args.json:
            output = {
                "status": overall_status,
                "total_anomalies": total_anomalies,
                "duration": args.duration,
                "service": service or "all",
                "query": query,
                "summary": {
                    "latency_spikes": latency_count,
                    "error_rate_increases": error_rate_count,
                    "traffic_drops": traffic_count,
                    "other": other_count
                },
                "affected_services": dict(service_counts),
                "anomalies": anomalies
            }
            print(json.dumps(output, indent=2))
        else:
            # Conversational output
            print(f"Watchdog Anomaly Detection")
            print(f"Duration: {args.duration}")
            if service:
                print(f"Service: {service}")
            print()
            print(f"Found {total_anomalies:,} anomalies")
            print()
            print("Anomaly type breakdown:")
            print(f"  Latency spikes: {latency_count}")
            print(f"  Error rate increases: {error_rate_count}")
            print(f"  Traffic drops: {traffic_count}")
            print(f"  Other: {other_count}")

            if error_rate_count > 0:
                print()
                print(f"CRITICAL: {error_rate_count} error rate increases detected")

            if service_counts:
                print()
                print("Affected services:")
                for service_name, count in service_counts.most_common(5):
                    print(f"  {service_name}: {count} anomalies")

            if anomalies:
                print()
                print("Recent anomalies:")
                for anomaly in anomalies[:5]:
                    print(f"  [{anomaly['severity'].upper()}] {anomaly['title']}")
                    if anomaly['service']:
                        print(f"    Service: {anomaly['service']}")

        obs.log_info(f"Query completed: {total_anomalies} anomalies")
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
