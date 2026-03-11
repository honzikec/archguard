package fileset

import (
	"os"
	"path/filepath"
)

func Discover(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if shouldIgnoreDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if isSupportedFile(path) {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}
