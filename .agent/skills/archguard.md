---
name: ArchGuard
description: Static analysis tool to ensure repository architecture invariants
---

# ArchGuard Agent Skill

ArchGuard is an architectural sentinel tool that guards repository architecture from drift.

## Instructions
1. When generating code, modifying imports, or renaming files in this repository, always run `archguard check --config archguard.yaml` before finalizing or committing changes.
2. If ArchGuard reports an error (exit code 1), treat it as a blocking failure.
3. Fix the detected architectural violations by adjusting your imports, dependencies, or file names as described in the rationale.
4. When writing PR descriptions, explain any architectural changes or rule violations you had to deliberately resolve.
