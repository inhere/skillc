package registryapp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/inhere/skillc/internal/app/sourceapp"
	"github.com/inhere/skillc/internal/domain/registry"
	"github.com/inhere/skillc/internal/domain/skill"
	"github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/gitx"
)

type gitSyncer interface {
	Sync(url, dir, ref string, opts gitx.SyncOptions) (string, error)
}

type materializer struct {
	git gitSyncer
}

func newMaterializer(git gitSyncer) *materializer {
	if git == nil {
		git = gitx.New("git")
	}
	return &materializer{git: git}
}

func (m *materializer) Materialize(entry registry.SkillEntry, targetDir string, opts gitx.SyncOptions) (skill.Skill, error) {
	resolvedRef := ""
	if strings.TrimSpace(entry.SourceURL) != "" {
		if sourceapp.IsGitURL(entry.SourceURL) {
			ref, err := m.git.Sync(entry.SourceURL, targetDir, entry.SourceRef, opts)
			if err != nil {
				return skill.Skill{}, err
			}
			resolvedRef = ref
		} else {
			if err := copyLocalSourceSnapshot(entry.SourceURL, targetDir); err != nil {
				return skill.Skill{}, err
			}
		}
	} else if strings.TrimSpace(entry.DownloadURL) != "" {
		if _, err := extractArchiveDownload(archiveDownloadReq{
			URL: entry.DownloadURL, Checksum: entry.Checksum, TargetDir: targetDir,
		}); err != nil {
			return skill.Skill{}, err
		}
	} else {
		return skill.Skill{}, fmt.Errorf("registry skill source_url or download_url is required for install: %s/%s", entry.RegistryID, entry.ID)
	}

	item, err := m.skillFromEntry(entry, targetDir)
	if err != nil {
		return skill.Skill{}, err
	}
	item.SourceResolvedRef = resolvedRef
	return item, nil
}

func copyLocalSourceSnapshot(sourceURL string, targetDir string) error {
	info, err := os.Stat(sourceURL)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("registry skill source_url must be git URL or local directory: %s", sourceURL)
	}
	if err := os.RemoveAll(targetDir); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(targetDir), 0o755); err != nil {
		return err
	}
	return os.CopyFS(targetDir, os.DirFS(sourceURL))
}

func (m *materializer) skillFromEntry(entry registry.SkillEntry, root string) (skill.Skill, error) {
	if strings.TrimSpace(entry.SourceURL) == "" && strings.TrimSpace(entry.DownloadURL) == "" {
		return skill.Skill{}, fmt.Errorf("registry skill source_url or download_url is required for install: %s/%s", entry.RegistryID, entry.ID)
	}
	return skill.Skill{
		ID:                  entry.ID,
		Name:                entry.Name,
		Description:         entry.Description,
		Version:             entry.Version,
		SupportedAgents:     append([]string(nil), entry.SupportedAgents...),
		SourceID:            entry.RegistryID,
		SourceName:          entry.RegistryID,
		SourceType:          source.TypeRegistry,
		QualifiedName:       entry.ID,
		SourceQualifiedName: entry.RegistryID + "/" + entry.ID,
		InstallEntry:        entry.InstallEntry,
		Path:                root,
		Checksum:            strings.TrimPrefix(entry.Checksum, "sha256:"),
		RegistryEntryID:     entry.ID,
		RegistryURL:         entry.RegistryURL,
		DownloadURL:         entry.DownloadURL,
		SourceURL:           entry.SourceURL,
		SourceRef:           entry.SourceRef,
	}, nil
}
