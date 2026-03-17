package pathutil

import (
	"path/filepath"
	"strings"
)

// ResolveImport attempts to resolve the absolute or project-relative path of an import statement.
func ResolveImport(sourceFile, rawImport string) string {
	if strings.HasPrefix(rawImport, ".") {
		dir := filepath.Dir(sourceFile)
		return filepath.Clean(filepath.Join(dir, rawImport))
	}
	return rawImport
}
