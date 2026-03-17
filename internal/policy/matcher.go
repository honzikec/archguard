package policy

import (
	"github.com/honzikec/archguard/internal/config"
	"github.com/honzikec/archguard/internal/model"
	"github.com/honzikec/archguard/internal/pathutil"
)

func matchesBoundaryRule(rule config.Rule, imp model.ImportRef) bool {
	if rule.Kind != "import_boundary" {
		return false
	}

	// Check if the source file matches ANY from_paths
	fromMatch := false
	for _, pattern := range rule.Conditions.FromPaths {
		if pathutil.MatchGlob(pattern, imp.SourceFile) {
			fromMatch = true
			break
		}
	}
	if !fromMatch {
		return false
	}

	// Check if the resolved import path matches ANY forbidden_paths
	for _, pattern := range rule.Conditions.ForbiddenPaths {
		if pathutil.MatchGlob(pattern, imp.ResolvedPath) {
			return true
		}
	}
	return false
}
