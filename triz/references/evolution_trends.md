# TRIZ Evolution Trends

Load this when: predicting where a system is heading, identifying which contradictions are transient vs structural, or explaining why a contradiction keeps re-appearing.

Systems evolve along these 8 trends regardless of implementation choices. Fighting a trend accumulates contradictions; moving with it resolves them.

## The 8 trends (quick reference)

| # | Trend | What to look for |
|---|-------|-----------------|
| 1 | S-curve stages | Where is the system in infancy/growth/maturity/decline? |
| 2 | Increasing ideality | What would the IFR (ideal final result) look like — no system at all? |
| 3 | Non-uniform subsystem development | Which subsystem is the least evolved? That's where the next contradiction appears. |
| 4 | Increasing dynamism | rigid → adjustable → dynamic → adaptive. Which stage are you at? |
| 5 | Complexity then simplification | Is the system at peak complexity? Consolidation is next. |
| 6 | Matching/mismatching rhythms | Is the contradiction about timing mismatch? Synchronize or decouple. |
| 7 | Super-system and micro-level | Can the contradiction be solved one level up or one level down? |
| 8 | Decreasing human involvement | Where is a human the bottleneck? That's the next automation target. |

## Software mappings

### Trend 1 — S-curve
- Monolith → services: infancy-to-growth contradiction
- SQL at scale → NoSQL: relational maturing, document in growth
- Batch → streaming: batch mature, streaming in growth

### Trend 2 — Ideality
- Database → managed cloud DB → serverless: progressively less infrastructure
- Manual CI → declarative CI → AI-generated CI: operator disappears
- Explicit config → convention → zero-config: config burden approaches zero

### Trend 3 — Non-uniform development
- ML inference speed outpaces feature pipeline → next bottleneck is feature engineering
- Frontend tooling (Bun, Vite) evolves faster than backend pipelines
- LLM capability outpaces prompt safety tooling → alignment gap

### Trend 4 — Dynamism
- Hard-coded → env vars → feature flags → ML-driven parameter tuning
- Fixed deploy → blue/green → canary → progressive delivery with auto-rollback
- Static typing → generics → macros → runtime adaptation

### Trend 5 — Complexity then simplification
- Microservices explosion → service mesh → platform engineering → unified platform
- Plugin ecosystem → framework consolidation (Rails, Next.js)
- Multiple CI systems → unified GitOps

### Trend 6 — Rhythms
- Request/response → async messaging when caller and callee rhythms differ
- Polling → webhooks when push rhythm matches subscriber capacity
- Sync deployment → async progressive rollout when release rhythm exceeds absorb rhythm

### Trend 7 — Level shift
- Upward: individual services → platform → cloud-native ecosystem
- Downward: OS processes → containers → WASM components → functions
- Both: microservices (micro-level) on cloud platform (super-system)

### Trend 8 — Human involvement
- Manual deploy → CI/CD → GitOps → autonomous continuous deployment
- Manual code review → linting → static analysis → LLM review → autonomous merge gate
- Manual incidents → runbooks → automated playbooks → self-healing

## Using trends with the matrix

When a contradiction keeps recurring: check trend 3 (non-uniform development) — one subsystem is holding back the rest. The matrix gives you the *how*; the trend tells you *why* the contradiction will return if you don't address the root.
