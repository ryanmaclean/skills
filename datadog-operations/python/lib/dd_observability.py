"""
Built-in Datadog observability for all scripts.
Every script automatically sends traces, logs, and metrics to Datadog.
"""

import os
import time
import json
import socket
import requests
from typing import Dict, Any, Optional, List
from datetime import datetime
from contextlib import contextmanager


class DatadogObservability:
    """
    Built-in Datadog observability for all scripts.
    Sends traces, logs, and metrics automatically.
    """

    def __init__(
        self,
        script_name: str,
        service: str = "datadog-skill",
        env: Optional[str] = None
    ):
        self.script_name = script_name
        self.service = service
        self.env = env or os.getenv("DD_ENV", "production")
        self.api_key = os.getenv("DD_API_KEY")
        self.site = os.getenv("DD_SITE", "datadoghq.com")

        # Session for HTTP requests
        self.session = requests.Session()

        # Track spans for tracing
        self.trace_id = self._generate_trace_id()
        self.span_id_counter = 0
        self.spans: List[Dict[str, Any]] = []

        # Script start time
        self.start_time = time.time()

        # Send script start log
        self._send_log("info", f"Script started: {script_name}")

    def _generate_trace_id(self) -> int:
        """Generate unique trace ID"""
        return int(time.time() * 1000000) % (2**63 - 1)

    def _generate_span_id(self) -> int:
        """Generate unique span ID"""
        self.span_id_counter += 1
        return (self.trace_id + self.span_id_counter) % (2**63 - 1)

    def _send_log(
        self,
        level: str,
        message: str,
        tags: Optional[Dict[str, Any]] = None
    ):
        """Send structured log to Datadog"""
        if not self.api_key:
            return

        log_entry = {
            "ddsource": "datadog-skill",
            "ddtags": f"env:{self.env},service:{self.service},script:{self.script_name}",
            "hostname": socket.gethostname(),
            "message": message,
            "level": level,
            "timestamp": int(time.time() * 1000),
            "script": self.script_name,
            "service": self.service,
            "env": self.env
        }

        if tags:
            log_entry.update(tags)

        try:
            self.session.post(
                f"https://http-intake.logs.{self.site}/api/v2/logs",
                headers={
                    "DD-API-KEY": self.api_key,
                    "Content-Type": "application/json"
                },
                json=[log_entry],
                timeout=5
            )
        except Exception:
            # Never fail script due to observability
            pass

    def _send_metric(
        self,
        metric_name: str,
        value: float,
        tags: Optional[List[str]] = None,
        metric_type: str = "gauge"
    ):
        """Send metric to Datadog"""
        if not self.api_key:
            return

        all_tags = [
            f"env:{self.env}",
            f"service:{self.service}",
            f"script:{self.script_name}"
        ]
        if tags:
            all_tags.extend(tags)

        payload = {
            "series": [{
                "metric": f"datadog.skill.{metric_name}",
                "points": [[int(time.time()), value]],
                "type": metric_type,
                "tags": all_tags
            }]
        }

        try:
            self.session.post(
                f"https://api.{self.site}/api/v2/series",
                headers={
                    "DD-API-KEY": self.api_key,
                    "Content-Type": "application/json"
                },
                json=payload,
                timeout=5
            )
        except Exception:
            pass

    def _send_trace(self):
        """Send trace spans to Datadog Agent"""
        if not self.spans:
            return

        # Try to send to local DD agent first (port 8126)
        try:
            response = requests.put(
                "http://localhost:8126/v0.4/traces",
                headers={"Content-Type": "application/json"},
                json=[self.spans],
                timeout=2
            )
            if response.status_code == 200:
                return
        except Exception:
            pass

        # Fallback: log trace information
        self._send_log(
            "info",
            f"Trace completed: {len(self.spans)} spans",
            {"trace_id": self.trace_id, "span_count": len(self.spans)}
        )

    @contextmanager
    def span(
        self,
        operation_name: str,
        resource: Optional[str] = None,
        tags: Optional[Dict[str, Any]] = None
    ):
        """Context manager for creating trace spans"""
        span_id = self._generate_span_id()
        start_time = time.time_ns()

        span_data = {
            "trace_id": self.trace_id,
            "span_id": span_id,
            "parent_id": self.spans[-1]["span_id"] if self.spans else 0,
            "name": operation_name,
            "resource": resource or operation_name,
            "service": self.service,
            "start": start_time,
            "meta": {
                "env": self.env,
                "script": self.script_name,
                **(tags or {})
            }
        }

        error = None
        try:
            yield span_data
        except Exception as e:
            error = e
            span_data["error"] = 1
            span_data["meta"]["error.type"] = type(e).__name__
            span_data["meta"]["error.message"] = str(e)
            raise
        finally:
            end_time = time.time_ns()
            span_data["duration"] = end_time - start_time

            # Send metric for operation duration
            duration_ms = (end_time - start_time) / 1_000_000
            self._send_metric(
                "operation.duration",
                duration_ms,
                tags=[
                    f"operation:{operation_name}",
                    f"status:{'error' if error else 'ok'}"
                ]
            )

            self.spans.append(span_data)

    def log_info(self, message: str, **kwargs):
        """Log info message"""
        self._send_log("info", message, kwargs)

    def log_warning(self, message: str, **kwargs):
        """Log warning message"""
        self._send_log("warning", message, kwargs)

    def log_error(self, message: str, **kwargs):
        """Log error message"""
        self._send_log("error", message, kwargs)

    def count(self, metric_name: str, value: int = 1, tags: Optional[List[str]] = None):
        """Send count metric"""
        self._send_metric(metric_name, value, tags, metric_type="count")

    def gauge(self, metric_name: str, value: float, tags: Optional[List[str]] = None):
        """Send gauge metric"""
        self._send_metric(metric_name, value, tags, metric_type="gauge")

    def finalize(self, exit_code: int = 0):
        """Finalize observability - call at script end"""
        duration_s = time.time() - self.start_time

        # Send completion metric
        self._send_metric(
            "execution.duration",
            duration_s * 1000,  # milliseconds
            tags=[f"exit_code:{exit_code}"]
        )

        # Send completion log
        self._send_log(
            "info" if exit_code == 0 else "error",
            f"Script completed: {self.script_name}",
            {
                "duration_ms": duration_s * 1000,
                "exit_code": exit_code
            }
        )

        # Send trace
        self._send_trace()

    def record_api_call(
        self,
        endpoint: str,
        method: str = "GET",
        status_code: Optional[int] = None,
        duration_ms: Optional[float] = None,
        error: Optional[str] = None
    ):
        """Record API call metrics"""
        tags = [
            f"endpoint:{endpoint}",
            f"method:{method}",
            f"status:{status_code}" if status_code else "status:unknown"
        ]

        if error:
            tags.append("error:true")
            self.log_error(f"API call failed: {endpoint}", error=error)

        self.count("api.calls", 1, tags)

        if duration_ms:
            self.gauge("api.duration", duration_ms, tags)

    def record_result(
        self,
        result_type: str,
        count: int,
        tags: Optional[List[str]] = None
    ):
        """Record result metrics (errors found, endpoints checked, etc)"""
        all_tags = [f"result_type:{result_type}"]
        if tags:
            all_tags.extend(tags)

        self.gauge(f"results.{result_type}", count, all_tags)


# Global observability instance
_dd_obs: Optional[DatadogObservability] = None


def init_observability(script_name: str) -> DatadogObservability:
    """Initialize observability for a script"""
    global _dd_obs
    _dd_obs = DatadogObservability(script_name)
    return _dd_obs


def get_observability() -> Optional[DatadogObservability]:
    """Get current observability instance"""
    return _dd_obs


def finalize_observability(exit_code: int = 0):
    """Finalize observability"""
    if _dd_obs:
        _dd_obs.finalize(exit_code)
