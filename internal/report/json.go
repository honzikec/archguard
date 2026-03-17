package report

import (
	"encoding/json"
	"fmt"

	"github.com/honzikec/archguard/internal/model"
)

type jsonFinding struct {
	RuleID      string `json:"rule_id"`
	RuleKind    string `json:"rule_kind"`
	Severity    string `json:"severity"`
	Message     string `json:"message"`
	FilePath    string `json:"file_path"`
	Line        int    `json:"line"`
	Column      int    `json:"column"`
	RawImport   string `json:"raw_import,omitempty"`
	Fingerprint string `json:"fingerprint"`
	Details     string `json:"details,omitempty"`
}

func PrintJSON(findings []model.Finding, summary Summary) {
	items := make([]jsonFinding, 0, len(findings))
	for _, f := range findings {
		items = append(items, jsonFinding{
			RuleID:      f.RuleID,
			RuleKind:    f.RuleKind,
			Severity:    f.Severity,
			Message:     f.Message,
			FilePath:    f.FilePath,
			Line:        f.Line,
			Column:      f.Column,
			RawImport:   f.RawImport,
			Fingerprint: f.Fingerprint,
			Details:     f.Details,
		})
	}

	payload := struct {
		Findings []jsonFinding `json:"findings"`
		Summary  Summary       `json:"summary"`
	}{
		Findings: items,
		Summary:  summary,
	}

	data, _ := json.MarshalIndent(payload, "", "  ")
	fmt.Println(string(data))
}
