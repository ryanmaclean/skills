---
name: rust-bsd-patches
description: Patch Rust crates that fail to build on NetBSD aarch64 (ARM64)
triggers:
  - netbsd rust build
  - cargo build netbsd
  - bsd crate patch
  - kqueue udata
  - mio netbsd
  - interprocess netbsd
  - rust netbsd arm64
---

# Rust NetBSD ARM64 Crate Patches

Reference for patching Rust crates that fail on NetBSD 10.x aarch64.
Discovered building nushell, zellij, and other tools on the fleet's NetBSD VMs.

## General Patching Workflow

```bash
# 1. Try building, capture the error
cargo build --release 2>&1 | tee /tmp/build.log

# 2. For reproducibility, add a [patch] section to Cargo.toml pointing at
#    a fixed fork on your Gitea instance:
#    [patch.crates-io]
#    mio = { git = "<GITEA_FORK_URL>/studio/mio.git", tag = "v0.8.11-netbsd" }

# 3. Rebuild
cargo build --release
```

## Gitea Forks

See deployment-specific details below for the actual Gitea base URL.

| Crate | Gitea URL | Tag | Notes |
|---|---|---|---|
| mio | `<GITEA_FORK_URL>/studio/mio` | `v0.8.11-netbsd` | kqueue UData fix for NetBSD (already correct in 0.8.11 upstream) |
| zellij | `<GITEA_FORK_URL>/studio/zellij` | `v0.44.0-netbsd` | Patches mio via `[patch.crates-io]` for NetBSD builds |

## Recommended Build Environment

```bash
# System OpenSSL (avoids vendored build needing perl)
export OPENSSL_DIR=/usr/pkg
export OPENSSL_LIB_DIR=/usr/pkg/lib
export OPENSSL_INCLUDE_DIR=/usr/pkg/include

# Disable jemalloc globally when possible
cargo build --release --no-default-features --features "..."
```

---

## Crate 1: mio (0.8.x and 1.0.x) — kqueue udata type mismatch

**File:** `src/sys/unix/selector/kqueue.rs`

**Problem:** The `UData` type alias is defined as `libc::intptr_t` but NetBSD 10.x
kqueue uses `*mut libc::c_void` for the udata field. This causes a type mismatch
at compile time.

**Fix:** Change the NetBSD type alias. On NetBSD, `libc::kevent.udata` is
`libc::intptr_t` (isize), NOT `*mut libc::c_void`. Fixed upstream in mio
1.1.0+ but NOT backported to the 1.0.x or 0.8.x series. Note: mio 1.0.3
already has the correct NetBSD UData type, only 0.8.x needs patching.

```rust
// Before (line 50, under #[cfg(target_os = "netbsd")]):
type UData = *mut libc::c_void;

// After:
type UData = libc::intptr_t;
```

**Affects:** nushell, tokio-based apps, anything pulling in mio transitively.

**Status:** Fixed upstream in mio >= 1.1.0. For 0.8.x and 1.0.x, patch in place
or pin to a forked branch.

---

## Crate 2: interprocess (2.2) — missing NetBSD platform support

**Problem:** Platform detection cfg blocks only match `target_os = "freebsd"`.
NetBSD is not recognized, so the crate falls through to unsupported-platform
compile errors.

**Fix:** Add `target_os = "netbsd"` alongside `target_os = "freebsd"` in ALL
platform cfg blocks throughout the crate.

```rust
// Before:
#[cfg(target_os = "freebsd")]

// After:
#[cfg(any(target_os = "freebsd", target_os = "netbsd"))]
```

Search for every `cfg` attribute referencing `freebsd` and add `netbsd`:

```bash
grep -rn 'target_os = "freebsd"' ~/.cargo/registry/src/*/interprocess-2.2*/
```

**Affects:** zellij (client/server IPC uses interprocess for Unix sockets).

**Status:** Fixed in interprocess 2.3.1 (NetBSD support added upstream).
Zellij 0.44.0 uses 2.3.1 via semver compat with `version = "2.2"` spec.
No patching needed for interprocess >= 2.3.1.

---

## Crate 3: openssl-sys — vendored build needs perl

**Problem:** When openssl-sys builds OpenSSL from source (vendored feature), the
Configure script requires perl, which may not be installed on NetBSD.

**Fix:** Either install perl or — preferably — point at the system OpenSSL
from pkgsrc so the vendored build is skipped entirely:

```bash
export OPENSSL_DIR=/usr/pkg
export OPENSSL_LIB_DIR=/usr/pkg/lib
export OPENSSL_INCLUDE_DIR=/usr/pkg/include
```

Ensure `pkg_add openssl` has been run so headers and libs exist at those paths.

**Affects:** anything using reqwest with native-tls, cargo itself when bootstrapping.

**Status:** Not a code bug — environment configuration issue. Set the env vars
in the build shell or in `~/.cargo/config.toml`:

```toml
[env]
OPENSSL_DIR = "/usr/pkg"
OPENSSL_LIB_DIR = "/usr/pkg/lib"
OPENSSL_INCLUDE_DIR = "/usr/pkg/include"
```

---

## Crate 4: aws-lc-sys — needs cmake

**Problem:** AWS LibCrypto (aws-lc-rs) requires cmake to build its C/ASM sources.
If cmake is missing the build fails with a cryptic "cmake not found" error.

**Fix:** Install cmake from pkgsrc:

```bash
pkg_add cmake
```

Or, better: switch the dependent crate to use rustls instead of native-tls/aws-lc:

```bash
cargo build --release --no-default-features --features rustls
```

**Affects:** reqwest, hyper-tls when using the ring/aws-lc backend.

**Status:** Not a code bug. Either install the build dependency or choose a
pure-Rust TLS backend.

---

## Crate 5: jemalloc-sys — ARM64 NetBSD alignment issues

**Problem:** jemalloc has memory alignment issues on NetBSD aarch64, causing
build failures or runtime crashes.

**Fix:** Build with `--no-default-features` to disable jemalloc and fall back
to the system allocator:

```bash
cargo build --release --no-default-features
```

If the crate exposes a feature flag for the allocator, disable it explicitly.
For tools like fd and ripgrep, check their Cargo.toml for a `jemalloc` feature
and omit it.

**Affects:** fd, ripgrep, and other tools that enable jemalloc by default.

**Status:** Upstream jemalloc has limited NetBSD aarch64 testing. System
allocator is the pragmatic choice.

---

## Crate 6: procs — no NetBSD support

**Problem:** procs reads process information through `/proc` filesystem
abstractions that do not exist on NetBSD. NetBSD uses `sysctl(3)` for process
enumeration, which is a completely different API.

**Fix:** None. This would require a full NetBSD backend using sysctl, which
does not exist upstream.

**Status:** Will not build. Skip this tool on NetBSD. Use native `ps` or
`top` instead.

---

## Quick Reference Table

| Crate | Severity | Fix | Upstream? |
|---|---|---|---|
| mio 0.8.x/1.0.x | Build fails | Patch UData type alias | Fixed in 1.1.0+ |
| interprocess 2.2 | Build fails | Add netbsd to cfg blocks | Fixed in 2.3.1 |
| openssl-sys | Build fails | Set OPENSSL_DIR env vars | N/A (config) |
| aws-lc-sys | Build fails | pkg_add cmake or use rustls | N/A (config) |
| jemalloc-sys | Build fails / crash | --no-default-features | No |
| procs | Build fails | Skip on NetBSD | No |

## Cargo.toml Patch Block Template

For reproducible builds with patched forks (see deployment-specific details below for the Gitea base URL):

```toml
[patch.crates-io]
mio = { git = "<GITEA_FORK_URL>/studio/mio.git", tag = "v0.8.11-netbsd" }
```

For building Zellij on NetBSD, clone the pre-patched fork:

```bash
git clone --branch v0.44.0-netbsd <GITEA_FORK_URL>/studio/zellij.git
cd zellij
cargo build --release
```

---

## Deployment-specific details

This skill is generic NetBSD/BSD Rust patching knowledge. Everything above
applies to any Gitea-hosted fork workflow. Concrete fork URLs and Gitea
instance coordinates live in the deployment's fleet-ops repo.

**For the rust-town fleet:**
- `fleet-ops/docs/gitea-ci.md` — contains the actual `<GITEA_FORK_URL>` base,
  fork-repo table (mio, zellij), and CI runner host reference

**For a different deployment:** substitute your own Gitea base URL for
`<GITEA_FORK_URL>` in the patch block templates above, and maintain your own
fork table in your fleet-ops docs.
