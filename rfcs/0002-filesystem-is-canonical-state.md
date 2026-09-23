# RFC 0002 — Filesystem-native durable state is canonical where possible

Status: Experimental/target architecture  
Date: 2026-09-23

## Problem

Agent systems often duplicate one transition into filesystem state, queue state, database rows, event logs, VCS metadata, and lineage records.

## Decision

For BOP-style work, durable card transitions remain filesystem-native. Where the filesystem exposes transaction/checkpoint/snapshot identity, that identity should be reused for version/history rather than recreated in another database.

Candidate semantics under test:

- HAMMER1 transaction IDs/history
- HAMMER2 snapshot/PFS/version primitives
- NetBSD LFS checkpoints/history retention
- NetBSD FFS/WAPBL + fss snapshots as a baseline

## Boundaries

BOP owns run/card identity and transition intent.  
The filesystem owns persistence/history/version semantics.  
Moth executes work.  
OpenLineage and Datadog consume derived facts.

## Non-goal

This RFC does not assume HAMMER2, LFS, or any candidate wins before measurement.
