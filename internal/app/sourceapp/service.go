package sourceapp

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	cfg "github.com/inhere/skillc/internal/domain/config"
	"github.com/inhere/skillc/internal/domain/skill"
	domainsource "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/configstore"
	"github.com/inhere/skillc/internal/infra/gitx"
	"github.com/inhere/skillc/internal/infra/repoindex"
	"github.com/inhere/skillc/internal/infra/sourcestore"
)

type gitRunner interface {
	Sync(url, dir, ref string) (string, error)
}

type gitRunnerFunc func(url, dir, ref string) (string, error)

func (f gitRunnerFunc) Sync(url, dir, ref string) (string, error) {
	return f(url, dir, ref)
}

type Service struct {
	configFile string
	baseDir    string
	store      *configstore.YAMLStore
	git        gitRunner
	scanner    *repoindex.Scanner
	indexStore *repoindex.Store
	now        func() time.Time
}

func NewService(configFile string, baseDir string) *Service {
	return &Service{
		configFile: configFile,
		baseDir:    baseDir,
		store:      configstore.NewYAMLStore(),
		git:        gitx.New("git"),
		scanner:    repoindex.NewScanner(),
		indexStore: repoindex.NewStore(),
		now:        time.Now,
	}
}

func (s *Service) AddLocal(path string) (domainsource.Source, error) {
	data, err := s.load()
	if err != nil {
		return domainsource.Source{}, err
	}

	src, err := domainsource.NewLocalSource(path)
	if err != nil {
		return domainsource.Source{}, err
	}
	if sourcestore.ExistsByPath(data, src.Path) {
		return domainsource.Source{}, fmt.Errorf("source already exists: %s", src.Path)
	}

	sourcestore.Add(&data, src)
	if err := s.store.Save(s.configFile, data); err != nil {
		return domainsource.Source{}, err
	}
	return src, nil
}

func (s *Service) AddGit(url string, ref string) (domainsource.Source, error) {
	data, err := s.load()
	if err != nil {
		return domainsource.Source{}, err
	}

	src, err := domainsource.NewGitSource(url, ref)
	if err != nil {
		return domainsource.Source{}, err
	}
	if sourcestore.ExistsByID(data, src.ID) {
		return domainsource.Source{}, fmt.Errorf("source already exists: %s", src.ID)
	}

	sourcestore.Add(&data, src)
	if err := s.store.Save(s.configFile, data); err != nil {
		return domainsource.Source{}, err
	}
	return src, nil
}

func (s *Service) List() ([]domainsource.Source, error) {
	data, err := s.load()
	if err != nil {
		return nil, err
	}
	return sourcestore.List(data), nil
}

func (s *Service) Remove(id string) error {
	data, err := s.load()
	if err != nil {
		return err
	}
	if !sourcestore.Remove(&data, id) {
		return fmt.Errorf("source not found: %s", id)
	}
	if err := s.store.Save(s.configFile, data); err != nil {
		return err
	}
	return s.rebuildIndex(data)
}

func (s *Service) Sync(id string) error {
	data, err := s.load()
	if err != nil {
		return err
	}

	for i, src := range data.Sources {
		if src.ID != id {
			continue
		}
		if src.Type != domainsource.TypeGit {
			data.Sources[i].Status = "ready"
			data.Sources[i].ErrorMessage = ""
			data.Sources[i].LastSyncAt = s.now().UTC().Format(time.RFC3339)
			if err := s.store.Save(s.configFile, data); err != nil {
				return err
			}
			return s.rebuildIndex(data)
		}

		targetDir := filepath.Join(data.RepoCacheDir, src.ID)
		if err := os.RemoveAll(targetDir); err != nil {
			data.Sources[i].Status = "error"
			data.Sources[i].ErrorMessage = err.Error()
			_ = s.store.Save(s.configFile, data)
			return err
		}
		resolvedRef, err := s.git.Sync(src.URL, targetDir, src.Ref)
		if err != nil {
			data.Sources[i].Status = "error"
			data.Sources[i].ErrorMessage = err.Error()
			_ = s.store.Save(s.configFile, data)
			return err
		}
		data.Sources[i].Path = targetDir
		data.Sources[i].ResolvedRef = resolvedRef
		data.Sources[i].LastSyncAt = s.now().UTC().Format(time.RFC3339)
		data.Sources[i].Status = "ready"
		data.Sources[i].ErrorMessage = ""
		if err := s.store.Save(s.configFile, data); err != nil {
			return err
		}
		return s.rebuildIndex(data)
	}

	return fmt.Errorf("source not found: %s", id)
}

func (s *Service) rebuildIndex(data cfg.Config) error {
	items := make([]skill.Skill, 0)
	for _, src := range data.Sources {
		if src.Status != "ready" || src.Path == "" {
			continue
		}
		scanned, err := s.scanner.Scan(src)
		if err != nil {
			return err
		}
		items = append(items, scanned...)
	}
	return s.indexStore.Save(data.IndexFile, items)
}

func (s *Service) load() (cfg.Config, error) {
	return s.store.Load(s.configFile, s.baseDir)
}
