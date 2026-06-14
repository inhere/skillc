package profileapp

import (
	"fmt"
	"sort"
	"strings"

	"github.com/inhere/skillc/internal/app/apputil"
	"github.com/inhere/skillc/internal/app/installapp"
	"github.com/inhere/skillc/internal/app/listapp"
	"github.com/inhere/skillc/internal/app/searchapp"
	"github.com/inhere/skillc/internal/domain/agent"
	cfg "github.com/inhere/skillc/internal/domain/config"
	"github.com/inhere/skillc/internal/domain/profile"
	"github.com/inhere/skillc/internal/domain/skill"
	"github.com/inhere/skillc/internal/infra/agentfs"
	"github.com/inhere/skillc/internal/infra/configstore"
	"github.com/inhere/skillc/internal/infra/repoindex"
)

type Service struct {
	configFile string
	baseDir    string
	store      *configstore.YAMLStore
}

type CreateFromInstalledReq struct {
	Agent   string
	Scope   string
	WorkDir string
}

type ApplyReq struct {
	Agent   string
	Scope   string
	WorkDir string
}

type ApplyResult struct {
	Plan          profile.ApplyPlan
	Installed     []installapp.RuntimeRecord
	InstallFailed []installapp.InstallItemError
}

func NewService(configFile string, baseDir string) *Service {
	return &Service{
		configFile: configFile,
		baseDir:    baseDir,
		store:      configstore.NewYAMLStore(),
	}
}

func (s *Service) List() ([]profile.NamedProfile, error) {
	config, err := s.store.Load(s.configFile, s.baseDir)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(config.Profiles))
	for name := range config.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]profile.NamedProfile, 0, len(names))
	for _, name := range names {
		out = append(out, profile.NamedProfile{Name: name, Profile: config.Profiles[name]})
	}
	return out, nil
}

func (s *Service) Show(name string) (profile.Profile, error) {
	if err := profile.ValidateName(name); err != nil {
		return profile.Profile{}, err
	}

	config, err := s.store.Load(s.configFile, s.baseDir)
	if err != nil {
		return profile.Profile{}, err
	}

	item, ok := config.Profiles[name]
	if !ok {
		return profile.Profile{}, fmt.Errorf("profile not found: %s", name)
	}
	return item, nil
}

func (s *Service) Create(name string, item profile.Profile) (profile.Profile, error) {
	if err := profile.ValidateName(name); err != nil {
		return profile.Profile{}, err
	}

	config, err := s.store.Load(s.configFile, s.baseDir)
	if err != nil {
		return profile.Profile{}, err
	}
	if _, ok := config.Profiles[name]; ok {
		return profile.Profile{}, fmt.Errorf("profile already exists: %s", name)
	}

	if err := s.saveLoaded(config, name, item); err != nil {
		return profile.Profile{}, err
	}
	return s.Show(name)
}

func (s *Service) CreateFromInstalled(name string, req CreateFromInstalledReq) (profile.Profile, error) {
	if err := profile.ValidateName(name); err != nil {
		return profile.Profile{}, err
	}

	config, err := s.store.Load(s.configFile, s.baseDir)
	if err != nil {
		return profile.Profile{}, err
	}
	agentName := firstNonEmpty(req.Agent, agent.DefaultAgentName)
	canonicalAgent, _, ok := config.ResolveAgentTool(agentName)
	if !ok {
		return profile.Profile{}, fmt.Errorf("unsupported agent: %s", agentName)
	}
	scope, err := apputil.ParseScope(firstNonEmpty(req.Scope, string(agent.ScopeProject)))
	if err != nil {
		return profile.Profile{}, err
	}
	workDir := firstNonEmpty(req.WorkDir, s.baseDir)
	items, err := listapp.NewService(config.LockFile).WithRuntime(config, workDir).List(canonicalAgent, string(scope))
	if err != nil {
		return profile.Profile{}, err
	}

	targets := make([]profile.Target, 0, len(items))
	for _, item := range items {
		if item.Status != "installed" {
			continue
		}
		targets = append(targets, profile.Target{
			Source: item.SourceID,
			Skill:  item.SkillID,
		})
	}
	return s.Create(name, profile.Profile{
		DefaultAgent: canonicalAgent,
		DefaultScope: string(scope),
		Targets:      targets,
	})
}

func (s *Service) CreateFromCollection(name string, selector string) (profile.Profile, error) {
	if err := profile.ValidateName(name); err != nil {
		return profile.Profile{}, err
	}

	sourceID, collection, ok := strings.Cut(selector, "/")
	if !ok || strings.TrimSpace(sourceID) == "" || strings.TrimSpace(collection) == "" || strings.Contains(collection, "/") {
		return profile.Profile{}, fmt.Errorf("collection selector must be <source>/<collection>: %s", selector)
	}

	config, err := s.store.Load(s.configFile, s.baseDir)
	if err != nil {
		return profile.Profile{}, err
	}
	items, err := searchapp.NewService(config.IndexFile).ListSourceSkills(strings.TrimSpace(sourceID), strings.TrimSpace(collection))
	if err != nil {
		return profile.Profile{}, err
	}

	targets := make([]profile.Target, 0, len(items))
	for _, item := range items {
		targets = append(targets, profile.Target{
			Source: item.SourceID,
			Skill:  item.ID,
		})
	}
	return s.Create(name, profile.Profile{Targets: targets})
}

func (s *Service) PlanApply(name string, req ApplyReq) (profile.ApplyPlan, error) {
	item, err := s.Show(name)
	if err != nil {
		return profile.ApplyPlan{}, err
	}

	config, err := s.store.Load(s.configFile, s.baseDir)
	if err != nil {
		return profile.ApplyPlan{}, err
	}
	agentName := firstNonEmpty(req.Agent, item.DefaultAgent, agent.DefaultAgentName)
	canonicalAgent, _, ok := config.ResolveAgentTool(agentName)
	if !ok {
		return profile.ApplyPlan{}, fmt.Errorf("unsupported agent: %s", agentName)
	}
	scope, err := apputil.ParseScope(firstNonEmpty(req.Scope, item.DefaultScope, string(agent.ScopeProject)))
	if err != nil {
		return profile.ApplyPlan{}, err
	}
	workDir := firstNonEmpty(req.WorkDir, s.baseDir)

	indexItems, err := repoindex.NewStore().Load(config.IndexFile)
	if err != nil {
		return profile.ApplyPlan{}, err
	}
	installed, err := listapp.NewService(config.LockFile).WithRuntime(config, workDir).List(canonicalAgent, string(scope))
	if err != nil {
		return profile.ApplyPlan{}, err
	}

	installedSet := make(map[string]struct{}, len(installed)*2)
	for _, current := range installed {
		if current.Status != "installed" {
			continue
		}
		installedSet[current.SourceID+"\x00"+current.SkillID] = struct{}{}
		installedSet["\x00"+current.SkillID] = struct{}{}
	}

	plan := profile.ApplyPlan{Profile: name, Agent: canonicalAgent, Scope: string(scope)}
	for _, target := range item.Targets {
		found, reason, ok := findTargetSkill(indexItems, target)
		if !ok {
			plan.Items = append(plan.Items, profile.ApplyPlanItem{
				Action: "error",
				Target: target,
				Reason: reason,
			})
			continue
		}
		if isTargetInstalled(installedSet, target, found) {
			plan.Items = append(plan.Items, profile.ApplyPlanItem{
				Action: "skip",
				Target: target,
				Skill:  found,
				Reason: "already installed",
			})
			continue
		}
		plan.Items = append(plan.Items, profile.ApplyPlanItem{
			Action: "install",
			Target: target,
			Skill:  found,
		})
	}
	return plan, nil
}

func (s *Service) Apply(name string, req ApplyReq) (ApplyResult, error) {
	plan, err := s.PlanApply(name, req)
	if err != nil {
		return ApplyResult{}, err
	}

	config, err := s.store.Load(s.configFile, s.baseDir)
	if err != nil {
		return ApplyResult{}, err
	}

	toInstall := make([]skill.Skill, 0)
	for _, item := range plan.Items {
		if item.Action == "error" {
			return ApplyResult{Plan: plan}, fmt.Errorf("profile apply plan has errors")
		}
		if item.Action == "install" {
			toInstall = append(toInstall, item.Skill)
		}
	}
	if len(toInstall) == 0 {
		return ApplyResult{Plan: plan}, nil
	}

	item, err := s.Show(name)
	if err != nil {
		return ApplyResult{}, err
	}
	workDir := firstNonEmpty(req.WorkDir, s.baseDir)
	installer := installapp.NewService(config.LockFile)
	if installMode := strings.TrimSpace(item.InstallMode); installMode != "" {
		if !agentfs.IsValidMode(installMode) {
			return ApplyResult{Plan: plan}, fmt.Errorf("invalid profile install_mode: %s", installMode)
		}
		installer = installer.WithInstallMode(agentfs.NormalizeMode(installMode))
	}
	result, err := installer.RunResolved(config, installapp.InstallReq{
		Agent:   plan.Agent,
		Scope:   plan.Scope,
		WorkDir: workDir,
		Profile: name,
	}, toInstall, nil)
	if err != nil {
		return ApplyResult{}, err
	}
	applyResult := ApplyResult{Plan: plan, Installed: result.Installed, InstallFailed: result.InstallFailed}
	if len(result.InstallFailed) > 0 {
		return applyResult, fmt.Errorf("profile apply failed: install failed")
	}
	return applyResult, nil
}

func (s *Service) Save(name string, item profile.Profile) error {
	if err := profile.ValidateName(name); err != nil {
		return err
	}

	config, err := s.store.Load(s.configFile, s.baseDir)
	if err != nil {
		return err
	}
	return s.saveLoaded(config, name, item)
}

func (s *Service) saveLoaded(config cfg.Config, name string, item profile.Profile) error {
	targets, err := profile.NormalizeTargets(item.Targets)
	if err != nil {
		return err
	}
	item.Targets = targets

	if config.Profiles == nil {
		config.Profiles = map[string]profile.Profile{}
	}
	config.Profiles[name] = item
	return s.store.Save(s.configFile, config, s.baseDir)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func findTargetSkill(items []skill.Skill, target profile.Target) (skill.Skill, string, bool) {
	candidates := make([]skill.Skill, 0, len(items))
	for _, item := range items {
		if target.Source != "" && item.SourceID != target.Source && item.SourceName != target.Source {
			continue
		}
		candidates = append(candidates, item)
	}

	exact := make([]skill.Skill, 0)
	for _, item := range candidates {
		if item.SourceQualifiedName == target.Skill || item.QualifiedName == target.Skill {
			exact = append(exact, item)
		}
	}
	if len(exact) > 1 {
		return skill.Skill{}, fmt.Sprintf("ambiguous skill target: %s", target.Skill), false
	}
	if len(exact) == 1 {
		return exact[0], "", true
	}
	if strings.Contains(target.Skill, "/") {
		return skill.Skill{}, "skill not found in index", false
	}

	exactID := make([]skill.Skill, 0)
	for _, item := range candidates {
		if item.ID == target.Skill {
			exactID = append(exactID, item)
		}
	}
	if len(exactID) > 1 {
		return skill.Skill{}, fmt.Sprintf("ambiguous skill target: %s", target.Skill), false
	}
	if len(exactID) == 1 {
		return exactID[0], "", true
	}

	tailMatches := make([]skill.Skill, 0)
	for _, item := range candidates {
		if idx := strings.LastIndex(item.QualifiedName, "/"); idx >= 0 && idx < len(item.QualifiedName)-1 && item.QualifiedName[idx+1:] == target.Skill {
			tailMatches = append(tailMatches, item)
		}
	}
	if len(tailMatches) > 1 {
		return skill.Skill{}, fmt.Sprintf("ambiguous skill target: %s", target.Skill), false
	}
	if len(tailMatches) == 1 {
		return tailMatches[0], "", true
	}
	return skill.Skill{}, "skill not found in index", false
}

func isTargetInstalled(installedSet map[string]struct{}, target profile.Target, found skill.Skill) bool {
	if target.Source != "" {
		if _, ok := installedSet[target.Source+"\x00"+target.Skill]; ok {
			return true
		}
	}
	if target.Source == "" {
		if _, ok := installedSet["\x00"+target.Skill]; ok {
			return true
		}
	}
	if _, ok := installedSet[found.SourceID+"\x00"+found.ID]; ok {
		return true
	}
	return false
}
