# Toyota Way Applied to Software Engineering

Concrete patterns and antipatterns for each TPS concept.

---

## Jidoka — 自働化 (じどうか) — Stop and fix now

**The principle**: a machine that passes defects downstream is worse than a
machine that stops. Stopping to fix is faster than shipping defects.

**In software**:

| Pattern | Implementation |
|---------|---------------|
| CI stops on red | Build fails → pipeline halts. Nobody pushes past it. |
| Type checker blocks commit | Compilation errors are jidoka. Can't ship what doesn't compile. |
| Linter as pre-commit hook | Runs before the code leaves your machine. |
| Alert → on-call → incident | Production alert = andon pull. Human decides before auto-scaling masks the problem. |
| `triz eval --ci` exits 1 | Merge blocked until criteria pass. |

**Antipatterns** — these all violate jidoka:
- `git push --no-verify` (bypassing the stop)
- "We'll fix it in the next sprint" (passing defects downstream)
- Disabling flaky tests instead of fixing them
- Auto-retry without human investigation
- Marking CI green by deleting the failing test

**The human touch**: jidoka is not just automation. The mandatory second move —
human judgment before resuming — is what separates it from automation.
A CI system that auto-retries on failure without alerting anyone is
自動化 (automation), not 自働化 (autonomation).

---

## Poka-yoke — ポカヨケ (ぽかよけ) — Make the wrong thing impossible

**The principle**: redesign the system so the error cannot be made,
rather than relying on people to not make it.

**In software** — ranked from strongest to weakest:

1. **Eliminate**: make the wrong state unrepresentable in the type system
   ```rust
   // WRONG: runtime check
   fn process(status: String) { if status != "active" { panic!() } }
   // RIGHT: type poka-yoke — wrong state can't be constructed
   enum Status { Active, Inactive }
   fn process(status: Status) { ... }
   ```

2. **Prevent**: compiler rejects the error
   - Rust borrow checker (memory safety poka-yoke)
   - Exhaustive match — forgetting a case is a compile error
   - `#[must_use]` — forgetting to handle a Result is a warning

3. **Detect early**: catch at the boundary, not deep inside
   - Schema validation at API entry, not in business logic
   - Database constraints (NOT NULL, foreign keys, CHECK constraints)
   - Config parsing at startup with typed structs, not stringly-typed maps

4. **Signal**: make the problem immediately visible
   - Required fields with no defaults
   - Naming conventions enforced by lint (`_unused` prefix convention)

**Antipatterns**:
- Runtime panics for invariants that could be compile-time
- Optional fields that should be required
- Stringly-typed configuration
- `unwrap()` in production code (deferred the poka-yoke to runtime)

---

## The 8 wastes — 無駄 (むだ) in software

Toyota identified 7 wastes. Software adds an eighth.

| Waste | 日本語 | Software manifestation |
|-------|--------|----------------------|
| Defects | 不良 | Bugs requiring rework, flaky tests, invalid prod data |
| Overproduction | 過剰生産 | Features nobody uses, premature optimization, gold-plating |
| Waiting | 待機 | Blocked PRs, slow CI, waiting for review or deploy approval |
| Non-utilized talent | 人材の無駄 | Senior engineers doing manual deploys, toil |
| Transportation | 運搬 | Unnecessary handoffs: dev → QA → release → ops → back |
| Inventory | 在庫 | Unmerged branches, unreleased features, stale WIP |
| Motion | 動作 | Context switching, too many meetings, tool sprawl |
| **Extra processing** | 過剰処理 | YAGNI violations, abstraction for its own sake, unused flexibility |

**Most common in software**: inventory (unmerged code) and waiting (review queues).
Unmerged code is the most dangerous — it hides integration risk and accumulates
merge debt. The longer a branch lives, the more it costs.

**Muri** — 無理 (むり) — is often the cause: teams running at 120% capacity
create WIP queues (inventory) and context-switch overhead (motion) trying to
keep everything moving.

---

## 5 Whys — なぜなぜ分析 (なぜなぜぶんせき) — Root cause, not symptoms

**The rule**: keep asking "why" until you reach a systemic cause you can fix.
Stop when the answer is a policy, process, or design failure — never a person.

**Software postmortem example**:
```
Alert: payment service returning 500s
なぜ①: database connections exhausted
なぜ②: connection pool set to 10, traffic spike hit 200 concurrent requests
なぜ③: pool size was never load-tested against production traffic patterns
なぜ④: no load testing stage in the release process
なぜ⑤: release checklist has no performance gate

Root cause: release process lacks a performance acceptance criterion.
Fix: add load test to release gate. Document connection pool sizing formula.
```

**Wrong stop** (person blamed): "engineer forgot to set the pool size" → なぜ③
**Right stop** (system blamed): "release process has no performance gate" → なぜ⑤

**Software-specific gotcha**: "human error" is never a root cause. It's a signal
that the system allowed the error to be consequential. The なぜなぜ continues.

---

## A3 thinking — A3思考 (えーさんしこう) — Clarity on one page

**The principle**: if your problem statement and solution don't fit on one A3
sheet (roughly a landscape piece of paper), your thinking isn't clear enough yet.

**Software mapping**: the RFC/design doc → A3 structure

| A3 section | Software equivalent | Common failure |
|-----------|-------------------|---------------|
| Background | Why this matters, what triggered it | Skipped — jumps to solution |
| Current state | What the system does now, with data | Vague — "it's slow" not "p99=4.2s" |
| Target | Specific measurable goal | Missing — "make it faster" not "p99<200ms" |
| Root cause analysis | 5 Whys or cause-effect | Skipped — solution proposed before cause understood |
| Countermeasures | The proposed changes | This is where most docs start |
| Acceptance criteria | How you'll know it worked | "We'll monitor it" is not criteria |
| Follow-up | Review date, owner | Never written |

**Why most design docs fail**: they start at "countermeasures" and never establish
current state or acceptance criteria. The A3 constraint forces the sections that
are always skipped.

---

## Andon — 行灯 (あんどん) — Make problems visible, stop the line

**The principle**: any worker can stop the entire production line by pulling the
andon cord. Stopping is not failure — passing defects downstream is failure.

**In software**:
- Red CI = andon pull. The correct response is to stop all other pushes and fix it.
- On-call alert = andon pull for production
- `BREAKING CHANGE` in a PR = andon pull for downstream teams
- `triz eval` drift flag = andon pull for council resolution

**The cultural requirement**: andon only works if pulling it is rewarded, not
punished. If stopping the line causes political problems, people stop pulling.
Equivalent: if raising a concern in code review causes friction, people stop raising concerns.

**Antipatterns**:
- Ignoring red CI and continuing to push ("it's someone else's test")
- Silencing flaky tests instead of fixing them
- On-call alert auto-resolved without investigation

---

## Kaizen — 改善 (かいぜん) — Continuous improvement, always

**The principle**: improvement has no end state. Every improvement creates the
conditions for the next improvement. The question is never "are we done?" but
"what's the next most valuable improvement?"

**In software**:
- Boy Scout Rule: leave code better than you found it (even if not your change)
- Weekly tech debt sessions (time-boxed, regular, not a big-bang rewrite)
- Automated refactoring: formatters, linters, migration scripts run in CI
- `triz eval` failed criteria → new `triz solve` → implementation → eval again
  (this loop IS kaizen: each cycle improves the resolution)

**Antipatterns**:
- "Tech debt sprint" as a one-time event (kaizen is not a sprint)
- Waiting for perfect before shipping (kaizen starts from imperfect)
- Rewrites instead of incremental improvement (rewrites are 無理, not 改善)

---

## Genchi genbutsu — 現地現物 (げんちげんぶつ) — Go and see

**The principle**: no decision without visiting the actual place (現地) and
observing the actual thing (現物). Summaries and reports are not substitutes.

**In software**:

| Genchi | Genbutsu |
|--------|---------|
| Go to the actual log line | Not the Datadog summary |
| Go to the actual code | Not the ticket description |
| Go to the actual user's workflow | Not the PM's interpretation |
| Go to the actual performance data | Not the intuition about bottlenecks |
| Reproduce locally | Not diagnose from a screenshot |

**Profiling is genchi genbutsu**: never optimize without a profiler.
Guessing at bottlenecks is the opposite of 現地現物.

**Code review is genchi genbutsu**: read the actual diff, not the PR title.
"Looks good to me" without reading the code is not a review.

---

## Nemawashi — 根回し (ねまわし) — Consensus before the meeting

**The principle**: consult each affected party individually before the formal
decision. No surprises in the room. The meeting ratifies; nemawashi decides.

**In software**:
- Talk to downstream teams before proposing breaking API changes
- Share the RFC draft with skeptics before it's published for comment
- Discuss the architecture informally before the design review
- "I'm thinking of doing X — does that affect you?" is nemawashi

**Why it matters**: the formal RFC/design review process only works if people
have already formed views. Surprises in the meeting create defensive reactions.
Nemawashi before the meeting means the meeting surfaces genuine improvements
rather than territorial reactions.

**Antipatterns**:
- Big-bang migration announced after implementation
- Breaking change in a PR with no prior discussion
- Architecture decision published for comment with a 24-hour deadline

---

## Hoshin kanri — 方針管理 (ほうしんかんり) — Strategy to action

**The principle**: cascade organizational direction down through every level so
daily work connects to strategic goals. Each level translates "what" into "how"
for the level below.

**In software**:
```
Company goal: reduce customer churn by 15%
  ↓ (translation)
Engineering direction: reduce time-to-value for new users
  ↓ (translation)
Team contradiction: onboarding speed (improving) vs setup correctness (worsening)
  ↓ (triz solve)
Resolution: Principle 11 — pre-configure safe defaults, validate asynchronously
  ↓ (implementation)
Corpus entry: confirmed resolution with acceptance criteria met
```

**OKRs without hoshin kanri**: number targets without the cascade. Teams hit
the number by gaming the metric. The strategic contradiction never gets resolved.

**Hoshin kanri with TRIZ**: the TRIZ Journal documents this pairing.
Hoshin identifies the strategic contradiction. TRIZ resolves it. A3 documents it.

---

## Shu-ha-ri — 守破離 (しゅはり) — Stages of mastery

**The principle**: mastery passes through obedience → deviation → transcendence.
Skipping stages produces dangerous half-knowledge.

**In software**:

**守 (obey)** — follow the framework/convention exactly. Don't deviate.
- Junior engineers: follow the team's patterns exactly, even if you disagree
- Learning a new language: write idiomatic code before writing clever code
- Using a new framework: read the docs before customizing
- Wrong at this stage: "I don't see why we need to follow this convention"

**破 (break)** — having mastered the rules, deviate knowingly with reason.
- Senior engineers: adapt patterns to context, with clear articulation of why
- Knowing when to break the convention and being able to defend it
- Wrong at this stage: deviating without being able to explain why

**離 (transcend)** — create new patterns; the rules are internalized.
- Principal engineers: define new conventions for the team
- Build the framework, not just use it
- Wrong at this stage (dangerous): transcending before mastering

**The danger of skipping 守**: junior engineers in 破 mode produce clever code
that nobody else understands, solving problems that didn't need to be solved.

---

## Hansei — 反省 (はんせい) — Reflect before you improve

**The principle**: genuine acknowledgment of failure before improvement can begin.
Hansei is not guilt. It is clear-eyed recognition of what happened and why.
Kaizen — 改善 — cannot begin without it.

**In software**:
- Blameless postmortem: the sequence is hansei → action items, not action items → done
- The postmortem ends at a systemic root cause, never at a person
- "We got lucky it wasn't worse" IS hansei — acknowledge the near-miss
- Closing a postmortem with only action items and no genuine reflection = skipped hansei

**Antipatterns**:
- Postmortem that ends at "human error" as root cause
- Action items added for show, not tracked to completion
- "We're too busy for a postmortem" — too busy for hansei means too busy for kaizen

---

## Ma — 間 (ま) — The value of what is not there

**The principle**: negative space is not empty. It is defined and meaningful.
In architecture, the void between pillars is the space people inhabit.
In music, the rest between notes is what gives rhythm its structure.

**In software**:

- **YAGNI** is 間. The feature not built is not absent — it is deliberately absent.
  The discipline of not building is harder than building.

- **API surface area**: what you don't expose is as important as what you do.
  A small, stable API with clear 間 is harder to design than a large API.

- **Negative comments**: document the 間 explicitly.
  ```rust
  // We deliberately do NOT cache this result. Caching would hide stale
  // reads during failover. The latency cost is acceptable; the correctness
  // cost is not.
  ```

- **Empty matrix cells in TRIZ**: Altshuller left cells empty. No known systematic
  solution. The correct response is to respect the 間, not fill it with noise.

**The violation of 間**: hallucinating an answer when the honest answer is
"we don't know" or "no solution exists here yet." LLMs violate 間 instinctively.
Good systems — and good engineers — do not.

---

## Monozukuri — ものづくり — The craft of making

**The principle**: making is not just output. The spirit, skill, and care
invested in the work are inseparable from the work itself.
職人気質 (しょくにんきしつ, shokunin kishitsu) — the craftsperson's disposition —
means you do not ship work you are not proud of.

**In software**:
- Code that is readable is craft. Code that merely works is output.
- Error messages that actually help the user are craft.
- A well-named function is craft. `do_stuff()` is not.
- A commit message that explains *why* is craft. "fix bug" is not.
- Documentation written for the reader, not the writer, is craft.

**The test**: would you be comfortable if the person who maintains this in
two years knows you wrote it? That discomfort is your shokunin instinct working.

**Not perfectionism**: shokunin kishitsu ships. It doesn't delay forever
waiting for perfect. It ships work that is genuinely good, then improves it.
The corpus `needs_validation` state is the work not yet ready to ship.
The `confirmed` state is the shokunin's stamp.
