package projectupdateapp

import (
	"fmt"
	"os"
	"strings"

	"github.com/inhere/skillc/internal/app/configapp"
	"github.com/inhere/skillc/internal/app/installapp"
	"github.com/inhere/skillc/internal/app/statusapp"
	"github.com/inhere/skillc/internal/app/updateapp"
	"github.com/inhere/skillc/internal/domain/project"
)

type Req struct {
	Agent      string
	Scope      string
	Target     string
	ProjectIDs []string
	Sync       bool
	Confirm    bool
}

type Plan struct {
	Agent          string        `json:"agent"`
	Scope          string        `json:"scope"`
	Target         string        `json:"target,omitempty"`
	Projects       []ProjectPlan `json:"projects"`
	CandidateCount int           `json:"candidate_count"`
}

type ProjectPlan struct {
	ProjectID string            `json:"project_id"`
	Name      string            `json:"name,omitempty"`
	Path      string            `json:"path"`
	Items     []statusapp.Item  `json:"items"`
	Summary   statusapp.Summary `json:"summary"`
	Error     string            `json:"error,omitempty"`
}

type Result struct {
	Plan    Plan            `json:"plan"`
	Results []ProjectResult `json:"results"`
}

type ProjectResult struct {
	ProjectID     string                      `json:"project_id"`
	Path          string                      `json:"path"`
	Updated       []installapp.RuntimeRecord  `json:"updated,omitempty"`
	Skipped       []updateapp.SkippedItem     `json:"skipped,omitempty"`
	Failed        []updateapp.FailedItem      `json:"failed,omitempty"`
	SyncFailed    []updateapp.SourceSyncError `json:"sync_failed,omitempty"`
	CleanupFailed []updateapp.FailedItem      `json:"cleanup_failed,omitempty"`
	Error         string                      `json:"error,omitempty"`
}

type Service struct {
	configFile    string
	baseDir       string
	configService *configapp.Service
}

func NewService(configFile string, baseDir string) *Service {
	return &Service{
		configFile:    configFile,
		baseDir:       baseDir,
		configService: configapp.NewService(configFile, baseDir),
	}
}

func (s *Service) Plan(req Req) (Plan, error) {
	config, err := s.configService.Show()
	if err != nil {
		return Plan{}, err
	}
	projects, err := selectProjects(config.Projects, req.ProjectIDs)
	if err != nil {
		return Plan{}, err
	}
	agentName := defaultString(req.Agent, "universal")
	scope := defaultString(req.Scope, "project")
	plan := Plan{Agent: agentName, Scope: scope, Target: req.Target}
	for _, item := range projects {
		projectPlan := ProjectPlan{ProjectID: item.ID, Name: item.Name, Path: item.Path}
		if _, err := os.Stat(item.Path); err != nil {
			projectPlan.Error = err.Error()
			plan.Projects = append(plan.Projects, projectPlan)
			continue
		}
		statusResult, err := statusapp.NewService(s.configFile, item.Path).Run(statusapp.Req{
			Agent:   agentName,
			Scope:   scope,
			WorkDir: item.Path,
			Sync:    req.Sync,
		})
		if err != nil {
			projectPlan.Error = err.Error()
			plan.Projects = append(plan.Projects, projectPlan)
			continue
		}
		projectPlan.Summary = statusResult.Summary
		for _, statusItem := range statusResult.Items {
			if req.Target != "" && !matchesStatusTarget(statusItem, req.Target) {
				continue
			}
			if statusItem.Status != statusapp.StatusOutdated && statusItem.Status != statusapp.StatusMissing {
				continue
			}
			projectPlan.Items = append(projectPlan.Items, statusItem)
		}
		plan.CandidateCount += len(projectPlan.Items)
		plan.Projects = append(plan.Projects, projectPlan)
	}
	return plan, nil
}

func (s *Service) Run(req Req) (Result, error) {
	if !req.Confirm {
		return Result{}, fmt.Errorf("confirmation required")
	}
	plan, err := s.Plan(req)
	if err != nil {
		return Result{}, err
	}
	result := Result{Plan: plan}
	for _, projectPlan := range plan.Projects {
		projectResult := ProjectResult{ProjectID: projectPlan.ProjectID, Path: projectPlan.Path}
		if projectPlan.Error != "" {
			projectResult.Error = projectPlan.Error
			result.Results = append(result.Results, projectResult)
			continue
		}
		for _, item := range projectPlan.Items {
			updateResult, err := updateapp.NewService(s.configFile, projectPlan.Path).Run(updateapp.Req{
				Target:       updateTarget(item),
				Agent:        plan.Agent,
				Scope:        plan.Scope,
				WorkDir:      projectPlan.Path,
				ProjectPaths: []string{projectPlan.Path},
			})
			projectResult.Updated = append(projectResult.Updated, updateResult.Updated...)
			projectResult.Skipped = append(projectResult.Skipped, updateResult.Skipped...)
			projectResult.Failed = append(projectResult.Failed, updateResult.Failed...)
			projectResult.SyncFailed = append(projectResult.SyncFailed, updateResult.SyncFailed...)
			projectResult.CleanupFailed = append(projectResult.CleanupFailed, updateResult.CleanupFailed...)
			if err != nil && projectResult.Error == "" {
				projectResult.Error = err.Error()
			}
		}
		result.Results = append(result.Results, projectResult)
	}
	return result, nil
}

func selectProjects(projects []project.Project, ids []string) ([]project.Project, error) {
	if len(ids) == 0 {
		return append([]project.Project(nil), projects...), nil
	}
	byID := make(map[string]project.Project, len(projects))
	for _, item := range projects {
		byID[item.ID] = item
	}
	out := make([]project.Project, 0, len(ids))
	for _, id := range ids {
		item, ok := byID[id]
		if !ok {
			return nil, fmt.Errorf("project not found: %s", id)
		}
		out = append(out, item)
	}
	return out, nil
}

func matchesStatusTarget(item statusapp.Item, target string) bool {
	return item.SkillID == target || item.QualifiedName == target || item.SourceQualifiedName == target
}

func updateTarget(item statusapp.Item) string {
	return item.SkillID
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
