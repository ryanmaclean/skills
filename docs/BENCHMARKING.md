# Shared benchmark and projection contract

Schema: `schemas/bench.v1.schema.json`  
Example: `examples/bench.v1.example.json`

## Principle

Measure once, project many times.

A benchmark/run fact should be emitted once as `ryanlab.bench.v1`, then translated into:

- Datadog metrics/events/notebooks
- GitHub Actions summaries
- experiment keep/discard decisions
- OpenLineage facets where appropriate

Do not make any projection canonical state.

## Producers

Initial producers:

- smolFire runtime/filesystem benchmarks
- BOP common state-machine workload
- agent-jail isolation comparisons
- Moth worker footprint measurements
- autoresearch experiment loops
- port-eval arms (`nu scripts/port-eval.nu bench <spec> <result>`; workload `port-eval:<spec-id>@<ref>`, see `port-eval/SKILL.md`)

## Core dimensions

Use stable strings:

- `project`
- `runtime`
- `filesystem`
- `workload`
- git `commit`
- immutable artifact hash where available

## Core metrics

The schema includes:

- artifact bytes
- RSS bytes
- boot-to-ready milliseconds
- rename p50
- fsync p50
- recovery milliseconds
- write amplification
- metadata bytes/run
- retained history bytes/GiB
- lineage reconstruction milliseconds

Projects may add metrics without changing the v1 schema because `metrics` permits extension.

## Datadog projection

Recommended metric names:

- `ryanlab.artifact.bytes`
- `ryanlab.runtime.rss_bytes`
- `ryanlab.runtime.boot_ms`
- `ryanlab.fs.rename_us`
- `ryanlab.fs.fsync_us`
- `ryanlab.fs.recovery_ms`
- `ryanlab.fs.write_amplification`
- `ryanlab.fs.metadata_bytes_per_run`
- `ryanlab.fs.history_bytes_per_gib`
- `ryanlab.lineage.reconstruction_ms`

Recommended tags:

- `project:<name>`
- `runtime:<name>`
- `filesystem:<name>`
- `workload:<name>`
- `commit:<short-sha>`

Do not put high-cardinality run UUIDs into metric tags. Run UUIDs belong in events/logs/traces or notebook-linked artifacts.

## OpenLineage projection

Only project fields that represent lineage facts. Performance measurements are not lineage state.

Possible custom facets may include runtime, filesystem version identity, artifact SHA256, and benchmark record location.

## Experiment loops

Autoresearch-style loops should preserve the complete raw benchmark record even when they compute a single score for keep/discard decisions.

Never discard raw measurements merely because the experiment lost.
