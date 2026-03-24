package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/honzikec/archguard/internal/config"
)

func TestCheckAliasViolation(t *testing.T) {
	code, out, errOut := runCmdInDir(t, fixturePath("monorepo_alias"), []string{"check", "--config", "archguard.yaml", "--format", "json"})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d stderr=%s", code, errOut)
	}
	if !strings.Contains(out, "AG-NO-INFRA") {
		t.Fatalf("expected rule id in output, got: %s", out)
	}
}

func TestSeverityThresholdBehavior(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "archguard.yaml"), `version: 1
project:
  roots: ["src"]
  include: ["**/*.ts"]
  exclude: ["**/node_modules/**"]
rules:
  - id: AG-NO-PKG
    kind: no_package
    severity: warning
    scope: ["src/domain/**"]
    target: ["axios"]
`)
	mustWriteFile(t, filepath.Join(dir, "src", "domain", "user.ts"), `import axios from "axios"`)

	code, _, errOut := runCmdInDir(t, dir, []string{"check", "--config", "archguard.yaml", "--severity-threshold", "error", "--format", "json"})
	if code != 0 {
		t.Fatalf("expected exit 0 for threshold error, got %d stderr=%s", code, errOut)
	}

	code, _, errOut = runCmdInDir(t, dir, []string{"check", "--config", "archguard.yaml", "--severity-threshold", "warning", "--format", "json"})
	if code != 1 {
		t.Fatalf("expected exit 1 for threshold warning, got %d stderr=%s", code, errOut)
	}
}

func TestInvalidConfigReturnsCode2(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "archguard.yaml"), `version: 1
project:
  roots: ["src"]
rules:
  - id: AG-1
    kind: nope
    severity: error
    scope: ["src/**"]
`)
	mustWriteFile(t, filepath.Join(dir, "src", "a.ts"), "export const a = 1")

	code, _, _ := runCmdInDir(t, dir, []string{"check", "--config", "archguard.yaml"})
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
}

func TestCycleFixtureFails(t *testing.T) {
	code, out, errOut := runCmdInDir(t, fixturePath("cycle_fail"), []string{"check", "--config", "archguard.yaml", "--format", "json"})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d stderr=%s", code, errOut)
	}
	if !strings.Contains(out, "AG-NO-CYCLES") {
		t.Fatalf("expected cycle rule in output")
	}
}

func TestSarifIncludesFingerprints(t *testing.T) {
	code, out, errOut := runCmdInDir(t, fixturePath("broken_architecture"), []string{"check", "--config", "archguard.yaml", "--format", "sarif"})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d stderr=%s", code, errOut)
	}
	if !strings.Contains(out, "partialFingerprints") {
		t.Fatalf("expected sarif fingerprints, got: %s", out)
	}
}

func TestMineIncludesCatalogMatches(t *testing.T) {
	code, out, errOut := runCmdInDir(t, fixturePath("broken_architecture"), []string{
		"mine",
		"--config", "archguard.yaml",
		"--format", "json",
		"--min-support", "1",
		"--max-prevalence", "1",
		"--show-low-confidence",
	})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d stderr=%s", code, errOut)
	}
	if !strings.Contains(out, "\"catalog_matches\"") {
		t.Fatalf("expected catalog_matches in mine json output, got: %s", out)
	}
}

func TestMineDetectsConstructionCatalogPattern(t *testing.T) {
	code, out, errOut := runCmdInDir(t, fixturePath("anti_pattern_direct_new"), []string{
		"mine",
		"--config", "archguard.yaml",
		"--format", "json",
		"--show-low-confidence",
	})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d stderr=%s", code, errOut)
	}
	if !strings.Contains(out, "CAT-SERVICES-VIA-COMPOSITION-ROOT") {
		t.Fatalf("expected construction catalog match, got: %s", out)
	}
}

func TestMineEmitConfigAdoptsCatalog(t *testing.T) {
	code, out, errOut := runCmdInDir(t, fixturePath("broken_architecture"), []string{
		"mine",
		"--config", "archguard.yaml",
		"--emit-config",
		"--adopt-catalog",
		"--adopt-threshold", "medium",
		"--min-support", "1",
		"--max-prevalence", "1",
		"--show-low-confidence",
	})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d stderr=%s", code, errOut)
	}
	if !strings.Contains(out, "kind: pattern") {
		t.Fatalf("expected adopted pattern rule in emitted config, got: %s", out)
	}
	if !strings.Contains(out, "derived_from_catalog") {
		t.Fatalf("expected derived_from_catalog trace in emitted config")
	}
}

func TestMineInteractiveWritesSelectedRules(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "archguard.yaml"), `version: 1
project:
  roots: ["src"]
  include: ["**/*.ts"]
  exclude: ["**/node_modules/**"]
rules: []
`)
	mustWriteFile(t, filepath.Join(dir, "src", "domain", "a.ts"), `import b from "../infra/b"`)
	mustWriteFile(t, filepath.Join(dir, "src", "infra", "b.ts"), `export const b = 1`)

	code, out, errOut := runCmdInDirWithInput(t, dir, []string{
		"mine",
		"--config", "archguard.yaml",
		"--catalog", "off",
		"--min-support", "1",
		"--max-prevalence", "1",
		"--interactive",
	}, "a\ne\ny\n")
	if code != 0 {
		t.Fatalf("expected interactive mine exit 0, got %d stderr=%s output=%s", code, errOut, out)
	}
	if !strings.Contains(out, "Updated archguard.yaml with") {
		t.Fatalf("expected updated config message, got: %s", out)
	}

	cfg, err := config.Load(filepath.Join(dir, "archguard.yaml"))
	if err != nil {
		t.Fatalf("expected written config to be valid: %v", err)
	}
	if len(cfg.Rules) == 0 {
		t.Fatal("expected at least one selected rule to be written")
	}
	for _, rule := range cfg.Rules {
		if rule.Severity != "error" {
			t.Fatalf("expected overridden severity=error, got rule %+v", rule)
		}
	}
}

func TestCheckConstructionPolicyPassCompositionRoot(t *testing.T) {
	code, out, errOut := runCmdInDir(t, fixturePath("composition_root_only"), []string{
		"check", "--config", "archguard.yaml", "--format", "json",
	})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d stderr=%s output=%s", code, errOut, out)
	}
}

func TestCheckConstructionPolicyFailAliasImportedService(t *testing.T) {
	code, out, errOut := runCmdInDir(t, fixturePath("alias_imported_service_new"), []string{
		"check", "--config", "archguard.yaml", "--format", "json",
	})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d stderr=%s output=%s", code, errOut, out)
	}
	if !strings.Contains(out, "AG-CONSTRUCTION-POLICY") {
		t.Fatalf("expected construction policy finding, got: %s", out)
	}
}

func TestCheckConstructionPolicyIgnoresUnresolvedDynamic(t *testing.T) {
	code, out, errOut := runCmdInDir(t, fixturePath("dynamic_new_unresolved"), []string{
		"check", "--config", "archguard.yaml", "--format", "json",
	})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d stderr=%s output=%s", code, errOut, out)
	}
}

func TestCheckPHPBoundaryViolation(t *testing.T) {
	code, out, errOut := runCmdInDir(t, fixturePath("php_boundary_fail"), []string{
		"check", "--config", "archguard.yaml", "--format", "json",
	})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d stderr=%s output=%s", code, errOut, out)
	}
	if !strings.Contains(out, "AG-PHP-NO-INFRA") {
		t.Fatalf("expected php no_import finding, got: %s", out)
	}
}

func TestCheckPHPNamespaceBoundaryViolation(t *testing.T) {
	code, out, errOut := runCmdInDir(t, fixturePath("php_namespace_boundary_fail"), []string{
		"check", "--config", "archguard.yaml", "--format", "json",
	})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d stderr=%s output=%s", code, errOut, out)
	}
	if !strings.Contains(out, "AG-PHP-NAMESPACE-NO-INFRA") {
		t.Fatalf("expected php namespace no_import finding, got: %s", out)
	}
}

func TestCheckResolvesConfigRelativePathsFromConfigDir(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "project")
	mustWriteFile(t, filepath.Join(projectDir, "archguard.yaml"), `version: 1
project:
  roots: ["src"]
  include: ["**/*.ts"]
  exclude: ["**/node_modules/**"]
rules:
  - id: AG-NO-INFRA
    kind: no_import
    severity: error
    scope: ["src/domain/**"]
    target: ["src/infra/**"]
`)
	mustWriteFile(t, filepath.Join(projectDir, "src", "domain", "user.ts"), `import db from "../infra/db"`)
	mustWriteFile(t, filepath.Join(projectDir, "src", "infra", "db.ts"), `export const db = 1`)

	code, out, errOut := runCmdInDir(t, dir, []string{
		"check",
		"--config", filepath.Join("project", "archguard.yaml"),
		"--format", "json",
	})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d stderr=%s output=%s", code, errOut, out)
	}

	var payload struct {
		Findings []struct {
			RuleID   string `json:"rule_id"`
			FilePath string `json:"file_path"`
		} `json:"findings"`
		Summary struct {
			FilesScanned   int      `json:"files_scanned"`
			ConfigDir      string   `json:"config_dir"`
			EffectiveRoots []string `json:"effective_roots"`
		} `json:"summary"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("failed to decode check output: %v", err)
	}
	if payload.Summary.FilesScanned != 2 {
		t.Fatalf("expected 2 scanned files, got %d", payload.Summary.FilesScanned)
	}
	if len(payload.Findings) != 1 || payload.Findings[0].RuleID != "AG-NO-INFRA" {
		t.Fatalf("expected AG-NO-INFRA finding, got %+v", payload.Findings)
	}
	if payload.Findings[0].FilePath != "src/domain/user.ts" {
		t.Fatalf("expected finding file path to remain config-root relative, got %s", payload.Findings[0].FilePath)
	}
	resolvedProjectDir := projectDir
	if evaled, err := filepath.EvalSymlinks(projectDir); err == nil && strings.TrimSpace(evaled) != "" {
		resolvedProjectDir = evaled
	}
	if payload.Summary.ConfigDir != filepath.ToSlash(filepath.Clean(resolvedProjectDir)) {
		t.Fatalf("unexpected config_dir, got %s", payload.Summary.ConfigDir)
	}
	wantRoot := filepath.ToSlash(filepath.Join(resolvedProjectDir, "src"))
	if len(payload.Summary.EffectiveRoots) != 1 || payload.Summary.EffectiveRoots[0] != wantRoot {
		t.Fatalf("unexpected effective roots, want [%s] got %v", wantRoot, payload.Summary.EffectiveRoots)
	}
}

func TestCheckParseErrorPolicyWarnAndError(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "archguard.yaml"), `version: 1
project:
  roots: ["src"]
  include: ["**/*.php"]
  exclude: ["**/vendor/**"]
  language: php
rules:
  - id: AG-PHP-NO-INFRA
    kind: no_import
    severity: error
    scope: ["src/**"]
    target: ["src/infra/**"]
`)
	mustWriteFile(t, filepath.Join(dir, "src", "bad.php"), `<?php
function broken( {
`)

	code, out, errOut := runCmdInDir(t, dir, []string{
		"check",
		"--config", "archguard.yaml",
		"--format", "json",
	})
	if code != 0 {
		t.Fatalf("expected warn policy to exit 0, got %d stderr=%s output=%s", code, errOut, out)
	}

	var payload struct {
		Summary struct {
			ParseErrors  int `json:"parse_errors"`
			FilesSkipped int `json:"files_skipped"`
		} `json:"summary"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("failed to decode warn output: %v", err)
	}
	if payload.Summary.ParseErrors != 1 || payload.Summary.FilesSkipped != 1 {
		t.Fatalf("expected parse_errors=1 files_skipped=1, got %+v", payload.Summary)
	}

	code, out, errOut = runCmdInDir(t, dir, []string{
		"check",
		"--config", "archguard.yaml",
		"--format", "json",
		"--parse-error-policy", "error",
	})
	if code != 2 {
		t.Fatalf("expected strict parse policy to exit 2, got %d stderr=%s output=%s", code, errOut, out)
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("failed to decode strict output: %v", err)
	}
	if payload.Summary.ParseErrors != 1 || payload.Summary.FilesSkipped != 1 {
		t.Fatalf("expected parse_errors=1 files_skipped=1, got %+v", payload.Summary)
	}
}

func TestCheckChangedOnlyUsesWorkingTreeDiff(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "archguard.yaml"), `version: 1
project:
  roots: ["src"]
  include: ["**/*.ts"]
  exclude: ["**/node_modules/**"]
rules:
  - id: AG-NO-AXIOS
    kind: no_package
    severity: error
    scope: ["src/domain/**"]
    target: ["axios"]
`)
	mustWriteFile(t, filepath.Join(dir, "src", "domain", "user.ts"), `export const user = 1`)

	mustRunGit(t, dir, "init")
	mustRunGit(t, dir, "config", "user.email", "test@example.com")
	mustRunGit(t, dir, "config", "user.name", "ArchGuard Test")
	mustRunGit(t, dir, "add", ".")
	mustRunGit(t, dir, "commit", "-m", "initial")

	mustWriteFile(t, filepath.Join(dir, "src", "domain", "user.ts"), `import axios from "axios"
export const user = 1`)

	code, out, errOut := runCmdInDir(t, dir, []string{
		"check",
		"--config", "archguard.yaml",
		"--format", "json",
		"--changed-only",
	})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d stderr=%s output=%s", code, errOut, out)
	}
	var payload struct {
		Summary struct {
			FilesScanned int `json:"files_scanned"`
		} `json:"summary"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("failed to decode changed-only output: %v", err)
	}
	if payload.Summary.FilesScanned != 1 {
		t.Fatalf("expected changed-only to scan 1 file, got %d", payload.Summary.FilesScanned)
	}
}

func TestCheckChangedAgainstUsesRefRange(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "archguard.yaml"), `version: 1
project:
  roots: ["src"]
  include: ["**/*.ts"]
  exclude: ["**/node_modules/**"]
rules:
  - id: AG-NO-AXIOS
    kind: no_package
    severity: error
    scope: ["src/domain/**"]
    target: ["axios"]
`)
	mustWriteFile(t, filepath.Join(dir, "src", "domain", "user.ts"), `export const user = 1`)

	mustRunGit(t, dir, "init")
	mustRunGit(t, dir, "config", "user.email", "test@example.com")
	mustRunGit(t, dir, "config", "user.name", "ArchGuard Test")
	mustRunGit(t, dir, "add", ".")
	mustRunGit(t, dir, "commit", "-m", "initial")

	mustWriteFile(t, filepath.Join(dir, "src", "domain", "user.ts"), `import axios from "axios"
export const user = 1`)
	mustRunGit(t, dir, "add", ".")
	mustRunGit(t, dir, "commit", "-m", "introduce axios")

	code, out, errOut := runCmdInDir(t, dir, []string{
		"check",
		"--config", "archguard.yaml",
		"--format", "json",
		"--changed-against", "HEAD~1",
	})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d stderr=%s output=%s", code, errOut, out)
	}
	var payload struct {
		Summary struct {
			FilesScanned int `json:"files_scanned"`
		} `json:"summary"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("failed to decode changed-against output: %v", err)
	}
	if payload.Summary.FilesScanned != 1 {
		t.Fatalf("expected changed-against to scan 1 file, got %d", payload.Summary.FilesScanned)
	}
}

func TestMineCalibrationConfidenceScenarios(t *testing.T) {
	type minePayload struct {
		CatalogMatches []struct {
			CatalogID         string  `json:"catalog_id"`
			Confidence        string  `json:"confidence"`
			Score             float64 `json:"score"`
			UnresolvedReasons []struct {
				Reason string `json:"reason"`
				Count  int    `json:"count"`
			} `json:"unresolved_reasons"`
		} `json:"catalog_matches"`
	}

	runMine := func(t *testing.T, fixture string) minePayload {
		t.Helper()
		code, out, errOut := runCmdInDir(t, fixturePath(fixture), []string{
			"mine", "--config", "archguard.yaml", "--format", "json", "--show-low-confidence",
		})
		if code != 0 {
			t.Fatalf("expected exit 0 for %s, got %d stderr=%s", fixture, code, errOut)
		}
		var payload minePayload
		if err := json.Unmarshal([]byte(out), &payload); err != nil {
			t.Fatalf("failed to decode mine output for %s: %v", fixture, err)
		}
		return payload
	}

	small := runMine(t, "construction_small_repo_clear_violation")
	if len(small.CatalogMatches) == 0 || small.CatalogMatches[0].CatalogID != "CAT-SERVICES-VIA-COMPOSITION-ROOT" {
		t.Fatalf("expected construction catalog match for small fixture, got %+v", small.CatalogMatches)
	}
	if small.CatalogMatches[0].Confidence == "LOW" {
		t.Fatalf("expected at least MEDIUM confidence for small clear violation, got %+v", small.CatalogMatches[0])
	}

	medium := runMine(t, "construction_medium_repo_mixed_usage")
	if len(medium.CatalogMatches) == 0 {
		t.Fatalf("expected construction match for medium fixture")
	}
	if medium.CatalogMatches[0].Confidence == "HIGH" {
		t.Fatalf("expected medium mixed-usage fixture to remain below HIGH, got %+v", medium.CatalogMatches[0])
	}
	if len(medium.CatalogMatches[0].UnresolvedReasons) < 2 {
		t.Fatalf("expected multiple unresolved reasons in medium fixture, got %+v", medium.CatalogMatches[0].UnresolvedReasons)
	}
	if medium.CatalogMatches[0].UnresolvedReasons[0].Count < medium.CatalogMatches[0].UnresolvedReasons[1].Count {
		t.Fatalf("expected unresolved reasons sorted by count desc, got %+v", medium.CatalogMatches[0].UnresolvedReasons)
	}

	large := runMine(t, "construction_large_repo_rare_violation")
	if len(large.CatalogMatches) == 0 {
		t.Fatalf("expected construction match for large fixture")
	}
	if large.CatalogMatches[0].Confidence == "HIGH" {
		t.Fatalf("expected sparse large fixture to remain below HIGH, got %+v", large.CatalogMatches[0])
	}
	if small.CatalogMatches[0].Score <= medium.CatalogMatches[0].Score || medium.CatalogMatches[0].Score <= large.CatalogMatches[0].Score {
		t.Fatalf("expected score ordering small > medium > large, got small=%.4f medium=%.4f large=%.4f",
			small.CatalogMatches[0].Score, medium.CatalogMatches[0].Score, large.CatalogMatches[0].Score)
	}

	dynamic := runMine(t, "dynamic_new_unresolved")
	if len(dynamic.CatalogMatches) == 0 {
		t.Fatalf("expected construction match for unresolved dynamic fixture")
	}
	if dynamic.CatalogMatches[0].Score >= 0.65 {
		t.Fatalf("expected unresolved-heavy fixture to reduce score below MEDIUM, got %+v", dynamic.CatalogMatches[0])
	}
}

func TestMineGoldenOutputs(t *testing.T) {
	snapshots := []struct {
		Name    string
		Fixture string
		Args    []string
	}{
		{
			Name:    "mine_small_json.golden",
			Fixture: "construction_small_repo_clear_violation",
			Args:    []string{"mine", "--config", "archguard.yaml", "--format", "json", "--show-low-confidence"},
		},
		{
			Name:    "mine_medium_text.golden",
			Fixture: "construction_medium_repo_mixed_usage",
			Args:    []string{"mine", "--config", "archguard.yaml", "--format", "text", "--show-low-confidence"},
		},
		{
			Name:    "mine_large_json.golden",
			Fixture: "construction_large_repo_rare_violation",
			Args:    []string{"mine", "--config", "archguard.yaml", "--format", "json", "--show-low-confidence"},
		},
		{
			Name:    "mine_emit_high.golden",
			Fixture: "construction_small_repo_clear_violation",
			Args:    []string{"mine", "--config", "archguard.yaml", "--emit-config", "--adopt-catalog", "--adopt-threshold", "high", "--show-low-confidence"},
		},
		{
			Name:    "mine_emit_medium.golden",
			Fixture: "construction_small_repo_clear_violation",
			Args:    []string{"mine", "--config", "archguard.yaml", "--emit-config", "--adopt-catalog", "--adopt-threshold", "medium", "--show-low-confidence"},
		},
	}

	for _, snapshot := range snapshots {
		snapshot := snapshot
		t.Run(snapshot.Name, func(t *testing.T) {
			code, out, errOut := runCmdInDir(t, fixturePath(snapshot.Fixture), snapshot.Args)
			if code != 0 {
				t.Fatalf("expected exit 0, got %d stderr=%s", code, errOut)
			}
			assertGolden(t, snapshot.Name, out)
		})
	}
}

func TestMineDebugTextIncludesScoreDetails(t *testing.T) {
	code, out, errOut := runCmdInDir(t, fixturePath("construction_medium_repo_mixed_usage"), []string{
		"mine", "--config", "archguard.yaml", "--format", "text", "--show-low-confidence", "--debug",
	})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d stderr=%s", code, errOut)
	}
	if !strings.Contains(out, "score_components:") || !strings.Contains(out, "unresolved_reasons:") {
		t.Fatalf("expected debug output to include score/evidence details, got: %s", out)
	}
}

func TestMineWorkspaceAutoDiscovery(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "package.json"), `{"workspaces":["apps/*"]}`)
	mustWriteFile(t, filepath.Join(dir, "apps", "web", "package.json"), `{"name":"web"}`)
	mustWriteFile(t, filepath.Join(dir, "apps", "api", "package.json"), `{"name":"api"}`)
	mustWriteFile(t, filepath.Join(dir, "apps", "web", "src", "index.ts"), `export const web = 1`)
	mustWriteFile(t, filepath.Join(dir, "apps", "api", "src", "index.ts"), `export const api = 1`)

	code, _, errOut := runCmdInDir(t, dir, []string{
		"mine", "--format", "json", "--catalog", "off", "--debug",
	})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d stderr=%s", code, errOut)
	}
	if !strings.Contains(errOut, "mine workspaces: 2 (auto_workspaces)") {
		t.Fatalf("expected auto workspace debug line, got: %s", errOut)
	}
}

func TestMineWorkspaceModeOff(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "package.json"), `{"workspaces":["apps/*"]}`)
	mustWriteFile(t, filepath.Join(dir, "apps", "web", "package.json"), `{"name":"web"}`)
	mustWriteFile(t, filepath.Join(dir, "apps", "web", "src", "index.ts"), `export const web = 1`)

	code, _, errOut := runCmdInDir(t, dir, []string{
		"mine", "--format", "json", "--catalog", "off", "--debug", "--workspace-mode", "off",
	})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d stderr=%s", code, errOut)
	}
	if !strings.Contains(errOut, "mine workspaces: 1 (off)") {
		t.Fatalf("expected workspace-mode off debug line, got: %s", errOut)
	}
}

func runCmdInDir(t *testing.T, dir string, args []string) (int, string, string) {
	t.Helper()
	cwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(cwd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	oldOut := os.Stdout
	oldErr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	code := execute(args)

	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout = oldOut
	os.Stderr = oldErr

	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	_, _ = outBuf.ReadFrom(rOut)
	_, _ = errBuf.ReadFrom(rErr)
	_ = rOut.Close()
	_ = rErr.Close()

	return code, outBuf.String(), errBuf.String()
}

func runCmdInDirWithInput(t *testing.T, dir string, args []string, input string) (int, string, string) {
	t.Helper()
	cwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(cwd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	oldOut := os.Stdout
	oldErr := os.Stderr
	oldIn := os.Stdin
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	rIn, wIn, _ := os.Pipe()
	if _, err := wIn.Write([]byte(input)); err != nil {
		t.Fatalf("stdin write failed: %v", err)
	}
	_ = wIn.Close()

	os.Stdout = wOut
	os.Stderr = wErr
	os.Stdin = rIn

	code := execute(args)

	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout = oldOut
	os.Stderr = oldErr
	os.Stdin = oldIn
	_ = rIn.Close()

	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	_, _ = outBuf.ReadFrom(rOut)
	_, _ = errBuf.ReadFrom(rErr)
	_ = rOut.Close()
	_ = rErr.Close()

	return code, outBuf.String(), errBuf.String()
}

func fixturePath(name string) string {
	return filepath.Join("..", "..", "fixtures", name)
}

func assertGolden(t *testing.T, name, actual string) {
	t.Helper()
	expectedPath := filepath.Join("testdata", name)
	expectedBytes, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("failed to read golden file %s: %v", expectedPath, err)
	}
	expected := string(expectedBytes)
	if actual != expected {
		t.Fatalf("golden mismatch for %s\nexpected:\n%s\nactual:\n%s", name, expected, actual)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustRunGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(out))
	}
	return string(out)
}
