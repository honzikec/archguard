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
- scoring metrics:
  - `scoped_files`
  - `eligible_files`
  - `violating_files`
  - `support`
  - `prevalence`
  - `score_components`:
    - `structural_fit`
    - `prevalence_support`
    - `naming_fit`
- optional precision metadata for construction patterns:
  - `resolved_count`
  - `unresolved_count`
  - `resolved_examples`
  - `unresolved_reasons`
  - `sample_locations` (`file:line`)

Only `MEDIUM+` confidence matches are shown by default.
Use `--show-low-confidence` to include `LOW`.

## Confidence semantics

Final score is weighted:
- `40% structural_fit`
- `40% prevalence_support`
- `20% naming_fit`

Confidence bands:
- `HIGH >= 0.85`
- `MEDIUM >= 0.65`
- `LOW < 0.65`

Calibration notes:
- support normalization uses `effective_min_support = min(configured_min_support, max(3, scoped_files))`
- `construction_policy` prevalence is computed over `eligible_files` (files with relevant `new` signals in scope, excluding allowed roots)
- unresolved-heavy repos lower `structural_fit`, which can keep matches in `LOW` even when naming hints are strong

Recommended operator flow:
- mine in recommend mode
- review confidence + evidence
- adopt with `--adopt-catalog --adopt-threshold=high` first
- expand to `medium` after review

## Adoption

Use `--emit-config --adopt-catalog` to include adopted catalog rules in emitted config.

Thresholds:
- `--adopt-threshold high` (default)
- `--adopt-threshold medium`

Adopted rules carry trace metadata in message:
- `derived_from_catalog: <catalog_id>`
