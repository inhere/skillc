package sourceapp

import (
	"fmt"
	"path/filepath"

	cfg "github.com/inhere/skillc/internal/domain/config"
	domainsource "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/configstore"
	"github.com/inhere/skillc/internal/infra/gitx"
	"github.com/inhere/skillc/internal/infra/sourcestore"
)

type gitRunner interface {
	Clone(url, dir, ref string) error
}

type gitRunnerFunc func(url, dir, ref string) error

func (f gitRunnerFunc) Clone(url, dir, ref string) error {
	return f(url, dir, ref)
}

type Service struct {
	configFile string
	baseDir    string
	store      *configstore.YAMLStore
	git        gitRunner
}

func NewService(configFile string, baseDir string) *Service {
	return &Service{
		configFile: configFile,
		baseDir:    baseDir,
		store:      configstore.NewYAMLStore(),
		git:        gitx.New("git"),
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
	return s.store.Save(s.configFile, data)
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
			return s.store.Save(s.configFile, data)
		}

		targetDir := filepath.Join(data.RepoCacheDir, src.ID)
		err := s.git.Clone(src.URL, targetDir, src.Ref)
		if err != nil {
			data.Sources[i].Status = "error"
			data.Sources[i].ErrorMessage = err.Error()
			_ = s.store.Save(s.configFile, data)
			return err
		}
		data.Sources[i].Path = targetDir
		data.Sources[i].Status = "ready"
		data.Sources[i].ErrorMessage = ""
		return s.store.Save(s.configFile, data)
	}

	return fmt.Errorf("source not found: %s", id)
}

func (s *Service) load() (cfg.Config, error) {
	return s.store.Load(s.configFile, s.baseDir)
}
