package cli

import (
	"fmt"
	"sort"

	"github.com/honzikec/archguard/internal/analysis"
	"github.com/honzikec/archguard/internal/catalog"
	"github.com/honzikec/archguard/internal/config"
	"github.com/honzikec/archguard/internal/framework"
	"github.com/honzikec/archguard/internal/language"
	"github.com/honzikec/archguard/internal/miner"
	"github.com/honzikec/archguard/internal/model"
	"github.com/honzikec/archguard/internal/policy"
	"github.com/honzikec/archguard/internal/workspace"
)

type guidedInitOutput struct {
	Recommendations guidedRecommendations
	Preset          guidedPreset
	Findings        []model.Finding
	StarterConfig   *config.Config
	StarterText     string
}

type guidedInitState struct {
	sourceConfig   *config.Config
	project        config.ProjectSettings
	workspaceRoots []string
	notes          []string
	preset         guidedPreset
	language       language.Resolution
	framework      framework.Resolution
	analysis       analysis.Result
	candidates     []miner.Candidate
	adoptedRules   []config.Rule
	findings       []model.Finding
	starterConfig  *config.Config
	starterText    string
}

type guidedMiningOptions struct {
	minSupport    int
	maxPrevalence float64
}

func buildGuidedInitRecommendation(cfg *config.Config, adoptThreshold string) (guidedInitOutput, error) {
	state, err := newGuidedInitState(cfg)
	if err != nil {
		return guidedInitOutput{}, err
	}
	if err := state.prepareProject(); err != nil {
		return guidedInitOutput{}, err
	}
	if err := state.runAnalysis(); err != nil {
		return guidedInitOutput{}, err
	}
	if err := state.buildStarterRules(adoptThreshold); err != nil {
		return guidedInitOutput{}, err
	}
	if err := state.renderStarterConfig(); err != nil {
		return guidedInitOutput{}, err
	}
	return state.output(), nil
}

func newGuidedInitState(cfg *config.Config) (guidedInitState, error) {
	if cfg == nil {
		return guidedInitState{}, fmt.Errorf("config is required")
	}
	project := cloneProjectSettings(cfg.Project)
	if project.Aliases == nil {
		project.Aliases = map[string][]string{}
	}
	return guidedInitState{
		sourceConfig:   cfg,
		project:        project,
		workspaceRoots: append([]string{}, cfg.Project.Roots...),
	}, nil
}

func (s *guidedInitState) prepareProject() error {
	initialLanguage := language.Resolve(s.sourceConfig.Project.Language, s.sourceConfig.Project.Roots)
	if initialLanguage.Adapter == nil {
		return fmt.Errorf("failed to resolve language adapter")
	}
	initialFramework := framework.Resolve(s.sourceConfig.Project.Framework, s.sourceConfig.Project.Roots)
	s.preset = detectGuidedPreset(s.project, initialLanguage, initialFramework, s.workspaceRoots)

	if s.preset.ID != phpBrownfieldPresetID {
		discovered, err := workspace.DiscoverRoots(s.sourceConfig.Project.Roots)
		if err != nil {
			return err
		}
		if len(discovered) > 1 && rootsAreDefault(s.sourceConfig.Project.Roots) {
			s.workspaceRoots = discovered
		}
		s.preset = detectGuidedPreset(s.project, initialLanguage, initialFramework, s.workspaceRoots)
	}
	if len(s.workspaceRoots) > 0 {
		s.project.Roots = s.workspaceRoots
	}

	s.notes = applyGuidedProjectDefaults(&s.project, s.preset)
	s.language = language.Resolve(s.project.Language, s.project.Roots)
	if s.language.Adapter == nil {
		return fmt.Errorf("failed to resolve language adapter")
	}
	s.framework = framework.Resolve(s.project.Framework, s.project.Roots)
	s.project.Language = recommendedLanguage(s.sourceConfig.Project.Language, s.language)
	s.project.Framework = recommendedFramework(s.sourceConfig.Project.Framework, s.framework)
	if s.preset.ID != phpBrownfieldPresetID && s.project.Tsconfig == "" && fileExists("tsconfig.json") {
		s.project.Tsconfig = "tsconfig.json"
	}
	return nil
}

func (s *guidedInitState) runAnalysis() error {
	result, err := analysis.Run(s.project, s.language.Adapter, analysis.Options{})
	if err != nil {
		return err
	}
	s.analysis = result
	return nil
}

func (s *guidedInitState) buildStarterRules(adoptThreshold string) error {
	candidates, err := s.proposeCandidates()
	if err != nil {
		return err
	}
	adopted, err := s.adoptCatalogRules(candidates, adoptThreshold)
	if err != nil {
		return err
	}
	s.candidates, s.adoptedRules, s.notes = applyPresetDefaults(candidates, adopted, s.preset, s.project, s.notes)
	return nil
}

func (s *guidedInitState) renderStarterConfig() error {
	emitOptions := miner.EmitOptions{
		NoCycleSeverity: config.SeverityWarning,
		Project:         &s.project,
	}
	s.starterConfig = miner.BuildStarterConfigWithCatalog(s.candidates, s.adoptedRules, emitOptions)
	findings, err := policy.Evaluate(s.starterConfig, s.analysis.Imports, s.analysis.Files, s.analysis.Graph)
	if err != nil {
		return err
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Fingerprint != findings[j].Fingerprint {
			return findings[i].Fingerprint < findings[j].Fingerprint
		}
		return findings[i].FilePath < findings[j].FilePath
	})
	s.findings = findings
	s.starterText = miner.EmitStarterConfigWithCatalog(s.candidates, s.adoptedRules, emitOptions)
	return nil
}

func (s guidedInitState) output() guidedInitOutput {
	return guidedInitOutput{
		Recommendations: guidedRecommendations{
			Language:       s.language,
			Framework:      s.framework,
			WorkspaceRoots: append([]string{}, s.workspaceRoots...),
			Project:        s.project,
			Notes:          append([]string{}, s.notes...),
		},
		Preset:        s.preset,
		Findings:      append([]model.Finding{}, s.findings...),
		StarterConfig: s.starterConfig,
		StarterText:   s.starterText,
	}
}

func (s guidedInitState) proposeCandidates() ([]miner.Candidate, error) {
	normalizedGraph, normalizedFiles, _ := framework.NormalizeMiningInputs(s.analysis.Graph, s.analysis.Files, s.framework.EffectiveProfile())
	options := guidedMiningDefaults(normalizedFiles)
	return miner.Propose(normalizedGraph, normalizedFiles, miner.Options{
		MinSupport:           options.minSupport,
		MaxPrevalence:        options.maxPrevalence,
		MaxCandidatesPerKind: 50,
	}), nil
}

func (s guidedInitState) adoptCatalogRules(candidates []miner.Candidate, adoptThreshold string) ([]config.Rule, error) {
	patterns, err := catalog.LoadBuiltin()
	if err != nil {
		return nil, err
	}
	catalogMatches, err := miner.MatchCatalog(patterns, candidates, s.analysis.Files, s.sourceConfig.Project, miner.CatalogOptions{})
	if err != nil {
		return nil, err
	}
	return miner.AdoptCatalogMatches(catalogMatches, adoptThreshold), nil
}

func guidedMiningDefaults(files []string) guidedMiningOptions {
	options := guidedMiningOptions{
		minSupport:    3,
		maxPrevalence: 0.02,
	}
	if len(files) < 20 {
		options.minSupport = 1
		options.maxPrevalence = 0.25
	}
	return options
}
