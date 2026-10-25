---
name: netbsd-pkg
description: NetBSD package management and Rust build setup on QEMU aarch64 VMs. Use for pkg_add/pkgin on NetBSD ARM64, installing Rust toolchains, or preparing NetBSD build VMs.
version: 1.0.0
triggers:
  - netbsd
  - pkg_add
  - pkgin
  - pkgsrc
  - netbsd vm
  - netbsd package
  - netbsd build
  - bsd package
  - qemu netbsd
---

# NetBSD Package Management on QEMU VMs

Reference skill for installing packages, resolving dependencies, and building
Rust projects on NetBSD 10.x aarch64 QEMU VMs.

For concrete VM host coordinates (jump hosts, ports, Ansible groups) for a
specific deployment, see `## Deployment-specific details` at the bottom of
this file.

---

## VM Access

VMs are accessed via ProxyJump through the Mac Mini host. General pattern:

```bash
# SSH into VM (substitute your jump-host user@host and port)
ssh -J <user>@<vm-host> -p <port> root@localhost

# One-liner for remote commands:
ssh <user>@<vm-host> "ssh -p <port> root@localhost 'COMMAND'"
```

---

## Pre-flight: Disable IPv6

QEMU user-mode NAT only supports IPv4. IPv6 causes pkg_add and curl to hang
for minutes before falling back. **Always disable IPv6 first.**

Persistent (survives reboot):

```bash
# Add to /etc/sysctl.conf on the VM:
net.inet6.ip6.accept_rtadv=0
```

Immediate (runtime):

```bash
sysctl -w net.inet6.ip6.accept_rtadv=0
```

Do this on every fresh VM before any package operations.

---

## CDN URL Format

The pkgsrc binary package CDN follows this structure:

```
https://cdn.netbsd.org/pub/pkgsrc/packages/NetBSD/{ARCH}/{VERSION}_{QUARTER}/All/
```

For the fleet's aarch64 VMs running NetBSD 10.0:

```
https://cdn.netbsd.org/pub/pkgsrc/packages/NetBSD/aarch64/10.0_2025Q4/All/
```

Adjust the quarter (`2025Q1`, `2025Q2`, etc.) as new releases appear. To check
what quarters are available, browse:

```
https://cdn.netbsd.org/pub/pkgsrc/packages/NetBSD/aarch64/
```

Package filenames follow the pattern: `PKGNAME-VERSION.tgz`
(e.g., `cmake-3.28.3.tgz`, `perl-5.38.2nb1.tgz`).

---

## Method 1: pkgin (Preferred)

pkgin is the apt/yum-like wrapper for pkgsrc. Use this when the VM has
reasonable network access.

### Initial setup

```bash
# Set PKG_PATH so pkg_add can resolve dependencies for bootstrapping
export PKG_PATH="https://cdn.netbsd.org/pub/pkgsrc/packages/NetBSD/aarch64/10.0_2025Q4/All"

# Install pkgin itself
pkg_add pkgin

# Configure the repository
echo "https://cdn.netbsd.org/pub/pkgsrc/packages/NetBSD/aarch64/10.0_2025Q4/All" > /usr/pkg/etc/pkgin/repositories.conf

# Update the package database
pkgin update
```

### Common operations

```bash
pkgin search cmake          # Search for packages
pkgin install cmake         # Install (auto-resolves deps)
pkgin upgrade               # Upgrade all installed packages
pkgin list                  # List installed packages
pkgin avail                 # List all available packages
pkgin avail | grep rust     # Find packages by keyword
pkgin remove cmake          # Remove a package
pkgin show-deps cmake       # Show what cmake depends on
pkgin show-full-deps cmake  # Show full dependency tree
```

### Bulk install for Rust builds

```bash
pkgin install perl cmake pkg-config openssl curl git-base
```

---

## Method 2: pkg_add with PKG_PATH

When pkgin is not yet installed, or for one-off installs.

```bash
export PKG_PATH="https://cdn.netbsd.org/pub/pkgsrc/packages/NetBSD/aarch64/10.0_2025Q4/All"
pkg_add cmake perl pkg-config openssl curl git-base
```

PKG_PATH is **critical** for automatic dependency resolution. Without it,
pkg_add will fail on every missing dependency and you must install each one
manually in topological order.

To make PKG_PATH persistent:

```bash
echo 'export PKG_PATH="https://cdn.netbsd.org/pub/pkgsrc/packages/NetBSD/aarch64/10.0_2025Q4/All"' >> /etc/profile
```

---

## Method 3: Download on Host, SCP to VM (Fastest)

Downloads over the double SSH hop (MBP -> jump host -> VM) are very slow.
For large packages or many dependencies, download on the Mac first.

### Step 1: Download on Mac

```bash
PKG_BASE="https://cdn.netbsd.org/pub/pkgsrc/packages/NetBSD/aarch64/10.0_2025Q4/All"
curl -L -o /tmp/cmake.tgz "${PKG_BASE}/cmake-3.28.3.tgz"
```

To find the exact filename, browse the All/ directory or use:

```bash
curl -s "${PKG_BASE}/" | grep -oP 'href="[^"]*cmake[^"]*\.tgz"'
```

### Step 2: Transfer to VM

General pattern — SCP to jump host, then from jump host into VM:

```bash
# Replace <user>@<vm-host> and <port> with your deployment's values
# (see ## Deployment-specific details below)
scp /tmp/cmake.tgz <user>@<vm-host>:/tmp/
ssh <user>@<vm-host> "scp -P <port> -o StrictHostKeyChecking=no /tmp/cmake.tgz root@localhost:/tmp/"
```

### Step 3: Install locally in VM

```bash
# Still set PKG_PATH so transitive deps can resolve from CDN
export PKG_PATH="https://cdn.netbsd.org/pub/pkgsrc/packages/NetBSD/aarch64/10.0_2025Q4/All"
pkg_add /tmp/cmake.tgz
```

### Bulk download helper

Run on Mac to grab multiple packages at once:

```nu
#!/usr/bin/env nu
let base = "https://cdn.netbsd.org/pub/pkgsrc/packages/NetBSD/aarch64/10.0_2025Q4/All"
let pkgs = ["cmake-3.28.3.tgz" "perl-5.38.2nb1.tgz" "pkg-config-0.29.2nb2.tgz"]
$pkgs | each { |p|
    print $"Downloading ($p)..."
    ^curl -L -o $"/tmp/($p)" $"($base)/($p)"
}
```

---

## Rust Build Environment

### Required packages

| Package | Why |
|---------|-----|
| perl | Required by openssl-sys build script |
| cmake | Required by aws-lc-sys, ring, and other -sys crates |
| pkg-config | Locates system libraries for -sys crates |
| openssl | TLS headers and libs for native-tls / openssl crate |
| curl | Cargo needs it for crate downloads (if not statically linked) |
| git-base | For cargo git dependencies |

### Environment variables

Set these in the shell before running `cargo build`, or add to `/etc/profile`:

```bash
export PATH=/usr/pkg/bin:/usr/pkg/sbin:/usr/sbin:/usr/bin:/bin:$PATH
export PKG_CONFIG_PATH=/usr/pkg/lib/pkgconfig
export OPENSSL_DIR=/usr/pkg
export OPENSSL_LIB_DIR=/usr/pkg/lib
export OPENSSL_INCLUDE_DIR=/usr/pkg/include
```

### Persistent profile setup

```bash
cat >> /etc/profile << 'PROFILE'
export PATH=/usr/pkg/bin:/usr/pkg/sbin:$PATH
export PKG_CONFIG_PATH=/usr/pkg/lib/pkgconfig
export OPENSSL_DIR=/usr/pkg
export OPENSSL_LIB_DIR=/usr/pkg/lib
export OPENSSL_INCLUDE_DIR=/usr/pkg/include
export PKG_PATH="https://cdn.netbsd.org/pub/pkgsrc/packages/NetBSD/aarch64/10.0_2025Q4/All"
PROFILE
```

---

## Known Patches for Rust Crates on NetBSD

### mio 0.8.x / 1.0.x: kqueue udata type mismatch

NetBSD 10.x defines `kevent.udata` as `intptr_t`, but mio expects `*mut c_void`.
The build fails with a type error in the kqueue selector.

**Fix:** In the mio source (usually in cargo registry or via `[patch]`), edit
`src/sys/unix/selector/kqueue.rs` and change the `UData` type alias:

```rust
// Before (broken on NetBSD):
type UData = intptr_t;

// After (correct):
type UData = *mut libc::c_void;
```

Or use a `[patch.crates-io]` section in `Cargo.toml` pointing to a fixed fork.

### interprocess 2.2.x: missing NetBSD target_os cfg

The interprocess crate gates Unix socket code behind specific OS cfgs but
omits NetBSD.

**Fix:** Add `target_os = "netbsd"` alongside every `target_os = "freebsd"`
gate in the interprocess source. The relevant files are typically in
`src/os/unix/` and `src/local_socket/`.

---

## Troubleshooting

### pkg_add hangs indefinitely

**Cause:** IPv6. The VM tries AAAA records first, which time out under QEMU
user-mode NAT.

**Fix:** `sysctl -w net.inet6.ip6.accept_rtadv=0` (see Pre-flight above).

### pkg_add: Can't process <URL>: Not Found

**Cause:** Wrong CDN path or package name. The exact filename includes a
version and sometimes a `nbN` suffix.

**Fix:** Browse the All/ directory to find the exact name:

```bash
# On Mac or any machine with fast internet:
curl -s "https://cdn.netbsd.org/pub/pkgsrc/packages/NetBSD/aarch64/10.0_2025Q4/All/" | grep -i PKGNAME
```

### pkg_add: no PKG_PATH set, can't resolve dependencies

**Cause:** PKG_PATH environment variable not exported.

**Fix:** `export PKG_PATH="https://cdn.netbsd.org/pub/pkgsrc/packages/NetBSD/aarch64/10.0_2025Q4/All"`

### Download speed is extremely slow in VM

**Cause:** Double SSH hop (MBP -> jump host -> QEMU NAT -> internet).

**Fix:** Use Method 3 above -- download on the Mac and SCP into the VM.

### cargo build fails: openssl headers not found

**Cause:** OPENSSL_DIR and related vars not set.

**Fix:** Export the Rust build environment variables listed above.

### cargo build fails: cmake not found

**Cause:** /usr/pkg/bin not in PATH, or cmake not installed.

**Fix:** `pkgin install cmake` and ensure PATH includes `/usr/pkg/bin`.

---

## Quick Reference

```bash
# === First-time VM setup (run once) ===
sysctl -w net.inet6.ip6.accept_rtadv=0
echo "net.inet6.ip6.accept_rtadv=0" >> /etc/sysctl.conf
export PKG_PATH="https://cdn.netbsd.org/pub/pkgsrc/packages/NetBSD/aarch64/10.0_2025Q4/All"
pkg_add pkgin
echo "$PKG_PATH" > /usr/pkg/etc/pkgin/repositories.conf
pkgin update

# === Install Rust build deps ===
pkgin install perl cmake pkg-config openssl curl git-base

# === Set up build environment ===
export PATH=/usr/pkg/bin:/usr/pkg/sbin:$PATH
export PKG_CONFIG_PATH=/usr/pkg/lib/pkgconfig
export OPENSSL_DIR=/usr/pkg
export OPENSSL_LIB_DIR=/usr/pkg/lib
export OPENSSL_INCLUDE_DIR=/usr/pkg/include

# === Build ===
cargo build --release
```

## Deployment-specific details

This skill is generic NetBSD package management and Rust build knowledge.
Everything above applies to any NetBSD 10.x aarch64 QEMU VM. Concrete VM
host coordinates, jump-host IPs, port assignments, and Ansible groups for a
specific deployment live in that deployment's fleet-ops docs.

**For the rust-town fleet:**
- `/Users/studio/rust-town/fleet-ops/docs/build-vm.md` — VM inventory,
  ProxyJump SSH patterns, port table (nb-vm-24 on minim4-24 port 2222,
  nb-vm-16 on minim4-16 port 2224), Ansible `[netbsd_vms]` group
- `/Users/studio/rust-town/fleet-ops/ansible/inventory` — source of truth
  for jump-host IPs (minim4-24, minim4-16)

**For a different deployment:** substitute your own jump-host credentials
and VM port in the SSH/SCP patterns above, and document them in your own
`build-vm.md` equivalent.
