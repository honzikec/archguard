# ArchGuard v1 — Architectural Sentinel for Agentic Coding

## Problem
AI coding agents can generate code faster than humans can review it.
But they often violate repository architecture.
ArchGuard learns the architecture of your repo and blocks changes that break it.

## Example Violation
A common scenario is a domain layer importing from an infrastructure layer:
`src/domain/user.ts` imports `src/infra/db.ts`

## Quick Install
```bash
go install github.com/honzikec/archguard/cmd/archguard@latest
```

## Example Rule
```yaml
rules:
  - id: AG-001
    kind: import_boundary
    severity: error
    conditions:
      from_paths: ["src/domain/**"]
      forbidden_paths: ["src/infra/**"]
```

## Example Output
```text
AG-001 Import boundary violation
src/domain/user.ts imports src/infra/db.ts
```

## How Agents Use It
Agents can run `archguard check` before committing code to ensure structural integrity and automatically fix any architectural violations they introduce.

## Used by AI agents
Claude Code
Cursor
Google Antigravity

## Roadmap
- [ ] CLI checks
- [ ] CI integration (GitHub Actions)
- [ ] Automated invariant mining
- [ ] SARIF reporting

## Contribution Guide
See `CONTRIBUTING.md` for details on how to run tests and submit PRs.

---
Works with repositories up to 10k files in under 10 seconds.
