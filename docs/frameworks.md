# Framework-Aware Mining Layer

ArchGuard keeps core analysis generic, then applies an optional framework-aware normalization layer during `mine`.

This lets contributors improve candidate quality for specific ecosystems (for example, Next.js app router) without changing parser/resolver/policy semantics.

## Design intent

- Keep `check` and rule enforcement generic and deterministic.
- Apply framework adaptation only to mining inputs.
- Preserve stable output behavior (`text`, `json`, `yaml`) and deterministic ordering.
- Make framework support pluggable and low-risk.

## Current architecture

1. CLI resolves framework profile in `runMine`:
   - explicit: `project.framework` (`generic` or `nextjs`)
   - auto-detect: known framework config files under project roots
2. `mine` passes framework into `miner.Options.Framework`.
3. `miner.Propose(...)` normalizes graph/files via `normalizeMiningInputs(...)`.
4. Candidate mining runs on normalized graph/files.

Key files:
- `internal/cli/mine.go`
- `internal/miner/candidates.go`
- `internal/miner/framework_normalize.go`
- `internal/config/schema.go`
- `internal/config/validate.go`

## Next.js profile behavior

Current Next.js normalization collapses path segments to reduce route-fragment noise:
- route groups: `(marketing)` -> `(group)`
- dynamic segments: `[slug]`, `[[...slug]]` -> `[param]`
- parallel routes: `@modal` -> `@slot`

This is intentionally lossy for mining because we care about architectural boundaries, not route parameter names.

## How to add a new framework profile

1. Add schema support:
   - extend allowed `project.framework` values in `internal/config/validate.go`
2. Add profile detection:
   - update `resolveMiningFramework(...)` in `internal/cli/mine.go`
   - support both explicit config and safe auto-detection
3. Add normalization logic:
   - extend `normalizeSubtreeForFramework(...)` in `internal/miner/framework_normalize.go`
   - keep normalization deterministic and idempotent
4. Keep scope limited:
   - normalize only mining graph/file inputs
   - do not change parser, resolver, or policy evaluation for `check`
5. Add tests:
   - config validation test for new `project.framework` value
   - normalization unit tests for segment mapping and edge aggregation
   - CLI test for framework auto-detection behavior
6. Update docs:
   - `docs/config.md` and `docs/cli.md`
   - this file (`docs/frameworks.md`)

## Contributor guardrails

- Do not introduce framework-specific behavior into rule semantics unless explicitly designed as a rule feature.
- Prefer normalization at directory/segment level over file-by-file heuristics.
- Avoid probabilistic or non-deterministic logic.
- Ensure normalized paths still produce stable sorting/dedupe.

## Validation checklist

- `GOCACHE=/tmp/go-build go test ./...` passes.
- `mine --debug` shows expected framework profile when auto-detected.
- Candidate count/shape improves on representative framework repos.
