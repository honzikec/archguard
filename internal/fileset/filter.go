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
