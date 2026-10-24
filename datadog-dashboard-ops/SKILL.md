---
name: datadog-dashboard-ops
description: "Create, update, and validate Datadog dashboards (especially unified ops dashboards), fix now-vs-prev tiles, add Vega/Vega-Lite visuals, and verify metrics/logs/APM. Use when asked to edit Datadog dashboards, repair empty widgets, add Sankey/Vega visuals, or keep dashboards in sync with live metrics and host tags."
---

# Datadog Dashboard Ops

## Overview
Use this skill to fetch/patch/push Datadog dashboards, repair empty widgets, and validate live metrics/queries against real host data.

## Workflow (dashboard edits)

1) **Pull the dashboard JSON**
```
./scripts/dd_get_dashboard.sh <dashboard_id> /tmp/dashboard.json
```

2) **Fix “now vs prev” tiles using real metrics**
```
./scripts/fix_now_vs_prev.py /tmp/dashboard.json ./references/now-vs-prev-map.txt
```

3) **Add Vega/Vega-Lite widgets (Sankey, flow, fallback)**
- Use `wildcard` widget with `specification` block. See `references/vega-wildcard.md`.

4) **Push the updated dashboard**
```
./scripts/dd_put_dashboard.sh <dashboard_id> /tmp/dashboard.json
```

5) **Verify live data**
- List metrics available in the last 24h:
```
./scripts/dd_list_metrics.sh
```
- Validate a metric quickly (use datadog-operations `query-metrics.sh` if available).

## Guardrails
- Prefer **real host-emitted metrics** over synthetic scripts.
- If a widget is empty, confirm the metric exists in the last 24h before changing queries.
- Avoid `.as_rate()` on gauges; only use for true counters.
- Always include `ai_agent.*` + non‑ai_agent metrics when both exist.

## Resources
- `scripts/dd_get_dashboard.sh`: fetch dashboard JSON
- `scripts/dd_put_dashboard.sh`: push dashboard JSON
- `scripts/dd_list_metrics.sh`: list recent metrics
- `scripts/fix_now_vs_prev.py`: repair now-vs-prev tiles using a mapping
- `references/now-vs-prev-map.txt`: canonical title → metric mapping
- `references/vega-wildcard.md`: wildcard Vega/Vega-Lite schema
