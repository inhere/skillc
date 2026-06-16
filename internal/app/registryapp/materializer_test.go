package registryapp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/x/assert"
	"github.com/inhere/skillc/internal/domain/registry"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/gitx"
)

func TestMaterializer_ToSkillMapsRegistryProvenance(t *testing.T) {
	baseDir := t.TempDir()
	entry := registry.SkillEntry{
		ID: "go-pro", Name: "Go Pro", Version: "1.2.0", RegistryID: "team",
		SourceURL: "https://example.com/skills.git", SourceRef: "main", InstallEntry: "skills/go-pro",
		RegistryURL: "https://example.com/registry.json",
		Checksum:    "sha256:abc123", SupportedAgents: []string{"codex"},
	}
	got, err := newMaterializer(nil).skillFromEntry(entry, filepath.Join(baseDir, "repo"))

	assert.NoErr(t, err)
	assert.Eq(t, "go-pro", got.ID)
	assert.Eq(t, "team", got.SourceID)
	assert.Eq(t, sourcepkg.TypeRegistry, got.SourceType)
	assert.Eq(t, "team/go-pro", got.SourceQualifiedName)
	assert.Eq(t, "skills/go-pro", got.InstallEntry)
	assert.Eq(t, "https://example.com/skills.git", got.SourceURL)
	assert.Eq(t, "main", got.SourceRef)
	assert.Eq(t, "go-pro", got.RegistryEntryID)
	assert.Eq(t, "https://example.com/registry.json", got.RegistryURL)
}

func TestMaterializer_MaterializeLocalSourceURLCopiesSnapshot(t *testing.T) {
	baseDir := t.TempDir()
	sourceRoot := filepath.Join(baseDir, "repo")
	targetRoot := filepath.Join(baseDir, "cache", "skills", "team", "go-pro", "1.0.0")
	assert.NoErr(t, os.MkdirAll(filepath.Join(sourceRoot, "skills", "go-pro"), 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceRoot, "skills", "go-pro", "SKILL.md"), []byte("# Go Pro"), 0o644))

	got, err := newMaterializer(nil).Materialize(registry.SkillEntry{
		ID: "go-pro", Name: "Go Pro", Version: "1.0.0", RegistryID: "team",
		SourceURL: sourceRoot, InstallEntry: "skills/go-pro",
	}, targetRoot, gitx.SyncOptions{})

	assert.NoErr(t, err)
	assert.Eq(t, targetRoot, got.Path)
	assert.FileExists(t, filepath.Join(targetRoot, "skills", "go-pro", "SKILL.md"))
}

func TestMaterializer_MaterializeDownloadURLArchive(t *testing.T) {
	baseDir := t.TempDir()
	archivePath := filepath.Join(baseDir, "go-pro.zip")
	writeZipArchive(t, archivePath, map[string]string{"skills/go-pro/SKILL.md": "# Go Pro"})
	targetRoot := filepath.Join(baseDir, "cache", "skills", "team", "go-pro", "1.0.0")

	got, err := newMaterializer(nil).Materialize(registry.SkillEntry{
		ID: "go-pro", Name: "Go Pro", Version: "1.0.0", RegistryID: "team",
		DownloadURL: archivePath, InstallEntry: "skills/go-pro",
	}, targetRoot, gitx.SyncOptions{})

	assert.NoErr(t, err)
	assert.Eq(t, targetRoot, got.Path)
	assert.Eq(t, archivePath, got.DownloadURL)
	assert.FileExists(t, filepath.Join(targetRoot, "skills", "go-pro", "SKILL.md"))
}
