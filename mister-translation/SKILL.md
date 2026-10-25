---
name: mister-translation
description: End-to-end MiSTer live translation pipeline — load game, start daemon, capture scaler buffer, OCR, translate, display via OSD. Use when debugging the translation stack, tailing logs, or walking the pipeline from ROM to subtitle.
trigger: mister translate, live translation, ocr pipeline, osd subtitle, scaler capture, ff6 translate
---

# MiSTer Translation Pipeline

End-to-end workflow from ROM load to OSD subtitle. Complements `mister-ops`.

## Stages

| # | Stage | Tool | Input → Output |
|---|-------|------|----------------|
| 1 | Load game | Wizzo API | ROM path → core loaded |
| 2 | Start daemon | systemd | — → `mister-translate.service` |
| 3 | Capture | `scaler_capture.py` | `/dev/mem @ 0x20000000` → JPEG every N ms |
| 4 | OCR | Tesseract or Coral TPU | JPEG → raw text |
| 5 | Translate | Ollama (qwen3-coder) or cloud | raw text → English |
| 6 | Display | `osd_subs.py` → `osd_info` | translated line → MiSTer OSD |

## Start the pipeline

```bash
# 1. Launch game via Wizzo
curl -X POST http://mister.local:8182/api/games/launch \
  -H 'Content-Type: application/json' \
  -d '{"path":"/media/fat/games/SNES/ff6j.sfc"}'

# 2. Start daemon on capture host
ssh mister.local 'systemctl start mister-translate'

# 3. Follow logs
ssh mister.local 'journalctl -u mister-translate -f'
```

## Debug tails

```bash
journalctl -u mister-translate -f          # daemon
tail -f /var/log/mister-ocr.log            # OCR per-frame
tail -f /var/log/mister-translate.log      # translation calls
```

## Game profile (consumed by daemon)

See `mister-game-profile` for full schema. Minimum:

```json
{
  "core": "SNES",
  "rom": "ff6j.sfc",
  "hotspots": [{"name":"dialogue","x":32,"y":160,"w":192,"h":48}],
  "glossary": {"セリス":"Celes"},
  "characters": [{"jp":"ティナ","en":"Terra"}]
}
```

## Common failures

| Symptom | Cause | Fix |
|---------|-------|-----|
| OCR returns blank | Scaler offset wrong, core not running | See `mister-ops` — confirm `0x20000000` |
| Translate stalls / times out | Ollama on QNAS can't reach cloud (macvlan = no internet) | Use host-net Ollama on i9 (10.0.2.230) |
| OSD not updating | `osd_info` not patched or `grabbed != 1` | Use Main_MiSTer v4 patch in `src/mister-tasbot/` |
| Wrong character translated | Glossary missing entry | Add to `glossary` in game profile |
| Daemon won't start | Port 8182 already bound (Wizzo) | Different port for translate daemon |

## Related

- `mister-ops` — device access, FIFO, memory map
- `mister-cores` — per-core DDR3 layouts
- `mister-game-profile` — profile JSON schema
- `ollama-deploy` — local model serving
- `translate` — generic multilingual translation via Ollama
