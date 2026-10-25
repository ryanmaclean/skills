---
name: triz-mlx-roadmap
description: "Apply TRIZ to MLX model-roadmap contradictions where improving one goal harms another (quality vs speed, memory vs context length, novelty vs reliability). Use when deciding which models and quantizations to convert next, what to run in parallel, and what to defer."
---

# TRIZ MLX Roadmap

## Overview

Turn model-porting ambiguity into an execution queue. Focus on contradictions, not preferences.

## Workflow

1. State the contradiction in one sentence.
- Use format: "Improve X without worsening Y under constraint Z."
- Example: "Increase benchmark quality without exceeding 24GB RAM and without cutting decode speed in half."

2. Quantify constraints before proposing models.
- Required: available disk, peak memory budget, minimum decode throughput, acceptable conversion time.
- In this environment, prefer NAS-backed model storage: `/Volumes/downloads/mlx/models`.

3. Map contradiction to TRIZ principles.
- `#1 Segmentation`: split work into independent lanes (convert, benchmark, eval).
- `#10 Preliminary action`: pre-define acceptance gates before conversion.
- `#15 Dynamics`: run q4 first, then raise bits only where quality fails.
- `#23 Feedback`: benchmark after each conversion; update queue.
- `#24 Intermediary`: use standard result JSONL format as handoff artifact.

4. Apply the MLX candidate sieve in order.
- Gate A: architecture support exists in local `mlx_lm.models` (or model already converts successfully).
- Gate B: no suitable `mlx-community` artifact already covers the need.
- Gate C: storage/memory fit at q4 or mixed quant on target host.
- Gate D: benchmark/release evidence justifies effort.

5. Emit a plan with one serial path and parallel lanes.
- Serial path: highest-risk blocker first (usually compatibility conversion).
- Parallel lane 1: benchmark harness and eval prompts.
- Parallel lane 2: second model conversion on spare disk/memory window.
- Parallel lane 3: reporting/decision updates.

## Output Contract

Always return:
- `Contradictions`: 2-4 explicit X vs Y statements.
- `Chosen principles`: TRIZ principle numbers with one-line rationale.
- `Next 72h plan`: serial critical path plus parallel lanes.
- `Stop/Go gates`: measurable thresholds (memory, tokens/sec, quality delta).

## Quantization Heuristics

- Use `q4` as first conversion for new >=20B dense models.
- Try `mixed_4_6` before `q6` when q4 quality is borderline.
- Use `q8` only when quality deltas are proven and memory headroom is clear.
- Defer unsupported `model_type` families until upstream MLX support lands.

## Minimal Plan Template

```text
Contradictions:
1) ...
2) ...

Chosen TRIZ principles:
- #...

Serial path:
1) ...
2) ...

Parallel lanes:
1) Lane A ...
2) Lane B ...
3) Lane C ...

Stop/Go gates:
- Memory: ...
- Throughput: ...
- Quality: ...
```
