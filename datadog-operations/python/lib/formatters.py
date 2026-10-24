"""
Common output formatters for conversational and structured output.
Consolidates all formatting logic to avoid duplication.
"""

from typing import Dict, Any, List, Optional


class OutputFormatter:
    """Common output formatting utilities"""

    @staticmethod
    def format_count(count: int, singular: str, plural: Optional[str] = None) -> str:
        """Format count with proper pluralization"""
        if plural is None:
            plural = f"{singular}s"
        return f"{count:,} {singular if count == 1 else plural}"

    @staticmethod
    def format_duration_ms(ms: float) -> str:
        """Format duration in milliseconds"""
        if ms < 1000:
            return f"{int(ms)}ms"
        else:
            return f"{ms / 1000:.2f}s"

    @staticmethod
    def format_percentage(value: float) -> str:
        """Format percentage"""
        return f"{value:.1f}%"

    @staticmethod
    def format_status(status: str) -> str:
        """Format status with emoji"""
        emojis = {
            "ok": "✅",
            "healthy": "✅",
            "warning": "⚠️",
            "degraded": "⚠️",
            "error": "🚨",
            "critical": "🚨",
            "info": "ℹ️",
            "unknown": "❓"
        }
        emoji = emojis.get(status.lower(), "")
        return f"{emoji} {status.upper()}" if emoji else status.upper()

    @staticmethod
    def format_section_header(title: str, emoji: str = "") -> str:
        """Format section header"""
        prefix = f"{emoji} " if emoji else ""
        return f"\n{prefix}{title}"

    @staticmethod
    def format_bullet(text: str, indent: int = 0) -> str:
        """Format bullet point"""
        spacing = "  " * indent
        return f"{spacing}• {text}"

    @staticmethod
    def format_metric(label: str, value: Any, indent: int = 0) -> str:
        """Format metric line"""
        spacing = "  " * indent
        return f"{spacing}• {label}: {value}"

    @staticmethod
    def format_error_list(errors: List[Dict[str, Any]], limit: int = 5) -> str:
        """Format list of errors"""
        lines = []
        for i, error in enumerate(errors[:limit]):
            message = error.get("message", "Unknown error")[:80]
            count = error.get("count", 1)
            lines.append(f"  • [{count}x] {message}")

        if len(errors) > limit:
            lines.append(f"  ... and {len(errors) - limit} more")

        return "\n".join(lines)

    @staticmethod
    def format_endpoint_list(endpoints: List[Dict[str, Any]], limit: int = 5) -> str:
        """Format list of endpoints"""
        lines = []
        for endpoint in endpoints[:limit]:
            name = endpoint.get("resource_name", "unknown")
            p95_ms = endpoint.get("p95_ms", 0)
            requests = endpoint.get("request_count", 0)
            lines.append(f"  • {name}")
            lines.append(f"    P95: {p95_ms}ms | {requests:,} requests")

        if len(endpoints) > limit:
            lines.append(f"  ... and {len(endpoints) - limit} more")

        return "\n".join(lines)


class ResultFormatter:
    """Format standard result structures"""

    @staticmethod
    def format_summary(result: Dict[str, Any]) -> str:
        """Format summary section"""
        lines = ["\n📊 Summary:"]

        summary = result.get("summary", {})
        for key, value in summary.items():
            # Convert snake_case to Title Case
            label = key.replace("_", " ").title()

            # Format value based on type
            if isinstance(value, int):
                formatted_value = f"{value:,}"
            elif isinstance(value, float):
                formatted_value = f"{value:.2f}"
            else:
                formatted_value = str(value)

            lines.append(f"  • {label}: {formatted_value}")

        return "\n".join(lines)

    @staticmethod
    def format_issues(issues: List[Dict[str, Any]]) -> str:
        """Format issues section"""
        if not issues:
            return ""

        # Group by severity
        critical = [i for i in issues if i.get("severity") == "critical"]
        warnings = [i for i in issues if i.get("severity") == "warning"]

        lines = []

        if critical:
            lines.append("\n🚨 Critical Issues:")
            for issue in critical:
                lines.append(f"  • {issue.get('message', 'Unknown issue')}")
                if "action" in issue:
                    lines.append(f"    Action: {issue['action']}")

        if warnings:
            lines.append("\n⚠️ Warnings:")
            for issue in warnings:
                lines.append(f"  • {issue.get('message', 'Unknown issue')}")
                if "action" in issue:
                    lines.append(f"    Action: {issue['action']}")

        return "\n".join(lines)

    @staticmethod
    def format_recommendations(recommendations: List[str]) -> str:
        """Format recommendations section"""
        if not recommendations:
            return ""

        lines = ["\n💡 Recommendations:"]
        for rec in recommendations:
            lines.append(f"  • {rec}")

        return "\n".join(lines)

    @staticmethod
    def format_no_data(service: str, resource_type: str = "data") -> str:
        """Format no data message"""
        return f"ℹ️ No {resource_type} found for service: {service}"


class TableFormatter:
    """Format data as simple tables"""

    @staticmethod
    def format_simple_table(
        data: List[Dict[str, Any]],
        columns: List[tuple[str, str]],  # (key, header)
        max_width: int = 80
    ) -> str:
        """Format data as simple aligned table"""
        if not data:
            return ""

        lines = []

        # Header
        headers = [h for _, h in columns]
        lines.append("  " + " | ".join(headers))
        lines.append("  " + "-" * min(max_width, len(lines[0])))

        # Rows
        for row in data:
            values = []
            for key, _ in columns:
                value = row.get(key, "")
                if isinstance(value, int):
                    values.append(f"{value:,}")
                elif isinstance(value, float):
                    values.append(f"{value:.2f}")
                else:
                    values.append(str(value))

            lines.append("  " + " | ".join(values))

        return "\n".join(lines)


class ConversationalFormatter:
    """High-level conversational output formatter"""

    def __init__(self, service: str, operation: str):
        self.service = service
        self.operation = operation
        self.formatter = OutputFormatter()
        self.result_formatter = ResultFormatter()

    def format_header(self, status: str = "ok") -> str:
        """Format conversational header"""
        emoji = "📊"
        if status in ["critical", "error"]:
            emoji = "🚨"
        elif status == "warning":
            emoji = "⚠️"

        return f"{emoji} {self.operation.replace('_', ' ').title()}: {self.service}"

    def format_result(self, result: Dict[str, Any]) -> str:
        """Format complete result conversationally"""
        lines = []

        # Header
        status = result.get("status", "ok")
        lines.append(self.format_header(status))

        # Summary
        if "summary" in result:
            lines.append(self.result_formatter.format_summary(result))

        # Issues
        if "issues" in result:
            lines.append(self.result_formatter.format_issues(result["issues"]))

        # Recommendations
        if "recommendations" in result:
            lines.append(self.result_formatter.format_recommendations(result["recommendations"]))

        # Custom sections
        if "details" in result:
            lines.append("\n🔍 Details:")
            details = result["details"]
            if isinstance(details, dict):
                for key, value in details.items():
                    label = key.replace("_", " ").title()
                    lines.append(f"  • {label}: {value}")
            elif isinstance(details, list):
                for item in details:
                    lines.append(f"  • {item}")

        return "\n".join(lines) + "\n"
