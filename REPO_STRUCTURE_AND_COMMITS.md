# ArchGuard Repo Structure (v1)

```text
archguard/
│
├─ cmd/
│   └─ archguard/
│       └─ main.go
│
├─ internal/
│   ├─ cli/
│   │   ├─ root.go
│   │   ├─ check.go
│   │   ├─ mine.go
│   │   └─ explain.go
│   │
│   ├─ config/
│   │   ├─ load.go
│   │   ├─ validate.go
│   │   └─ schema.go
│   │
│   ├─ fileset/
│   │   ├─ discover.go
│   │   └─ filter.go
│   │
│   ├─ parser/
│   │   ├─ typescript.go
│   │   └─ imports.go
│   │
│   ├─ pathutil/
│   │   ├─ normalize.go
│   │   ├─ glob.go
│   │   └─ resolve.go
│   │
│   ├─ graph/
│   │   ├─ graph.go
│   │   └─ build.go
│   │
│   ├─ policy/
│   │   ├─ evaluator.go
│   │   ├─ matcher.go
│   │   └─ rules.go
│   │
│   ├─ miner/
│   │   ├─ candidates.go
│   │   ├─ propose.go
│   │   └─ confidence.go
│   │
│   ├─ report/
│   │   ├─ text.go
│   │   ├─ json.go
│   │   └─ sarif.go
│   │
│   ├─ model/
│   │   ├─ finding.go
│   │   ├─ rule.go
│   │   └─ importref.go
│   │
│   └─ testutil/
│       └─ fixtures.go
│
├─ fixtures/
│   ├─ layered_app/
│   ├─ broken_architecture/
│   └─ monorepo_alias/
│
├─ .agent/
│   └─ skills/
│       └─ archguard.md
│
├─ .github/
│   └─ workflows/
│       ├─ ci.yml
│       └─ archguard.yml
│
├─ docs/
│   ├─ architecture.md
│   ├─ rule-engine.md
│   └─ miner.md
│
├─ archguard.yaml.example
├─ PROJECT_PLAN.md
├─ README.md
├─ CONTRIBUTING.md
├─ LICENSE
├─ go.mod
└─ go.sum
```

---

# The First 10 Commits (Critical)

You want the repo to **look mature immediately**, even if the code is minimal.

Here is the exact commit sequence I recommend.

---

# Commit 1

### `Initial repository`

Add:

* README
* LICENSE (MIT or Apache 2)
* PROJECT_PLAN.md
* go.mod

Minimal README with project pitch.

---

# Commit 2

### `Add CLI skeleton`

Add:

```text
cmd/archguard/main.go
internal/cli/root.go
```

CLI should already run:

```bash
archguard --help
```

Output example:

```
ArchGuard — Architectural Sentinel

Commands:
  check
  mine
  explain
```

This is important because contributors immediately see something working.

---

# Commit 3

### `Add config loader`

Add:

```text
internal/config/load.go
internal/config/schema.go
internal/config/validate.go
```

Add example config:

```text
archguard.yaml.example
```

Goal:

```bash
archguard check
```

should fail with:

```
archguard.yaml not found
```

---

# Commit 4

### `Add file discovery`

Add:

```text
internal/fileset/discover.go
internal/fileset/filter.go
```

Feature:

```bash
archguard check
```

should print:

```
Scanning 214 files
```

---

# Commit 5

### `Add TypeScript parser`

Add:

```text
internal/parser/typescript.go
internal/parser/imports.go
```

Using:

```go
github.com/smacker/go-tree-sitter
```

Test command:

```
archguard check --debug
```

Output example:

```
Detected imports:

src/domain/foo.ts -> src/infra/db.ts
src/ui/page.ts -> src/domain/foo.ts
```

---

# Commit 6

### `Add rule evaluation engine`

Add:

```text
internal/policy/rules.go
internal/policy/matcher.go
internal/policy/evaluator.go
```

Now:

```bash
archguard check
```

should detect boundary violations.

Example output:

```
AG-001 Import boundary violation

src/domain/user.ts
imports
src/infra/db.ts
```

Exit code must be `1`.

---

# Commit 7

### `Add reporting formats`

Add:

```text
internal/report/text.go
internal/report/json.go
internal/report/sarif.go
```

Now:

```
archguard check --format sarif
```

outputs valid SARIF.

This unlocks **GitHub code scanning**.

---

# Commit 8

### `Add miner prototype`

Add:

```text
internal/graph/build.go
internal/miner/candidates.go
```

Command:

```
archguard mine
```

Example output:

```
Candidate rule:

src/domain/** should not import src/infra/**

Confidence: HIGH
Support: 137 files
Violations: 1
```

---

# Commit 9

### `Add GitHub Action`

Add:

```text
.github/workflows/archguard.yml
```

Example step:

```yaml
- name: Run ArchGuard
  run: archguard check --format sarif
```

This is **very important for adoption**.

---

# Commit 10

### `Add test fixtures`

Add:

```text
fixtures/
```

Include:

```
layered_app
broken_architecture
monorepo_alias
```

Add tests verifying rule detection.

---

# README Structure (Critical for Stars)

The README should follow this structure:

1. Problem
2. Example violation
3. Quick install
4. Example rule
5. Example output
6. How agents use it
7. Roadmap
8. Contribution guide

Example snippet:

```
AI coding agents can generate code faster than humans can review it.

But they often violate repository architecture.

ArchGuard learns the architecture of your repo and blocks changes that break it.
```

---

# Psychological Trick That Boosts Stars

Add this section in README:

## Used by AI agents

```
Claude Code
Cursor
Google Antigravity
```

Even if unofficial.

Developers instantly understand the category.

---

# First Release Tag

Tag as:

```
v0.1.0
```

Not `v1`.

Reason:

people are far more likely to try early tools.

---

# The Most Important Early Metric

You should optimize for:

```
"works on my repo in under 5 minutes"
```

Not features.

The first 500 GitHub stars usually come from:

```
devs running it on their own repos
```

---

# One More Important Trick

Add this to the README:

```
Works with repositories up to 10k files in under 10 seconds.
```

Performance credibility matters enormously for devtools.

