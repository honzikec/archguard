package policy

import (
	"regexp"

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

func matchesBannedPackageRule(rule config.Rule, imp model.ImportRef) bool {
	if rule.Kind != "banned_package" || !imp.IsPackageImport {
		return false
	}

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

	for _, pkg := range rule.Conditions.ForbiddenPackages {
		if imp.RawImport == pkg {
			return true
		}
	}
	return false
}

func matchesFileConventionRule(rule config.Rule, file string) bool {
	if rule.Kind != "file_convention" {
		return false
	}

	pathMatch := false
	for _, pattern := range rule.Conditions.PathPatterns {
		if pathutil.MatchGlob(pattern, file) {
			pathMatch = true
			break
		}
	}
	if !pathMatch {
		return false
	}

	if rule.Conditions.FilenameRegex != "" {
		matched, _ := regexp.MatchString(rule.Conditions.FilenameRegex, file)
		return !matched // violation if it does NOT match the regex
	}
	return false
}
