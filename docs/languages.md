# Language Adapter Layer

ArchGuard now routes file discovery and import parsing through a language adapter contract.

## Current adapters

- `javascript`
  - supported files: `.ts`, `.tsx`, `.js`, `.jsx`, `.mjs`, `.cjs`
  - import extraction: deterministic static parsing via existing parser
- `php`
  - supported files: `.php`, `.phtml`
  - import extraction: deterministic regex-based extraction of `use` and `require/include` string literals
  - path resolution: relative include/require plus Composer `autoload.psr-4` / `autoload-dev.psr-4` namespace mapping

## Selection model

- `project.language` controls adapter selection:
  - `auto` (default): detect from project files/configs
  - `javascript` or `php`: explicit selection
- if auto-detection finds no strong match, ArchGuard falls back to `javascript`

## Why this exists

- isolates language-specific parsing/discovery from policy engine
- allows adding new languages as encapsulated adapters
- keeps `check` and `mine` orchestration stable

## Contributor bootstrap

- generate a starter adapter package with:
  - `archguard init adapter --name <adapter_id>`
- this creates `adapter.go` + `adapter_test.go`; wire it in `internal/language/adapter.go` and update `docs/config.md`/validation enums.
- all adapters must pass shared contract conformance:
  - `go test ./internal/language -run TestLanguageAdaptersConformance`

## Current PHP limitations

- PSR-4 resolution currently maps class-like namespace imports to file paths only (no symbol/type validation)
- Composer `autoload.classmap`, `autoload.files`, and non-PSR include conventions are not resolved
- dynamic include expressions are ignored unless the path is a string literal
