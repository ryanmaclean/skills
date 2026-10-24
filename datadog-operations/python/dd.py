#!/usr/bin/env python3
"""
Unified Datadog Skill CLI
Single entry point for all Datadog observability operations.

Usage:
    dd <command> [options]

Commands:
    Query Operations:
        apm             Query APM traces for performance analysis
        logs            Search logs for errors and patterns
        security        Query security monitoring signals
        watchdog        Query Watchdog anomalies
        metrics         Query and analyze metrics
        slos            Check SLO status and error budgets
        llm             Analyze LLM observability data
        cost            Analyze usage costs (FinOps)
        database        Query Database Monitoring (db)
        catalog         Query Service Catalog

    Automation Operations:
        monitors        Manage monitors (list, create, mute, etc)
        dashboards      Create dashboards
        incidents       Manage incidents
        workflows       Trigger workflows
        synthetics      Manage synthetic tests

    Smart Operations:
        health          Comprehensive service health check
        deploy          Check deploy readiness
        context         Detect service context
        investigate     Comprehensive service investigation
        verify          Verify Datadog setup and configuration

Examples:
    dd apm --service payment-api
    dd health
    dd deploy
    dd logs --query "status:error"
    dd monitors list
    dd incidents create --title "API Down"
    dd verify
    dd database --host postgres-prod
    dd catalog list --team backend
    dd investigate --service payment-api
"""

import sys
import os
from pathlib import Path

# Add lib to path
sys.path.insert(0, str(Path(__file__).parent / "lib"))

# Command mappings
COMMANDS = {
    # Query operations
    "apm": "query_apm",
    "logs": "search_logs",
    "security": "query_security_signals",
    "watchdog": "query_watchdog",
    "metrics": "query_metrics",
    "slos": "query_slos",
    "llm": "analyze_llm",
    "cost": "analyze_usage_cost",
    "database": "query_database",
    "catalog": "query_service_catalog",

    # Automation operations
    "monitors": "manage_monitors",
    "dashboards": "create_dashboard",
    "incidents": "manage_incidents",
    "workflows": "trigger_workflow",
    "synthetics": "manage_synthetics",

    # Smart operations
    "health": "smart_health",
    "deploy": "deploy_check",
    "context": "detect_context",
    "investigate": "investigate_service",
    "verify": "verify_setup",

    # Aliases
    "budget": "calculate_error_budget",
    "errors": "search_logs",
    "db": "query_database",
}


def show_help():
    """Show main help"""
    print(__doc__)
    sys.exit(0)


def main():
    if len(sys.argv) < 2 or sys.argv[1] in ["-h", "--help", "help"]:
        show_help()

    command = sys.argv[1]

    # Check if valid command
    if command not in COMMANDS:
        print(f"Error: Unknown command '{command}'", file=sys.stderr)
        print(f"\nRun 'dd --help' to see available commands", file=sys.stderr)
        sys.exit(1)

    # Get script name
    script_name = COMMANDS[command]
    script_path = Path(__file__).parent / f"{script_name}.py"

    # Check if script exists
    if not script_path.exists():
        print(f"Error: Script not found: {script_path}", file=sys.stderr)
        sys.exit(1)

    # Execute script with remaining args
    import subprocess
    result = subprocess.run(
        ["python3", str(script_path)] + sys.argv[2:],
        cwd=Path(__file__).parent
    )

    sys.exit(result.returncode)


if __name__ == "__main__":
    main()
