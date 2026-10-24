#!/usr/bin/env python3
"""
Query Datadog APM - Refactored with base classes.
Shows how much simpler scripts become with shared functionality.
"""

import sys
import argparse
from pathlib import Path
from typing import Dict, Any
from datetime import datetime

# Add lib to path
sys.path.insert(0, str(Path(__file__).parent / "lib"))

from base_script import QueryScript
from formatters import ConversationalFormatter, OutputFormatter


class QueryAPMScript(QueryScript):
    """Query APM traces for performance analysis"""

    def __init__(self):
        super().__init__(
            script_name="query-apm",
            description="Query Datadog APM for performance analysis"
        )

    def setup_query_args(self, parser: argparse.ArgumentParser):
        """Setup APM-specific arguments"""
        parser.add_argument(
            "--status",
            choices=["error", "ok", "all"],
            help="Filter by status (default: all)"
        )
        parser.add_argument(
            "--limit",
            type=int,
            default=20,
            help="Max endpoints to return (default: 20)"
        )

    def execute(self) -> Dict[str, Any]:
        """Execute APM query"""
        # Detect service
        service = self.detect_service(self.args.service)

        # Get time range
        from_time, to_time = self.get_time_range(self.args.duration)

        self.obs.log_info(f"Querying APM: service={service}, duration={self.args.duration}")

        # Query APM with tracking
        with self.track_api_call("/api/v2/spans/analytics/aggregate", "POST"):
            data = self.client.query_apm_traces(
                service=service,
                from_time=from_time,
                to_time=to_time,
                status=self.args.status if self.args.status != "all" else None,
                limit=self.args.limit
            )

        # Parse results
        if "data" not in data or "buckets" not in data["data"]:
            self.obs.log_warning("No trace data found")
            return {
                "status": "no_data",
                "service": service,
                "duration": self.args.duration,
                "endpoints": []
            }

        buckets = data["data"]["buckets"]
        endpoints = []

        for bucket in buckets:
            resource_name = bucket.get("by", {}).get("resource_name", "unknown")
            computes = bucket.get("computes", {})

            endpoints.append({
                "resource_name": resource_name,
                "request_count": computes.get("c0", 0),
                "p50_ms": int(computes.get("c1", 0) / 1_000_000),
                "p95_ms": int(computes.get("c2", 0) / 1_000_000),
                "p99_ms": int(computes.get("c3", 0) / 1_000_000)
            })

        # Calculate stats
        total_endpoints = len(endpoints)
        total_requests = sum(e["request_count"] for e in endpoints)
        avg_p95 = int(sum(e["p95_ms"] for e in endpoints) / total_endpoints) if total_endpoints > 0 else 0
        slow_endpoints = [e for e in endpoints if e["p95_ms"] > 500]

        # Record metrics
        self.obs.record_result("endpoints", total_endpoints)
        self.obs.record_result("requests", total_requests)
        self.obs.record_result("slow_endpoints", len(slow_endpoints))
        self.obs.gauge("apm.avg_p95_ms", avg_p95)

        return {
            "status": "ok",
            "service": service,
            "duration": self.args.duration,
            "summary": {
                "total_endpoints": total_endpoints,
                "total_requests": total_requests,
                "avg_p95_ms": avg_p95,
                "slow_endpoints_count": len(slow_endpoints)
            },
            "endpoints": endpoints,
            "slow_endpoints": slow_endpoints
        }

    def format_output(self, result: Dict[str, Any]) -> str:
        """Format output conversationally"""
        formatter = ConversationalFormatter(result["service"], "APM Analysis")
        out_fmt = OutputFormatter()

        if result["status"] == "no_data":
            return out_fmt.format_no_data(result["service"], "trace data")

        lines = [formatter.format_header()]

        lines.append(f"Duration: {result['duration']}")
        lines.append("")

        # Summary
        summary = result["summary"]
        lines.append("📊 Summary:")
        lines.append(out_fmt.format_metric("Endpoints", summary["total_endpoints"]))
        lines.append(out_fmt.format_metric("Requests", f"{summary['total_requests']:,}"))
        lines.append(out_fmt.format_metric("Avg P95", f"{summary['avg_p95_ms']}ms"))

        # Slow endpoints
        slow = result.get("slow_endpoints", [])
        if slow:
            lines.append("")
            lines.append(f"⚠️ {len(slow)} slow endpoints (P95 > 500ms):")
            for endpoint in slow[:5]:
                lines.append(f"  • {endpoint['resource_name']}")
                lines.append(f"    P95: {endpoint['p95_ms']}ms | {endpoint['request_count']:,} requests")
        else:
            lines.append("")
            lines.append("✅ All endpoints performing well")

        return "\n".join(lines)


if __name__ == "__main__":
    script = QueryAPMScript()
    script.run()
