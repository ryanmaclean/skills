#!/usr/bin/env python3
"""
Analyze LLM observability data for GenAI applications.
Query token usage, costs, latency, and errors with built-in observability.
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
from context_detector import detect_context


# Model pricing per 1K tokens (January 2026)
MODEL_PRICING = {
    'gpt-4': {'input': 0.03, 'output': 0.06},
    'gpt-4-32k': {'input': 0.06, 'output': 0.12},
    'gpt-4-turbo': {'input': 0.01, 'output': 0.03},
    'gpt-3.5-turbo': {'input': 0.0015, 'output': 0.002},
    'claude-3-5-sonnet': {'input': 0.003, 'output': 0.015},
    'claude-3-opus': {'input': 0.015, 'output': 0.075},
    'claude-3-sonnet': {'input': 0.003, 'output': 0.015},
    'claude-3-haiku': {'input': 0.00025, 'output': 0.00125},
    'claude-2.1': {'input': 0.008, 'output': 0.024},
    'gemini-pro': {'input': 0.00025, 'output': 0.00125},
    'gemini-ultra': {'input': 0.005, 'output': 0.015},
}

DEFAULT_PRICING = {'input': 0.03, 'output': 0.06}


def parse_duration(duration: str) -> timedelta:
    """Parse duration string like '1h', '24h', '7d', '30d'"""
    if duration.endswith('h'):
        return timedelta(hours=int(duration[:-1]))
    elif duration.endswith('d'):
        return timedelta(days=int(duration[:-1]))
    else:
        raise ValueError(f"Invalid duration: {duration}. Use format like '1h', '24h', '7d', '30d'")


def calculate_cost(prompt_tokens: float, completion_tokens: float, model: str = None) -> float:
    """Calculate estimated cost based on token counts"""
    pricing = MODEL_PRICING.get(model, DEFAULT_PRICING)

    prompt_cost = (prompt_tokens / 1000) * pricing['input']
    completion_cost = (completion_tokens / 1000) * pricing['output']

    return prompt_cost + completion_cost


def generate_recommendations(summary: Dict) -> List[Dict]:
    """Generate cost optimization recommendations"""
    recommendations = []

    avg_tokens = summary.get('avg_tokens_per_request', 0)
    if avg_tokens > 4000:
        recommendations.append({
            'type': 'high_token_usage',
            'priority': 'high',
            'message': f"Average tokens per request is {avg_tokens}. Consider prompt optimization or response truncation."
        })

    avg_latency = summary.get('avg_latency_ms', 0)
    if avg_latency > 3000:
        recommendations.append({
            'type': 'high_latency',
            'priority': 'medium',
            'message': f"Average latency is {avg_latency}ms. Consider using faster models for non-critical operations."
        })

    total_cost = summary.get('total_cost_usd', 0)
    if total_cost > 100:
        recommendations.append({
            'type': 'high_cost',
            'priority': 'high',
            'message': f"Total cost is ${total_cost:.2f}. Review if all operations require premium models."
        })

    error_rate = summary.get('error_rate_percent', 0)
    if error_rate > 5:
        recommendations.append({
            'type': 'high_error_rate',
            'priority': 'critical',
            'message': f"Error rate is {error_rate:.1f}%. Investigate failed requests to avoid wasted token costs."
        })

    return recommendations


def main():
    obs = init_observability("analyze-llm")

    parser = argparse.ArgumentParser(
        description="Analyze LLM observability data for GenAI applications"
    )
    parser.add_argument(
        "--service",
        required=True,
        help="Service name"
    )
    parser.add_argument(
        "--model",
        help="Filter by model name"
    )
    parser.add_argument(
        "--duration",
        default="24h",
        help="Time range: 1h, 24h, 7d, 30d (default: 24h)"
    )
    parser.add_argument(
        "--format",
        choices=["json", "summary"],
        default="json",
        help="Output format (default: json)"
    )

    args = parser.parse_args()

    try:
        obs.log_info(f"Analyzing LLM data for service: {args.service}")

        # Parse duration (for validation and time_range output)
        with obs.span("parse_duration"):
            duration = parse_duration(args.duration)
            to_time = datetime.now()
            from_time = to_time - duration

        # Build query
        with obs.span("build_query"):
            query = f"service:{args.service}"
            if args.model:
                query += f" llm.model:{args.model}"

        # Create client
        with obs.span("create_client"):
            client = DatadogClient()

        # Query 1: Token usage and latency by operation
        with obs.span("query_token_usage"):
            start = datetime.now()

            # Use correct Datadog API v2 format with data.attributes wrapper
            # Uses relative time format (now-1h, now) which is more reliable
            payload = {
                "data": {
                    "type": "aggregate_request",
                    "attributes": {
                        "filter": {
                            "from": f"now-{args.duration}",
                            "to": "now",
                            "query": query
                        },
                        "compute": [
                            {"aggregation": "count"},
                            {"aggregation": "sum", "metric": "@llm.tokens.prompt"},
                            {"aggregation": "sum", "metric": "@llm.tokens.completion"},
                            {"aggregation": "sum", "metric": "@llm.tokens.total"},
                            {"aggregation": "avg", "metric": "@llm.tokens.prompt"},
                            {"aggregation": "avg", "metric": "@llm.tokens.completion"},
                            {"aggregation": "avg", "metric": "@duration"},
                            {"aggregation": "max", "metric": "@duration"}
                        ],
                        "group_by": [
                            {
                                "facet": "resource_name",
                                "limit": 20
                            }
                        ]
                    }
                }
            }

            response = client._request("POST", "/api/v2/spans/analytics/aggregate", json=payload)
            token_data = response.json()

            api_duration = (datetime.now() - start).total_seconds() * 1000
            obs.record_api_call("/api/v2/spans/analytics/aggregate", "POST", 200, api_duration)

        # Check if data exists - handle API v2 response format
        # v2 format: data is array of items with .attributes.by and .attributes.compute
        items = []
        if "data" in token_data:
            if isinstance(token_data["data"], list):
                items = token_data["data"]
            elif isinstance(token_data["data"], dict):
                buckets = token_data["data"].get("attributes", {}).get("buckets", [])
                if not buckets:
                    buckets = token_data["data"].get("buckets", [])
                items = buckets

        if not items:
            obs.log_warning("No LLM trace data found")
            output = {
                "status": "no_data",
                "service": args.service,
                "model_filter": args.model or "all",
                "duration": args.duration,
                "message": "No LLM spans found. Ensure LLM instrumentation is enabled and spans are tagged with llm.* attributes."
            }
            print(json.dumps(output, indent=2))
            finalize_observability(0)
            sys.exit(0)

        # Query 2: Error rates by model
        with obs.span("query_error_rates"):
            start = datetime.now()

            error_payload = {
                "data": {
                    "type": "aggregate_request",
                    "attributes": {
                        "filter": {
                            "from": f"now-{args.duration}",
                            "to": "now",
                            "query": query
                        },
                        "compute": [
                            {"aggregation": "count"},
                            {"aggregation": "cardinality", "metric": "@trace_id"}
                        ],
                        "group_by": [
                            {"facet": "@llm.model", "limit": 20},
                            {"facet": "status", "limit": 10}
                        ]
                    }
                }
            }

            response = client._request("POST", "/api/v2/spans/analytics/aggregate", json=error_payload)
            error_data = response.json()

            api_duration = (datetime.now() - start).total_seconds() * 1000
            obs.record_api_call("/api/v2/spans/analytics/aggregate", "POST", 200, api_duration)

        # Parse operations data
        with obs.span("parse_operations"):
            # Use items already extracted above
            operations = []

            for item in items:
                # v2 format: item.attributes.by and item.attributes.compute
                if "attributes" in item:
                    attrs = item["attributes"]
                    by_data = attrs.get('by', {})
                    computes = attrs.get('compute', {})
                else:
                    # Fallback for bucket format
                    by_data = item.get('by', {})
                    computes = item.get('computes', {})

                operation = by_data.get('resource_name', 'unknown')
                # c0=count, c1=sum prompt, c2=sum completion, c3=sum total,
                # c4=avg prompt, c5=avg completion, c6=avg duration, c7=max duration
                request_count = computes.get('c0', 0)
                total_prompt = int(computes.get('c1', 0))
                total_completion = int(computes.get('c2', 0))
                total_tokens = int(computes.get('c3', 0))
                avg_prompt = int(computes.get('c4', 0))
                avg_completion = int(computes.get('c5', 0))
                avg_duration_ns = computes.get('c6', 0)
                max_duration_ns = computes.get('c7', 0)

                # Calculate cost
                estimated_cost = calculate_cost(total_prompt, total_completion, args.model)
                cost_per_request = estimated_cost / request_count if request_count > 0 else 0

                operations.append({
                    'operation': operation,
                    'request_count': request_count,
                    'total_prompt_tokens': total_prompt,
                    'total_completion_tokens': total_completion,
                    'total_tokens': total_tokens,
                    'avg_prompt_tokens': avg_prompt,
                    'avg_completion_tokens': avg_completion,
                    'avg_ms': int(avg_duration_ns / 1_000_000) if avg_duration_ns else 0,
                    'max_ms': int(max_duration_ns / 1_000_000) if max_duration_ns else 0,
                    'estimated_cost_usd': round(estimated_cost, 2),
                    'cost_per_request_usd': round(cost_per_request, 3)
                })

        # Parse error data
        with obs.span("parse_errors"):
            # Handle API v2 response format
            error_items = []
            if "data" in error_data:
                if isinstance(error_data["data"], list):
                    error_items = error_data["data"]
                elif isinstance(error_data["data"], dict):
                    error_buckets = error_data["data"].get("attributes", {}).get("buckets", [])
                    if not error_buckets:
                        error_buckets = error_data["data"].get("buckets", [])
                    error_items = error_buckets

            # Group by model
            models_map = {}
            for item in error_items:
                # v2 format: item.attributes.by and item.attributes.compute
                if "attributes" in item:
                    attrs = item["attributes"]
                    by_data = attrs.get('by', {})
                    computes = attrs.get('compute', {})
                else:
                    by_data = item.get('by', {})
                    computes = item.get('computes', {})

                model_name = by_data.get('@llm.model', by_data.get('llm_model', 'unknown'))
                status = by_data.get('status', 'ok')
                count = computes.get('c0', 0)

                if model_name not in models_map:
                    models_map[model_name] = {'total': 0, 'errors': 0}

                models_map[model_name]['total'] += count
                if status == 'error':
                    models_map[model_name]['errors'] += count

            models = []
            for model_name, counts in models_map.items():
                error_rate = (counts['errors'] / counts['total'] * 100) if counts['total'] > 0 else 0
                models.append({
                    'model': model_name,
                    'total_requests': counts['total'],
                    'error_count': counts['errors'],
                    'error_rate': round(error_rate, 1)
                })

        # Calculate summary
        with obs.span("calculate_summary"):
            total_requests = sum(op['request_count'] for op in operations)
            total_tokens = sum(op['total_tokens'] for op in operations)
            total_prompt = sum(op['total_prompt_tokens'] for op in operations)
            total_completion = sum(op['total_completion_tokens'] for op in operations)
            total_cost = sum(op['estimated_cost_usd'] for op in operations)
            avg_cost_per_request = total_cost / total_requests if total_requests > 0 else 0
            avg_tokens_per_request = total_tokens // total_requests if total_requests > 0 else 0
            avg_latency = sum(op['avg_ms'] for op in operations) // len(operations) if operations else 0

            total_errors = sum(m['error_count'] for m in models)
            overall_error_rate = (total_errors / total_requests * 100) if total_requests > 0 else 0

            high_cost_ops = len([op for op in operations if op['estimated_cost_usd'] > 1])
            slow_ops = len([op for op in operations if op['avg_ms'] > 2000])

            summary = {
                'total_requests': total_requests,
                'total_tokens': total_tokens,
                'total_prompt_tokens': total_prompt,
                'total_completion_tokens': total_completion,
                'avg_tokens_per_request': avg_tokens_per_request,
                'total_cost_usd': round(total_cost, 2),
                'avg_cost_per_request_usd': round(avg_cost_per_request, 3),
                'avg_latency_ms': avg_latency,
                'error_rate_percent': round(overall_error_rate, 2),
                'high_cost_operations_count': high_cost_ops,
                'slow_operations_count': slow_ops
            }

        # Record metrics
        obs.record_result("llm_requests", total_requests)
        obs.record_result("llm_tokens", total_tokens)
        obs.gauge("llm.total_cost", total_cost)
        obs.gauge("llm.error_rate", overall_error_rate)

        # Generate recommendations
        with obs.span("generate_recommendations"):
            recommendations = generate_recommendations(summary)

        # Output
        if args.format == "summary":
            print(f"LLM Observability Analysis - {args.service}")
            print(f"Duration: {args.duration}")
            print()
            print("Token Usage:")
            print(f"  Total tokens: {total_tokens:,}")
            print(f"  Prompt tokens: {total_prompt:,}")
            print(f"  Completion tokens: {total_completion:,}")
            print(f"  Avg per request: {avg_tokens_per_request:,}")
            print()
            print("Cost Analysis:")
            print(f"  Total cost: ${total_cost:.2f}")
            print(f"  Avg per request: ${avg_cost_per_request:.3f}")
            print(f"  High-cost ops: {high_cost_ops}")
            print()
            print("Performance:")
            print(f"  Total requests: {total_requests:,}")
            print(f"  Avg latency: {avg_latency}ms")
            print(f"  Slow operations: {slow_ops}")
            print(f"  Error rate: {overall_error_rate:.2f}%")

            if recommendations:
                print()
                print("Recommendations:")
                for rec in recommendations:
                    print(f"  [{rec['priority'].upper()}] {rec['message']}")
        else:
            output = {
                "status": "ok",
                "service": args.service,
                "model_filter": args.model or "all",
                "duration": args.duration,
                "time_range": {
                    "from": int(from_time.timestamp()),
                    "to": int(to_time.timestamp())
                },
                "summary": summary,
                "operations": operations,
                "models": models,
                "optimization_suggestions": recommendations
            }
            print(json.dumps(output, indent=2))

        obs.log_info(f"Analysis completed: {total_requests} requests, ${total_cost:.2f} cost")
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
