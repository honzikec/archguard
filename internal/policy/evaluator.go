package policy

import (
	"github.com/honzikec/archguard/internal/config"
	"github.com/honzikec/archguard/internal/model"
	"github.com/honzikec/archguard/internal/pathutil"
)

// Evaluate applies configuration rules against the provided imports and files and returns any violations.
func Evaluate(cfg *config.Config, imports []model.ImportRef, files []string) []model.Finding {
	var findings []model.Finding

	for _, rule := range cfg.Rules {
		if rule.Kind == KindImportBoundary || rule.Kind == KindBannedPackage {
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
				} else if matchesBannedPackageRule(rule, imp) {
					finding := model.Finding{
						RuleID:    rule.ID,
						RuleKind:  rule.Kind,
						Severity:  rule.Severity,
						Message:   rule.ID + " Banned package violation",
						Rationale: rule.Rationale,
						FilePath:  imp.SourceFile,
						Line:      imp.Line,
						Column:    imp.Column,
						RawImport: imp.RawImport,
					}
					findings = append(findings, finding)
				}
			}
		} else if rule.Kind == KindFileConvention {
			for _, file := range files {
				if matchesFileConventionRule(rule, file) {
					finding := model.Finding{
						RuleID:    rule.ID,
						RuleKind:  rule.Kind,
						Severity:  rule.Severity,
						Message:   rule.ID + " File convention violation",
						Rationale: rule.Rationale,
						FilePath:  file,
						Line:      1,
						Column:    1,
						RawImport: "",
					}
					findings = append(findings, finding)
				}
			}
		}
	}

	return findings
}
