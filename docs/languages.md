# Language Adapter Layer

ArchGuard now routes file discovery and import parsing through a language adapter contract.

## Current adapters

- `javascript` (default)
  - supported files: `.ts`, `.tsx`, `.js`, `.jsx`, `.mjs`, `.cjs`
  - import extraction: deterministic static parsing via existing parser

## Why this exists

- isolates language-specific parsing/discovery from policy engine
- allows adding new languages as encapsulated adapters
- keeps `check` and `mine` orchestration stable

## PHP follow-up plan (design only)

Planned adapter goals:
- support `.php` discovery in relevant roots
- extract namespace/class import edges (`use`, `require`, `include`) deterministically
- integrate Composer autoload / PSR mapping for resolution
- keep unresolved symbols explicit (no speculative inference)

PHP implementation is intentionally deferred to the next cycle.
