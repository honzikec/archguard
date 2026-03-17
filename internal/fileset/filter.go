package fileset

import (
	"path"
	"strings"

	"github.com/honzikec/archguard/internal/pathutil"
)

func shouldSkipDir(dirPath string, excludePatterns []string) bool {
	base := path.Base(dirPath)
	if strings.HasPrefix(base, ".") && base != "." {
		return true
	}
	if pathutil.MatchAny(excludePatterns, dirPath) {
		return true
	}
	return false
}

func isSupportedFile(path string) bool {
	if strings.Contains(path, ".gen.") {
		return false
	}
	for _, ext := range []string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs"} {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}
