package model

type Finding struct {
	RuleID    string
	RuleKind  string
	Severity  string
	Message   string
	Rationale string
	FilePath  string
	Line      int
	Column    int
	RawImport string
}
