# PDSA Metrics Reference

Use one primary metric per cycle.

## Build/Infra examples

- Build success rate: `successful_runs / total_runs` over 2h window
- Time to first boot: minutes from build start to QEMU boot marker
- Artifact size: bytes for kernel, base archive, boot ISO
- Lane availability: reachable lanes / expected lanes
- Inference heartbeat freshness: minutes since last successful API call

## Guardrail examples

- Disk free must stay above threshold (e.g., 15%)
- No kernel panic in smoke logs
- No increase in critical error signature count
- No service outage on publish endpoints

## Decision hints

- ADOPT: primary metric hits target and all guardrails pass
- ADAPT: metric improves but target not met, guardrails pass
- ABANDON: metric worsens or guardrails fail
