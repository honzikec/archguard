package language

import (
	"sort"
	"strings"

	"github.com/honzikec/archguard/internal/language/contracts"
	"github.com/honzikec/archguard/internal/language/javascript"
)

type Resolution struct {
	Selected string            `json:"selected"`
	Reason   string            `json:"reason"`
	Matched  []string          `json:"matched,omitempty"`
	Details  map[string]string `json:"details,omitempty"`
	Adapter  contracts.Adapter `json:"-"`
}

func RegisteredAdapters() []contracts.Adapter {
	adapters := []contracts.Adapter{javascript.New()}
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

func Resolve(roots []string) Resolution {
	adapters := RegisteredAdapters()
	matched := make([]contracts.Adapter, 0, len(adapters))
	details := map[string]string{}

	for _, adapter := range adapters {
		d := adapter.Detect(roots)
		if !d.Matched {
			continue
		}
		matched = append(matched, adapter)
		details[adapter.ID()] = strings.TrimSpace(d.Reason)
	}

	out := Resolution{Reason: "auto_none", Details: details}
	if len(matched) == 0 {
		if len(adapters) == 0 {
			return out
		}
		out.Selected = adapters[0].ID()
		out.Adapter = adapters[0]
		out.Reason = "fallback_first_registered"
		return out
	}

	for _, adapter := range matched {
		out.Matched = append(out.Matched, adapter.ID())
	}
	if len(matched) == 1 {
		out.Selected = matched[0].ID()
		out.Adapter = matched[0]
		out.Reason = "auto_detected"
		return out
	}

	out.Selected = matched[0].ID()
	out.Adapter = matched[0]
	out.Reason = "auto_ambiguous_first_selected"
	return out
}
