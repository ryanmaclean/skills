#!/usr/bin/env python3
"""
Verify Datadog setup and configuration.
Checks environment variables, API connectivity, agent reachability, and tracers.
"""

import sys
import os
import socket
import subprocess
import json
from pathlib import Path
from typing import Dict, Any, List, Optional
import argparse

# Add lib to path
sys.path.insert(0, str(Path(__file__).parent / "lib"))

from base_script import BaseScript
from formatters import OutputFormatter, ConversationalFormatter
from dd_observability import get_observability


class VerifySetupScript(BaseScript):
    """Verify Datadog setup and configuration"""

    def __init__(self):
        super().__init__(
            script_name="verify-setup",
            description="Verify Datadog setup and configuration"
        )
        self.results: List[str] = []
        self.warnings: List[str] = []
        self.errors: List[str] = []

    def setup_args(self, parser: argparse.ArgumentParser):
        """Setup script-specific arguments"""
        parser.add_argument(
            "--skip-agent-check",
            action="store_true",
            help="Skip Datadog Agent connectivity checks"
        )

    def check_environment_variables(self) -> Dict[str, Any]:
        """Check required environment variables"""
        with self.obs.span("check_environment_variables"):
            checks = {}

            # DD_API_KEY
            if os.getenv("DD_API_KEY"):
                self.obs.log_info("DD_API_KEY is set")
                self.results.append("DD_API_KEY: configured")
                checks["DD_API_KEY"] = "configured"
            else:
                self.obs.log_error("DD_API_KEY is not set")
                self.errors.append("DD_API_KEY: missing")
                checks["DD_API_KEY"] = "missing"

            # DD_APP_KEY
            if os.getenv("DD_APP_KEY"):
                self.obs.log_info("DD_APP_KEY is set")
                self.results.append("DD_APP_KEY: configured")
                checks["DD_APP_KEY"] = "configured"
            else:
                self.obs.log_warning("DD_APP_KEY is not set")
                self.warnings.append("DD_APP_KEY: missing (required for API queries)")
                checks["DD_APP_KEY"] = "missing"

            # DD_SITE
            dd_site = os.getenv("DD_SITE", "datadoghq.com")
            self.obs.log_info(f"DD_SITE: {dd_site}")
            self.results.append(f"DD_SITE: {dd_site}")
            checks["DD_SITE"] = dd_site

            # DD_SERVICE
            dd_service = os.getenv("DD_SERVICE")
            if dd_service:
                self.obs.log_info(f"DD_SERVICE: {dd_service}")
                self.results.append(f"DD_SERVICE: {dd_service}")
                checks["DD_SERVICE"] = dd_service
            else:
                self.obs.log_warning("DD_SERVICE not set")
                self.warnings.append("DD_SERVICE: not set (recommended)")
                checks["DD_SERVICE"] = "not_set"

            # DD_ENV
            dd_env = os.getenv("DD_ENV")
            if dd_env:
                self.obs.log_info(f"DD_ENV: {dd_env}")
                self.results.append(f"DD_ENV: {dd_env}")
                checks["DD_ENV"] = dd_env
            else:
                self.obs.log_warning("DD_ENV not set")
                self.warnings.append("DD_ENV: not set (recommended)")
                checks["DD_ENV"] = "not_set"

            return checks

    def check_agent_connectivity(self) -> Dict[str, Any]:
        """Check Datadog Agent connectivity"""
        with self.obs.span("check_agent_connectivity"):
            agent_checks = {}

            dd_agent_host = os.getenv("DD_AGENT_HOST", "localhost")
            dd_trace_port = int(os.getenv("DD_TRACE_AGENT_PORT", "8126"))
            dd_statsd_port = int(os.getenv("DD_DOGSTATSD_PORT", "8125"))

            self.obs.log_info(f"Checking agent at {dd_agent_host}:{dd_trace_port}")

            # Check APM endpoint (HTTP)
            try:
                import requests
                response = requests.get(
                    f"http://{dd_agent_host}:{dd_trace_port}/info",
                    timeout=5
                )
                if response.status_code == 200:
                    self.obs.log_info("APM agent is reachable")
                    self.results.append(f"APM_AGENT: reachable at {dd_agent_host}:{dd_trace_port}")
                    agent_checks["apm"] = "reachable"
                else:
                    self.obs.log_warning("APM agent returned non-200 status")
                    self.warnings.append(f"APM_AGENT: unexpected status {response.status_code}")
                    agent_checks["apm"] = "warning"
            except Exception as e:
                self.obs.log_error(f"APM agent unreachable: {e}")
                self.errors.append(f"APM_AGENT: unreachable at {dd_agent_host}:{dd_trace_port}")
                agent_checks["apm"] = "unreachable"

            # Check DogStatsD port (UDP)
            try:
                sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
                sock.settimeout(1)
                sock.sendto(b"test", (dd_agent_host, dd_statsd_port))
                sock.close()
                self.obs.log_info("DogStatsD port is accessible")
                self.results.append(f"DOGSTATSD: accessible at {dd_agent_host}:{dd_statsd_port}")
                agent_checks["dogstatsd"] = "accessible"
            except Exception as e:
                self.obs.log_warning(f"DogStatsD port check failed: {e}")
                self.warnings.append(f"DOGSTATSD: cannot verify (UDP test failed)")
                agent_checks["dogstatsd"] = "unknown"

            agent_checks["host"] = dd_agent_host
            agent_checks["trace_port"] = dd_trace_port
            agent_checks["statsd_port"] = dd_statsd_port

            return agent_checks

    def check_api_connectivity(self) -> Dict[str, Any]:
        """Check Datadog API connectivity"""
        with self.obs.span("check_api_connectivity"):
            api_checks = {}

            if not os.getenv("DD_API_KEY") or not os.getenv("DD_APP_KEY"):
                self.obs.log_warning("Cannot check API - keys not set")
                api_checks["status"] = "skipped"
                return api_checks

            try:
                # Simple validation request
                import requests
                dd_site = os.getenv("DD_SITE", "datadoghq.com")
                response = requests.get(
                    f"https://api.{dd_site}/api/v1/validate",
                    headers={
                        "DD-API-KEY": os.getenv("DD_API_KEY"),
                        "DD-APPLICATION-KEY": os.getenv("DD_APP_KEY")
                    },
                    timeout=10
                )

                if response.status_code == 200:
                    data = response.json()
                    if data.get("valid"):
                        self.obs.log_info("API credentials are valid")
                        self.results.append("API_CONNECTIVITY: credentials valid")
                        api_checks["status"] = "valid"
                    else:
                        self.obs.log_error("API credentials are invalid")
                        self.errors.append("API_CONNECTIVITY: credentials invalid")
                        api_checks["status"] = "invalid"
                else:
                    self.obs.log_error(f"API validation failed with status {response.status_code}")
                    self.errors.append(f"API_CONNECTIVITY: validation failed ({response.status_code})")
                    api_checks["status"] = "failed"

            except Exception as e:
                self.obs.log_error(f"API connectivity check failed: {e}")
                self.errors.append(f"API_CONNECTIVITY: {str(e)}")
                api_checks["status"] = "error"
                api_checks["error"] = str(e)

            return api_checks

    def check_tracers(self) -> Dict[str, Any]:
        """Check for installed Datadog tracers"""
        with self.obs.span("check_tracers"):
            tracers = {}

            # Python
            try:
                if Path("requirements.txt").exists():
                    with open("requirements.txt", "r") as f:
                        content = f.read()
                        if "ddtrace" in content:
                            self.obs.log_info("Python tracer found in requirements.txt")
                            self.results.append("TRACER_PYTHON: installed")
                            tracers["python"] = "installed"
                        else:
                            self.obs.log_warning("Python project but no ddtrace in requirements.txt")
                            self.warnings.append("TRACER_PYTHON: not installed")
                            tracers["python"] = "not_installed"
            except Exception:
                pass

            # Node.js
            try:
                if Path("package.json").exists():
                    with open("package.json", "r") as f:
                        content = f.read()
                        if "dd-trace" in content:
                            self.obs.log_info("Node.js tracer found in package.json")
                            self.results.append("TRACER_NODEJS: installed")
                            tracers["nodejs"] = "installed"
                        else:
                            self.obs.log_warning("Node.js project but no dd-trace in package.json")
                            self.warnings.append("TRACER_NODEJS: not installed")
                            tracers["nodejs"] = "not_installed"
            except Exception:
                pass

            # .NET
            try:
                csproj_files = list(Path(".").glob("*.csproj"))
                if csproj_files:
                    found = False
                    for csproj in csproj_files:
                        with open(csproj, "r") as f:
                            if "Datadog.Trace" in f.read():
                                found = True
                                break
                    if found:
                        self.obs.log_info(".NET tracer found in project file")
                        self.results.append("TRACER_DOTNET: installed")
                        tracers["dotnet"] = "installed"
                    else:
                        self.obs.log_warning(".NET project but no Datadog.Trace")
                        self.warnings.append("TRACER_DOTNET: not installed")
                        tracers["dotnet"] = "not_installed"
            except Exception:
                pass

            return tracers

    def check_container_environment(self) -> Dict[str, Any]:
        """Check container environment configuration"""
        with self.obs.span("check_container_environment"):
            container_checks = {}

            # Check if running in container
            is_container = (
                Path("/.dockerenv").exists() or
                os.getenv("KUBERNETES_SERVICE_HOST") is not None
            )

            container_checks["is_container"] = is_container

            if is_container:
                self.obs.log_info("Container environment detected")

                if os.getenv("DD_AGENT_HOST"):
                    self.obs.log_info("DD_AGENT_HOST is set for container")
                    self.results.append("CONTAINER_CONFIG: DD_AGENT_HOST set")
                    container_checks["agent_host_configured"] = True
                else:
                    self.obs.log_error("DD_AGENT_HOST not set in container environment")
                    self.errors.append("CONTAINER_CONFIG: DD_AGENT_HOST not set (required in containers)")
                    container_checks["agent_host_configured"] = False

            return container_checks

    def execute(self) -> Dict[str, Any]:
        """Execute verification checks"""
        self.obs.log_info("Starting Datadog setup verification")

        # Run all checks
        env_checks = self.check_environment_variables()
        api_checks = self.check_api_connectivity()
        tracer_checks = self.check_tracers()
        container_checks = self.check_container_environment()

        # Only check agent if not skipped
        agent_checks = {}
        if not self.args.skip_agent_check:
            agent_checks = self.check_agent_connectivity()

        # Determine overall status
        status = "ok" if len(self.errors) == 0 else "error"

        # Record results
        self.obs.record_result("checks_passed", len(self.results))
        self.obs.record_result("warnings", len(self.warnings))
        self.obs.record_result("errors", len(self.errors))

        result = {
            "status": status,
            "summary": {
                "checks_passed": len(self.results),
                "warnings": len(self.warnings),
                "errors": len(self.errors)
            },
            "checks": {
                "environment": env_checks,
                "api": api_checks,
                "agent": agent_checks,
                "tracers": tracer_checks,
                "container": container_checks
            },
            "results": self.results,
            "warnings": self.warnings,
            "errors": self.errors,
            "recommendations": self.generate_recommendations()
        }

        return result

    def generate_recommendations(self) -> List[str]:
        """Generate recommendations based on findings"""
        recommendations = []

        if "DD_API_KEY: missing" in self.errors:
            recommendations.append("Set DD_API_KEY environment variable for Datadog API access")

        if "DD_SERVICE: not set (recommended)" in self.warnings:
            recommendations.append("Set DD_SERVICE to identify your service in Datadog")

        if "DD_ENV: not set (recommended)" in self.warnings:
            recommendations.append("Set DD_ENV to track environment (e.g., production, staging)")

        if any("TRACER" in w and "not installed" in w for w in self.warnings):
            recommendations.append("Install language-specific Datadog tracer for APM")

        if any("AGENT: unreachable" in e for e in self.errors):
            recommendations.append("Install and start Datadog Agent for metrics and traces")

        if any("CONTAINER" in e and "DD_AGENT_HOST" in e for e in self.errors):
            recommendations.append("Set DD_AGENT_HOST to agent service name in container environments")

        return recommendations

    def format_output(self, result: Dict[str, Any]) -> str:
        """Format result as conversational text"""
        formatter = OutputFormatter()
        lines = []

        # Header
        if result["status"] == "ok":
            lines.append("✅ Datadog Setup Verification")
        else:
            lines.append("⚠️ Datadog Setup Verification")

        lines.append("")

        # Summary
        lines.append("📊 Summary:")
        lines.append(f"  • Checks passed: {result['summary']['checks_passed']}")
        lines.append(f"  • Warnings: {result['summary']['warnings']}")
        lines.append(f"  • Errors: {result['summary']['errors']}")
        lines.append("")

        # Environment
        lines.append("🔧 Environment Variables:")
        env = result["checks"]["environment"]
        for key, value in env.items():
            if value == "missing":
                lines.append(f"  • {key}: ❌ Not set")
            elif value == "not_set":
                lines.append(f"  • {key}: ⚠️ Not set (optional)")
            else:
                lines.append(f"  • {key}: ✅ {value}")
        lines.append("")

        # API
        if result["checks"]["api"]:
            lines.append("🌐 API Connectivity:")
            api_status = result["checks"]["api"]["status"]
            if api_status == "valid":
                lines.append("  • ✅ API credentials are valid")
            elif api_status == "skipped":
                lines.append("  • ⚠️ Skipped (keys not set)")
            else:
                lines.append(f"  • ❌ {api_status}")
            lines.append("")

        # Agent
        if result["checks"]["agent"]:
            lines.append("🔗 Datadog Agent:")
            agent = result["checks"]["agent"]
            if agent.get("apm") == "reachable":
                lines.append(f"  • ✅ APM endpoint reachable at {agent['host']}:{agent['trace_port']}")
            elif agent.get("apm") == "unreachable":
                lines.append(f"  • ❌ APM endpoint unreachable at {agent['host']}:{agent['trace_port']}")

            if agent.get("dogstatsd") == "accessible":
                lines.append(f"  • ✅ DogStatsD accessible at {agent['host']}:{agent['statsd_port']}")
            elif agent.get("dogstatsd") == "unknown":
                lines.append(f"  • ⚠️ DogStatsD status unknown")
            lines.append("")

        # Tracers
        if result["checks"]["tracers"]:
            lines.append("📦 Installed Tracers:")
            for lang, status in result["checks"]["tracers"].items():
                emoji = "✅" if status == "installed" else "❌"
                lines.append(f"  • {emoji} {lang.upper()}: {status}")
            lines.append("")

        # Errors
        if result["errors"]:
            lines.append("❌ Errors:")
            for error in result["errors"]:
                lines.append(f"  • {error}")
            lines.append("")

        # Warnings
        if result["warnings"]:
            lines.append("⚠️ Warnings:")
            for warning in result["warnings"]:
                lines.append(f"  • {warning}")
            lines.append("")

        # Recommendations
        if result["recommendations"]:
            lines.append("💡 Recommendations:")
            for rec in result["recommendations"]:
                lines.append(f"  • {rec}")
            lines.append("")

        return "\n".join(lines)


def main():
    script = VerifySetupScript()
    script.run()


if __name__ == "__main__":
    main()
