# Hardware path

The semantic core remains:

```
append(record)
commit() -> tid
recover() -> last_tid
read(tid)
```

Hardware is an implementation, not the semantic definition.

## Cheapest path

Use three SuperStation One systems before any ASIC work.

### A — software oracle
Reference implementation.

### B — FPGA implementation
Minimal RTL:
- ingress descriptor registers/FIFO
- TID counter
- commit FSM
- BRAM-backed state
- completion FIFO
- CRC/integrity

### C — fault injection
Resets, replay, malformed requests, interrupted commits, backpressure.

## v0 external contract

Request fields:

```
run_id
object_id
op
length
submit
```

Response fields:

```
status
tid
complete
```

## v0 exclusions

Do not add yet:
- SHA-256
- PCIe
- NVMe
- VirtIO
- general ring API
- soft CPU
- filesystem
- network stack

## Formal properties

Prove before hardware deployment:

- TID monotonicity
- no committed TID reuse
- completion implies committed state
- reset cannot promote uncommitted state
- duplicate request identity cannot yield duplicate commits

## Next stages

1. BRAM only
2. persistent-media acknowledgement
3. shared-memory/ring transport
4. optional DMA
5. optional NVMe durability semantics
6. multi-node ordering
7. ASIC only after measured FPGA benefit

See:
- `../../rfcs/0004-three-superstation-pre-asic-lab.md`
- smolFire `docs/SUPERSTATION-PRE-ASIC-PLAN.md`
