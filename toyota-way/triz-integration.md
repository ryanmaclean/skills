# Toyota Way × TRIZ — Integration Map

Japanese management concepts informing the triz CLI tool design.
Romaji accompanied by correct kanji and readings throughout.

---

## 1. ポカヨケ (ぽかよけ) × TRIZ Principle 11 (Beforehand Cushioning)

The deepest connection. Academic papers at P2SL Berkeley explicitly link
mistakeproofing principles to TRIZ inventive principles.

ポカヨケ = prevent the error before it can occur.
Principle 11 = prepare compensating means in advance.

**In triz**: `triz eval --ci` IS ポカヨケ. Acceptance criteria written at
spec time, checked before merge. Spec 014 states it directly:
*"the criteria exist before the failure, not after."*

The merge gate enforces ポカヨケ. The `triz eval` spec is the most Japanese
thing in the tool — it just wasn't named that way when written.

**Spec 014 annotation**: note that `triz eval --ci` implements ポカヨケ
at the merge gate, implementing Principle 11 as a CI tool.

---

## 2. 自働化 (じどうか) × Principle 25 (Self-Service) + Principle 10 (Preliminary Action)

自働化 = automation with a human touch. Two mandatory moves:
1. Autonomous detection — machine stops itself
2. Human judgment — nothing resumes without a person deciding

**In triz**:
- `triz eval --ci` exits 1 on failure = the autonomous stop
- `require_approval_for: [triz_corpus_confirm]` in `.triz.policy.yaml`
  = the human-touch gate before corpus is written

**行灯 (あんどん) parallel**: when a council agent recommends a principle
outside `matrix_principles`, spec 007 surfaces it as `"drift": true` rather
than silently accepting it. The agent pulled the 行灯. A human must decide
whether to proceed or investigate.

---

## 3. なぜなぜ分析 (なぜなぜぶんせき) — The missing piece

なぜなぜ分析 drills from symptom to root contradiction by asking なぜ (why)
five times. TRIZ starts from the contradiction — but doesn't help users *find*
or *articulate* it. This is the gap most users fall into.

```
symptom:  deploys are slow
なぜ①:   CI pipeline takes 20 minutes
なぜ②:   full test suite runs on every commit
なぜ③:   no dependency graph between tests and modules
なぜ④:   tests were written without isolation in mind
なぜ⑤:   no specification required it

矛盾 (むじゅん):  test coverage (改善) vs deploy speed (悪化)
→ triz solve --improving "test coverage" --worsening "deploy speed"
```

**Spec 016 candidate**: `triz whys` — interactive なぜなぜ分析 guide.
Pure Rust, no LLM for the drill structure. LLM only for final
contradiction articulation. Output: a ready-to-run `triz solve` invocation.

---

## 4. A3思考 (えーさんしこう) × `triz solve` output

A3 = entire problem documented on one A3 sheet (420×297mm).
Toyota's 計画・実行・確認・改善 (PDCA) made visible and portable.

`triz solve` produces most of an A3 already:

| A3 section | 日本語 | triz field |
|-----------|--------|-----------|
| 背景 | はいけい | `--context` flag |
| 現状 | げんじょう | contradiction statement |
| 目標 | もくひょう | `--improving` parameter |
| 原因分析 | げんいんぶんせき | matrix lookup + principle |
| 対策 | たいさく | `resolution` field |
| 受入基準 | うけいれきじゅん | `acceptance_criteria` |
| フォロー | — | `triz eval` |

**Small spec 003 addition**: `--output a3` flag renders solve output as
structured A3 markdown. Useful for Japanese stakeholders who recognise the format.

---

## 5. 方針管理 (ほうしんかんり) × Corpus as strategic alignment

方針管理 = policy deployment. Cascade organizational 方針 (policy/direction)
down to implementation. TRIZ finds the *どうやって (how)* when 方針管理
sets the *なに (what)*.

The TRIZ Journal documents this pairing: "Combining TRIZ with the disciplined
system of 方針管理 gives companies flexibility and accountability while asking
the right questions and finding the strongest solutions."

**In triz**: the corpus is the 方針管理 audit trail — which contradictions were
resolved, by which principles, confirmed by whom. The `org` and `team` fields
in corpus entries (spec 005) are the deployment mechanism.

`triz toto` (spec 012) — time-series forecast of contradiction hotspots —
is a 方針管理 diagnostic: which strategic contradictions are becoming more
frequent? Which principles are the factory over-relying on?

---

## 6. 守破離 (しゅはり) × The `--tier` system

Altshuller's 5 levels of invention map almost exactly to 守破離:

| 守破離 | 漢字の意味 | Altshuller level | triz tool |
|--------|-----------|-----------------|-----------|
| 守 (まもる) | Protect, obey | 1–2: known solutions | `--tier fast`, literal principle application |
| 破 (やぶる) | Break, question | 3: resolve within field | `--tier standard`, council, cross-domain |
| 離 (はなれる) | Leave, transcend | 4–5: new phenomena | `--tier quality`, council-spawn, drift detection |

The `--auto-tier` flag in spec 003 implicitly walks 守 → 破 → 離.
Worth making this parallel explicit in `triz doctor` output and documentation.

---

## 7. 根回し (ねまわし) × `council.nu`

根回し = "going around the roots." Consult each stakeholder *individually*
before the formal meeting so no surprises emerge in the room. The decision
is made before the decision meeting.

**In triz**: `council.nu` (spec 004) with N-parallel framings is computational
根回し. Each perspective consulted before committing to a resolution.

The key parallel: in 根回し, disagreement surfaces *before* the decision.
In council, minority votes are preserved in output — not discarded — so
dissent is documented even when the majority carries.

稟議 (りんぎ) is the formal record after 根回し completes.
The corpus tombstone = the 判子 (はんこ, approval stamp) on the 稟議.

---

## 8. 間 (ま) × Empty matrix cells

間 = negative space. The meaningful pause. What is *not* there defines what is.
In architecture: the void between pillars. In music: the rest between notes.
In conversation: the silence that must not be filled.

**In TRIZ**: empty matrix cells are 間. Altshuller left 30%+ of the matrix
empty deliberately — no known systematic solution exists for those contradiction
pairs. They represent the frontier of the methodology in 1979.

The triz spec rule — *"empty cell → `no_recommendation: true`, no LLM call,
no interpolation"* — honours 間.

Filling the silence with a hallucinated principle is the violation of 間.
The temptation is real: an LLM will gladly invent a plausible-sounding
principle for any input. The architecture refuses this. The 間 is preserved.

---

## 9. 反省 (はんせい) × `triz eval` failure

反省 = self-reflection. Mandatory acknowledgment of failure before improvement.
In Toyota culture, 反省 is not optional and not brief. 改善 (kaizen) cannot
begin without genuine 反省. The sequence is: 反省 → 改善, not: 改善 → 反省.

**In triz**: when `triz eval` returns failed criteria, that IS the 反省 moment.
The feedback loop from eval → solve is 反省 → 改善 made executable:

```bash
triz eval seed0002 --output json \
  | jq -r '.criteria[] | select(.result=="fail") | .text' \
  | triz solve --improving "criterion compliance" --worsening "implementation speed"
```

The failed criteria ARE the 反省. The new `triz solve` invocation IS the 改善.

---

## 10. ものづくり × The corpus as shokunin's record

ものづくり is not just 製造 (manufacturing). It is the spirit, skill, and craft
invested in making. 職人気質 (しょくにんきしつ, shokunin kishitsu) = the craftsman's
pride that refuses to ship work unworthy of the maker's name.

**In triz**: the corpus is the 職人's record. `needs_validation` entries are
work not yet stamped. `confirmed` entries are work the shokunin has approved.

The two-state corpus (unconfirmed → confirmed via tombstone) maps directly to
the 職人's practice of not stamping work until satisfied. `triz eval --auto-confirm`
on 100% pass rate = the stamp. No shortcuts.

---

## Mapping summary

| 日本語 | 読み | triz equivalent | Spec | Gap? |
|--------|------|----------------|------|------|
| ポカヨケ | ぽかよけ | `triz eval --ci` merge gate | 014 | Name it explicitly |
| 自働化 | じどうか | eval stop + approval gate | 006/014 | Name it explicitly |
| 行灯 | あんどん | Drift detection in council | 007 | Already there |
| なぜなぜ分析 | なぜなぜぶんせき | **Missing** | — | → Spec 016 |
| A3思考 | えーさんしこう | `--output a3` flag | 003 | Small addition |
| 方針管理 | ほうしんかんり | Corpus org/team + triz toto | 005/012 | `triz hoshin` view possible |
| 守破離 | しゅはり | `--tier` fast/standard/quality | 003 | Document the parallel |
| 根回し | ねまわし | `council.nu` parallel framings | 004 | Rename in docs |
| 稟議 | りんぎ | Corpus tombstone = 判子 | 005 | Note the parallel |
| 間 | ま | Empty cell → no LLM call | 002 | Already correct — preserve it |
| 反省 | はんせい | `triz eval` failed criteria | 014 | Make the 反省→改善 loop visible |
| 改善 | かいぜん | Eval → solve feedback loop | 014 | Already specced |
| ものづくり | ものづくり | Confirmed corpus as craft record | 005 | Philosophy only |
| 現地現物 | げんちげんぶつ | `triz doctor` / read before fixing | 011 | Philosophy only |

**Most actionable gap**: なぜなぜ分析 → `triz whys` (Spec 016).
Guides users from symptom to contradiction. No LLM needed for structure.
Closes the articulation gap that causes most `triz solve` failures.
