#!/bin/sh
# Regenerates the canonical durable-tid v0 fixtures (RECORD-V0.md).
# POSIX sh + openssl only, no build step, no language toolchain.
# Run from this directory: ./generate.sh
set -e

DOMAIN_HEX=$(printf 'durable-tid/record/v0' | xxd -p | tr -d '\n')
DOMAIN_HEX="${DOMAIN_HEX}000000"  # right-pad the 21-byte string to 24 bytes

hash_record() {
  # $1 = 104-byte header hex with content_hash field already zeroed
  # $2 = payload hex (may be empty)
  printf '%s%s%s' "$DOMAIN_HEX" "$1" "$2" | xxd -r -p | openssl dgst -sha256 -r | awk '{print $1}'
}

le64() { # $1 decimal -> 8-byte little-endian hex
  printf '%016x' "$1" | fold -w2 | tac | tr -d '\n'
}
le32() { # $1 decimal -> 4-byte little-endian hex
  printf '%08x' "$1" | fold -w2 | tac | tr -d '\n'
}

ZERO7=00000000000000
ZERO8=0000000000000000
ZERO6=000000000000
ZERO32=0000000000000000000000000000000000000000000000000000000000000000
ZERO4=00000000

# ---------------------------------------------------------------
# genesis-submit: tid=0, parent=0, op=submit(1), status=PROPOSED(0),
# hash absent, empty payload.
REQ1=112233445566778899aabbccddeeff00
HDR1="00${ZERO7}${ZERO8}${REQ1}$(le64 1)${ZERO8}${ZERO8}0100${ZERO6}${ZERO32}${ZERO4}${ZERO4}"
PAY1=""
printf '%s%s' "$HDR1" "$PAY1" > genesis-submit.hex

cat > genesis-submit.json <<EOF
{
  "name": "genesis-submit",
  "schema_version": 0,
  "epoch": 0,
  "request_id": "${REQ1}",
  "object_id": 1,
  "tid": 0,
  "parent_tid": 0,
  "operation": "submit",
  "status": "PROPOSED",
  "content_hash": null,
  "payload_hex": "",
  "hex_file": "genesis-submit.hex"
}
EOF

# ---------------------------------------------------------------
# committed-persistent: tid=1, parent=0, op=durable_ack(4),
# status=PERSISTENT(3), payload=DEADBEEF, real content_hash.
REQ2=212233445566778899aabbccddeeff01
REQ2=$(printf '%s' "$REQ2" | tr -d ' ')
PAY2=deadbeef
HDR2_ZEROHASH="00${ZERO7}${ZERO8}${REQ2}$(le64 1)$(le64 1)${ZERO8}0403${ZERO6}${ZERO32}$(le32 4)${ZERO4}"
HASH2=$(hash_record "$HDR2_ZEROHASH" "$PAY2")
HDR2="00${ZERO7}${ZERO8}${REQ2}$(le64 1)$(le64 1)${ZERO8}0403${ZERO6}${HASH2}$(le32 4)${ZERO4}"
printf '%s%s' "$HDR2" "$PAY2" > committed-persistent.hex

cat > committed-persistent.json <<EOF
{
  "name": "committed-persistent",
  "schema_version": 0,
  "epoch": 0,
  "request_id": "${REQ2}",
  "object_id": 1,
  "tid": 1,
  "parent_tid": 0,
  "operation": "durable_ack",
  "status": "PERSISTENT",
  "content_hash": "${HASH2}",
  "payload_hex": "${PAY2}",
  "hex_file": "committed-persistent.hex"
}
EOF

# ---------------------------------------------------------------
# duplicate-submit-rejected-shape: same request_id as genesis-submit,
# op=duplicate_submit(2), tid stays 0 (never assigned to a duplicate).
HDR3="00${ZERO7}${ZERO8}${REQ1}$(le64 1)${ZERO8}${ZERO8}0200${ZERO6}${ZERO32}${ZERO4}${ZERO4}"
printf '%s%s' "$HDR3" "" > duplicate-submit-rejected-shape.hex

cat > duplicate-submit-rejected-shape.json <<EOF
{
  "name": "duplicate-submit-rejected-shape",
  "schema_version": 0,
  "epoch": 0,
  "request_id": "${REQ1}",
  "object_id": 1,
  "tid": 0,
  "parent_tid": 0,
  "operation": "duplicate_submit",
  "status": "PROPOSED",
  "content_hash": null,
  "payload_hex": "",
  "hex_file": "duplicate-submit-rejected-shape.hex"
}
EOF

echo "generated: genesis-submit, committed-persistent, duplicate-submit-rejected-shape"
