# Contributing to ArchGuard

## Prerequisites

- Go 1.21+

## Build and test

```bash
GOCACHE=/tmp/go-build go build -o archguard ./cmd/archguard/main.go
GOCACHE=/tmp/go-build go test ./...
```

## Development notes

- Config schema is strict v1 (`version: 1`)
- Output formats (`text`, `json`, `sarif`) should remain deterministic
- Add tests for new parser/resolver/policy behavior
- Framework-aware mining extensions should follow `docs/frameworks.md`

## Fixture expectations

When adding new fixture projects, include:
- `archguard.yaml`
- minimal source files reproducing the scenario
- one clear expected behavior (pass/fail)

## Pull requests

- Keep changes focused
- Add/adjust tests for behavior changes
- Update docs when CLI/config/rule behavior changes
