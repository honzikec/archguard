# Config v1

`archguard.yaml` is strict and rejects unknown keys.

## Schema

```yaml
version: 1
project:
  roots: ["."]
  include: ["**/*.ts", "**/*.tsx", "**/*.js", "**/*.jsx", "**/*.mjs", "**/*.cjs"]
  exclude: ["**/node_modules/**", "**/dist/**", "**/build/**", "**/.next/**", "**/coverage/**", "**/.git/**"]
  tsconfig: "tsconfig.json" # optional
  aliases:
    "@/*": ["src/*"]

rules:
  - id: AG-ID
    kind: no_import|no_package|file_pattern|no_cycle
    severity: error|warning
    scope: ["src/**"]
    target: []      # required except for no_cycle
    except: []      # optional
    message: "..." # optional
```

## Validation rules

- `version` must be `2`
- `id`, `kind`, `severity`, `scope` are required for every rule
- `target` is required for `no_import`, `no_package`, `file_pattern`
- `target` must be empty for `no_cycle`
- `file_pattern.target` entries must be valid regexes
- duplicate rule IDs are rejected
- invalid globs/regexes fail config load
