#!/usr/bin/env python3
"""
Calculate Error Budget for SLOs.
Determines remaining error budget and time-to-breach estimates.
"""

import sys
import json
import argparse
from pathlib import Path
from datetime import datetime, timedelta
from typing import List, Dict

# Add lib to path
sys.path.insert(0, str(Path(__file__).parent / "lib"))

from dd_observability import init_observability, finalize_observability
from datadog_client import DatadogClient
from context_detector import detect_context


def calculate_budget_status(remaining_errors: int, allowed_errors: int, burn_rate: float) -> tuple:
    """
    Calculate error budget status and time to exhaustion.
    Returns: (status, days_to_exhaustion)
    """
    if remaining_errors < 0:
        return "exhausted", 0.0

    if burn_rate <= 0:
        return "healthy", float('inf')

    days_to_exhaustion = remaining_errors / burn_rate

    if days_to_exhaustion < 7:
        return "critical", days_to_exhaustion
    elif days_to_exhaustion < 14:
        return "warning", days_to_exhaustion
    else:
        return "healthy", days_to_exhaustion


def generate_recommendations(status: str, burn_rate: float, allowed_errors: int, budget_pct: float) -> List[Dict]:
    """Generate recommendations based on error budget status"""
    recommendations = []

    if status == "exhausted":
        recommendations.append({
            'priority': 'critical',
            'action': 'Error budget exhausted - SLO breach. Implement immediate remediation.'
        })
    elif status == "critical":
        recommendations.append({
            'priority': 'critical',
            'action': 'Error budget critically low - less than 7 days remaining. Freeze non-critical deployments.'
        })
    elif status == "warning":
        recommendations.append({
            'priority': 'high',
            'action': 'Error budget below 50% - review error patterns and improve reliability.'
        })

    # Check burn rate
    normal_rate = allowed_errors / 30
    if burn_rate > (normal_rate * 1.5):
        recommendations.append({
            'priority': 'high',
            'action': 'Burn rate 50% above normal - investigate recent changes.'
        })

    if budget_pct < 25 and status != "exhausted":
        recommendations.append({
            'priority': 'medium',
            'action': 'Less than 25% error budget remaining - monitor closely and reduce deployment frequency.'
        })

    return recommendations


def main():
    obs = init_observability("calculate-error-budget")

    parser = argparse.ArgumentParser(
        description="Calculate error budget and time-to-breach estimates"
    )
    parser.add_argument(
        "--service",
        required=True,
        help="Service name"
    )
    parser.add_argument(
        "--slo-target",
        type=float,
        default=99.9,
        help="SLO target percentage (default: 99.9)"
    )
    parser.add_argument(
        "--json",
        action="store_true",
        help="Output as JSON"
    )

    args = parser.parse_args()

    try:
        obs.log_info(f"Calculating error budget for service: {args.service}")

        # Validate SLO target
        if args.slo_target < 0 or args.slo_target > 100:
            raise ValueError("SLO target must be between 0 and 100")

        # Calculate time range (last 30 days)
        with obs.span("calculate_timerange"):
            to_time = datetime.now()
            from_time = to_time - timedelta(days=30)
            from_ns = int(from_time.timestamp() * 1e9)
            to_ns = int(to_time.timestamp() * 1e9)

        # Create client
        with obs.span("create_client"):
            client = DatadogClient()

        # Query APM for request counts and errors
        with obs.span("query_apm_traces"):
            start = datetime.now()

            payload = {
                "filter": {
                    "from": str(from_ns),
                    "to": str(to_ns),
                    "query": f"service:{args.service}"
                },
                "compute": [
                    {"aggregation": "count", "metric": "*"},
                    {"aggregation": "count", "metric": "error"}
                ],
                "group_by": []
            }

            response = client._request("POST", "/api/v2/spans/analytics/aggregate", json=payload)
            data = response.json()

            api_duration = (datetime.now() - start).total_seconds() * 1000
            obs.record_api_call("/api/v2/spans/analytics/aggregate", "POST", 200, api_duration)

        # Parse results
        with obs.span("parse_results"):
            buckets = data.get('data', {}).get('buckets', [])

            if not buckets:
                obs.log_error("No data found for service")
                output = {
                    "status": "no_data",
                    "service": args.service,
                    "message": "No APM data found for this service"
                }

                if args.json:
                    print(json.dumps(output, indent=2))
                else:
                    print(f"Error: No APM data found for service: {args.service}")

                finalize_observability(1)
                sys.exit(1)

            bucket = buckets[0]
            computes = bucket.get('computes', {})

            total_requests = computes.get('c0', 0)
            error_count = computes.get('c1', 0)

            if total_requests == 0:
                obs.log_error("No requests found")
                print(f"Error: No requests found for service: {args.service}", file=sys.stderr)
                finalize_observability(1)
                sys.exit(1)

        # Calculate error rate and success rate
        with obs.span("calculate_rates"):
            current_error_rate = (error_count / total_requests) * 100
            current_success_rate = 100 - current_error_rate

            obs.log_info(f"Total requests: {total_requests:,}, Errors: {error_count:,}")
            obs.log_info(f"Current success rate: {current_success_rate:.4f}%")

        # Calculate error budget
        with obs.span("calculate_budget"):
            error_budget_percent = 100 - args.slo_target
            allowed_errors = int((total_requests * error_budget_percent) / 100)
            remaining_errors = allowed_errors - error_count
            budget_remaining_pct = (remaining_errors / allowed_errors * 100) if allowed_errors > 0 else 0

            # Calculate burn rate (errors per day)
            burn_rate = error_count / 30

            # Calculate status and time to exhaustion
            status, days_to_exhaustion = calculate_budget_status(remaining_errors, allowed_errors, burn_rate)

            obs.record_result("remaining_errors", remaining_errors)
            obs.record_result("burn_rate", int(burn_rate))
            obs.gauge("error_budget.remaining_pct", budget_remaining_pct)

        # Generate recommendations
        with obs.span("generate_recommendations"):
            recommendations = generate_recommendations(
                status, burn_rate, allowed_errors, budget_remaining_pct
            )

        # Output
        output = {
            "status": status,
            "service": args.service,
            "slo_target": args.slo_target,
            "period_days": 30,
            "current_metrics": {
                "total_requests": total_requests,
                "error_count": error_count,
                "success_rate": round(current_success_rate, 4),
                "error_rate": round(current_error_rate, 4)
            },
            "error_budget": {
                "allowed_errors": allowed_errors,
                "consumed_errors": error_count,
                "remaining_errors": remaining_errors,
                "budget_remaining_percent": round(budget_remaining_pct, 2)
            },
            "burn_rate": {
                "errors_per_day": round(burn_rate, 2),
                "days_to_exhaustion": round(days_to_exhaustion, 1) if days_to_exhaustion != float('inf') else "infinite",
                "acceptable_rate": round(allowed_errors / 30, 2)
            },
            "recommendations": recommendations
        }

        if args.json:
            print(json.dumps(output, indent=2))
        else:
            # Conversational output
            print(f"Error Budget Analysis: {args.service}")
            print(f"SLO Target: {args.slo_target}%")
            print()
            print(f"Status: {status.upper()}")
            print()
            print("Current Metrics (30 days):")
            print(f"  Total requests: {total_requests:,}")
            print(f"  Errors: {error_count:,}")
            print(f"  Success rate: {current_success_rate:.4f}%")
            print()
            print("Error Budget:")
            print(f"  Allowed errors: {allowed_errors:,}")
            print(f"  Consumed: {error_count:,}")
            print(f"  Remaining: {remaining_errors:,}")
            print(f"  Budget remaining: {budget_remaining_pct:.2f}%")
            print()
            print("Burn Rate:")
            print(f"  Current: {burn_rate:.2f} errors/day")
            print(f"  Acceptable: {allowed_errors / 30:.2f} errors/day")
            if days_to_exhaustion != float('inf'):
                print(f"  Days to exhaustion: {days_to_exhaustion:.1f}")
            else:
                print(f"  Days to exhaustion: infinite")

            if status in ["critical", "exhausted"]:
                print()
                print(f"ALERT: Error budget is {status}")

            if recommendations:
                print()
                print("Recommendations:")
                for rec in recommendations:
                    print(f"  [{rec['priority'].upper()}] {rec['action']}")

        obs.log_info(f"Calculation completed: {status}, {budget_remaining_pct:.1f}% remaining")
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
        obs.log_error(f"Calculation failed: {str(e)}", error_type=type(e).__name__)
        print(f"Error: {e}", file=sys.stderr)
        finalize_observability(1)
        sys.exit(1)


if __name__ == "__main__":
    main()
