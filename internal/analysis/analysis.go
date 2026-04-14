package analysis

import (
	"fmt"
	"io"

	"github.com/honzikec/archguard/internal/config"
	"github.com/honzikec/archguard/internal/fileset"
	"github.com/honzikec/archguard/internal/graph"
	"github.com/honzikec/archguard/internal/language/contracts"
	"github.com/honzikec/archguard/internal/model"
	"github.com/honzikec/archguard/internal/pathutil"
)

type Options struct {
	FileFilter   func([]string) ([]string, error)
	TraceImports io.Writer
}

type Diagnostics struct {
	ParseErrors              int
	FilesSkipped             int
	NonLiteralDynamicImports int
}

type Result struct {
	Files       []string
	Imports     []model.ImportRef
	Graph       *graph.Graph
	Diagnostics Diagnostics
}

func Run(project config.ProjectSettings, adapter contracts.Adapter, opts Options) (Result, error) {
	if adapter == nil {
		return Result{}, fmt.Errorf("language adapter is nil")
	}

	files, err := fileset.DiscoverWithAdapter(project, adapter)
	if err != nil {
		return Result{}, fmt.Errorf("failed to discover files: %w", err)
	}
	if opts.FileFilter != nil {
		files, err = opts.FileFilter(files)
		if err != nil {
			return Result{}, err
		}
	}

	resolver, err := pathutil.NewResolver(".", project)
	if err != nil {
		return Result{}, fmt.Errorf("failed to initialize resolver: %w", err)
	}

	allImports := make([]model.ImportRef, 0)
	diagnostics := Diagnostics{}
	diagnosticAdapter, hasDiagnostics := adapter.(contracts.DiagnosticAdapter)
	for _, file := range files {
		var parsed []model.ImportRef
		var parseDiagnostics contracts.ParseDiagnostics
		var parseErr error
		if hasDiagnostics {
			parsed, parseDiagnostics, parseErr = diagnosticAdapter.ParseFileWithDiagnostics(file)
		} else {
			parsed, parseErr = adapter.ParseFile(file)
		}
		if parseErr != nil {
			diagnostics.ParseErrors++
			diagnostics.FilesSkipped++
			continue
		}
		diagnostics.NonLiteralDynamicImports += parseDiagnostics.NonLiteralDynamicImports
		for i := range parsed {
			resolved, isPackage := resolver.Resolve(parsed[i].SourceFile, parsed[i].RawImport)
			parsed[i].ResolvedPath = resolved
			parsed[i].IsPackageImport = isPackage
			if opts.TraceImports != nil {
				fmt.Fprintf(opts.TraceImports, "%s -> %s (resolved=%s package=%t)\n", parsed[i].SourceFile, parsed[i].RawImport, parsed[i].ResolvedPath, parsed[i].IsPackageImport)
			}
		}
		allImports = append(allImports, parsed...)
	}

	return Result{
		Files:       files,
		Imports:     allImports,
		Graph:       graph.Build(allImports, files),
		Diagnostics: diagnostics,
	}, nil
}
