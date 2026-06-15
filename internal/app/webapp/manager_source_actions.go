package webapp

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/inhere/skillc/internal/app/sourceapp"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
)

type sourceActionReq struct {
	Confirm bool   `json:"confirm,omitempty"`
	ID      string `json:"id,omitempty"`
	Value   string `json:"value,omitempty"`
	Ref     string `json:"ref,omitempty"`
	Sync    bool   `json:"sync,omitempty"`
}

type sourceActionPlan struct {
	Action   string             `json:"action"`
	SourceID string             `json:"source_id,omitempty"`
	Source   sourcepkg.Source   `json:"source,omitempty"`
	Existing bool               `json:"existing,omitempty"`
	Impact   sourceRemoveImpact `json:"impact,omitempty"`
	Items    []sourcePlanItem   `json:"items"`
	Warnings []string           `json:"warnings,omitempty"`
}

type sourcePlanItem struct {
	Action string `json:"action"`
	Target string `json:"target"`
	Reason string `json:"reason,omitempty"`
}

type sourceRemoveImpact struct {
	InstalledCount     int `json:"installed_count"`
	ProfileTargetCount int `json:"profile_target_count"`
	IndexedSkillCount  int `json:"indexed_skill_count"`
	CollectionCount    int `json:"collection_count"`
}

type sourceActionResult struct {
	Error   string           `json:"error,omitempty"`
	Plan    sourceActionPlan `json:"plan"`
	Added   bool             `json:"added,omitempty"`
	Synced  bool             `json:"synced,omitempty"`
	Removed bool             `json:"removed,omitempty"`
}

func (m *Manager) PlanSourceAdd(req sourceActionReq) (sourceActionPlan, error) {
	value := strings.TrimSpace(req.Value)
	if value == "" {
		return sourceActionPlan{}, fmt.Errorf("source value is required")
	}
	config, err := m.config()
	if err != nil {
		return sourceActionPlan{}, err
	}
	if sourceapp.IsGitURL(value) {
		for _, src := range config.Sources {
			if src.Type == sourcepkg.TypeGit && src.URL == value {
				return sourceActionPlan{
					Action:   "exists",
					SourceID: src.ID,
					Source:   src,
					Existing: true,
					Items:    []sourcePlanItem{{Action: "skip", Target: src.ID, Reason: "source already exists"}},
				}, nil
			}
		}
		src, err := sourcepkg.NewGitSource(value, req.Ref)
		if err != nil {
			return sourceActionPlan{}, err
		}
		return sourceAddPlan("add_git", src, req.Sync), nil
	}
	src, err := sourcepkg.NewLocalSource(value)
	if err != nil {
		return sourceActionPlan{}, err
	}
	for _, current := range config.Sources {
		if current.Type == sourcepkg.TypeLocal && filepath.Clean(current.Path) == filepath.Clean(src.Path) {
			return sourceActionPlan{
				Action:   "exists",
				SourceID: current.ID,
				Source:   current,
				Existing: true,
				Items:    []sourcePlanItem{{Action: "skip", Target: current.ID, Reason: "source already exists"}},
			}, nil
		}
	}
	return sourceAddPlan("add_local", src, req.Sync), nil
}

func sourceAddPlan(action string, src sourcepkg.Source, syncNow bool) sourceActionPlan {
	items := []sourcePlanItem{{Action: "add", Target: src.ID, Reason: "source is not configured"}}
	if syncNow {
		items = append(items, sourcePlanItem{Action: "sync", Target: src.ID, Reason: "sync requested"})
	}
	return sourceActionPlan{Action: action, SourceID: src.ID, Source: src, Items: items}
}

func (m *Manager) PlanSourceSync(req sourceActionReq) (sourceActionPlan, error) {
	id := strings.TrimSpace(req.ID)
	if id == "" {
		return sourceActionPlan{}, fmt.Errorf("source id is required")
	}
	if id == "__all__" || id == "all" {
		return sourceActionPlan{
			Action: "sync_all",
			Items:  []sourcePlanItem{{Action: "sync", Target: "all", Reason: "sync all configured sources"}},
		}, nil
	}
	src, err := m.findSource(id)
	if err != nil {
		return sourceActionPlan{}, err
	}
	return sourceActionPlan{
		Action:   "sync",
		SourceID: src.ID,
		Source:   src,
		Items:    []sourcePlanItem{{Action: "sync", Target: src.ID}},
	}, nil
}

func (m *Manager) PlanSourceRemove(req sourceActionReq) (sourceActionPlan, error) {
	id := strings.TrimSpace(req.ID)
	if id == "" {
		return sourceActionPlan{}, fmt.Errorf("source id is required")
	}
	src, err := m.findSource(id)
	if err != nil {
		return sourceActionPlan{}, err
	}
	impact, warnings, err := m.sourceRemoveImpact(src.ID)
	if err != nil {
		return sourceActionPlan{}, err
	}
	return sourceActionPlan{
		Action:   "remove",
		SourceID: src.ID,
		Source:   src,
		Impact:   impact,
		Warnings: warnings,
		Items:    []sourcePlanItem{{Action: "remove", Target: src.ID, Reason: "source config and indexed skills will be removed"}},
	}, nil
}

func (m *Manager) RunSourceAdd(req sourceActionReq) (sourceActionResult, error) {
	plan, err := m.PlanSourceAdd(req)
	if err != nil {
		return sourceActionResult{}, err
	}
	if plan.Existing {
		return sourceActionResult{Plan: plan}, nil
	}
	src, _, err := sourceapp.NewService(m.configFile, m.baseDir).EnsureSource(req.Value, req.Ref)
	if err != nil {
		return sourceActionResult{Plan: plan, Error: err.Error()}, nil
	}
	result := sourceActionResult{Plan: plan, Added: true}
	if req.Sync {
		if err := sourceapp.NewService(m.configFile, m.baseDir).Sync(src.ID); err != nil {
			result.Error = err.Error()
			return result, nil
		}
		result.Synced = true
	}
	return result, nil
}

func (m *Manager) RunSourceSync(req sourceActionReq) (sourceActionResult, error) {
	plan, err := m.PlanSourceSync(req)
	if err != nil {
		return sourceActionResult{}, err
	}
	svc := sourceapp.NewService(m.configFile, m.baseDir)
	if plan.Action == "sync_all" {
		err = svc.SyncAll()
	} else {
		err = svc.Sync(plan.SourceID)
	}
	result := sourceActionResult{Plan: plan}
	if err != nil {
		result.Error = err.Error()
		return result, nil
	}
	result.Synced = true
	return result, nil
}

func (m *Manager) RunSourceRemove(req sourceActionReq) (sourceActionResult, error) {
	plan, err := m.PlanSourceRemove(req)
	if err != nil {
		return sourceActionResult{}, err
	}
	result := sourceActionResult{Plan: plan}
	if err := sourceapp.NewService(m.configFile, m.baseDir).Remove(plan.SourceID); err != nil {
		result.Error = err.Error()
		return result, nil
	}
	result.Removed = true
	return result, nil
}

func (m *Manager) findSource(id string) (sourcepkg.Source, error) {
	items, err := m.Sources()
	if err != nil {
		return sourcepkg.Source{}, err
	}
	for _, src := range items {
		if src.ID == id {
			return src, nil
		}
	}
	return sourcepkg.Source{}, fmt.Errorf("source not found: %s", id)
}

func (m *Manager) sourceRemoveImpact(sourceID string) (sourceRemoveImpact, []string, error) {
	config, err := m.config()
	if err != nil {
		return sourceRemoveImpact{}, nil, err
	}
	records, err := loadLock(config.LockFile)
	if err != nil {
		return sourceRemoveImpact{}, nil, err
	}
	indexItems, err := loadIndex(config.IndexFile)
	if err != nil {
		return sourceRemoveImpact{}, nil, err
	}
	impact := sourceRemoveImpact{}
	for _, recordsInScope := range records {
		for _, record := range recordsInScope {
			if record.SourceID == sourceID {
				impact.InstalledCount++
			}
		}
	}
	for _, item := range config.Profiles {
		for _, target := range item.Targets {
			if target.Source == sourceID {
				impact.ProfileTargetCount++
			}
		}
	}
	collections := map[string]struct{}{}
	for _, item := range indexItems {
		if item.SourceID != sourceID {
			continue
		}
		impact.IndexedSkillCount++
		if item.Collection != "" {
			collections[item.Collection] = struct{}{}
		}
	}
	impact.CollectionCount = len(collections)
	warnings := make([]string, 0, 2)
	if impact.InstalledCount > 0 {
		warnings = append(warnings, fmt.Sprintf("%d installed lock record(s) reference this source", impact.InstalledCount))
	}
	if impact.ProfileTargetCount > 0 {
		warnings = append(warnings, fmt.Sprintf("%d profile target(s) reference this source", impact.ProfileTargetCount))
	}
	return impact, warnings, nil
}
