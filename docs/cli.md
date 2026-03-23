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
- `--max-candidates-per-kind` (`0` = unlimited, default `200`)
- `--interactive` (interactive rule selection and write-back to config)
- `--workspace-mode` (`auto|off`, default `auto`)
- `--emit-config`
- `--emit-no-cycle-severity` (`warning|error`, default `warning`)
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
- candidate generation applies `--min-support` consistently across `no_import`, `no_package`, `file_pattern`, and `no_cycle`
- zero-violation `no_import`/`no_package` candidates are emitted only when target usage is meaningful elsewhere (to reduce low-signal noise)
- to keep large repos practical, mining caps candidates per kind by default (`--max-candidates-per-kind=200`)
- in `auto`, mine discovers monorepo workspaces from `package.json workspaces`, `pnpm-workspace.yaml`, and `nx/turbo` conventions and mines each workspace independently before merging
- mine resolves a framework profile (`generic|nextjs|react|react_router|react_native|angular`) and applies normalization only to mining inputs
- check/mine resolve language adapter (`auto|javascript|php`) before discovery/parsing
- With `--emit-config --adopt-catalog`, adopted catalog rules are appended to emitted config
- `--interactive` supports selecting mined rules (`a`/`n`/index list), optional severity override, and confirmation before writing config
- emitted `no_cycle` rules default to `warning` unless overridden via `--emit-no-cycle-severity=error`

## `archguard explain`

Flags:
- `--config`
- `--format` (`text|json`)
- `--rule` (required)

## `archguard init`

Flags:
- `--config`
- `--force`
- `--from-brief` (compile architecture brief YAML/JSON to config)
- `--out` (output path when using `--from-brief`, default: `--config`)

Subcommands:
- `archguard init` writes a starter config file
- `archguard init --from-brief architecture-brief.yaml --out archguard.yaml` compiles a harness-agnostic architecture brief into validated config
- `archguard init profile --name <id> [--dir <path>] [--force]` scaffolds a framework profile package for contributors
- `archguard init adapter --name <id> [--dir <path>] [--force]` scaffolds a language adapter package for contributors

## `archguard version`

Print build metadata.
