---
name: apfs
description: >
  APFS space efficiency techniques for macOS developers. Use when copying files,
  cloning card bundles, creating build sandboxes, or writing Rust code that copies
  directories on macOS. Trigger phrases: "copy-on-write", "clonefile", "cp -c",
  "COW clone", "APFS", "ditto vs cp", "sparse file", "bop new card bundle".
  Covers clonefile() syscall, cp -c, ditto, sparse files, disk images, and snapshots.
---

# APFS Space Efficiency

## The core rule

`cp -r` on APFS wastes space. Use `cp -c` or `clonefile()` for COW clones.

---

## Quick reference

| Operation | Command | COW? | Notes |
|-----------|---------|------|-------|
| Copy dir, COW | `cp -cR src/ dst/` | Yes | APFS only; falls back to copy on HFS+ |
| Copy with xattrs/resource forks | `ditto src/ dst/` | No | Preserves xattrs, ACLs, resource forks; no COW |
| Copy + xattrs + COW | Not possible in one tool | — | Use ditto for xattr needs; accept copy cost |
| Sparse file | `truncate -s 10G file` | N/A | Allocates no blocks until written |
| Disk image sandbox | `hdiutil create -type SPARSEBUNDLE -size 20g -fs "Case-sensitive APFS" img` | N/A | Isolated APFS-in-APFS for Linux-compatible builds |

---

## When COW breaks

`cp -c` silently falls back to a full copy when:
- Source and destination are on different volumes
- Destination filesystem is HFS+, exFAT, or non-APFS
- File size exceeds APFS clone limit (~8 EB in theory, no practical limit)

Detect APFS: `diskutil info / | grep "File System Personality"` → must show `APFS`.

---

## clonefile() in Rust

The `cp -c` flag calls `clonefile(2)` internally. Call it directly from Rust for card bundle cloning:

```rust
use std::ffi::CString;

fn clone_file(src: &std::path::Path, dst: &std::path::Path) -> std::io::Result<()> {
    let src = CString::new(src.to_str().unwrap()).unwrap();
    let dst = CString::new(dst.to_str().unwrap()).unwrap();
    let ret = unsafe { libc::clonefile(src.as_ptr(), dst.as_ptr(), 0) };
    if ret == 0 {
        Ok(())
    } else {
        // ENOTSUP (45) = not APFS; fall back to std::fs::copy
        Err(std::io::Error::last_os_error())
    }
}
```

Add to `Cargo.toml`: `libc = "0.2"`

**ENOTSUP (errno 45)** = destination is not APFS. Always fall back to `std::fs::copy`.
Available: macOS 10.12+. `clonefile` does NOT preserve resource forks — use `copyfile(3)` if those matter.

For directories: `clonefile` clones the entire tree atomically in one syscall. Faster than recursive `cp -c`.

---

## ditto — when to use it over cp

Use `ditto` when you need to preserve:
- Extended attributes (xattrs): `com.apple.quarantine`, custom metadata
- Resource forks (legacy but still present in some app bundles)
- ACLs
- HFS+ compression metadata

`ditto` does NOT use COW. Space cost = full copy. Trade-off is explicit.

```sh
ditto src/ dst/          # preserves xattrs, resource forks, ACLs
ditto --norsrc src/ dst/ # skip resource forks (faster, Linux-compatible output)
ditto -x archive.cpio dst/ # extract cpio archive with APFS awareness
```

---

## Sparse files

APFS sparse files allocate no blocks until written. Good for pre-sized log files, databases, disk images.

```sh
truncate -s 10G logfile     # 0 bytes allocated, reports 10G size
dd if=/dev/zero bs=1 count=0 seek=10G of=logfile  # same
```

Read true allocation: `du -sh logfile` shows allocated blocks; `ls -lsk logfile` shows apparent size.
`du --apparent-size` is not available on macOS — use `ls -l` for apparent, `du` for real.

---

## Snapshots (build rollback / test isolation)

```sh
# Create
tmutil snapshot
diskutil apfs addSnapshot disk3s1 -name "pre-migration"

# List
diskutil apfs listSnapshots disk3s1

# Delete
tmutil deleteLocalSnapshots /
diskutil apfs deleteSnapshot disk3s1 -name "pre-migration"
```

Snapshots are differential — they consume space only for changed blocks. Safe to create before risky migrations.

---

## Space audit commands

```sh
diskutil info /              # true free space (more accurate than df)
du -sh target/               # allocated space (ignores sparse holes)
ls -lsk file                 # blocks (col 2) vs apparent size (col 5)
diskutil apfs listSnapshots disk1s5
ioreg -rc AppleAPFSContainer # volume space breakdown
```

---

## bop card bundle cloning

For `bop new` — prefer `clonefile()` over `cp -c`:
- `clonefile` clones a directory tree in one syscall (atomic, no shell overhead)
- Fall back to `std::fs::copy` recursively on ENOTSUP
- Do NOT use `ditto` for card bundles — no xattrs needed, and no COW is wasteful

Pattern:
```rust
match clone_dir(&template, &dest) {
    Ok(()) => {}  // APFS COW, near-zero space
    Err(e) if e.raw_os_error() == Some(45) => copy_dir_recursive(&template, &dest)?,
    Err(e) => return Err(e),
}
```
