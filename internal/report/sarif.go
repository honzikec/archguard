package report

import (
	"encoding/json"
	"fmt"

	"github.com/honzikec/archguard/internal/model"
)

type SarifLog struct {
	Version string     `json:"version"`
	Schema  string     `json:"$schema"`
	Runs    []SarifRun `json:"runs"`
}

type SarifRun struct {
	Tool       SarifTool      `json:"tool"`
	Results    []SarifResult  `json:"results"`
	Properties map[string]any `json:"properties,omitempty"`
}

type SarifTool struct {
	Driver SarifDriver `json:"driver"`
}

type SarifDriver struct {
	Name           string      `json:"name"`
	InformationURI string      `json:"informationUri"`
	Rules          []SarifRule `json:"rules"`
}

type SarifRule struct {
	ID               string           `json:"id"`
	ShortDescription SarifMessageText `json:"shortDescription"`
	Help             SarifMessageText `json:"help"`
}

type SarifResult struct {
	RuleID              string            `json:"ruleId"`
	Level               string            `json:"level"`
	Message             SarifMessageText  `json:"message"`
	Locations           []SarifLocation   `json:"locations"`
	PartialFingerprints map[string]string `json:"partialFingerprints,omitempty"`
}

type SarifMessageText struct {
	Text string `json:"text"`
}

type SarifLocation struct {
	PhysicalLocation SarifPhysicalLocation `json:"physicalLocation"`
}

type SarifPhysicalLocation struct {
	ArtifactLocation SarifArtifactLocation `json:"artifactLocation"`
	Region           SarifRegion           `json:"region"`
}

type SarifArtifactLocation struct {
	URI string `json:"uri"`
}

type SarifRegion struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn"`
}

func PrintSARIF(findings []model.Finding, summary Summary) {
	run := SarifRun{
		Tool: SarifTool{
			Driver: SarifDriver{
				Name:           "ArchGuard",
				InformationURI: "https://github.com/honzikec/archguard",
				Rules:          []SarifRule{},
			},
		},
		Results: []SarifResult{},
		Properties: map[string]any{
			"summary": summary,
		},
	}

	seenRules := map[string]struct{}{}
	for _, f := range findings {
		if _, ok := seenRules[f.RuleID]; !ok {
			run.Tool.Driver.Rules = append(run.Tool.Driver.Rules, SarifRule{
				ID:               f.RuleID,
				ShortDescription: SarifMessageText{Text: f.RuleKind},
				Help:             SarifMessageText{Text: f.Message},
			})
			seenRules[f.RuleID] = struct{}{}
		}

		level := "warning"
		if f.Severity == "error" {
			level = "error"
		}
		line := f.Line
		if line <= 0 {
			line = 1
		}
		col := f.Column
		if col <= 0 {
			col = 1
		}
		result := SarifResult{
			RuleID:  f.RuleID,
			Level:   level,
			Message: SarifMessageText{Text: f.Message},
			Locations: []SarifLocation{{
				PhysicalLocation: SarifPhysicalLocation{
					ArtifactLocation: SarifArtifactLocation{URI: f.FilePath},
					Region:           SarifRegion{StartLine: line, StartColumn: col},
				},
			}},
		}
		if f.Fingerprint != "" {
			result.PartialFingerprints = map[string]string{"primaryLocationLineHash": f.Fingerprint}
		}
		run.Results = append(run.Results, result)
	}

	log := SarifLog{
		Version: "2.1.0",
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Runs:    []SarifRun{run},
	}

	data, _ := json.MarshalIndent(log, "", "  ")
	fmt.Println(string(data))
}
