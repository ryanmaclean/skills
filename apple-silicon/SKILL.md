---
name: apple-silicon
description: "Apple Silicon optimization for VMs, ML inference, and compression. Covers Metal GPU acceleration, MLX model inference, CoreML conversion, M-series zstd compression, Apple Virtualization Framework, and performance tuning. Use when working on macOS M-series performance, QEMU/VM setup, or squeezing more out of Apple hardware."
author: gastown
version: 1.0.0
---

# Apple Silicon Optimization Skill

Comprehensive guide for optimizing applications on Apple Silicon (M1/M2/M3/M4) including GPU acceleration, ML inference, virtualization, and compression.

## Overview

Apple Silicon's unified memory architecture and specialized hardware enable significant performance gains when properly optimized:

| Component | Capability | Use Case |
|-----------|------------|----------|
| **Neural Engine** | 16+ TOPS | Quantized model inference |
| **Metal GPU** | Up to 76 cores | Vector search, embeddings |
| **Unified Memory** | Up to 192GB | Large model loading |
| **ProRes Engine** | Hardware decode | Media processing |


## References

- `references/metal-mlx.md` — Metal GPU acceleration + MLX inference
- `references/coreml-zstd.md` — CoreML conversion + M-series zstd compression
- `references/vf-tuning.md` — Apple Virtualization Framework, performance tuning, Datadog, best practices
