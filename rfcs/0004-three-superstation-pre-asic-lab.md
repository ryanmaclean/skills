# RFC 0004 — Three-SuperStation pre-ASIC lab

Status: Accepted experimental plan  
Date: 2026-09-23

## Context

There are three SuperStation One systems available. Each provides a Cyclone V SoC FPGA platform suitable for validating the durable transaction primitive before spending money on ASIC tapeout.

## Decision

Use the three systems as a differential hardware lab.

### Roles

- **SS1-A — control**
  - software durable-tid reference implementation
  - no FPGA acceleration

- **SS1-B — DUT**
  - RTL TID allocator
  - commit FSM
  - BRAM-backed descriptor/completion queues
  - later: persistent-media integration

- **SS1-C — torture/fault injection**
  - reset/power interruption
  - duplicate/replay submission
  - corrupted-record tests
  - timing/backpressure abuse

## Research sequence

1. software append/commit/recover baseline
2. RTL simulation and formal properties
3. FPGA BRAM sequencer
4. differential testing against software
5. persistent-media acknowledgement
6. optional NVMe durability semantics
7. multi-node ordering/causality experiments
8. ASIC only if FPGA results justify it

## Required invariants

- committed TIDs never decrease
- committed TIDs never repeat
- completion implies durable commit
- allocated-but-uncommitted TIDs do not survive recovery
- duplicate submission does not create duplicate commits
- recovery converges with the software control implementation

## Cost rule

Do not spend on ASIC before the FPGA implementation demonstrates a measurable advantage in latency, CPU cost, determinism, or power.

## Architecture rule

The FPGA/ASIC owns only ordering/commit mechanics.

It does **not** own:
- BOP cards/runs
- filesystem namespace
- MCP/HTTP
- OpenLineage
- Datadog
- model execution

Those remain projections or higher-level consumers.
