# durable-tid canonical record encoding v0

Status: frozen for v0. Any field-order, width, or semantic change requires a
new `schema_version` and a v1 document; v0 fixtures must keep validating
against v0 forever.

This is the one canonical persistent/wire encoding for a durable-tid record,
shared by the JS reference model (`model-js/`), any future native reference
code, formal models, RTL fixtures, FPGA work, and future ASIC work. No
compiler-native struct layout and no unordered JSON are canonical — this
document is canonical, and JSON fixtures under `fixtures/` are a
human-readable *projection* of the bytes, not the format itself.

## Design constraints carried over from DESIGN.md

- fixed integer widths, explicit endianness, fixed field order
- no dependence on wall-clock time for correctness
- append-only facts; a record is never rewritten, only superseded
- transport-independent (this is the payload that rides inside a function
  call, an mmap queue slot, an ioctl, VirtIO, or PCIe/FPGA DMA — not the
  transport itself)

## Byte order

**All multi-byte integers are little-endian.** `request_id` and
`content_hash` are raw opaque byte strings (not integers) and have no
internal byte-order interpretation — they are copied byte-for-byte.

## Fixed header layout (v0)

Total fixed header size: **104 bytes**, followed immediately by
`payload_length` bytes of payload with **no trailing padding**.

| Offset | Size | Field           | Type        | Notes |
|-------:|-----:|-----------------|-------------|-------|
| 0      | 1    | `schema_version`| `uint8`     | `0` for this document |
| 1      | 7    | `reserved0`     | `uint8[7]`  | MUST be zero on write; readers MUST reject non-zero |
| 8      | 8    | `epoch`         | `uint64 LE` | monotonic epoch counter; see `replay_old_epoch` guard |
| 16     | 16   | `request_id`    | `uint8[16]` | opaque 128-bit caller-supplied id, used for idempotent dedup |
| 32     | 8    | `object_id`     | `uint64 LE` | identity of the object/stream being mutated |
| 40     | 8    | `tid`           | `uint64 LE` | `0` = `TID_UNASSIGNED` (record not yet committed); committed TIDs start at `1` and are strictly monotonic per `object_id` |
| 48     | 8    | `parent_tid`    | `uint64 LE` | `0` = `NO_PARENT` (genesis record for this `object_id`) |
| 56     | 1    | `operation`     | `uint8` enum | see Operation enum below |
| 57     | 1    | `status`        | `uint8` enum | see Status enum below |
| 58     | 6    | `reserved1`     | `uint8[6]`  | MUST be zero on write; readers MUST reject non-zero |
| 64     | 32   | `content_hash`  | `uint8[32]` | SHA-256 (see Hashing below); all-zero = `HASH_ABSENT` |
| 96     | 4    | `payload_length`| `uint32 LE` | byte length of the payload that follows the header |
| 100    | 4    | `reserved2`     | `uint8[4]`  | MUST be zero on write; readers MUST reject non-zero |
| 104    | `payload_length` | `payload` | bytes | opaque application payload, or a payload *reference* (e.g. a content-addressed pointer) — this document does not constrain payload contents |

Reserved fields exist purely to keep 8-byte alignment for every subsequent
field (`epoch`, `object_id`, `tid`, `parent_tid` all start on 8-byte
boundaries; `content_hash` and `payload_length` start on 8-byte boundaries
too). Alignment is a debuggability/inspection convenience for hex dumps and
RTL/FPGA register mapping — it is not required for correctness, but a v0
reader MUST still reject a record whose reserved bytes are non-zero, so that
reserved space stays available for a future version without silently
changing v0 semantics.

## Operation enum (`operation`, uint8)

| Value | Name |
|------:|------|
| 0 | `OP_UNSPECIFIED` (invalid; readers MUST reject) |
| 1 | `submit` |
| 2 | `duplicate_submit` |
| 3 | `device_complete` |
| 4 | `durable_ack` |
| 5 | `publish_completion` |
| 6 | `reset` |
| 7 | `recover` |
| 8 | `replay_old_epoch` |

Values `9`-`255` are reserved for future versions and MUST be rejected by a
v0 reader.

## Status enum (`status`, uint8)

Mirrors the semantic states in the JS model 1:1.

| Value | Name |
|------:|------|
| 0 | `PROPOSED` |
| 1 | `COHERENT_VISIBLE` |
| 2 | `DEVICE_COMPLETE` |
| 3 | `PERSISTENT` |
| 4 | `TRUSTED_COMPLETE` |

Values `5`-`255` are reserved and MUST be rejected by a v0 reader.

## Optional / default representation

There are no optional fields with variable presence in v0 — every field is
always present at its fixed offset. "Absence" is represented by a defined
sentinel value instead of a variable-length/optional encoding, so the header
size never varies:

- `tid = 0` means `TID_UNASSIGNED` (no committed TID yet).
- `parent_tid = 0` means `NO_PARENT` (genesis for this `object_id`).
- `content_hash = 0x00 * 32` means `HASH_ABSENT` (no hash computed/carried
  for this record — e.g. a `PROPOSED` record before content is fixed).
- `payload_length = 0` means an empty payload; the payload region is absent
  (zero bytes follow the header), not padded.

A real committed TID or a real content hash is never allowed to be the
all-zero sentinel; a generator that produces a colliding value MUST reroll
or fail rather than emit a record that could be misread as "absent".

## Hashing and domain separation

`content_hash` (when not `HASH_ABSENT`) is:

```
content_hash = SHA-256(domain_prefix || canonical_header_zeroed || payload)
```

where:

- `domain_prefix` is the fixed 24-byte ASCII string `"durable-tid/record/v0"`
  (21 visible bytes) right-padded with `\x00` to exactly 24 bytes, kept
  fixed-width so the prefix itself cannot be format-confused with a v1
  prefix of different length. This prevents a durable-tid v0 record hash
  from colliding with a hash computed over the same bytes by an unrelated
  protocol, or by a future durable-tid version with a different layout.
- `canonical_header_zeroed` is the 104-byte fixed header from this document
  with the `content_hash` field (offset 64, 32 bytes) replaced by zero
  bytes, and all reserved fields already zero per the write rule above.
  This lets a writer compute the hash before it knows the hash, and lets a
  verifier recompute it by zeroing the same field.
- `payload` is the exact `payload_length` payload bytes, unmodified.

A verifier recomputes `content_hash` this way and compares constant-time;
mismatch means the record MUST be treated as corrupt, never as "absent".

## Canonical test vectors

Three golden fixtures are checked in under `../fixtures/`, each as a pair of
files: `<name>.hex` (the exact wire bytes, hex-encoded, one line, no
separators) and `<name>.json` (the same record as a structured/annotated
projection, including the expected `content_hash` restated in hex for easy
diffing). The JS reference model's fixture tests load the `.hex` file, parse
it per this document, and assert the parsed fields and recomputed hash match
the `.json` projection exactly.

1. `genesis-submit` — first record for a fresh `object_id`: `tid =
   TID_UNASSIGNED (0)`, `parent_tid = NO_PARENT (0)`, `operation = submit`,
   `status = PROPOSED`, `content_hash = HASH_ABSENT`, empty payload.
2. `committed-persistent` — a later record for the same `object_id` after
   commit: `tid = 1`, `parent_tid = 0`, `operation = durable_ack`, `status =
   PERSISTENT`, a real 4-byte payload (`0xDEADBEEF`), and its real
   `content_hash` computed per the formula above.
3. `duplicate-submit-rejected-shape` — a `duplicate_submit` record carrying
   the *same* `request_id` as `genesis-submit`, `tid = TID_UNASSIGNED (0)`
   (a duplicate never gets its own TID — see the model invariant "duplicate
   request commits at most once"), used by the JS model's duplicate-submit
   test to assert the second submission does not advance state.

Fixture generation (`../fixtures/generate.sh`) is a plain POSIX `/bin/sh` +
`openssl`/`shasum` script with no build step, so the fixtures can be
regenerated and diffed without a language toolchain; it is not itself part
of the canonical format, only a convenience for keeping the checked-in
fixtures reproducible.

## Non-goals for v0

- no variable-length header (fields never move or become optional)
- no compression
- no multi-record framing/batching (one record = one fixed header + one
  payload region; batching is a transport concern layered above this)
- no big-endian variant
