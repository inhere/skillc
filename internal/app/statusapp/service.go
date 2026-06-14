package statusapp

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/inhere/skillc/internal/app/apputil"
	"github.com/inhere/skillc/internal/app/configapp"
	"github.com/inhere/skillc/internal/app/listapp"
	"github.com/inhere/skillc/internal/app/sourceapp"
	"github.com/inhere/skillc/internal/domain/agent"
	"github.com/inhere/skillc/internal/domain/skill"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/repoindex"
)

const (
	StatusInstalled   = "installed"
	StatusMissing     = "missing"
	StatusOutdated    = "outdated"
	StatusOrphan      = "orphan"
	StatusUnmanaged   = "unmanaged"
	StatusSourceError = "source-error"
)

type sourceSyncer interface {
	Sync(id string) error
}

type Req struct {
	Agent   string
	Scope   string
	Profile string
	WorkDir string
	Sync    bool
}

type Result struct {
	Items      []Item
	SyncFailed []SourceSyncError
	Summary    Summary
}

type Item struct {
	SkillID        string
	QualifiedName  string
	SourceID       string
	Agent          string
	Scope          string
	Profile        string
	Status         string
	CurrentVersion string
	LatestVersion  string
	InstalledPath  string
	Reason         string
}

type Summary struct {
	Installed   int
	Missing     int
	Outdated    int
	Orphan      int
	Unmanaged   int
	SourceError int
}

type SourceSyncError struct {
	SourceID string
	Reason   string
}

type Service struct {
	configFile    string
	baseDir       string
	configService *configapp.Service
	indexStore    *repoindex.Store
	syncer        sourceSyncer
}

func NewService(configFile string, baseDir string) *Service {
	return &Service{
		configFile:    configFile,
		baseDir:       baseDir,
		configService: configapp.NewService(configFile, baseDir),
		indexStore:    repoindex.NewStore(),
		syncer:        sourceapp.NewService(configFile, baseDir),
	}
}

func (s *Service) Run(req Req) (Result, error) {
	config, err := s.configService.Show()
	if err != nil {
		return Result{}, err
	}
	if req.WorkDir == "" {
		req.WorkDir = s.baseDir
	}
	scope, err := apputil.ParseScope(defaultString(req.Scope, string(agent.ScopeProject)))
	if err != nil {
		return Result{}, err
	}
	agentName := defaultString(req.Agent, agent.DefaultAgentName)
	canonicalAgent, _, ok := config.ResolveAgentTool(agentName)
	if !ok {
		return Result{}, fmt.Errorf("unsupported agent: %s", agentName)
	}

	listSvc := listapp.NewService(config.LockFile).WithRuntime(config, req.WorkDir)
	listItems, err := listSvc.List(canonicalAgent, string(scope))
	if err != nil {
		return Result{}, err
	}
	listItems = filterByProfile(listItems, req.Profile)

	result := Result{}
	syncFailed := map[string]string{}
	if req.Sync {
		result.SyncFailed = s.syncSources(config.Sources)
		for _, failed := range result.SyncFailed {
			syncFailed[failed.SourceID] = failed.Reason
		}
	}
	sourceIDs := sourceIDByQualifier(config.Sources)

	indexItems, err := s.loadIndex(config.IndexFile)
	if err != nil {
		return Result{}, err
	}
	for _, current := range listItems {
		result.Items = append(result.Items, classifyListItem(current, indexItems, syncFailed, sourceIDs))
	}
	if req.Profile == "" {
		unmanaged, err := listSvc.ScanUnrecorded(canonicalAgent, scope)
		if err != nil {
			return Result{}, err
		}
		for _, group := range unmanaged {
			for _, skillID := range group.Skills {
				result.Items = append(result.Items, Item{
					SkillID: skillID,
					Agent:   group.AgentName,
					Scope:   string(scope),
					Status:  StatusUnmanaged,
					Reason:  "installed directory has no lock record",
				})
			}
		}
	}
	sortItems(result.Items)
	result.Summary = summarize(result.Items)
	return result, nil
}

func (s *Service) loadIndex(path string) ([]skill.Skill, error) {
	items, err := s.indexStore.Load(path)
	if err == nil {
		return items, nil
	}
	if os.IsNotExist(err) {
		return []skill.Skill{}, nil
	}
	return nil, err
}

func (s *Service) syncSources(sources []sourcepkg.Source) []SourceSyncError {
	out := make([]SourceSyncError, 0)
	for _, source := range sources {
		id := source.ID
		if id == "" {
			continue
		}
		if err := s.syncer.Sync(id); err != nil {
			out = append(out, SourceSyncError{SourceID: id, Reason: err.Error()})
		}
	}
	return out
}

func classifyListItem(current listapp.Item, indexItems []skill.Skill, syncFailed map[string]string, sourceIDs map[string]string) Item {
	item := Item{
		SkillID:        current.SkillID,
		QualifiedName:  current.QualifiedName,
		SourceID:       current.SourceID,
		Agent:          current.Agent,
		Scope:          current.Scope,
		Profile:        current.Profile,
		CurrentVersion: current.Version,
		InstalledPath:  current.InstalledPath,
	}
	if reason, ok := syncFailed[current.SourceID]; ok {
		item.Status = StatusSourceError
		item.Reason = reason
		return item
	}
	if sourceID, reason, ok := sourceQualifiedSyncFailure(current.SourceQualifiedName, syncFailed, sourceIDs); ok {
		item.SourceID = sourceID
		item.Status = StatusSourceError
		item.Reason = reason
		return item
	}
	if current.Status == StatusMissing {
		item.Status = StatusMissing
		item.Reason = "installed path is missing"
		return item
	}
	latest, ok := findLatest(indexItems, current)
	if !ok {
		item.Status = StatusOrphan
		item.Reason = "skill not found in source index"
		return item
	}
	if item.SourceID == "" {
		item.SourceID = latest.SourceID
	}
	if item.QualifiedName == "" {
		item.QualifiedName = latest.QualifiedName
	}
	item.LatestVersion = latest.Version
	if current.Version != "" && latest.Version != "" && current.Version != latest.Version {
		item.Status = StatusOutdated
		item.Reason = fmt.Sprintf("version %s -> %s", current.Version, latest.Version)
		return item
	}
	item.Status = StatusInstalled
	return item
}

func findLatest(items []skill.Skill, current listapp.Item) (skill.Skill, bool) {
	if current.SourceID != "" {
		for _, item := range items {
			if current.SkillID == item.ID && item.SourceID != "" && current.SourceID == item.SourceID {
				return item, true
			}
		}
		return skill.Skill{}, false
	}
	if current.SourceQualifiedName != "" {
		for _, item := range items {
			if current.SkillID == item.ID && item.SourceQualifiedName != "" && current.SourceQualifiedName == item.SourceQualifiedName {
				return item, true
			}
		}
		return skill.Skill{}, false
	}
	if current.QualifiedName != "" {
		var found skill.Skill
		for _, item := range items {
			if current.SkillID != item.ID || item.QualifiedName == "" || current.QualifiedName != item.QualifiedName {
				continue
			}
			if found.ID != "" {
				return skill.Skill{}, false
			}
			found = item
		}
		return found, found.ID != ""
	}
	for _, item := range items {
		if current.SkillID == item.ID && item.SourceID == "" && item.SourceQualifiedName == "" && item.QualifiedName == "" {
			return item, true
		}
	}
	return skill.Skill{}, false
}

func sourceIDByQualifier(sources []sourcepkg.Source) map[string]string {
	out := make(map[string]string, len(sources))
	for _, source := range sources {
		if source.ID == "" {
			continue
		}
		out[source.ID] = source.ID
		if source.Name != "" {
			out[source.Name] = source.ID
		}
	}
	return out
}

func sourceQualifiedSyncFailure(sourceQualifiedName string, syncFailed map[string]string, sourceIDs map[string]string) (string, string, bool) {
	qualifier, _, ok := strings.Cut(sourceQualifiedName, "/")
	if !ok || qualifier == "" {
		return "", "", false
	}
	sourceID := qualifier
	if resolved, ok := sourceIDs[qualifier]; ok {
		sourceID = resolved
	}
	reason, failed := syncFailed[sourceID]
	return sourceID, reason, failed
}

func filterByProfile(items []listapp.Item, profileName string) []listapp.Item {
	if profileName == "" {
		return items
	}
	out := make([]listapp.Item, 0, len(items))
	for _, item := range items {
		if item.Profile == profileName {
			out = append(out, item)
		}
	}
	return out
}

func sortItems(items []Item) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Status != items[j].Status {
			return items[i].Status < items[j].Status
		}
		if items[i].SkillID != items[j].SkillID {
			return items[i].SkillID < items[j].SkillID
		}
		return filepath.Clean(items[i].InstalledPath) < filepath.Clean(items[j].InstalledPath)
	})
}

func summarize(items []Item) Summary {
	var summary Summary
	for _, item := range items {
		switch item.Status {
		case StatusInstalled:
			summary.Installed++
		case StatusMissing:
			summary.Missing++
		case StatusOutdated:
			summary.Outdated++
		case StatusOrphan:
			summary.Orphan++
		case StatusUnmanaged:
			summary.Unmanaged++
		case StatusSourceError:
			summary.SourceError++
		}
	}
	return summary
}

func defaultString(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
