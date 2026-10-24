#!/usr/bin/env python3
"""
Comprehensive Service Investigation.
Runs multiple Datadog checks and correlates results for a service.

Performs:
- APM performance analysis
- Security signal detection
- Watchdog anomaly detection
- Error log analysis
- SLO status check

Provides correlated insights and prioritized recommendations.
"""

import sys
import json
from pathlib import Path
from typing import Dict, Any, List, Optional
from datetime import datetime
import argparse

# Add lib to path
sys.path.insert(0, str(Path(__file__).parent / "lib"))

from base_script import QueryScript
from formatters import OutputFormatter
from dd_observability import get_observability


class InvestigateServiceScript(QueryScript):
    """Comprehensive service investigation with multiple checks"""

    def __init__(self):
        super().__init__(
            script_name="investigate-service",
            description="Comprehensive service investigation"
        )
        self.issues_found = 0
        self.warnings = 0
        self.critical = 0

    def setup_query_args(self, parser: argparse.ArgumentParser):
        """Setup investigation arguments"""
        parser.add_argument(
            "--skip-apm",
            action="store_true",
            help="Skip APM performance check"
        )
        parser.add_argument(
            "--skip-security",
            action="store_true",
            help="Skip security signals check"
        )
        parser.add_argument(
            "--skip-watchdog",
            action="store_true",
            help="Skip Watchdog anomaly check"
        )
        parser.add_argument(
            "--skip-logs",
            action="store_true",
            help="Skip log error analysis"
        )
        parser.add_argument(
            "--skip-slos",
            action="store_true",
            help="Skip SLO status check"
        )

    def check_apm_performance(self, service: str) -> Dict[str, Any]:
        """Check APM performance metrics"""
        with self.obs.span("check_apm_performance", tags={"service": service}):
            try:
                from_time, to_time = self.get_time_range(self.args.duration)

                with self.track_api_call("/api/v2/spans/analytics/aggregate", "POST"):
                    data = self.client.query_apm_traces(
                        service=service,
                        from_time=from_time,
                        to_time=to_time,
                        limit=20
                    )

                if "data" not in data or "buckets" not in data["data"]:
                    return {"status": "no_data", "slow_endpoints": 0, "error_count": 0}

                buckets = data["data"]["buckets"]
                endpoints = []

                for bucket in buckets:
                    resource_name = bucket.get("by", {}).get("resource_name", "unknown")
                    computes = bucket.get("computes", {})
                    p95_ns = computes.get("c2", 0)
                    count = computes.get("c0", 0)

                    endpoints.append({
                        "resource_name": resource_name,
                        "p95_ms": int(p95_ns / 1_000_000) if p95_ns else 0,
                        "count": count
                    })

                slow_endpoints = [e for e in endpoints if e["p95_ms"] > 500]

                status = "ok"
                if len(slow_endpoints) > 5:
                    status = "critical"
                    self.critical += 1
                    self.issues_found += 1
                elif len(slow_endpoints) > 0:
                    status = "warning"
                    self.warnings += 1
                    self.issues_found += 1

                self.obs.record_result("slow_endpoints", len(slow_endpoints))

                return {
                    "status": status,
                    "slow_endpoints": len(slow_endpoints),
                    "total_endpoints": len(endpoints),
                    "endpoints": slow_endpoints
                }

            except Exception as e:
                self.obs.log_error(f"APM check failed: {e}")
                return {"status": "error", "error": str(e)}

    def check_security_signals(self, service: str) -> Dict[str, Any]:
        """Check security monitoring signals"""
        with self.obs.span("check_security_signals", tags={"service": service}):
            try:
                from_time, to_time = self.get_time_range(self.args.duration)

                with self.track_api_call("/api/v2/security_monitoring/signals/search", "POST"):
                    data = self.client.get_security_signals(
                        from_time=from_time,
                        to_time=to_time,
                        service=service
                    )

                if "data" not in data:
                    return {"status": "ok", "critical": 0, "high": 0, "total": 0}

                signals = data["data"]
                critical_signals = [s for s in signals if s.get("attributes", {}).get("severity") == "critical"]
                high_signals = [s for s in signals if s.get("attributes", {}).get("severity") == "high"]

                status = "ok"
                if len(critical_signals) > 0:
                    status = "critical"
                    self.critical += 1
                    self.issues_found += 1
                elif len(high_signals) > 0:
                    status = "warning"
                    self.warnings += 1
                    self.issues_found += 1

                self.obs.record_result("security_signals", len(signals))

                return {
                    "status": status,
                    "critical": len(critical_signals),
                    "high": len(high_signals),
                    "total": len(signals),
                    "signals": [
                        {
                            "severity": s.get("attributes", {}).get("severity"),
                            "rule": s.get("attributes", {}).get("rule", {}).get("name"),
                            "timestamp": s.get("attributes", {}).get("timestamp")
                        }
                        for s in (critical_signals + high_signals)[:5]
                    ]
                }

            except Exception as e:
                self.obs.log_error(f"Security check failed: {e}")
                return {"status": "error", "error": str(e)}

    def check_watchdog_anomalies(self, service: str) -> Dict[str, Any]:
        """Check Watchdog anomaly detection"""
        with self.obs.span("check_watchdog_anomalies", tags={"service": service}):
            try:
                # Query logs for Watchdog alerts
                from_time, to_time = self.get_time_range(self.args.duration)
                query = f"service:{service} source:watchdog"

                with self.track_api_call("/api/v2/logs/events/search", "POST"):
                    data = self.client.search_logs(
                        query=query,
                        from_time=from_time,
                        to_time=to_time,
                        limit=100
                    )

                if "data" not in data:
                    return {"status": "ok", "latency_spikes": 0, "error_spikes": 0}

                logs = data["data"]

                # Categorize anomalies
                latency_anomalies = [
                    log for log in logs
                    if "latency" in str(log.get("attributes", {}).get("message", "")).lower()
                ]

                error_anomalies = [
                    log for log in logs
                    if "error" in str(log.get("attributes", {}).get("message", "")).lower()
                ]

                status = "ok"
                if len(error_anomalies) > 0:
                    status = "critical"
                    self.critical += 1
                    self.issues_found += 1
                elif len(latency_anomalies) > 0:
                    status = "warning"
                    self.warnings += 1
                    self.issues_found += 1

                self.obs.record_result("watchdog_anomalies", len(logs))

                return {
                    "status": status,
                    "latency_spikes": len(latency_anomalies),
                    "error_spikes": len(error_anomalies),
                    "total": len(logs)
                }

            except Exception as e:
                self.obs.log_error(f"Watchdog check failed: {e}")
                return {"status": "error", "error": str(e)}

    def check_error_logs(self, service: str) -> Dict[str, Any]:
        """Check error logs"""
        with self.obs.span("check_error_logs", tags={"service": service}):
            try:
                from_time, to_time = self.get_time_range(self.args.duration)
                query = f"service:{service} status:error"

                with self.track_api_call("/api/v2/logs/events/search", "POST"):
                    data = self.client.search_logs(
                        query=query,
                        from_time=from_time,
                        to_time=to_time,
                        limit=100
                    )

                if "data" not in data:
                    return {"status": "ok", "error_count": 0}

                error_count = len(data["data"])

                status = "ok"
                if error_count > 500:
                    status = "critical"
                    self.critical += 1
                    self.issues_found += 1
                elif error_count > 100:
                    status = "warning"
                    self.warnings += 1
                    self.issues_found += 1

                self.obs.record_result("error_logs", error_count)

                # Extract error patterns
                error_messages = []
                for log in data["data"][:10]:
                    msg = log.get("attributes", {}).get("message", "")
                    if msg:
                        error_messages.append(msg[:200])

                return {
                    "status": status,
                    "error_count": error_count,
                    "sample_errors": error_messages[:5]
                }

            except Exception as e:
                self.obs.log_error(f"Log check failed: {e}")
                return {"status": "error", "error": str(e)}

    def check_slo_status(self, service: str) -> Dict[str, Any]:
        """Check SLO status"""
        with self.obs.span("check_slo_status", tags={"service": service}):
            try:
                with self.track_api_call("/api/v1/slo", "GET"):
                    slos = self.client.get_slos(tags=[f"service:{service}"])

                if not slos:
                    return {"status": "no_slos", "breaching": 0, "total": 0}

                breaching_slos = []
                warning_slos = []

                for slo in slos:
                    slo_status = slo.get("sli_value", 100.0)
                    target = slo.get("target_threshold", 99.0)

                    if slo_status < target:
                        breaching_slos.append({
                            "name": slo.get("name"),
                            "status": slo_status,
                            "target": target
                        })
                    elif slo_status < target + 1:
                        warning_slos.append({
                            "name": slo.get("name"),
                            "status": slo_status,
                            "target": target
                        })

                status = "ok"
                if len(breaching_slos) > 0:
                    status = "critical"
                    self.critical += 1
                    self.issues_found += 1
                elif len(warning_slos) > 0:
                    status = "warning"
                    self.warnings += 1

                self.obs.record_result("slo_breaches", len(breaching_slos))

                return {
                    "status": status,
                    "breaching": len(breaching_slos),
                    "warning": len(warning_slos),
                    "total": len(slos),
                    "breaching_slos": breaching_slos,
                    "warning_slos": warning_slos
                }

            except Exception as e:
                self.obs.log_error(f"SLO check failed: {e}")
                return {"status": "error", "error": str(e)}

    def execute(self) -> Dict[str, Any]:
        """Execute comprehensive investigation"""
        # Detect service
        service = self.detect_service(self.args.service)

        self.obs.log_info(f"Investigating service: {service}")
        self.obs.log_info(f"Duration: {self.args.duration}")

        # Run all checks
        checks = {}

        if not self.args.skip_apm:
            self.obs.log_info("Running APM performance check...")
            checks["apm"] = self.check_apm_performance(service)

        if not self.args.skip_security:
            self.obs.log_info("Running security signals check...")
            checks["security"] = self.check_security_signals(service)

        if not self.args.skip_watchdog:
            self.obs.log_info("Running Watchdog anomaly check...")
            checks["watchdog"] = self.check_watchdog_anomalies(service)

        if not self.args.skip_logs:
            self.obs.log_info("Running error log analysis...")
            checks["logs"] = self.check_error_logs(service)

        if not self.args.skip_slos:
            self.obs.log_info("Running SLO status check...")
            checks["slos"] = self.check_slo_status(service)

        # Determine overall status
        overall_status = "healthy"
        if self.critical > 0:
            overall_status = "critical"
        elif self.warnings > 0:
            overall_status = "warning"

        # Generate recommendations
        recommendations = self.generate_recommendations(checks)

        result = {
            "service": service,
            "duration": self.args.duration,
            "timestamp": datetime.utcnow().isoformat() + "Z",
            "overall_status": overall_status,
            "issues_found": self.issues_found,
            "severity": {
                "critical": self.critical,
                "warnings": self.warnings
            },
            "checks": checks,
            "recommendations": recommendations
        }

        return result

    def generate_recommendations(self, checks: Dict[str, Any]) -> List[Dict[str, str]]:
        """Generate prioritized recommendations"""
        recommendations = []

        if self.critical > 0:
            recommendations.append({
                "priority": "critical",
                "action": "Immediate attention required - create incident and investigate"
            })

        # APM recommendations
        apm = checks.get("apm", {})
        if apm.get("status") in ["warning", "critical"]:
            recommendations.append({
                "priority": "high",
                "action": f"Optimize {apm.get('slow_endpoints', 0)} slow endpoints or increase capacity"
            })

        # Security recommendations
        security = checks.get("security", {})
        if security.get("critical", 0) > 0:
            recommendations.append({
                "priority": "critical",
                "action": "Security incident detected - engage security team immediately"
            })

        # Log recommendations
        logs = checks.get("logs", {})
        if logs.get("error_count", 0) > 100:
            recommendations.append({
                "priority": "high",
                "action": f"Investigate {logs.get('error_count')} error logs for patterns"
            })

        # SLO recommendations
        slos = checks.get("slos", {})
        if slos.get("breaching", 0) > 0:
            recommendations.append({
                "priority": "critical",
                "action": "SLO breach detected - escalate per SLO policy"
            })

        # Watchdog recommendations
        watchdog = checks.get("watchdog", {})
        if watchdog.get("error_spikes", 0) > 0:
            recommendations.append({
                "priority": "high",
                "action": "Error rate spike detected - investigate root cause"
            })

        return recommendations

    def format_output(self, result: Dict[str, Any]) -> str:
        """Format result as conversational text"""
        formatter = OutputFormatter()
        lines = []

        # Header
        status_emoji = {
            "healthy": "✅",
            "warning": "⚠️",
            "critical": "🚨"
        }.get(result["overall_status"], "❓")

        lines.append(f"{status_emoji} Service Investigation: {result['service']}")
        lines.append(f"Duration: {result['duration']} | Status: {result['overall_status'].upper()}")
        lines.append("")

        # Summary
        lines.append("📊 Summary:")
        lines.append(f"  • Issues found: {result['issues_found']}")
        lines.append(f"  • Critical: {result['severity']['critical']}")
        lines.append(f"  • Warnings: {result['severity']['warnings']}")
        lines.append("")

        # Checks
        checks = result["checks"]

        for check_name, check_data in checks.items():
            if check_data.get("status") == "error":
                continue

            status = check_data.get("status", "unknown")
            emoji = "✅" if status == "ok" else "⚠️" if status == "warning" else "🚨"

            lines.append(f"{emoji} {check_name.upper().replace('_', ' ')}:")

            if check_name == "apm":
                lines.append(f"  • Slow endpoints (>500ms): {check_data.get('slow_endpoints', 0)}")
                lines.append(f"  • Total endpoints: {check_data.get('total_endpoints', 0)}")

            elif check_name == "security":
                lines.append(f"  • Critical signals: {check_data.get('critical', 0)}")
                lines.append(f"  • High signals: {check_data.get('high', 0)}")

            elif check_name == "watchdog":
                lines.append(f"  • Latency spikes: {check_data.get('latency_spikes', 0)}")
                lines.append(f"  • Error spikes: {check_data.get('error_spikes', 0)}")

            elif check_name == "logs":
                lines.append(f"  • Error count: {check_data.get('error_count', 0)}")

            elif check_name == "slos":
                lines.append(f"  • Breaching: {check_data.get('breaching', 0)}")
                lines.append(f"  • Warning: {check_data.get('warning', 0)}")
                lines.append(f"  • Total SLOs: {check_data.get('total', 0)}")

            lines.append("")

        # Recommendations
        if result["recommendations"]:
            lines.append("💡 Recommendations:")
            for rec in result["recommendations"]:
                priority = rec["priority"]
                emoji = "🚨" if priority == "critical" else "⚠️" if priority == "high" else "ℹ️"
                lines.append(f"  {emoji} [{priority.upper()}] {rec['action']}")
            lines.append("")

        return "\n".join(lines)


def main():
    script = InvestigateServiceScript()
    script.run()


if __name__ == "__main__":
    main()
