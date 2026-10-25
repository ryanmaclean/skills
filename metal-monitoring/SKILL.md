---
name: "metal-monitoring"
description: "Validate Apple Metal GPU telemetry in Datadog and verify DCGM shim ingestion across workstations. Use when GPU metrics look missing, stale, or inconsistent on Apple Silicon training hosts, or when checking the DCGM-shim pipeline."
---

# Metal Monitoring

## Overview
Use this skill when you need to confirm that Apple Silicon training hosts are emitting GPU-related metrics to Datadog and dashboards are lighting up with DCGM-style series.

## When To Use
- User asks if GPU metrics are live, missing, stale, or inconsistent.
- DCGM dashboard widgets are empty on Apple hosts.
- You need to verify host-by-host ingestion before long training runs.

## Workflow
1. Load Datadog auth from local env:
```bash
source ~/.zshrc >/dev/null 2>&1 || true
set -a; source ~/.tundra-otel.env >/dev/null 2>&1 || true; set +a
```
2. Run the metric liveness check:
```bash
python3 ~/.codex/skills/metal-monitoring/scripts/check_metal_metrics.py \
  --hosts mbp-m1.local studio.local minim4-24.local
```
3. If a host is stale, run one-shot collector on that host:
```bash
bash ~/train/autoresearch-mlx/scripts/run_metal_sampler_once.sh
```
4. If one-shot fails with `403`, fix key/site precedence:
- Ensure valid `export DD_API_KEY=...`, `DD_APP_KEY`, `DD_SITE` in `~/.zshrc`.
- Keep `.tundra-otel.env` values exported (or use `set -a` while sourcing).

## Expected Metrics
Primary checks:
- `dcgm.power_usage`
- `dcgm.gpu_utilization`
- `dcgm.temperature`

Common Apple shim metrics:
- `dcgm.fb_used`, `dcgm.fb_free`
- `dcgm.sm_clock`, `dcgm.mem.clock`
- `autoresearch.metal.gpu_power_watts`

See detailed mapping in [references/metric-mapping.md](references/metric-mapping.md).

## Output Criteria
Treat host as healthy when all are true:
- At least one non-null point for the three primary `dcgm.*` metrics.
- Latest point age is within the query window.
- Hostname tags align with intended host identity (`*.local`).

## Resources
- Script: [scripts/check_metal_metrics.py](scripts/check_metal_metrics.py)
- Reference: [references/metric-mapping.md](references/metric-mapping.md)
