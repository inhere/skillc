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
	if strings.TrimSpace(entry.SourceURL) == "" {
		return skill.Skill{}, fmt.Errorf("registry skill source_url is required for install: %s/%s", entry.RegistryID, entry.ID)
	}

	resolvedRef := ""
	if sourceapp.IsGitURL(entry.SourceURL) {
		ref, err := m.git.Sync(entry.SourceURL, targetDir, entry.SourceRef, opts)
		if err != nil {
			return skill.Skill{}, err
		}
		resolvedRef = ref
	} else {
		info, err := os.Stat(entry.SourceURL)
		if err != nil {
			return skill.Skill{}, err
		}
		if !info.IsDir() {
			return skill.Skill{}, fmt.Errorf("registry skill source_url must be git URL or local directory: %s", entry.SourceURL)
		}
		if err := os.RemoveAll(targetDir); err != nil {
			return skill.Skill{}, err
		}
		if err := os.MkdirAll(filepath.Dir(targetDir), 0o755); err != nil {
			return skill.Skill{}, err
		}
		if err := os.CopyFS(targetDir, os.DirFS(entry.SourceURL)); err != nil {
			return skill.Skill{}, err
		}
	}

	item, err := m.skillFromEntry(entry, targetDir)
	if err != nil {
		return skill.Skill{}, err
	}
	item.SourceResolvedRef = resolvedRef
	return item, nil
}

func (m *materializer) skillFromEntry(entry registry.SkillEntry, root string) (skill.Skill, error) {
	if strings.TrimSpace(entry.SourceURL) == "" {
		return skill.Skill{}, fmt.Errorf("registry skill source_url is required for install: %s/%s", entry.RegistryID, entry.ID)
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
