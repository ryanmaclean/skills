#!/usr/bin/env nu
# Blameless postmortem template with 5 Whys section.
# Prints a structured markdown document to stdout.
# Fill in bracketed fields; run triz whys if contradiction is unclear.
#
# Usage:
#   nu postmortem.nu "Payment service 500s on checkout"
#   nu postmortem.nu "Payment service 500s" --save incident-2026-03-07.md

def main [
    title: string = "Incident Title"
    --save: string = ""     # Write to file instead of stdout
] {
    let date = (date now | format date "%Y-%m-%d")
    let doc = $"# Postmortem: ($title)

**Date**: ($date)
**Severity**: [P1 / P2 / P3]
**Duration**: [start time] → [end time] ([N] minutes)
**Author**: [name]
**Reviewers**: [names]

---

## Impact

- **Users affected**: [number or %]
- **Services affected**: [list]
- **Data loss**: [yes/no — if yes, describe]

---

## Timeline

| Time | Event |
|------|-------|
| HH:MM | [first alert / symptom observed] |
| HH:MM | [investigation started] |
| HH:MM | [root cause identified] |
| HH:MM | [mitigation applied] |
| HH:MM | [service restored] |

---

## Root Cause — 5 Whys

**Symptom**: [observable failure — what users experienced]

| # | Why? | Answer |
|---|------|--------|
| 1 | Why did [symptom] occur? | [answer] |
| 2 | Why did [answer 1] happen? | [answer] |
| 3 | Why did [answer 2] happen? | [answer] |
| 4 | Why did [answer 3] happen? | [answer] |
| 5 | Why did [answer 4] happen? | [answer — this should be a policy or system gap] |

**Root cause**: [answer 5 restated clearly]

> If a system or policy failure is not at the bottom, keep drilling.
> \"Human error\" is never a valid stop — it is always the next why.

**Contradiction identified** (if applicable):
Run: `triz whys` to articulate improving/worsening parameters.
Then: `triz solve \"[improving]\" \"[worsening]\" -c \"[context]\"`

---

## Countermeasures

| Action | Owner | Due | Status |
|--------|-------|-----|--------|
| [immediate fix applied] | [name] | done | [x] |
| [preventive change] | [name] | [date] | [ ] |
| [detection improvement] | [name] | [date] | [ ] |
| [process change] | [name] | [date] | [ ] |

---

## What went well

- [thing that worked — detection, communication, rollback speed]

## What went poorly

- [thing that failed — detection gap, runbook missing, etc.]

---

## Hansei

*Reflection before improvement. Acknowledge failure honestly.*

[One paragraph: what did we miss, what assumption was wrong,
what would we do differently if we could replay this?]

---

## Follow-up

- [ ] All countermeasures have owners and due dates
- [ ] Runbook updated with new failure mode
- [ ] Alert threshold adjusted if detection was slow
- [ ] Postmortem shared with affected teams
- [ ] Kaizen item added to backlog if systemic change needed
"

    if ($save | is-empty) {
        print $doc
    } else {
        $doc | save --force $save
        print $"Saved to ($save)"
    }
}
