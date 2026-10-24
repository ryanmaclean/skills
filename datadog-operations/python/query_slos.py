#!/usr/bin/env python3
"""
Query Datadog SLOs (Service Level Objectives).
Check SLO status and error budgets with built-in observability.
"""

import sys
import json
import argparse
from pathlib import Path
from typing import Optional, List, Dict, Any

# Add lib to path
sys.path.insert(0, str(Path(__file__).parent / "lib"))

from dd_observability import init_observability, finalize_observability
from datadog_client import create_client
from context_detector import detect_context


def calculate_status(slo_value: float, target: float, warning: float) -> str:
    """Calculate SLO status based on thresholds"""
    if slo_value < target:
        return "breaching"
    elif slo_value < warning:
        return "warning"
    else:
        return "ok"


def calculate_budget_status(error_budget_remaining: float) -> str:
    """Calculate error budget status"""
    if error_budget_remaining <= 0:
        return "exhausted"
    elif error_budget_remaining < 20:
        return "low"
    else:
        return "healthy"


def main():
    obs = init_observability("query-slos")

    parser = argparse.ArgumentParser(
        description="Query Datadog SLOs"
    )
    parser.add_argument(
        "--service",
        help="Filter by service name (auto-detected if not provided)"
    )
    parser.add_argument(
        "--tag",
        help="Filter by tag (e.g., team:backend)"
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

        # Create client
        with obs.span("create_client"):
            client = create_client()

        # Build tags for query
        tags = []
        if args.tag:
            tags.append(args.tag)

        # Query SLOs
        with obs.span("query_slos", tags={"service": service or "all"}):
            start_time = sys.modules['datetime'].datetime.now()

            slos = client.get_slos(tags=tags if tags else None)

            api_duration = (sys.modules['datetime'].datetime.now() - start_time).total_seconds() * 1000
            obs.record_api_call("/api/v1/slo", "GET", 200, api_duration)

        # Filter by service if specified
        if service:
            with obs.span("filter_by_service"):
                filtered_slos = []
                for slo in slos:
                    slo_tags = slo.get('tags', [])
                    if any(f"service:{service}" in tag for tag in slo_tags):
                        filtered_slos.append(slo)
                slos = filtered_slos

        # Parse results
        with obs.span("parse_results"):
            total_slos = len(slos)
            obs.log_info(f"Found {total_slos} SLOs")
            obs.record_result("slos", total_slos)

            if total_slos == 0:
                output = {
                    "status": "ok",
                    "total_slos": 0,
                    "slos": []
                }

                if args.json:
                    print(json.dumps(output, indent=2))
                else:
                    print("No SLOs found for the specified criteria")

                finalize_observability(0)
                sys.exit(0)

            # Calculate status breakdown
            breaching_count = 0
            warning_count = 0
            ok_count = 0
            budget_exhausted = 0
            budget_low = 0

            formatted_slos = []

            for slo in slos:
                slo_value = slo.get('slo_value', 0.0) or 0.0
                target = slo.get('target_threshold', 0.0) or 0.0
                warning_threshold = slo.get('warning_threshold', 0.0) or 0.0
                error_budget = slo.get('error_budget_remaining', 0.0) or 0.0

                status = calculate_status(slo_value, target, warning_threshold)
                budget_status = calculate_budget_status(error_budget)

                if status == "breaching":
                    breaching_count += 1
                elif status == "warning":
                    warning_count += 1
                else:
                    ok_count += 1

                if budget_status == "exhausted":
                    budget_exhausted += 1
                elif budget_status == "low":
                    budget_low += 1

                formatted_slos.append({
                    'id': slo.get('id'),
                    'name': slo.get('name'),
                    'type': slo.get('type'),
                    'current_value': round(slo_value * 100, 2) if slo_value else 0,
                    'target': round(target * 100, 2) if target else 0,
                    'warning': round(warning_threshold * 100, 2) if warning_threshold else 0,
                    'error_budget_remaining': round(error_budget, 0) if error_budget else 0,
                    'timeframe': slo.get('timeframe'),
                    'tags': slo.get('tags', []),
                    'status': status,
                    'budget_status': budget_status
                })

        # Record metrics
        obs.record_result("breaching_slos", breaching_count)
        obs.record_result("budget_exhausted", budget_exhausted)
        obs.gauge("slos.breaching", breaching_count)
        obs.gauge("slos.budget_exhausted", budget_exhausted)

        # Determine overall status
        if breaching_count > 0:
            overall_status = "breaching"
        elif warning_count > 0:
            overall_status = "warning"
        else:
            overall_status = "ok"

        # Output
        if args.json:
            output = {
                "status": overall_status,
                "total_slos": total_slos,
                "summary": {
                    "breaching": breaching_count,
                    "warning": warning_count,
                    "ok": ok_count,
                    "budget_exhausted": budget_exhausted,
                    "budget_low": budget_low
                },
                "slos": formatted_slos
            }
            print(json.dumps(output, indent=2))
        else:
            # Conversational output
            print(f"SLO Status Report")
            if service:
                print(f"Service: {service}")
            print()
            print(f"Found {total_slos:,} SLOs")
            print()
            print("Status breakdown:")
            print(f"  Breaching: {breaching_count}")
            print(f"  Warning: {warning_count}")
            print(f"  OK: {ok_count}")
            print()
            print("Error budget status:")
            print(f"  Exhausted: {budget_exhausted}")
            print(f"  Low (<20%): {budget_low}")

            if breaching_count > 0:
                print()
                print(f"ALERT: {breaching_count} SLO(s) breaching target")
                print()
                print("Breaching SLOs:")
                for slo in formatted_slos:
                    if slo['status'] == 'breaching':
                        print(f"  {slo['name']}")
                        print(f"    Current: {slo['current_value']}% | Target: {slo['target']}%")
                        print(f"    Error budget: {slo['error_budget_remaining']}%")

            if budget_exhausted > 0:
                print()
                print(f"WARNING: {budget_exhausted} SLO(s) have exhausted error budget")

        obs.log_info(f"Query completed: {total_slos} SLOs, {breaching_count} breaching")
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
