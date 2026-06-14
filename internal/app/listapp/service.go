package listapp

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/inhere/skillc/internal/app/apputil"
	"github.com/inhere/skillc/internal/domain/agent"
	cfg "github.com/inhere/skillc/internal/domain/config"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/infra/lockstore"
)

type Item struct {
	SkillID             string
	QualifiedName       string
	SourceQualifiedName string
	Agent               string
	Scope               string
	Version             string
	SourceID            string
	SourceType          string
	Profile             string
	InstalledPath       string
	Checksum            string
	UpdatedAt           string
	Status              string
}

type Service struct {
	lockFile string
	store    *lockstore.Store
	config   cfg.Config
	workDir  string
}

func NewService(lockFile string) *Service {
	return &Service{
		lockFile: lockFile,
		store:    lockstore.NewStore(),
	}
}

func (s *Service) WithRuntime(config cfg.Config, workDir string) *Service {
	clone := *s
	clone.config = config
	clone.workDir = workDir
	return &clone
}

func (s *Service) List(agentName string, scope string) ([]Item, error) {
	records, err := s.store.Load(s.lockFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []Item{}, nil
		}
		return nil, err
	}

	items := make([]Item, 0)
	currentWorkDir := s.runtimeWorkDir()
	for scopeKey, grouped := range records {
		recordScope := scopeFromKey(scopeKey)
		if scope != "" && string(recordScope) != scope {
			continue
		}
		// project scope records are keyed by project dir; only show current project
		if recordScope == agent.ScopeProject && scopeKey != currentWorkDir {
			continue
		}
		for _, record := range grouped {
			for _, currentAgent := range record.Agents {
				if agentName != "" && currentAgent != agentName {
					continue
				}
				item, err := s.toItem(scopeKey, recordScope, record, currentAgent)
				if err != nil {
					return nil, err
				}
				items = append(items, item)
			}
		}
	}

	sort.Slice(items, func(i, j int) bool {
		left := items[i].QualifiedName
		if left == "" {
			left = items[i].SkillID
		}
		right := items[j].QualifiedName
		if right == "" {
			right = items[j].SkillID
		}
		if left == right {
			return items[i].Agent < items[j].Agent
		}
		return left < right
	})
	return items, nil
}

func (s *Service) toItem(scopeKey string, scope agent.Scope, record lockpkg.Record, agentName string) (Item, error) {
	installedPath, err := s.resolveInstalledPath(scopeKey, scope, agentName, record)
	if err != nil {
		return Item{}, err
	}
	status := "installed"
	if _, err := os.Stat(installedPath); err != nil {
		if os.IsNotExist(err) {
			status = "missing"
		} else {
			return Item{}, err
		}
	}
	return Item{
		SkillID:             record.SkillID,
		QualifiedName:       record.QualifiedName,
		SourceQualifiedName: record.SourceQualifiedName,
		Agent:               agentName,
		Scope:               string(scope),
		Version:             record.Version,
		SourceID:            record.SourceID,
		SourceType:          record.SourceType,
		Profile:             record.Profile,
		InstalledPath:       installedPath,
		Checksum:            record.Checksum,
		UpdatedAt:           record.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		Status:              status,
	}, nil
}

func (s *Service) resolveInstalledPath(scopeKey string, scope agent.Scope, agentName string, record lockpkg.Record) (string, error) {
	baseDir := s.runtimeWorkDir()
	if scope == agent.ScopeProject {
		baseDir = scopeKey
	}
	targetRoot, err := agent.ResolveInstallPath(s.runtimeConfig(), baseDir, agentName, scope)
	if err != nil {
		return "", err
	}
	return filepath.Join(targetRoot, record.SkillID), nil
}

func (s *Service) runtimeConfig() cfg.Config {
	if len(s.config.AgentTools) != 0 {
		return s.config
	}
	return cfg.DefaultConfig()
}

func (s *Service) runtimeWorkDir() string {
	if strings.TrimSpace(s.workDir) != "" {
		return s.workDir
	}
	cwd, err := os.Getwd()
	if err == nil && strings.TrimSpace(cwd) != "" {
		return cwd
	}
	return filepath.Dir(s.lockFile)
}

func scopeFromKey(scopeKey string) agent.Scope {
	return apputil.ScopeFromKey(scopeKey)
}

// UnrecordedGroup holds skill dir names found on disk but not in the lock file.
type UnrecordedGroup struct {
	AgentName string
	Skills    []string
}

// ScanUnrecorded scans the configured agent tool directories and returns skills
// that exist on disk but have no lock file record for the given scope.
func (s *Service) ScanUnrecorded(agentName string, scope agent.Scope) ([]UnrecordedGroup, error) {
	recordedPaths, err := s.collectRecordedPaths(agentName, scope)
	if err != nil {
		return nil, err
	}

	rc := s.runtimeConfig()
	workDir := s.runtimeWorkDir()

	var groups []UnrecordedGroup
	for name, tool := range rc.AgentTools {
		if agentName != "" {
			canonicalName, _, ok := rc.ResolveAgentTool(agentName)
			if !ok || canonicalName != name {
				continue
			}
		}
		skillsDir, err := resolveSkillsDir(rc, workDir, name, tool, scope)
		if err != nil || skillsDir == "" {
			continue
		}

		entries, err := os.ReadDir(skillsDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}

		var unrecorded []string
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			fullPath := filepath.Join(skillsDir, entry.Name())
			if !recordedPaths[fullPath] {
				unrecorded = append(unrecorded, entry.Name())
			}
		}
		if len(unrecorded) > 0 {
			sort.Strings(unrecorded)
			groups = append(groups, UnrecordedGroup{AgentName: name, Skills: unrecorded})
		}
	}

	sort.Slice(groups, func(i, j int) bool { return groups[i].AgentName < groups[j].AgentName })
	return groups, nil
}

// collectRecordedPaths returns the set of installed paths from the lock file.
func (s *Service) collectRecordedPaths(agentName string, scope agent.Scope) (map[string]bool, error) {
	records, err := s.store.Load(s.lockFile)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]bool{}, nil
		}
		return nil, err
	}

	workDir := s.runtimeWorkDir()
	rc := s.runtimeConfig()
	paths := make(map[string]bool)

	for scopeKey, grouped := range records {
		recScope := scopeFromKey(scopeKey)
		if recScope != scope {
			continue
		}
		if recScope == agent.ScopeProject && scopeKey != workDir {
			continue
		}
		for _, record := range grouped {
			for _, ag := range record.Agents {
				if agentName != "" {
					canonicalName, _, ok := rc.ResolveAgentTool(agentName)
					if !ok || canonicalName != ag {
						continue
					}
				}
				baseDir := workDir
				if recScope == agent.ScopeProject {
					baseDir = scopeKey
				}
				targetRoot, err := agent.ResolveInstallPath(rc, baseDir, ag, recScope)
				if err != nil {
					continue
				}
				flatPath := filepath.Join(targetRoot, record.SkillID)
				paths[flatPath] = true
			}
		}
	}
	return paths, nil
}

func resolveSkillsDir(rc cfg.Config, workDir string, agentName string, _ cfg.AgentToolConfig, scope agent.Scope) (string, error) {
	return agent.ResolveInstallPath(rc, workDir, agentName, scope)
}
