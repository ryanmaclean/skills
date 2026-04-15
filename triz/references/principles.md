# TRIZ 40 Principles

Software-mapped one-liners. Load this when you need to understand or explain a specific principle.

| # | Principle | Software application |
|---|-----------|---------------------|
| 1 | Segmentation | Split into independent parts: microservices, namespaces, feature flags, worktrees |
| 2 | Extraction | Remove the interfering part: pure functions, config-as-code, side-effect isolation |
| 3 | Local quality | Non-uniform structure: tiered storage, adaptive compression, hot/cold data paths |
| 4 | Asymmetry | Introduce asymmetry: write-optimized vs read-optimized paths, asymmetric encryption |
| 5 | Combining | Merge parallel operations: batch APIs, request coalescing, co-location |
| 6 | Universality | One thing serves many: shared schema, protocol buffers, content negotiation |
| 7 | Nesting | Object inside object: embedded caches, inline metadata, nested transactions |
| 8 | Counterweight | Balance with opposing force: backpressure, circuit breaker, rate limiting |
| 9 | Preliminary anti-action | Pre-cancel known stress: chaos engineering, pre-validation, schema migration before deploy |
| 10 | Preliminary action | Do in advance: pre-warming, pre-computation, speculative execution, eager loading |
| 11 | Cushion in advance | Prepare for failure: retry budgets, fallback defaults, graceful degradation |
| 12 | Equipotentiality | Remove need to change conditions: idempotent APIs, event sourcing, immutable infra |
| 13 | Inversion | Flip the approach: pull not push, reactive not polling, consumer-driven contracts |
| 14 | Spheroidality | Use curves/cycles: ring buffers, rolling windows, circular queues |
| 15 | Dynamics | Make adaptive: adaptive timeouts, auto-scaling, dynamic configuration |
| 16 | Partial/excessive action | Slightly more or less: over-provision then trim, canary at 1% |
| 17 | Transition to another dimension | Move to different dimension: async to sync, 2D to 3D data model, time-series instead of snapshot |
| 18 | Mechanical vibration | Periodic oscillation: heartbeats, polling with jitter, pulsed sampling |
| 19 | Periodic action | Replace continuous with periodic: batch instead of stream, scheduled instead of realtime |
| 20 | Continuity of useful action | Eliminate idle: streaming pipelines, work stealing, continuous integration |
| 21 | Skipping | Rush through at high speed: short-circuit evaluation, fast path for common case |
| 22 | Blessing in disguise | Use harmful factor usefully: use failure signal as health signal; use backpressure as load indicator |
| 23 | Feedback | Introduce feedback loop: observability gates, acceptance criteria in spec, A/B metrics |
| 24 | Intermediary | Add intermediate object: message queue, API gateway, event bus, proxy |
| 25 | Self-service | Object serves/fixes itself: self-healing systems, auto-remediation, self-describing formats |
| 26 | Copying | Use simpler copy: read replicas, shadow mode, feature parity via mock |
| 27 | Cheap short-lived objects | Replace expensive with disposable: ephemeral containers, serverless, immutable deploys |
| 28 | Mechanics substitution | Replace physical with field: async messaging instead of synchronous call; events instead of RPC |
| 29 | Pneumatics/hydraulics | Use gas/liquid: streaming instead of batch; flow-based data processing |
| 30 | Flexible shells | Use flexible/thin elements: thin clients, adapter layers, shim interfaces |
| 31 | Porous materials | Make permeable: pluggable architectures, open extension points, composable middleware |
| 32 | Color changes | Change properties: dark mode, feature toggles, dynamic schema evolution |
| 33 | Homogeneity | Same material: monorepo, shared libraries, uniform serialization |
| 34 | Discarding/recovering | Discard then restore: stateless services, checkpoint-restart, tombstone records |
| 35 | Parameter changes | Change concentration/flexibility: feature flags, runtime config, A/B rollout |
| 36 | Phase transitions | Use transition effects: cache warming phase, migration phase, shadow write phase |
| 37 | Thermal expansion | Use thermal-like expansion: elastic scaling, burst capacity, demand-driven provisioning |
| 38 | Strong oxidants | Use enriched interaction: privileged fast path, priority queues, preemptive scheduling |
| 39 | Inert atmosphere | Replace active with inert: read-only replicas, immutable state, append-only corpus |
| 40 | Composite materials | Combine dissimilar materials: tiered consistency (strong + eventual), hybrid storage |
