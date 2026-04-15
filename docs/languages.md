# Language Adapter Layer

ArchGuard now routes file discovery and import parsing through a language adapter contract.

## Current adapters

- `javascript`
  - supported files: `.ts`, `.tsx`, `.mts`, `.cts`, `.js`, `.jsx`, `.mjs`, `.cjs`
  - import extraction: deterministic Tree-sitter AST extraction for static imports/re-exports, import attributes, literal `require`, and literal dynamic `import`
  - path resolution: relative imports, `tsconfig.json`/`jsconfig.json` `baseUrl`, local/package `extends`, `paths` aliases, local npm/pnpm/yarn workspace packages, package `exports`, package `imports`, and package entry fields
- `php`
  - supported files: `.php`, `.phtml`
  - import extraction: deterministic Tree-sitter AST extraction of `use` declarations and static `require/include` string literals
  - path resolution: relative include/require plus Composer `autoload.psr-4` / `autoload-dev.psr-4` namespace mapping
  - framework aliases: resolves Yii-style local aliases (for example `@common/config/main.php`) when target files exist

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

## Current JavaScript/TypeScript limitations

- non-literal dynamic imports are ignored and counted in debug output
- package-based `tsconfig extends` is resolved only when the package is local to the workspace or already present in `node_modules`; ArchGuard never installs packages
- package `exports`/`imports` support covers string targets, condition maps, arrays, exact subpaths, and simple `*` subpaths that point to local files
- local workspace package imports fall back to package subpaths and `src/` subpaths when package metadata does not resolve to a source file
- full Node/bundler resolution is intentionally out of scope; unresolved external bare imports remain package imports
