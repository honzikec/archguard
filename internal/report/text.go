package report

import (
	"bytes"
	"fmt"
	"os"
	"sort"

	"github.com/honzikec/archguard/internal/model"
)

func PrintText(findings []model.Finding, summary Summary) error {
	var b bytes.Buffer
	if len(findings) == 0 {
		fmt.Fprintln(&b, "No architectural violations found.")
		printSummary(&b, summary)
		_, err := os.Stdout.Write(b.Bytes())
		return err
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
		fmt.Fprintf(&b, "%s\n", key)
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
			fmt.Fprintf(&b, "  - %s:%d:%d %s", f.FilePath, f.Line, f.Column, f.Message)
			if f.RawImport != "" {
				fmt.Fprintf(&b, " (import: %s)", f.RawImport)
			}
			if f.Details != "" {
				fmt.Fprintf(&b, " [%s]", f.Details)
			}
			fmt.Fprintln(&b)
		}
	}
	printSummary(&b, summary)
	_, err := os.Stdout.Write(b.Bytes())
	return err
}

func printSummary(b *bytes.Buffer, summary Summary) {
	fmt.Fprintln(b)
	fmt.Fprintf(b, "Summary: files=%d imports=%d findings=%d (error=%d warning=%d) suppressed=%d parse_errors=%d files_skipped=%d duration_ms=%d\n",
		summary.FilesScanned,
		summary.ImportsScanned,
		summary.FindingsTotal,
		summary.FindingsError,
		summary.FindingsWarning,
		summary.SuppressedFindings,
		summary.ParseErrors,
		summary.FilesSkipped,
		summary.DurationMS,
	)
	if summary.ConfigDir != "" {
		fmt.Fprintf(b, "Config: dir=%s roots=%v\n", summary.ConfigDir, summary.EffectiveRoots)
	}
	if summary.WorkspacePackageImports > 0 || summary.UnresolvedLocalImports > 0 || summary.IgnoredResolutionCases > 0 {
		fmt.Fprintf(b, "Resolver: workspace_package_imports=%d unresolved_local_imports=%d ignored_resolution_cases=%d\n",
			summary.WorkspacePackageImports,
			summary.UnresolvedLocalImports,
			summary.IgnoredResolutionCases,
		)
	}
}
