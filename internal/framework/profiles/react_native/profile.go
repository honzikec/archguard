package react_native

import (
	"path"
	"strings"

	"github.com/honzikec/archguard/internal/framework/common"
	"github.com/honzikec/archguard/internal/framework/contracts"
)

var platformSuffixes = []string{".ios", ".android", ".native", ".web"}

type Profile struct{}

func New() contracts.Profile {
	return Profile{}
}

func (Profile) ID() string {
	return "react_native"
}

func (Profile) Detect(roots []string) contracts.Detection {
	if common.HasPackageDependency(roots, []string{"react-native", "expo"}) {
		return contracts.Detection{Matched: true, Reason: "react-native or expo dependency detected"}
	}
	if common.HasFileWithSuffix(roots, []string{".ios.ts", ".ios.tsx", ".android.ts", ".android.tsx", ".native.ts", ".native.tsx"}, 6) {
		return contracts.Detection{Matched: true, Reason: "platform-specific react-native files detected"}
	}
	return contracts.Detection{}
}

func (Profile) NormalizeSubtree(subtree string) string {
	subtree = path.Clean(strings.TrimSpace(subtree))
	if subtree == "" {
		return "."
	}
	return subtree
}

func (p Profile) NormalizeFile(file string) string {
	dir := p.NormalizeSubtree(path.Dir(file))
	base := collapsePlatformSuffix(path.Base(file))
	return path.Join(dir, base)
}

func collapsePlatformSuffix(base string) string {
	ext := path.Ext(base)
	name := strings.TrimSuffix(base, ext)
	for _, suffix := range platformSuffixes {
		if strings.HasSuffix(name, suffix) {
			name = strings.TrimSuffix(name, suffix) + ".platform"
			break
		}
	}
	return name + ext
}
