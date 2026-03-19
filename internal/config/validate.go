package config

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/honzikec/archguard/internal/framework"
	"github.com/honzikec/archguard/internal/language"
)

func Validate(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if cfg.Version != 1 {
		return fmt.Errorf("unsupported config version %d, expected 1", cfg.Version)
	}

	if err := validateProject(cfg.Project); err != nil {
		return err
	}

	seen := map[string]struct{}{}
	for i, rule := range cfg.Rules {
		if rule.ID == "" {
			return fmt.Errorf("rules[%d].id is required", i)
		}
		if _, ok := seen[rule.ID]; ok {
			return fmt.Errorf("duplicate rule id: %s", rule.ID)
		}
		seen[rule.ID] = struct{}{}

		if !isValidKind(rule.Kind) {
			return fmt.Errorf("rule %s has unsupported kind %q", rule.ID, rule.Kind)
		}
		if !isValidSeverity(rule.Severity) {
			return fmt.Errorf("rule %s has unsupported severity %q", rule.ID, rule.Severity)
		}
		if len(rule.Scope) == 0 {
			return fmt.Errorf("rule %s must define scope", rule.ID)
		}
		for _, p := range rule.Scope {
			if err := validateGlob(p); err != nil {
				return fmt.Errorf("rule %s scope pattern %q invalid: %w", rule.ID, p, err)
			}
		}
		for _, p := range rule.Except {
			if err := validateGlob(p); err != nil {
				return fmt.Errorf("rule %s except pattern %q invalid: %w", rule.ID, p, err)
			}
		}
		if err := validateRuleByKind(rule); err != nil {
			return err
		}
	}

	return nil
}

func validateProject(project ProjectSettings) error {
	if frameworkID := strings.ToLower(strings.TrimSpace(project.Framework)); frameworkID != "" {
		allowed := map[string]struct{}{}
		for _, id := range framework.RegisteredFrameworks() {
			allowed[id] = struct{}{}
		}
		if _, ok := allowed[frameworkID]; !ok {
			return fmt.Errorf("project.framework has unsupported value %q", project.Framework)
		}
	}
	if languageID := strings.ToLower(strings.TrimSpace(project.Language)); languageID != "" && languageID != "auto" {
		allowed := map[string]struct{}{}
		for _, id := range language.RegisteredLanguages() {
			allowed[id] = struct{}{}
		}
		if _, ok := allowed[languageID]; !ok {
			return fmt.Errorf("project.language has unsupported value %q", project.Language)
		}
	}
	for _, root := range project.Roots {
		if strings.TrimSpace(root) == "" {
			return fmt.Errorf("project.roots contains empty path")
		}
	}
	for _, p := range project.Include {
		if err := validateGlob(p); err != nil {
			return fmt.Errorf("project.include pattern %q invalid: %w", p, err)
		}
	}
	for _, p := range project.Exclude {
		if err := validateGlob(p); err != nil {
			return fmt.Errorf("project.exclude pattern %q invalid: %w", p, err)
		}
	}
	for alias, targets := range project.Aliases {
		if strings.TrimSpace(alias) == "" {
			return fmt.Errorf("project.aliases contains empty alias key")
		}
		if len(targets) == 0 {
			return fmt.Errorf("project.aliases[%s] must contain at least one target", alias)
		}
		for _, t := range targets {
			if strings.TrimSpace(t) == "" {
				return fmt.Errorf("project.aliases[%s] contains empty target", alias)
			}
		}
	}
	return nil
}

func validateRuleByKind(rule Rule) error {
	switch rule.Kind {
	case KindNoImport:
		if len(rule.Target) == 0 {
			return fmt.Errorf("rule %s kind no_import requires target patterns", rule.ID)
		}
		for _, p := range rule.Target {
			if err := validateGlob(p); err != nil {
				return fmt.Errorf("rule %s target pattern %q invalid: %w", rule.ID, p, err)
			}
		}
	case KindNoPackage:
		if len(rule.Target) == 0 {
			return fmt.Errorf("rule %s kind no_package requires target packages", rule.ID)
		}
	case KindFilePattern:
		if len(rule.Target) == 0 {
			return fmt.Errorf("rule %s kind file_pattern requires regex target", rule.ID)
		}
		for _, pattern := range rule.Target {
			if _, err := regexp.Compile(pattern); err != nil {
				return fmt.Errorf("rule %s target regex %q invalid: %w", rule.ID, pattern, err)
			}
		}
	case KindNoCycle:
		if len(rule.Target) > 0 {
			return fmt.Errorf("rule %s kind no_cycle does not support target", rule.ID)
		}
	case KindPattern:
		if strings.TrimSpace(rule.Template) == "" {
			return fmt.Errorf("rule %s kind pattern requires template", rule.ID)
		}
		if err := validatePatternTemplate(rule); err != nil {
			return err
		}
	default:
		return fmt.Errorf("rule %s has unknown kind %q", rule.ID, rule.Kind)
	}
	return nil
}

func isValidKind(kind string) bool {
	switch kind {
	case KindNoImport, KindNoPackage, KindFilePattern, KindNoCycle, KindPattern:
		return true
	default:
		return false
	}
}

func validatePatternTemplate(rule Rule) error {
	switch rule.Template {
	case "dependency_constraint":
		if len(rule.Target) == 0 {
			return fmt.Errorf("rule %s template dependency_constraint requires target", rule.ID)
		}
		relation := "imports"
		if rule.Params != nil && strings.TrimSpace(rule.Params["relation"]) != "" {
			relation = rule.Params["relation"]
		}
		switch relation {
		case "imports":
			for _, p := range rule.Target {
				if err := validateGlob(p); err != nil {
					return fmt.Errorf("rule %s target pattern %q invalid: %w", rule.ID, p, err)
				}
			}
		case "packages":
			// package globs are validated at evaluation time via matcher semantics.
		default:
			return fmt.Errorf("rule %s template dependency_constraint unsupported relation %q", rule.ID, relation)
		}
	case "construction_policy":
		if len(rule.Target) == 0 {
			return fmt.Errorf("rule %s template construction_policy requires target service globs", rule.ID)
		}
		for _, p := range rule.Target {
			if err := validateGlob(p); err != nil {
				return fmt.Errorf("rule %s target pattern %q invalid: %w", rule.ID, p, err)
			}
		}
		if rule.Params != nil && strings.TrimSpace(rule.Params["service_name_regex"]) != "" {
			if _, err := regexp.Compile(rule.Params["service_name_regex"]); err != nil {
				return fmt.Errorf("rule %s invalid service_name_regex: %w", rule.ID, err)
			}
		}
	default:
		return fmt.Errorf("rule %s has unsupported pattern template %q", rule.ID, rule.Template)
	}
	return nil
}

func isValidSeverity(severity string) bool {
	switch severity {
	case SeverityError, SeverityWarning:
		return true
	default:
		return false
	}
}

func validateGlob(pattern string) error {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return fmt.Errorf("glob pattern cannot be empty")
	}
	// basic validation by stripping doublestar for filepath.Match compatibility
	trial := strings.ReplaceAll(pattern, "**", "*")
	if _, err := filepath.Match(trial, "x"); err != nil {
		return err
	}
	return nil
}
