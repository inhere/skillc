package webapp

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/inhere/skillc/internal/app/configapp"
	"github.com/inhere/skillc/internal/app/profileapp"
	"github.com/inhere/skillc/internal/app/projectapp"
	"github.com/inhere/skillc/internal/app/projectupdateapp"
	"github.com/inhere/skillc/internal/app/registryapp"
	"github.com/inhere/skillc/internal/app/searchapp"
	"github.com/inhere/skillc/internal/app/sourceapp"
	"github.com/inhere/skillc/internal/app/statusapp"
	cfg "github.com/inhere/skillc/internal/domain/config"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/domain/profile"
	projectpkg "github.com/inhere/skillc/internal/domain/project"
	"github.com/inhere/skillc/internal/domain/registry"
	"github.com/inhere/skillc/internal/domain/skill"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/lockstore"
	"github.com/inhere/skillc/internal/infra/repoindex"
)

type ManagerReq struct {
	Agent   string
	Scope   string
	WorkDir string
}

type Summary struct {
	ProjectPath  string        `json:"project_path"`
	SourceCount  int           `json:"source_count"`
	ProfileCount int           `json:"profile_count"`
	SkillCount   int           `json:"skill_count"`
	Status       StatusSummary `json:"status"`
}

type StatusSummary struct {
	Installed   int `json:"installed"`
	Missing     int `json:"missing"`
	Outdated    int `json:"outdated"`
	Orphan      int `json:"orphan"`
	Unmanaged   int `json:"unmanaged"`
	SourceError int `json:"source_error"`
}

type Manager struct {
	configFile string
	baseDir    string
}

func NewManager(configFile string, baseDir string) *Manager {
	return &Manager{configFile: configFile, baseDir: baseDir}
}

func (m *Manager) Summary(req ManagerReq) (Summary, error) {
	config, err := m.config()
	if err != nil {
		return Summary{}, err
	}
	indexItems, err := loadIndex(config.IndexFile)
	if err != nil {
		return Summary{}, err
	}
	statusResult, err := m.Status(req)
	if err != nil {
		return Summary{}, err
	}

	projectPath := req.WorkDir
	if projectPath == "" {
		projectPath = m.baseDir
	}
	return Summary{
		ProjectPath:  projectPath,
		SourceCount:  len(config.Sources),
		ProfileCount: len(config.Profiles),
		SkillCount:   len(indexItems),
		Status:       toStatusSummary(statusResult.Summary),
	}, nil
}

func (m *Manager) Sources() ([]sourcepkg.Source, error) {
	return sourceapp.NewService(m.configFile, m.baseDir).List()
}

func (m *Manager) Collections(sourceID string) ([]repoindex.SourceCollectionSummary, error) {
	config, err := m.config()
	if err != nil {
		return nil, err
	}
	return searchapp.NewService(config.IndexFile).ListSourceCollections(sourceID)
}

func (m *Manager) Skills(keyword string) ([]skill.Skill, error) {
	config, err := m.config()
	if err != nil {
		return nil, err
	}
	return searchapp.NewService(config.IndexFile).Search(keyword, "", "")
}

func (m *Manager) Profiles() ([]profile.NamedProfile, error) {
	return profileapp.NewService(m.configFile, m.baseDir).List()
}

func (m *Manager) Projects() ([]projectpkg.Project, error) {
	return projectapp.NewService(m.configFile, m.baseDir).List()
}

func (m *Manager) Registries() ([]registry.Registry, error) {
	return registryapp.NewService(m.configFile, m.baseDir).List()
}

func (m *Manager) RegistrySkills(keyword string, registryID string) ([]registry.SkillEntry, error) {
	return registryapp.NewService(m.configFile, m.baseDir).SearchSkills(registryapp.SearchReq{
		Keyword:    keyword,
		RegistryID: registryID,
	})
}

func (m *Manager) RegistrySources(keyword string, registryID string) ([]registry.Entry, error) {
	items, err := registryapp.NewService(m.configFile, m.baseDir).Search(keyword)
	if err != nil {
		return nil, err
	}
	return filterRegistrySourcesByID(items, registryID), nil
}

func (m *Manager) Status(req ManagerReq) (statusapp.Result, error) {
	return statusapp.NewService(m.configFile, m.baseDir).Run(statusapp.Req{
		Agent:   req.Agent,
		Scope:   req.Scope,
		WorkDir: req.WorkDir,
	})
}

func (m *Manager) InstallMap() ([]ProjectInstall, error) {
	config, err := m.config()
	if err != nil {
		return nil, err
	}
	records, err := loadLock(config.LockFile)
	if err != nil {
		return nil, err
	}
	return BuildProjectInstallIndex(records), nil
}

func (m *Manager) VersionDrift() ([]VersionDriftGroup, error) {
	config, err := m.config()
	if err != nil {
		return nil, err
	}
	records, err := loadLock(config.LockFile)
	if err != nil {
		return nil, err
	}
	indexItems, err := loadIndex(config.IndexFile)
	if err != nil {
		return nil, err
	}
	return BuildVersionDrift(BuildProjectInstallIndex(records), indexItems), nil
}

func (m *Manager) PlanProfileApply(name string, req ManagerReq) (profile.ApplyPlan, error) {
	return profileapp.NewService(m.configFile, m.baseDir).PlanApply(name, profileapp.ApplyReq{
		Agent:   req.Agent,
		Scope:   req.Scope,
		WorkDir: req.WorkDir,
	})
}

func (m *Manager) PlanAllProjectsUpdate(req WebUpdateAllReq) (projectupdateapp.Plan, error) {
	return projectupdateapp.NewService(m.configFile, m.baseDir).Plan(projectupdateapp.Req{
		Agent:      req.Agent,
		Scope:      req.Scope,
		Target:     req.Target,
		ProjectIDs: req.ProjectIDs,
		Sync:       true,
	})
}

func (m *Manager) RunAllProjectsUpdate(req WebUpdateAllReq) (updateAllProjectsActionResult, error) {
	result, err := projectupdateapp.NewService(m.configFile, m.baseDir).Run(projectupdateapp.Req{
		Agent:      req.Agent,
		Scope:      req.Scope,
		Target:     req.Target,
		ProjectIDs: req.ProjectIDs,
		Sync:       true,
		Confirm:    true,
	})
	out := toUpdateAllProjectsActionResult(result)
	if err != nil {
		if out.Plan.Agent != "" || len(out.Results) > 0 {
			out.Error = err.Error()
			return out, nil
		}
		return out, err
	}
	return out, nil
}

func (m *Manager) History(limit int) ([]HistoryRecord, error) {
	return newHistoryStore(m.historyFile()).List(limit)
}

func (m *Manager) historyFile() string {
	config, err := m.config()
	if err == nil && config.RegistryCacheDir != "" {
		return filepath.Join(filepath.Dir(config.RegistryCacheDir), "skillc-web-history.jsonl")
	}
	return filepath.Join(filepath.Dir(m.configFile), "skillc-web-history.jsonl")
}

func (m *Manager) config() (cfg.Config, error) {
	return configapp.NewService(m.configFile, m.baseDir).Show()
}

func loadLock(path string) (lockpkg.File, error) {
	records, err := lockstore.NewStore().Load(path)
	if err == nil {
		return records, nil
	}
	if os.IsNotExist(err) {
		return lockpkg.File{}, nil
	}
	return nil, err
}

func loadIndex(path string) ([]skill.Skill, error) {
	items, err := repoindex.NewStore().Load(path)
	if err == nil {
		return items, nil
	}
	if os.IsNotExist(err) {
		return []skill.Skill{}, nil
	}
	return nil, err
}

func filterRegistrySourcesByID(items []registry.Entry, registryID string) []registry.Entry {
	registryID = strings.ToLower(strings.TrimSpace(registryID))
	if registryID == "" {
		return items
	}
	out := make([]registry.Entry, 0, len(items))
	for _, item := range items {
		if strings.ToLower(item.RegistryID) == registryID {
			out = append(out, item)
		}
	}
	return out
}

func toStatusSummary(summary statusapp.Summary) StatusSummary {
	return StatusSummary{
		Installed:   summary.Installed,
		Missing:     summary.Missing,
		Outdated:    summary.Outdated,
		Orphan:      summary.Orphan,
		Unmanaged:   summary.Unmanaged,
		SourceError: summary.SourceError,
	}
}
