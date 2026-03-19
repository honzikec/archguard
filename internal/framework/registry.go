package framework

import (
	"sort"
	"strings"

	"github.com/honzikec/archguard/internal/framework/profiles/angular"
	"github.com/honzikec/archguard/internal/framework/profiles/nextjs"
	"github.com/honzikec/archguard/internal/framework/profiles/react"
	"github.com/honzikec/archguard/internal/framework/profiles/react_native"
	"github.com/honzikec/archguard/internal/framework/profiles/react_router"
)

func RegisteredProfiles() []Profile {
	profiles := []Profile{
		nextjs.New(),
		react.New(),
		react_router.New(),
		react_native.New(),
		angular.New(),
	}
	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].ID() < profiles[j].ID()
	})
	return profiles
}

func RegisteredFrameworks() []string {
	profiles := RegisteredProfiles()
	out := make([]string, 0, len(profiles)+1)
	out = append(out, "generic")
	for _, p := range profiles {
		out = append(out, p.ID())
	}
	return out
}

func FindProfile(profileID string) (Profile, bool) {
	profileID = strings.ToLower(strings.TrimSpace(profileID))
	for _, p := range RegisteredProfiles() {
		if p.ID() == profileID {
			return p, true
		}
	}
	return nil, false
}

func Resolve(explicitFramework string, roots []string) Resolution {
	explicit := strings.ToLower(strings.TrimSpace(explicitFramework))
	resolution := Resolution{Explicit: explicit, Reason: "auto_none"}

	if explicit != "" {
		switch explicit {
		case "generic":
			resolution.Reason = "explicit_generic"
			return resolution
		default:
			if _, ok := FindProfile(explicit); ok {
				resolution.Selected = explicit
				resolution.Reason = "explicit"
				return resolution
			}
			resolution.Reason = "explicit_unknown"
			return resolution
		}
	}

	profiles := RegisteredProfiles()
	matched := make([]string, 0)
	reasons := map[string]string{}
	for _, p := range profiles {
		d := p.Detect(roots)
		if !d.Matched {
			continue
		}
		matched = append(matched, p.ID())
		reasons[p.ID()] = strings.TrimSpace(d.Reason)
	}
	sort.Strings(matched)
	resolution.Matched = matched
	if len(reasons) > 0 {
		resolution.MatchedReason = reasons
	}

	switch len(matched) {
	case 0:
		resolution.Reason = "auto_none"
	case 1:
		resolution.Selected = matched[0]
		resolution.Reason = "auto_detected"
	default:
		resolution.Reason = "auto_ambiguous"
	}
	return resolution
}
