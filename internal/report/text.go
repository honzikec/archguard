package report

import (
	"fmt"
	"sort"

	"github.com/honzikec/archguard/internal/model"
)

func PrintText(findings []model.Finding, summary Summary) {
	if len(findings) == 0 {
		fmt.Println("No architectural violations found.")
		printSummary(summary)
		return
	}

	byRule := map[string][]model.Finding{}
	order := make([]string, 0)
	for _, f := range findings {
		key := fmt.Sprintf("%s (%s)", f.RuleID, f.Severity)
		if _, ok := byRule[key]; !ok {
			order = append(order, key)
		}
		byRule[key] = append(byRule[key], f)
	}
	sort.Strings(order)

	for _, key := range order {
		fmt.Printf("%s\n", key)
		items := byRule[key]
		sort.Slice(items, func(i, j int) bool {
			if items[i].FilePath != items[j].FilePath {
				return items[i].FilePath < items[j].FilePath
			}
			if items[i].Line != items[j].Line {
				return items[i].Line < items[j].Line
			}
			return items[i].Column < items[j].Column
		})
		for _, f := range items {
			fmt.Printf("  - %s:%d:%d %s", f.FilePath, f.Line, f.Column, f.Message)
			if f.RawImport != "" {
				fmt.Printf(" (import: %s)", f.RawImport)
			}
			if f.Details != "" {
				fmt.Printf(" [%s]", f.Details)
			}
			fmt.Println()
		}
	}
	printSummary(summary)
}

func printSummary(summary Summary) {
	fmt.Println()
	fmt.Printf("Summary: files=%d imports=%d findings=%d (error=%d warning=%d) parse_errors=%d files_skipped=%d duration_ms=%d\n",
		summary.FilesScanned,
		summary.ImportsScanned,
		summary.FindingsTotal,
		summary.FindingsError,
		summary.FindingsWarning,
		summary.ParseErrors,
		summary.FilesSkipped,
		summary.DurationMS,
	)
	if summary.ConfigDir != "" {
		fmt.Printf("Config: dir=%s roots=%v\n", summary.ConfigDir, summary.EffectiveRoots)
	}
}
