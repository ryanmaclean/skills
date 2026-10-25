---
name: mister-game-profile
description: JSON schema for MiSTer translation game profiles — core, ROM, hotspots, glossary, characters. Use when adding a new game to the translation pipeline or debugging why a profile isn't matching.
trigger: game profile, translation profile, hotspot, glossary, character map
---

# MiSTer Game Profile Schema

As of 2026-04-06. Consumed by the translation daemon. See `mister-translation`.

## Full schema

```json
{
  "core": "SNES",
  "rom": "ff6j.sfc",
  "display_name": "Final Fantasy VI (Japan)",
  "lang_src": "ja",
  "lang_dst": "en",
  "hotspots": [
    {"name":"dialogue",  "x":32,  "y":160, "w":192, "h":48},
    {"name":"menu_name", "x":16,  "y":8,   "w":96,  "h":16}
  ],
  "glossary": {
    "セリス": "Celes",
    "ティナ": "Terra",
    "マッシュ": "Sabin"
  },
  "characters": [
    {"jp":"ティナ",    "en":"Terra",  "role":"protagonist"},
    {"jp":"セリス",    "en":"Celes",  "role":"general"}
  ],
  "notes": "Dialogue box lives in lower third; menus use top-left hotspot."
}
```

## Fields

| Field | Type | Required | Purpose |
|-------|------|----------|---------|
| `core` | string | yes | Core name matching `/media/fat/games/<CORE>/` (NES, SNES, Genesis, TG16, GBA, GB) |
| `rom` | string | yes | Filename inside that directory |
| `display_name` | string | no | Human label for logs/OSD |
| `lang_src` | ISO-639 | yes | Source language for OCR + translator |
| `lang_dst` | ISO-639 | yes | Target language |
| `hotspots[]` | array | yes | Regions of the scaler buffer to OCR. Coords in game pixels (pre-scaler) |
| `hotspots[].name` | string | yes | Logical name — `dialogue`, `menu_name`, `hp`, etc. |
| `hotspots[].x/y/w/h` | int | yes | Rect in source-res pixels (256x224 for SNES NTSC) |
| `glossary` | object | no | Raw→preferred translation overrides. Applied post-OCR, pre-translator |
| `characters[]` | array | no | Name pairs for consistent character translation |
| `notes` | string | no | Freeform, logged at daemon start |

## Location

Daemon reads from `/media/fat/Scripts/translate/profiles/<core>_<rom>.json`.

## Hotspot coordinates — how to find them

1. `screenshot` via `/tmp/MiSTer_cmd` at the scene you want
2. Open in image editor, note bounding box of text in source-res pixels
3. Source res: SNES 256x224, NES 256x240, Genesis 320x224, GBA 240x160

The scaler upscales to 1080p but daemon passes source-res coords — it scales
internally using the core's native res.

## Common mistakes

| Mistake | Fix |
|---------|-----|
| Hotspot in 1080p coords instead of source-res | Use native (e.g. 256x224 for SNES) |
| Glossary not applied | Keys are exact-match — check whitespace and fullwidth chars |
| Wrong `core` name | Must match `/media/fat/games/` dir name exactly (case-sensitive) |
| No characters array → inconsistent name TL | Add characters explicitly for protagonists |

## Related

- `mister-translation` — pipeline workflow
- `mister-ops` — core filesystem layout
- `translate` — generic Ollama-based translation
