---
name: toyota-mlx
description: Apply Toyota Way (A3, PDCA, standard work) with TRIZ for MLX model work. Use for planning, decomposition, benchmarking, and continuous improvement across agents.
---

# toyota-mlx

Use this when coordinating MLX work across teams.

## Standard Work

Every MLX task should include:

1. Problem statement (one sentence)
2. Contradiction (TRIZ): what improves vs what worsens
3. A3 fields:
- Current condition
- Target condition
- Root cause hypothesis
- Countermeasures
- 24h check metric

## PDCA cadence

- Plan: define one small experiment
- Do: run benchmark or conversion
- Check: compare to baseline
- Act: keep, revise, or rollback

## MLX benchmark minimum

Record each run with:

- model + quant
- context length
- prompt class (code/chat/reasoning)
- tok/s
- peak memory
- pass/fail notes

## Pull system

- Keep `todo` queue at 2x active executors
- Limit each engineer to 1 `in_progress`
- Founding engineer handles unblock + review + decomposition

## Jidoka and Andon

- If auth/safety/tooling fails, stop and mark blocked with exact unblock action.
- Do not brute-force retries.

## Yokoten

When a fix works (script, prompt, harness), write it once in shared docs and reference it from future issues.
