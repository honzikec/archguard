package pathutil

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/honzikec/archguard/internal/config"
	"gopkg.in/yaml.v3"
)

type jsPackage struct {
	Name     string
	Dir      string
	RelDir   string
	Main     string
	Module   string
	Types    string
	TSConfig string
	Exports  json.RawMessage
	Imports  map[string]json.RawMessage
}

type packageJSONFile struct {
	Name       string                     `json:"name"`
	Workspaces json.RawMessage            `json:"workspaces"`
	Main       string                     `json:"main"`
	Module     string                     `json:"module"`
	Types      string                     `json:"types"`
	Typings    string                     `json:"typings"`
	TSConfig   string                     `json:"tsconfig"`
	Exports    json.RawMessage            `json:"exports"`
	Imports    map[string]json.RawMessage `json:"imports"`
}

func (r *Resolver) tsConfigPackageDirs(baseDir, packageName string) []string {
	out := make([]string, 0, 2)
	seen := map[string]struct{}{}
	if pkg, ok := r.jsPackages[packageName]; ok {
		out = append(out, pkg.Dir)
		seen[pkg.Dir] = struct{}{}
	}
	for _, dir := range findNodeModulePackageDirs(baseDir, r.root, packageName) {
		if _, ok := seen[dir]; ok {
			continue
		}
		seen[dir] = struct{}{}
		out = append(out, dir)
	}
	return out
}

func tsConfigExtendsCandidates(packageDir, subpath, packageTSConfig string) []string {
	out := make([]string, 0)
	if subpath == "" {
		if strings.TrimSpace(packageTSConfig) != "" {
			out = append(out, filepath.Join(packageDir, packageTSConfig))
		}
		out = append(out, filepath.Join(packageDir, "tsconfig.json"))
		return out
	}
	base := filepath.Join(packageDir, filepath.FromSlash(subpath))
	out = append(out, base)
	if !strings.HasSuffix(strings.ToLower(base), ".json") {
		out = append(out, base+".json")
	}
	out = append(out, filepath.Join(base, "tsconfig.json"))
	return out
}

func readPackageTSConfigField(packageDir string) string {
	data, err := os.ReadFile(filepath.Join(packageDir, "package.json"))
	if err != nil {
		return ""
	}
	var decoded packageJSONFile
	if err := json.Unmarshal(data, &decoded); err != nil {
		return ""
	}
	return strings.TrimSpace(decoded.TSConfig)
}

func findNodeModulePackageDirs(baseDir, root, packageName string) []string {
	baseDir = filepath.Clean(baseDir)
	root = filepath.Clean(root)
	out := make([]string, 0)
	seen := map[string]struct{}{}
	for dir := baseDir; ; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, "node_modules", filepath.FromSlash(packageName))
		if fi, err := os.Stat(candidate); err == nil && fi.IsDir() {
			candidate = filepath.Clean(candidate)
			if _, ok := seen[candidate]; !ok {
				seen[candidate] = struct{}{}
				out = append(out, candidate)
			}
		}
		if sameCleanPath(dir, root) || dir == filepath.Dir(dir) {
			break
		}
	}
	return out
}

func sameCleanPath(left, right string) bool {
	return filepath.Clean(left) == filepath.Clean(right)
}

func (r *Resolver) loadJSPackageMappings(project config.ProjectSettings) error {
	dirs := make([]string, 0)
	addDir := func(dir string) {
		dir = filepath.Clean(dir)
		for _, existing := range dirs {
			if sameCleanPath(existing, dir) {
				return
			}
		}
		dirs = append(dirs, dir)
	}

	if hasFile(r.root, "package.json") {
		addDir(r.root)
	}
	workspaceDirs, err := discoverJSPackageWorkspaceDirs(r.root)
	if err != nil {
		return err
	}
	for _, dir := range workspaceDirs {
		addDir(dir)
	}
	for _, root := range project.Roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		projectRoot := root
		if !filepath.IsAbs(projectRoot) {
			projectRoot = filepath.Join(r.root, projectRoot)
		}
		if packageDir, ok := nearestPackageDir(projectRoot, r.root); ok {
			addDir(packageDir)
		}
	}

	sort.Strings(dirs)
	for _, dir := range dirs {
		if err := r.loadJSPackageFile(dir); err != nil {
			return err
		}
	}
	sort.Slice(r.jsPackageRoots, func(i, j int) bool {
		return len(r.jsPackageRoots[i].RelDir) > len(r.jsPackageRoots[j].RelDir)
	})
	return nil
}

func (r *Resolver) loadJSPackageFile(dir string) error {
	path := filepath.Join(dir, "package.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", path, err)
	}
	var decoded packageJSONFile
	if err := json.Unmarshal(data, &decoded); err != nil {
		return fmt.Errorf("failed to parse %s: %w", path, err)
	}
	rel, err := filepath.Rel(r.root, dir)
	if err != nil {
		rel = dir
	}
	rel = Normalize(rel)
	if rel == "" {
		rel = "."
	}
	types := strings.TrimSpace(decoded.Types)
	if types == "" {
		types = strings.TrimSpace(decoded.Typings)
	}
	pkg := jsPackage{
		Name:     strings.TrimSpace(decoded.Name),
		Dir:      filepath.Clean(dir),
		RelDir:   rel,
		Main:     strings.TrimSpace(decoded.Main),
		Module:   strings.TrimSpace(decoded.Module),
		Types:    types,
		TSConfig: strings.TrimSpace(decoded.TSConfig),
		Exports:  decoded.Exports,
		Imports:  decoded.Imports,
	}
	r.jsPackageRoots = append(r.jsPackageRoots, pkg)
	if pkg.Name != "" {
		if _, exists := r.jsPackages[pkg.Name]; !exists {
			r.jsPackages[pkg.Name] = pkg
		}
	}
	return nil
}

func discoverJSPackageWorkspaceDirs(root string) ([]string, error) {
	patterns, err := discoverJSPackageWorkspacePatterns(root)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0)
	seen := map[string]struct{}{}
	for _, pattern := range patterns {
		matches, err := expandJSPackageWorkspacePattern(root, pattern)
		if err != nil {
			return nil, err
		}
		for _, match := range matches {
			if _, ok := seen[match]; ok {
				continue
			}
			seen[match] = struct{}{}
			out = append(out, match)
		}
	}
	sort.Strings(out)
	return out, nil
}

func discoverJSPackageWorkspacePatterns(root string) ([]string, error) {
	if patterns, ok, err := discoverJSPackageWorkspaces(root); err != nil {
		return nil, err
	} else if ok && len(patterns) > 0 {
		return patterns, nil
	}
	if patterns, ok, err := discoverPNPMWorkspaces(root); err != nil {
		return nil, err
	} else if ok && len(patterns) > 0 {
		return patterns, nil
	}
	if hasFile(root, "nx.json") || hasFile(root, "turbo.json") {
		return []string{"apps/*", "packages/*"}, nil
	}
	return nil, nil
}

func discoverJSPackageWorkspaces(root string) ([]string, bool, error) {
	path := filepath.Join(root, "package.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	var decoded packageJSONFile
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, true, err
	}
	if len(decoded.Workspaces) == 0 {
		return nil, true, nil
	}
	var direct []string
	if err := json.Unmarshal(decoded.Workspaces, &direct); err == nil {
		return normalizeWorkspacePatterns(direct), true, nil
	}
	var object struct {
		Packages []string `json:"packages"`
	}
	if err := json.Unmarshal(decoded.Workspaces, &object); err == nil {
		return normalizeWorkspacePatterns(object.Packages), true, nil
	}
	return nil, true, nil
}

func discoverPNPMWorkspaces(root string) ([]string, bool, error) {
	path := filepath.Join(root, "pnpm-workspace.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	var decoded struct {
		Packages []string `yaml:"packages"`
	}
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		return nil, true, err
	}
	return normalizeWorkspacePatterns(decoded.Packages), true, nil
}

func normalizeWorkspacePatterns(patterns []string) []string {
	out := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" || strings.HasPrefix(pattern, "!") {
			continue
		}
		pattern = strings.TrimPrefix(Normalize(pattern), "./")
		if pattern != "" {
			out = append(out, pattern)
		}
	}
	return out
}

func expandJSPackageWorkspacePattern(root, pattern string) ([]string, error) {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return nil, nil
	}
	if strings.Contains(pattern, "**") {
		prefix := strings.Split(pattern, "**")[0]
		return collectJSPackageDirs(filepath.Join(root, filepath.FromSlash(prefix)), 6), nil
	}
	matches, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(pattern)))
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}
		dir := match
		if !info.IsDir() {
			dir = filepath.Dir(match)
		}
		if hasFile(dir, "package.json") {
			out = append(out, filepath.Clean(dir))
		}
	}
	sort.Strings(out)
	return out, nil
}

func collectJSPackageDirs(base string, maxDepth int) []string {
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
			depth = strings.Count(Normalize(rel), "/") + 1
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
				clean := filepath.Clean(path)
				if _, ok := seen[clean]; !ok {
					seen[clean] = struct{}{}
					out = append(out, clean)
				}
			}
		}
		return nil
	})
	sort.Strings(out)
	return out
}

func nearestPackageDir(start, root string) (string, bool) {
	start = filepath.Clean(start)
	if info, err := os.Stat(start); err == nil && !info.IsDir() {
		start = filepath.Dir(start)
	}
	root = filepath.Clean(root)
	for dir := start; ; dir = filepath.Dir(dir) {
		if hasFile(dir, "package.json") {
			return filepath.Clean(dir), true
		}
		if sameCleanPath(dir, root) || dir == filepath.Dir(dir) {
			break
		}
	}
	return "", false
}

func hasFile(dir, name string) bool {
	fi, err := os.Stat(filepath.Join(dir, name))
	return err == nil && !fi.IsDir()
}

func (r *Resolver) resolveJSPackageImport(sourceFile, rawImport string) (string, bool, bool) {
	if strings.HasPrefix(rawImport, "#") {
		pkg, ok := r.sourcePackage(sourceFile)
		if !ok {
			r.diagnostics.UnresolvedLocalImports++
			return "", true, false
		}
		resolved, matched, ignored := r.resolvePackageImports(pkg, rawImport)
		if ignored {
			r.diagnostics.IgnoredResolutionCases++
		}
		if resolved != "" {
			r.diagnostics.WorkspacePackageImports++
			return resolved, true, true
		}
		if matched {
			r.diagnostics.UnresolvedLocalImports++
			return "", true, false
		}
		r.diagnostics.UnresolvedLocalImports++
		return "", true, false
	}

	packageName, subpath, ok := parseJSPackageSpecifier(rawImport)
	if !ok {
		return "", false, false
	}
	pkg, ok := r.jsPackages[packageName]
	if !ok {
		return "", false, false
	}
	resolved, ignored := r.resolveJSPackageSubpath(pkg, subpath)
	if ignored {
		r.diagnostics.IgnoredResolutionCases++
	}
	if resolved != "" {
		r.diagnostics.WorkspacePackageImports++
		return resolved, true, true
	}
	r.diagnostics.UnresolvedLocalImports++
	return "", true, false
}

func (r *Resolver) sourcePackage(sourceFile string) (jsPackage, bool) {
	sourceFile = Normalize(sourceFile)
	for _, pkg := range r.jsPackageRoots {
		rel := strings.TrimSuffix(Normalize(pkg.RelDir), "/")
		if rel == "." || rel == "" || sourceFile == rel || strings.HasPrefix(sourceFile, rel+"/") {
			return pkg, true
		}
	}
	return jsPackage{}, false
}

func (r *Resolver) resolveJSPackageSubpath(pkg jsPackage, subpath string) (string, bool) {
	ignored := false
	if len(pkg.Exports) > 0 {
		key := "."
		if subpath != "" {
			key = "./" + strings.TrimPrefix(subpath, "/")
		}
		if resolved, matched, exportIgnored := r.resolvePackageExports(pkg, pkg.Exports, key); resolved != "" {
			return resolved, exportIgnored
		} else if matched || exportIgnored {
			ignored = true
		}
	}

	if subpath == "" {
		for _, target := range []string{pkg.Module, pkg.Main, pkg.Types} {
			if strings.TrimSpace(target) == "" {
				continue
			}
			if resolved, ok := r.probePackageTarget(pkg, target, ""); ok {
				return resolved, ignored
			}
			ignored = true
		}
		for _, target := range []string{"src/index", "index"} {
			if resolved, ok := r.probePackageTarget(pkg, target, ""); ok {
				return resolved, ignored
			}
		}
		return "", ignored
	}

	for _, target := range []string{subpath, filepath.Join("src", subpath)} {
		if resolved, ok := r.probePackageTarget(pkg, target, ""); ok {
			return resolved, ignored
		}
	}
	return "", ignored
}

func (r *Resolver) resolvePackageExports(pkg jsPackage, raw json.RawMessage, key string) (string, bool, bool) {
	key = strings.TrimSpace(key)
	if key == "" {
		key = "."
	}
	if isJSONString(raw) {
		if key != "." {
			return "", false, false
		}
		resolved, ok, ignored := r.resolvePackageTargetValue(pkg, raw, "")
		return resolved, true, !ok || ignored
	}

	var entries map[string]json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil {
		return "", false, true
	}
	if target, ok := entries[key]; ok {
		resolved, targetOK, ignored := r.resolvePackageTargetValue(pkg, target, "")
		return resolved, true, !targetOK || ignored
	}
	if key != "." {
		if resolved, matched, ignored := r.resolvePackageWildcardMap(pkg, entries, key); matched {
			return resolved, true, ignored || resolved == ""
		}
		return "", false, false
	}
	if isConditionalPackageMap(entries) {
		resolved, ok, ignored := r.resolvePackageConditionMap(pkg, entries, "")
		return resolved, true, !ok || ignored
	}
	return "", false, true
}

func (r *Resolver) resolvePackageImports(pkg jsPackage, rawImport string) (string, bool, bool) {
	if len(pkg.Imports) == 0 {
		return "", false, false
	}
	if target, ok := pkg.Imports[rawImport]; ok {
		resolved, targetOK, ignored := r.resolvePackageTargetValue(pkg, target, "")
		return resolved, true, !targetOK || ignored
	}
	return r.resolvePackageWildcardMap(pkg, pkg.Imports, rawImport)
}

func (r *Resolver) resolvePackageWildcardMap(pkg jsPackage, entries map[string]json.RawMessage, key string) (string, bool, bool) {
	type wildcardEntry struct {
		pattern string
		target  json.RawMessage
	}
	wildcards := make([]wildcardEntry, 0)
	for pattern, target := range entries {
		if strings.Contains(pattern, "*") {
			wildcards = append(wildcards, wildcardEntry{pattern: pattern, target: target})
		}
	}
	sort.Slice(wildcards, func(i, j int) bool {
		return len(wildcards[i].pattern) > len(wildcards[j].pattern)
	})
	for _, entry := range wildcards {
		prefix, suffix, _ := strings.Cut(entry.pattern, "*")
		if !strings.HasPrefix(key, prefix) || !strings.HasSuffix(key, suffix) {
			continue
		}
		wildcard := strings.TrimSuffix(strings.TrimPrefix(key, prefix), suffix)
		resolved, ok, ignored := r.resolvePackageTargetValue(pkg, entry.target, wildcard)
		return resolved, true, !ok || ignored
	}
	return "", false, false
}

func (r *Resolver) resolvePackageTargetValue(pkg jsPackage, raw json.RawMessage, wildcard string) (string, bool, bool) {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		if strings.TrimSpace(wildcard) != "" {
			s = strings.ReplaceAll(s, "*", wildcard)
		}
		if !strings.HasPrefix(strings.TrimSpace(s), ".") && !filepath.IsAbs(strings.TrimSpace(s)) {
			return "", false, true
		}
		resolved, ok := r.probePackageTarget(pkg, s, "")
		if !ok {
			return "", false, false
		}
		return resolved, true, false
	}

	var entries map[string]json.RawMessage
	if err := json.Unmarshal(raw, &entries); err == nil && len(entries) > 0 {
		return r.resolvePackageConditionMap(pkg, entries, wildcard)
	}

	var many []json.RawMessage
	if err := json.Unmarshal(raw, &many); err == nil {
		for _, item := range many {
			if resolved, ok, ignored := r.resolvePackageTargetValue(pkg, item, wildcard); ok {
				return resolved, true, ignored
			}
		}
		return "", false, true
	}

	return "", false, true
}

func (r *Resolver) resolvePackageConditionMap(pkg jsPackage, entries map[string]json.RawMessage, wildcard string) (string, bool, bool) {
	for _, condition := range []string{"import", "default", "module", "browser", "node", "development", "require", "types"} {
		target, ok := entries[condition]
		if !ok {
			continue
		}
		if resolved, targetOK, ignored := r.resolvePackageTargetValue(pkg, target, wildcard); targetOK {
			return resolved, true, ignored
		}
	}
	return "", false, true
}

func (r *Resolver) probePackageTarget(pkg jsPackage, target, wildcard string) (string, bool) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", false
	}
	if wildcard != "" {
		target = strings.ReplaceAll(target, "*", wildcard)
	}
	if strings.HasPrefix(target, ".") {
		return r.probeLocal(filepath.Join(pkg.Dir, target))
	}
	if filepath.IsAbs(target) {
		return r.probeLocal(target)
	}
	return r.probeLocal(filepath.Join(pkg.Dir, filepath.FromSlash(target)))
}

func isJSONString(raw json.RawMessage) bool {
	var s string
	return json.Unmarshal(raw, &s) == nil
}

func isConditionalPackageMap(entries map[string]json.RawMessage) bool {
	if len(entries) == 0 {
		return false
	}
	for key := range entries {
		if strings.HasPrefix(key, ".") || strings.HasPrefix(key, "#") {
			return false
		}
	}
	return true
}

func parseJSPackageSpecifier(spec string) (string, string, bool) {
	spec = strings.TrimSpace(Normalize(spec))
	if spec == "" || strings.HasPrefix(spec, ".") || strings.HasPrefix(spec, "/") || strings.HasPrefix(spec, "#") {
		return "", "", false
	}
	if strings.Contains(spec, "\\") || strings.Contains(spec, ":") {
		return "", "", false
	}
	parts := strings.Split(spec, "/")
	if strings.HasPrefix(spec, "@") {
		if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
			return "", "", false
		}
		name := parts[0] + "/" + parts[1]
		return name, strings.Join(parts[2:], "/"), true
	}
	if parts[0] == "" {
		return "", "", false
	}
	return parts[0], strings.Join(parts[1:], "/"), true
}
