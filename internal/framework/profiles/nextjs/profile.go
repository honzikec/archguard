package nextjs

import (
	"path"
	"strings"

	"github.com/honzikec/archguard/internal/framework/common"
	"github.com/honzikec/archguard/internal/framework/contracts"
)

type Profile struct{}

func New() contracts.Profile {
	return Profile{}
}

func (Profile) ID() string {
	return "nextjs"
}

func (Profile) Detect(roots []string) contracts.Detection {
	if common.HasAnyFileNamed(roots, []string{"next.config.js", "next.config.mjs", "next.config.ts", "next.config.cjs"}) {
		return contracts.Detection{Matched: true, Reason: "next.config.* found"}
	}
	if common.HasPackageDependency(roots, []string{"next"}) {
		return contracts.Detection{Matched: true, Reason: "package.json contains next dependency"}
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
	return path.Join(dir, path.Base(file))
}

func normalizeSegment(segment string) string {
	segment = strings.TrimSpace(segment)
	if segment == "" {
		return segment
	}
	if strings.HasPrefix(segment, "@") && len(segment) > 1 {
		return "@slot"
	}
	if strings.HasPrefix(segment, "(") && strings.HasSuffix(segment, ")") {
		return "(group)"
	}
	if strings.HasPrefix(segment, "[") && strings.HasSuffix(segment, "]") {
		return "[param]"
	}
	return segment
}
