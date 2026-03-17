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
	Tool    SarifTool     `json:"tool"`
	Results []SarifResult `json:"results"`
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
}

type SarifResult struct {
	RuleID    string           `json:"ruleId"`
	Level     string           `json:"level"`
	Message   SarifMessageText `json:"message"`
	Locations []SarifLocation  `json:"locations"`
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

func PrintSARIF(findings []model.Finding) {
	run := SarifRun{
		Tool: SarifTool{
			Driver: SarifDriver{
				Name:           "ArchGuard",
				InformationURI: "https://github.com/honzikec/archguard",
				Rules:          make([]SarifRule, 0),
			},
		},
		Results: make([]SarifResult, 0),
	}

	seenRules := make(map[string]bool)

	for _, f := range findings {
		if !seenRules[f.RuleID] {
			run.Tool.Driver.Rules = append(run.Tool.Driver.Rules, SarifRule{
				ID: f.RuleID,
				ShortDescription: SarifMessageText{
					Text: f.Message,
				},
			})
			seenRules[f.RuleID] = true
		}

		level := "warning"
		if f.Severity == "error" {
			level = "error"
		}

		// Ensure proper default line numbers if empty
		startLine := f.Line
		if startLine == 0 {
			startLine = 1
		}

		run.Results = append(run.Results, SarifResult{
			RuleID: f.RuleID,
			Level:  level,
			Message: SarifMessageText{
				Text: f.Message + ": " + f.Rationale,
			},
			Locations: []SarifLocation{
				{
					PhysicalLocation: SarifPhysicalLocation{
						ArtifactLocation: SarifArtifactLocation{
							URI: f.FilePath,
						},
						Region: SarifRegion{
							StartLine:   startLine,
							StartColumn: f.Column,
						},
					},
				},
			},
		})
	}

	log := SarifLog{
		Version: "2.1.0",
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		Runs:    []SarifRun{run},
	}

	data, _ := json.MarshalIndent(log, "", "  ")
	fmt.Println(string(data))
}
