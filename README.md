# ArchGuard v0.2

ArchGuard is a deterministic architecture policy checker for TypeScript/JavaScript (and early PHP) projects.

It enforces architectural boundaries (imports, packages, file patterns, and cycles) with CI-friendly output formats (`text`, `json`, `sarif`).

## What it enforces

- `no_import`: block imports from scoped source paths to forbidden local paths
- `no_package`: block package imports from scoped source paths
- `file_pattern`: enforce filename regex rules in selected paths
- `no_cycle`: detect subtree dependency cycles

## Quickstart (5 minutes)

```bash
# Build locally
GOCACHE=/tmp/go-build go build -o archguard ./cmd/archguard/main.go

# Create starter config
./archguard init

# Run checks
./archguard check --config archguard.yaml
```

## CLI

```bash
archguard check   --config archguard.yaml --format text|json|sarif
archguard mine    --config archguard.yaml --format text|yaml|json --catalog builtin
archguard explain --config archguard.yaml --rule RULE_ID
archguard init    --config archguard.yaml
archguard init profile --name my_framework
archguard version
```

Default check behavior:
- Blocking threshold: `error`
- Exit codes: `0` pass, `1` blocking violations, `2` runtime/config/usage error

Mining note:
- `mine` uses a framework-aware normalization layer (`generic|nextjs|react|react_router|react_native|angular`) and keeps `check` semantics generic.
- language adapter selection is `project.language: auto|javascript|php` (default `auto`).
- large repos are capped to `200` mined candidates per kind by default (`--max-candidates-per-kind=0` disables cap).

## Example config

```yaml
version: 1
project:
  roots: ["."]
  include: ["**/*.ts", "**/*.tsx", "**/*.js", "**/*.jsx", "**/*.mjs", "**/*.cjs"] # add "**/*.php" for PHP repos
  exclude: ["**/node_modules/**", "**/dist/**", "**/build/**", "**/.next/**", "**/coverage/**", "**/.git/**"]
  language: auto # auto|javascript|php
  framework: nextjs # optional; generic|nextjs|react|react_router|react_native|angular
  aliases:
    "@/*": ["src/*"]

rules:
  - id: AG-NO-INFRA-IN-DOMAIN
    kind: no_import
    severity: error
    scope: ["src/domain/**"]
    target: ["src/infra/**"]

  - id: AG-NO-AXIOS-IN-DOMAIN
    kind: no_package
    severity: warning
    scope: ["src/domain/**"]
    target: ["axios"]

  - id: AG-SERVICE-NAMING
    kind: file_pattern
    severity: warning
    scope: ["src/services/**"]
    target: ["^.*\\.service\\.(ts|js)$"]

  - id: AG-NO-SRC-CYCLES
    kind: no_cycle
    severity: error
    scope: ["src/**"]
```

## GitHub Actions

```yaml
- name: Install ArchGuard
  run: go install github.com/honzikec/archguard/cmd/archguard@latest

- name: Run ArchGuard
  run: archguard check --config archguard.yaml --format sarif > archguard-results.sarif

- name: Upload SARIF
  if: always()
  uses: github/codeql-action/upload-sarif@v4
  with:
    sarif_file: archguard-results.sarif
    category: archguard
```

## Docs

- [Config](docs/config.md)
- [Rules](docs/rules.md)
- [CLI](docs/cli.md)
- [Framework Layer](docs/frameworks.md)
- [Language Adapters](docs/languages.md)
- [Extension Guide](docs/extensions.md)
- [Pattern Catalog](docs/catalog.md)
- [Catalog Sources](docs/catalog-sources.md)
- [GitHub CI](docs/ci-github.md)
- [Troubleshooting](docs/troubleshooting.md)

## Contributing

See `CONTRIBUTING.md`.
