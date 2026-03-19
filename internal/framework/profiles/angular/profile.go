package angular

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
	return "angular"
}

func (Profile) Detect(roots []string) contracts.Detection {
	if common.HasAnyFileNamed(roots, []string{"angular.json"}) {
		return contracts.Detection{Matched: true, Reason: "angular.json found"}
	}
	if common.HasPackageDependency(roots, []string{"@angular/core"}) {
		return contracts.Detection{Matched: true, Reason: "@angular/core dependency detected"}
	}
	return contracts.Detection{}
}

func (Profile) NormalizeSubtree(subtree string) string {
	subtree = path.Clean(strings.TrimSpace(subtree))
	if subtree == "." || subtree == "" {
		return "."
	}
	parts := strings.Split(subtree, "/")
	for i := range parts {
		if strings.HasPrefix(parts[i], ":") {
			parts[i] = "[param]"
		}
	}
	return path.Clean(strings.Join(parts, "/"))
}

func (p Profile) NormalizeFile(file string) string {
	dir := p.NormalizeSubtree(path.Dir(file))
	base := path.Base(file)
	base = strings.ReplaceAll(base, "-routing", "-routes")
	base = strings.ReplaceAll(base, ".routing", ".routes")
	return path.Join(dir, base)
}
