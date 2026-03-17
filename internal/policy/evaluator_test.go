package policy_test

import (
	"testing"

	"github.com/honzikec/archguard/internal/config"
	"github.com/honzikec/archguard/internal/model"
	"github.com/honzikec/archguard/internal/policy"
)

func TestEvaluate_LayeredApp(t *testing.T) {
	cfg := &config.Config{
		Rules: []config.Rule{
			{
				ID:   "AG-001",
				Kind: policy.KindImportBoundary,
				Conditions: config.Conditions{
					FromPaths:      []string{"src/domain/**"},
					ForbiddenPaths: []string{"src/infra/**"},
				},
			},
		},
	}

	// Should not find a violation if domain doesn't import infra
	importsOk := []model.ImportRef{
		{SourceFile: "src/domain/user.ts", RawImport: "lodash", IsPackageImport: true},
	}

	findings := policy.Evaluate(cfg, importsOk, nil)
	if len(findings) > 0 {
		t.Errorf("expected 0 findings, got %d", len(findings))
	}

	// Should find a violation if domain imports infra
	importsViolate := []model.ImportRef{
		{SourceFile: "src/domain/user.ts", RawImport: "../infra/db.ts", ResolvedPath: "src/infra/db.ts"},
	}

	findings = policy.Evaluate(cfg, importsViolate, nil)
	if len(findings) != 1 {
		t.Errorf("expected 1 finding, got %d", len(findings))
	}
}
