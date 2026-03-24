# GitHub CI

ArchGuard supports two CI modes:

- `enforce` (recommended for production): fail job on any non-zero exit
- `audit`: upload SARIF and keep violations non-blocking

Required workflow permissions:

- `contents: read`
- `security-events: write`

## Enforce mode (recommended)

```yaml
- name: Run ArchGuard (enforce)
  id: archguard
  run: |
    set +e
    archguard check --config archguard.yaml --format sarif --parse-error-policy error > archguard-results.sarif
    code=$?
    echo "exit_code=$code" >> "$GITHUB_OUTPUT"
    exit 0

- name: Upload SARIF
  if: always()
  uses: github/codeql-action/upload-sarif@v4
  with:
    sarif_file: archguard-results.sarif
    category: archguard

- name: Enforce result
  if: always() && steps.archguard.outputs.exit_code != '0'
  run: exit 1
```

## Audit mode (non-blocking violations)

```yaml
- name: Run ArchGuard (audit)
  id: archguard
  run: |
    set +e
    archguard check --config archguard.yaml --format sarif > archguard-results.sarif
    code=$?
    echo "exit_code=$code" >> "$GITHUB_OUTPUT"
    exit 0

- name: Upload SARIF
  if: always()
  uses: github/codeql-action/upload-sarif@v4
  with:
    sarif_file: archguard-results.sarif
    category: archguard

- name: Fail only on runtime/config errors
  if: always() && steps.archguard.outputs.exit_code == '2'
  run: exit 1
```

## Changed-file strategies

- local checks: `--changed-only`
- PR/merge-base checks: `--changed-against origin/main`
- full-repo enforcement is usually preferred for release/main branch gates

Notes:

- Uploading SARIF from forked pull requests is blocked by GitHub token permissions.
- Guard the upload step to skip fork PRs, or use a `pull_request_target` strategy only if your repository security model allows it.
- `--changed-against <ref>` requires that the ref exists in checkout history (set `actions/checkout` `fetch-depth: 0` when needed).
