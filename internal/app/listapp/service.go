package listapp

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

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
	flatPath := filepath.Join(targetRoot, record.SkillID)
	legacyPath := filepath.Join(targetRoot, legacyInstallDir(record.SkillID, record.SourceQualifiedName, record.SourceID))
	resolvedPath, err := preferExistingInstallPath(flatPath, legacyPath)
	if err != nil {
		return "", err
	}
	return resolvedPath, nil
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

func legacyInstallDir(skillID string, sourceQualifiedName string, sourceID string) string {
	if sourceQualifiedName != "" {
		return strings.ReplaceAll(sourceQualifiedName, "/", "--")
	}
	if sourceID != "" {
		return sourceID + "--" + skillID
	}
	return skillID
}

func preferExistingInstallPath(flatPath string, legacyPath string) (string, error) {
	if _, err := os.Stat(flatPath); err == nil {
		return flatPath, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if legacyPath == flatPath {
		return flatPath, nil
	}
	if _, err := os.Stat(legacyPath); err == nil {
		return legacyPath, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	return flatPath, nil
}

func scopeFromKey(scopeKey string) agent.Scope {
	if scopeKey == lockpkg.GlobalKey {
		return agent.ScopeUser
	}
	return agent.ScopeProject
}
