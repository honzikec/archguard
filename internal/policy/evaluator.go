package policy

import (
	"github.com/honzikec/archguard/internal/config"
	"github.com/honzikec/archguard/internal/model"
	"github.com/honzikec/archguard/internal/pathutil"
)

// Evaluate applies configuration rules against the provided imports and returns any violations.
func Evaluate(cfg *config.Config, imports []model.ImportRef) []model.Finding {
	var findings []model.Finding

	for _, rule := range cfg.Rules {
		for _, imp := range imports {
			// Resolve the path if needed
			if imp.ResolvedPath == "" && !imp.IsPackageImport {
				imp.ResolvedPath = pathutil.ResolveImport(imp.SourceFile, imp.RawImport)
			}

			if matchesBoundaryRule(rule, imp) {
				finding := model.Finding{
					RuleID:    rule.ID,
					RuleKind:  rule.Kind,
					Severity:  rule.Severity,
					Message:   rule.ID + " Import boundary violation",
					Rationale: rule.Rationale,
					FilePath:  imp.SourceFile,
					Line:      imp.Line,
					Column:    imp.Column,
					RawImport: imp.RawImport,
				}
				findings = append(findings, finding)
			}
		}
	}

	return findings
}
