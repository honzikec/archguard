# ArchGuard v0.2

ArchGuard is a deterministic architecture policy checker for TypeScript/JavaScript (and early PHP) projects.

It enforces architectural boundaries (imports, packages, file patterns, and cycles) with CI-friendly output formats (`text`, `json`, `sarif`).

## What it enforces

- `no_import`: block imports from scoped source paths to forbidden local paths
- `no_package`: block package imports from scoped source paths
- `file_pattern`: enforce filename regex rules in selected paths
- `no_cycle`: detect subtree dependency cycles

## Install

```bash
# Pinned Go install, using the release tag you want
go install github.com/honzikec/archguard/cmd/archguard@vX.Y.Z

# Homebrew (tap published from GoReleaser)
brew install honzikec/tap/archguard

# npm wrapper around the released binary
npx @archguard/cli check --config archguard.yaml

# Or download a tagged binary from GitHub Releases
```

ArchGuard requires Go 1.24+ when building from source.

## Quickstart (5 minutes)

```bash
# Build locally
GOCACHE=/tmp/go-build go build -o archguard ./cmd/archguard/main.go

# Guided onboarding preview
./archguard init --guided

# Write starter config + baseline
./archguard init --guided --write-config --write-baseline

# Run checks
./archguard check --config archguard.yaml
```

## CLI

```bash
archguard check   --config archguard.yaml --format text|json|sarif
archguard check   --config archguard.yaml --changed-only
archguard check   --config archguard.yaml --changed-against origin/main --parse-error-policy error
archguard mine    --config archguard.yaml --format text|yaml|json --catalog builtin
archguard explain --config archguard.yaml --rule RULE_ID
archguard explain --config archguard.yaml --finding FINGERPRINT
archguard init    --config archguard.yaml
archguard init    --guided [--write-config] [--write-baseline]
archguard init profile --name my_framework
archguard version
```

Default check behavior:
- Blocking threshold: `error`
- Parse/read error policy: `warn` (set `--parse-error-policy=error` in CI)
- Exit codes: `0` pass, `1` blocking violations, `2` runtime/config/usage error
- relative project paths are resolved from the directory of `--config`

Mining note:
- `mine` uses a framework-aware normalization layer (`generic|nextjs|react|react_router|react_native|angular`) and keeps `check` semantics generic.
- language adapter selection is `project.language: auto|javascript|php` (default `auto`).
- large repos are capped to `200` mined candidates per kind by default (`--max-candidates-per-kind=0` disables cap).
- monorepos are mined workspace-by-workspace by default (`--workspace-mode=auto`) using workspace metadata/conventions.

## Example config

```yaml
version: 1
project:
  roots: ["."]
  include: ["**/*.ts", "**/*.tsx", "**/*.mts", "**/*.cts", "**/*.js", "**/*.jsx", "**/*.mjs", "**/*.cjs", "**/*.php", "**/*.phtml"]
  exclude: ["**/node_modules/**", "**/dist/**", "**/build/**", "**/.next/**", "**/coverage/**", "**/.git/**", "**/vendor/**", "**/runtime/**", "**/storage/**", "**/cache/**", "**/migrations/**"]
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

For production gating, use the published GitHub Action.

```yaml
- name: ArchGuard
  uses: honzikec/archguard-action@v1
  with:
    config: archguard.yaml
    format: sarif
    parse-error-policy: error
    severity-threshold: error
    enforce: true
    upload-sarif: true
```

Brownfield adoption:
- `archguard check --write-baseline archguard-baseline.json` records current findings without failing the run
- `archguard check --baseline archguard-baseline.json` suppresses only findings already present in that baseline
- JSON/text summaries include `suppressed_findings`; suppressed findings are omitted from SARIF

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
- [Onboarding (First 30 Minutes)](docs/onboarding-30-minutes.md)
- [Troubleshooting](docs/troubleshooting.md)

## Contributing

See `CONTRIBUTING.md`.
