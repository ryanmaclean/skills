#!/usr/bin/env python3
"""
Create Datadog Dashboards - Generate dashboards from pre-built templates.
Supports APM, security, cost, and LLM observability dashboards.
"""

import sys
import json
import argparse
from pathlib import Path
from typing import Optional, Dict, Any
from datetime import datetime

# Add lib to path
sys.path.insert(0, str(Path(__file__).parent / "lib"))

from dd_observability import init_observability, finalize_observability
from datadog_client import create_client


def build_apm_dashboard(title: str, service_filter: str) -> Dict[str, Any]:
    """Build APM performance dashboard"""
    return {
        "title": title,
        "description": f"APM performance metrics for {service_filter or 'all services'}",
        "widgets": [
            {
                "definition": {
                    "type": "timeseries",
                    "requests": [{
                        "q": f"avg:trace.express.request.duration{{{service_filter}}} by {{resource_name}}",
                        "display_type": "line"
                    }],
                    "title": "Request Latency by Endpoint"
                }
            },
            {
                "definition": {
                    "type": "query_value",
                    "requests": [{
                        "q": f"avg:trace.express.request.duration{{{service_filter}}}",
                        "aggregator": "avg"
                    }],
                    "title": "Average Latency (ms)",
                    "precision": 2
                }
            },
            {
                "definition": {
                    "type": "timeseries",
                    "requests": [{
                        "q": f"sum:trace.express.request.errors{{{service_filter}}}.as_count()",
                        "display_type": "bars"
                    }],
                    "title": "Error Count"
                }
            },
            {
                "definition": {
                    "type": "timeseries",
                    "requests": [{
                        "q": f"sum:trace.express.request.hits{{{service_filter}}}.as_count()",
                        "display_type": "area"
                    }],
                    "title": "Request Throughput"
                }
            },
            {
                "definition": {
                    "type": "toplist",
                    "requests": [{
                        "q": f"top(avg:trace.express.request.duration{{{service_filter}}} by {{resource_name}}, 10, 'mean', 'desc')"
                    }],
                    "title": "Slowest Endpoints"
                }
            },
            {
                "definition": {
                    "type": "query_value",
                    "requests": [{
                        "q": f"sum:trace.express.request.errors{{{service_filter}}}.as_count() / sum:trace.express.request.hits{{{service_filter}}}.as_count() * 100",
                        "aggregator": "avg"
                    }],
                    "title": "Error Rate (%)",
                    "precision": 2
                }
            }
        ],
        "layout_type": "ordered"
    }


def build_logs_dashboard(title: str, service_filter: str) -> Dict[str, Any]:
    """Build logs analysis dashboard"""
    return {
        "title": title,
        "description": f"Log analysis for {service_filter or 'all services'}",
        "widgets": [
            {
                "definition": {
                    "type": "timeseries",
                    "requests": [{
                        "q": f"sum:datadog.estimated_usage.logs.ingested_bytes{{{service_filter}}}.as_count()",
                        "display_type": "bars"
                    }],
                    "title": "Log Volume (Bytes)"
                }
            },
            {
                "definition": {
                    "type": "log_stream",
                    "query": f"status:error {service_filter}",
                    "columns": ["host", "service", "message"],
                    "title": "Recent Errors"
                }
            },
            {
                "definition": {
                    "type": "toplist",
                    "requests": [{
                        "q": f"top(count:logs{{{service_filter}}} by {{status}}, 10, 'sum', 'desc')"
                    }],
                    "title": "Log Status Breakdown"
                }
            }
        ],
        "layout_type": "ordered"
    }


def build_security_dashboard(title: str, service_filter: str) -> Dict[str, Any]:
    """Build security monitoring dashboard"""
    return {
        "title": title,
        "description": f"Security monitoring for {service_filter or 'all services'}",
        "widgets": [
            {
                "definition": {
                    "type": "query_value",
                    "requests": [{
                        "q": f"sum:datadog.security.appsec.threat{{{service_filter}}}.as_count()",
                        "aggregator": "sum"
                    }],
                    "title": "Security Threats (24h)",
                    "precision": 0
                }
            },
            {
                "definition": {
                    "type": "timeseries",
                    "requests": [{
                        "q": f"sum:datadog.security.appsec.threat{{{service_filter}}} by {{attack_type}}.as_count()",
                        "display_type": "bars"
                    }],
                    "title": "Threats by Type"
                }
            },
            {
                "definition": {
                    "type": "toplist",
                    "requests": [{
                        "q": f"top(sum:datadog.security.appsec.threat{{{service_filter}}} by {{http.client_ip}}.as_count(), 10, 'sum', 'desc')"
                    }],
                    "title": "Top Attack Sources (IP)"
                }
            },
            {
                "definition": {
                    "type": "log_stream",
                    "query": f"source:security {service_filter} severity:high",
                    "columns": ["severity", "message", "source"],
                    "title": "High Severity Events"
                }
            }
        ],
        "layout_type": "ordered"
    }


def build_cost_dashboard(title: str) -> Dict[str, Any]:
    """Build cost analysis dashboard"""
    return {
        "title": title,
        "description": "Datadog usage and cost analysis",
        "widgets": [
            {
                "definition": {
                    "type": "timeseries",
                    "requests": [{
                        "q": "sum:datadog.estimated_usage.apm.ingested_spans{*}.as_count()",
                        "display_type": "area"
                    }],
                    "title": "APM Spans Ingested"
                }
            },
            {
                "definition": {
                    "type": "timeseries",
                    "requests": [{
                        "q": "sum:datadog.estimated_usage.logs.ingested_bytes{*}.as_count()",
                        "display_type": "area"
                    }],
                    "title": "Log Volume Ingested"
                }
            },
            {
                "definition": {
                    "type": "query_value",
                    "requests": [{
                        "q": "avg:datadog.estimated_usage.hosts{*}",
                        "aggregator": "avg"
                    }],
                    "title": "Average Hosts",
                    "precision": 0
                }
            },
            {
                "definition": {
                    "type": "toplist",
                    "requests": [{
                        "q": "top(sum:datadog.estimated_usage.apm.ingested_spans{*} by {service}.as_count(), 10, 'sum', 'desc')"
                    }],
                    "title": "Top Services by APM Volume"
                }
            }
        ],
        "layout_type": "ordered"
    }


def build_llm_dashboard(title: str, service_filter: str) -> Dict[str, Any]:
    """Build LLM observability dashboard"""
    return {
        "title": title,
        "description": f"LLM observability for {service_filter or 'all services'}",
        "widgets": [
            {
                "definition": {
                    "type": "timeseries",
                    "requests": [{
                        "q": f"sum:llm.tokens.total{{{service_filter}}}.as_count()",
                        "display_type": "area"
                    }],
                    "title": "Total Tokens Used"
                }
            },
            {
                "definition": {
                    "type": "query_value",
                    "requests": [{
                        "q": f"avg:llm.request.duration{{{service_filter}}}",
                        "aggregator": "avg"
                    }],
                    "title": "Average Latency (ms)",
                    "precision": 2
                }
            },
            {
                "definition": {
                    "type": "timeseries",
                    "requests": [{
                        "q": f"sum:llm.tokens.prompt{{{service_filter}}} by {{llm.model}}.as_count()",
                        "display_type": "bars"
                    }],
                    "title": "Token Usage by Model"
                }
            },
            {
                "definition": {
                    "type": "toplist",
                    "requests": [{
                        "q": f"top(sum:llm.tokens.total{{{service_filter}}} by {{resource_name}}.as_count(), 10, 'sum', 'desc')"
                    }],
                    "title": "Most Expensive Operations"
                }
            },
            {
                "definition": {
                    "type": "query_value",
                    "requests": [{
                        "q": f"sum:llm.request.error{{{service_filter}}}.as_count() / sum:llm.request.total{{{service_filter}}}.as_count() * 100",
                        "aggregator": "avg"
                    }],
                    "title": "Error Rate (%)",
                    "precision": 2
                }
            }
        ],
        "layout_type": "ordered"
    }


def create_dashboard(obs, client, title: str, dashboard_type: str, service: Optional[str] = None, output_json: bool = False):
    """Create a dashboard"""
    with obs.span("create_dashboard", tags={"type": dashboard_type, "service": service}):
        obs.log_info(f"Creating dashboard: {title} (type={dashboard_type})")

        # Build service filter
        service_filter = f"service:{service}" if service else ""

        # Generate dashboard JSON based on type
        with obs.span("build_dashboard_json"):
            if dashboard_type == "apm":
                dashboard_json = build_apm_dashboard(title, service_filter)
            elif dashboard_type == "logs":
                dashboard_json = build_logs_dashboard(title, service_filter)
            elif dashboard_type == "security":
                dashboard_json = build_security_dashboard(title, service_filter)
            elif dashboard_type == "cost":
                dashboard_json = build_cost_dashboard(title)
            elif dashboard_type == "llm":
                dashboard_json = build_llm_dashboard(title, service_filter)
            else:
                raise ValueError(f"Unknown dashboard type: {dashboard_type}")

        # API call
        start = datetime.now()
        try:
            response = client._request("POST", "/api/v1/dashboard", json=dashboard_json)
            api_duration = (datetime.now() - start).total_seconds() * 1000
            obs.record_api_call("/api/v1/dashboard", "POST", response.status_code, api_duration)
            data = response.json()
        except Exception as e:
            obs.log_error(f"Failed to create dashboard: {str(e)}")
            raise

        dashboard_id = data.get("id")
        dashboard_url = data.get("url") or f"https://app.{client.site}/dashboard/{dashboard_id}"

        obs.count("dashboard.created", 1, tags=[f"type:{dashboard_type}"])
        obs.log_info(f"Dashboard created: {dashboard_id}")

        output = {
            "id": dashboard_id,
            "title": data.get("title"),
            "url": dashboard_url,
            "type": dashboard_type,
            "status": "created"
        }

        if output_json:
            print(json.dumps(output, indent=2))
        else:
            print(f"✅ Dashboard created successfully")
            print()
            print(f"Title: {output['title']}")
            print(f"Type: {output['type']}")
            print(f"ID: {output['id']}")
            print(f"URL: {output['url']}")


def main():
    obs = init_observability("create-dashboard")

    parser = argparse.ArgumentParser(
        description="Create Datadog Dashboards",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Dashboard Types:
  apm       - APM performance dashboard (latency, errors, throughput)
  logs      - Log analysis dashboard
  security  - Security monitoring dashboard
  cost      - Cost analysis dashboard
  llm       - LLM observability dashboard

Examples:
  # Create APM dashboard for service
  create_dashboard.py --service payment-api --title "Payment API Performance" --type apm

  # Create security dashboard
  create_dashboard.py --service payment-api --title "Payment API Security" --type security

  # Create cost analysis dashboard
  create_dashboard.py --title "Infrastructure Costs" --type cost

  # Create LLM dashboard
  create_dashboard.py --service ai-service --title "LLM Metrics" --type llm
        """
    )

    parser.add_argument("--title", required=True, help="Dashboard title")
    parser.add_argument("--type", required=True, choices=["apm", "logs", "security", "cost", "llm"], help="Dashboard type")
    parser.add_argument("--service", help="Service name (for service-specific dashboards)")
    parser.add_argument("--json", action="store_true", help="Output as JSON")

    args = parser.parse_args()

    try:
        # Create Datadog client
        with obs.span("create_client"):
            client = create_client()

        # Create dashboard
        create_dashboard(obs, client, args.title, args.type, args.service, args.json)

        obs.log_info("Dashboard creation completed")
        finalize_observability(0)
        sys.exit(0)

    except ValueError as e:
        obs.log_error(f"Invalid parameters: {str(e)}")
        print(f"Error: {e}", file=sys.stderr)
        finalize_observability(1)
        sys.exit(1)
    except Exception as e:
        obs.log_error(f"Dashboard creation failed: {str(e)}", error_type=type(e).__name__)
        print(f"Error: {e}", file=sys.stderr)
        finalize_observability(1)
        sys.exit(1)


if __name__ == "__main__":
    main()
