package contracts

import "github.com/honzikec/archguard/internal/model"

type Detection struct {
	Matched bool
	Reason  string
}

type ParseDiagnostics struct {
	NonLiteralDynamicImports int
}

type Adapter interface {
	ID() string
	Detect(roots []string) Detection
	SupportsFile(path string) bool
	ParseFile(path string) ([]model.ImportRef, error)
}

type DiagnosticAdapter interface {
	ParseFileWithDiagnostics(path string) ([]model.ImportRef, ParseDiagnostics, error)
}
