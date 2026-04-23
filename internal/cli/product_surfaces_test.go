package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/honzikec/archguard/internal/baseline"
	"github.com/honzikec/archguard/internal/config"
)

func TestCheckJSONIncludesFindingMetadata(t *testing.T) {
	code, out, errOut := runCmdInDir(t, fixturePath("root_boundary_fail"), []string{
		"check", "--config", "archguard.yaml", "--format", "json",
	})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d stderr=%s output=%s", code, errOut, out)
	}

	var payload struct {
		Findings []struct {
			RuleID        string `json:"rule_id"`
			Evidence      string `json:"evidence"`
			Remediation   string `json:"remediation"`
			MatchedScope  string `json:"matched_scope"`
			MatchedTarget string `json:"matched_target"`
		} `json:"findings"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("failed to decode check output: %v", err)
	}
	if len(payload.Findings) == 0 {
		t.Fatal("expected at least one finding")
	}
	finding := payload.Findings[0]
	if finding.RuleID != "AG-NO-INFRA-IN-DOMAIN" {
		t.Fatalf("unexpected rule id: %+v", finding)
	}
	if finding.Evidence == "" || finding.Remediation == "" || finding.MatchedScope == "" || finding.MatchedTarget == "" {
		t.Fatalf("expected enriched finding metadata, got %+v", finding)
	}
}

func TestExplainFindingUsesCurrentFingerprint(t *testing.T) {
	code, out, errOut := runCmdInDir(t, fixturePath("root_boundary_fail"), []string{
		"check", "--config", "archguard.yaml", "--format", "json",
	})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d stderr=%s output=%s", code, errOut, out)
	}

	var payload struct {
		Findings []struct {
			Fingerprint string `json:"fingerprint"`
		} `json:"findings"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("failed to decode check output: %v", err)
	}
	if len(payload.Findings) == 0 || payload.Findings[0].Fingerprint == "" {
		t.Fatalf("expected finding fingerprint, got %+v", payload.Findings)
	}

	code, out, errOut = runCmdInDir(t, fixturePath("root_boundary_fail"), []string{
		"explain", "--config", "archguard.yaml", "--finding", payload.Findings[0].Fingerprint,
	})
	if code != 0 {
		t.Fatalf("expected explain exit 0, got %d stderr=%s output=%s", code, errOut, out)
	}
	if !strings.Contains(out, "Matched scope:") || !strings.Contains(out, "Evidence:") || !strings.Contains(out, "How to fix:") {
		t.Fatalf("expected enriched explain output, got: %s", out)
	}
}

func TestInitGuidedWritesConfigAndBaseline(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "src", "domain", "user.ts"), `import { db } from "../infra/db"
export const user = db
`)
	mustWriteFile(t, filepath.Join(dir, "src", "domain", "profile.ts"), `export const profile = "ok"`)
	mustWriteFile(t, filepath.Join(dir, "src", "domain", "account.ts"), `export const account = "ok"`)
	mustWriteFile(t, filepath.Join(dir, "src", "domain", "session.ts"), `export const session = "ok"`)
	mustWriteFile(t, filepath.Join(dir, "src", "infra", "db.ts"), `export const db = 1`)

	code, out, errOut := runCmdInDir(t, dir, []string{
		"init", "--guided", "--write-config", "--write-baseline",
	})
	if code != 0 {
		t.Fatalf("expected guided init exit 0, got %d stderr=%s output=%s", code, errOut, out)
	}
	if !strings.Contains(out, "Guided init summary") || !strings.Contains(out, "GitHub Action:") {
		t.Fatalf("expected guided init summary, got: %s", out)
	}

	cfg, err := config.Load(filepath.Join(dir, "archguard.yaml"))
	if err != nil {
		t.Fatalf("expected guided config to load: %v", err)
	}
	if len(cfg.Rules) == 0 {
		t.Fatal("expected guided config to include rules")
	}

	entries, err := baseline.Load(filepath.Join(dir, "archguard-baseline.json"))
	if err != nil {
		t.Fatalf("expected guided baseline to load: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one baseline entry")
	}
}

func TestInitGuidedMonorepoRecommendsWorkspaceRoots(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "package.json"), `{"workspaces":["apps/*","packages/*"]}`)
	mustWriteFile(t, filepath.Join(dir, "apps", "web", "package.json"), `{"name":"web","dependencies":{"react":"18.0.0"}}`)
	mustWriteFile(t, filepath.Join(dir, "apps", "web", "src", "page.ts"), `import { Button } from "@acme/ui/button"
export const page = Button
`)
	mustWriteFile(t, filepath.Join(dir, "packages", "ui", "package.json"), `{"name":"@acme/ui","exports":{"./button":"./src/button.ts"}}`)
	mustWriteFile(t, filepath.Join(dir, "packages", "ui", "src", "button.ts"), `export const Button = 1`)

	code, out, errOut := runCmdInDir(t, dir, []string{"init", "--guided"})
	if code != 0 {
		t.Fatalf("expected guided init exit 0, got %d stderr=%s output=%s", code, errOut, out)
	}
	if !strings.Contains(out, "Preset: React Shared Packages") {
		t.Fatalf("expected workspace preset recommendation, got: %s", out)
	}
	if !strings.Contains(out, "apps/web") || !strings.Contains(out, "packages/ui") {
		t.Fatalf("expected workspace roots in output, got: %s", out)
	}
}

func TestActionRunScriptCapturesExitCodeAndSarif(t *testing.T) {
	dir := t.TempDir()
	fakeBinary := filepath.Join(dir, "archguard")
	mustWriteFile(t, fakeBinary, `#!/usr/bin/env bash
echo '{"runs":[]}'
exit 1
`)
	if err := os.Chmod(fakeBinary, 0o755); err != nil {
		t.Fatal(err)
	}
	outputPath := filepath.Join(dir, "github-output.txt")
	cmd := exec.Command("bash", filepath.Join("scripts", "run-archguard-action.sh"))
	cmd.Dir = filepath.Join("..", "..")
	cmd.Env = append(os.Environ(),
		"ARCHGUARD_BINARY="+fakeBinary,
		"INPUT_FORMAT=sarif",
		"INPUT_CONFIG=archguard.yaml",
		"INPUT_PARSE_ERROR_POLICY=error",
		"INPUT_SEVERITY_THRESHOLD=error",
		"GITHUB_OUTPUT="+outputPath,
		"RUNNER_TEMP="+dir,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected wrapper script to exit 0, got %v\n%s", err, string(output))
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "exit_code=1") || !strings.Contains(string(data), "sarif_file=") {
		t.Fatalf("expected exit code and sarif file outputs, got: %s", string(data))
	}
	sarifPath := parseGithubOutputValue(t, string(data), "sarif_file")
	sarifBytes, err := os.ReadFile(sarifPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(sarifBytes, []byte(`{"runs":[]}`)) {
		t.Fatalf("expected SARIF content to be written, got: %s", string(sarifBytes))
	}
}

func TestNPMWrapperPreservesExitCodeAndOutput(t *testing.T) {
	dir := t.TempDir()
	fakeBinary := filepath.Join(dir, "archguard")
	mustWriteFile(t, fakeBinary, `#!/usr/bin/env bash
echo "stdout:$*"
echo "stderr:$*" >&2
exit 17
`)
	if err := os.Chmod(fakeBinary, 0o755); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("node", filepath.Join("npm", "bin", "archguard.js"), "check", "--config", "demo.yaml")
	cmd.Dir = filepath.Join("..", "..")
	cmd.Env = append(os.Environ(), "ARCHGUARD_BINARY="+fakeBinary)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected exit error, got %v", err)
	}
	if exitErr.ExitCode() != 17 {
		t.Fatalf("expected exit code 17, got %d", exitErr.ExitCode())
	}
	if !strings.Contains(stdout.String(), "stdout:check --config demo.yaml") {
		t.Fatalf("expected stdout passthrough, got: %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "stderr:check --config demo.yaml") {
		t.Fatalf("expected stderr passthrough, got: %s", stderr.String())
	}
}

func parseGithubOutputValue(t *testing.T, output, key string) string {
	t.Helper()
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, key+"=") {
			return strings.TrimPrefix(line, key+"=")
		}
	}
	t.Fatalf("missing %s in output %q", key, output)
	return ""
}
