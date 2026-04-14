package pathutil

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tailscale/hujson"

	"github.com/honzikec/archguard/internal/config"
)

type Resolver struct {
	root       string
	baseURLs   []string
	aliases    map[string][]aliasTarget
	aliasOrder []string
	extensions []string
	probeCache map[string]probeResult
}

type aliasTarget struct {
	Pattern string
	BaseDir string
}

type probeResult struct {
	path string
	ok   bool
}

type tsConfigFile struct {
	Extends         json.RawMessage `json:"extends"`
	CompilerOptions struct {
		BaseURL string              `json:"baseUrl"`
		Paths   map[string][]string `json:"paths"`
	} `json:"compilerOptions"`
}

type resolvedTSConfig struct {
	BaseURL    string
	UseBaseURL bool
	Paths      map[string][]aliasTarget
}

type composerFile struct {
	Autoload struct {
		PSR4 map[string]any `json:"psr-4"`
	} `json:"autoload"`
	AutoloadDev struct {
		PSR4 map[string]any `json:"psr-4"`
	} `json:"autoload-dev"`
}

func NewResolver(root string, project config.ProjectSettings) (*Resolver, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve project root: %w", err)
	}

	r := &Resolver{
		root:       absRoot,
		aliases:    map[string][]aliasTarget{},
		extensions: []string{".ts", ".tsx", ".mts", ".cts", ".js", ".jsx", ".mjs", ".cjs", ".php"},
		probeCache: map[string]probeResult{},
	}

	tsconfigPath, err := detectTSConfig(absRoot, project.Tsconfig)
	if err != nil {
		return nil, err
	}
	if tsconfigPath != "" {
		if err := r.loadTSConfig(tsconfigPath); err != nil {
			return nil, err
		}
	}
	if err := r.loadComposerMappings(project); err != nil {
		return nil, err
	}

	for alias, targets := range project.Aliases {
		normalizedAlias := Normalize(alias)
		for _, target := range targets {
			r.aliases[normalizedAlias] = append(r.aliases[normalizedAlias], aliasTarget{
				Pattern: Normalize(target),
				BaseDir: r.root,
			})
		}
	}
	r.aliasOrder = make([]string, 0, len(r.aliases))
	for alias := range r.aliases {
		r.aliasOrder = append(r.aliasOrder, alias)
	}
	sort.Slice(r.aliasOrder, func(i, j int) bool {
		// Match more specific aliases first.
		return len(r.aliasOrder[i]) > len(r.aliasOrder[j])
	})

	return r, nil
}

func detectTSConfig(root, explicit string) (string, error) {
	if explicit != "" {
		p := explicit
		if !filepath.IsAbs(p) {
			p = filepath.Join(root, p)
		}
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("tsconfig not found: %s", p)
		}
		return p, nil
	}
	for _, name := range []string{"tsconfig.json", "jsconfig.json"} {
		p := filepath.Join(root, name)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", nil
}

func (r *Resolver) loadTSConfig(path string) error {
	resolved, err := readTSConfigChain(path, map[string]struct{}{})
	if err != nil {
		return err
	}
	if resolved.UseBaseURL && resolved.BaseURL != "" {
		r.baseURLs = append(r.baseURLs, resolved.BaseURL)
	}
	for alias, targets := range resolved.Paths {
		normalizedAlias := Normalize(alias)
		r.aliases[normalizedAlias] = append(r.aliases[normalizedAlias], targets...)
	}
	return nil
}

func readTSConfigChain(path string, seen map[string]struct{}) (resolvedTSConfig, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return resolvedTSConfig{}, err
	}
	if _, ok := seen[absPath]; ok {
		return resolvedTSConfig{}, fmt.Errorf("tsconfig extends cycle detected at %s", absPath)
	}
	seen[absPath] = struct{}{}
	defer delete(seen, absPath)

	data, err := os.ReadFile(path)
	if err != nil {
		return resolvedTSConfig{}, fmt.Errorf("failed to read %s: %w", path, err)
	}
	var cfg tsConfigFile
	if err := unmarshalJSONC(data, &cfg); err != nil {
		return resolvedTSConfig{}, fmt.Errorf("failed to parse %s: %w", path, err)
	}

	resolved := resolvedTSConfig{Paths: map[string][]aliasTarget{}}
	for _, spec := range tsConfigExtends(cfg.Extends) {
		parentPath, ok, err := resolveTSConfigExtends(filepath.Dir(absPath), spec)
		if err != nil {
			return resolvedTSConfig{}, err
		}
		if !ok {
			continue
		}
		parent, err := readTSConfigChain(parentPath, seen)
		if err != nil {
			return resolvedTSConfig{}, err
		}
		resolved.BaseURL = parent.BaseURL
		resolved.UseBaseURL = parent.UseBaseURL
		for alias, targets := range parent.Paths {
			resolved.Paths[alias] = append([]aliasTarget{}, targets...)
		}
	}

	base := cfg.CompilerOptions.BaseURL
	if base != "" {
		if !filepath.IsAbs(base) {
			base = filepath.Join(filepath.Dir(absPath), base)
		}
		resolved.BaseURL = filepath.Clean(base)
		resolved.UseBaseURL = true
	}
	if resolved.BaseURL == "" {
		resolved.BaseURL = filepath.Dir(absPath)
	}

	for alias, targets := range cfg.CompilerOptions.Paths {
		normalizedAlias := Normalize(alias)
		resolved.Paths[normalizedAlias] = nil
		for _, target := range targets {
			resolved.Paths[normalizedAlias] = append(resolved.Paths[normalizedAlias], aliasTarget{
				Pattern: Normalize(target),
				BaseDir: resolved.BaseURL,
			})
		}
	}
	return resolved, nil
}

func tsConfigExtends(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var single string
	if err := json.Unmarshal(raw, &single); err == nil && strings.TrimSpace(single) != "" {
		return []string{strings.TrimSpace(single)}
	}
	var many []string
	if err := json.Unmarshal(raw, &many); err == nil {
		out := make([]string, 0, len(many))
		for _, item := range many {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	}
	return nil
}

func resolveTSConfigExtends(baseDir, spec string) (string, bool, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return "", false, nil
	}
	if !filepath.IsAbs(spec) && !strings.HasPrefix(spec, ".") {
		// Package-based extends are intentionally ignored by the lightweight
		// resolver; local repository config remains usable without node_modules.
		return "", false, nil
	}
	base := spec
	if !filepath.IsAbs(base) {
		base = filepath.Join(baseDir, spec)
	}
	candidates := []string{base}
	if !strings.HasSuffix(strings.ToLower(base), ".json") {
		candidates = append(candidates, base+".json")
	}
	candidates = append(candidates, filepath.Join(base, "tsconfig.json"))
	for _, candidate := range candidates {
		if fi, err := os.Stat(candidate); err == nil && !fi.IsDir() {
			return filepath.Clean(candidate), true, nil
		}
	}
	return "", false, nil
}

func (r *Resolver) loadComposerMappings(project config.ProjectSettings) error {
	seen := map[string]struct{}{}
	candidates := []string{filepath.Join(r.root, "composer.json")}
	for _, root := range project.Roots {
		root = strings.TrimSpace(root)
		if root == "" || root == "." {
			continue
		}
		candidates = append(candidates, filepath.Join(r.root, root, "composer.json"))
	}

	for _, candidate := range candidates {
		abs, err := filepath.Abs(candidate)
		if err != nil {
			return err
		}
		if _, ok := seen[abs]; ok {
			continue
		}
		seen[abs] = struct{}{}
		if _, err := os.Stat(abs); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("failed to stat %s: %w", abs, err)
		}
		if err := r.loadComposerFile(abs); err != nil {
			return err
		}
	}
	return nil
}

func (r *Resolver) loadComposerFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", path, err)
	}
	var cfg composerFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse %s: %w", path, err)
	}
	for ns, value := range cfg.Autoload.PSR4 {
		if err := r.addComposerPSR4(path, ns, value); err != nil {
			return err
		}
	}
	for ns, value := range cfg.AutoloadDev.PSR4 {
		if err := r.addComposerPSR4(path, ns, value); err != nil {
			return err
		}
	}
	return nil
}

func (r *Resolver) addComposerPSR4(composerPath, namespace string, rawTargets any) error {
	namespace = strings.TrimSpace(namespace)
	namespace = strings.TrimSuffix(namespace, "\\")
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return nil
	}

	nsPath := Normalize(strings.ReplaceAll(namespace, "\\", "/"))
	alias := nsPath + "/*"

	targets, err := composerTargets(rawTargets)
	if err != nil {
		return fmt.Errorf("invalid composer psr-4 mapping %q in %s: %w", namespace, composerPath, err)
	}
	for _, target := range targets {
		target = strings.TrimSpace(target)
		if target == "" {
			continue
		}
		absTarget := target
		if !filepath.IsAbs(absTarget) {
			absTarget = filepath.Join(filepath.Dir(composerPath), absTarget)
		}
		relTarget, err := filepath.Rel(r.root, absTarget)
		if err != nil {
			relTarget = absTarget
		}
		normalized := Normalize(relTarget)
		normalized = strings.TrimSuffix(normalized, "/")
		if normalized == "." {
			normalized = ""
		}
		if normalized == "" {
			r.aliases[alias] = append(r.aliases[alias], aliasTarget{Pattern: "*", BaseDir: r.root})
			continue
		}
		r.aliases[alias] = append(r.aliases[alias], aliasTarget{Pattern: normalized + "/*", BaseDir: r.root})
	}
	return nil
}

func composerTargets(v any) ([]string, error) {
	switch x := v.(type) {
	case string:
		return []string{x}, nil
	case []any:
		out := make([]string, 0, len(x))
		for _, item := range x {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("target entry must be string")
			}
			out = append(out, s)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("targets must be string or array of strings")
	}
}

func unmarshalJSONC(data []byte, v any) error {
	// Strip UTF-8 BOM when present.
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		data = data[3:]
	}
	clean, err := hujson.Standardize(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(clean, v)
}

func (r *Resolver) Resolve(sourceFile, rawImport string) (string, bool) {
	sourceFile = Normalize(sourceFile)
	rawImport = strings.TrimSpace(rawImport)

	if strings.HasPrefix(rawImport, ".") {
		base := filepath.Join(r.root, filepath.Dir(sourceFile), rawImport)
		if resolved, ok := r.probeLocal(base); ok {
			return resolved, false
		}
		return "", false
	}

	if strings.HasPrefix(rawImport, "/") {
		base := filepath.Join(r.root, rawImport)
		if resolved, ok := r.probeLocal(base); ok {
			return resolved, false
		}
		return "", false
	}

	for _, alias := range r.aliasOrder {
		targets := r.aliases[alias]
		if resolved, ok := r.resolveAlias(alias, targets, rawImport); ok {
			return resolved, false
		}
	}

	for _, baseURL := range r.baseURLs {
		if resolved, ok := r.probeLocal(filepath.Join(baseURL, rawImport)); ok {
			return resolved, false
		}
	}

	if isPHPSourceFile(sourceFile) && strings.HasPrefix(rawImport, "@") {
		if resolved, ok := r.resolvePHPAliasPath(rawImport); ok {
			return resolved, false
		}
		// Yii-style aliases (e.g. @common/config/main.php) are local path aliases,
		// not package identifiers. Keep unresolved aliases out of package constraints.
		return "", false
	}
	if isPHPSourceFile(sourceFile) && strings.Contains(rawImport, "/") && strings.HasSuffix(strings.ToLower(rawImport), ".php") {
		if resolved, ok := r.resolvePHPIncludePath(sourceFile, rawImport); ok {
			return resolved, false
		}
		return "", false
	}
	if isPHPSourceFile(sourceFile) && isLikelyPHPBareSymbolImport(rawImport) {
		return "", false
	}

	if strings.Contains(rawImport, `\`) {
		if resolved, ok := r.resolvePHPNamespace(rawImport); ok {
			return resolved, false
		}
		if r.isLikelyLocalPHPNamespace(rawImport) {
			// Keep unresolved local namespaces out of package constraints.
			return "", false
		}
	}

	return "", true
}

func (r *Resolver) resolvePHPAliasPath(rawImport string) (string, bool) {
	trimmed := strings.TrimSpace(rawImport)
	trimmed = strings.TrimPrefix(trimmed, "@")
	trimmed = strings.TrimPrefix(trimmed, "/")
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" {
		return "", false
	}
	base := filepath.Join(r.root, Normalize(trimmed))
	return r.probeLocal(base)
}

func (r *Resolver) resolvePHPNamespace(rawImport string) (string, bool) {
	normalized := Normalize(rawImport)
	normalized = strings.Trim(normalized, "/")
	if normalized == "" || strings.Contains(normalized, ":") {
		return "", false
	}
	base := filepath.Join(r.root, normalized)
	return r.probeLocal(base)
}

func (r *Resolver) isLikelyLocalPHPNamespace(rawImport string) bool {
	normalized := Normalize(rawImport)
	normalized = strings.Trim(normalized, "/")
	if normalized == "" || strings.Contains(normalized, ":") {
		return false
	}
	parts := strings.Split(normalized, "/")
	if len(parts) == 0 || parts[0] == "" {
		return false
	}
	if strings.EqualFold(parts[0], "vendor") {
		return false
	}
	fi, err := os.Stat(filepath.Join(r.root, parts[0]))
	if err != nil {
		return false
	}
	return fi.IsDir()
}

func isPHPSourceFile(sourceFile string) bool {
	sourceFile = strings.ToLower(sourceFile)
	return strings.HasSuffix(sourceFile, ".php") || strings.HasSuffix(sourceFile, ".phtml")
}

func isLikelyPHPBareSymbolImport(rawImport string) bool {
	rawImport = strings.TrimSpace(rawImport)
	if rawImport == "" {
		return false
	}
	if strings.Contains(rawImport, `\`) || strings.Contains(rawImport, "/") || strings.HasPrefix(rawImport, "@") {
		return false
	}
	if strings.Contains(rawImport, ".") || strings.Contains(rawImport, ":") {
		return false
	}
	for _, r := range rawImport {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			continue
		}
		return false
	}
	return true
}

func (r *Resolver) resolvePHPIncludePath(sourceFile, rawImport string) (string, bool) {
	sourceDir := filepath.Dir(sourceFile)
	candidates := []string{
		filepath.Join(r.root, sourceDir, rawImport),
		filepath.Join(r.root, rawImport),
	}
	for _, base := range candidates {
		if resolved, ok := r.probeLocal(base); ok {
			return resolved, true
		}
	}
	return "", false
}

func (r *Resolver) resolveAlias(alias string, targets []aliasTarget, rawImport string) (string, bool) {
	alias = Normalize(alias)
	rawImport = Normalize(rawImport)
	rawImport = strings.TrimPrefix(rawImport, "/")

	wildcard := strings.Contains(alias, "*")
	if !wildcard {
		if rawImport != alias {
			return "", false
		}
		for _, target := range targets {
			base := r.aliasTargetBase(target, "")
			if resolved, ok := r.probeLocal(base); ok {
				return resolved, true
			}
		}
		return "", false
	}

	prefix := strings.Split(alias, "*")[0]
	suffix := strings.Split(alias, "*")[1]
	if !strings.HasPrefix(rawImport, prefix) || !strings.HasSuffix(rawImport, suffix) {
		return "", false
	}
	middle := strings.TrimSuffix(strings.TrimPrefix(rawImport, prefix), suffix)

	for _, target := range targets {
		base := r.aliasTargetBase(target, middle)
		if resolved, ok := r.probeLocal(base); ok {
			return resolved, true
		}
	}

	return "", false
}

func (r *Resolver) aliasTargetBase(target aliasTarget, middle string) string {
	pattern := Normalize(target.Pattern)
	if strings.Contains(pattern, "*") {
		pattern = strings.ReplaceAll(pattern, "*", middle)
	} else if middle != "" {
		pattern = filepath.Join(pattern, middle)
	}

	baseRoot := target.BaseDir
	if baseRoot == "" {
		baseRoot = r.root
	}
	if filepath.IsAbs(pattern) {
		return pattern
	}
	return filepath.Join(baseRoot, pattern)
}

func (r *Resolver) probeLocal(base string) (string, bool) {
	base = filepath.Clean(base)
	if cached, ok := r.probeCache[base]; ok {
		return cached.path, cached.ok
	}
	candidates := []string{base}

	ext := filepath.Ext(base)
	if ext == "" || !r.isKnownExtension(ext) {
		for _, e := range r.extensions {
			candidates = append(candidates, base+e)
		}
		for _, e := range r.extensions {
			candidates = append(candidates, filepath.Join(base, "index"+e))
		}
	}

	for _, candidate := range candidates {
		if fi, err := os.Stat(candidate); err == nil && !fi.IsDir() {
			rel, err := filepath.Rel(r.root, candidate)
			if err != nil {
				resolved := Normalize(candidate)
				r.probeCache[base] = probeResult{path: resolved, ok: true}
				return resolved, true
			}
			resolved := Normalize(rel)
			r.probeCache[base] = probeResult{path: resolved, ok: true}
			return resolved, true
		}
	}
	r.probeCache[base] = probeResult{}
	return "", false
}

func (r *Resolver) isKnownExtension(ext string) bool {
	for _, e := range r.extensions {
		if ext == e {
			return true
		}
	}
	return false
}
