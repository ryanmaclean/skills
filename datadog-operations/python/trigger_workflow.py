#!/usr/bin/env python3
"""
Trigger Datadog Workflows - Execute automation workflows for incident response.
List available workflows and trigger them with input data.
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


def list_workflows(obs, client, output_json: bool = False):
    """List available workflows"""
    with obs.span("list_workflows"):
        obs.log_info("Listing workflows")

        # API call
        start = datetime.now()
        try:
            response = client._request("GET", "/api/v2/workflows")
            api_duration = (datetime.now() - start).total_seconds() * 1000
            obs.record_api_call("/api/v2/workflows", "GET", response.status_code, api_duration)
            data = response.json()
        except Exception as e:
            obs.log_error(f"Failed to list workflows: {str(e)}")
            raise

        # Process results
        with obs.span("process_results"):
            workflows_data = data.get("data", [])
            total = len(workflows_data)

            obs.record_result("workflows", total)

            # Format workflows
            workflows = []
            for w in workflows_data:
                attrs = w.get("attributes", {})
                workflows.append({
                    "id": w.get("id"),
                    "name": attrs.get("name"),
                    "description": attrs.get("description"),
                    "created": attrs.get("created_at"),
                    "modified": attrs.get("modified_at")
                })

            output = {
                "total": total,
                "workflows": workflows
            }

            if output_json:
                print(json.dumps(output, indent=2))
            else:
                print(f"🔄 Available Workflows")
                print()
                print(f"Total: {total} workflows")

                if workflows:
                    print()
                    print("Workflows:")
                    for w in workflows:
                        print(f"  • [{w['id']}] {w['name']}")
                        if w['description']:
                            print(f"    {w['description']}")

            obs.log_info(f"Listed {total} workflows")


def run_workflow(obs, client, workflow_id: str, input_data: Optional[str] = None, output_json: bool = False):
    """Run a workflow"""
    with obs.span("run_workflow", tags={"workflow_id": workflow_id}):
        obs.log_info(f"Triggering workflow: {workflow_id}")

        # Parse input data
        with obs.span("parse_input"):
            if input_data:
                try:
                    input_json = json.loads(input_data)
                except json.JSONDecodeError as e:
                    raise ValueError(f"Invalid JSON input: {e}")
            else:
                input_json = {}

        # Build request body
        payload = {
            "data": {
                "attributes": {
                    "input": input_json
                }
            }
        }

        # API call
        start = datetime.now()
        try:
            response = client._request("POST", f"/api/v2/workflows/{workflow_id}/instances", json=payload)
            api_duration = (datetime.now() - start).total_seconds() * 1000
            obs.record_api_call(f"/api/v2/workflows/{workflow_id}/instances", "POST", response.status_code, api_duration)
            data = response.json()
        except Exception as e:
            obs.log_error(f"Failed to trigger workflow: {str(e)}")
            raise

        instance_data = data.get("data", {})
        attrs = instance_data.get("attributes", {})
        instance_id = instance_data.get("id")

        obs.count("workflow.triggered", 1, tags=[f"workflow_id:{workflow_id}"])
        obs.log_info(f"Workflow instance created: {instance_id}")

        output = {
            "instance_id": instance_id,
            "workflow_id": workflow_id,
            "status": "triggered",
            "created_at": attrs.get("created_at")
        }

        if output_json:
            print(json.dumps(output, indent=2))
        else:
            print(f"✅ Workflow triggered successfully")
            print()
            print(f"Workflow ID: {workflow_id}")
            print(f"Instance ID: {instance_id}")
            print(f"Status: triggered")
            if input_json:
                print()
                print("Input data:")
                print(json.dumps(input_json, indent=2))


def main():
    obs = init_observability("trigger-workflow")

    parser = argparse.ArgumentParser(
        description="Trigger Datadog Workflows",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # List all workflows
  trigger_workflow.py list

  # Trigger workflow without input
  trigger_workflow.py run --id abc123

  # Trigger workflow with input data
  trigger_workflow.py run --id abc123 --input '{"service": "payment-api", "severity": "high"}'
        """
    )

    subparsers = parser.add_subparsers(dest="command", help="Command to execute")

    # List command
    list_parser = subparsers.add_parser("list", help="List workflows")
    list_parser.add_argument("--json", action="store_true", help="Output as JSON")

    # Run command
    run_parser = subparsers.add_parser("run", help="Run workflow")
    run_parser.add_argument("--id", required=True, help="Workflow ID")
    run_parser.add_argument("--input", help="Input data as JSON string")
    run_parser.add_argument("--json", action="store_true", help="Output as JSON")

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
            list_workflows(obs, client, args.json)
        elif args.command == "run":
            run_workflow(obs, client, args.id, args.input, args.json)

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
