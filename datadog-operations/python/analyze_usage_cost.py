#!/usr/bin/env python3
"""
Analyze Datadog usage and costs for FinOps optimization.
Queries usage/billing APIs with built-in observability.
"""

import sys
import json
import argparse
from pathlib import Path
from datetime import datetime, timedelta
from typing import Dict, List, Optional

# Add lib to path
sys.path.insert(0, str(Path(__file__).parent / "lib"))

from dd_observability import init_observability, finalize_observability
from datadog_client import DatadogClient


def parse_duration(duration: str) -> int:
    """Parse duration to days"""
    if duration == "7d":
        return 7
    elif duration == "30d":
        return 30
    elif duration == "90d":
        return 90
    else:
        raise ValueError(f"Invalid duration: {duration}. Use: 7d, 30d, or 90d")


def calculate_date_range(days_ago: int) -> tuple:
    """Calculate start and end dates for usage API"""
    end_date = datetime.now().date()
    start_date = end_date - timedelta(days=days_ago)
    return start_date.strftime("%Y-%m-%d"), end_date.strftime("%Y-%m-%d")


def query_usage_endpoint(client: DatadogClient, endpoint: str) -> Dict:
    """Query usage API endpoint with error handling"""
    try:
        response = client._request("GET", endpoint)
        return response.json()
    except Exception:
        return {}


def generate_recommendations(apm_data: Dict, logs_data: Dict, infra_data: Dict, metrics_data: Dict) -> List[Dict]:
    """Generate cost optimization recommendations"""
    recommendations = []

    # APM recommendations
    indexed_spans = apm_data.get('indexed_spans', 0)
    ingested_spans = apm_data.get('ingested_spans_bytes', 0)
    apm_cost = apm_data.get('estimated_cost_usd', {}).get('total', 0)

    if ingested_spans > 0 and indexed_spans > 0:
        retention_rate = (indexed_spans / (ingested_spans / 1000))
        if retention_rate > 0.15:
            recommendations.append({
                'category': 'apm',
                'priority': 'high',
                'issue': 'High APM span retention rate',
                'detail': f'Retaining {retention_rate*100:.1f}% of ingested spans. Industry standard is 10-15%.',
                'recommendation': 'Review tag-based retention filters to reduce indexed spans. Focus on high-value traces (errors, slow requests) and sample normal traffic.',
                'potential_savings_usd': round(apm_cost * 0.30, 2)
            })

    avg_hosts = apm_data.get('avg_hosts', 0)
    if avg_hosts > 10:
        recommendations.append({
            'category': 'apm',
            'priority': 'medium',
            'issue': f'APM enabled on {avg_hosts} hosts',
            'detail': 'Verify all hosts require APM monitoring.',
            'recommendation': 'Disable APM on non-production hosts, batch jobs, and internal services.',
            'potential_savings_usd': round(apm_cost * 0.20, 2)
        })

    # Logs recommendations
    ingested_gb = logs_data.get('ingested_gb', 0)
    logs_cost = logs_data.get('estimated_cost_usd', {}).get('total', 0)

    if ingested_gb > 100:
        recommendations.append({
            'category': 'logs',
            'priority': 'high',
            'issue': 'High log ingestion volume',
            'detail': f'Ingesting {ingested_gb:.1f}GB of logs.',
            'recommendation': 'Implement log filtering at source. Exclude debug logs, health checks, and high-frequency events.',
            'potential_savings_usd': round(logs_cost * 0.40, 2)
        })

    indexed_bytes = logs_data.get('indexed_bytes', 0)
    ingested_bytes = logs_data.get('ingested_bytes', 0)
    if indexed_bytes > 0 and ingested_bytes > 0:
        indexed_ratio = indexed_bytes / ingested_bytes
        if indexed_ratio > 0.30:
            recommendations.append({
                'category': 'logs',
                'priority': 'medium',
                'issue': 'High log indexing rate',
                'detail': f'Indexing {indexed_ratio*100:.1f}% of ingested logs.',
                'recommendation': 'Review log indexing rules. Index only logs needed for search/alerting.',
                'potential_savings_usd': round(logs_cost * 0.25, 2)
            })

    # Infrastructure recommendations
    avg_containers = infra_data.get('avg_containers', 0)
    infra_cost = infra_data.get('estimated_cost_usd', {}).get('total', 0)

    if avg_containers > 50:
        recommendations.append({
            'category': 'infrastructure',
            'priority': 'medium',
            'issue': 'High container count',
            'detail': f'Monitoring {avg_containers} containers on average.',
            'recommendation': 'Exclude ephemeral and test containers. Use container exclusion rules.',
            'potential_savings_usd': round(infra_cost * 0.15, 2)
        })

    # Custom metrics recommendations
    custom_metrics = metrics_data.get('avg_custom_metrics', 0)
    metrics_cost = metrics_data.get('estimated_cost_usd', 0)

    if custom_metrics > 100:
        recommendations.append({
            'category': 'metrics',
            'priority': 'low',
            'issue': 'High custom metrics count',
            'detail': f'Using {custom_metrics} custom metrics.',
            'recommendation': 'Audit custom metrics for unused or redundant metrics. Use metric tags instead of separate metrics.',
            'potential_savings_usd': round(metrics_cost * 0.10, 2)
        })

    # General recommendation
    recommendations.append({
        'category': 'general',
        'priority': 'info',
        'issue': 'Cost visibility',
        'detail': 'Regular cost analysis enables proactive optimization.',
        'recommendation': 'Schedule weekly cost reviews. Set up usage alerts for anomaly detection.',
        'potential_savings_usd': 0
    })

    return recommendations


def main():
    obs = init_observability("analyze-usage-cost")

    parser = argparse.ArgumentParser(
        description="Analyze Datadog usage and costs for FinOps"
    )
    parser.add_argument(
        "--duration",
        choices=["7d", "30d", "90d"],
        default="30d",
        help="Analysis duration (default: 30d)"
    )
    parser.add_argument(
        "--product",
        choices=["apm", "logs", "infrastructure", "all"],
        default="all",
        help="Product filter (default: all)"
    )
    parser.add_argument(
        "--json",
        action="store_true",
        help="Output as JSON"
    )

    args = parser.parse_args()

    try:
        obs.log_info(f"Analyzing Datadog usage and costs for {args.duration}")

        # Calculate date range
        with obs.span("calculate_dates"):
            days_ago = parse_duration(args.duration)
            start_date, end_date = calculate_date_range(days_ago)
            obs.log_info(f"Date range: {start_date} to {end_date}")

        # Create client
        with obs.span("create_client"):
            client = DatadogClient()

        # Initialize data structures
        apm_data = {}
        logs_data = {}
        infra_data = {}
        metrics_data = {}

        # Query APM usage
        if args.product in ["all", "apm"]:
            with obs.span("query_apm_usage"):
                start = datetime.now()

                # APM traces
                traces_response = query_usage_endpoint(
                    client,
                    f"/api/v1/usage/traces?start_hr={start_date}T00&end_hr={end_date}T00"
                )

                api_duration = (datetime.now() - start).total_seconds() * 1000
                obs.record_api_call("/api/v1/usage/traces", "GET", 200, api_duration)

                # Initialize variables with defaults
                indexed_spans = 0
                span_cost = 0.0

                # Calculate APM metrics
                if traces_response.get('usage'):
                    usage = traces_response['usage']
                    ingested_bytes = sum(u.get('ingested_events_bytes', 0) for u in usage)
                    indexed_spans = sum(u.get('indexed_events_count', 0) for u in usage)
                    ingested_gb = ingested_bytes / (1024 ** 3)

                    # Estimate costs (2026 pricing: $1.70 per million indexed spans)
                    span_cost = (indexed_spans / 1_000_000) * 1.70

                    apm_data = {
                        'ingested_spans_bytes': ingested_bytes,
                        'ingested_spans_gb': round(ingested_gb, 2),
                        'indexed_spans': indexed_spans,
                        'avg_hosts': 0,
                        'estimated_cost_usd': {
                            'indexed_spans': round(span_cost, 2),
                            'total': round(span_cost, 2)
                        }
                    }

                obs.log_info(f"APM: {indexed_spans:,} indexed spans, est. cost: ${span_cost:.2f}")

        # Query Logs usage
        if args.product in ["all", "logs"]:
            with obs.span("query_logs_usage"):
                start = datetime.now()

                logs_response = query_usage_endpoint(
                    client,
                    f"/api/v1/usage/logs?start_hr={start_date}T00&end_hr={end_date}T00"
                )

                api_duration = (datetime.now() - start).total_seconds() * 1000
                obs.record_api_call("/api/v1/usage/logs", "GET", 200, api_duration)

                # Initialize variables with defaults
                logs_ingested_gb = 0.0
                logs_total_cost = 0.0

                if logs_response.get('usage'):
                    usage = logs_response['usage']
                    ingested_bytes = sum(u.get('ingested_events_bytes', 0) for u in usage)
                    indexed_bytes = sum(u.get('indexed_events_count', 0) for u in usage)
                    logs_ingested_gb = ingested_bytes / (1024 ** 3)
                    indexed_gb = indexed_bytes / (1024 ** 3)

                    # Estimate costs (Ingested: $0.10/GB)
                    ingested_cost = logs_ingested_gb * 0.10
                    indexed_cost = indexed_gb * 0.10
                    logs_total_cost = ingested_cost + indexed_cost

                    logs_data = {
                        'ingested_bytes': ingested_bytes,
                        'ingested_gb': round(logs_ingested_gb, 2),
                        'indexed_bytes': indexed_bytes,
                        'indexed_gb': round(indexed_gb, 2),
                        'estimated_cost_usd': {
                            'ingested': round(ingested_cost, 2),
                            'indexed': round(indexed_cost, 2),
                            'total': round(logs_total_cost, 2)
                        }
                    }

                obs.log_info(f"Logs: {logs_ingested_gb:.1f}GB ingested, est. cost: ${logs_total_cost:.2f}")

        # Query Infrastructure usage
        if args.product in ["all", "infrastructure"]:
            with obs.span("query_infra_usage"):
                start = datetime.now()

                hosts_response = query_usage_endpoint(
                    client,
                    f"/api/v1/usage/hosts?start_hr={start_date}T00&end_hr={end_date}T00"
                )

                api_duration = (datetime.now() - start).total_seconds() * 1000
                obs.record_api_call("/api/v1/usage/hosts", "GET", 200, api_duration)

                # Initialize variables with defaults
                avg_hosts = 0
                avg_containers = 0
                infra_total_cost = 0.0

                if hosts_response.get('usage'):
                    usage = hosts_response['usage']
                    host_counts = [u.get('host_count', 0) for u in usage if u.get('host_count')]
                    container_counts = [u.get('container_count', 0) for u in usage if u.get('container_count')]

                    avg_hosts = sum(host_counts) // len(host_counts) if host_counts else 0
                    avg_containers = sum(container_counts) // len(container_counts) if container_counts else 0

                    # Estimate costs ($15/host/month, $1/container/month)
                    host_cost = avg_hosts * 15
                    container_cost = avg_containers * 1
                    infra_total_cost = host_cost + container_cost

                    infra_data = {
                        'avg_hosts': avg_hosts,
                        'avg_containers': avg_containers,
                        'estimated_cost_usd': {
                            'hosts': round(host_cost, 2),
                            'containers': round(container_cost, 2),
                            'total': round(infra_total_cost, 2)
                        }
                    }

                obs.log_info(f"Infrastructure: {avg_hosts} hosts, {avg_containers} containers, est. cost: ${infra_total_cost:.2f}")

        # Query Custom Metrics usage
        if args.product == "all":
            with obs.span("query_metrics_usage"):
                start = datetime.now()

                metrics_response = query_usage_endpoint(
                    client,
                    f"/api/v1/usage/timeseries?start_hr={start_date}T00&end_hr={end_date}T00"
                )

                api_duration = (datetime.now() - start).total_seconds() * 1000
                obs.record_api_call("/api/v1/usage/timeseries", "GET", 200, api_duration)

                # Initialize variables with defaults
                avg_metrics = 0
                metrics_cost = 0.0

                if metrics_response.get('usage'):
                    usage = metrics_response['usage']
                    metric_counts = [u.get('num_custom_timeseries', 0) for u in usage if u.get('num_custom_timeseries')]
                    avg_metrics = sum(metric_counts) // len(metric_counts) if metric_counts else 0

                    # Estimate cost ($0.05 per metric per month)
                    metrics_cost = avg_metrics * 0.05

                    metrics_data = {
                        'avg_custom_metrics': avg_metrics,
                        'estimated_cost_usd': round(metrics_cost, 2)
                    }

                obs.log_info(f"Custom metrics: {avg_metrics}, est. cost: ${metrics_cost:.2f}")

        # Calculate totals
        with obs.span("calculate_totals"):
            total_cost = 0.0

            if apm_data:
                total_cost += apm_data.get('estimated_cost_usd', {}).get('total', 0)
            if logs_data:
                total_cost += logs_data.get('estimated_cost_usd', {}).get('total', 0)
            if infra_data:
                total_cost += infra_data.get('estimated_cost_usd', {}).get('total', 0)
            if metrics_data:
                total_cost += metrics_data.get('estimated_cost_usd', 0)

            obs.gauge("usage.total_cost", total_cost)

        # Generate recommendations
        with obs.span("generate_recommendations"):
            recommendations = generate_recommendations(apm_data, logs_data, infra_data, metrics_data)
            potential_savings = sum(r.get('potential_savings_usd', 0) for r in recommendations)

        # Record results
        obs.record_result("total_cost", int(total_cost))
        obs.record_result("potential_savings", int(potential_savings))

        # Output
        output = {
            "status": "ok",
            "analysis_period": {
                "start_date": start_date,
                "end_date": end_date,
                "duration": args.duration
            },
            "product_filter": args.product,
            "usage_summary": {
                "apm": apm_data,
                "logs": logs_data,
                "infrastructure": infra_data,
                "custom_metrics": metrics_data
            },
            "cost_summary": {
                "total_estimated_monthly_usd": round(total_cost, 2),
                "potential_savings_usd": round(potential_savings, 2),
                "optimization_opportunity_pct": round((potential_savings / total_cost * 100) if total_cost > 0 else 0, 1)
            },
            "recommendations": recommendations,
            "next_steps": [
                "Review high-priority recommendations for immediate cost reduction",
                "Implement APM sampling for high-volume services",
                "Configure log exclusion filters for noisy patterns",
                "Audit and remove unused custom metrics",
                "Set up cost anomaly alerts in Datadog"
            ]
        }

        if args.json:
            print(json.dumps(output, indent=2))
        else:
            # Summary output
            print(f"Datadog Usage & Cost Analysis")
            print(f"Period: {start_date} to {end_date}")
            print()
            print(f"Total estimated monthly cost: ${total_cost:.2f}")
            print(f"Potential savings: ${potential_savings:.2f} ({(potential_savings/total_cost*100) if total_cost > 0 else 0:.1f}%)")
            print()
            print("Top recommendations:")
            for rec in recommendations[:5]:
                if rec['priority'] in ['high', 'critical']:
                    print(f"  [{rec['priority'].upper()}] {rec['issue']}")
                    print(f"    Potential savings: ${rec['potential_savings_usd']:.2f}")

        obs.log_info(f"Analysis completed: ${total_cost:.2f} cost, ${potential_savings:.2f} potential savings")
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
        obs.log_error(f"Analysis failed: {str(e)}", error_type=type(e).__name__)
        print(f"Error: {e}", file=sys.stderr)
        finalize_observability(1)
        sys.exit(1)


if __name__ == "__main__":
    main()
