package doctorapp

import (
	"os/exec"

	"github.com/inhere/skillc/internal/app/configapp"
)

type Result struct {
	GitAvailable bool
	ConfigOK     bool
	LockFile     string
	RepoCacheDir string
}

type Service struct {
	configFile  string
	baseDir     string
	gitLookPath func(file string) (string, error)
}

func NewService(configFile string, baseDir string) *Service {
	return &Service{
		configFile:  configFile,
		baseDir:     baseDir,
		gitLookPath: exec.LookPath,
	}
}

func (s *Service) Check() (Result, error) {
	cfg, err := configapp.NewService(s.configFile, s.baseDir).Show()
	if err != nil {
		return Result{}, err
	}

	_, err = s.gitLookPath("git")
	return Result{
		GitAvailable: err == nil,
		ConfigOK:     true,
		LockFile:     cfg.LockFile,
		RepoCacheDir: cfg.RepoCacheDir,
	}, nil
}
