package fileset

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/honzikec/archguard/internal/config"
	"github.com/honzikec/archguard/internal/pathutil"
)

func Discover(project config.ProjectSettings) ([]string, error) {
	seen := map[string]struct{}{}
	files := make([]string, 0)

	for _, root := range project.Roots {
		root = filepath.Clean(root)
		if root == "" {
			continue
		}
		if _, err := os.Stat(root); err != nil {
			return nil, fmt.Errorf("project root not found: %s", root)
		}
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			normalized := pathutil.Normalize(path)
			if d.IsDir() {
				if shouldSkipDir(normalized, project.Exclude) {
					return filepath.SkipDir
				}
				return nil
			}
			if !isSupportedFile(normalized) {
				return nil
			}

			rel, err := filepath.Rel(".", path)
			if err != nil {
				return err
			}
			rel = pathutil.Normalize(rel)
			if !pathutil.MatchAny(project.Include, rel) {
				return nil
			}
			if pathutil.MatchAny(project.Exclude, rel) {
				return nil
			}
			if _, ok := seen[rel]; ok {
				return nil
			}
			seen[rel] = struct{}{}
			files = append(files, rel)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	sort.Strings(files)
	return files, nil
}
