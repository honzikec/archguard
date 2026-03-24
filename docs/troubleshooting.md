# Troubleshooting

## Alias imports not resolved

- Ensure `project.tsconfig` is correct, or define `project.aliases`
- Confirm aliases map to real files and index files

## Too many false positives

- Narrow `project.roots`
- Add `project.exclude`
- Add rule-level `except`
- Start with `severity: warning` and raise gradually

## Slow checks

- Limit roots to relevant source directories
- Exclude generated/build folders
- Use `--changed-only` in local workflows

## Config load fails

- v1 is strict; unknown keys fail
- verify enum values for `kind`/`severity`
- verify regex patterns in `file_pattern.target`

## Parse/read errors in CI

- run with `--parse-error-policy=error` to fail fast on skipped files
- inspect summary fields `parse_errors` and `files_skipped`
- use `--debug` to print per-file parse/read failures

## Changed-file checks miss expected files

- local mode should use `--changed-only`
- PR mode should use `--changed-against <base_ref>` (for example `origin/main`)
- combine both flags to union local and ref-range changes
