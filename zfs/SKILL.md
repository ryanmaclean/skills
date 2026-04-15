---
name: zfs
description: >
  ZFS (OpenZFS) space efficiency and snapshotting for FreeBSD and Linux developers.
  Use when cloning datasets, creating build sandboxes, replicating storage, or writing
  Rust/C code that works with ZFS pools on macOS, FreeBSD, or Linux.
  Trigger phrases: "zfs snapshot", "zfs clone", "COW dataset", "zpool", "send/receive",
  "zfs dedup", "OpenZFS", "FreeBSD storage", "build rollback on ZFS".
  Covers: clone, snapshot, send/receive, dedup, Rust patterns, FreeBSD vs Linux differences.
---

# ZFS — OpenZFS Reference

ZFS is the COW filesystem for FreeBSD (first-class, ships by default since FreeBSD 13)
and available on Linux via OpenZFS (Ubuntu, Debian, Arch). Commands are identical across
platforms — pool naming and device paths differ.

---

## The core rule

On ZFS, the unit of COW cloning is a **dataset** (filesystem or volume), not a file.
To clone a directory tree: snapshot the dataset, then clone the snapshot.

```sh
zfs snapshot pool/dataset@tag
zfs clone pool/dataset@tag pool/dataset-clone
```

Clone is instant, zero space until divergence. This is the ZFS equivalent of `cp -c` on APFS.

---

## Quick reference

| Operation | Command | Notes |
|-----------|---------|-------|
| Create snapshot | `zfs snapshot pool/ds@name` | Instant, space = changed blocks only |
| List snapshots | `zfs list -t snapshot` | |
| Clone snapshot | `zfs clone pool/ds@snap pool/ds-clone` | COW — zero space at creation |
| Rollback | `zfs rollback pool/ds@snap` | Destroys data after snapshot |
| Destroy clone/snap | `zfs destroy pool/ds@snap` | Clone must be destroyed first |
| Send (backup) | `zfs send pool/ds@snap \| zfs receive backup/ds` | Stream to remote or file |
| Incremental send | `zfs send -i pool/ds@prev pool/ds@now \| zfs receive backup/ds` | Delta only |
| Space usage | `zfs list -o name,used,avail,refer` | |
| Enable dedup | `zfs set dedup=on pool/ds` | Block-level; RAM-intensive |
| Enable compression | `zfs set compression=lz4 pool/ds` | Almost always worth enabling |

---

## Snapshots for build isolation

```sh
# Before a risky migration
zfs snapshot pool/src@pre-migration-2026-03-07

# Something went wrong — roll back
zfs rollback pool/src@pre-migration-2026-03-07

# Done — clean up
zfs destroy pool/src@pre-migration-2026-03-07
```

Snapshots consume space proportional to blocks changed after the snapshot, not the full dataset.

---

## Dataset clones for test environments

```sh
# Snapshot the production dataset
zfs snapshot pool/prod@test-base

# Create a clone — writable, COW, instant
zfs clone pool/prod@test-base pool/test

# Use pool/test — all writes go to clone, prod is untouched

# Destroy when done (must destroy clone before its origin snapshot)
zfs destroy pool/test
zfs destroy pool/prod@test-base
```

---

## Send/receive — replication

```sh
# Full send to remote
zfs send pool/ds@snap | ssh user@remote zfs receive backup/ds

# Incremental (only changed blocks since @prev)
zfs send -i pool/ds@prev pool/ds@now | ssh user@remote zfs receive -F backup/ds

# Send to a file
zfs send pool/ds@snap | gzip > ds-backup.gz
zfs send pool/ds@snap > ds-backup.zfs  # raw, faster to restore

# Restore
zfs receive pool/ds < ds-backup.zfs
```

---

## Compression — always enable

```sh
zfs set compression=lz4 pool/dataset   # fast, good ratio — default choice
zfs set compression=zstd pool/dataset  # better ratio, slower — good for cold data
```

LZ4 compression typically saves 30–50% on source code and logs with near-zero CPU cost.
Enable at dataset creation; only new writes are compressed.

---

## Dedup — use carefully

```sh
zfs set dedup=on pool/dataset
```

Block-level deduplication. Requires ~5GB RAM per TB of data for the dedup table.
Only worth enabling when data has high block-level duplication (VM images, backups).
NOT recommended for source code or build artifacts — too much RAM, little gain.

---

## Rust patterns

ZFS has no stable `clonefile()`-style syscall for userspace. Options:

**1. Shell out (simplest, recommended for most cases):**
```rust
use std::process::Command;

fn zfs_snapshot(dataset: &str, tag: &str) -> std::io::Result<()> {
    let status = Command::new("zfs")
        .args(["snapshot", &format!("{dataset}@{tag}")])
        .status()?;
    if status.success() { Ok(()) } else {
        Err(std::io::Error::other("zfs snapshot failed"))
    }
}

fn zfs_clone(origin: &str, tag: &str, clone: &str) -> std::io::Result<()> {
    let snap = format!("{origin}@{tag}");
    let status = Command::new("zfs")
        .args(["clone", &snap, clone])
        .status()?;
    if status.success() { Ok(()) } else {
        Err(std::io::Error::other("zfs clone failed"))
    }
}
```

**2. libzfs FFI (advanced, avoids process overhead):**
`libzfs` is available on FreeBSD and Linux but has no stable API — version-specific.
Use the `zfs-sys` crate if it exists for your OpenZFS version, otherwise shell out.

---

## FreeBSD vs Linux differences

| Aspect | FreeBSD | Linux (OpenZFS) |
|--------|---------|-----------------|
| Pool device naming | `/dev/ada0`, `/dev/da0`, `/dev/gpt/label` | `/dev/sda`, `/dev/nvme0n1` |
| Import on boot | Automatic via `/etc/rc.conf`: `zfs_enable="YES"` | `zpool import -a` in initrd or systemd unit |
| Default pool name | `zroot` (installer default) | `rpool` (Ubuntu default) |
| ARC (cache) tuning | `vfs.zfs.arc_max` in `/boot/loader.conf` | `zfs_arc_max` in `/etc/modprobe.d/zfs.conf` |
| Encrypted root | Native ZFS encryption or GELI (FreeBSD) | Native ZFS encryption or LUKS |
| `zpool status` | Identical | Identical |
| `zfs` commands | Identical | Identical |

---

## Space audit commands

```sh
zpool list                          # pool-level free/used
zfs list -o name,used,avail,refer   # dataset breakdown
zfs list -t snapshot                # all snapshots and their sizes
zpool iostat -v                     # I/O per vdev
zdb -b pool/dataset                 # block-level stats (verbose)
```

---

## Pools — bare minimum setup

```sh
# Single disk pool (development only — no redundancy)
zpool create mypool /dev/ada1

# Mirror (RAID-1 equivalent)
zpool create mypool mirror /dev/ada1 /dev/ada2

# Status
zpool status mypool

# Destroy (irreversible)
zpool destroy mypool
```

Always enable compression on the pool or root dataset after creation:
```sh
zfs set compression=lz4 mypool
```
