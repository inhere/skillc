package profileapp

import (
	"fmt"
	"sort"

	cfg "github.com/inhere/skillc/internal/domain/config"
	"github.com/inhere/skillc/internal/domain/profile"
	"github.com/inhere/skillc/internal/infra/configstore"
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

func (s *Service) List() ([]profile.NamedProfile, error) {
	config, err := s.store.Load(s.configFile, s.baseDir)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(config.Profiles))
	for name := range config.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]profile.NamedProfile, 0, len(names))
	for _, name := range names {
		out = append(out, profile.NamedProfile{Name: name, Profile: config.Profiles[name]})
	}
	return out, nil
}

func (s *Service) Show(name string) (profile.Profile, error) {
	if err := profile.ValidateName(name); err != nil {
		return profile.Profile{}, err
	}

	config, err := s.store.Load(s.configFile, s.baseDir)
	if err != nil {
		return profile.Profile{}, err
	}

	item, ok := config.Profiles[name]
	if !ok {
		return profile.Profile{}, fmt.Errorf("profile not found: %s", name)
	}
	return item, nil
}

func (s *Service) Create(name string, item profile.Profile) (profile.Profile, error) {
	if err := profile.ValidateName(name); err != nil {
		return profile.Profile{}, err
	}

	config, err := s.store.Load(s.configFile, s.baseDir)
	if err != nil {
		return profile.Profile{}, err
	}
	if _, ok := config.Profiles[name]; ok {
		return profile.Profile{}, fmt.Errorf("profile already exists: %s", name)
	}

	if err := s.saveLoaded(config, name, item); err != nil {
		return profile.Profile{}, err
	}
	return s.Show(name)
}

func (s *Service) Save(name string, item profile.Profile) error {
	if err := profile.ValidateName(name); err != nil {
		return err
	}

	config, err := s.store.Load(s.configFile, s.baseDir)
	if err != nil {
		return err
	}
	return s.saveLoaded(config, name, item)
}

func (s *Service) saveLoaded(config cfg.Config, name string, item profile.Profile) error {
	targets, err := profile.NormalizeTargets(item.Targets)
	if err != nil {
		return err
	}
	item.Targets = targets

	if config.Profiles == nil {
		config.Profiles = map[string]profile.Profile{}
	}
	config.Profiles[name] = item
	return s.store.Save(s.configFile, config, s.baseDir)
}
