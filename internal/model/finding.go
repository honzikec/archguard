package model

type Finding struct {
	RuleID      string
	RuleKind    string
	Severity    string
	Message     string
	FilePath    string
	Line        int
	Column      int
	RawImport   string
	Fingerprint string
	Details     string
}
