# TRIZ Innovation Levels

Load this when: calibrating how hard a contradiction is, or setting expectations for what the matrix lookup will return.

## The 5 levels (Altshuller)

| Level | Name | Frequency | Search radius | TRIZ tool |
|-------|------|-----------|---------------|-----------|
| 1 | Routine design | 32% | Within your specialty | No matrix needed — select from known solutions |
| 2 | Minor improvement | 45% | Across the industry | Matrix — top 1-2 principles, compromise acceptable |
| 3 | Major improvement | 18% | Adjacent industries / sciences | Matrix + physical contradictions + sub-principles |
| 4 | New concept | 4% | All fields of science | ARIZ, Su-Field, physical contradiction separation |
| 5 | Discovery | 1% | New scientific phenomenon | No tool — frame the IFR and search literature |

## Identifying the level before you run solve

Ask: "How many domains would I need to search to find a solution?"

- One domain, known solutions → Level 1
- Same industry, seen before → Level 2
- Never solved this way in our industry → Level 3
- Physically contradictory requirement → Level 4
- Would require a new law of physics → Level 5

## Software examples by level

**Level 1** — add a cache; use pagination; add an index
**Level 2** — introduce a message queue to decouple producer/consumer rates; use a CDN
**Level 3** — event sourcing (from accounting); CRDTs (from lattice math); FRP (from signal processing)
**Level 4** — homomorphic encryption; differential privacy; Byzantine fault tolerance
**Level 5** — public key cryptography; neural networks

## Implication for triz solve

- **Level 1-2**: single `triz solve` call will give you a direct answer
- **Level 3**: run `triz council --n 3` for multiple framings; review sub-principles in principles.md
- **Level 4**: check physical_contradictions.md first; matrix may give a direction but not a solution
- **Level 4-5**: use the matrix output as a framing device, not an answer

## The uncomfortable truth

77% of contradictions are Levels 1-2 and need no TRIZ at all — just domain knowledge.
TRIZ matrix is most useful for the 18% that are Level 3.
If the team thinks they're at Level 4+, first verify it isn't actually a Level 3 framed poorly.
