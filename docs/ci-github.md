# GitHub CI

Recommended workflow pattern:

1. Run `archguard check --format sarif > archguard-results.sarif`
2. Upload SARIF with `if: always()`
3. Let check step fail on blocking findings

This preserves both hard-gating and code-scanning visibility.
