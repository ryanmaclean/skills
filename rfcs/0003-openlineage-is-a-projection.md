# RFC 0003 — OpenLineage is a projection, not a canonical database

Status: Accepted design constraint  
Date: 2026-09-23

## Decision

OpenLineage events are generated from BOP run identity plus filesystem-native object/version facts.

Suggested mapping:

- card/template → Job
- execution UUID → Run
- parent relationship → ParentRunFacet
- dependencies → JobDependenciesRunFacet
- transition to running → START
- done → COMPLETE
- failed → FAIL
- object/path → Dataset
- HAMMER TID / LFS checkpoint / HAMMER2 or FFS snapshot identity → DatasetVersion

## Constraint

Do not add a second canonical lineage state store unless a concrete requirement cannot be derived from existing facts.
