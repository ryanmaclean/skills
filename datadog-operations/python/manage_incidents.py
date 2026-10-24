#!/usr/bin/env python3
"""
Manage Datadog Incidents - Create, update, and query incidents.
Comprehensive incident management with Datadog observability.
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


def list_incidents(obs, client, status_filter: Optional[str] = None, output_json: bool = False):
    """List incidents with optional status filter"""
    with obs.span("list_incidents", tags={"status": status_filter}):
        obs.log_info(f"Listing incidents (status={status_filter})")

        # Build query parameters
        params = {}
        if status_filter:
            params["filter[state]"] = status_filter

        # API call
        start = datetime.now()
        try:
            response = client._request("GET", "/api/v2/incidents", params=params)
            api_duration = (datetime.now() - start).total_seconds() * 1000
            obs.record_api_call("/api/v2/incidents", "GET", response.status_code, api_duration)
            data = response.json()
        except Exception as e:
            obs.log_error(f"Failed to list incidents: {str(e)}")
            raise

        # Process results
        with obs.span("process_results"):
            incidents_data = data.get("data", [])
            total = len(incidents_data)

            # Count by status
            active = sum(1 for i in incidents_data if i.get("attributes", {}).get("state") == "active")
            stable = sum(1 for i in incidents_data if i.get("attributes", {}).get("state") == "stable")
            resolved = sum(1 for i in incidents_data if i.get("attributes", {}).get("state") == "resolved")

            obs.record_result("incidents", total)
            obs.gauge("incidents.active", active, tags=["state:active"])
            obs.gauge("incidents.stable", stable, tags=["state:stable"])
            obs.gauge("incidents.resolved", resolved, tags=["state:resolved"])

            # Format incidents
            incidents = []
            for i in incidents_data:
                attrs = i.get("attributes", {})
                incidents.append({
                    "id": i.get("id"),
                    "title": attrs.get("title"),
                    "state": attrs.get("state"),
                    "severity": attrs.get("severity"),
                    "created": attrs.get("created"),
                    "modified": attrs.get("modified"),
                    "customer_impact": attrs.get("customer_impact_scope")
                })

            output = {
                "total": total,
                "summary": {
                    "active": active,
                    "stable": stable,
                    "resolved": resolved
                },
                "incidents": incidents
            }

            if output_json:
                print(json.dumps(output, indent=2))
            else:
                print(f"📋 Incident Summary")
                print()
                print(f"Total incidents: {total}")
                print(f"  Active: {active}")
                print(f"  Stable: {stable}")
                print(f"  Resolved: {resolved}")

                if incidents:
                    print()
                    print("Recent incidents:")
                    for inc in incidents[:10]:  # Show first 10
                        state_emoji = {"active": "🔴", "stable": "🟡", "resolved": "✅"}.get(inc["state"], "❓")
                        severity_label = inc.get("severity", "UNKNOWN")
                        print(f"  {state_emoji} [{inc['id']}] {inc['title']}")
                        print(f"      State: {inc['state']} | Severity: {severity_label}")

            obs.log_info(f"Listed {total} incidents")


def create_incident(obs, client, title: str, service: str, severity: str = "UNKNOWN", output_json: bool = False):
    """Create a new incident"""
    with obs.span("create_incident", tags={"title": title, "severity": severity}):
        obs.log_info(f"Creating incident: {title}")

        # Build request body
        payload = {
            "data": {
                "type": "incidents",
                "attributes": {
                    "title": title,
                    "severity": severity,
                    "customer_impacted": False,
                    "fields": {
                        "service": {
                            "type": "textbox",
                            "value": service
                        }
                    }
                }
            }
        }

        # API call
        start = datetime.now()
        try:
            response = client._request("POST", "/api/v2/incidents", json=payload)
            api_duration = (datetime.now() - start).total_seconds() * 1000
            obs.record_api_call("/api/v2/incidents", "POST", response.status_code, api_duration)
            data = response.json()
        except Exception as e:
            obs.log_error(f"Failed to create incident: {str(e)}")
            raise

        incident_data = data.get("data", {})
        attrs = incident_data.get("attributes", {})
        incident_id = incident_data.get("id")

        obs.count("incident.created", 1, tags=[f"severity:{severity}"])
        obs.log_info(f"Incident created: {incident_id}")

        output = {
            "id": incident_id,
            "title": attrs.get("title"),
            "severity": attrs.get("severity"),
            "state": attrs.get("state"),
            "created": attrs.get("created"),
            "status": "created"
        }

        if output_json:
            print(json.dumps(output, indent=2))
        else:
            print(f"✅ Incident created successfully")
            print()
            print(f"ID: {output['id']}")
            print(f"Title: {output['title']}")
            print(f"Severity: {output['severity']}")
            print(f"State: {output['state']}")


def update_incident(obs, client, incident_id: str, new_status: str, output_json: bool = False):
    """Update incident status"""
    with obs.span("update_incident", tags={"incident_id": incident_id, "new_status": new_status}):
        obs.log_info(f"Updating incident {incident_id} to status: {new_status}")

        # Build request body
        payload = {
            "data": {
                "type": "incidents",
                "id": incident_id,
                "attributes": {
                    "state": new_status
                }
            }
        }

        # API call
        start = datetime.now()
        try:
            response = client._request("PATCH", f"/api/v2/incidents/{incident_id}", json=payload)
            api_duration = (datetime.now() - start).total_seconds() * 1000
            obs.record_api_call(f"/api/v2/incidents/{incident_id}", "PATCH", response.status_code, api_duration)
            data = response.json()
        except Exception as e:
            obs.log_error(f"Failed to update incident: {str(e)}")
            raise

        incident_data = data.get("data", {})
        attrs = incident_data.get("attributes", {})

        obs.count("incident.updated", 1, tags=[f"new_status:{new_status}"])
        obs.log_info(f"Incident updated: {incident_id}")

        output = {
            "id": incident_data.get("id"),
            "title": attrs.get("title"),
            "state": attrs.get("state"),
            "modified": attrs.get("modified"),
            "status": "updated"
        }

        if output_json:
            print(json.dumps(output, indent=2))
        else:
            print(f"✅ Incident updated successfully")
            print()
            print(f"ID: {output['id']}")
            print(f"Title: {output['title']}")
            print(f"New State: {output['state']}")


def get_incident(obs, client, incident_id: str, output_json: bool = False):
    """Get incident details"""
    with obs.span("get_incident", tags={"incident_id": incident_id}):
        obs.log_info(f"Fetching incident: {incident_id}")

        # API call
        start = datetime.now()
        try:
            response = client._request("GET", f"/api/v2/incidents/{incident_id}")
            api_duration = (datetime.now() - start).total_seconds() * 1000
            obs.record_api_call(f"/api/v2/incidents/{incident_id}", "GET", response.status_code, api_duration)
            data = response.json()
        except Exception as e:
            obs.log_error(f"Failed to get incident: {str(e)}")
            raise

        incident_data = data.get("data", {})
        attrs = incident_data.get("attributes", {})
        included = data.get("included", [])

        # Extract timeline events
        timeline = []
        for item in included:
            if item.get("type") == "incident_timeline":
                timeline.append({
                    "type": item.get("attributes", {}).get("content", {}).get("content_type"),
                    "message": item.get("attributes", {}).get("content", {}).get("message"),
                    "timestamp": item.get("attributes", {}).get("timestamp")
                })

        obs.count("incident.get", 1)
        obs.log_info(f"Fetched incident: {incident_id}")

        output = {
            "id": incident_data.get("id"),
            "title": attrs.get("title"),
            "state": attrs.get("state"),
            "severity": attrs.get("severity"),
            "customer_impacted": attrs.get("customer_impacted"),
            "created": attrs.get("created"),
            "modified": attrs.get("modified"),
            "resolved": attrs.get("resolved"),
            "timeline": timeline
        }

        if output_json:
            print(json.dumps(output, indent=2))
        else:
            print(f"📋 Incident Details")
            print()
            print(f"ID: {output['id']}")
            print(f"Title: {output['title']}")
            print(f"State: {output['state']}")
            print(f"Severity: {output['severity']}")
            print(f"Customer Impacted: {output['customer_impacted']}")
            print(f"Created: {output['created']}")
            print(f"Modified: {output['modified']}")
            if output['resolved']:
                print(f"Resolved: {output['resolved']}")

            if timeline:
                print()
                print("Timeline:")
                for event in timeline[:5]:  # Show first 5 events
                    print(f"  • [{event['timestamp']}] {event['type']}")
                    if event['message']:
                        print(f"    {event['message']}")


def main():
    obs = init_observability("manage-incidents")

    parser = argparse.ArgumentParser(
        description="Manage Datadog Incidents",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Status values:
  active, stable, resolved

Severity values:
  SEV-1, SEV-2, SEV-3, SEV-4, SEV-5, UNKNOWN

Examples:
  # List active incidents
  manage_incidents.py list --status active

  # Create critical incident
  manage_incidents.py create --title "Payment API Down" \\
    --service payment-api --severity SEV-1

  # Update incident status
  manage_incidents.py update --id abc123 --status resolved

  # Get incident details
  manage_incidents.py get --id abc123
        """
    )

    subparsers = parser.add_subparsers(dest="command", help="Command to execute")

    # List command
    list_parser = subparsers.add_parser("list", help="List incidents")
    list_parser.add_argument("--status", choices=["active", "stable", "resolved"], help="Filter by status")
    list_parser.add_argument("--json", action="store_true", help="Output as JSON")

    # Create command
    create_parser = subparsers.add_parser("create", help="Create incident")
    create_parser.add_argument("--title", required=True, help="Incident title")
    create_parser.add_argument("--service", required=True, help="Service name")
    create_parser.add_argument("--severity", default="UNKNOWN", choices=["SEV-1", "SEV-2", "SEV-3", "SEV-4", "SEV-5", "UNKNOWN"], help="Severity level")
    create_parser.add_argument("--json", action="store_true", help="Output as JSON")

    # Update command
    update_parser = subparsers.add_parser("update", help="Update incident")
    update_parser.add_argument("--id", required=True, help="Incident ID")
    update_parser.add_argument("--status", required=True, choices=["active", "stable", "resolved"], help="New status")
    update_parser.add_argument("--json", action="store_true", help="Output as JSON")

    # Get command
    get_parser = subparsers.add_parser("get", help="Get incident details")
    get_parser.add_argument("--id", required=True, help="Incident ID")
    get_parser.add_argument("--json", action="store_true", help="Output as JSON")

    args = parser.parse_args()

    if not args.command:
        parser.print_help()
        finalize_observability(1)
        sys.exit(1)

    try:
        # Create Datadog client
        with obs.span("create_client"):
            client = create_client()

        # Execute command
        if args.command == "list":
            list_incidents(obs, client, args.status, args.json)
        elif args.command == "create":
            create_incident(obs, client, args.title, args.service, args.severity, args.json)
        elif args.command == "update":
            update_incident(obs, client, args.id, args.status, args.json)
        elif args.command == "get":
            get_incident(obs, client, args.id, args.json)

        obs.log_info(f"Command completed: {args.command}")
        finalize_observability(0)
        sys.exit(0)

    except ValueError as e:
        obs.log_error(f"Invalid parameters: {str(e)}")
        print(f"Error: {e}", file=sys.stderr)
        finalize_observability(1)
        sys.exit(1)
    except Exception as e:
        obs.log_error(f"Command failed: {str(e)}", error_type=type(e).__name__)
        print(f"Error: {e}", file=sys.stderr)
        finalize_observability(1)
        sys.exit(1)


if __name__ == "__main__":
    main()
