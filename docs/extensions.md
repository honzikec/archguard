# Extending ArchGuard (Frameworks and Languages)

This guide is for contributors adding new framework or language support in isolated PRs.

## Framework profile checklist

Required changes:
- add package `internal/framework/profiles/<profile_id>`
- implement `ID`, `Detect`, `NormalizeSubtree`, `NormalizeFile`
- add package-local tests (`profile_test.go`)
- register profile in `internal/framework/registry.go`
- update docs (`docs/frameworks.md`, `docs/config.md`, `docs/cli.md`)

PR scope rule:
- framework profile PRs should not change policy/check semantics

## Language adapter checklist

Required changes:
- add package `internal/language/<adapter_id>`
- implement adapter contract from `internal/language/contracts`
- register adapter in `internal/language/adapter.go`
- keep parser/discovery behavior behind adapter boundary
- add adapter tests and update `docs/languages.md`
- update config validation/docs for `project.language` values when adding new adapter ids

PR scope rule:
- language adapter PRs should avoid framework profile changes unless strictly required

## Conformance expectations

Every new profile/adapter must satisfy:
- deterministic registration order
- deterministic output on repeated runs
- idempotent normalization for frameworks
- no regressions in `check` behavior for existing JS/TS repositories
