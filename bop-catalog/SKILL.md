---
name: bop-catalog
description: Discover callable bop cards via the fleet catalog endpoint. Any LM can list available capabilities, read their schemas, and invoke them by id. Trigger phrases — bop catalog, discover card, list cards, available capabilities, fleet capabilities.
---

# Bop Catalog — AX Discovery Endpoint

Phase 18 Wave 1 artifact. Turns bop from tribal to self-describing.

## Single stable endpoint

```
http://10.0.3.210:3000/string/bop/releases/download/catalog%2Flatest/latest.json
```

The `catalog/latest` release tag is a rolling pointer. Versioned snapshots live at `catalog/v1-YYYYMMDD` tags on the same repo (`string/bop` on QNAS Gitea).

## Usage (<=3 tool calls)

1. **List** — what capabilities exist?
   ```
   curl -s <endpoint> | jq '.cards[] | {id, title, min_trust, side_effects}'
   ```

2. **Inspect** — full manifest for one card (embedded in the catalog, no extra fetch needed):
   ```
   curl -s <endpoint> | jq '.cards[] | select(.id == "example-hello") | .manifest'
   ```

3. **Invoke** — dispatch with validated input:
   ```
   bop submit example-hello --input '{"name": "agent"}'
   ```

## Schemas

- `~/bop/schema/manifest.v1.json` — per-card manifest (id, version, inputs, outputs, min_trust, side_effects, model_hints, examples)
- `~/bop/schema/catalog.v1.json` — aggregate catalog (schema_version, generated_at, cards[])

Both validate as JSON Schema draft 2020-12.

## Regenerating the catalog

```
nu ~/bop/scripts/generate-catalog.nu
tea release create --login qnas --repo string/bop \
  --tag "catalog/v1-$(date +%Y%m%d)" \
  --asset ~/bop/catalog/latest.json \
  --note "..."
```

Then re-upload `latest.json` to the `catalog/latest` rolling tag.

## Versioning

- `v1` is the current schema. Changes that break LM parsing bump to `v2` with a new endpoint path. `v1` stays working.
- Per-card `version` follows SemVer.
- `catalog/latest` tag is a rolling pointer; dated `catalog/v1-YYYYMMDD` tags give point-in-time snapshots.

## Trust tiers

Cards declare `min_trust: L0|L1|L2|L3`. The dispatcher must refuse to invoke a card for an agent below that tier. See fleet trust-tiers docs.
