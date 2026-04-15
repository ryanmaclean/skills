# TRIZ Physical Contradictions

Load this when: the contradiction test gives "a parameter must be **both** X **and** not-X at the same time" — e.g. "the service must be available AND down for maintenance."

This is a different contradiction type from the 39×39 matrix (technical contradictions). Use separation strategies instead.

## The test

- **Technical contradiction**: improving A worsens B → use the matrix
- **Physical contradiction**: A must be both X and not-X → use separation strategies below

## Four separation strategies

| # | Strategy | When to use | Key principles |
|---|----------|-------------|----------------|
| 1 | **Separation in time** | The conflicting requirements are needed at different times | 19, 15, 10, 21 |
| 2 | **Separation in space** | The conflicting requirements are needed in different locations | 1, 17, 3, 13 |
| 3 | **Separation between whole and parts** | The whole needs one property; each part needs the opposite | 1, 5, 12, 40 |
| 4 | **Separation upon condition** | Requirements differ depending on state or context | 15, 23, 11, 35 |

## Software examples by strategy

### 1 — Time
- Maintenance window: service available (normally) AND down (scheduled window)
- Write-behind cache: consistent (at flush) AND fast-write (immediately)
- Blue/green deploy: old live AND new being validated — at different times
- Feature flag: enabled (canary) AND disabled (general) — separated by rollout

### 2 — Space
- Read replica: local fast AND globally consistent — different regions
- Edge cache: fast (at edge) AND fresh (at origin)
- Multi-tenancy: shared infrastructure AND isolated per-tenant namespace
- Monorepo: unified code AND independently deployed services

### 3 — Whole vs parts
- Microservices: each service is simple AND system has complex capability
- Stateless services + stateful storage: service is stateless AND system retains state
- Immutable infra + mutable data: servers immutable AND data persists
- Council vote: individual agents disagree AND council returns single decision

### 4 — Condition
- Circuit breaker: API available (normal) AND fast-failing (degraded) — depends on state
- `--auto-tier`: model is fast (simple) AND powerful (complex) — depends on input
- Optimistic locking: lockless (no conflict) AND serialized (conflict detected)
- Semantic search fallback: strict match (high confidence) AND fuzzy (low confidence)

## Resolution path

1. Identify the physical contradiction ("must be X AND not-X")
2. Ask: *when* do the requirements conflict? *where*? *at what level*? *under what condition*?
3. Pick the matching strategy
4. Use the linked principles for implementation patterns
5. If no separation strategy fits, reframe as a technical contradiction and use the matrix
