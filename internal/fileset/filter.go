package fileset

import (
	"strings"
)

func shouldIgnoreDir(name string) bool {
	// Ignore directories listed in PROJECT_PLAN.md
	if name == "node_modules" ||
		name == "dist" ||
		name == "build" ||
		name == ".next" ||
		name == "coverage" {
		return true
	}
	// Ignore directories starting with '.' unless it's '.' itself
	if strings.HasPrefix(name, ".") && name != "." {
		return true
	}
	return false
}

func isSupportedFile(path string) bool {
	// Ignore generated files
	if strings.Contains(path, ".gen.") {
		return false
	}
	// Support specified extensions
	return strings.HasSuffix(path, ".ts") ||
		strings.HasSuffix(path, ".tsx") ||
		strings.HasSuffix(path, ".js") ||
		strings.HasSuffix(path, ".jsx")
}
