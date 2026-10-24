#!/usr/bin/env python3
"""
Manage Datadog Synthetic Tests - Create API and browser tests, get results.
Comprehensive synthetic monitoring with Datadog observability.
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


def list_synthetics(obs, client, tag_filter: Optional[str] = None, output_json: bool = False):
    """List synthetic tests"""
    with obs.span("list_synthetics", tags={"tag": tag_filter}):
        obs.log_info(f"Listing synthetic tests (tag={tag_filter})")

        # API call
        start = datetime.now()
        try:
            response = client._request("GET", "/api/v1/synthetics/tests")
            api_duration = (datetime.now() - start).total_seconds() * 1000
            obs.record_api_call("/api/v1/synthetics/tests", "GET", response.status_code, api_duration)
            data = response.json()
        except Exception as e:
            obs.log_error(f"Failed to list synthetic tests: {str(e)}")
            raise

        # Process results
        with obs.span("process_results"):
            tests_data = data.get("tests", [])

            # Filter by tag if specified
            if tag_filter:
                tests_data = [t for t in tests_data if any(tag_filter in tag for tag in t.get("tags", []))]

            total = len(tests_data)

            # Count by type
            api_tests = sum(1 for t in tests_data if t.get("type") == "api")
            browser_tests = sum(1 for t in tests_data if t.get("type") == "browser")

            # Count by status
            active = sum(1 for t in tests_data if t.get("status") == "live")
            paused = sum(1 for t in tests_data if t.get("status") == "paused")

            obs.record_result("synthetics", total)
            obs.gauge("synthetics.api", api_tests, tags=["type:api"])
            obs.gauge("synthetics.browser", browser_tests, tags=["type:browser"])
            obs.gauge("synthetics.active", active, tags=["status:active"])
            obs.gauge("synthetics.paused", paused, tags=["status:paused"])

            # Format tests
            tests = []
            for t in tests_data:
                tests.append({
                    "id": t.get("public_id"),
                    "name": t.get("name"),
                    "type": t.get("type"),
                    "status": t.get("status"),
                    "url": t.get("config", {}).get("request", {}).get("url"),
                    "locations": t.get("locations", []),
                    "tags": t.get("tags", []),
                    "monitor_id": t.get("monitor_id")
                })

            output = {
                "total": total,
                "summary": {
                    "api": api_tests,
                    "browser": browser_tests,
                    "active": active,
                    "paused": paused
                },
                "tests": tests
            }

            if output_json:
                print(json.dumps(output, indent=2))
            else:
                print(f"🔍 Synthetic Tests Summary")
                print()
                print(f"Total tests: {total}")
                print(f"  API tests: {api_tests}")
                print(f"  Browser tests: {browser_tests}")
                print(f"  Active: {active}")
                print(f"  Paused: {paused}")

                if tests:
                    print()
                    print("Tests:")
                    for t in tests[:10]:  # Show first 10
                        type_emoji = {"api": "🌐", "browser": "🖥️"}.get(t["type"], "❓")
                        status_emoji = {"live": "🟢", "paused": "⏸️"}.get(t["status"], "❓")
                        print(f"  {type_emoji}{status_emoji} [{t['id']}] {t['name']}")
                        print(f"      URL: {t['url']} | Type: {t['type']}")

            obs.log_info(f"Listed {total} synthetic tests")


def create_api_test(obs, client, name: str, url: str, method: str = "GET", output_json: bool = False):
    """Create an API synthetic test"""
    with obs.span("create_api_test", tags={"name": name, "method": method}):
        obs.log_info(f"Creating API synthetic test: {name}")

        # Build request body
        payload = {
            "name": name,
            "type": "api",
            "subtype": "http",
            "status": "live",
            "config": {
                "request": {
                    "method": method,
                    "url": url,
                    "timeout": 30
                },
                "assertions": [
                    {
                        "type": "statusCode",
                        "operator": "is",
                        "target": 200
                    },
                    {
                        "type": "responseTime",
                        "operator": "lessThan",
                        "target": 2000
                    }
                ]
            },
            "locations": ["aws:us-east-1"],
            "options": {
                "tick_every": 300,
                "min_failure_duration": 0,
                "min_location_failed": 1,
                "monitor_options": {
                    "notify_audit": False,
                    "notify_no_data": False
                }
            },
            "message": f"Synthetic test alert: {name}",
            "tags": ["synthetic"]
        }

        # API call
        start = datetime.now()
        try:
            response = client._request("POST", "/api/v1/synthetics/tests/api", json=payload)
            api_duration = (datetime.now() - start).total_seconds() * 1000
            obs.record_api_call("/api/v1/synthetics/tests/api", "POST", response.status_code, api_duration)
            data = response.json()
        except Exception as e:
            obs.log_error(f"Failed to create API test: {str(e)}")
            raise

        test_id = data.get("public_id")
        obs.count("synthetic.created", 1, tags=["type:api", f"method:{method}"])
        obs.log_info(f"API test created: {test_id}")

        output = {
            "id": test_id,
            "name": data.get("name"),
            "type": data.get("type"),
            "url": data.get("config", {}).get("request", {}).get("url"),
            "status": "created"
        }

        if output_json:
            print(json.dumps(output, indent=2))
        else:
            print(f"✅ API synthetic test created successfully")
            print()
            print(f"ID: {output['id']}")
            print(f"Name: {output['name']}")
            print(f"URL: {output['url']}")
            print(f"Method: {method}")


def create_browser_test(obs, client, name: str, url: str, output_json: bool = False):
    """Create a browser synthetic test"""
    with obs.span("create_browser_test", tags={"name": name}):
        obs.log_info(f"Creating browser synthetic test: {name}")

        # Build request body
        payload = {
            "name": name,
            "type": "browser",
            "status": "live",
            "config": {
                "request": {
                    "url": url
                },
                "assertions": []
            },
            "locations": ["aws:us-east-1"],
            "options": {
                "tick_every": 900,
                "min_failure_duration": 0,
                "min_location_failed": 1,
                "device_ids": ["laptop_large"],
                "monitor_options": {
                    "notify_audit": False,
                    "notify_no_data": False
                }
            },
            "message": f"Browser test alert: {name}",
            "tags": ["synthetic", "browser"],
            "steps": [
                {
                    "name": "Navigate to URL",
                    "type": "goToUrl",
                    "params": {
                        "url": url
                    },
                    "allowFailure": False,
                    "timeout": 60
                }
            ]
        }

        # API call
        start = datetime.now()
        try:
            response = client._request("POST", "/api/v1/synthetics/tests/browser", json=payload)
            api_duration = (datetime.now() - start).total_seconds() * 1000
            obs.record_api_call("/api/v1/synthetics/tests/browser", "POST", response.status_code, api_duration)
            data = response.json()
        except Exception as e:
            obs.log_error(f"Failed to create browser test: {str(e)}")
            raise

        test_id = data.get("public_id")
        obs.count("synthetic.created", 1, tags=["type:browser"])
        obs.log_info(f"Browser test created: {test_id}")

        output = {
            "id": test_id,
            "name": data.get("name"),
            "type": data.get("type"),
            "url": data.get("config", {}).get("request", {}).get("url"),
            "status": "created"
        }

        if output_json:
            print(json.dumps(output, indent=2))
        else:
            print(f"✅ Browser synthetic test created successfully")
            print()
            print(f"ID: {output['id']}")
            print(f"Name: {output['name']}")
            print(f"URL: {output['url']}")


def get_test(obs, client, test_id: str, output_json: bool = False):
    """Get synthetic test details"""
    with obs.span("get_test", tags={"test_id": test_id}):
        obs.log_info(f"Fetching test: {test_id}")

        # API call
        start = datetime.now()
        try:
            response = client._request("GET", f"/api/v1/synthetics/tests/{test_id}")
            api_duration = (datetime.now() - start).total_seconds() * 1000
            obs.record_api_call(f"/api/v1/synthetics/tests/{test_id}", "GET", response.status_code, api_duration)
            data = response.json()
        except Exception as e:
            obs.log_error(f"Failed to get test: {str(e)}")
            raise

        obs.count("synthetic.get", 1)
        obs.log_info(f"Fetched test: {test_id}")

        output = {
            "id": data.get("public_id"),
            "name": data.get("name"),
            "type": data.get("type"),
            "status": data.get("status"),
            "url": data.get("config", {}).get("request", {}).get("url"),
            "locations": data.get("locations", []),
            "monitor_id": data.get("monitor_id"),
            "tags": data.get("tags", []),
            "created_at": data.get("created_at"),
            "modified_at": data.get("modified_at")
        }

        if output_json:
            print(json.dumps(output, indent=2))
        else:
            print(f"🔍 Synthetic Test Details")
            print()
            print(f"ID: {output['id']}")
            print(f"Name: {output['name']}")
            print(f"Type: {output['type']}")
            print(f"Status: {output['status']}")
            print(f"URL: {output['url']}")
            print(f"Locations: {', '.join(output['locations'])}")
            print(f"Monitor ID: {output['monitor_id']}")
            print(f"Tags: {', '.join(output['tags'])}")


def main():
    obs = init_observability("manage-synthetics")

    parser = argparse.ArgumentParser(
        description="Manage Datadog Synthetic Tests",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # List all synthetic tests
  manage_synthetics.py list

  # List tests with specific tag
  manage_synthetics.py list --tag production

  # Create API uptime check
  manage_synthetics.py create-api --name "Payment API Uptime" \\
    --url "https://api.example.com/health" --method GET

  # Create browser test
  manage_synthetics.py create-browser --name "Login Flow" \\
    --url "https://app.example.com/login"

  # Get test results
  manage_synthetics.py get --id abc-123-def
        """
    )

    subparsers = parser.add_subparsers(dest="command", help="Command to execute")

    # List command
    list_parser = subparsers.add_parser("list", help="List synthetic tests")
    list_parser.add_argument("--tag", help="Filter by tag")
    list_parser.add_argument("--json", action="store_true", help="Output as JSON")

    # Create API test command
    create_api_parser = subparsers.add_parser("create-api", help="Create API test")
    create_api_parser.add_argument("--name", required=True, help="Test name")
    create_api_parser.add_argument("--url", required=True, help="URL to test")
    create_api_parser.add_argument("--method", default="GET", help="HTTP method (default: GET)")
    create_api_parser.add_argument("--json", action="store_true", help="Output as JSON")

    # Create browser test command
    create_browser_parser = subparsers.add_parser("create-browser", help="Create browser test")
    create_browser_parser.add_argument("--name", required=True, help="Test name")
    create_browser_parser.add_argument("--url", required=True, help="URL to test")
    create_browser_parser.add_argument("--json", action="store_true", help="Output as JSON")

    # Get test command
    get_parser = subparsers.add_parser("get", help="Get test details")
    get_parser.add_argument("--id", required=True, help="Test ID")
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
            list_synthetics(obs, client, args.tag, args.json)
        elif args.command == "create-api":
            create_api_test(obs, client, args.name, args.url, args.method, args.json)
        elif args.command == "create-browser":
            create_browser_test(obs, client, args.name, args.url, args.json)
        elif args.command == "get":
            get_test(obs, client, args.id, args.json)

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
