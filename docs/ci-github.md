# GitHub CI

ArchGuard supports two CI modes:

- `enforce` (recommended for production): fail job on any non-zero exit
- `audit`: upload SARIF and keep violations non-blocking

Required workflow permissions:

- `contents: read`
- `security-events: write`

Use the published GitHub Action for the common path:

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

## Enforce mode (recommended)

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

## Audit mode (non-blocking violations)

```yaml
- name: ArchGuard
  uses: honzikec/archguard-action@v1
  with:
    config: archguard.yaml
    format: sarif
    parse-error-policy: error
    severity-threshold: error
    enforce: false
    upload-sarif: true
```

## Changed-file strategies

- local checks: `--changed-only`
- PR/merge-base checks: `--changed-against origin/main`
- full-repo enforcement is usually preferred for release/main branch gates

Notes:

- The action installs the published ArchGuard binary and preserves normal CLI exit-code behavior.
- Uploading SARIF from forked pull requests is blocked by GitHub token permissions.
- Guard the upload step to skip fork PRs, or use a `pull_request_target` strategy only if your repository security model allows it.
- `--changed-against <ref>` requires that the ref exists in checkout history (set `actions/checkout` `fetch-depth: 0` when needed).
