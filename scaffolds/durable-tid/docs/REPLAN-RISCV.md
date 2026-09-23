# RISC-V hardware replan

## Principle

The durable-tid block is below the CPU.

RISC-V is a programmable control plane around the primitive, not the primitive itself.

## Phase 0 — semantic proof

No CPU.

- TLA+/PlusCal
- JavaScript executable model
- crash/fault model
- differential traces

## Phase 1 — pure RTL on SuperStation One

No RISC-V yet.

```
MMIO test host
    |
durable-tid RTL
    |
BRAM
```

Prove:
- ordering
- allocation vs commit separation
- crash/reset behaviour
- idempotent request identity
- completion-after-durability invariant

## Phase 2 — add a tiny open RISC-V control plane

Preferred candidate: lowRISC Ibex (Apache-2.0).

```
Ibex RV32
  |
 MMIO
  |
durable-tid RTL
```

RISC-V responsibilities:
- configure
- inspect
- recovery policy
- diagnostics
- slow/error paths

RTL responsibilities:
- sequencing
- commit state
- completion visibility
- hot-path integrity

Do not put the hot path in firmware.

## Phase 3 — reuse OpenTitan IP

Evaluate, rather than rewrite:
- DMA
- SHA/HMAC
- FIFO/SRAM primitives
- integrity/error-detection blocks

Only import blocks whose licensing remains compatible with the MIT/BSD/Apache policy.

## Phase 4 — persistent media

```
Ibex control
     |
durable-tid
     |
DMA / storage adapter
     |
media
```

The durable acknowledgement boundary must be explicit.

## Phase 5 — instruction/coprocessor experiment

Only after MMIO measurements.

Question:
Does a custom RISC-V instruction materially improve latency/CPU cost over MMIO?

Reference:
- CORE-V CV-X-IF
- PicoRV32 PCPI as a simple historical model

Possible semantic instruction:

```
tid.commit descriptor -> tid/status
```

Do not create a custom ISA merely because RISC-V allows it.

## Phase 6 — capability-aware DMA research

Evaluate CHERI-RISC-V / CHERIoT-style capability protection for DMA buffers and descriptor access.

This is a security follow-on, not a prerequisite.

## Phase 7 — ASIC gate

Tapeout only if the fixed-function block proves a measurable benefit over:
- software reference
- dedicated CPU core
- FPGA implementation

RISC-V remains optional on silicon:
- omit it if the state machine is sufficient
- include a small core only if recovery/configuration flexibility earns the area/power cost
