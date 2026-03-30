package configapp

import (
	"errors"

	cfg "github.com/inhere/skillc/internal/domain/config"
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

func (s *Service) Init() (cfg.Config, error) {
	data := cfg.DefaultConfig()
	if err := s.store.Save(s.configFile, data); err != nil {
		return cfg.Config{}, err
	}
	return data, nil
}

func (s *Service) Show() (cfg.Config, error) {
	return s.store.Load(s.configFile, s.baseDir)
}

func (s *Service) Get(key string) (string, error) {
	data, err := s.Show()
	if err != nil {
		return "", err
	}

	switch key {
	case "proxy_url":
		return data.ProxyURL, nil
	case "lock_file":
		return data.LockFile, nil
	case "repo_cache_dir":
		return data.RepoCacheDir, nil
	case "skill_cache_dir":
		return data.SkillCacheDir, nil
	case "registry_cache_dir":
		return data.RegistryCacheDir, nil
	default:
		return "", errors.New("unsupported config key: " + key)
	}
}

func (s *Service) Set(key string, value string) error {
	data, err := s.Show()
	if err != nil {
		return err
	}

	switch key {
	case "proxy_url":
		data.ProxyURL = value
	case "lock_file":
		data.LockFile = value
	case "repo_cache_dir":
		data.RepoCacheDir = value
	case "skill_cache_dir":
		data.SkillCacheDir = value
	case "registry_cache_dir":
		data.RegistryCacheDir = value
	default:
		return errors.New("unsupported config key: " + key)
	}

	return s.store.Save(s.configFile, data)
}
