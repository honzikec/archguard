package baseline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/honzikec/archguard/internal/model"
)

type File struct {
	Version     int     `json:"version"`
	GeneratedAt string  `json:"generated_at,omitempty"`
	Findings    []Entry `json:"findings"`
}

type Entry struct {
	Fingerprint string `json:"fingerprint"`
	RuleID      string `json:"rule_id"`
	RuleKind    string `json:"rule_kind"`
	Severity    string `json:"severity"`
	FilePath    string `json:"file_path"`
	Line        int    `json:"line"`
	Column      int    `json:"column"`
	Message     string `json:"message,omitempty"`
	RawImport   string `json:"raw_import,omitempty"`
	Details     string `json:"details,omitempty"`
}

func Write(path string, findings []model.Finding, generatedAt time.Time) error {
	entries := make([]Entry, 0, len(findings))
	for _, finding := range findings {
		if finding.Fingerprint == "" {
			continue
		}
		entries = append(entries, entryFromFinding(finding))
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Fingerprint != entries[j].Fingerprint {
			return entries[i].Fingerprint < entries[j].Fingerprint
		}
		return entries[i].FilePath < entries[j].FilePath
	})

	payload := File{
		Version:     1,
		GeneratedAt: generatedAt.UTC().Format(time.RFC3339),
		Findings:    entries,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, data, 0o644)
}

func Load(path string) (map[string]Entry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var payload File
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse baseline: %w", err)
	}
	if payload.Version != 1 {
		return nil, fmt.Errorf("unsupported baseline version %d, expected 1", payload.Version)
	}
	out := make(map[string]Entry, len(payload.Findings))
	for _, entry := range payload.Findings {
		if entry.Fingerprint == "" {
			continue
		}
		out[entry.Fingerprint] = entry
	}
	return out, nil
}

func Filter(findings []model.Finding, entries map[string]Entry) ([]model.Finding, int) {
	if len(entries) == 0 {
		return findings, 0
	}
	out := make([]model.Finding, 0, len(findings))
	suppressed := 0
	for _, finding := range findings {
		if _, ok := entries[finding.Fingerprint]; ok && finding.Fingerprint != "" {
			suppressed++
			continue
		}
		out = append(out, finding)
	}
	return out, suppressed
}

func entryFromFinding(f model.Finding) Entry {
	return Entry{
		Fingerprint: f.Fingerprint,
		RuleID:      f.RuleID,
		RuleKind:    f.RuleKind,
		Severity:    f.Severity,
		FilePath:    f.FilePath,
		Line:        f.Line,
		Column:      f.Column,
		Message:     f.Message,
		RawImport:   f.RawImport,
		Details:     f.Details,
	}
}
