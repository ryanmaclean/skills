---
name: mister-ops
description: Operational knowledge for the MiSTer FPGA device — SSH access, /media/fat filesystem, /tmp/MiSTer_cmd FIFO, scaler buffer memory map, Wizzo Remote API, OSD overlay via osd_info, and hard-won gotchas (0xFF200000 = DO NOT WRITE, /dev/fb0 does not composite). Use whenever the user mentions MiSTer, scaler, save state, OSD subtitles, or translation pipeline plumbing.
trigger: mister, fpga, scaler, osd_info, wizzo, mister_cmd, /media/fat, ddr3
---

# MiSTer Ops

Physical device. One instance. Bricks are expensive. As of 2026-04-06.

## Access

```
Host mister.local
  User root
  IdentityFile ~/.ssh/mister_ed25519
  ServerAliveInterval 30
  ServerAliveCountMax 3
  ControlMaster auto
  ControlPath ~/.ssh/cm-%r@%h:%p
  ControlPersist 10m
```

Never run MiSTer SSH in a harness background shell — exits 255 on disconnect. Use
ControlMaster or `ansible mister -m shell -a "..."`. See `ssh-fleet`.

## Filesystem (`/media/fat` = SD card)

| Path | Purpose |
|------|---------|
| `/media/fat/games/<CORE>/` | ROMs per core |
| `/media/fat/config/` | Per-core configs |
| `/media/fat/Scripts/` | User scripts |
| `/media/fat/linux/` | Kernel, boot, MiSTer binary |
| `/tmp/MiSTer_cmd` | Command FIFO |
| `/dev/mem` | HPS<->FPGA mmap |
| `/dev/fb0` | Linux framebuffer — does NOT composite during gameplay |

## Memory map (`/dev/mem`)

| Addr | Meaning | Access |
|------|---------|--------|
| `0x20000000` | Scaler output, live RGB24, ~3ms latency | READ for OCR |
| `0xFF200000` | lwhps2fpga bridge | NEVER WRITE — bricks FPGA |
| DDR3 per-core save-state regions | varies — see `mister-cores` | R/W |

Capture pattern: mmap `/dev/mem @ 0x20000000`, read one frame, dump JPEG, feed OCR.

## `/tmp/MiSTer_cmd` FIFO

```bash
echo "load_rom SNES /media/fat/games/SNES/ff6.sfc" > /tmp/MiSTer_cmd
echo "reset"      > /tmp/MiSTer_cmd
echo "screenshot" > /tmp/MiSTer_cmd
echo "menu"       > /tmp/MiSTer_cmd
```

Blocking reader — if no reader attached, writes block. Don't re-read after write.

## Wizzo Remote API

Base: `http://mister.local:8182`

| Method | Path | Body | Purpose |
|--------|------|------|---------|
| GET | `/api/games/index` | — | List all games |
| POST | `/api/games/launch` | `{"path":"/media/fat/games/..."}` | Launch ROM |
| POST | `/api/controls/menu` | — | Trigger OSD menu |

## OSD overlay (`osd_info`)

Only confirmed way to composite text over live gameplay on HDMI. Requires
Main_MiSTer v4 patched with `osd_info` + forced `grabbed=1`. Driver: `osd_subs.py`
reads stdin/file → one OSD line at a time.

Source: `/Users/studio/rust-town/mister-tasbot/` (patched binary + py).

## Common mistakes

| Mistake | Fix |
|---------|-----|
| Writing to `0xFF200000` | DON'T. Bricks FPGA. |
| `/dev/fb0` overlay not visible during gameplay | Use `osd_info` via `osd_subs.py` |
| SSH in background bash exits 255 | Use ControlMaster or ansible |
| Scaler buffer returns zeros | Core not running or wrong offset — confirm `0x20000000` |
| FIFO writes hang | Reader not attached — don't re-read after write |
| OSD line not updating | `grabbed` not forced to 1 in patched binary |
| D-pad not mapping | Use HAT axis, not buttons (fixed in v4 patch) |

## Verified cores (6)

NES, SNES, Genesis, TG16, GBA, GB. See `mister-cores` for DDR3 layouts.

Best demos: DQ VI (J) SNES, CT JP SNES.

## Related

- `mister-translation` — full pipeline workflow
- `mister-cores` — per-core DDR3 layouts
- `mister-game-profile` — profile JSON schema
- `mister-forum` — forum browsing (separate concern)
