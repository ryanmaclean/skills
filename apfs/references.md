# APFS Reference — Extended Detail

Load this file when you need deeper coverage beyond the SKILL.md quick reference.

---

## NSFileManager and copyfile(3) in Swift/Obj-C

`NSFileManager.copyItem(at:to:)` calls `copyfile(3)` internally on APFS, which
does use `clonefile` when source and destination are on the same APFS volume.
So Swift apps get COW for free via the standard API — no need to call `clonefile` directly
unless you need the explicit ENOTSUP fallback or directory-level atomicity.

```swift
try FileManager.default.copyItem(at: source, to: destination)
// On same APFS volume: COW clone. Cross-volume: full copy. Transparent.
```

URLResourceKey space queries:
```swift
// Allocated blocks on disk (what du shows):
let allocated = try url.resourceValues(forKeys: [.fileAllocatedSizeKey]).fileAllocatedSize

// Apparent size (what ls -l shows, ignores sparse holes):
let apparent = try url.resourceValues(forKeys: [.fileSizeKey]).fileSize

// Total allocated including resource fork:
let total = try url.resourceValues(forKeys: [.totalFileAllocatedSizeKey]).totalFileAllocatedSize
```

---

## asr — Apple Software Restore

Block-level volume clone, COW-aware. Use for full-volume duplication (e.g. test environments).

```sh
# Clone a volume — DESTRUCTIVE on target
sudo asr restore \
  --source /Volumes/SourceVolume \
  --target /Volumes/TargetVolume \
  --erase \
  --noprompt

# Restore from disk image
sudo asr restore --source my-image.dmg --target /Volumes/Target --erase
```

Use over `ditto` when: cloning an entire APFS volume, restoring from a disk image,
or when you need block-level fidelity including HFS+ metadata.

---

## Disk images as isolated build sandboxes

Case-sensitive APFS image — for Linux-compatible builds on macOS:

```sh
# Create 20GB case-sensitive APFS sparse bundle (grows on demand)
hdiutil create \
  -type SPARSEBUNDLE \
  -size 20g \
  -fs "Case-sensitive APFS" \
  -volname "build-sandbox" \
  build-sandbox.sparsebundle

# Mount
hdiutil attach build-sandbox.sparsebundle

# Compact (reclaim space from deleted files)
hdiutil compact build-sandbox.sparsebundle
```

`.sparsebundle` = directory of band files (better for Time Machine, incremental backup)
`.sparseimage` = single file (simpler, better for rsync)

`hdiutil compact` only reclaims space that APFS has freed — it won't help if the
guest filesystem still marks blocks as used. Run `fstrim` equivalent inside first if possible.

---

## Hardlinks vs clones on APFS

| Aspect | Hardlink (`ln`) | Clone (`cp -c` / `clonefile`) |
|--------|-----------------|-------------------------------|
| Space | Shared until deletion | Shared until mutation (COW) |
| Inode | Same inode | Different inodes |
| Directory hardlinks | Forbidden | Supported (via `clonefile`) |
| Mutation isolation | No — writes visible to all | Yes — writes create private copy |
| Detection | `stat -f %l` > 1 | No standard detection API |
| Use case | Immutable shared data | Template systems, card bundles |

Detect hardlink count: `stat -f %l file`
Detect if file is a clone: No direct API. Heuristic: same volume, different inode, same content hash.

---

## APFS snapshot limits and space behavior

- Max snapshots per APFS volume: 10,000 (soft limit; practical limit is disk pressure)
- Space: snapshots track changed extents only — first snapshot is nearly free
- Time Machine uses snapshots internally; manual `tmutil snapshot` creates an hourly-style snapshot
- Space reclamation: deleting a snapshot doesn't immediately free space if blocks are still live

```sh
# Create named snapshot
diskutil apfs addSnapshot disk3s1 -name "pre-migration-2026-03-07"

# List
diskutil apfs listSnapshots disk3s1

# Delete
diskutil apfs deleteSnapshot disk3s1 -name "pre-migration-2026-03-07"

# Or via tmutil (deletes all local snapshots for a date)
tmutil deleteLocalSnapshots 2026-03-07
```

---

## fs_usage — tracing file operations

```sh
# Trace all file operations for a process
sudo fs_usage -f filesys -w <pid>

# Trace clonefile calls specifically
sudo fs_usage -f filesys | grep -i clone
```

Use this to verify `cp -c` is actually issuing `clonefile` vs falling back to `copyfile`.

---

## Apple documentation references

- **APFS Programming Guide** (no longer publicly maintained; was in ADC)
- **TN3128** — "Reading and writing files" — covers COW semantics in NSFileManager
- **WWDC 2017 Session 715** — "What's New in File Systems" — introduced APFS to developers
- **WWDC 2019 Session 709** — "Optimizing Storage in Your App"
- `man 2 clonefile` — authoritative syscall documentation
- `man 3 copyfile` — higher-level API with COPYFILE_CLONE flag

---

## jj workspace and COW

`jj workspace add` creates a new working copy linked to the same repo store.
It does NOT use `clonefile` for the working copy — it creates a fresh checkout.
`target/` directories in each jj workspace are separate and NOT deduplicated automatically.

To share build artifacts across jj workspaces: use a symlink to a shared `target/`
or configure cargo's `target-dir` in `.cargo/config.toml`:

```toml
[build]
target-dir = "$HOME/triz/.cargo-target"
```

This is a single shared target across all workspaces — trades isolation for disk savings.
