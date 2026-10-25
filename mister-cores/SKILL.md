---
name: mister-cores
description: Per-core DDR3 memory layouts for MiSTer save-state reading — NES nametable, SNES WRAM, Genesis VDP, TG16, GBA, GB offsets. Use when reading game state directly from /dev/mem instead of (or alongside) the scaler buffer.
trigger: ddr3, save state, wram, nametable, vdp, mister memory, core offset
---

# MiSTer Per-Core Memory Layouts

As of 2026-04-06. Addresses are HPS-side via `/dev/mem` mmap. For access pattern
see `mister-ops` (scaler is at `0x20000000`, NEVER write `0xFF200000`).

## Six verified cores

| Core | Base | Region | Offset | Size | Notes |
|------|------|--------|--------|------|-------|
| NES | DDR3 | Nametable | `0x00102400` | 2 KB | PPU VRAM, 32x30 tile grid |
| NES | DDR3 | CPU RAM | `0x00100000` | 2 KB | Internal RAM, mirrored |
| SNES | DDR3 | WRAM | `0x3F800000 + 0x08` | 128 KB | Main CPU RAM. Offset 0x08 past header |
| SNES | DDR3 | VRAM | `0x3F820000` | 64 KB | Tile + BG data |
| Genesis | DDR3 | VDP VRAM | `0x3F840000` | 64 KB | Confirm per-ROM |
| Genesis | DDR3 | 68K RAM | `0x3F860000` | 64 KB | Main CPU RAM |
| TG16 | DDR3 | Work RAM | `0x3F880000` | 8 KB | PC Engine RAM |
| GBA | DDR3 | EWRAM | `0x3F8A0000` | 256 KB | External Work RAM |
| GBA | DDR3 | IWRAM | `0x3F8E0000` | 32 KB | Internal Work RAM |
| GB | DDR3 | Work RAM | `0x3F8F0000` | 8 KB | GB/GBC RAM |

Values above are the known-good snapshots from this project. Per-core layouts
can shift between core revisions — re-verify after `update_all` sync.

## Access pattern

```python
import mmap, os
fd = os.open("/dev/mem", os.O_RDWR | os.O_SYNC)
mem = mmap.mmap(fd, 0x1000, offset=0x3F800008)  # SNES WRAM
wram = mem[:0x20000]
```

Requires root. Offset must be page-aligned (4 KB) — adjust offset and mask.

## Where offsets come from

Source: `Main_MiSTer` core integration headers, typically in
`support/<core>/*.h`. See `/Users/studio/rust-town/mister-tasbot/`
for the patched binary and scanned offsets.

## Common mistakes

| Mistake | Fix |
|---------|-----|
| Writing anywhere near `0xFF200000` | NEVER. Bricks FPGA. See `mister-ops` |
| Reading before core boots | Returns zeros — confirm core loaded via Wizzo API |
| Wrong endianness | SNES/Genesis vary — check per-core notes |
| Non-page-aligned mmap offset | Round down to 4 KB boundary, mask within buffer |
| Offset drifts after core update | Re-scan via `Main_MiSTer` headers |

## Related

- `mister-ops` — device access, `/dev/mem` mmap pattern
- `mister-translation` — the pipeline that consumes this data
- `mister-game-profile` — hotspot coords (scaler buffer side)
