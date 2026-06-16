package registryapp

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/x/assert"
)

func TestArchiveMaterializer_ExtractsZipDownload(t *testing.T) {
	baseDir := t.TempDir()
	archivePath := filepath.Join(baseDir, "go-pro.zip")
	writeZipArchive(t, archivePath, map[string]string{
		"skills/go-pro/SKILL.md": "# Go Pro",
	})
	targetDir := filepath.Join(baseDir, "cache")

	warning, err := extractArchiveDownload(archiveDownloadReq{
		URL:       archivePath,
		TargetDir: targetDir,
	})

	assert.NoErr(t, err)
	assert.Eq(t, ArchiveChecksumMissingWarning, warning)
	assert.FileExists(t, filepath.Join(targetDir, "skills", "go-pro", "SKILL.md"))
}

func writeZipArchive(t *testing.T, path string, files map[string]string) {
	t.Helper()
	out, err := os.Create(path)
	assert.Require(t, assert.NoErr(t, err))
	defer out.Close()
	zw := zip.NewWriter(out)
	for name, body := range files {
		w, err := zw.Create(name)
		assert.Require(t, assert.NoErr(t, err))
		_, err = w.Write([]byte(body))
		assert.Require(t, assert.NoErr(t, err))
	}
	assert.Require(t, assert.NoErr(t, zw.Close()))
}
