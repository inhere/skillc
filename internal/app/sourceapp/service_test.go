package sourceapp

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gookit/goutil/testutil/assert"
	"github.com/inhere/skillc/internal/app/configapp"
	"github.com/inhere/skillc/internal/app/searchapp"
	"github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/gitx"
)

func TestMain(m *testing.M) {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	testHome := filepath.Join(cwd, "testdata", "home")
	if err := os.RemoveAll(testHome); err != nil {
		panic(err)
	}
	if err := os.MkdirAll(testHome, 0o755); err != nil {
		panic(err)
	}
	if err := os.Setenv("HOME", testHome); err != nil {
		panic(err)
	}
	if err := os.Setenv("USERPROFILE", testHome); err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

type gitRunnerStub struct {
	syncFn func(url, dir, ref string, opts gitx.SyncOptions) (string, error)
}

func (s gitRunnerStub) Sync(url, dir, ref string, opts gitx.SyncOptions) (string, error) {
	return s.syncFn(url, dir, ref, opts)
}

type localPullerStub struct {
	pullFn func(dir string, opts gitx.SyncOptions) (string, error)
}

func (s localPullerStub) Pull(dir string, opts gitx.SyncOptions) (string, error) {
	if s.pullFn != nil {
		return s.pullFn(dir, opts)
	}
	return "", nil
}

func TestService_AddListRemoveLocalSource(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	service := NewService(configFile, baseDir)
	want, err := source.NewLocalSource(filepath.Join(baseDir, "skills"))
	assert.NoErr(t, err)

	src, err := service.AddLocal(filepath.Join(baseDir, "skills"))
	assert.NoErr(t, err)
	assert.Eq(t, want.ID, src.ID)
	assert.Eq(t, want.Name, src.Name)
	assert.Eq(t, want.Path, src.Path)

	_, err = service.AddLocal(filepath.Join(baseDir, "skills"))
	assert.Error(t, err)

	list, err := service.List()
	assert.NoErr(t, err)
	assert.Len(t, list, 1)
	assert.Eq(t, src.ID, list[0].ID)

	err = service.Remove(src.ID)
	assert.NoErr(t, err)

	list, err = service.List()
	assert.NoErr(t, err)
	assert.Len(t, list, 0)
}

func TestService_SyncLocalRebuildsIndex(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	sourceRoot := filepath.Join(baseDir, "skills")
	skillDir := filepath.Join(sourceRoot, "hello-skill")
	assert.NoErr(t, os.MkdirAll(skillDir, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`---
id: hello-skill
name: Hello Skill
description: Friendly greeting helper
supported_agents:
  - claude-code
install_entry: .
---
# Hello Skill
`), 0o644))

	cfg, err := configapp.NewService(configFile, baseDir).Init()
	assert.NoErr(t, err)

	service := NewService(configFile, baseDir)
	src, err := service.AddLocal(sourceRoot)
	assert.NoErr(t, err)

	err = service.Sync(src.ID)
	assert.NoErr(t, err)

	results, err := searchapp.NewService(cfg.IndexFile).Search("greeting", "claude-code", source.TypeLocal)
	assert.NoErr(t, err)
	assert.Len(t, results, 1)
	assert.Eq(t, "hello-skill", results[0].ID)
}

func TestService_AddGitAndSyncStatus(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	cfg, err := configapp.NewService(configFile, baseDir).Show()
	assert.NoErr(t, err)
	service := NewService(configFile, baseDir)
	service.git = gitRunnerStub{syncFn: func(url, dir, ref string, opts gitx.SyncOptions) (string, error) {
		assert.Eq(t, filepath.Join(cfg.RepoCacheDir, "git-repo"), dir)
		return "deadbeefcafebabe", os.MkdirAll(dir, 0o755)
	}}

	src, err := service.AddGit("https://example.com/repo.git", "main")
	assert.NoErr(t, err)
	assert.Eq(t, "main", src.Ref)

	err = service.Sync(src.ID)
	assert.NoErr(t, err)

	list, err := service.List()
	assert.NoErr(t, err)
	assert.Len(t, list, 1)
	assert.Eq(t, "ready", list[0].Status)
	assert.NotEmpty(t, list[0].Path)
	assert.Eq(t, "deadbeefcafebabe", list[0].ResolvedRef)
	assert.NotEmpty(t, list[0].LastSyncAt)
}

func TestService_AddGitWithoutRefSyncsDefaultBranch(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	service := NewService(configFile, baseDir)

	calledRef := "unexpected"
	service.git = gitRunnerStub{syncFn: func(url, dir, ref string, opts gitx.SyncOptions) (string, error) {
		calledRef = ref
		return "deadbeefcafebabe", os.MkdirAll(dir, 0o755)
	}}

	src, err := service.AddGit("https://example.com/repo.git", "")
	assert.NoErr(t, err)
	assert.Eq(t, "", src.Ref)

	err = service.Sync(src.ID)
	assert.NoErr(t, err)
	assert.Eq(t, "", calledRef)
}

func TestService_SyncGitBuildsSyncOptionsWithProxyAndProgressOnTTY(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	cfgService := configapp.NewService(configFile, baseDir)

	_, err := cfgService.Init()
	assert.NoErr(t, err)
	assert.NoErr(t, cfgService.Set("proxy_url", "http://localhost:7890"))

	service := NewService(configFile, baseDir)
	service.isInteractive = func() bool { return true }
	progressBuf := &bytes.Buffer{}
	service.progressWriter = progressBuf

	var calledOpts gitx.SyncOptions
	service.git = gitRunnerStub{syncFn: func(url, dir, ref string, opts gitx.SyncOptions) (string, error) {
		calledOpts = opts
		return "deadbeefcafebabe", os.MkdirAll(dir, 0o755)
	}}

	src, err := service.AddGit("https://example.com/repo.git", "main")
	assert.NoErr(t, err)

	err = service.Sync(src.ID)
	assert.NoErr(t, err)
	assert.Eq(t, "http://localhost:7890", calledOpts.ProxyURL)
	if calledOpts.Progress != progressBuf {
		t.Fatalf("expected progress writer to be forwarded on tty")
	}
}

func TestService_SyncGitBuildsSyncOptionsWithoutProgressWhenNotTTY(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	service := NewService(configFile, baseDir)
	service.isInteractive = func() bool { return false }

	var calledOpts gitx.SyncOptions
	service.git = gitRunnerStub{syncFn: func(url, dir, ref string, opts gitx.SyncOptions) (string, error) {
		calledOpts = opts
		return "deadbeefcafebabe", os.MkdirAll(dir, 0o755)
	}}

	src, err := service.AddGit("https://example.com/repo.git", "main")
	assert.NoErr(t, err)

	err = service.Sync(src.ID)
	assert.NoErr(t, err)
	assert.Eq(t, "", calledOpts.ProxyURL)
	assert.Nil(t, calledOpts.Progress)
}

func TestService_SyncGitSourceReusesExistingCachePath(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	service := NewService(configFile, baseDir)
	var syncDirs []string
	service.git = gitRunnerStub{syncFn: func(url, dir, ref string, opts gitx.SyncOptions) (string, error) {
		syncDirs = append(syncDirs, dir)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
		cacheMarker := filepath.Join(dir, "cached.txt")
		if len(syncDirs) == 1 {
			if err := os.WriteFile(cacheMarker, []byte("old"), 0o644); err != nil {
				return "", err
			}
		} else {
			_, err := os.Stat(cacheMarker)
			assert.NoErr(t, err)
		}
		return "deadbeefcafebabe", nil
	}}

	src, err := service.AddGit("https://example.com/repo.git", "main")
	assert.NoErr(t, err)

	err = service.Sync(src.ID)
	assert.NoErr(t, err)
	assert.NoErr(t, service.Sync(src.ID))

	list, err := service.List()
	assert.NoErr(t, err)
	assert.Eq(t, 2, len(syncDirs))
	assert.Eq(t, syncDirs[0], syncDirs[1])
	assert.Eq(t, syncDirs[0], list[0].Path)
}

func TestService_SyncGitUpdatesLastSyncAt(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	service := NewService(configFile, baseDir)
	service.now = func() time.Time { return time.Unix(1710000000, 0).UTC() }
	service.git = gitRunnerStub{syncFn: func(url, dir, ref string, opts gitx.SyncOptions) (string, error) {
		return "0123456789abcdef", os.MkdirAll(dir, 0o755)
	}}

	src, err := service.AddGit("https://example.com/repo.git", "main")
	assert.NoErr(t, err)
	assert.NoErr(t, service.Sync(src.ID))

	list, err := service.List()
	assert.NoErr(t, err)
	assert.Eq(t, "2024-03-09T16:00:00Z", list[0].LastSyncAt)
}

func TestService_SyncMissingGitSetsSourceError(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	service := NewService(configFile, baseDir)
	service.git = gitRunnerStub{syncFn: func(url, dir, ref string, opts gitx.SyncOptions) (string, error) {
		return "", errors.New("git executable not found")
	}}

	src, err := service.AddGit("https://example.com/repo.git", "main")
	assert.NoErr(t, err)

	err = service.Sync(src.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "git executable not found")

	list, err := service.List()
	assert.NoErr(t, err)
	assert.Eq(t, "error", list[0].Status)
	assert.Contains(t, list[0].ErrorMessage, "git executable not found")
}

func TestService_SyncLocalWithGitDirPullsBeforeIndex(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	sourceRoot := filepath.Join(baseDir, "skills")
	assert.NoErr(t, os.MkdirAll(filepath.Join(sourceRoot, ".git"), 0o755))

	service := NewService(configFile, baseDir)
	var pulledDir string
	service.localPull = localPullerStub{pullFn: func(dir string, opts gitx.SyncOptions) (string, error) {
		pulledDir = dir
		return "abc123", nil
	}}

	src, err := service.AddLocal(sourceRoot)
	assert.NoErr(t, err)

	err = service.Sync(src.ID)
	assert.NoErr(t, err)
	assert.Eq(t, sourceRoot, pulledDir)

	list, err := service.List()
	assert.NoErr(t, err)
	assert.Eq(t, "ready", list[0].Status)
}

func TestService_SyncLocalWithoutGitDirSkipsPull(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	sourceRoot := filepath.Join(baseDir, "skills")
	assert.NoErr(t, os.MkdirAll(sourceRoot, 0o755))

	service := NewService(configFile, baseDir)
	pulled := false
	service.localPull = localPullerStub{pullFn: func(dir string, opts gitx.SyncOptions) (string, error) {
		pulled = true
		return "", nil
	}}

	src, err := service.AddLocal(sourceRoot)
	assert.NoErr(t, err)

	err = service.Sync(src.ID)
	assert.NoErr(t, err)
	assert.False(t, pulled)
}

func TestService_MatchSourcesPartialID(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	skillsA := filepath.Join(baseDir, "golang-edge-skills")
	skillsB := filepath.Join(baseDir, "golang-base-skills")
	skillsC := filepath.Join(baseDir, "python-skills")
	for _, d := range []string{skillsA, skillsB, skillsC} {
		assert.NoErr(t, os.MkdirAll(d, 0o755))
	}

	service := NewService(configFile, baseDir)
	_, err := service.AddLocal(skillsA)
	assert.NoErr(t, err)
	_, err = service.AddLocal(skillsB)
	assert.NoErr(t, err)
	_, err = service.AddLocal(skillsC)
	assert.NoErr(t, err)

	list, err := service.List()
	assert.NoErr(t, err)
	assert.Len(t, list, 3)

	// 精确匹配
	matches, err := service.MatchSources(list[0].ID)
	assert.NoErr(t, err)
	assert.Len(t, matches, 1)
	assert.Eq(t, list[0].ID, matches[0].ID)

	// 部分匹配多个
	matches, err = service.MatchSources("golang")
	assert.NoErr(t, err)
	assert.Len(t, matches, 2)

	// 部分匹配单个
	matches, err = service.MatchSources("edge")
	assert.NoErr(t, err)
	assert.Len(t, matches, 1)

	// 无匹配
	matches, err = service.MatchSources("rust")
	assert.NoErr(t, err)
	assert.Len(t, matches, 0)
}

func TestService_EnsureSource_Local(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	localDir := filepath.Join(baseDir, "my-skills")
	assert.NoErr(t, os.MkdirAll(localDir, 0o755))

	service := NewService(configFile, baseDir)

	// 首次调用：新增 source
	src, isNew, err := service.EnsureSource(localDir, "")
	assert.NoErr(t, err)
	assert.True(t, isNew)
	assert.Eq(t, "local-my-skills", src.ID)

	// 再次调用：返回已有 source，不报错
	src2, isNew2, err := service.EnsureSource(localDir, "")
	assert.NoErr(t, err)
	assert.False(t, isNew2)
	assert.Eq(t, src.ID, src2.ID)

	// 只有一条记录
	list, err := service.List()
	assert.NoErr(t, err)
	assert.Len(t, list, 1)
}

func TestService_EnsureSource_Git(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")

	service := NewService(configFile, baseDir)

	// 首次调用：新增 git source
	src, isNew, err := service.EnsureSource("https://github.com/example/skills.git", "main")
	assert.NoErr(t, err)
	assert.True(t, isNew)
	assert.Eq(t, "git-example-skills", src.ID)

	// 再次调用：返回已有 source
	src2, isNew2, err := service.EnsureSource("https://github.com/example/skills.git", "main")
	assert.NoErr(t, err)
	assert.False(t, isNew2)
	assert.Eq(t, src.ID, src2.ID)

	list, err := service.List()
	assert.NoErr(t, err)
	assert.Len(t, list, 1)
}
