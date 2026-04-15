---
name: toyota-way
description: Toyota Production System (TPS) and Toyota Way for software engineering — lean thinking, waste elimination, root cause analysis, and continuous improvement. Use when identifying waste (muda/muri/mura), running blameless postmortems, doing 5 Whys, writing A3 problem statements, or connecting lean philosophy to TRIZ contradiction resolution.
---

# toyota-way — Toyota Way & TPS Reference

Toyota Production System (TPS) and Toyota Way concepts for software engineering.
English-first. Japanese terms shown inline with kanji and reading.

Use when applying lean thinking, identifying waste, structuring problem-solving,
or connecting the triz tool to Japanese management philosophy.

---

## Scripts

| Script | Use |
|--------|-----|
| `script/postmortem.nu "Incident Title"` | Blameless postmortem with 5 Whys + triz integration |
| `script/postmortem.nu "Title" --save file.md` | Write postmortem to file |
| `script/a3.nu "Problem Title"` | A3 one-page problem clarity template |
| `script/a3.nu "Title" --save file.md` | Write A3 to file |

Both scripts print a structured markdown template to stdout. Fill in bracketed fields.
Both embed a `triz solve` invocation in the root cause section — run it if a contradiction emerges.

---

## The two pillars

**Just-in-Time** — make only what is needed, only when needed, only in the
amount needed. Eliminates inventory (work-in-progress), over-production, waiting.

**Jidoka** — 自働化 (じどうか) — autonomation with a human touch. The kanji 働
(human work) is deliberate — not 自動化 (pure automation). Machines detect defects
and stop automatically; humans investigate before anything resumes. Two mandatory
moves: autonomous detection, then human judgment. Neither alone is jidoka.

---

## The three wastes

**Muda** — 無駄 (むだ) — waste. Unnecessary work, unused features, dead code,
over-engineering. The target of most lean effort.

**Muri** — 無理 (むり) — overburden. Asking more than capacity allows. Context
window exhaustion, sprints that require heroics, agents without timeouts.

**Mura** — ムラ (むら) — unevenness. Inconsistency, variability, flakiness.
Non-deterministic builds, variable LLM output without temperature control,
batch-and-queue workflows that create feast-or-famine load.

All three interact: muri creates mura, mura creates muda. Fix muri first.

---

## Problem-solving tools

**Poka-yoke** — ポカヨケ (ぽかよけ) — mistake-proofing. Prevent errors before
they occur rather than inspect them out afterward. Named from 馬鹿避け (fool-avoiding).
The device makes the wrong action impossible, or immediately visible.

Software: type systems, required fields, merge gates, acceptance criteria checked
before merge. Rust's borrow checker is poka-yoke. `triz eval --ci` is poka-yoke.
Pairs directly with TRIZ Principle 11 (Beforehand Cushioning) — academic literature
at P2SL Berkeley explicitly connects mistakeproofing to TRIZ inventive principles.

**5 Whys** — なぜなぜ分析 (なぜなぜぶんせき) — ask "why" five times to reach root
cause. Each answer becomes the next question. Stop when you reach a cause you can
actually fix. The analysis is complete when the root cause is a policy or system
failure, not a person. Pairs naturally with TRIZ: 5 Whys finds the contradiction;
TRIZ resolves it.

**A3 thinking** — A3思考 (えーさんしこう) — entire problem documented on one A3
sheet (420×297mm). Structured sections: background → current state → target →
root cause analysis → countermeasures → acceptance criteria → follow-up.
Toyota's PDCA (計画・実行・確認・改善, plan-do-check-improve) made visible and portable.
Forces clarity: if it doesn't fit on one page, the thinking isn't clear yet.

**Andon** — 行灯 (あんどん) — the signal light showing production status. Origin:
a paper lantern. Anyone on the line can pull the andon cord to stop production
when they spot a defect. Stopping to fix is faster than passing defects downstream.
Software: CI red build, `triz eval` exit 1, drift flag in council output.

**Genchi genbutsu** — 現地現物 (げんちげんぶつ) — "go and see." Go to the actual
place (現地), observe the actual thing (現物). No decision without direct observation.
Software: reproduce the bug locally before diagnosing, read the actual code before
proposing a fix, run `triz doctor` before assuming the environment is healthy.

---

## Strategy and alignment

**Hoshin kanri** — 方針管理 (ほうしんかんり) — policy deployment. Cascade
organizational direction (方針) down to implementation through the whole structure.
Sets the "what" at each level; teams determine the "how." The TRIZ Journal documents
pairing with TRIZ: hoshin kanri defines the strategic contradiction to resolve;
TRIZ finds the resolution; A3 documents it; the corpus records it.

**Nemawashi** — 根回し (ねまわし) — "going around the roots." Build consensus by
consulting each stakeholder individually before the formal meeting. No surprises in
the room; the decision is effectively made before the decision meeting. The formal
meeting ratifies what nemawashi already achieved.

**Ringi** — 稟議 (りんぎ) — the formal written proposal circulated for sequential
approval stamps (判子, hanko). Creates an accountability trail. Each stamp = explicit
acknowledgment of responsibility. Software: the PR description as ringi document,
required reviewers as hanko holders, merge as the final stamp.

**Heijunka** — 平準化 (へいじゅんか) — production leveling. Smooth out demand
spikes; avoid batch-and-queue. Even flow over time beats maximum throughput followed
by idle. Software: steady spec dispatch over waves rather than sprint crunch, `--tier
fast` for most calls rather than quality-tier for everything.

---

## Learning and mastery

**Shu-ha-ri** — 守破離 (しゅはり) — three stages of mastery from martial arts.

守 (まもる, obey) — learn the rules exactly as taught. Do not deviate.
Understand the form before questioning it.

破 (やぶる, break) — having mastered the rules, examine why they exist.
Begin to adapt. The deviation is informed, not ignorant.

離 (はなれる, leave) — transcend the rules entirely. Create new forms.
The rules are internalized so deeply they no longer constrain.

Altshuller's 5 levels of invention map almost exactly to 守破離. The triz
`--tier` system (fast/standard/quality) is a rough 守破離 proxy.

**Hansei** — 反省 (はんせい) — self-reflection before improvement. Mandatory
acknowledgment of failure or shortcoming. In Toyota culture, hansei is not
optional and not brief. Kaizen (改善, continuous improvement) cannot begin
without genuine hansei. The sequence is hansei → kaizen, never the reverse.

**Kaizen** — 改善 (かいぜん) — continuous improvement. Small, daily, everyone's
responsibility. Not a project with an end date — a permanent practice. The
improvement never completes. Each kaizen creates the conditions for the next.

---

## Philosophy

**Monozukuri** — ものづくり — the art and spirit of making things. Not just
manufacturing (製造, せいぞう). Covers technique (技術), spirit (精神), and the
craftsman's pride (職人気質, しょくにんきしつ). A shokunin (職人, craftsperson)
does not ship work they are not proud of. The work carries the maker's name.

**Ma** — 間 (ま) — negative space. The meaningful pause between notes, the void
between pillars, the silence in conversation. What is *not* there defines what
is there. In TRIZ: empty matrix cells are 間. Altshuller left them empty
deliberately — no known solution exists for those contradiction pairs.
Filling the silence with a hallucinated answer violates 間.

**Shokunin kishitsu** — 職人気質 (しょくにんきしつ) — the craftsman's disposition.
Pride that refuses to release work that falls short of one's own standard.
Not perfectionism (which delays indefinitely) but craft integrity (which ships
when the work is genuinely good). The corpus `confirmed` state is the shokunin's
stamp. `needs_validation` is the work not yet ready to bear the maker's name.

---

## Quick reference

| Term | 日本語 | Reading | Core idea |
|------|--------|---------|-----------|
| Jidoka | 自働化 | じどうか | Auto-stop + human judgment |
| Poka-yoke | ポカヨケ | ぽかよけ | Prevent errors before they occur |
| Muda | 無駄 | むだ | Waste |
| Muri | 無理 | むり | Overburden |
| Mura | ムラ | むら | Unevenness |
| Kaizen | 改善 | かいぜん | Continuous improvement |
| Hansei | 反省 | はんせい | Reflection before improvement |
| Ma | 間 | ま | Meaningful absence |
| Nemawashi | 根回し | ねまわし | Consensus before the meeting |
| Shu-ha-ri | 守破離 | しゅはり | Obey → Question → Transcend |
| Hoshin kanri | 方針管理 | ほうしんかんり | Goals cascaded to action |
| Monozukuri | ものづくり | ものづくり | Craft as calling |
| Andon | 行灯 | あんどん | Stop-signal: problem here |
| Genchi genbutsu | 現地現物 | げんちげんぶつ | Go and see yourself |
| Heijunka | 平準化 | へいじゅんか | Level the flow |
| 5 Whys | なぜなぜ分析 | なぜなぜぶんせき | Drill to root cause |
| A3 thinking | A3思考 | えーさんしこう | One-page problem clarity |
| Ringi | 稟議 | りんぎ | Formal approval trail |
| Shokunin kishitsu | 職人気質 | しょくにんきしつ | Craftsman's pride |
