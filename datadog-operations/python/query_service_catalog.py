#!/usr/bin/env python3
"""
Query Datadog Service Catalog.
List services, ownership, dependencies, and metadata.
"""

import sys
import json
from pathlib import Path
from typing import Dict, Any, List, Optional
import argparse

# Add lib to path
sys.path.insert(0, str(Path(__file__).parent / "lib"))

from base_script import QueryScript
from formatters import OutputFormatter, TableFormatter
from dd_observability import get_observability


class QueryServiceCatalogScript(QueryScript):
    """Query Service Catalog for service information"""

    def __init__(self):
        super().__init__(
            script_name="query-service-catalog",
            description="Query Datadog Service Catalog"
        )

    def setup_query_args(self, parser: argparse.ArgumentParser):
        """Setup service catalog query arguments"""
        parser.add_argument(
            "action",
            choices=["list", "get"],
            help="Action to perform: list or get"
        )
        parser.add_argument(
            "--team",
            help="Filter services by team"
        )
        parser.add_argument(
            "--tier",
            choices=["critical", "high", "medium", "low"],
            help="Filter services by tier"
        )
        parser.add_argument(
            "--kind",
            help="Filter services by kind (web, api, db, etc)"
        )

    def list_services(self) -> Dict[str, Any]:
        """List services from catalog"""
        with self.obs.span("list_services"):
            try:
                with self.track_api_call("/api/v2/services/definitions", "GET"):
                    import requests
                    dd_site = self.client.site
                    response = requests.get(
                        f"https://api.{dd_site}/api/v2/services/definitions",
                        headers={
                            "DD-API-KEY": self.client.api_key,
                            "DD-APPLICATION-KEY": self.client.app_key
                        },
                        timeout=30
                    )
                    response.raise_for_status()
                    data = response.json()

                if "data" not in data:
                    self.obs.log_warning("No services found in catalog")
                    return {"services": [], "total": 0}

                services = []
                for item in data["data"]:
                    attrs = item.get("attributes", {})

                    # Extract team and owner
                    contacts = attrs.get("contacts", [])
                    team = None
                    owner = None
                    for contact in contacts:
                        if contact.get("type") == "team":
                            team = contact.get("contact")
                        elif contact.get("type") == "email":
                            owner = contact.get("contact")

                    # Extract repos
                    links = attrs.get("links", [])
                    repos = [link["url"] for link in links if link.get("type") == "repo"]
                    docs = [link["url"] for link in links if link.get("type") == "doc"]

                    service = {
                        "name": item.get("id"),
                        "kind": attrs.get("kind"),
                        "description": attrs.get("description"),
                        "tier": attrs.get("tier"),
                        "lifecycle": attrs.get("lifecycle"),
                        "application": attrs.get("application"),
                        "team": team,
                        "owner": owner,
                        "languages": attrs.get("languages", []),
                        "tags": attrs.get("tags", []),
                        "repos": repos,
                        "docs": docs
                    }

                    services.append(service)

                self.obs.record_result("services", len(services))
                return {"services": services, "total": len(services)}

            except Exception as e:
                self.obs.log_error(f"Failed to list services: {e}")
                raise

    def get_service(self, service_name: str) -> Dict[str, Any]:
        """Get details for specific service"""
        with self.obs.span("get_service", tags={"service": service_name}):
            try:
                with self.track_api_call(f"/api/v2/services/definitions/{service_name}", "GET"):
                    import requests
                    dd_site = self.client.site
                    response = requests.get(
                        f"https://api.{dd_site}/api/v2/services/definitions/{service_name}",
                        headers={
                            "DD-API-KEY": self.client.api_key,
                            "DD-APPLICATION-KEY": self.client.app_key
                        },
                        timeout=30
                    )
                    response.raise_for_status()
                    data = response.json()

                if "data" not in data:
                    self.obs.log_warning(f"Service not found: {service_name}")
                    return None

                attrs = data["data"].get("attributes", {})

                # Extract contacts
                contacts = attrs.get("contacts", [])
                team = None
                owner = None
                for contact in contacts:
                    if contact.get("type") == "team":
                        team = contact.get("contact")
                    elif contact.get("type") == "email":
                        owner = contact.get("contact")

                # Extract links
                links = attrs.get("links", [])
                repos = [link["url"] for link in links if link.get("type") == "repo"]
                docs = [link["url"] for link in links if link.get("type") == "doc"]
                dashboards = [link["url"] for link in links if link.get("type") == "dashboard"]

                service = {
                    "name": data["data"].get("id"),
                    "kind": attrs.get("kind"),
                    "description": attrs.get("description"),
                    "tier": attrs.get("tier"),
                    "lifecycle": attrs.get("lifecycle"),
                    "application": attrs.get("application"),
                    "team": team,
                    "owner": owner,
                    "languages": attrs.get("languages", []),
                    "tags": attrs.get("tags", []),
                    "repos": repos,
                    "docs": docs,
                    "dashboards": dashboards,
                    "integrations": attrs.get("integrations", {}),
                    "schema_version": attrs.get("schema_version")
                }

                return service

            except Exception as e:
                self.obs.log_error(f"Failed to get service: {e}")
                raise

    def filter_services(self, services: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
        """Apply filters to service list"""
        filtered = services

        if self.args.team:
            filtered = [s for s in filtered if s.get("team") and self.args.team.lower() in s["team"].lower()]

        if self.args.tier:
            filtered = [s for s in filtered if s.get("tier") == self.args.tier]

        if self.args.kind:
            filtered = [s for s in filtered if s.get("kind") and self.args.kind.lower() in s["kind"].lower()]

        return filtered

    def execute(self) -> Dict[str, Any]:
        """Execute service catalog query"""
        action = self.args.action

        if action == "list":
            self.obs.log_info("Listing services from catalog")

            # Get all services
            data = self.list_services()
            services = data["services"]

            # Apply filters
            if self.args.team or self.args.tier or self.args.kind:
                services = self.filter_services(services)
                self.obs.log_info(f"Filtered to {len(services)} services")

            # Calculate statistics
            total = len(services)
            by_kind = {}
            by_tier = {}
            with_team = 0
            with_docs = 0

            for service in services:
                kind = service.get("kind", "unknown")
                by_kind[kind] = by_kind.get(kind, 0) + 1

                tier = service.get("tier", "unknown")
                by_tier[tier] = by_tier.get(tier, 0) + 1

                if service.get("team"):
                    with_team += 1

                if service.get("docs"):
                    with_docs += 1

            result = {
                "status": "ok",
                "action": "list",
                "summary": {
                    "total_services": total,
                    "by_kind": by_kind,
                    "by_tier": by_tier,
                    "with_team": with_team,
                    "with_docs": with_docs
                },
                "services": services
            }

            return result

        elif action == "get":
            if not self.args.service:
                self.error("--service is required for 'get' action")

            service_name = self.args.service
            self.obs.log_info(f"Getting service: {service_name}")

            service = self.get_service(service_name)

            if not service:
                return {
                    "status": "not_found",
                    "action": "get",
                    "service": service_name,
                    "error": "Service not found in catalog"
                }

            result = {
                "status": "ok",
                "action": "get",
                "service": service
            }

            return result

    def format_output(self, result: Dict[str, Any]) -> str:
        """Format result as conversational text"""
        formatter = OutputFormatter()
        lines = []

        if result["action"] == "list":
            # Header
            lines.append("📋 Service Catalog")
            lines.append("")

            # Summary
            summary = result["summary"]
            lines.append("📊 Summary:")
            lines.append(f"  • Total services: {summary['total_services']}")
            lines.append(f"  • With team assigned: {summary['with_team']}")
            lines.append(f"  • With documentation: {summary['with_docs']}")
            lines.append("")

            # By kind
            if summary["by_kind"]:
                lines.append("🏷️  By Kind:")
                for kind, count in sorted(summary["by_kind"].items(), key=lambda x: x[1], reverse=True):
                    lines.append(f"  • {kind or 'unknown'}: {count}")
                lines.append("")

            # By tier
            if summary["by_tier"]:
                lines.append("⭐ By Tier:")
                tier_order = {"critical": 1, "high": 2, "medium": 3, "low": 4, "unknown": 5}
                for tier, count in sorted(summary["by_tier"].items(), key=lambda x: tier_order.get(x[0], 99)):
                    lines.append(f"  • {tier or 'unknown'}: {count}")
                lines.append("")

            # Service list (top 10)
            if result["services"]:
                lines.append("🔍 Services (top 10):")
                for service in result["services"][:10]:
                    name = service["name"]
                    kind = service.get("kind", "unknown")
                    tier = service.get("tier", "")
                    team = service.get("team", "no team")

                    tier_emoji = {
                        "critical": "🔴",
                        "high": "🟠",
                        "medium": "🟡",
                        "low": "🟢"
                    }.get(tier, "⚪")

                    lines.append(f"  • {tier_emoji} {name}")
                    lines.append(f"    Kind: {kind} | Team: {team}")

                if len(result["services"]) > 10:
                    lines.append(f"  ... and {len(result['services']) - 10} more")
                lines.append("")

        elif result["action"] == "get":
            if result["status"] == "not_found":
                lines.append(f"❌ Service not found: {result['service']}")
                lines.append("")
                lines.append("The service may not be registered in the Service Catalog.")
                lines.append("Use 'dd catalog list' to see all available services.")
            else:
                service = result["service"]

                # Header
                tier_emoji = {
                    "critical": "🔴",
                    "high": "🟠",
                    "medium": "🟡",
                    "low": "🟢"
                }.get(service.get("tier"), "⚪")

                lines.append(f"{tier_emoji} Service: {service['name']}")
                lines.append("")

                # Basic info
                lines.append("ℹ️  Details:")
                if service.get("kind"):
                    lines.append(f"  • Kind: {service['kind']}")
                if service.get("tier"):
                    lines.append(f"  • Tier: {service['tier']}")
                if service.get("lifecycle"):
                    lines.append(f"  • Lifecycle: {service['lifecycle']}")
                if service.get("description"):
                    lines.append(f"  • Description: {service['description']}")
                lines.append("")

                # Ownership
                lines.append("👥 Ownership:")
                if service.get("team"):
                    lines.append(f"  • Team: {service['team']}")
                if service.get("owner"):
                    lines.append(f"  • Owner: {service['owner']}")
                if not service.get("team") and not service.get("owner"):
                    lines.append("  • No ownership information")
                lines.append("")

                # Tech stack
                if service.get("languages"):
                    lines.append("💻 Languages:")
                    for lang in service["languages"]:
                        lines.append(f"  • {lang}")
                    lines.append("")

                # Links
                if service.get("repos"):
                    lines.append("📦 Repositories:")
                    for repo in service["repos"]:
                        lines.append(f"  • {repo}")
                    lines.append("")

                if service.get("docs"):
                    lines.append("📚 Documentation:")
                    for doc in service["docs"]:
                        lines.append(f"  • {doc}")
                    lines.append("")

                if service.get("dashboards"):
                    lines.append("📊 Dashboards:")
                    for dashboard in service["dashboards"]:
                        lines.append(f"  • {dashboard}")
                    lines.append("")

                # Tags
                if service.get("tags"):
                    lines.append("🏷️  Tags:")
                    lines.append(f"  {', '.join(service['tags'])}")
                    lines.append("")

        return "\n".join(lines)


def main():
    script = QueryServiceCatalogScript()
    script.run()


if __name__ == "__main__":
    main()
