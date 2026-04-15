# TRIZ 39 Parameters

Use these to map your contradiction to the matrix.
Pick the closest match — exact wording is less important than conceptual fit.

## Performance & dynamics

| # | Parameter | Software analogy |
|---|-----------|-----------------|
| 9 | Speed | Throughput, latency, response time |
| 10 | Force | Load, request pressure, compute intensity |
| 11 | Stress / pressure | Queue depth, backpressure, resource contention |
| 15 | Duration of action (moving) | Request lifetime, session length, job duration |
| 16 | Duration of action (stationary) | Lock hold time, batch window, cache TTL |
| 21 | Power | CPU/GPU utilization, energy budget |
| 39 | Productivity | Throughput per unit cost; jobs completed per hour |

## Reliability & quality

| # | Parameter | Software analogy |
|---|-----------|-----------------|
| 13 | Stability of composition | System invariants, schema stability, backward compat |
| 14 | Strength | Fault tolerance, resilience, error resistance |
| 27 | Reliability | Uptime, MTBF, correctness under failure |
| 28 | Measurement accuracy | Observability precision, metric fidelity |
| 29 | Manufacturing precision | Deployment repeatability, test determinism |
| 30 | Object-generated harmful effects | Side effects, coupling, blast radius |
| 31 | Harmful side effects | Cascading failures, unintended state mutation |

## Structure & complexity

| # | Parameter | Software analogy |
|---|-----------|-----------------|
| 1 | Weight of moving object | Payload size, request body, message size |
| 2 | Weight of stationary object | Binary size, container image, dependency weight |
| 3 | Length/area of moving object | Code path length, call depth |
| 4 | Length/area of stationary object | Schema width, API surface area |
| 7 | Volume of moving object | Working set size, in-flight data volume |
| 8 | Volume of stationary object | Storage footprint, index size |
| 12 | Shape | Data structure, API shape, interface design |
| 36 | Device complexity | System complexity, number of moving parts |
| 37 | Difficulty of detecting/measuring | Observability gap, debugging difficulty |

## Resources & loss

| # | Parameter | Software analogy |
|---|-----------|-----------------|
| 19 | Use of energy (moving) | Per-request compute cost |
| 20 | Use of energy (stationary) | Idle resource cost, standby cost |
| 22 | Loss of energy | Wasted compute, unnecessary work |
| 23 | Loss of substance | Data loss, dropped messages, eviction |
| 24 | Loss of information | Log gaps, audit holes, observability loss |
| 25 | Loss of time | Latency overhead, wait time, blocking |
| 26 | Quantity of substance | Data volume, record count, corpus size |

## Operability

| # | Parameter | Software analogy |
|---|-----------|-----------------|
| 17 | Temperature | Load level, thermal throttling, hot path |
| 18 | Illumination | Visibility, logging level, dashboards |
| 32 | Ease of manufacture | Developer experience, build complexity |
| 33 | Ease of operation | Ops burden, runbook complexity, toil |
| 34 | Ease of repair | Debuggability, rollback ease, MTTR |
| 35 | Adaptability / versatility | Configurability, multi-tenancy, extensibility |
| 38 | Extent of automation | CI/CD coverage, auto-remediation level |

## Geometry (less common in software)

| # | Parameter | Software analogy |
|---|-----------|-----------------|
| 5 | Area of moving object | Memory footprint per request |
| 6 | Area of stationary object | Disk footprint, persistent memory use |
