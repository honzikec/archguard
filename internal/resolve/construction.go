package resolve

import (
	"path/filepath"
	"regexp"

	"github.com/honzikec/archguard/internal/astfacts"
	"github.com/honzikec/archguard/internal/config"
	"github.com/honzikec/archguard/internal/pathutil"
)

type ResolvedConstruction struct {
	FilePath         string
	Line             int
	Column           int
	ClassName        string
	ResolvedFile     string
	ResolvedClass    string
	IsResolved       bool
	IsService        bool
	UnresolvedReason string
}

type importSymbol struct {
	path         string
	kind         string
	importedName string
}

func ResolveConstructions(files []string, project config.ProjectSettings, serviceGlobs []string, serviceNameRegex string) ([]ResolvedConstruction, error) {
	factsByFile := map[string]astfacts.FileFacts{}
	serviceClassByFile := map[string]map[string]struct{}{}

	compiledServiceRegex := regexp.MustCompile(`.*Service$`)
	if serviceNameRegex != "" {
		re, err := regexp.Compile(serviceNameRegex)
		if err != nil {
			return nil, err
		}
		compiledServiceRegex = re
	}

	for _, file := range files {
		facts, err := astfacts.ParseFile(file)
		if err != nil {
			continue
		}
		factsByFile[file] = facts
		if !pathutil.MatchAny(serviceGlobs, file) {
			continue
		}
		for _, cls := range facts.Classes {
			if !compiledServiceRegex.MatchString(cls.Name) {
				continue
			}
			if serviceClassByFile[file] == nil {
				serviceClassByFile[file] = map[string]struct{}{}
			}
			serviceClassByFile[file][cls.Name] = struct{}{}
		}
	}

	resolver, err := pathutil.NewResolver(".", project)
	if err != nil {
		return nil, err
	}

	resolved := make([]ResolvedConstruction, 0)
	for _, file := range files {
		facts, ok := factsByFile[file]
		if !ok {
			continue
		}

		localClasses := map[string]struct{}{}
		for _, cls := range facts.Classes {
			localClasses[cls.Name] = struct{}{}
		}

		imports := map[string]importSymbol{}
		for _, imp := range facts.Imports {
			resolvedPath, isPackage := resolver.Resolve(file, imp.Module)
			if isPackage || resolvedPath == "" {
				continue
			}
			if imp.Default != "" {
				imports[imp.Default] = importSymbol{
					path:         resolvedPath,
					kind:         "default",
					importedName: "default",
				}
			}
			for local, imported := range imp.Named {
				imports[local] = importSymbol{
					path:         resolvedPath,
					kind:         "named",
					importedName: imported,
				}
			}
		}

		for _, n := range facts.NewExprs {
			r := ResolvedConstruction{
				FilePath:  file,
				Line:      n.Line,
				Column:    n.Column,
				ClassName: n.ClassName,
			}

			if !n.IsIdentifier || n.ClassName == "" {
				r.UnresolvedReason = "dynamic_constructor"
				resolved = append(resolved, r)
				continue
			}

			if _, ok := localClasses[n.ClassName]; ok {
				r.IsResolved = true
				r.ResolvedFile = file
				r.ResolvedClass = n.ClassName
				if _, ok := serviceClassByFile[file][n.ClassName]; ok {
					r.IsService = true
				}
				resolved = append(resolved, r)
				continue
			}

			if imp, ok := imports[n.ClassName]; ok {
				targetFacts, ok := factsByFile[imp.path]
				if !ok {
					r.UnresolvedReason = "import_target_not_parsed"
					resolved = append(resolved, r)
					continue
				}
				resolvedClass, ok := resolveImportedClass(targetFacts, imp)
				if !ok {
					if imp.kind == "default" {
						r.UnresolvedReason = "default_export_not_class"
					} else {
						r.UnresolvedReason = "imported_symbol_not_class"
					}
					resolved = append(resolved, r)
					continue
				}

				r.IsResolved = true
				r.ResolvedFile = imp.path
				r.ResolvedClass = resolvedClass
				if _, ok := serviceClassByFile[imp.path][resolvedClass]; ok {
					r.IsService = true
				}
				resolved = append(resolved, r)
				continue
			}

			r.UnresolvedReason = "symbol_not_resolved"
			resolved = append(resolved, r)
		}
	}

	// normalize order for deterministic output
	for i := range resolved {
		resolved[i].FilePath = filepath.ToSlash(resolved[i].FilePath)
		resolved[i].ResolvedFile = filepath.ToSlash(resolved[i].ResolvedFile)
	}

	return resolved, nil
}

func resolveImportedClass(facts astfacts.FileFacts, imp importSymbol) (string, bool) {
	if imp.kind == "default" {
		if facts.DefaultExportedClass == "" {
			return "", false
		}
		if !hasClass(facts, facts.DefaultExportedClass) {
			return "", false
		}
		return facts.DefaultExportedClass, true
	}

	className, ok := facts.ExportedClassByName[imp.importedName]
	if !ok || className == "" {
		return "", false
	}
	if !hasClass(facts, className) {
		return "", false
	}
	return className, true
}

func hasClass(facts astfacts.FileFacts, className string) bool {
	for _, cls := range facts.Classes {
		if cls.Name == className {
			return true
		}
	}
	return false
}
