#!/usr/bin/env python3
"""
Manage Datadog Monitors - Create, list, mute, unmute, and delete monitors.
Production-grade automation with comprehensive Datadog observability.
"""

import sys
import json
import argparse
from pathlib import Path
from typing import Optional, Dict, Any
from datetime import datetime, timedelta

# Add lib to path
sys.path.insert(0, str(Path(__file__).parent / "lib"))

from dd_observability import init_observability, finalize_observability
from datadog_client import create_client


def list_monitors(obs, client, service: Optional[str] = None, tag: Optional[str] = None, output_json: bool = False):
    """List monitors with optional filters"""
    with obs.span("list_monitors", tags={"service": service, "tag": tag}):
        obs.log_info(f"Listing monitors (service={service}, tag={tag})")

        # Build tags filter
        tags = []
        if service:
            tags.append(f"service:{service}")
        if tag:
            tags.append(tag)

        # API call
        start = datetime.now()
        try:
            response = client.get_monitors(tags=tags if tags else None)
            api_duration = (datetime.now() - start).total_seconds() * 1000
            obs.record_api_call("/api/v1/monitor", "GET", 200, api_duration)
        except Exception as e:
            obs.log_error(f"Failed to list monitors: {str(e)}")
            raise

        # Process results
        with obs.span("process_results"):
            total = len(response)

            # Count by state
            alert = sum(1 for m in response if m.get("overall_state") == "Alert")
            warn = sum(1 for m in response if m.get("overall_state") == "Warn")
            ok = sum(1 for m in response if m.get("overall_state") in ["OK", "No Data"])

            obs.record_result("monitors", total)
            obs.gauge("monitors.alert", alert, tags=["state:alert"])
            obs.gauge("monitors.warn", warn, tags=["state:warn"])
            obs.gauge("monitors.ok", ok, tags=["state:ok"])

            # Format monitors
            monitors = []
            for m in response:
                monitors.append({
                    "id": m.get("id"),
                    "name": m.get("name"),
                    "type": m.get("type"),
                    "query": m.get("query"),
                    "state": m.get("overall_state"),
                    "tags": m.get("tags", []),
                    "message": m.get("message", "")
                })

            output = {
                "total": total,
                "summary": {
                    "alert": alert,
                    "warn": warn,
                    "ok": ok
                },
                "monitors": monitors
            }

            if output_json:
                print(json.dumps(output, indent=2))
            else:
                print(f"📊 Monitor Summary")
                print()
                print(f"Total monitors: {total}")
                print(f"  Alert: {alert}")
                print(f"  Warn: {warn}")
                print(f"  OK/No Data: {ok}")

                if monitors:
                    print()
                    print("Monitors:")
                    for m in monitors[:10]:  # Show first 10
                        state_emoji = {"Alert": "🔴", "Warn": "🟡", "OK": "🟢", "No Data": "⚪"}.get(m["state"], "❓")
                        print(f"  {state_emoji} [{m['id']}] {m['name']}")
                        print(f"      State: {m['state']} | Type: {m['type']}")

            obs.log_info(f"Listed {total} monitors")


def create_monitor(obs, client, name: str, query: str, message: str, monitor_type: str = "metric alert", output_json: bool = False):
    """Create a new monitor"""
    with obs.span("create_monitor", tags={"name": name, "type": monitor_type}):
        obs.log_info(f"Creating monitor: {name}")

        # Build request body
        payload = {
            "name": name,
            "type": monitor_type,
            "query": query,
            "message": message,
            "tags": [],
            "options": {
                "notify_no_data": True,
                "no_data_timeframe": 20
            }
        }

        # API call
        start = datetime.now()
        try:
            response = client._request("POST", "/api/v1/monitor", json=payload)
            api_duration = (datetime.now() - start).total_seconds() * 1000
            obs.record_api_call("/api/v1/monitor", "POST", response.status_code, api_duration)
            data = response.json()
        except Exception as e:
            obs.log_error(f"Failed to create monitor: {str(e)}")
            raise

        monitor_id = data.get("id")
        obs.count("monitor.created", 1, tags=[f"type:{monitor_type}"])
        obs.log_info(f"Monitor created: {monitor_id}")

        output = {
            "id": data.get("id"),
            "name": data.get("name"),
            "type": data.get("type"),
            "query": data.get("query"),
            "message": data.get("message"),
            "created": data.get("created"),
            "status": "created"
        }

        if output_json:
            print(json.dumps(output, indent=2))
        else:
            print(f"✅ Monitor created successfully")
            print()
            print(f"ID: {output['id']}")
            print(f"Name: {output['name']}")
            print(f"Type: {output['type']}")


def mute_monitor(obs, client, monitor_id: int, duration_hours: Optional[int] = None, output_json: bool = False):
    """Mute a monitor"""
    with obs.span("mute_monitor", tags={"monitor_id": monitor_id, "duration": duration_hours}):
        obs.log_info(f"Muting monitor: {monitor_id}")

        # Calculate end time if duration specified
        payload = {"scope": "*"}
        if duration_hours:
            end_time = int((datetime.now() + timedelta(hours=duration_hours)).timestamp())
            payload["end"] = end_time

        # API call
        start = datetime.now()
        try:
            response = client._request("POST", f"/api/v1/monitor/{monitor_id}/mute", json=payload)
            api_duration = (datetime.now() - start).total_seconds() * 1000
            obs.record_api_call(f"/api/v1/monitor/{monitor_id}/mute", "POST", response.status_code, api_duration)
            data = response.json()
        except Exception as e:
            obs.log_error(f"Failed to mute monitor: {str(e)}")
            raise

        obs.count("monitor.muted", 1)
        obs.log_info(f"Monitor muted: {monitor_id}")

        output = {
            "id": data.get("id"),
            "name": data.get("name"),
            "status": "muted"
        }

        if output_json:
            print(json.dumps(output, indent=2))
        else:
            print(f"🔇 Monitor muted successfully")
            print()
            print(f"ID: {output['id']}")
            print(f"Name: {output['name']}")
            if duration_hours:
                print(f"Duration: {duration_hours} hours")


def unmute_monitor(obs, client, monitor_id: int, output_json: bool = False):
    """Unmute a monitor"""
    with obs.span("unmute_monitor", tags={"monitor_id": monitor_id}):
        obs.log_info(f"Unmuting monitor: {monitor_id}")

        # API call
        start = datetime.now()
        try:
            response = client._request("POST", f"/api/v1/monitor/{monitor_id}/unmute", json={"scope": "*"})
            api_duration = (datetime.now() - start).total_seconds() * 1000
            obs.record_api_call(f"/api/v1/monitor/{monitor_id}/unmute", "POST", response.status_code, api_duration)
            data = response.json()
        except Exception as e:
            obs.log_error(f"Failed to unmute monitor: {str(e)}")
            raise

        obs.count("monitor.unmuted", 1)
        obs.log_info(f"Monitor unmuted: {monitor_id}")

        output = {
            "id": data.get("id"),
            "name": data.get("name"),
            "status": "unmuted"
        }

        if output_json:
            print(json.dumps(output, indent=2))
        else:
            print(f"🔔 Monitor unmuted successfully")
            print()
            print(f"ID: {output['id']}")
            print(f"Name: {output['name']}")


def delete_monitor(obs, client, monitor_id: int, output_json: bool = False):
    """Delete a monitor"""
    with obs.span("delete_monitor", tags={"monitor_id": monitor_id}):
        obs.log_info(f"Deleting monitor: {monitor_id}")

        # API call
        start = datetime.now()
        try:
            response = client._request("DELETE", f"/api/v1/monitor/{monitor_id}")
            api_duration = (datetime.now() - start).total_seconds() * 1000
            obs.record_api_call(f"/api/v1/monitor/{monitor_id}", "DELETE", response.status_code, api_duration)
        except Exception as e:
            obs.log_error(f"Failed to delete monitor: {str(e)}")
            raise

        obs.count("monitor.deleted", 1)
        obs.log_info(f"Monitor deleted: {monitor_id}")

        output = {
            "id": monitor_id,
            "status": "deleted"
        }

        if output_json:
            print(json.dumps(output, indent=2))
        else:
            print(f"🗑️  Monitor deleted successfully")
            print()
            print(f"ID: {monitor_id}")


def main():
    obs = init_observability("manage-monitors")

    parser = argparse.ArgumentParser(
        description="Manage Datadog Monitors",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # List all monitors
  manage_monitors.py list

  # List monitors for a service
  manage_monitors.py list --service payment-api

  # Create error rate monitor
  manage_monitors.py create --name "High Error Rate" \\
    --query "avg(last_5m):sum:trace.express.request.errors{service:my-service}.as_count() > 10" \\
    --message "Error rate is high @slack-alerts"

  # Mute monitor for 2 hours
  manage_monitors.py mute --id 12345 --duration 2

  # Unmute monitor
  manage_monitors.py unmute --id 12345

  # Delete monitor
  manage_monitors.py delete --id 12345
        """
    )

    subparsers = parser.add_subparsers(dest="command", help="Command to execute")

    # List command
    list_parser = subparsers.add_parser("list", help="List monitors")
    list_parser.add_argument("--service", help="Filter by service tag")
    list_parser.add_argument("--tag", help="Filter by tag")
    list_parser.add_argument("--json", action="store_true", help="Output as JSON")

    # Create command
    create_parser = subparsers.add_parser("create", help="Create monitor")
    create_parser.add_argument("--name", required=True, help="Monitor name")
    create_parser.add_argument("--query", required=True, help="Monitor query")
    create_parser.add_argument("--message", required=True, help="Alert message")
    create_parser.add_argument("--type", default="metric alert", help="Monitor type (default: metric alert)")
    create_parser.add_argument("--json", action="store_true", help="Output as JSON")

    # Mute command
    mute_parser = subparsers.add_parser("mute", help="Mute monitor")
    mute_parser.add_argument("--id", type=int, required=True, help="Monitor ID")
    mute_parser.add_argument("--duration", type=int, help="Duration in hours")
    mute_parser.add_argument("--json", action="store_true", help="Output as JSON")

    # Unmute command
    unmute_parser = subparsers.add_parser("unmute", help="Unmute monitor")
    unmute_parser.add_argument("--id", type=int, required=True, help="Monitor ID")
    unmute_parser.add_argument("--json", action="store_true", help="Output as JSON")

    # Delete command
    delete_parser = subparsers.add_parser("delete", help="Delete monitor")
    delete_parser.add_argument("--id", type=int, required=True, help="Monitor ID")
    delete_parser.add_argument("--json", action="store_true", help="Output as JSON")

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
            list_monitors(obs, client, args.service, args.tag, args.json)
        elif args.command == "create":
            create_monitor(obs, client, args.name, args.query, args.message, args.type, args.json)
        elif args.command == "mute":
            mute_monitor(obs, client, args.id, args.duration, args.json)
        elif args.command == "unmute":
            unmute_monitor(obs, client, args.id, args.json)
        elif args.command == "delete":
            delete_monitor(obs, client, args.id, args.json)

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
