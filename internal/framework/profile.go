package framework

import (
	"strings"

	"github.com/honzikec/archguard/internal/framework/contracts"
)

type Detection = contracts.Detection
type Profile = contracts.Profile

type Resolution struct {
	Explicit      string            `json:"explicit"`
	Selected      string            `json:"selected"`
	Reason        string            `json:"reason"`
	Matched       []string          `json:"matched,omitempty"`
	MatchedReason map[string]string `json:"matched_reason,omitempty"`
}

func (r Resolution) EffectiveProfile() string {
	if strings.TrimSpace(r.Selected) == "" {
		return "generic"
	}
	return r.Selected
}

type NormalizationStats struct {
	OriginalNodes   int `json:"original_nodes"`
	NormalizedNodes int `json:"normalized_nodes"`
	OriginalFiles   int `json:"original_files"`
	NormalizedFiles int `json:"normalized_files"`
}
