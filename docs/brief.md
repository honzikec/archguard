# Architecture Brief (`archguard-brief.v1`)

Use an architecture brief when you want an LLM/agent (any harness) to capture intent in structured form, then compile it deterministically into `archguard.yaml`.

Compile command:

```bash
archguard init --from-brief architecture-brief.yaml --out archguard.yaml
```

The compiler is strict:
- unknown keys/enums fail
- unknown layer references fail
- generated config is validated with normal ArchGuard config validation before write

## Example brief

```yaml
version: 1
project:
  roots: ["src"]
  include: ["**/*.ts", "**/*.tsx"]
  framework: generic
  language: auto
layers:
  - id: domain
    paths: ["src/domain/**"]
  - id: infra
    paths: ["src/infra/**"]
  - id: services
    paths: ["src/services/**"]
  - id: bootstrap
    paths: ["src/bootstrap/**"]
policies:
  - id: AG-DOMAIN-NO-INFRA
    type: deny_import
    severity: error
    from: ["layer:domain"]
    to: ["layer:infra"]
    message: "Domain must not import infra."

  - type: deny_package
    severity: warning
    scope: ["layer:domain"]
    packages: ["axios", "node-fetch"]

  - type: file_pattern
    severity: warning
    scope: ["layer:services"]
    pattern: "^.*\\.service\\.(ts|js)$"

  - type: no_cycle
    severity: error
    scope: ["src/**"]

  - type: construction_policy
    severity: warning
    scope: ["src/**"]
    services: ["layer:services"]
    allow_in: ["layer:bootstrap"]
    service_name_regex: ".*Service$"
```

## Policy type mapping

- `deny_import` -> `kind: no_import` (`from` -> `scope`, `to` -> `target`)
- `deny_package` -> `kind: no_package` (`scope`, `packages` -> `target`)
- `file_pattern` -> `kind: file_pattern` (`pattern` -> regex target)
- `no_cycle` -> `kind: no_cycle`
- `construction_policy` -> `kind: pattern`, `template: construction_policy`
  - `allow_in` and `except` both compile into config `except`

## Layer selectors

Any selector field can use:
- raw glob: `src/domain/**`
- layer reference: `layer:domain`

Layer references are expanded to the layer `paths` globs during compile.
