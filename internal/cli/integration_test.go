package cli

import (
	"bytes"
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

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
