# Rules

## `no_import`

Blocks local imports from scoped files to target path globs.

## `no_package`

Blocks package imports from scoped files to target package names/globs.

## `file_pattern`

For files in `scope`, filename must match at least one regex in `target`.

## `no_cycle`

Detects dependency cycles between scoped subtrees.

## Exceptions

`except` can suppress a rule when it matches source path and/or target path/import.
