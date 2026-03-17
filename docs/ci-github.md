# GitHub CI

Recommended workflow pattern:

1. Run `archguard check --format sarif > archguard-results.sarif`
2. Upload SARIF with `if: always()` and `github/codeql-action/upload-sarif@v4`
3. Let check step fail on blocking findings

This preserves both hard-gating and code-scanning visibility.

Required workflow permissions:

- `contents: read`
- `security-events: write`

Notes:

- Uploading SARIF from forked pull requests is blocked by GitHub token permissions.
- Guard the upload step to skip fork PRs, or use a `pull_request_target` strategy only if your repository security model allows it.
