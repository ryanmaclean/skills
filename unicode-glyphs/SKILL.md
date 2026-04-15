---
name: unicode-glyphs
description: >
  Use when choosing Unicode symbols, icons, or glyphs for card identity tokens,
  terminal pane titles, filenames, UI labels, or priority/status encoding.
  Trigger phrases: "which symbol", "pick an icon", "glyph for", "unicode for",
  "what emoji", "token for", "BMP safe", "playing card suit".
  Covers playing cards, geometric shapes, special marks, BMP vs SMP safety,
  font recommendations, and filename pitfalls.
---

# Unicode Glyphs — Reference for Card & Terminal Systems

## Core Principle

Two glyphs per card: `glyph` (SMP, rich display) + `token` (BMP, portability-safe fallback).

## BMP vs SMP — The Critical Divide

| Tier | Range | Examples | Safe for filenames? | Terminal? |
|------|-------|----------|---------------------|-----------|
| BMP | U+0000–U+FFFF | ♠♥♦♣ ★☆ ⚀–⚅ ※ ∴ | ✅ Yes | ✅ Yes |
| SMP | U+10000+ | 🂡🂱🃁 🃟 alchemical | ⚠️ APFS/macOS often OK, portability risk | ⚠️ Depends on font |

**Rule:** `glyph` field = SMP playing card (QL/Finder/SwiftUI). `token` field = BMP fallback for scripts, cross-platform tooling, and conservative terminal surfaces.

## Playing Cards (U+1F0A0–U+1F0FF) — Semantic Encoding

**Suit → Team:**
- ♠ Spades (U+2660) → CLI / infrastructure
- ♥ Hearts (U+2665) → Architecture / design
- ♦ Diamonds (U+2666) → Quality / QA
- ♣ Clubs (U+2663) → Platform / tooling

**SMP card glyphs** (primary use: `glyph` field; optional for macOS-local filenames):
- 🂠 U+1F0A0 Card Back = unrevealed / hidden estimate
- 🂡 U+1F0A1 Ace of Spades = P1/CLI
- 🂱 U+1F0B1 Ace of Hearts = P1/Arch
- 🃁 U+1F0C1 Ace of Diamonds = P1/Quality
- 🃑 U+1F0D1 Ace of Clubs = P1/Platform
- 🃏 U+1F0CF Black Joker = wildcard/emergency ← use this one

**Joker rendering (do not use U+1F0BF):**
| Codepoint | Name | Fallback | Verdict |
|-----------|------|----------|---------|
| U+1F0CF | Black Joker 🃏 | — | ✅ best supported |
| U+1F0DF | White Joker 🃟 | — | ✅ decent |
| U+1F0BF | Red Joker | ◆ (diamond suit!) | ❌ silent misread — looks like a diamonds card |

The red joker's fallback is ◆ — indistinguishable from an intentional diamonds suit marker. Never use U+1F0BF.

**Rank → Priority:** Ace=P1, King/Queen=P2, Jack=P3, 2–10=P4

## BMP Glyphs — Priority & Status

**Dice U+2680–U+2685** (P1–P6 encoding, universally understood):
```
⚀ P1  ⚁ P2  ⚂ P3  ⚃ P4  ⚄ P5  ⚅ P6
```

**Geometric shapes** (state encoding):
```
● Running    ○ Pending    ■ Done    □ Blocked
▲ Urgent     ★ Featured   ◆ Critical
```

**Chess pieces U+2654–U+265F** (role encoding):
```
♔ King = critical  ♘ Knight = tricky  ♙ Pawn = atomic task
```

## Special Marks

| Glyph | Codepoint | Meaning | Best use |
|-------|-----------|---------|----------|
| ※ | U+203B | komejirushi / "attention" | `decision_required: true` cards |
| ∴ | U+2234 | therefore / ergo | merge-ready, conclusion reached |
| ∵ | U+2235 | because / rationale | spec/design stage |
| ⚠ | U+26A0 | warning | blocked, needs review |
| ✓ | U+2713 | check | done stages |

## Block Elements (U+2580–U+259F)

Only glyphs **guaranteed** in every terminal font. Use for progress/density display, NOT identity:
```
▁▂▃▄▅▆▇█  — progress bars
░▒▓█      — density levels
```

## Font Recommendations

| Use case | Recommended | Notes |
|----------|-------------|-------|
| Terminal (primary) | JetBrains Mono + Nerd Fonts | Best coverage, ligatures, free |
| Terminal (widest Unicode) | Iosevka | Covers most of BMP + many SMP blocks |
| macOS Terminal default | SF Mono | Verify U+1F0A0–U+1F0FF in your terminal profile |
| QL/SwiftUI | System default | Built-in macOS UI fonts render card glyphs well |

**Contrast caveat:** Card glyphs typically read best over dark backgrounds. There is no Unicode "reverse" card-face variant; do inversion in UI styling (background/foreground), not by codepoint swap.

## Filename Safety Rules

```
✅ Conservative portable set: ♠ ♥ ♦ ♣ ★ ⚀ ※ ∴ (all BMP)
⚠️ macOS/APFS local workflows can use SMP: 🂠 🂡 🂻 🃏
```

macOS 26.3 local validation with `🂠-feat-auth.jobcard` succeeded for:

- `rg`, `tail`, `eza`, `bat`
- `bop inspect feat-auth`
- `bop logs feat-auth`
- `bop logs -f feat-auth` (running cards)

Portability warning remains: shell/tool behavior can diverge across Linux distros, remote CI, legacy terminals, and older scripting environments.

`exa` is deprecated in Homebrew; use `eza`.

## Two-Glyph Architecture (jobcard Meta)

```rust
// In Meta struct:
pub glyph: Option<String>,  // SMP playing card — QL/Finder/SwiftUI hero
pub token: Option<String>,  // BMP fallback — pane title, safe aliases, portability mode
```

**Example:**
```json
{
  "glyph": "🂡",     // Ace of Spades — QL preview hero
  "token": "⚀",      // Die face 1 — P1 priority in terminal
  "id": "feat-auth"   // keep ID stable; filename may carry glyph prefix
}
```

## Stage Display Names

Never use `.to_uppercase()` or `.capitalized` on stage names — "QA" → "Qa" bug.
Use a lookup table:

```rust
static STAGE_LABELS: &[(&str, &str)] = &[
    ("spec",      "Spec"),
    ("plan",      "Plan"),
    ("implement", "Code"),
    ("qa",        "QA"),
];
```

## Quick Checklist

- [ ] Rich display surface (SwiftUI/QL)? → `glyph` field, SMP ok
- [ ] Terminal/filename surface? → `token` field by default; allow SMP filenames only for macOS-local workflows
- [ ] Priority visible at a glance? → dice ⚀–⚅
- [ ] Decision needed? → prepend ※
- [ ] Merge-ready? → ∴ in status
- [ ] Font recommendation needed? → JetBrains Mono + Nerd Fonts
