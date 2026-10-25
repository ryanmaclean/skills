# Apple Metal -> Datadog Metric Mapping

Primary dashboard-compatible metrics emitted by the Apple shim:

- `dcgm.power_usage` <- `autoresearch.metal.gpu_power_watts`
- `dcgm.gpu_utilization` <- `autoresearch.metal.gpu_active_pct`
- `dcgm.temperature` <- powermetrics die temp or fallback estimate
- `dcgm.fb_used`, `dcgm.fb_free` <- unified memory approximation
- `dcgm.sm_clock`, `dcgm.mem.clock` <- active GPU MHz (proxy)

Additional raw Apple metrics:

- `autoresearch.metal.gpu_power_watts`
- `autoresearch.metal.gpu_active_pct`
- `autoresearch.metal.gpu_active_mhz`
- `autoresearch.metal.gpu_temperature_c`
- `autoresearch.metal.gpu_temperature_est_c`

Notes:

- `dcgm.vgpu_license_status` should remain host-level semantic for Apple.
- Do not repurpose license metrics for process fanout values.
- Host tags should be canonical (`*.local`) for multi-host dashboards.
