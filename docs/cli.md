# CLI

## `archguard check`

Flags:
- `--config` (default `archguard.yaml`)
- `--format` (`text|json|sarif`)
- `--quiet`
- `--debug`
- `--changed-only`
- `--severity-threshold` (`warning|error`, default `error`)
- `--max-findings` (`0` = unlimited)

Exit codes:
- `0` no blocking findings
- `1` blocking findings
- `2` runtime/config/usage errors

## `archguard mine`

Flags:
- `--config`
- `--format` (`text|yaml|json`)
- `--quiet`
- `--debug`
- `--min-support`
- `--max-prevalence`
- `--emit-config`
- `--catalog` (`builtin|off`, default `builtin`)
- `--catalog-format` (`text|json`)
- `--show-low-confidence`
- `--adopt-catalog` (used with `--emit-config`)
- `--adopt-threshold` (`high|medium`, default `high`)

Output notes:
- JSON output includes `candidates[]` and `catalog_matches[]`
- With `--emit-config --adopt-catalog`, adopted catalog rules are appended to emitted config

## `archguard explain`

Flags:
- `--config`
- `--format` (`text|json`)
- `--rule` (required)

## `archguard init`

Flags:
- `--config`
- `--force`

## `archguard version`

Print build metadata.
