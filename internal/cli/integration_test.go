package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
