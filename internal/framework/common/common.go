package common

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func RootsOrDefault(roots []string) []string {
	if len(roots) == 0 {
		return []string{"."}
	}
	out := make([]string, 0, len(roots))
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		out = append(out, root)
	}
	if len(out) == 0 {
		return []string{"."}
	}
	return out
}

func HasAnyFileNamed(roots []string, names []string) bool {
	roots = RootsOrDefault(roots)
	for _, root := range roots {
		for _, name := range names {
			if _, err := os.Stat(filepath.Join(root, name)); err == nil {
				return true
			}
		}
	}
	return false
}

func HasAnyDirectory(roots []string, relDirs []string) bool {
	roots = RootsOrDefault(roots)
	for _, root := range roots {
		for _, rel := range relDirs {
			p := filepath.Join(root, rel)
			st, err := os.Stat(p)
			if err == nil && st.IsDir() {
				return true
			}
		}
	}
	return false
}

func HasPackageDependency(roots []string, dependencies []string) bool {
	roots = RootsOrDefault(roots)
	packageFiles := make([]string, 0)
	for _, root := range roots {
		prefixes := []string{
			root,
			filepath.Join(root, "*"),
			filepath.Join(root, "*", "*"),
		}
		for _, prefix := range prefixes {
			matches, err := filepath.Glob(filepath.Join(prefix, "package.json"))
			if err != nil {
				continue
			}
			packageFiles = append(packageFiles, matches...)
		}
	}
	for _, pkgPath := range packageFiles {
		content, err := os.ReadFile(pkgPath)
		if err != nil {
			continue
		}
		text := string(content)
		for _, dep := range dependencies {
			if strings.Contains(text, "\""+dep+"\"") {
				return true
			}
		}
	}
	return false
}

func HasFileWithSuffix(roots []string, suffixes []string, maxDepth int) bool {
	roots = RootsOrDefault(roots)
	for _, root := range roots {
		if hasSuffixByScan(filepath.Clean(root), suffixes, maxDepth) {
			return true
		}
	}
	return false
}

func hasSuffixByScan(root string, suffixes []string, maxDepth int) bool {
	found := false
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, relErr := filepath.Rel(root, p)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		depth := 0
		if rel != "." {
			depth = strings.Count(rel, "/") + 1
		}
		if d.IsDir() {
			base := d.Name()
			if base == "node_modules" || strings.HasPrefix(base, ".") {
				return filepath.SkipDir
			}
			if maxDepth > 0 && depth > maxDepth {
				return filepath.SkipDir
			}
			return nil
		}
		for _, suffix := range suffixes {
			if strings.HasSuffix(d.Name(), suffix) {
				found = true
				return fs.SkipAll
			}
		}
		return nil
	})
	return found
}
