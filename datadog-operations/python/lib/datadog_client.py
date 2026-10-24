"""
Clean Datadog API client for observability queries.
Handles authentication, retries, and common query patterns.
"""

import os
import requests
from typing import Dict, List, Optional, Any
from datetime import datetime, timedelta
import time


class DatadogClient:
    """Simplified Datadog API client"""

    def __init__(
        self,
        api_key: Optional[str] = None,
        app_key: Optional[str] = None,
        site: str = "datadoghq.com"
    ):
        self.api_key = api_key or os.getenv("DD_API_KEY")
        self.app_key = app_key or os.getenv("DD_APP_KEY")
        self.site = site or os.getenv("DD_SITE", "datadoghq.com")

        if not self.api_key or not self.app_key:
            raise ValueError("DD_API_KEY and DD_APP_KEY must be set")

        self.base_url = f"https://api.{self.site}"
        self.session = requests.Session()
        self.session.headers.update({
            "DD-API-KEY": self.api_key,
            "DD-APPLICATION-KEY": self.app_key,
            "Content-Type": "application/json"
        })

    def _request(
        self,
        method: str,
        endpoint: str,
        **kwargs
    ) -> requests.Response:
        """Make API request with retry logic"""
        url = f"{self.base_url}{endpoint}"
        max_retries = 3
        retry_delay = 1

        for attempt in range(max_retries):
            try:
                response = self.session.request(method, url, timeout=30, **kwargs)

                # Retry on rate limit
                if response.status_code == 429:
                    if attempt < max_retries - 1:
                        time.sleep(retry_delay * (attempt + 1))
                        continue

                response.raise_for_status()
                return response

            except requests.exceptions.Timeout:
                if attempt < max_retries - 1:
                    time.sleep(retry_delay)
                    continue
                raise

            except requests.exceptions.RequestException as e:
                if attempt < max_retries - 1 and response.status_code >= 500:
                    time.sleep(retry_delay)
                    continue
                raise

        return response

    def query_apm_traces(
        self,
        service: str,
        from_time: datetime,
        to_time: datetime,
        status: Optional[str] = None,
        limit: int = 20
    ) -> Dict[str, Any]:
        """Query APM trace analytics"""

        # Calculate duration string for relative time format
        duration_seconds = int((to_time - from_time).total_seconds())
        if duration_seconds <= 3600:
            duration_str = "1h"
        elif duration_seconds <= 86400:
            duration_str = "24h"
        else:
            duration_str = "7d"

        query = f"service:{service}"
        if status:
            query += f" status:{status}"

        # Use correct Datadog API v2 format with data.attributes wrapper
        # Uses relative time format (now-1h, now) which is more reliable
        payload = {
            "data": {
                "type": "aggregate_request",
                "attributes": {
                    "filter": {
                        "from": f"now-{duration_str}",
                        "to": "now",
                        "query": query
                    },
                    "compute": [
                        {"aggregation": "count"},
                        {"aggregation": "avg", "metric": "@duration"},
                        {"aggregation": "max", "metric": "@duration"},
                        {"aggregation": "min", "metric": "@duration"}
                    ],
                    "group_by": [
                        {
                            "facet": "resource_name",
                            "limit": limit
                        }
                    ]
                }
            }
        }

        response = self._request("POST", "/api/v2/spans/analytics/aggregate", json=payload)
        return response.json()

    def search_logs(
        self,
        query: str,
        from_time: datetime,
        to_time: datetime,
        limit: int = 100
    ) -> Dict[str, Any]:
        """Search logs"""

        payload = {
            "filter": {
                "query": query,
                "from": from_time.isoformat(),
                "to": to_time.isoformat()
            },
            "sort": "timestamp",
            "page": {
                "limit": limit
            }
        }

        response = self._request("POST", "/api/v2/logs/events/search", json=payload)
        return response.json()

    def get_security_signals(
        self,
        from_time: datetime,
        to_time: datetime,
        service: Optional[str] = None
    ) -> Dict[str, Any]:
        """Get security monitoring signals"""

        query = ""
        if service:
            query = f"service:{service}"

        payload = {
            "filter": {
                "query": query,
                "from": from_time.isoformat(),
                "to": to_time.isoformat()
            },
            "sort": "-timestamp",
            "page": {
                "limit": 100
            }
        }

        response = self._request("POST", "/api/v2/security_monitoring/signals/search", json=payload)
        return response.json()

    def get_slos(self, tags: Optional[List[str]] = None) -> List[Dict[str, Any]]:
        """Get SLOs, optionally filtered by tags"""

        params = {}
        if tags:
            params["tags_query"] = ",".join(tags)

        response = self._request("GET", "/api/v1/slo", params=params)
        data = response.json()
        return data.get("data", [])

    def get_slo_history(
        self,
        slo_id: str,
        from_time: datetime,
        to_time: datetime
    ) -> Dict[str, Any]:
        """Get SLO history for error budget calculation"""

        from_ts = int(from_time.timestamp())
        to_ts = int(to_time.timestamp())

        response = self._request(
            "GET",
            f"/api/v1/slo/{slo_id}/history",
            params={"from_ts": from_ts, "to_ts": to_ts}
        )
        return response.json()

    def query_metrics(
        self,
        query: str,
        from_time: datetime,
        to_time: datetime
    ) -> Dict[str, Any]:
        """Query metrics"""

        from_ts = int(from_time.timestamp())
        to_ts = int(to_time.timestamp())

        params = {
            "query": query,
            "from": from_ts,
            "to": to_ts
        }

        response = self._request("GET", "/api/v1/query", params=params)
        return response.json()

    def get_monitors(
        self,
        tags: Optional[List[str]] = None,
        monitor_tags: Optional[List[str]] = None
    ) -> List[Dict[str, Any]]:
        """Get monitors"""

        params = {}
        if tags:
            params["tags"] = ",".join(tags)
        if monitor_tags:
            params["monitor_tags"] = ",".join(monitor_tags)

        response = self._request("GET", "/api/v1/monitor", params=params)
        return response.json()

    def create_incident(
        self,
        title: str,
        customer_impacted: bool = True,
        fields: Optional[Dict[str, Any]] = None
    ) -> Dict[str, Any]:
        """Create an incident"""

        payload = {
            "data": {
                "type": "incidents",
                "attributes": {
                    "title": title,
                    "customer_impacted": customer_impacted,
                    "fields": fields or {}
                }
            }
        }

        response = self._request("POST", "/api/v2/incidents", json=payload)
        return response.json()


def create_client() -> DatadogClient:
    """Create Datadog client from environment variables"""
    return DatadogClient()


if __name__ == "__main__":
    # Test connection
    try:
        client = create_client()
        print("✓ Datadog client initialized")
        print(f"  Site: {client.site}")
    except Exception as e:
        print(f"✗ Failed to initialize client: {e}")
