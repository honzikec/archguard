package react_router

import (
	"path"
	"regexp"
	"strings"

	"github.com/honzikec/archguard/internal/framework/common"
	"github.com/honzikec/archguard/internal/framework/contracts"
)

var reDynamicToken = regexp.MustCompile(`[$:][A-Za-z0-9_]+`)

type Profile struct{}

func New() contracts.Profile {
	return Profile{}
}

func (Profile) ID() string {
	return "react_router"
}

func (Profile) Detect(roots []string) contracts.Detection {
	if common.HasAnyDirectory(roots, []string{"app/routes", "src/routes"}) {
		return contracts.Detection{Matched: true, Reason: "routes directory detected", Score: 100}
	}
	if common.HasPackageDependency(roots, []string{"react-router", "react-router-dom", "@react-router/dev"}) {
		return contracts.Detection{Matched: true, Reason: "react-router dependency detected", Score: 30}
	}
	return contracts.Detection{}
}

func (Profile) NormalizeSubtree(subtree string) string {
	subtree = path.Clean(strings.TrimSpace(subtree))
	if subtree == "." || subtree == "" {
		return "."
	}

	parts := strings.Split(subtree, "/")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		out = append(out, normalizeSegment(part))
	}
	return path.Clean(strings.Join(out, "/"))
}

func (p Profile) NormalizeFile(file string) string {
	dir := p.NormalizeSubtree(path.Dir(file))
	base := path.Base(file)
	base = strings.ReplaceAll(base, "_index", "index")
	base = reDynamicToken.ReplaceAllString(base, "[param]")
	return path.Join(dir, base)
}

func normalizeSegment(segment string) string {
	segment = strings.TrimSpace(segment)
	if segment == "" {
		return segment
	}
	if segment == "_index" {
		return "index"
	}
	if strings.HasPrefix(segment, ":") || strings.HasPrefix(segment, "$") {
		return "[param]"
	}
	if strings.HasPrefix(segment, "[") && strings.HasSuffix(segment, "]") {
		return "[param]"
	}
	return segment
}
