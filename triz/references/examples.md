# TRIZ Software Contradiction Examples

Use when the parameters don't obviously match your problem.
Find the closest analogy and run the oracle with those parameter numbers.

## Speed vs Reliability
*"Faster deploys break more often"*

- Improving: #9 Speed → Worsening: #27 Reliability
- Matrix recommends: #10 Preliminary action, #11 Cushion in advance, #27 Cheap short-lived
- Applied: Blue-green deploys (cushion). Feature flags (partial action). Canary at 1%.

## Isolation vs Shared State
*"Agents need isolated workspaces AND shared schema"*

- Improving: #13 Stability of composition → Worsening: #39 Productivity
- Matrix recommends: #1 Segmentation, #5 Combining, #6 Universality
- Applied: Isolate the runtime (worktree per agent). Share the format spec (one schema). The spec is the coordination point, not the filesystem location.

## Observability vs Performance
*"More logging slows the hot path"*

- Improving: #18 Illumination → Worsening: #9 Speed
- Matrix recommends: #19 Periodic action, #24 Intermediary, #35 Parameter changes
- Applied: Async log drain (intermediary). Sampling at 1% (partial action). Dynamic log level toggle (parameter changes).

## Correctness vs Throughput
*"Strong consistency limits write throughput"*

- Improving: #27 Reliability → Worsening: #39 Productivity
- Matrix recommends: #35 Parameter changes, #10 Preliminary action, #13 Inversion
- Applied: Tunable consistency per operation (parameter changes). Write-ahead log (preliminary action). Event sourcing with read-model reconciliation (inversion).

## Autonomy vs Safety
*"Agents must act freely AND humans must approve"*

- Improving: #38 Extent of automation → Worsening: #27 Reliability
- Matrix recommends: #23 Feedback, #24 Intermediary, #25 Self-service
- Applied: `decision_required: true` gate on the card (intermediary). Agent approves its own low-risk changes; escalates high-risk (self-service + feedback).

## Simplicity vs Completeness
*"CLI must be minimal AND support all workflows"*

- Improving: #33 Ease of operation → Worsening: #35 Adaptability
- Matrix recommends: #2 Extraction, #6 Universality, #1 Segmentation
- Applied: The FORMAT is the product (universality). The CLI is one thin reference implementation. Remove any command not required to bootstrap or observe — agents participate with filesystem writes alone.

## Test Coverage vs Deploy Speed
*"Full test suite on every commit is too slow"*

- Improving: #27 Reliability → Worsening: #9 Speed (or #25 Loss of time)
- Matrix recommends: #1 Segmentation, #10 Preliminary action, #15 Dynamics
- Applied: Dependency graph between tests and modules (segmentation). Run only affected tests (dynamics/adaptive). Spec-time acceptance criteria eliminate suite growth (preliminary action).

## Multi-Tenancy vs Data Isolation
*"Shared infra reduces cost but tenants must not see each other"*

- Improving: #39 Productivity → Worsening: #13 Stability of composition
- Matrix recommends: #1 Segmentation, #17 Transition to another dimension, #35 Parameter changes
- Applied: Row-level security (segmentation in data dimension). Separate namespaces/schemas (transition to another dimension). Per-tenant config (parameter changes).

---

## Reframing guide

When the matrix returns no recommendations (empty cell):

1. **Generalize the parameter** — "deploy speed" → #9 Speed or #25 Loss of time; try both
2. **Flip the contradiction** — improving A worsening B often has more matrix coverage than improving B worsening A
3. **Find the physical analogy** — "memory pressure" → #11 Stress/pressure or #21 Power
4. **Ask `triz whys`** — drill from the symptom to find the real parameter pair
