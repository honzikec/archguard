package contracts

type Detection struct {
	Matched bool
	Reason  string
	Score   int
}

type Profile interface {
	ID() string
	Detect(roots []string) Detection
	NormalizeSubtree(subtree string) string
	NormalizeFile(file string) string
}
