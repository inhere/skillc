package projectapp

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/inhere/skillc/internal/app/apputil"
	"github.com/inhere/skillc/internal/domain/agent"
	cfg "github.com/inhere/skillc/internal/domain/config"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/domain/project"
	"github.com/inhere/skillc/internal/infra/configstore"
	"github.com/inhere/skillc/internal/infra/lockstore"
)

type AddReq struct {
	ID          string
	Name        string
	Path        string
	Description string
}

type ImportResult struct {
	Added   []project.Project `json:"added"`
	Skipped []ImportSkip      `json:"skipped,omitempty"`
}

type ImportSkip struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

type Service struct {
	configFile string
	baseDir    string
	store      *configstore.YAMLStore
	lockStore  *lockstore.Store
}

func NewService(configFile string, baseDir string) *Service {
	return &Service{
		configFile: configFile,
		baseDir:    baseDir,
		store:      configstore.NewYAMLStore(),
		lockStore:  lockstore.NewStore(),
	}
}

func (s *Service) List() ([]project.Project, error) {
	data, err := s.load()
	if err != nil {
		return nil, err
	}
	items := append([]project.Project(nil), data.Projects...)
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}

func (s *Service) Add(req AddReq) (project.Project, error) {
	data, err := s.load()
	if err != nil {
		return project.Project{}, err
	}
	item, err := project.New(req.ID, req.Name, req.Path)
	if err != nil {
		return project.Project{}, err
	}
	item.Description = req.Description
	if err := ensureProjectDir(item.Path); err != nil {
		return project.Project{}, err
	}
	if req.ID != "" && containsID(data.Projects, item.ID) {
		return project.Project{}, fmt.Errorf("project id already registered: %s", item.ID)
	}
	if req.ID == "" {
		item.ID = uniqueID(item.ID, data.Projects)
	}
	for _, current := range data.Projects {
		if filepath.Clean(current.Path) == filepath.Clean(item.Path) {
			return project.Project{}, fmt.Errorf("project path already registered: %s", item.Path)
		}
	}
	data.Projects = append(data.Projects, item)
	if err := s.save(data); err != nil {
		return project.Project{}, err
	}
	return item, nil
}

func (s *Service) Remove(id string) error {
	data, err := s.load()
	if err != nil {
		return err
	}
	out := data.Projects[:0]
	removed := false
	for _, item := range data.Projects {
		if item.ID == id {
			removed = true
			continue
		}
		out = append(out, item)
	}
	if !removed {
		return fmt.Errorf("project not found: %s", id)
	}
	data.Projects = out
	return s.save(data)
}

func (s *Service) ImportFromLock() (ImportResult, error) {
	data, err := s.load()
	if err != nil {
		return ImportResult{}, err
	}
	records, err := s.lockStore.Load(data.LockFile)
	if os.IsNotExist(err) {
		return ImportResult{}, nil
	}
	if err != nil {
		return ImportResult{}, err
	}
	result := ImportResult{}
	keys := make([]string, 0, len(records))
	for key := range records {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if key == lockpkg.GlobalKey || apputil.ScopeFromKey(key) != agent.ScopeProject {
			result.Skipped = append(result.Skipped, ImportSkip{Path: key, Reason: "not a project scope key"})
			continue
		}
		if _, err := os.Stat(key); err != nil {
			result.Skipped = append(result.Skipped, ImportSkip{Path: key, Reason: "project path does not exist"})
			continue
		}
		if containsPath(data.Projects, key) {
			result.Skipped = append(result.Skipped, ImportSkip{Path: key, Reason: "project already registered"})
			continue
		}
		item, err := project.New("", "", key)
		if err != nil {
			return ImportResult{}, err
		}
		item.ID = uniqueID(item.ID, data.Projects)
		data.Projects = append(data.Projects, item)
		result.Added = append(result.Added, item)
	}
	if len(result.Added) > 0 {
		if err := s.save(data); err != nil {
			return ImportResult{}, err
		}
	}
	return result, nil
}

func (s *Service) load() (cfg.Config, error) {
	return s.store.Load(s.configFile, s.baseDir)
}

func (s *Service) save(data cfg.Config) error {
	return s.store.Save(s.configFile, data, s.baseDir)
}

func ensureProjectDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("project path is not a directory: %s", path)
	}
	return nil
}

func uniqueID(id string, projects []project.Project) string {
	if !containsID(projects, id) {
		return id
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", id, i)
		if !containsID(projects, candidate) {
			return candidate
		}
	}
}

func containsID(projects []project.Project, id string) bool {
	for _, item := range projects {
		if item.ID == id {
			return true
		}
	}
	return false
}

func containsPath(projects []project.Project, path string) bool {
	path = filepath.Clean(path)
	for _, item := range projects {
		if filepath.Clean(item.Path) == path {
			return true
		}
	}
	return false
}
