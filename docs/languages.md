# Language Adapter Layer

ArchGuard now routes file discovery and import parsing through a language adapter contract.

## Current adapters

- `javascript`
  - supported files: `.ts`, `.tsx`, `.js`, `.jsx`, `.mjs`, `.cjs`
  - import extraction: deterministic static parsing via existing parser
- `php`
  - supported files: `.php`, `.phtml`
  - import extraction: deterministic regex-based extraction of `use` and `require/include` string literals

## Selection model

- `project.language` controls adapter selection:
  - `auto` (default): detect from project files/configs
  - `javascript` or `php`: explicit selection
- if auto-detection finds no strong match, ArchGuard falls back to `javascript`

## Why this exists

- isolates language-specific parsing/discovery from policy engine
- allows adding new languages as encapsulated adapters
- keeps `check` and `mine` orchestration stable

## Current PHP limitations

- namespace `use` imports are currently treated as package-like references
- Composer autoload / PSR namespace resolution is not implemented yet
- dynamic include expressions are ignored unless the path is a string literal
