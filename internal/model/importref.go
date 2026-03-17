package model

type ImportRef struct {
	SourceFile      string
	RawImport       string
	ResolvedPath    string
	IsPackageImport bool
	Line            int
	Column          int
	Kind            string
}
