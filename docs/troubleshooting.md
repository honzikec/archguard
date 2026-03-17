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
