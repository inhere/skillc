package sourceapp

import (
	"fmt"

	cfg "github.com/inhere/skillc/internal/domain/config"
	domainsource "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/configstore"
	"github.com/inhere/skillc/internal/infra/sourcestore"
)

type Service struct {
	configFile string
	baseDir    string
	store      *configstore.YAMLStore
}

func NewService(configFile string, baseDir string) *Service {
	return &Service{
		configFile: configFile,
		baseDir:    baseDir,
		store:      configstore.NewYAMLStore(),
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

func (s *Service) load() (cfg.Config, error) {
	return s.store.Load(s.configFile, s.baseDir)
}
