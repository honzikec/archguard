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
- `catalog_matches[]` includes scoring metrics (`scoped_files`, `eligible_files`, `violating_files`, `support`, `prevalence`, `score_components`)
- construction matches also include precision evidence (`resolved_examples`, `unresolved_reasons`, `sample_locations`)
- text output stays concise by default; use `--debug` to print detailed catalog scoring/evidence
- `--debug` also prints framework/language resolution and mining normalization stats
- mine resolves a framework profile (`generic|nextjs|react|react_router|react_native|angular`) and applies normalization only to mining inputs
- check/mine resolve language adapter (`auto|javascript|php`) before discovery/parsing
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

Subcommands:
- `archguard init` writes a starter config file
- `archguard init profile --name <id> [--dir <path>] [--force]` scaffolds a framework profile package for contributors

## `archguard version`

Print build metadata.
