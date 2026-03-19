package language

import (
	"sort"
	"strings"

	"github.com/honzikec/archguard/internal/language/contracts"
	"github.com/honzikec/archguard/internal/language/javascript"
	"github.com/honzikec/archguard/internal/language/php"
)

const defaultLanguageAdapter = "javascript"

type Resolution struct {
	Explicit string            `json:"explicit"`
	Selected string            `json:"selected"`
	Reason   string            `json:"reason"`
	Matched  []string          `json:"matched,omitempty"`
	Details  map[string]string `json:"details,omitempty"`
	Adapter  contracts.Adapter `json:"-"`
}

func RegisteredAdapters() []contracts.Adapter {
	adapters := []contracts.Adapter{javascript.New(), php.New()}
	sort.Slice(adapters, func(i, j int) bool {
		return adapters[i].ID() < adapters[j].ID()
	})
	return adapters
}

func RegisteredLanguages() []string {
	adapters := RegisteredAdapters()
	ids := make([]string, 0, len(adapters))
	for _, adapter := range adapters {
		ids = append(ids, adapter.ID())
	}
	return ids
}

func FindAdapter(adapterID string) (contracts.Adapter, bool) {
	adapterID = strings.ToLower(strings.TrimSpace(adapterID))
	for _, adapter := range RegisteredAdapters() {
		if adapter.ID() == adapterID {
			return adapter, true
		}
	}
	return nil, false
}

func Resolve(explicitLanguage string, roots []string) Resolution {
	explicit := strings.ToLower(strings.TrimSpace(explicitLanguage))
	out := Resolution{Explicit: explicit, Reason: "auto_none", Details: map[string]string{}}

	if explicit != "" && explicit != "auto" {
		if adapter, ok := FindAdapter(explicit); ok {
			out.Selected = explicit
			out.Adapter = adapter
			out.Reason = "explicit"
			return out
		}
		out.Reason = "explicit_unknown"
		return out
	}

	adapters := RegisteredAdapters()
	matched := make([]contracts.Adapter, 0, len(adapters))
	for _, adapter := range adapters {
		d := adapter.Detect(roots)
		if !d.Matched {
			continue
		}
		matched = append(matched, adapter)
		out.Details[adapter.ID()] = strings.TrimSpace(d.Reason)
	}
	for _, adapter := range matched {
		out.Matched = append(out.Matched, adapter.ID())
	}
	sort.Strings(out.Matched)

	switch len(matched) {
	case 0:
		if fallback, ok := FindAdapter(defaultLanguageAdapter); ok {
			out.Selected = fallback.ID()
			out.Adapter = fallback
			out.Reason = "auto_none_fallback_default"
			return out
		}
		if len(adapters) > 0 {
			out.Selected = adapters[0].ID()
			out.Adapter = adapters[0]
			out.Reason = "fallback_first_registered"
		}
		return out
	case 1:
		out.Selected = matched[0].ID()
		out.Adapter = matched[0]
		out.Reason = "auto_detected"
		return out
	default:
		if explicit == "auto" {
			if fallback, ok := FindAdapter(defaultLanguageAdapter); ok {
				out.Selected = fallback.ID()
				out.Adapter = fallback
				out.Reason = "auto_ambiguous_default"
				return out
			}
		}
		out.Selected = matched[0].ID()
		out.Adapter = matched[0]
		out.Reason = "auto_ambiguous_first_selected"
		return out
	}
}
