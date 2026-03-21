package report

import (
	"fmt"
	"os"
	"strings"

	"github.com/honzikec/archguard/internal/model"
	sarifreport "github.com/owenrumney/go-sarif/v3/pkg/report"
	sarif "github.com/owenrumney/go-sarif/v3/pkg/report/v210/sarif"
)

func PrintSARIF(findings []model.Finding, summary Summary) {
	report := sarifreport.NewV210Report()
	run := sarif.NewRunWithInformationURI("ArchGuard", "https://github.com/honzikec/archguard")
	run.WithProperties(sarif.NewPropertyBag().Add("summary", summary))

	seenRules := map[string]bool{}
	for _, f := range findings {
		if !seenRules[f.RuleID] {
			rule := run.AddRule(f.RuleID).WithDescription(f.RuleKind)
			if msg := strings.TrimSpace(f.Message); msg != "" {
				rule.WithMarkdownHelp(msg)
			}
			seenRules[f.RuleID] = true
		}

		level := "warning"
		if strings.EqualFold(f.Severity, "error") {
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
		result := run.CreateResultForRule(f.RuleID).
			WithLevel(level).
			WithMessage(sarif.NewTextMessage(f.Message)).
			AddLocation(sarif.NewLocationWithPhysicalLocation(
				sarif.NewPhysicalLocation().
					WithArtifactLocation(sarif.NewSimpleArtifactLocation(f.FilePath)).
					WithRegion(sarif.NewRegion().WithStartLine(line).WithStartColumn(col)),
			))
		if f.Fingerprint != "" {
			result.WithPartialFingerprints(map[string]string{
				"primaryLocationLineHash": f.Fingerprint,
			})
		}
	}
	report.AddRun(run)

	if err := report.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: generated SARIF failed validation: %v\n", err)
	}
	if err := report.PrettyWrite(os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write SARIF output: %v\n", err)
	}
}
