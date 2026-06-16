package webapp

import (
	"fmt"
	"strings"

	"github.com/inhere/skillc/internal/app/installapp"
	"github.com/inhere/skillc/internal/app/registryapp"
	"github.com/inhere/skillc/internal/domain/registry"
	"github.com/inhere/skillc/internal/domain/skill"
)

type WebRegistryInstallReq struct {
	ManagerReq
	Confirm bool   `json:"confirm,omitempty"`
	Target  string `json:"target"`
}

type WebRegistrySyncReq struct {
	Confirm    bool   `json:"confirm,omitempty"`
	RegistryID string `json:"registry_id,omitempty"`
}

type WebRegistryAddSourceReq struct {
	Confirm bool   `json:"confirm,omitempty"`
	EntryID string `json:"entry_id"`
	ID      string `json:"id,omitempty"`
	Name    string `json:"name,omitempty"`
	Sync    bool   `json:"sync,omitempty"`
}

type registryInstallPlan struct {
	Target       string   `json:"target"`
	RegistryID   string   `json:"registry_id"`
	SkillID      string   `json:"skill_id"`
	Name         string   `json:"name,omitempty"`
	Version      string   `json:"version,omitempty"`
	Agent        string   `json:"agent"`
	Scope        string   `json:"scope"`
	InstallEntry string   `json:"install_entry"`
	SourceURL    string   `json:"source_url,omitempty"`
	DownloadURL  string   `json:"download_url,omitempty"`
	Checksum     string   `json:"checksum,omitempty"`
	Warnings     []string `json:"warnings,omitempty"`
}

type registryInstallActionResult struct {
	Error     string                `json:"error,omitempty"`
	Plan      registryInstallPlan   `json:"plan"`
	Installed []actionRuntimeRecord `json:"installed"`
	Failed    []actionErrorItem     `json:"failed,omitempty"`
}

type registrySyncPlan struct {
	RegistryID string   `json:"registry_id,omitempty"`
	Items      []string `json:"items"`
}

type registryAddSourcePlan struct {
	EntryID  string `json:"entry_id"`
	SourceID string `json:"source_id"`
	Name     string `json:"name,omitempty"`
	Type     string `json:"type"`
	Location string `json:"location"`
	Sync     bool   `json:"sync"`
}

func (m *Manager) PlanRegistryInstall(req WebRegistryInstallReq) (registryInstallPlan, error) {
	item, err := registryapp.NewService(m.configFile, m.baseDir).InfoSkill(req.Target)
	if err != nil {
		return registryInstallPlan{}, err
	}
	plan := registryInstallPlan{
		Target:       req.Target,
		RegistryID:   item.RegistryID,
		SkillID:      item.ID,
		Name:         item.Name,
		Version:      item.Version,
		Agent:        req.Agent,
		Scope:        req.Scope,
		InstallEntry: item.InstallEntry,
		SourceURL:    item.SourceURL,
		DownloadURL:  item.DownloadURL,
		Checksum:     item.Checksum,
	}
	if item.DownloadURL != "" && item.Checksum == "" {
		plan.Warnings = append(plan.Warnings, registryapp.ArchiveChecksumMissingWarning)
	}
	return plan, nil
}

func (m *Manager) RunRegistryInstall(req WebRegistryInstallReq) (registryInstallActionResult, error) {
	plan, err := m.PlanRegistryInstall(req)
	if err != nil {
		return registryInstallActionResult{}, err
	}
	config, err := m.config()
	if err != nil {
		return registryInstallActionResult{Plan: plan}, err
	}
	item, err := registryapp.NewService(m.configFile, m.baseDir).MaterializeSkill(req.Target)
	if err != nil {
		return registryInstallActionResult{Plan: plan, Error: err.Error()}, nil
	}
	result, err := installapp.NewService(config.LockFile).RunResolved(config, installapp.InstallReq{
		SkillID: item.SourceQualifiedName,
		Agent:   req.Agent,
		Scope:   req.Scope,
		WorkDir: req.WorkDir,
	}, []skill.Skill{item}, nil)
	out := registryInstallActionResult{
		Plan:      plan,
		Installed: runtimeRecords(result.Installed),
		Failed:    installErrors(result.InstallFailed),
	}
	if err != nil {
		out.Error = err.Error()
		return out, nil
	}
	return out, nil
}

func (m *Manager) PlanRegistrySync(req WebRegistrySyncReq) (registrySyncPlan, error) {
	items, err := registryapp.NewService(m.configFile, m.baseDir).List()
	if err != nil {
		return registrySyncPlan{}, err
	}
	plan := registrySyncPlan{RegistryID: strings.TrimSpace(req.RegistryID)}
	for _, item := range items {
		if plan.RegistryID == "" || item.ID == plan.RegistryID {
			plan.Items = append(plan.Items, item.ID)
		}
	}
	if plan.RegistryID != "" && len(plan.Items) == 0 {
		return registrySyncPlan{}, fmt.Errorf("registry not found: %s", plan.RegistryID)
	}
	return plan, nil
}

func (m *Manager) RunRegistrySync(req WebRegistrySyncReq) (registrySyncPlan, error) {
	plan, err := m.PlanRegistrySync(req)
	if err != nil {
		return registrySyncPlan{}, err
	}
	service := registryapp.NewService(m.configFile, m.baseDir)
	if plan.RegistryID == "" {
		return plan, service.SyncAll()
	}
	return plan, service.Sync(plan.RegistryID)
}

func (m *Manager) PlanRegistryAddSource(req WebRegistryAddSourceReq) (registryAddSourcePlan, error) {
	item, err := registryapp.NewService(m.configFile, m.baseDir).Info(req.EntryID)
	if err != nil {
		return registryAddSourcePlan{}, err
	}
	return registryAddSourcePlan{
		EntryID:  req.EntryID,
		SourceID: firstNonEmptyWeb(req.ID, item.ID),
		Name:     firstNonEmptyWeb(req.Name, item.Name),
		Type:     item.Type,
		Location: registryEntryLocation(item),
		Sync:     req.Sync,
	}, nil
}

func (m *Manager) RunRegistryAddSource(req WebRegistryAddSourceReq) (registryAddSourcePlan, error) {
	plan, err := m.PlanRegistryAddSource(req)
	if err != nil {
		return registryAddSourcePlan{}, err
	}
	_, err = registryapp.NewService(m.configFile, m.baseDir).AddSource(registryapp.AddSourceReq{
		EntryID: req.EntryID,
		ID:      req.ID,
		Name:    req.Name,
		Sync:    req.Sync,
	})
	return plan, err
}

func registryEntryLocation(item registry.Entry) string {
	if item.URL != "" {
		return item.URL
	}
	return item.Path
}

func firstNonEmptyWeb(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
