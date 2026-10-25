#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import os
import sys
import time
import urllib.parse
import urllib.request
from typing import Any


def q(series: str, host: str, t_from: int, t_to: int, site: str,
      api_key: str, app_key: str) -> dict[str, Any]:
    query = f"avg:{series}" + "{host:" + host + "}"
    url = (
        f"https://api.{site}/api/v1/query?from={t_from}&to={t_to}&query="
        + urllib.parse.quote(query, safe="")
    )
    req = urllib.request.Request(
        url,
        headers={
            "DD-API-KEY": api_key,
            "DD-APPLICATION-KEY": app_key,
        },
    )
    with urllib.request.urlopen(req, timeout=20) as resp:
        return json.loads(resp.read().decode("utf-8"))


def latest_value_and_age(payload: dict[str, Any], now: int) -> tuple[Any, Any]:
    series = payload.get("series") or []
    if not series:
        return None, None
    pts = [p for p in (series[0].get("pointlist") or []) if p[1] is not None]
    if not pts:
        return None, None
    ts_ms, val = pts[-1]
    age_s = max(0, now - int(ts_ms // 1000))
    return val, age_s


def main() -> int:
    p = argparse.ArgumentParser(
        description="Check Datadog Metal/DCGM shim metric liveness by host"
    )
    p.add_argument(
        "--hosts",
        nargs="+",
        default=["mbp-m1.local", "studio.local", "minim4-24.local"],
    )
    p.add_argument(
        "--metrics",
        nargs="+",
        default=[
            "dcgm.power_usage",
            "dcgm.gpu_utilization",
            "dcgm.temperature",
        ],
    )
    p.add_argument("--window-sec", type=int, default=900)
    p.add_argument("--site", default=os.getenv("DD_SITE", "datadoghq.com"))
    args = p.parse_args()

    api_key = os.getenv("DD_API_KEY")
    app_key = os.getenv("DD_APP_KEY")
    if not api_key or not app_key:
        print("Missing DD_API_KEY or DD_APP_KEY", file=sys.stderr)
        return 2

    now = int(time.time())
    t_from = now - args.window_sec

    print("host\tmetric\tvalue\tage_s\tstatus")
    bad = 0
    for host in args.hosts:
        for metric in args.metrics:
            try:
                data = q(metric, host, t_from, now, args.site, api_key, app_key)
                val, age = latest_value_and_age(data, now)
                if val is None:
                    status = "missing"
                    bad += 1
                elif age is not None and age > args.window_sec:
                    status = "stale"
                    bad += 1
                else:
                    status = "ok"
                vtxt = "-" if val is None else f"{val:.4f}"
                atxt = "-" if age is None else str(age)
                print(f"{host}\t{metric}\t{vtxt}\t{atxt}\t{status}")
            except Exception as exc:
                bad += 1
                print(f"{host}\t{metric}\t-\t-\terror:{type(exc).__name__}")

    return 1 if bad else 0


if __name__ == "__main__":
    raise SystemExit(main())
