package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/honzikec/archguard/internal/config"
)

func TestInitFromBriefGeneratesConfig(t *testing.T) {
	dir := t.TempDir()
	briefPath := filepath.Join(dir, "architecture-brief.yaml")
	outPath := filepath.Join(dir, "archguard.generated.yaml")
	mustWriteFile(t, briefPath, `version: 1
layers:
  - id: domain
    paths: ["src/domain/**"]
  - id: infra
    paths: ["src/infra/**"]
policies:
  - id: AG-DOMAIN-NO-INFRA
    type: deny_import
    severity: error
    from: ["layer:domain"]
    to: ["layer:infra"]
`)

	code, out, errOut := runCmdInDir(t, dir, []string{
		"init",
		"--from-brief", filepath.Base(briefPath),
		"--out", filepath.Base(outPath),
	})
	if code != 0 {
		t.Fatalf("expected init --from-brief exit 0, got %d stderr=%s", code, errOut)
	}
	if !strings.Contains(out, "Created "+filepath.Base(outPath)+" from brief") {
		t.Fatalf("expected success output, got %s", out)
	}

	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("expected generated config to exist: %v", err)
	}
	cfg, err := config.Load(outPath)
	if err != nil {
		t.Fatalf("expected generated config to be loadable: %v", err)
	}
	if len(cfg.Rules) != 1 || cfg.Rules[0].Kind != "no_import" {
		t.Fatalf("unexpected compiled config rules: %+v", cfg.Rules)
	}
}

func TestInitFromBriefOutRequiresFromBrief(t *testing.T) {
	dir := t.TempDir()
	code, _, errOut := runCmdInDir(t, dir, []string{
		"init",
		"--out", "archguard.generated.yaml",
	})
	if code != 2 {
		t.Fatalf("expected exit 2, got %d stderr=%s", code, errOut)
	}
	if !strings.Contains(errOut, "--out requires --from-brief") {
		t.Fatalf("expected --out requires error, got: %s", errOut)
	}
}
