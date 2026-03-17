package policy

import "github.com/honzikec/archguard/internal/config"

type compiledRule struct {
	Rule          config.Rule
	TargetRegexes []compiledRegex
}

type compiledRegex struct {
	Pattern string
	Regexp  regexLike
}

type regexLike interface {
	MatchString(string) bool
}
