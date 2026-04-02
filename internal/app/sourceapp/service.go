package sourceapp

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/gookit/goutil/x/ccolor"
	cfg "github.com/inhere/skillc/internal/domain/config"
	"github.com/inhere/skillc/internal/domain/skill"
	domainsource "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/configstore"
	"github.com/inhere/skillc/internal/infra/gitx"
	"github.com/inhere/skillc/internal/infra/repoindex"
	"github.com/inhere/skillc/internal/infra/sourcestore"
	"golang.org/x/term"
)

type gitRunner interface {
	Sync(url, dir, ref string, opts gitx.SyncOptions) (string, error)
}

type gitRunnerFunc func(url, dir, ref string, opts gitx.SyncOptions) (string, error)

func (f gitRunnerFunc) Sync(url, dir, ref string, opts gitx.SyncOptions) (string, error) {
	return f(url, dir, ref, opts)
}

type Service struct {
	configFile     string
	baseDir        string
	store          *configstore.YAMLStore
	git            gitRunner
	scanner        *repoindex.Scanner
	indexStore     *repoindex.Store
	now            func() time.Time
	isInteractive  func() bool
	progressWriter io.Writer
}

func NewService(configFile string, baseDir string) *Service {
	return &Service{
		configFile:     configFile,
		baseDir:        baseDir,
		store:          configstore.NewYAMLStore(),
		git:            gitx.New("git"),
		scanner:        repoindex.NewScanner(),
		indexStore:     repoindex.NewStore(),
		now:            time.Now,
		isInteractive:  func() bool { return term.IsTerminal(int(os.Stderr.Fd())) },
		progressWriter: os.Stderr,
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

// List 列出
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

// SyncAll 同步所有源
func (s *Service) SyncAll() error {
	list, err := s.List()
	if err != nil {
		return err
	}
	for _, src := range list {
		if err := s.Sync(src.ID); err != nil {
			return err
		}
	}
	return nil
}

// Sync 同步源
func (s *Service) Sync(id string) error {
	data, err := s.load()
	if err != nil {
		return err
	}

	for i, src := range data.Sources {
		if src.ID != id {
			continue
		}
		// 本地源直接返回
		if src.Type != domainsource.TypeGit {
			data.Sources[i].Status = "ready"
			data.Sources[i].ErrorMessage = ""
			data.Sources[i].LastSyncAt = s.now().UTC().Format(time.RFC3339)
			if err := s.store.Save(s.configFile, data); err != nil {
				return err
			}
			return s.rebuildIndex(data)
		}

		// Git 源同步
		targetDir := filepath.Join(data.RepoCacheDir, src.ID)
		if err := os.RemoveAll(targetDir); err != nil {
			data.Sources[i].Status = "error"
			data.Sources[i].ErrorMessage = err.Error()
			_ = s.store.Save(s.configFile, data)
			return err
		}

		ccolor.Infof("Syncing Git source %s to %s\n", src.ID, targetDir)
		resolvedRef, err := s.git.Sync(src.URL, targetDir, src.Ref, s.gitSyncOptions(data))
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

func (s *Service) gitSyncOptions(data cfg.Config) gitx.SyncOptions {
	opts := gitx.SyncOptions{ProxyURL: data.ProxyURL}
	if s.isInteractive != nil && s.isInteractive() {
		opts.Progress = s.progressWriter
	}
	return opts
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
