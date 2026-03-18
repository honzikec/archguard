# Pattern Catalog

ArchGuard ships a built-in, versioned pattern catalog used by `archguard mine`.

## Design

- Catalog source: built-in YAML files under `internal/catalog/patterns/`
- No runtime network dependency
- Deterministic scoring and ranking

## Mine behavior

Default (`--catalog builtin`):
- classic mined candidates (`candidates[]`)
- catalog matches (`catalog_matches[]`)

Catalog matches include:
- `catalog_id`
- `score`
- `confidence` (`HIGH`, `MEDIUM`, `LOW`)
- evidence
- proposed concrete rule

Only `MEDIUM+` confidence matches are shown by default.
Use `--show-low-confidence` to include `LOW`.

## Adoption

Use `--emit-config --adopt-catalog` to include adopted catalog rules in emitted config.

Thresholds:
- `--adopt-threshold high` (default)
- `--adopt-threshold medium`

Adopted rules carry trace metadata in message:
- `derived_from_catalog: <catalog_id>`
