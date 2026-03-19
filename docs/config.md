# Config v1

`archguard.yaml` is strict and rejects unknown keys.

## Schema

```yaml
version: 1
project:
  roots: ["."]
  include: ["**/*.ts", "**/*.tsx", "**/*.js", "**/*.jsx", "**/*.mjs", "**/*.cjs"]
  exclude: ["**/node_modules/**", "**/dist/**", "**/build/**", "**/.next/**", "**/coverage/**", "**/.git/**"]
  framework: generic|nextjs|react_router|react_native|angular # optional; affects mining normalization only
  tsconfig: "tsconfig.json" # optional
  aliases:
    "@/*": ["src/*"]

rules:
  - id: AG-ID
    kind: no_import|no_package|file_pattern|no_cycle|pattern
    severity: error|warning
    scope: ["src/**"]
    target: []      # required except for no_cycle
    except: []      # optional
    template: dependency_constraint|construction_policy # required for kind=pattern
    params: {}      # optional key/value params for template behavior
    message: "..." # optional
```

Example `pattern` rule:

```yaml
- id: AG-CAT-LAYERED-DOMAIN-INFRA
  kind: pattern
  template: dependency_constraint
  severity: warning
  scope: ["src/domain/**"]
  target: ["src/infra/**"]
  params:
    relation: imports
  message: "[derived_from_catalog=CAT-LAYERED-DOMAIN-INFRA] Domain should avoid infra dependencies."
```

## Validation rules

- `version` must be `1`
- `id`, `kind`, `severity`, `scope` are required for every rule
- `target` is required for `no_import`, `no_package`, `file_pattern`
- `target` must be empty for `no_cycle`
- `kind: pattern` requires a supported `template`
- `file_pattern.target` entries must be valid regexes
- duplicate rule IDs are rejected
- invalid globs/regexes fail config load
