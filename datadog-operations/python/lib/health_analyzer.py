"""
Smart health analyzer - comprehensive service health checks.
Analyzes APM, logs, errors, and SLOs to give conversational health status.
"""

from typing import Dict, List, Optional, Any
from datetime import datetime, timedelta
from dataclasses import dataclass, field

try:
    from datadog_client import DatadogClient
    from context_detector import ServiceContext
except ImportError:
    from lib.datadog_client import DatadogClient
    from lib.context_detector import ServiceContext


@dataclass
class HealthIssue:
    """Represents a health issue"""
    severity: str  # critical, warning, info
    category: str  # performance, errors, security, slo
    message: str
    details: Dict[str, Any] = field(default_factory=dict)
    action: Optional[str] = None


@dataclass
class HealthReport:
    """Comprehensive health report"""
    status: str  # healthy, degraded, critical
    service: str
    checked_at: datetime
    issues: List[HealthIssue] = field(default_factory=list)
    metrics: Dict[str, Any] = field(default_factory=dict)
    recommendations: List[str] = field(default_factory=list)

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary"""
        return {
            "status": self.status,
            "service": self.service,
            "checked_at": self.checked_at.isoformat(),
            "issues": [
                {
                    "severity": issue.severity,
                    "category": issue.category,
                    "message": issue.message,
                    "details": issue.details,
                    "action": issue.action
                }
                for issue in self.issues
            ],
            "metrics": self.metrics,
            "recommendations": self.recommendations
        }


class HealthAnalyzer:
    """Analyzes service health across multiple signals"""

    def __init__(self, client: DatadogClient):
        self.client = client

    def analyze(
        self,
        context: ServiceContext,
        duration_hours: int = 1,
        check_since_deploy: bool = True
    ) -> HealthReport:
        """
        Perform comprehensive health analysis.

        Args:
            context: Service context
            duration_hours: How far back to look (default 1 hour)
            check_since_deploy: If True, check since last deploy time
        """
        service = context.service_name
        if not service:
            raise ValueError("Service name not detected in context")

        # Determine time range
        to_time = datetime.now()
        if check_since_deploy and context.last_deploy_time:
            # Check since last deploy, but cap at 24 hours
            from_time = max(
                context.last_deploy_time,
                to_time - timedelta(hours=24)
            )
        else:
            from_time = to_time - timedelta(hours=duration_hours)

        report = HealthReport(
            status="healthy",
            service=service,
            checked_at=to_time
        )

        # Run all checks
        self._check_apm_performance(service, from_time, to_time, report)
        self._check_error_logs(service, from_time, to_time, report)
        self._check_security_signals(service, from_time, to_time, report)
        self._check_slos(service, report)

        # Determine overall status
        report.status = self._calculate_overall_status(report)

        # Generate recommendations
        report.recommendations = self._generate_recommendations(report)

        return report

    def _check_apm_performance(
        self,
        service: str,
        from_time: datetime,
        to_time: datetime,
        report: HealthReport
    ):
        """Check APM trace performance"""
        try:
            data = self.client.query_apm_traces(
                service=service,
                from_time=from_time,
                to_time=to_time,
                limit=50
            )

            if "data" not in data or "buckets" not in data["data"]:
                return

            buckets = data["data"]["buckets"]
            if not buckets:
                report.metrics["apm_endpoints"] = 0
                return

            # Analyze endpoints
            slow_endpoints = []
            total_requests = 0

            for bucket in buckets:
                resource_name = bucket.get("by", {}).get("resource_name", "unknown")
                computes = bucket.get("computes", {})

                request_count = computes.get("c0", 0)
                p95_ns = computes.get("c2", 0)
                p95_ms = p95_ns / 1_000_000 if p95_ns else 0

                total_requests += request_count

                # Flag slow endpoints (P95 > 500ms)
                if p95_ms > 500:
                    slow_endpoints.append({
                        "endpoint": resource_name,
                        "p95_ms": int(p95_ms),
                        "requests": request_count
                    })

            report.metrics["total_requests"] = total_requests
            report.metrics["endpoints_checked"] = len(buckets)
            report.metrics["slow_endpoints"] = len(slow_endpoints)

            # Report slow endpoints
            if slow_endpoints:
                for endpoint in slow_endpoints[:5]:  # Top 5
                    report.issues.append(HealthIssue(
                        severity="warning",
                        category="performance",
                        message=f"Slow endpoint: {endpoint['endpoint']}",
                        details={
                            "p95_latency_ms": endpoint["p95_ms"],
                            "request_count": endpoint["requests"]
                        },
                        action="Investigate performance bottleneck or scale resources"
                    ))

        except Exception as e:
            report.issues.append(HealthIssue(
                severity="info",
                category="monitoring",
                message=f"Could not check APM performance: {str(e)}"
            ))

    def _check_error_logs(
        self,
        service: str,
        from_time: datetime,
        to_time: datetime,
        report: HealthReport
    ):
        """Check error logs"""
        try:
            # Query for errors
            data = self.client.search_logs(
                query=f"service:{service} status:error",
                from_time=from_time,
                to_time=to_time,
                limit=100
            )

            logs = data.get("data", [])
            error_count = len(logs)

            report.metrics["error_logs"] = error_count

            if error_count > 0:
                # Analyze error patterns
                error_messages = {}
                for log in logs:
                    attrs = log.get("attributes", {})
                    message = attrs.get("message", "")[:100]
                    error_messages[message] = error_messages.get(message, 0) + 1

                # Find most common errors
                top_errors = sorted(
                    error_messages.items(),
                    key=lambda x: x[1],
                    reverse=True
                )[:3]

                severity = "critical" if error_count > 50 else "warning"

                report.issues.append(HealthIssue(
                    severity=severity,
                    category="errors",
                    message=f"Found {error_count} error logs",
                    details={
                        "error_count": error_count,
                        "top_errors": [
                            {"message": msg, "count": count}
                            for msg, count in top_errors
                        ]
                    },
                    action="Review error logs and fix recurring issues"
                ))

        except Exception as e:
            report.issues.append(HealthIssue(
                severity="info",
                category="monitoring",
                message=f"Could not check error logs: {str(e)}"
            ))

    def _check_security_signals(
        self,
        service: str,
        from_time: datetime,
        to_time: datetime,
        report: HealthReport
    ):
        """Check security monitoring signals"""
        try:
            data = self.client.get_security_signals(
                from_time=from_time,
                to_time=to_time,
                service=service
            )

            signals = data.get("data", [])
            signal_count = len(signals)

            report.metrics["security_signals"] = signal_count

            if signal_count > 0:
                # Count by severity
                severities = {}
                for signal in signals:
                    attrs = signal.get("attributes", {})
                    severity = attrs.get("severity", "unknown")
                    severities[severity] = severities.get(severity, 0) + 1

                critical_count = severities.get("critical", 0)
                high_count = severities.get("high", 0)

                if critical_count > 0 or high_count > 0:
                    report.issues.append(HealthIssue(
                        severity="critical" if critical_count > 0 else "warning",
                        category="security",
                        message=f"Security signals detected",
                        details={
                            "critical": critical_count,
                            "high": high_count,
                            "total": signal_count
                        },
                        action="Review security signals immediately"
                    ))

        except Exception as e:
            report.issues.append(HealthIssue(
                severity="info",
                category="monitoring",
                message=f"Could not check security signals: {str(e)}"
            ))

    def _check_slos(self, service: str, report: HealthReport):
        """Check SLO status"""
        try:
            slos = self.client.get_slos(tags=[f"service:{service}"])

            report.metrics["slos_checked"] = len(slos)

            if not slos:
                return

            breaching_slos = []

            for slo in slos:
                attrs = slo.get("attributes", {})
                name = attrs.get("name", "unknown")
                slo_value = attrs.get("slo_value")
                target = attrs.get("target_threshold")

                if slo_value is not None and target is not None:
                    if slo_value < target:
                        breaching_slos.append({
                            "name": name,
                            "current": slo_value,
                            "target": target
                        })

            report.metrics["breaching_slos"] = len(breaching_slos)

            if breaching_slos:
                report.issues.append(HealthIssue(
                    severity="critical",
                    category="slo",
                    message=f"{len(breaching_slos)} SLO(s) breaching target",
                    details={"slos": breaching_slos},
                    action="Review SLO breaches and take corrective action"
                ))

        except Exception as e:
            report.issues.append(HealthIssue(
                severity="info",
                category="monitoring",
                message=f"Could not check SLOs: {str(e)}"
            ))

    def _calculate_overall_status(self, report: HealthReport) -> str:
        """Calculate overall health status from issues"""
        critical_count = sum(1 for issue in report.issues if issue.severity == "critical")
        warning_count = sum(1 for issue in report.issues if issue.severity == "warning")

        if critical_count > 0:
            return "critical"
        elif warning_count > 0:
            return "degraded"
        else:
            return "healthy"

    def _generate_recommendations(self, report: HealthReport) -> List[str]:
        """Generate actionable recommendations"""
        recommendations = []

        # Check error rate
        error_count = report.metrics.get("error_logs", 0)
        if error_count > 50:
            recommendations.append(
                f"High error rate ({error_count} errors) - investigate root cause before deploying"
            )

        # Check slow endpoints
        slow_count = report.metrics.get("slow_endpoints", 0)
        if slow_count > 5:
            recommendations.append(
                f"{slow_count} slow endpoints detected - consider optimization or scaling"
            )

        # Check SLOs
        breaching = report.metrics.get("breaching_slos", 0)
        if breaching > 0:
            recommendations.append(
                "SLOs breaching - deployment not recommended until issues resolved"
            )

        # Check security
        security_signals = report.metrics.get("security_signals", 0)
        if security_signals > 0:
            recommendations.append(
                "Security signals detected - review before deploying changes"
            )

        if not recommendations:
            recommendations.append("Service health looks good")

        return recommendations
