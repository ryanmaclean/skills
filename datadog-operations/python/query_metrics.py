#!/usr/bin/env python3
"""
Query Datadog Metrics API for time series analysis.
Provides statistical analysis, trend detection, and anomaly identification.
"""

import sys
import json
import argparse
from pathlib import Path
from datetime import datetime, timedelta
from typing import Optional, List
import statistics

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


def calculate_percentile(values: List[float], percentile: float) -> float:
    """Calculate percentile from list of values"""
    if not values:
        return 0.0
    sorted_values = sorted(values)
    index = int(len(sorted_values) * percentile)
    return sorted_values[min(index, len(sorted_values) - 1)]


def detect_anomalies(values: List[float], mean: float, stddev: float) -> List[float]:
    """Detect anomalies (values > 2 standard deviations from mean)"""
    upper_threshold = mean + (2 * stddev)
    lower_threshold = mean - (2 * stddev)

    return [v for v in values if v > upper_threshold or v < lower_threshold]


def analyze_trend(values: List[float]) -> dict:
    """Analyze trend by comparing first half vs second half"""
    if len(values) < 2:
        return {
            'direction': 'stable',
            'change_percent': 0.0,
            'status': 'normal'
        }

    midpoint = len(values) // 2
    first_half = values[:midpoint]
    second_half = values[midpoint:]

    first_avg = statistics.mean(first_half) if first_half else 0
    second_avg = statistics.mean(second_half) if second_half else 0

    if first_avg == 0:
        change_pct = 0.0
    else:
        change_pct = ((second_avg - first_avg) / first_avg) * 100

    if change_pct > 10:
        direction = 'increasing'
        status = 'warning'
    elif change_pct < -10:
        direction = 'decreasing'
        status = 'improving'
    else:
        direction = 'stable'
        status = 'normal'

    return {
        'direction': direction,
        'change_percent': round(change_pct, 2),
        'status': status,
        'first_half_avg': round(first_avg, 2),
        'second_half_avg': round(second_avg, 2)
    }


def main():
    obs = init_observability("query-metrics")

    parser = argparse.ArgumentParser(
        description="Query Datadog Metrics for time series analysis"
    )
    parser.add_argument(
        "--metric",
        required=True,
        help="Metric name (e.g., system.cpu.user, trace.express.request.duration)"
    )
    parser.add_argument(
        "--service",
        help="Filter by service name (auto-detected if not provided)"
    )
    parser.add_argument(
        "--duration",
        default="1h",
        help="Time range: 1h, 24h, 7d, 30d (default: 1h)"
    )
    parser.add_argument(
        "--aggregation",
        choices=["avg", "sum", "min", "max"],
        default="avg",
        help="Aggregation function (default: avg)"
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

        # Build query
        with obs.span("build_query"):
            query = f"{args.aggregation}:{args.metric}"
            if service:
                query += f"{{service:{service}}}"
            obs.log_info(f"Metric query: {query}")

        # Parse duration
        with obs.span("parse_duration"):
            duration = parse_duration(args.duration)
            to_time = datetime.now()
            from_time = to_time - duration

        # Create client
        with obs.span("create_client"):
            client = create_client()

        # Query metrics
        with obs.span("query_metrics", tags={"metric": args.metric, "duration": args.duration}):
            start = datetime.now()

            data = client.query_metrics(
                query=query,
                from_time=from_time,
                to_time=to_time
            )

            api_duration = (datetime.now() - start).total_seconds() * 1000
            obs.record_api_call("/api/v1/query", "GET", 200, api_duration)

        # Parse results
        with obs.span("parse_results"):
            series = data.get('series', [])

            if not series:
                obs.log_warning("No metric data found")
                obs.record_result("datapoints", 0)

                output = {
                    "status": "no_data",
                    "metric": args.metric,
                    "service": service or "all",
                    "duration": args.duration,
                    "message": "No data points found for the specified metric and time range"
                }

                if args.json:
                    print(json.dumps(output, indent=2))
                else:
                    print(f"No data found for metric: {args.metric}")

                finalize_observability(0)
                sys.exit(0)

            # Extract all data points
            all_values = []
            time_series = []

            for s in series:
                pointlist = s.get('pointlist', [])
                for point in pointlist:
                    if point and len(point) == 2:
                        timestamp, value = point
                        if value is not None:
                            all_values.append(value)
                            time_series.append({
                                'timestamp': int(timestamp),
                                'value': round(value, 2)
                            })

            if not all_values:
                obs.log_warning("No data points in time series")
                obs.record_result("datapoints", 0)

                output = {
                    "status": "no_data",
                    "metric": args.metric,
                    "service": service or "all",
                    "duration": args.duration,
                    "message": "Time series contains no data points"
                }

                if args.json:
                    print(json.dumps(output, indent=2))
                else:
                    print(f"Time series contains no data points")

                finalize_observability(0)
                sys.exit(0)

        # Calculate statistics
        with obs.span("calculate_stats"):
            point_count = len(all_values)
            min_val = min(all_values)
            max_val = max(all_values)
            avg_val = statistics.mean(all_values)
            p50 = calculate_percentile(all_values, 0.50)
            p95 = calculate_percentile(all_values, 0.95)
            p99 = calculate_percentile(all_values, 0.99)
            stddev = statistics.stdev(all_values) if len(all_values) > 1 else 0.0

            obs.record_result("datapoints", point_count)
            obs.gauge("metrics.avg_value", avg_val)

        # Trend analysis
        with obs.span("analyze_trend"):
            trend = analyze_trend(all_values)

        # Anomaly detection
        with obs.span("detect_anomalies"):
            anomalies = detect_anomalies(all_values, avg_val, stddev)
            anomaly_count = len(anomalies)
            anomaly_pct = (anomaly_count / point_count * 100) if point_count > 0 else 0

            obs.record_result("anomalies", anomaly_count)

            upper_threshold = avg_val + (2 * stddev)
            lower_threshold = avg_val - (2 * stddev)

            if anomaly_count > 0:
                anomaly_status = "detected"
            else:
                anomaly_status = "none"

        # Determine overall status
        if anomaly_count > (point_count / 10):
            overall_status = "critical"
        elif anomaly_count > 0 or trend['direction'] == 'increasing':
            overall_status = "warning"
        else:
            overall_status = "ok"

        # Get metadata
        unit = series[0].get('unit', 'unknown') if series else 'unknown'
        scope = series[0].get('scope', 'unknown') if series else 'unknown'

        # Output
        if args.json:
            output = {
                "status": overall_status,
                "metadata": {
                    "metric": args.metric,
                    "service": service or "all",
                    "duration": args.duration,
                    "aggregation": args.aggregation,
                    "unit": unit,
                    "scope": scope,
                    "query": query
                },
                "statistics": {
                    "count": point_count,
                    "min": round(min_val, 2),
                    "max": round(max_val, 2),
                    "avg": round(avg_val, 2),
                    "p50": round(p50, 2),
                    "p95": round(p95, 2),
                    "p99": round(p99, 2),
                    "stddev": round(stddev, 2)
                },
                "trend": trend,
                "anomalies": {
                    "status": anomaly_status,
                    "count": anomaly_count,
                    "percentage": round(anomaly_pct, 2),
                    "threshold_lower": round(lower_threshold, 2),
                    "threshold_upper": round(upper_threshold, 2),
                    "detected_values": [round(v, 2) for v in anomalies[:10]]
                },
                "time_series": time_series[-100:]  # Last 100 points
            }
            print(json.dumps(output, indent=2))
        else:
            # Conversational output
            print(f"Metric Analysis: {args.metric}")
            print(f"Duration: {args.duration}")
            if service:
                print(f"Service: {service}")
            print()
            print(f"Data points: {point_count:,}")
            print()
            print("Statistics:")
            print(f"  Min: {min_val:.2f}")
            print(f"  Max: {max_val:.2f}")
            print(f"  Avg: {avg_val:.2f}")
            print(f"  P50: {p50:.2f}")
            print(f"  P95: {p95:.2f}")
            print(f"  P99: {p99:.2f}")
            print(f"  StdDev: {stddev:.2f}")
            print()
            print("Trend Analysis:")
            print(f"  Direction: {trend['direction']}")
            print(f"  Change: {trend['change_percent']}%")
            print(f"  Status: {trend['status']}")
            print()
            print("Anomaly Detection:")
            print(f"  Status: {anomaly_status}")
            print(f"  Count: {anomaly_count} ({anomaly_pct:.1f}%)")
            print(f"  Threshold: {lower_threshold:.2f} to {upper_threshold:.2f}")

            if anomaly_count > 0:
                print()
                print(f"WARNING: {anomaly_count} anomalous data points detected")

        obs.log_info(f"Query completed: {point_count} data points, {anomaly_count} anomalies")
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
