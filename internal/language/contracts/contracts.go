package contracts

import "github.com/honzikec/archguard/internal/model"

type Detection struct {
	Matched bool
	Reason  string
}

type Adapter interface {
	ID() string
	Detect(roots []string) Detection
	SupportsFile(path string) bool
	ParseFile(path string) ([]model.ImportRef, error)
}
