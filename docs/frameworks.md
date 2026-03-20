# Framework-Aware Mining Layer

ArchGuard keeps policy enforcement generic and applies optional framework-aware normalization only during `mine`.

## Design principles

- `check` semantics stay framework-agnostic.
- Framework behavior is isolated in profile modules.
- Profile selection and normalization are deterministic.
- Contributors can add one framework profile in one encapsulated package.

## Architecture

1. `mine` resolves profile via `internal/framework.Resolve(...)`.
   - explicit `project.framework` wins
   - auto-detect uses weighted evidence and selects a unique strongest match when possible
   - ambiguous auto-detect falls back to `generic`
2. `mine` normalizes graph/files through `internal/framework.NormalizeMiningInputs(...)`.
3. Candidate mining and catalog matching run on normalized inputs.

Profile packages:
- `internal/framework/profiles/nextjs`
- `internal/framework/profiles/react`
- `internal/framework/profiles/react_router`
- `internal/framework/profiles/react_native`
- `internal/framework/profiles/angular`

Shared helpers:
- `internal/framework/common`

## Built-in profiles

- `nextjs`
  - route groups `(x)` -> `(group)`
  - dynamic segments `[id]`/`[[...id]]` -> `[param]`
  - parallel routes `@slotName` -> `@slot`
- `react`
  - detects plain React projects when `react` is present without stronger framework markers
  - normalizes ancillary segments (`__tests__` -> `tests`, `__mocks__` -> `mocks`)
  - strips ancillary file suffixes (`.test`, `.spec`, `.stories`, `.story`)
- `react_router`
  - route param segments `:id`/`$id` -> `[param]`
  - `_index` normalization
  - file token normalization for route params
- `react_native`
  - platform files collapse (`.ios/.android/.native/.web`) -> `.platform`
- `angular`
  - route param subtree normalization (`:id` -> `[param]`)
  - route filename normalization (`-routing` / `.routing` -> `-routes` / `.routes`)

## Contributor workflow

1. Scaffold a profile:
   - `archguard init profile --name my_framework`
2. Implement profile logic only in its package.
3. Wire registration once in `internal/framework/registry.go`.
4. Add profile-local tests + conformance coverage.
5. Update `docs/config.md`, `docs/cli.md`, and this file.

## Guardrails

- Do not add framework-specific behavior to `check` in profile PRs.
- Keep normalization idempotent and deterministic.
- Keep profile PRs focused: registry wiring + one profile package + tests/docs.
