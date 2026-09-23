# RFC 0001 — One home per primitive

Status: Accepted  
Date: 2026-09-23

## Problem

Agent-driven projects repeatedly reimplement scheduling, persistence, telemetry, routing, and provenance because each repository sees only its local needs.

## Decision

Each primitive has one canonical owner. Sibling projects integrate with or project from that owner rather than creating a second canonical implementation.

Current ownership is defined in `docs/PROJECTS.md` and each repo's `.project.toml`.

## Consequences

- Cross-repo links are preferred over copied documentation/code.
- New architectural code must check sibling ownership first.
- Experimental duplicate implementations must identify themselves explicitly as controls/benchmarks.
- Derived projections may be duplicated; canonical state may not.

## Agent rule

Read `AGENTS.md`, `.project.toml`, and the registry before architectural changes.
