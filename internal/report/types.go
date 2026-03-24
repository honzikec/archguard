package report

import "github.com/honzikec/archguard/internal/model"

type Summary struct {
	FilesScanned    int      `json:"files_scanned"`
	ImportsScanned  int      `json:"imports_scanned"`
	FindingsTotal   int      `json:"findings_total"`
	FindingsError   int      `json:"findings_error"`
	FindingsWarning int      `json:"findings_warning"`
	ParseErrors     int      `json:"parse_errors"`
	FilesSkipped    int      `json:"files_skipped"`
	ConfigDir       string   `json:"config_dir,omitempty"`
	EffectiveRoots  []string `json:"effective_roots,omitempty"`
	DurationMS      int      `json:"duration_ms"`
}

type CheckReport struct {
	Findings []model.Finding `json:"findings"`
	Summary  Summary         `json:"summary"`
}
