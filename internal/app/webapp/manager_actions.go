package webapp

import (
	"github.com/inhere/skillc/internal/app/installapp"
	"github.com/inhere/skillc/internal/app/profileapp"
	"github.com/inhere/skillc/internal/app/updateapp"
	"github.com/inhere/skillc/internal/domain/profile"
)

type WebUpdateReq struct {
	ManagerReq
	Target string
}

type actionRuntimeRecord struct {
	SkillID       string `json:"skill_id"`
	SourceID      string `json:"source_id,omitempty"`
	Version       string `json:"version,omitempty"`
	Agent         string `json:"agent,omitempty"`
	Scope         string `json:"scope,omitempty"`
	InstalledPath string `json:"installed_path,omitempty"`
}

type actionErrorItem struct {
	SkillID string `json:"skill_id"`
	Reason  string `json:"reason"`
}

type actionSourceErrorItem struct {
	SourceID string `json:"source_id"`
	Reason   string `json:"reason"`
}

type profileApplyActionResult struct {
	Error         string                `json:"error,omitempty"`
	Plan          profile.ApplyPlan     `json:"plan"`
	Installed     []actionRuntimeRecord `json:"installed"`
	InstallFailed []actionErrorItem     `json:"install_failed,omitempty"`
}

type updateRunActionResult struct {
	Error         string                  `json:"error,omitempty"`
	Updated       []actionRuntimeRecord   `json:"updated"`
	Skipped       []actionErrorItem       `json:"skipped,omitempty"`
	Failed        []actionErrorItem       `json:"failed,omitempty"`
	SyncFailed    []actionSourceErrorItem `json:"sync_failed,omitempty"`
	CleanupFailed []actionErrorItem       `json:"cleanup_failed,omitempty"`
}

func (m *Manager) ApplyProfile(name string, req ManagerReq) (profileApplyActionResult, error) {
	result, err := profileapp.NewService(m.configFile, m.baseDir).Apply(name, profileapp.ApplyReq{
		Agent:   req.Agent,
		Scope:   req.Scope,
		WorkDir: req.WorkDir,
	})
	out := profileApplyActionResult{
		Plan:          result.Plan,
		Installed:     runtimeRecords(result.Installed),
		InstallFailed: installErrors(result.InstallFailed),
	}
	if err != nil {
		if hasProfileApplyActionPayload(out) {
			out.Error = err.Error()
			return out, nil
		}
		return out, err
	}
	return out, nil
}

func (m *Manager) RunUpdate(req WebUpdateReq) (updateRunActionResult, error) {
	result, err := updateapp.NewService(m.configFile, m.baseDir).Run(updateapp.Req{
		Target:  req.Target,
		Agent:   req.Agent,
		Scope:   req.Scope,
		WorkDir: req.WorkDir,
	})
	out := updateRunActionResult{
		Updated:       runtimeRecords(result.Updated),
		Skipped:       skippedErrors(result.Skipped),
		Failed:        failedErrors(result.Failed),
		SyncFailed:    sourceSyncErrors(result.SyncFailed),
		CleanupFailed: failedErrors(result.CleanupFailed),
	}
	if err != nil {
		if hasUpdateRunActionPayload(out) {
			out.Error = err.Error()
			return out, nil
		}
		return out, err
	}
	return out, nil
}

func hasProfileApplyActionPayload(result profileApplyActionResult) bool {
	return result.Plan.Profile != "" || len(result.Installed) > 0 || len(result.InstallFailed) > 0
}

func hasUpdateRunActionPayload(result updateRunActionResult) bool {
	return len(result.Updated) > 0 ||
		len(result.Skipped) > 0 ||
		len(result.Failed) > 0 ||
		len(result.SyncFailed) > 0 ||
		len(result.CleanupFailed) > 0
}

func runtimeRecords(records []installapp.RuntimeRecord) []actionRuntimeRecord {
	out := make([]actionRuntimeRecord, 0, len(records))
	for _, record := range records {
		out = append(out, actionRuntimeRecord{
			SkillID:       record.SkillID,
			SourceID:      record.SourceID,
			Version:       record.Version,
			Agent:         record.Agent,
			Scope:         record.Scope,
			InstalledPath: record.InstalledPath,
		})
	}
	return out
}

func installErrors(items []installapp.InstallItemError) []actionErrorItem {
	out := make([]actionErrorItem, 0, len(items))
	for _, item := range items {
		out = append(out, actionErrorItem{SkillID: item.SkillID, Reason: item.Reason})
	}
	return out
}

func skippedErrors(items []updateapp.SkippedItem) []actionErrorItem {
	out := make([]actionErrorItem, 0, len(items))
	for _, item := range items {
		out = append(out, actionErrorItem{SkillID: item.SkillID, Reason: item.Reason})
	}
	return out
}

func failedErrors(items []updateapp.FailedItem) []actionErrorItem {
	out := make([]actionErrorItem, 0, len(items))
	for _, item := range items {
		out = append(out, actionErrorItem{SkillID: item.SkillID, Reason: item.Reason})
	}
	return out
}

func sourceSyncErrors(items []updateapp.SourceSyncError) []actionSourceErrorItem {
	out := make([]actionSourceErrorItem, 0, len(items))
	for _, item := range items {
		out = append(out, actionSourceErrorItem{SourceID: item.SourceID, Reason: item.Reason})
	}
	return out
}
