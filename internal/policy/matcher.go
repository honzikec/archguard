package policy

import (
	"path"
	"regexp"

	"github.com/honzikec/archguard/internal/config"
	"github.com/honzikec/archguard/internal/pathutil"
)

func compileRules(rules []config.Rule) ([]compiledRule, error) {
	out := make([]compiledRule, 0, len(rules))
	for _, rule := range rules {
		cr := compiledRule{Rule: rule}
		if rule.Kind == config.KindFilePattern {
			cr.TargetRegexes = make([]compiledRegex, 0, len(rule.Target))
			for _, p := range rule.Target {
				re, err := regexp.Compile(p)
				if err != nil {
					return nil, err
				}
				cr.TargetRegexes = append(cr.TargetRegexes, compiledRegex{Pattern: p, Regexp: re})
			}
		}
		out = append(out, cr)
	}
	return out, nil
}

func matchesScope(scope []string, source string) bool {
	return pathutil.MatchAny(scope, source)
}

func isExcepted(excepts []string, source, target string) bool {
	if len(excepts) == 0 {
		return false
	}
	if pathutil.MatchAny(excepts, source) {
		return true
	}
	if target != "" && pathutil.MatchAny(excepts, target) {
		return true
	}
	return false
}

func packageMatches(targets []string, pkg string) bool {
	for _, t := range targets {
		if t == pkg || pathutil.MatchGlob(t, pkg) {
			return true
		}
	}
	return false
}

func subtree(pathValue string) string {
	return path.Dir(pathValue)
}
