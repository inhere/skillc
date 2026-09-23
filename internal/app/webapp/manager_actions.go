package webapp

import (
	"github.com/inhere/skillc/internal/app/installapp"
	"github.com/inhere/skillc/internal/app/profileapp"
	"github.com/inhere/skillc/internal/app/projectupdateapp"
	"github.com/inhere/skillc/internal/app/updateapp"
	"github.com/inhere/skillc/internal/domain/profile"
)

type WebUpdateReq struct {
	ManagerReq
	Target string
	// Force 覆盖安装目录里的本地改动（覆盖前仍会备份）。
	Force bool
	// Merge 显式要求按文件三方合并（默认行为；与 Force 组合时冲突取上游）。
	Merge bool
	// NoMerge 关闭按文件合并，回到「跳过本地改动 + Force 整体覆盖」。
	NoMerge bool
}

type WebUpdateAllReq struct {
	ManagerReq
	Target     string
	ProjectIDs []string
	// Force 覆盖安装目录里的本地改动（覆盖前仍会备份）。
	Force bool
	// Merge 显式要求按文件三方合并（默认行为；与 Force 组合时冲突取上游）。
	Merge bool
	// NoMerge 关闭按文件合并。
	NoMerge bool
}

type updateAllProjectsReq struct {
	Confirm    bool     `json:"confirm,omitempty"`
	Target     string   `json:"target,omitempty"`
	ProjectIDs []string `json:"project_ids,omitempty"`
	Force      bool     `json:"force,omitempty"`
	Merge      bool     `json:"merge,omitempty"`
	NoMerge    bool     `json:"no_merge,omitempty"`
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

type actionMergeItem struct {
	SkillID    string   `json:"skill_id"`
	Path       string   `json:"path,omitempty"`
	Updated    []string `json:"updated,omitempty"`
	KeptLocal  []string `json:"kept_local,omitempty"`
	Conflicts  []string `json:"conflicts,omitempty"`
	Removed    []string `json:"removed,omitempty"`
	BackupPath string   `json:"backup_path,omitempty"`
}

type actionBackupItem struct {
	SkillID string `json:"skill_id"`
	Path    string `json:"path"`
}

type updateRunActionResult struct {
	Error         string                  `json:"error,omitempty"`
	Updated       []actionRuntimeRecord   `json:"updated"`
	BackedUp      []actionBackupItem      `json:"backed_up,omitempty"`
	Merged        []actionMergeItem       `json:"merged,omitempty"`
	Skipped       []actionErrorItem       `json:"skipped,omitempty"`
	Failed        []actionErrorItem       `json:"failed,omitempty"`
	SyncFailed    []actionSourceErrorItem `json:"sync_failed,omitempty"`
	CleanupFailed []actionErrorItem       `json:"cleanup_failed,omitempty"`
}

type updateAllProjectsActionResult struct {
	Error   string                `json:"error,omitempty"`
	Plan    projectupdateapp.Plan `json:"plan"`
	Results []projectUpdateResult `json:"results,omitempty"`
}

type projectUpdateResult struct {
	ProjectID     string                  `json:"project_id"`
	Path          string                  `json:"path"`
	Updated       []actionRuntimeRecord   `json:"updated,omitempty"`
	BackedUp      []actionBackupItem      `json:"backed_up,omitempty"`
	Merged        []actionMergeItem       `json:"merged,omitempty"`
	Skipped       []actionErrorItem       `json:"skipped,omitempty"`
	Failed        []actionErrorItem       `json:"failed,omitempty"`
	SyncFailed    []actionSourceErrorItem `json:"sync_failed,omitempty"`
	CleanupFailed []actionErrorItem       `json:"cleanup_failed,omitempty"`
	Error         string                  `json:"error,omitempty"`
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
		Force:   req.Force,
		Merge:   req.Merge,
		NoMerge: req.NoMerge,
	})
	out := updateRunActionResult{
		Updated:       runtimeRecords(result.Updated),
		BackedUp:      backupItems(result.BackedUp),
		Merged:        mergeReports(result.Merged),
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

func toUpdateAllProjectsActionResult(result projectupdateapp.Result) updateAllProjectsActionResult {
	out := updateAllProjectsActionResult{Plan: result.Plan}
	for _, item := range result.Results {
		converted := projectUpdateResult{ProjectID: item.ProjectID, Path: item.Path, Error: item.Error}
		converted.Updated = append(converted.Updated, runtimeRecords(item.Updated)...)
		converted.BackedUp = append(converted.BackedUp, backupItems(item.BackedUp)...)
		converted.Merged = append(converted.Merged, mergeReports(item.Merged)...)
		converted.Skipped = append(converted.Skipped, skippedErrors(item.Skipped)...)
		converted.Failed = append(converted.Failed, failedErrors(item.Failed)...)
		converted.SyncFailed = append(converted.SyncFailed, sourceSyncErrors(item.SyncFailed)...)
		converted.CleanupFailed = append(converted.CleanupFailed, failedErrors(item.CleanupFailed)...)
		out.Results = append(out.Results, converted)
	}
	return out
}

func backupItems(items []updateapp.BackupItem) []actionBackupItem {
	out := make([]actionBackupItem, 0, len(items))
	for _, item := range items {
		out = append(out, actionBackupItem{SkillID: item.SkillID, Path: item.Path})
	}
	return out
}

func mergeReports(items []updateapp.MergeReport) []actionMergeItem {
	out := make([]actionMergeItem, 0, len(items))
	for _, item := range items {
		out = append(out, actionMergeItem{
			SkillID:    item.SkillID,
			Path:       item.Path,
			Updated:    item.Updated,
			KeptLocal:  item.KeptLocal,
			Conflicts:  item.Conflicts,
			Removed:    item.Removed,
			BackupPath: item.BackupPath,
		})
	}
	return out
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
