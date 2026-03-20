package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/honzikec/archguard/internal/pathutil"
)

func DiscoverRoots(projectRoots []string) ([]string, error) {
	if len(projectRoots) == 0 {
		projectRoots = []string{"."}
	}

	absCWD, err := filepath.Abs(".")
	if err != nil {
		return nil, err
	}
	resolvedCWD, err := filepath.EvalSymlinks(absCWD)
	if err != nil {
		resolvedCWD = absCWD
	}

	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, root := range projectRoots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		absRoot, err := filepath.Abs(root)
		if err != nil {
			return nil, err
		}
		workspaceRoots, err := discoverWithinRoot(absRoot)
		if err != nil {
			return nil, err
		}
		for _, ws := range workspaceRoots {
			rel := relativeToCWD(ws, absCWD, resolvedCWD)
			rel = pathutil.Normalize(rel)
			if _, ok := seen[rel]; ok {
				continue
			}
			seen[rel] = struct{}{}
			out = append(out, rel)
		}
	}
	sort.Strings(out)
	return out, nil
}

func relativeToCWD(path, absCWD, resolvedCWD string) string {
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	resolvedPath, err := filepath.EvalSymlinks(pathAbs)
	if err != nil {
		resolvedPath = pathAbs
	}
	if rel, err := filepath.Rel(resolvedCWD, resolvedPath); err == nil {
		normalized := pathutil.Normalize(rel)
		if normalized == "" {
			return "."
		}
		return normalized
	}
	if rel, err := filepath.Rel(absCWD, pathAbs); err == nil {
		normalized := pathutil.Normalize(rel)
		if normalized == "" {
			return "."
		}
		return normalized
	}
	return pathutil.Normalize(pathAbs)
}

func discoverWithinRoot(absRoot string) ([]string, error) {
	patterns, err := discoverWorkspacePatterns(absRoot)
	if err != nil {
		return nil, err
	}
	if len(patterns) == 0 {
		return []string{absRoot}, nil
	}

	roots := make([]string, 0)
	seen := map[string]struct{}{}
	for _, pattern := range patterns {
		matches, err := expandWorkspacePattern(absRoot, pattern)
		if err != nil {
			return nil, err
		}
		for _, m := range matches {
			if _, ok := seen[m]; ok {
				continue
			}
			seen[m] = struct{}{}
			roots = append(roots, m)
		}
	}
	if len(roots) == 0 {
		return []string{absRoot}, nil
	}
	sort.Strings(roots)
	return roots, nil
}

func discoverWorkspacePatterns(absRoot string) ([]string, error) {
	if patterns, ok, err := discoverFromPackageJSON(absRoot); err != nil {
		return nil, err
	} else if ok && len(patterns) > 0 {
		return patterns, nil
	}

	if patterns, ok, err := discoverFromPNPM(absRoot); err != nil {
		return nil, err
	} else if ok && len(patterns) > 0 {
		return patterns, nil
	}

	if hasFile(absRoot, "nx.json") || hasFile(absRoot, "turbo.json") {
		return []string{"apps/*", "packages/*"}, nil
	}
	return nil, nil
}

func discoverFromPackageJSON(absRoot string) ([]string, bool, error) {
	path := filepath.Join(absRoot, "package.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}

	var decoded struct {
		Workspaces json.RawMessage `json:"workspaces"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, true, err
	}
	if len(decoded.Workspaces) == 0 {
		return nil, true, nil
	}

	var direct []string
	if err := json.Unmarshal(decoded.Workspaces, &direct); err == nil {
		return normalizePatterns(direct), true, nil
	}

	var object struct {
		Packages []string `json:"packages"`
	}
	if err := json.Unmarshal(decoded.Workspaces, &object); err == nil {
		return normalizePatterns(object.Packages), true, nil
	}
	return nil, true, nil
}

func discoverFromPNPM(absRoot string) ([]string, bool, error) {
	path := filepath.Join(absRoot, "pnpm-workspace.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	lines := strings.Split(string(data), "\n")
	patterns := make([]string, 0)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "-") {
			continue
		}
		p := strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
		p = strings.Trim(p, `"'`)
		if p == "" {
			continue
		}
		patterns = append(patterns, p)
	}
	return normalizePatterns(patterns), true, nil
}

func normalizePatterns(patterns []string) []string {
	out := make([]string, 0, len(patterns))
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, strings.TrimPrefix(pathutil.Normalize(p), "./"))
	}
	return out
}

func expandWorkspacePattern(absRoot, pattern string) ([]string, error) {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return nil, nil
	}
	if strings.Contains(pattern, "**") {
		prefix := strings.Split(pattern, "**")[0]
		base := filepath.Join(absRoot, filepath.FromSlash(prefix))
		return collectPackageDirs(base, 6), nil
	}

	fullPattern := filepath.Join(absRoot, filepath.FromSlash(pattern))
	matches, err := filepath.Glob(fullPattern)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		info, err := os.Stat(m)
		if err != nil {
			continue
		}
		dir := m
		if !info.IsDir() {
			dir = filepath.Dir(m)
		}
		if hasFile(dir, "package.json") {
			out = append(out, filepath.Clean(dir))
		}
	}
	sort.Strings(out)
	return out, nil
}

func collectPackageDirs(base string, maxDepth int) []string {
	base = filepath.Clean(base)
	info, err := os.Stat(base)
	if err != nil || !info.IsDir() {
		return nil
	}

	out := make([]string, 0)
	seen := map[string]struct{}{}
	_ = filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, relErr := filepath.Rel(base, path)
		if relErr != nil {
			return nil
		}
		depth := 0
		if rel != "." {
			depth = strings.Count(pathutil.Normalize(rel), "/") + 1
		}
		if d.IsDir() {
			name := d.Name()
			if name == "node_modules" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			if maxDepth > 0 && depth > maxDepth {
				return filepath.SkipDir
			}
			if hasFile(path, "package.json") {
				p := filepath.Clean(path)
				if _, ok := seen[p]; !ok {
					seen[p] = struct{}{}
					out = append(out, p)
				}
			}
		}
		return nil
	})
	sort.Strings(out)
	return out
}

func hasFile(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}
