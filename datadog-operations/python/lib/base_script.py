"""
Base script class with common functionality.
All Datadog skill scripts inherit from this to avoid duplication.
"""

import sys
import json
import argparse
from typing import Dict, Any, Optional, List
from datetime import datetime, timedelta
from abc import ABC, abstractmethod

from dd_observability import init_observability, finalize_observability, get_observability
from datadog_client import DatadogClient, create_client
from context_detector import detect_context, ServiceContext


class BaseScript(ABC):
    """
    Base class for all Datadog skill scripts.
    Handles common patterns: observability, CLI, output formatting, error handling.
    """

    def __init__(self, script_name: str, description: str):
        self.script_name = script_name
        self.description = description
        self.obs = None
        self.client: Optional[DatadogClient] = None
        self.args = None
        self.context: Optional[ServiceContext] = None

    def parse_duration(self, duration: str) -> timedelta:
        """Parse duration string like '1h', '24h', '7d'"""
        if duration.endswith('h'):
            return timedelta(hours=int(duration[:-1]))
        elif duration.endswith('d'):
            return timedelta(days=int(duration[:-1]))
        elif duration.endswith('m'):
            return timedelta(minutes=int(duration[:-1]))
        else:
            raise ValueError(f"Invalid duration: {duration}. Use format like '1h', '24h', '7d'")

    def get_time_range(self, duration: str) -> tuple[datetime, datetime]:
        """Get time range from duration string"""
        to_time = datetime.now()
        from_time = to_time - self.parse_duration(duration)
        return from_time, to_time

    def detect_service(self, explicit_service: Optional[str] = None) -> str:
        """Detect or use explicit service name"""
        if explicit_service:
            return explicit_service

        with self.obs.span("detect_context"):
            self.context = detect_context()
            service = self.context.service_name

            if not service:
                self.obs.log_error("Could not detect service name")
                self.error("Could not detect service name. Specify with --service or run in a git repository.")

            self.obs.log_info(f"Auto-detected service: {service}")
            return service

    def setup_common_args(self, parser: argparse.ArgumentParser):
        """Add common arguments to parser"""
        parser.add_argument(
            "--service",
            help="Service name (auto-detected if not provided)"
        )
        parser.add_argument(
            "--json",
            action="store_true",
            help="Output as JSON"
        )
        parser.add_argument(
            "--working-dir",
            default=".",
            help="Project directory (default: current directory)"
        )

    @abstractmethod
    def setup_args(self, parser: argparse.ArgumentParser):
        """Setup script-specific arguments. Override in subclass."""
        pass

    @abstractmethod
    def execute(self) -> Dict[str, Any]:
        """Execute script logic. Override in subclass. Return result dict."""
        pass

    @abstractmethod
    def format_output(self, result: Dict[str, Any]) -> str:
        """Format result as conversational text. Override in subclass."""
        pass

    def output(self, result: Dict[str, Any]):
        """Output result in appropriate format"""
        if self.args.json:
            print(json.dumps(result, indent=2))
        else:
            print(self.format_output(result))

    def error(self, message: str, exit_code: int = 1):
        """Handle error and exit"""
        if self.obs:
            self.obs.log_error(message)
        print(f"Error: {message}", file=sys.stderr)
        if self.obs:
            finalize_observability(exit_code)
        sys.exit(exit_code)

    def run(self):
        """Main entry point - handles full script lifecycle"""
        # Initialize observability
        self.obs = init_observability(self.script_name)

        try:
            # Parse arguments
            with self.obs.span("parse_args"):
                parser = argparse.ArgumentParser(description=self.description)
                self.setup_common_args(parser)
                self.setup_args(parser)
                self.args = parser.parse_args()

            # Create Datadog client
            with self.obs.span("create_client"):
                self.client = create_client()

            # Execute script logic
            with self.obs.span("execute"):
                result = self.execute()

            # Output result
            with self.obs.span("output"):
                self.output(result)

            # Finalize
            self.obs.log_info(f"Script completed successfully: {self.script_name}")
            finalize_observability(0)
            sys.exit(0)

        except KeyError as e:
            self.error(f"Missing environment variable: {e}. Set DD_API_KEY and DD_APP_KEY.")
        except ValueError as e:
            self.error(f"Invalid input: {e}")
        except Exception as e:
            self.error(f"Unexpected error: {e}")


class QueryScript(BaseScript):
    """
    Base class for query scripts (APM, logs, metrics, etc).
    Adds common query-related functionality.
    """

    def setup_common_query_args(self, parser: argparse.ArgumentParser):
        """Add common query arguments"""
        parser.add_argument(
            "--duration",
            default="1h",
            help="Time range: 1h, 24h, 7d (default: 1h)"
        )

    def setup_args(self, parser: argparse.ArgumentParser):
        """Setup query script arguments"""
        self.setup_common_query_args(parser)
        self.setup_query_args(parser)

    @abstractmethod
    def setup_query_args(self, parser: argparse.ArgumentParser):
        """Setup script-specific query arguments. Override in subclass."""
        pass

    def track_api_call(self, endpoint: str, method: str = "GET"):
        """Context manager for tracking API calls"""
        class APICallTracker:
            def __init__(self, obs, endpoint, method):
                self.obs = obs
                self.endpoint = endpoint
                self.method = method
                self.start = None

            def __enter__(self):
                self.start = datetime.now()
                return self

            def __exit__(self, exc_type, exc_val, exc_tb):
                duration = (datetime.now() - self.start).total_seconds() * 1000
                status = 200 if exc_type is None else 500
                self.obs.record_api_call(self.endpoint, self.method, status, duration)

        return APICallTracker(self.obs, endpoint, method)


class AutomationScript(BaseScript):
    """
    Base class for automation scripts (monitors, incidents, workflows, etc).
    Adds common automation-related functionality.
    """

    def setup_automation_args(self, parser: argparse.ArgumentParser):
        """Add common automation arguments"""
        parser.add_argument(
            "action",
            help="Action to perform"
        )

    def setup_args(self, parser: argparse.ArgumentParser):
        """Setup automation script arguments"""
        self.setup_automation_args(parser)
        self.setup_action_args(parser)

    @abstractmethod
    def setup_action_args(self, parser: argparse.ArgumentParser):
        """Setup script-specific action arguments. Override in subclass."""
        pass

    def execute(self) -> Dict[str, Any]:
        """Execute automation based on action"""
        action = self.args.action

        # Call action handler
        handler_name = f"action_{action.replace('-', '_')}"
        if not hasattr(self, handler_name):
            self.error(f"Unknown action: {action}")

        handler = getattr(self, handler_name)
        return handler()
