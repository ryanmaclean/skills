---
name: zig-cc
description: >
  Use zig cc for cross-compiling C to NetBSD/Linux ARM64 from macOS.
  Trigger: cross-compile, zig cc, netbsd build, janet build, ii build, C compile ARM64, aarch64-netbsd
---

# zig cc — Cross-Compile C for BSD/Linux ARM64

`zig cc` knows targets that GCC doesn't: `aarch64-netbsd`, `aarch64-linux-gnu`, etc.
No sysroot needed. Knows NEON SIMD.

## Quick Reference

```bash
# Compile C for NetBSD ARM64 (from Mac)
zig cc -target aarch64-netbsd -O2 -o output source.c

# Compile C for Linux ARM64
zig cc -target aarch64-linux-gnu -O2 -o output source.c

# Build Janet for NetBSD from Mac
cd /tmp/janet && zig cc -target aarch64-netbsd -O2 -o janet src/*.c -Isrc/include

# Build ii (suckless IRC) for NetBSD
cd /tmp/ii && zig cc -target aarch64-netbsd -O2 -o ii ii.c -DVERSION=\"1.9\"

# Use as Rust cross-linker (cargo-zigbuild does this automatically)
cargo zigbuild --release --target aarch64-unknown-linux-gnu
```

## Why Not GCC
GCC needs a full cross-compilation toolchain + target sysroot.
zig cc has every target built in. One binary, all platforms.

## Available Targets
`zig targets | python3 -c "import sys,json;[print(t) for t in json.load(sys.stdin)['native']['triple']]"`

Key ones for us: aarch64-netbsd, aarch64-linux-gnu, x86_64-linux-gnu, aarch64-macos
