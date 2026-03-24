# Onboarding (First 30 Minutes)

This guide gets ArchGuard usable in a real repository quickly, including monorepos and mixed-language projects.

## 1) Install and run baseline (5 minutes)

```bash
go install github.com/honzikec/archguard/cmd/archguard@latest
archguard init
archguard check --config archguard.yaml --format text
```

## 2) Scope the project roots (10 minutes)

Edit `archguard.yaml`:

- narrow `project.roots` to real source roots
- keep `project.exclude` aggressive (`node_modules`, build output, coverage)
- keep rules on `warning` first, then raise to `error`

Monorepo tip:

- use one root config at the repository root
- keep `project.roots` focused to workspace source directories
- use `archguard mine --workspace-mode auto` for workspace-aware discovery

## 3) Lock language mode (5 minutes)

- JS/TS repo: `project.language: javascript`
- PHP repo: `project.language: php` and include `**/*.php`
- mixed repo: prefer explicit language selection per config to avoid ambiguous auto-detection

Tip: maintain separate configs per language area when architecture differs significantly.

## 4) Choose local and CI workflows (10 minutes)

Local fast loop:

```bash
archguard check --config archguard.yaml --changed-only --format text
```

CI enforcement:

```bash
archguard check --config archguard.yaml --format sarif --parse-error-policy error
```

PR range filtering (optional):

```bash
archguard check --config archguard.yaml --changed-against origin/main --format text
```

## Recommended adoption path

1. Start with warnings and narrow scope.
2. Run for a few days in audit mode.
3. Promote stable rules to `error`.
4. Enable enforce mode in CI.
