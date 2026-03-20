package react

import (
	"path"
	"strings"

	"github.com/honzikec/archguard/internal/framework/common"
	"github.com/honzikec/archguard/internal/framework/contracts"
)

var reactAncillarySuffixes = []string{".test", ".spec", ".stories", ".story"}

type Profile struct{}

func New() contracts.Profile {
	return Profile{}
}

func (Profile) ID() string {
	return "react"
}

func (Profile) Detect(roots []string) contracts.Detection {
	if common.HasAnyFileNamed(roots, []string{"next.config.js", "next.config.mjs", "next.config.ts", "next.config.cjs", "angular.json"}) {
		return contracts.Detection{}
	}
	if common.HasPackageDependency(roots, []string{"next", "react-router", "react-router-dom", "@react-router/dev", "react-native", "expo", "@angular/core"}) {
		return contracts.Detection{}
	}
	if !common.HasPackageDependency(roots, []string{"react"}) {
		return contracts.Detection{}
	}
	if common.HasAnyDirectory(roots, []string{"src"}) {
		return contracts.Detection{Matched: true, Reason: "react dependency with src directory", Score: 25}
	}
	if common.HasFileWithSuffix(roots, []string{".jsx", ".tsx"}, 8) {
		return contracts.Detection{Matched: true, Reason: "react dependency with jsx/tsx files", Score: 20}
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
		switch parts[i] {
		case "__tests__":
			parts[i] = "tests"
		case "__mocks__":
			parts[i] = "mocks"
		}
	}
	return path.Clean(strings.Join(parts, "/"))
}

func (p Profile) NormalizeFile(file string) string {
	dir := p.NormalizeSubtree(path.Dir(file))
	base := path.Base(file)
	ext := path.Ext(base)
	name := strings.TrimSuffix(base, ext)
	for _, suffix := range reactAncillarySuffixes {
		if strings.HasSuffix(name, suffix) {
			name = strings.TrimSuffix(name, suffix)
			break
		}
	}
	return path.Join(dir, name+ext)
}
