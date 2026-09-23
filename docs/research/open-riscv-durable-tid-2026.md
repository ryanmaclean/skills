# Open RISC-V prior art for durable-tid

Updated: 2026-09-23

## Research question

What existing open or proprietary hardware already combines pieces of:

- asynchronous submission/completion
- storage datapath offload
- durable ordering
- DMA
- hashing/integrity
- small programmable control-plane cores

and what should durable-tid reuse instead of reimplement?

## Strongest open-source candidates

### lowRISC Ibex — preferred CPU control-plane candidate

Repository: https://github.com/lowRISC/ibex  
License: Apache-2.0  
Status: active

Why it fits:
- small RV32 core
- SystemVerilog
- explicit FPGA and ASIC register-file variants
- extensive verification culture
- RVFI/formal hooks already present
- avoids building a CPU

Use:
- control-plane CPU only after the RTL durable-tid core is proven
- MMIO/control, recovery policy, diagnostics
- not the commit datapath itself

### OpenTitan IP — preferred reusable hardware-IP library

Repository: https://github.com/lowRISC/opentitan  
License: Apache-2.0 unless otherwise noted

Relevant blocks:
- DMA controller
- SHA/HMAC engines
- SRAM/FIFO primitives
- integrity/error-detection primitives
- Ibex integration
- security/verification infrastructure

Use:
- evaluate DMA/hash primitives before writing our own
- preserve durable-tid as a separate semantic block
- copy patterns, not the entire OpenTitan SoC

### LiteX — useful SoC/integration harness

Repository: https://github.com/enjoy-digital/litex  
License: BSD

Why it fits:
- quick FPGA SoC assembly/simulation
- RISC-V integration
- board bring-up and bus plumbing

Use:
- possible FPGA laboratory harness
- not canonical hardware architecture

### PicoRV32 — frozen minimal reference

Repository: https://github.com/YosysHQ/picorv32  
License: ISC  
Status: archived 2026-09-08

Why it matters:
- extremely small
- simple co-processor interface
- permissive license
- formally verifiable

Use:
- size/control reference only
- do not make an archived project the strategic CPU dependency

### CVA6 / CORE-V / CV-X-IF — custom-extension reference

Repositories:
- https://github.com/openhwgroup/cva6
- https://github.com/openhwgroup/core-v-xif

CV-X-IF is specifically designed to connect custom coprocessors without modifying the CPU.

License note:
- files use Solderpad/Apache-compatible licensing patterns
- verify exact component licensing before importing because project policy prefers MIT/BSD/Apache only

Use:
- reference architecture for later custom instruction/coprocessor experiments
- MMIO should come first

### FROST

Repository: https://github.com/twosigma/frost  
License: Apache-2.0

Why not first:
- RV64 out-of-order Linux-capable CPU
- useful high-performance open-core reference
- far larger than the control-plane CPU durable-tid needs

### VexRiscv / VexiiRiscv

Core license: MIT.

Caution:
- generator depends on SpinalHDL core, which is LGPL-3.0
- generated RTL can be separately owned/permissive, but this conflicts with the project's preference to avoid LGPL dependencies in the development stack

Conclusion:
- do not choose as the default path

## Capability-security track

### CHERI-RISC-V

The active RISC-V CHERI work explicitly discusses capability-safe DMA and peripheral access.

Potential future use:
- constrain DMA descriptors and storage buffers by capability
- reduce IOMMU/page-table dependence in a tightly controlled SoC
- strong match for a tiny, auditable persistence device

Not v0:
- first prove ordering/durability
- add memory-safety/capability semantics only after the primitive is stable

## Proprietary/commercial prior art

### NVIDIA BlueField SNAP / DPA

BlueField SNAP accelerates NVMe/virtio-blk storage paths.

Particularly relevant:
- BlueField DPA is a cluster of RISC-V processors
- storage datapath work can execute on the DPA rather than host CPU
- full offload separates software control plane from firmware/hardware dataplane
- asynchronous request handling and zero-copy paths are already commercial practice

Lesson:
- small RISC-V cores beside storage hardware are a proven architecture
- our differentiator should be minimal semantics, formal verification, open RTL, and durable transaction identity rather than generic storage offload

### Intel/Altera IPU

Commercial IPUs offload NVMe/TCP and storage infrastructure from host CPUs.

Lesson:
- infrastructure isolation + storage offload has clear value
- these systems remain broad programmable infrastructure processors; durable-tid should remain much smaller

### ScaleFlux computational-storage ASIC

Custom SSD-controller silicon combines embedded CPU cores with purpose-built accelerators in the storage path.

Lesson:
- inline computation close to NAND is commercially viable
- ASICs are justified when fixed functions remove enough host/data-movement cost
- do not tape out until FPGA measurements establish the fixed function

### Samsung SmartSSD

FPGA sits next to the SSD controller and performs inline acceleration.

Lesson:
- FPGA-before-ASIC is a mature path
- storage controller and accelerator can remain separate blocks connected by a high-bandwidth internal path

## Reuse hierarchy

Prefer:

1. existing formally verified/permissive IP block
2. existing permissive CPU/bus/DMA primitive
3. small adapter
4. custom RTL only for the novel durable-tid primitive

Do not write:
- another CPU
- another generic DMA engine
- another SHA core
- another SoC bus
- another general ring implementation

until existing permissive IP is shown insufficient.
