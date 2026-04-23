package policy

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/honzikec/archguard/internal/config"
	"github.com/honzikec/archguard/internal/model"
	"github.com/honzikec/archguard/internal/pathutil"
	"github.com/honzikec/archguard/internal/pkgid"
)

func firstMatchingScope(scope []string, source string) string {
	for _, pattern := range scope {
		if pathutil.MatchGlob(pattern, source) {
			return pattern
		}
	}
	return ""
}

func firstMatchingTarget(rule config.Rule, target string) string {
	if target == "" {
		return ""
	}
	switch rule.Kind {
	case config.KindNoPackage:
		for _, pattern := range rule.Target {
			if pattern == target || pathutil.MatchGlob(pattern, target) {
				return pattern
			}
			canonical := pkgid.Canonical(target)
			if canonical != target && (pattern == canonical || pathutil.MatchGlob(pattern, canonical)) {
				return pattern
			}
		}
	case config.KindFilePattern:
		base := path.Base(target)
		for _, pattern := range rule.Target {
			if compiled, err := regexp.Compile(pattern); err == nil && compiled.MatchString(base) {
				return pattern
			}
		}
	default:
		for _, pattern := range rule.Target {
			if pathutil.MatchGlob(pattern, target) {
				return pattern
			}
		}
	}
	return ""
}

func targetPreview(rule config.Rule) string {
	if len(rule.Target) == 0 {
		return ""
	}
	if len(rule.Target) == 1 {
		return rule.Target[0]
	}
	return strings.Join(rule.Target, ", ")
}

func RuleWhy(rule config.Rule) string {
	switch rule.Kind {
	case config.KindNoImport:
		return "This rule keeps one part of the codebase from depending on a forbidden local path."
	case config.KindNoPackage:
		return "This rule keeps scoped code away from packages that should stay outside that layer."
	case config.KindFilePattern:
		return "This rule keeps file naming predictable so architectural roles stay easy to scan and review."
	case config.KindNoCycle:
		return "This rule prevents dependency cycles so modules stay easier to change, test, and reason about."
	case config.KindPattern:
		switch rule.Template {
		case "construction_policy":
			return "This rule reserves service construction for composition roots instead of feature code."
		case "dependency_constraint":
			return "This rule encodes a catalog or template-based dependency constraint."
		}
	}
	return "This rule protects an architectural invariant in the repository."
}

func RuleRemediation(rule config.Rule) string {
	switch rule.Kind {
	case config.KindNoImport:
		return "Move the dependency behind an allowed boundary, invert the dependency, or narrow the rule with an intentional exception."
	case config.KindNoPackage:
		return "Replace the package in this layer, move the code to an allowed layer, or carve out a narrowly scoped exception."
	case config.KindFilePattern:
		return "Rename the file to match the expected pattern or adjust the rule if the naming convention changed intentionally."
	case config.KindNoCycle:
		return "Break one import edge in the cycle by introducing an interface, event, shared abstraction, or composition root."
	case config.KindPattern:
		switch rule.Template {
		case "construction_policy":
			return "Construct services in a composition root and pass them in, rather than instantiating them directly here."
		case "dependency_constraint":
			return "Move the dependency to an allowed boundary or refine the catalog-derived constraint if the architecture changed intentionally."
		}
	}
	return "Move the code back within the intended architectural boundary or update the rule deliberately."
}

func FindingOptions(rule config.Rule) string {
	return fmt.Sprintf("Exception path(s): %v. For brownfield adoption, use `archguard check --write-baseline archguard-baseline.json`.", rule.Except)
}

func buildEvidence(rule config.Rule, source, target, rawImport string) string {
	scope := firstMatchingScope(rule.Scope, source)
	matchedTarget := firstMatchingTarget(rule, target)
	parts := make([]string, 0, 4)
	if scope != "" {
		parts = append(parts, fmt.Sprintf("source %s matched scope %s", source, scope))
	}
	if rawImport != "" {
		parts = append(parts, fmt.Sprintf("import %s", rawImport))
	}
	if target != "" {
		if matchedTarget != "" {
			parts = append(parts, fmt.Sprintf("target %s matched %s", target, matchedTarget))
		} else {
			parts = append(parts, fmt.Sprintf("target %s", target))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "; ")
}

func applyFindingMetadata(rule config.Rule, finding *model.Finding, source, target string) {
	if finding == nil {
		return
	}
	finding.MatchedScope = firstMatchingScope(rule.Scope, source)
	finding.MatchedTarget = firstMatchingTarget(rule, target)
	finding.Evidence = buildEvidence(rule, source, target, finding.RawImport)
	finding.Remediation = RuleRemediation(rule)
}
