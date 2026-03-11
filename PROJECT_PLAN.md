# PROJECT_PLAN.md

# ArchGuard v1 — Architectural Sentinel for Agentic Coding

## Objective

ArchGuard is a **CLI-first static analysis tool** that:

1. Mines architectural invariants from a repository.
2. Encodes those invariants as deterministic policies.
3. Enforces them during code review or CI to prevent architectural drift.

The primary use case is **guarding AI-generated code** so that agent-produced changes cannot violate repository architecture.

ArchGuard acts as a **policy gate** in CI/CD pipelines.

---

# Design Principles

1. **Deterministic first**

   * No LLM usage.
   * All findings must be reproducible.

2. **Fast local execution**

   * Must run in CI in seconds.
   * Sub-second startup time.

3. **Low false positives**

   * Findings must be explainable and traceable.

4. **Simple integration**

   * Works as CLI + GitHub Action.

5. **Agent-compatible**

   * AI coding agents can call the CLI before committing changes.

---

# Technology Stack

| Component      | Technology        |
| -------------- | ----------------- |
| Language       | Go                |
| Parser         | tree-sitter       |
| Policy format  | YAML              |
| Output formats | Text, JSON, SARIF |
| Build system   | Go modules        |

Reasons for Go:

* static binaries
* fast startup
* easy CI distribution
* strong ecosystem for CLI tooling

---

# Version 1 Scope

## Supported

* TypeScript
* JavaScript
* TSX
* JSX
* import boundary rules
* banned package rules
* file naming conventions
* invariant mining from import graphs
* SARIF reporting
* GitHub Action integration

## Explicit Non-Goals

The following **must NOT be implemented in v1**:

* LLM calls
* clone detection
* semantic architecture inference
* Slack/Jira integration
* Git history mining
* IDE plugins
* autofix
* multi-language support
* architecture scoring
* entropy metrics

---

# CLI Commands

## check

Evaluate rules against files.

```
archguard check
archguard check --files file1.ts,file2.ts
archguard check --format text
archguard check --format json
archguard check --format sarif
archguard check --config archguard.yaml
```

### Behavior

* Load config
* discover files
* extract imports
* evaluate policies
* emit findings

### Exit Codes

| Code | Meaning                        |
| ---- | ------------------------------ |
| 0    | no error findings              |
| 1    | rule violation detected        |
| 2    | runtime error / invalid config |

---

## mine

Discover candidate architectural invariants.

```
archguard mine
archguard mine --format yaml
archguard mine --format json
```

### Output

Candidate rules with:

* rule type
* conditions
* confidence
* support counts
* example files

---

## explain

Explain a rule.

```
archguard explain --rule AG-001
```

Output:

* rule definition
* rationale
* examples
* severity
* affected paths

---

# Repository Configuration

Default configuration file:

```
archguard.yaml
```

---

# Configuration Schema

```
version: 1

rules:
  - id: AG-001
    kind: import_boundary
    severity: error
    rationale: "Domain code must not depend on infrastructure."
    conditions:
      from_paths:
        - "src/domain/**"
      forbidden_paths:
        - "src/infra/**"

  - id: AG-002
    kind: banned_package
    severity: error
    rationale: "Domain code must not depend on HTTP libraries."
    conditions:
      from_paths:
        - "src/domain/**"
      forbidden_packages:
        - "axios"

  - id: AG-003
    kind: file_convention
    severity: warning
    rationale: "Service files must use .service.ts naming."
    conditions:
      path_patterns:
        - "src/services/**"
      filename_regex: ".*\\.service\\.ts$"
```

---

# Rule Types

## import_boundary

Prevents imports across architectural layers.

Example:

```
domain -> infra
```

Evaluation:

1. If file path matches `from_paths`
2. resolve each import target
3. check if target matches `forbidden_paths`
4. emit finding

---

## banned_package

Blocks specific packages.

Example:

```
src/domain/** must not import axios
```

Evaluation:

1. match file path
2. check package imports
3. emit finding

---

## file_convention

Enforces naming conventions.

Example:

```
services must end in .service.ts
```

Evaluation:

1. match path
2. test filename regex

---

# File Discovery

Ignored directories:

```
node_modules
dist
build
.next
coverage
```

Also ignore:

* directories beginning with `.`
* files containing `.gen.`

Supported extensions:

```
.ts
.tsx
.js
.jsx
```

---

# Import Resolution

Resolution order:

1. relative file imports
2. directory index imports
3. tsconfig path aliases
4. package imports

Example supported cases:

```
import foo from "./foo"
import foo from "./foo/index"
import foo from "@/utils/foo"
import foo from "axios"
```

If resolution fails:

* treat as package import

---

# Internal Architecture

Project structure:

```
cmd/
  archguard/

internal/
  cli/
  config/
  fileset/
  parser/
  pathutil/
  graph/
  policy/
  miner/
  report/
  model/
  testutil/
```

---

# Package Responsibilities

## cli

Command definitions and argument parsing.

Files:

```
root.go
check.go
mine.go
explain.go
```

---

## config

Load and validate YAML configuration.

Files:

```
load.go
schema.go
validate.go
```

---

## fileset

Find source files to analyze.

Files:

```
discover.go
filter.go
```

---

## parser

Wrap tree-sitter.

Extract import statements.

Files:

```
typescript.go
imports.go
```

---

## pathutil

Utility functions:

* path normalization
* glob matching
* alias resolution

Files:

```
resolve.go
glob.go
normalize.go
```

---

## graph

Build dependency graph.

Files:

```
graph.go
build.go
query.go
```

---

## policy

Evaluate rules.

Files:

```
rules.go
matcher.go
evaluator.go
```

---

## miner

Infer candidate invariants.

Files:

```
candidates.go
confidence.go
propose.go
```

---

## report

Format findings.

Files:

```
text.go
json.go
sarif.go
```

---

# Core Data Structures

## Rule

```
type Rule struct {
    ID        string
    Kind      RuleKind
    Severity  Severity
    Rationale string
    Conditions RuleConditions
}
```

---

## ImportRef

```
type ImportRef struct {
    SourceFile string
    RawImport string
    ResolvedPath string
    IsPackageImport bool
    Line int
    Column int
}
```

---

## Finding

```
type Finding struct {
    RuleID string
    RuleKind string
    Severity string
    Message string
    Rationale string
    FilePath string
    Line int
    Column int
    RawImport string
}
```

---

# Miner Algorithm (v1)

### Step 1

Build full import graph.

Nodes:

```
files
```

Edges:

```
imports
```

---

### Step 2

Group files by subtree:

Example:

```
src/domain
src/ui
src/infra
```

---

### Step 3

Compute import prevalence.

For subtree pair A → B:

```
files_in_A_importing_B / total_files_in_A
```

---

### Step 4

Propose boundary rule when:

```
prevalence <= 2%
AND files_in_A >= 20
```

---

### Step 5

Confidence levels

| Confidence | Condition                   |
| ---------- | --------------------------- |
| High       | <=1% prevalence, >=50 files |
| Medium     | <=2% prevalence, >=20 files |

Rules below medium are ignored.

---

# Testing Strategy

## Unit Tests

Test:

* config parsing
* glob matching
* parser extraction
* path resolution
* rule evaluation

---

## Fixture Repositories

Create 4 fixture repos:

1. layered architecture repo
2. monorepo with tsconfig aliases
3. intentionally broken repo
4. mixed JS/TS project

---

## Snapshot Tests

Validate:

* text output
* JSON output
* SARIF output
* miner YAML output

---

# Performance Targets

| Repo Size | Target      |
| --------- | ----------- |
| 1k files  | <2 seconds  |
| 5k files  | <10 seconds |
| startup   | <150ms      |

---

# Agent Integration

Provide:

```
.agent/skills/archguard.md
```

Example instruction for agents:

1. run `archguard check` before commit
2. treat errors as blocking
3. explain violations in PR description

---

# GitHub Action

Provide:

```
.github/workflows/archguard.yml
```

Example step:

```
- name: Run ArchGuard
  run: archguard check --format sarif
```

Upload SARIF results for code scanning.

---

# Development Milestones

## Milestone 1

CLI + config loader.

Acceptance:

* CLI commands exist
* config validation works

---

## Milestone 2

Parser implementation.

Acceptance:

* imports extracted correctly

---

## Milestone 3

Path resolution.

Acceptance:

* relative + alias imports resolve

---

## Milestone 4

Policy engine.

Acceptance:

* import boundaries enforced

---

## Milestone 5

Reporting.

Acceptance:

* SARIF compatible with GitHub

---

## Milestone 6

Miner implementation.

Acceptance:

* candidate rules generated

---

## Milestone 7

Explain command.

Acceptance:

* rule explanation works

---

# Launch Criteria

Before announcing:

* tested on 3+ open source repositories
* false positives measured
* GitHub Action works
* documentation complete

---

# Anti-Goals

Do not implement:

* entropy scoring
* architecture index
* AI-based policy inference
* public repo leaderboard

Focus on **deterministic architectural enforcement**.
