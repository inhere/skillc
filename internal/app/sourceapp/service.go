package sourceapp

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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

type localPuller interface {
	Pull(dir string, opts gitx.SyncOptions) (string, error)
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
	localPull      localPuller
	scanner        *repoindex.Scanner
	indexStore     *repoindex.Store
	now            func() time.Time
	isInteractive  func() bool
	progressWriter io.Writer
}

type AddReq struct {
	Value string
	Type  domainsource.Type
	ID    string
	Name  string
	Ref   string
}

func NewService(configFile string, baseDir string) *Service {
	gitClient := gitx.New("git")
	return &Service{
		configFile:     configFile,
		baseDir:        baseDir,
		store:          configstore.NewYAMLStore(),
		git:            gitClient,
		localPull:      gitClient,
		scanner:        repoindex.NewScanner(),
		indexStore:     repoindex.NewStore(),
		now:            time.Now,
		isInteractive:  func() bool { return term.IsTerminal(int(os.Stderr.Fd())) },
		progressWriter: os.Stderr,
	}
}

func (s *Service) Add(req AddReq) (domainsource.Source, error) {
	value := strings.TrimSpace(req.Value)
	if value == "" {
		return domainsource.Source{}, fmt.Errorf("source value is required")
	}
	opts := domainsource.SourceOptions{ID: req.ID, Name: req.Name}
	if req.Type == domainsource.TypeGit || (req.Type == "" && isGitURL(value)) {
		return s.AddGitWithOptions(value, req.Ref, opts)
	}
	return s.AddLocalWithOptions(value, opts)
}

func (s *Service) AddLocal(path string) (domainsource.Source, error) {
	return s.AddLocalWithOptions(path, domainsource.SourceOptions{})
}

func (s *Service) AddLocalWithOptions(path string, opts domainsource.SourceOptions) (domainsource.Source, error) {
	data, err := s.load()
	if err != nil {
		return domainsource.Source{}, err
	}

	src, err := domainsource.NewLocalSourceWithOptions(path, opts)
	if err != nil {
		return domainsource.Source{}, err
	}
	if sourcestore.ExistsByPath(data, src.Path) {
		return domainsource.Source{}, fmt.Errorf("source already exists: %s", src.Path)
	}
	if opts.ID != "" && sourcestore.ExistsByID(data, src.ID) {
		return domainsource.Source{}, fmt.Errorf("source already exists: %s", src.ID)
	}
	if opts.ID == "" {
		src.ID = uniqueSourceID(src.ID, data.Sources)
	}

	sourcestore.Add(&data, src)
	if err := s.store.Save(s.configFile, data, s.baseDir); err != nil {
		return domainsource.Source{}, err
	}
	return src, nil
}

func (s *Service) AddGit(url string, ref string) (domainsource.Source, error) {
	return s.AddGitWithOptions(url, ref, domainsource.SourceOptions{})
}

func (s *Service) AddGitWithOptions(url string, ref string, opts domainsource.SourceOptions) (domainsource.Source, error) {
	data, err := s.load()
	if err != nil {
		return domainsource.Source{}, err
	}

	src, err := domainsource.NewGitSourceWithOptions(url, ref, opts)
	if err != nil {
		return domainsource.Source{}, err
	}
	if existsByGitURL(data, src.URL) {
		return domainsource.Source{}, fmt.Errorf("source already exists: %s", src.URL)
	}
	if sourcestore.ExistsByID(data, src.ID) {
		if opts.ID != "" {
			return domainsource.Source{}, fmt.Errorf("source already exists: %s", src.ID)
		}
		src.ID = uniqueSourceID(src.ID, data.Sources)
	}

	sourcestore.Add(&data, src)
	if err := s.store.Save(s.configFile, data, s.baseDir); err != nil {
		return domainsource.Source{}, err
	}
	return src, nil
}

func (s *Service) Info(partial string) (domainsource.Source, error) {
	matches, err := s.MatchSources(partial)
	if err != nil {
		return domainsource.Source{}, err
	}
	switch len(matches) {
	case 0:
		return domainsource.Source{}, fmt.Errorf("source not found: %s", partial)
	case 1:
		return matches[0], nil
	default:
		return domainsource.Source{}, fmt.Errorf("multiple sources matched: %s", partial)
	}
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
	if err := s.store.Save(s.configFile, data, s.baseDir); err != nil {
		return err
	}
	return s.rebuildIndex(data)
}

// MatchSources 按部分 ID 匹配源（大小写不敏感包含匹配）。
// 精确匹配优先：若有一个源完全等于 partial，直接返回该单个源。
func (s *Service) MatchSources(partial string) ([]domainsource.Source, error) {
	list, err := s.List()
	if err != nil {
		return nil, err
	}
	lower := strings.ToLower(partial)
	for _, src := range list {
		if strings.ToLower(src.ID) == lower {
			return []domainsource.Source{src}, nil
		}
	}
	var matches []domainsource.Source
	for _, src := range list {
		if strings.Contains(strings.ToLower(src.ID), lower) {
			matches = append(matches, src)
		}
	}
	return matches, nil
}

// isGitURL 判断是否是 git URL（http/https/git@/ssh）
func isGitURL(s string) bool {
	return strings.HasPrefix(s, "http://") ||
		strings.HasPrefix(s, "https://") ||
		strings.HasPrefix(s, "git@") ||
		strings.HasPrefix(s, "ssh://")
}

func IsGitURL(value string) bool {
	return isGitURL(value)
}

// EnsureSource 保证 source 存在：若不存在则新增并 sync，若已存在则直接返回已有 source。
// urlOrPath 可以是 git URL 或本地路径；ref 仅对 git URL 有效。
func (s *Service) EnsureSource(urlOrPath string, ref string) (domainsource.Source, bool, error) {
	if isGitURL(urlOrPath) {
		src, err := s.AddGit(urlOrPath, ref)
		if err != nil {
			// 已存在时从列表中找回
			if strings.Contains(err.Error(), "source already exists") {
				existing, listErr := s.findGitSourceByURL(urlOrPath)
				if listErr != nil {
					return domainsource.Source{}, false, listErr
				}
				return existing, false, nil
			}
			return domainsource.Source{}, false, err
		}
		return src, true, nil
	}

	src, err := s.AddLocal(urlOrPath)
	if err != nil {
		if strings.Contains(err.Error(), "source already exists") {
			existing, listErr := s.findLocalSourceByPath(urlOrPath)
			if listErr != nil {
				return domainsource.Source{}, false, listErr
			}
			return existing, false, nil
		}
		return domainsource.Source{}, false, err
	}
	return src, true, nil
}

func (s *Service) findGitSourceByURL(url string) (domainsource.Source, error) {
	list, err := s.List()
	if err != nil {
		return domainsource.Source{}, err
	}
	for _, src := range list {
		if src.Type == domainsource.TypeGit && src.URL == url {
			return src, nil
		}
	}
	return domainsource.Source{}, fmt.Errorf("source not found for url: %s", url)
}

func (s *Service) findLocalSourceByPath(path string) (domainsource.Source, error) {
	list, err := s.List()
	if err != nil {
		return domainsource.Source{}, err
	}
	clean := filepath.Clean(path)
	absPath, err := filepath.Abs(clean)
	if err == nil {
		clean = absPath
	}
	for _, src := range list {
		if src.Type == domainsource.TypeLocal && filepath.Clean(src.Path) == clean {
			return src, nil
		}
	}
	return domainsource.Source{}, fmt.Errorf("source not found for path: %s", path)
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

	gitOpts := s.gitSyncOptions(data)

	for i, src := range data.Sources {
		if src.ID != id {
			continue
		}
		ccolor.Infof("- Syncing source %s(type=%s) ...\n", src.ID, src.Type)

		// 本地源：若目录含 .git 则先 pull，再重建 index
		if src.Type != domainsource.TypeGit {
			gitDir := filepath.Join(src.Path, ".git")
			if info, statErr := os.Stat(gitDir); statErr == nil && info.IsDir() {
				ccolor.Infof("- Local source %s has .git, pulling updates ...\n", src.ID)
				if _, pullErr := s.localPull.Pull(src.Path, gitOpts); pullErr != nil {
					ccolor.Warnf("Warning: git pull failed for %s: %v\n", src.Path, pullErr)
				}
			}
			data.Sources[i].Status = "ready"
			data.Sources[i].ErrorMessage = ""
			data.Sources[i].LastSyncAt = s.now().UTC().Format(time.RFC3339)
			if err := s.store.Save(s.configFile, data, s.baseDir); err != nil {
				return err
			}
			return s.rebuildIndex(data)
		}

		// Git 源同步 先更新再处理 index
		targetDir := filepath.Join(data.RepoCacheDir, src.ID)
		ccolor.Infof("> Update Git source %s\n", src.URL)
		resolvedRef, err := s.git.Sync(src.URL, targetDir, src.Ref, gitOpts)
		if err != nil {
			data.Sources[i].Status = "error"
			data.Sources[i].ErrorMessage = err.Error()
			_ = s.store.Save(s.configFile, data, s.baseDir)
			return err
		}

		data.Sources[i].Path = targetDir
		data.Sources[i].ResolvedRef = resolvedRef
		data.Sources[i].LastSyncAt = s.now().UTC().Format(time.RFC3339)
		data.Sources[i].Status = "ready"
		data.Sources[i].ErrorMessage = ""
		if err := s.store.Save(s.configFile, data, s.baseDir); err != nil {
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

func uniqueSourceID(id string, sources []domainsource.Source) string {
	if !containsSourceID(sources, id) {
		return id
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", id, i)
		if !containsSourceID(sources, candidate) {
			return candidate
		}
	}
}

func containsSourceID(sources []domainsource.Source, id string) bool {
	for _, source := range sources {
		if source.ID == id {
			return true
		}
	}
	return false
}

func existsByGitURL(config cfg.Config, url string) bool {
	for _, source := range config.Sources {
		if source.Type == domainsource.TypeGit && source.URL == url {
			return true
		}
	}
	return false
}
