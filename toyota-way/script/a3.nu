#!/usr/bin/env nu
# A3 thinking template — one-page problem clarity.
# Structured sections: background → current → target → root cause →
# countermeasures → acceptance criteria → follow-up.
# If it doesn't fit on one page, the thinking isn't clear yet.
#
# Usage:
#   nu a3.nu "CI pipeline too slow"
#   nu a3.nu "CI pipeline too slow" --save a3-ci-pipeline.md

def main [
    title: string = "Problem Title"
    --save: string = ""
] {
    let date = (date now | format date "%Y-%m-%d")
    let doc = $"# A3: ($title)

**Owner**: [name] | **Date**: ($date) | **Team**: [team]

---

## 1. Background

*Why does this matter? What is the strategic context?*

[2-3 sentences. Connect to team or org goal. No more than this —
if it needs more, the scope is too large.]

---

## 2. Current State

*What is actually happening? Measured, not estimated.*

| Metric | Current | Source |
|--------|---------|--------|
| [e.g. p95 deploy time] | [value] | [dashboard link] |
| [e.g. deploy frequency] | [value] | [source] |

[Optional: flow diagram or timeline showing the current process]

---

## 3. Target State

*What does good look like? Specific and measurable.*

| Metric | Target | By when |
|--------|--------|---------|
| [same metric] | [target value] | [date] |

[One sentence on why this target, not a different one]

---

## 4. Root Cause Analysis

*5 Whys — stop at a system/policy cause, not a person.*

**Problem statement**: [restate measurable gap between current and target]

| # | Why? | Answer |
|---|------|--------|
| 1 | [why does the gap exist?] | [answer] |
| 2 | [why answer 1?] | [answer] |
| 3 | [why answer 2?] | [answer] |
| 4 | [why answer 3?] | [answer] |
| 5 | [why answer 4?] | [root — policy or system gap] |

**Root cause**: [answer 5 in one sentence]

**Contradiction** (if improving X worsens Y):
`triz solve \"[improving]\" \"[worsening]\" -c \"[context]\"`

---

## 5. Countermeasures

*Specific actions that address the root cause. Not symptoms.*

| # | Countermeasure | Addresses why # | Owner | Due |
|---|---------------|----------------|-------|-----|
| 1 | [action] | [why #] | [name] | [date] |
| 2 | [action] | [why #] | [name] | [date] |

---

## 6. Acceptance Criteria

*How will we know the countermeasures worked?*

- [ ] [metric] reaches [target] by [date]
- [ ] [observable behavior] confirmed in [environment]
- [ ] No regression on [related metric]

---

## 7. Follow-up

*Planned check-in: [date]. Owner confirms metrics and closes open items.*

| Item | Status |
|------|--------|
| Countermeasures complete | [ ] |
| Target metrics achieved | [ ] |
| Learnings shared with team | [ ] |
| Kaizen item created if systemic | [ ] |

---

*A3 rule: if this doesn't fit one page when printed, cut until it does.*
*The constraint is the point — clarity emerges from the limit.*
"

    if ($save | is-empty) {
        print $doc
    } else {
        $doc | save --force $save
        print $"Saved to ($save)"
    }
}
