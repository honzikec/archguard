package brief

import (
	"fmt"
	"sort"
	"strings"

	"github.com/honzikec/archguard/internal/config"
)

func Compile(spec *Brief) (*config.Config, error) {
	if err := validateSpec(spec); err != nil {
		return nil, err
	}

	layerPaths := map[string][]string{}
	for _, layer := range spec.Layers {
		layerPaths[strings.TrimSpace(layer.ID)] = append([]string{}, layer.Paths...)
	}

	cfg := &config.Config{
		Version: 1,
		Project: compileProject(spec.Project),
		Rules:   make([]config.Rule, 0, len(spec.Policies)),
	}

	usedRuleIDs := map[string]struct{}{}
	for idx, policy := range spec.Policies {
		rule, err := compilePolicyIntent(policy, idx, layerPaths, usedRuleIDs)
		if err != nil {
			return nil, err
		}
		cfg.Rules = append(cfg.Rules, rule)
	}

	if err := config.Validate(cfg); err != nil {
		return nil, fmt.Errorf("compiled config invalid: %w", err)
	}
	return cfg, nil
}

func compileProject(project Project) config.ProjectSettings {
	out := config.DefaultProjectSettings()
	out.Framework = "generic"
	out.Language = "auto"
	out.Aliases = map[string][]string{}

	if len(project.Roots) > 0 {
		out.Roots = append([]string{}, project.Roots...)
	}
	if len(project.Include) > 0 {
		out.Include = append([]string{}, project.Include...)
	}
	if len(project.Exclude) > 0 {
		out.Exclude = append([]string{}, project.Exclude...)
	}
	if framework := strings.TrimSpace(project.Framework); framework != "" {
		out.Framework = framework
	}
	if language := strings.TrimSpace(project.Language); language != "" {
		out.Language = language
	}
	if tsconfig := strings.TrimSpace(project.Tsconfig); tsconfig != "" {
		out.Tsconfig = tsconfig
	}
	if project.Aliases != nil {
		out.Aliases = copyAliases(project.Aliases)
	}
	return out
}

func compilePolicyIntent(policy PolicyIntent, index int, layerPaths map[string][]string, usedRuleIDs map[string]struct{}) (config.Rule, error) {
	policyType := strings.TrimSpace(policy.Type)
	ruleID, err := resolveRuleID(policy.ID, policyType, index, usedRuleIDs)
	if err != nil {
		return config.Rule{}, err
	}
	severity := strings.TrimSpace(policy.Severity)
	if severity == "" {
		severity = config.SeverityWarning
	}
	message := strings.TrimSpace(policy.Message)

	resolve := func(selectors []string, fieldName string) ([]string, error) {
		return resolveSelectors(selectors, layerPaths, fieldName, policyType)
	}

	switch policyType {
	case "deny_import":
		scope, err := resolve(policy.From, "from")
		if err != nil {
			return config.Rule{}, err
		}
		target, err := resolve(policy.To, "to")
		if err != nil {
			return config.Rule{}, err
		}
		except, err := resolve(policy.Except, "except")
		if err != nil {
			return config.Rule{}, err
		}
		return config.Rule{
			ID:       ruleID,
			Kind:     config.KindNoImport,
			Severity: severity,
			Scope:    scope,
			Target:   target,
			Except:   except,
			Message:  message,
		}, nil

	case "deny_package":
		scope, err := resolve(policy.Scope, "scope")
		if err != nil {
			return config.Rule{}, err
		}
		except, err := resolve(policy.Except, "except")
		if err != nil {
			return config.Rule{}, err
		}
		packages := normalizeStringList(policy.Packages)
		if len(packages) == 0 {
			return config.Rule{}, fmt.Errorf("policy %s type deny_package requires packages", ruleID)
		}
		return config.Rule{
			ID:       ruleID,
			Kind:     config.KindNoPackage,
			Severity: severity,
			Scope:    scope,
			Target:   packages,
			Except:   except,
			Message:  message,
		}, nil

	case "file_pattern":
		scope, err := resolve(policy.Scope, "scope")
		if err != nil {
			return config.Rule{}, err
		}
		except, err := resolve(policy.Except, "except")
		if err != nil {
			return config.Rule{}, err
		}
		pattern := strings.TrimSpace(policy.Pattern)
		if pattern == "" {
			return config.Rule{}, fmt.Errorf("policy %s type file_pattern requires pattern", ruleID)
		}
		return config.Rule{
			ID:       ruleID,
			Kind:     config.KindFilePattern,
			Severity: severity,
			Scope:    scope,
			Target:   []string{pattern},
			Except:   except,
			Message:  message,
		}, nil

	case "no_cycle":
		scope, err := resolve(policy.Scope, "scope")
		if err != nil {
			return config.Rule{}, err
		}
		except, err := resolve(policy.Except, "except")
		if err != nil {
			return config.Rule{}, err
		}
		return config.Rule{
			ID:       ruleID,
			Kind:     config.KindNoCycle,
			Severity: severity,
			Scope:    scope,
			Except:   except,
			Message:  message,
		}, nil

	case "construction_policy":
		scope, err := resolve(policy.Scope, "scope")
		if err != nil {
			return config.Rule{}, err
		}
		services, err := resolve(policy.Services, "services")
		if err != nil {
			return config.Rule{}, err
		}
		allowIn, err := resolve(policy.AllowIn, "allow_in")
		if err != nil {
			return config.Rule{}, err
		}
		except, err := resolve(policy.Except, "except")
		if err != nil {
			return config.Rule{}, err
		}
		except = append(allowIn, except...)
		except = dedupeStrings(except)

		params := map[string]string{}
		if regex := strings.TrimSpace(policy.ServiceNameRegex); regex != "" {
			params["service_name_regex"] = regex
		}
		if len(params) == 0 {
			params = nil
		}
		return config.Rule{
			ID:       ruleID,
			Kind:     config.KindPattern,
			Severity: severity,
			Scope:    scope,
			Target:   services,
			Except:   except,
			Template: "construction_policy",
			Params:   params,
			Message:  message,
		}, nil
	default:
		return config.Rule{}, fmt.Errorf("unsupported policy type %q", policyType)
	}
}

func resolveSelectors(selectors []string, layers map[string][]string, fieldName, policyType string) ([]string, error) {
	if len(selectors) == 0 {
		return nil, nil
	}

	out := make([]string, 0)
	for _, selector := range selectors {
		selector = strings.TrimSpace(selector)
		if selector == "" {
			continue
		}
		if strings.HasPrefix(selector, "layer:") {
			layerID := strings.TrimSpace(strings.TrimPrefix(selector, "layer:"))
			layerGlobs, ok := layers[layerID]
			if !ok {
				return nil, fmt.Errorf("policy type %s references unknown layer %q in %s", policyType, selector, fieldName)
			}
			out = append(out, layerGlobs...)
			continue
		}
		out = append(out, selector)
	}
	out = dedupeStrings(out)
	return out, nil
}

func resolveRuleID(rawID, policyType string, index int, used map[string]struct{}) (string, error) {
	if id := strings.TrimSpace(rawID); id != "" {
		if _, exists := used[id]; exists {
			return "", fmt.Errorf("duplicate policy id: %s", id)
		}
		used[id] = struct{}{}
		return id, nil
	}

	normalizedType := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(policyType), "-", "_"))
	if normalizedType == "" {
		normalizedType = "POLICY"
	}
	base := "AG-BRIEF-" + normalizedType
	start := index + 1
	for i := start; ; i++ {
		candidate := fmt.Sprintf("%s-%03d", base, i)
		if _, exists := used[candidate]; exists {
			continue
		}
		used[candidate] = struct{}{}
		return candidate, nil
	}
}

func copyAliases(in map[string][]string) map[string][]string {
	if in == nil {
		return nil
	}
	out := make(map[string][]string, len(in))
	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		out[key] = append([]string{}, in[key]...)
	}
	return out
}

func normalizeStringList(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return dedupeStrings(out)
}

func dedupeStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
