#!/usr/bin/env python3
"""
Query Datadog Database Monitoring.
Analyze database performance, slow queries, and connection metrics.
"""

import sys
import json
from pathlib import Path
from typing import Dict, Any, List
import argparse

# Add lib to path
sys.path.insert(0, str(Path(__file__).parent / "lib"))

from base_script import QueryScript
from formatters import OutputFormatter, ConversationalFormatter
from dd_observability import get_observability


class QueryDatabaseScript(QueryScript):
    """Query Database Monitoring for performance analysis"""

    def __init__(self):
        super().__init__(
            script_name="query-database",
            description="Query Datadog Database Monitoring"
        )

    def setup_query_args(self, parser: argparse.ArgumentParser):
        """Setup database query arguments"""
        parser.add_argument(
            "--host",
            required=True,
            help="Database host to query"
        )
        parser.add_argument(
            "--db-type",
            choices=["postgresql", "mysql", "mongodb", "redis"],
            default="postgresql",
            help="Database type (default: postgresql)"
        )
        parser.add_argument(
            "--show-queries",
            action="store_true",
            help="Show slow query details"
        )

    def query_connection_metrics(self, host: str, db_type: str) -> Dict[str, Any]:
        """Query database connection metrics"""
        with self.obs.span("query_connection_metrics"):
            from_time, to_time = self.get_time_range(self.args.duration)

            # Build metric query based on database type
            if db_type == "postgresql":
                query = f"avg:postgresql.connections.count{{host:{host}}}"
            elif db_type == "mysql":
                query = f"avg:mysql.net.connections{{host:{host}}}"
            elif db_type == "mongodb":
                query = f"avg:mongodb.connections.current{{host:{host}}}"
            elif db_type == "redis":
                query = f"avg:redis.clients.connected{{host:{host}}}"
            else:
                query = f"avg:database.connections{{host:{host}}}"

            try:
                with self.track_api_call("/api/v1/query", "GET"):
                    data = self.client.query_metrics(
                        query=query,
                        from_time=from_time,
                        to_time=to_time
                    )

                if "series" in data and len(data["series"]) > 0:
                    series = data["series"][0]
                    if "pointlist" in series and len(series["pointlist"]) > 0:
                        last_point = series["pointlist"][-1]
                        connection_count = int(last_point[1]) if last_point[1] else 0
                        self.obs.gauge("database.connections", connection_count)
                        return {"connection_count": connection_count}

            except Exception as e:
                self.obs.log_warning(f"Failed to fetch connection metrics: {e}")

            return {"connection_count": 0}

    def query_slow_queries(self, host: str, db_type: str) -> Dict[str, Any]:
        """Query slow database queries from APM"""
        with self.obs.span("query_slow_queries"):
            from_time, to_time = self.get_time_range(self.args.duration)

            from_ns = int(from_time.timestamp() * 1e9)
            to_ns = int(to_time.timestamp() * 1e9)

            # Query APM for database operations
            query = f"resource_type:sql host:{host}"

            payload = {
                "filter": {
                    "from": str(from_ns),
                    "to": str(to_ns),
                    "query": query
                },
                "compute": [
                    {"aggregation": "count", "metric": "*"},
                    {"aggregation": "pc95", "metric": "duration"},
                    {"aggregation": "avg", "metric": "duration"}
                ],
                "group_by": [
                    {
                        "facet": "resource_name",
                        "limit": 20,
                        "sort": {"order": "desc", "aggregation": "pc95", "metric": "duration"}
                    }
                ]
            }

            try:
                with self.track_api_call("/api/v2/spans/analytics/aggregate", "POST"):
                    import requests
                    dd_site = self.client.site
                    response = requests.post(
                        f"https://api.{dd_site}/api/v2/spans/analytics/aggregate",
                        headers={
                            "DD-API-KEY": self.client.api_key,
                            "DD-APPLICATION-KEY": self.client.app_key,
                            "Content-Type": "application/json"
                        },
                        json=payload,
                        timeout=30
                    )
                    response.raise_for_status()
                    data = response.json()

                if "data" in data and "buckets" in data["data"]:
                    buckets = data["data"]["buckets"]
                    queries = []

                    for bucket in buckets:
                        query_text = bucket.get("by", {}).get("resource_name", "unknown")
                        computes = bucket.get("computes", {})

                        count = computes.get("c0", 0)
                        p95_ns = computes.get("c1", 0)
                        avg_ns = computes.get("c2", 0)

                        queries.append({
                            "query": query_text[:200],  # Truncate long queries
                            "count": count,
                            "p95_ms": int(p95_ns / 1_000_000) if p95_ns else 0,
                            "avg_ms": int(avg_ns / 1_000_000) if avg_ns else 0
                        })

                    self.obs.record_result("slow_queries", len(queries))
                    return {"queries": queries, "count": len(queries)}

            except Exception as e:
                self.obs.log_warning(f"Failed to fetch slow queries: {e}")

            return {"queries": [], "count": 0}

    def execute(self) -> Dict[str, Any]:
        """Execute database query"""
        host = self.args.host
        db_type = self.args.db_type

        self.obs.log_info(f"Querying database: {host} ({db_type})")

        # Query connection metrics
        connection_data = self.query_connection_metrics(host, db_type)

        # Query slow queries
        slow_query_data = self.query_slow_queries(host, db_type)

        # Calculate statistics
        queries = slow_query_data.get("queries", [])
        slow_queries = [q for q in queries if q["p95_ms"] > 1000]  # > 1s

        slowest_p95 = max([q["p95_ms"] for q in queries], default=0)
        avg_duration = int(sum([q["avg_ms"] for q in queries]) / len(queries)) if queries else 0

        # Determine status
        status = "ok"
        issues = []

        if slowest_p95 > 5000:
            status = "critical"
            issues.append({
                "severity": "critical",
                "message": f"Very slow queries detected (P95: {slowest_p95}ms)",
                "action": "Investigate query optimization or add indexes"
            })
        elif slowest_p95 > 2000:
            status = "warning"
            issues.append({
                "severity": "warning",
                "message": f"Slow queries detected (P95: {slowest_p95}ms)",
                "action": "Consider query optimization"
            })

        result = {
            "status": status,
            "host": host,
            "db_type": db_type,
            "duration": self.args.duration,
            "summary": {
                "connection_count": connection_data.get("connection_count", 0),
                "query_patterns": len(queries),
                "slow_queries": len(slow_queries),
                "slowest_p95_ms": slowest_p95,
                "avg_duration_ms": avg_duration
            },
            "issues": issues
        }

        if self.args.show_queries:
            result["queries"] = queries

        return result

    def format_output(self, result: Dict[str, Any]) -> str:
        """Format result as conversational text"""
        formatter = OutputFormatter()
        lines = []

        # Header
        status_emoji = "✅" if result["status"] == "ok" else "⚠️" if result["status"] == "warning" else "🚨"
        lines.append(f"{status_emoji} Database Monitoring: {result['host']}")
        lines.append(f"Type: {result['db_type'].upper()} | Duration: {result['duration']}")
        lines.append("")

        # Summary
        summary = result["summary"]
        lines.append("📊 Summary:")
        lines.append(f"  • Active connections: {summary['connection_count']}")
        lines.append(f"  • Query patterns: {summary['query_patterns']}")
        lines.append(f"  • Slow queries (>1s): {summary['slow_queries']}")
        lines.append(f"  • Slowest P95: {summary['slowest_p95_ms']}ms")
        lines.append(f"  • Average duration: {summary['avg_duration_ms']}ms")
        lines.append("")

        # Issues
        if result.get("issues"):
            for issue in result["issues"]:
                severity = issue["severity"]
                emoji = "🚨" if severity == "critical" else "⚠️"
                lines.append(f"{emoji} {issue['message']}")
                lines.append(f"  Action: {issue['action']}")
            lines.append("")

        # Query details
        if self.args.show_queries and "queries" in result:
            lines.append("🔍 Slow Queries:")
            for i, query in enumerate(result["queries"][:5], 1):
                lines.append(f"  {i}. {query['query'][:80]}...")
                lines.append(f"     P95: {query['p95_ms']}ms | Avg: {query['avg_ms']}ms | Count: {query['count']:,}")
            if len(result["queries"]) > 5:
                lines.append(f"  ... and {len(result['queries']) - 5} more")
            lines.append("")

        # Recommendations
        recommendations = []
        if summary['slowest_p95_ms'] > 2000:
            recommendations.append("Optimize slow queries or add database indexes")
        if summary['connection_count'] > 100:
            recommendations.append("Consider connection pooling to reduce database load")
        if summary['query_patterns'] > 50:
            recommendations.append("High query diversity - review N+1 query patterns")

        if recommendations:
            lines.append("💡 Recommendations:")
            for rec in recommendations:
                lines.append(f"  • {rec}")
            lines.append("")

        return "\n".join(lines)


def main():
    script = QueryDatabaseScript()
    script.run()


if __name__ == "__main__":
    main()
