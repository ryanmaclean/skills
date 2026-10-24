"""
Conversational output formatter - natural language summaries.
Makes Datadog data readable and actionable for AI conversations.
"""

from typing import Dict, List, Any, TYPE_CHECKING

if TYPE_CHECKING:
    from health_analyzer import HealthReport, HealthIssue
    from context_detector import ServiceContext


class ConversationalFormatter:
    """Format health reports and context as natural language"""

    @staticmethod
    def format_context(context: Any) -> str:
        """Format service context as natural language"""
        if not context.service_name:
            return "❌ Could not detect service context from your project"

        lines = [
            f"✓ Detected service: **{context.service_name}**",
            f"  Method: {context.detection_method}",
            f"  Confidence: {int(context.confidence * 100)}%"
        ]

        if context.repository:
            lines.append(f"  Repository: {context.repository}")

        if context.current_branch:
            lines.append(f"  Branch: {context.current_branch}")

        if context.last_commit_sha:
            lines.append(f"  Last commit: {context.last_commit_sha}")

        if context.environment:
            lines.append(f"  Environment: {context.environment}")

        return "\n".join(lines)

    @staticmethod
    def format_health_report(report: Any) -> str:
        """Format health report as conversational text"""

        status_emoji = {
            "healthy": "✅",
            "degraded": "⚠️",
            "critical": "🚨"
        }

        emoji = status_emoji.get(report.status, "❓")

        lines = [
            f"{emoji} Service: **{report.service}**",
            f"Status: **{report.status.upper()}**",
            ""
        ]

        # Summary metrics
        metrics = report.metrics
        if metrics:
            lines.append("📊 Metrics:")
            if "total_requests" in metrics:
                lines.append(f"  • {metrics['total_requests']:,} requests")
            if "endpoints_checked" in metrics:
                lines.append(f"  • {metrics['endpoints_checked']} endpoints checked")
            if "error_logs" in metrics:
                lines.append(f"  • {metrics['error_logs']} error logs")
            if "security_signals" in metrics:
                lines.append(f"  • {metrics['security_signals']} security signals")
            if "slos_checked" in metrics:
                lines.append(f"  • {metrics['slos_checked']} SLOs monitored")
            lines.append("")

        # Issues
        if report.issues:
            critical = [i for i in report.issues if i.severity == "critical"]
            warnings = [i for i in report.issues if i.severity == "warning"]

            if critical:
                lines.append("🚨 Critical Issues:")
                for issue in critical:
                    lines.append(f"  • {issue.message}")
                    if issue.action:
                        lines.append(f"    Action: {issue.action}")
                lines.append("")

            if warnings:
                lines.append("⚠️ Warnings:")
                for issue in warnings:
                    lines.append(f"  • {issue.message}")
                    if issue.action:
                        lines.append(f"    Action: {issue.action}")
                lines.append("")

        # Recommendations
        if report.recommendations:
            lines.append("💡 Recommendations:")
            for rec in report.recommendations:
                lines.append(f"  • {rec}")
            lines.append("")

        return "\n".join(lines)

    @staticmethod
    def format_deploy_readiness(
        report: Any,
        context: Any
    ) -> str:
        """Format deploy readiness check"""

        can_deploy = report.status in ["healthy", "degraded"]
        emoji = "✅" if can_deploy else "🛑"

        lines = [
            f"{emoji} Deploy Readiness: **{report.service}**",
            ""
        ]

        if can_deploy:
            lines.append("✓ Safe to deploy")
        else:
            lines.append("✗ Deployment NOT recommended")

        lines.append("")

        # Show blockers
        blockers = [
            issue for issue in report.issues
            if issue.severity == "critical"
        ]

        if blockers:
            lines.append("🚫 Blocking Issues:")
            for issue in blockers:
                lines.append(f"  • {issue.category}: {issue.message}")
            lines.append("")

        # Show concerns
        concerns = [
            issue for issue in report.issues
            if issue.severity == "warning"
        ]

        if concerns:
            lines.append("⚠️ Concerns:")
            for issue in concerns:
                lines.append(f"  • {issue.category}: {issue.message}")
            lines.append("")

        # Metrics summary
        metrics = report.metrics
        if metrics.get("error_logs", 0) > 0:
            lines.append(f"📈 Error rate: {metrics['error_logs']} errors in monitoring period")

        if metrics.get("slow_endpoints", 0) > 0:
            lines.append(f"🐌 Slow endpoints: {metrics['slow_endpoints']} endpoints > 500ms P95")

        if metrics.get("breaching_slos", 0) > 0:
            lines.append(f"📉 SLOs: {metrics['breaching_slos']} breaching targets")

        lines.append("")

        # Final recommendation
        if can_deploy:
            if report.status == "healthy":
                lines.append("💚 Go ahead and deploy!")
            else:
                lines.append("🟡 You can deploy, but monitor closely")
        else:
            lines.append("🔴 Fix critical issues before deploying")

        return "\n".join(lines)

    @staticmethod
    def format_quick_summary(report: Any) -> str:
        """Ultra-short summary for inline context"""

        if report.status == "healthy":
            return f"✅ {report.service} is healthy"

        issues_text = []
        metrics = report.metrics

        if metrics.get("error_logs", 0) > 10:
            issues_text.append(f"{metrics['error_logs']} errors")

        if metrics.get("slow_endpoints", 0) > 0:
            issues_text.append(f"{metrics['slow_endpoints']} slow endpoints")

        if metrics.get("breaching_slos", 0) > 0:
            issues_text.append(f"{metrics['breaching_slos']} SLOs breaching")

        if issues_text:
            return f"⚠️ {report.service}: {', '.join(issues_text)}"

        return f"ℹ️ {report.service} has minor issues"

    @staticmethod
    def format_error_details(issues: List[Any]) -> str:
        """Format error details for debugging"""

        error_issues = [i for i in issues if i.category == "errors"]
        if not error_issues:
            return "No error details available"

        lines = ["🔍 Error Details:", ""]

        for issue in error_issues:
            details = issue.details
            if "top_errors" in details:
                for error in details["top_errors"]:
                    msg = error["message"]
                    count = error["count"]
                    lines.append(f"  • [{count}x] {msg}")

        return "\n".join(lines)


def format_context(context: ServiceContext) -> str:
    """Convenience function"""
    return ConversationalFormatter.format_context(context)


def format_health_report(report: HealthReport) -> str:
    """Convenience function"""
    return ConversationalFormatter.format_health_report(report)


def format_deploy_readiness(report: HealthReport, context: ServiceContext) -> str:
    """Convenience function"""
    return ConversationalFormatter.format_deploy_readiness(report, context)


def format_quick_summary(report: HealthReport) -> str:
    """Convenience function"""
    return ConversationalFormatter.format_quick_summary(report)
