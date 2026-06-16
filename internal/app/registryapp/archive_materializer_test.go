package registryapp

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
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

func TestArchiveMaterializer_VerifiesSha256Checksum(t *testing.T) {
	baseDir := t.TempDir()
	archivePath := filepath.Join(baseDir, "go-pro.zip")
	writeZipArchive(t, archivePath, map[string]string{"SKILL.md": "# Go Pro"})
	checksum := "sha256:" + sha256FileHex(t, archivePath)

	warning, err := extractArchiveDownload(archiveDownloadReq{
		URL: archivePath, Checksum: checksum, TargetDir: filepath.Join(baseDir, "cache"),
	})

	assert.NoErr(t, err)
	assert.Eq(t, "", warning)
}

func TestArchiveMaterializer_RejectsChecksumMismatch(t *testing.T) {
	baseDir := t.TempDir()
	archivePath := filepath.Join(baseDir, "go-pro.zip")
	writeZipArchive(t, archivePath, map[string]string{"SKILL.md": "# Go Pro"})

	_, err := extractArchiveDownload(archiveDownloadReq{
		URL: archivePath, Checksum: strings.Repeat("0", 64), TargetDir: filepath.Join(baseDir, "cache"),
	})

	if err == nil {
		t.Fatal("expected checksum mismatch error")
	}
	assert.Contains(t, err.Error(), "checksum mismatch")
}

func TestArchiveMaterializer_ExtractsTarGzDownload(t *testing.T) {
	baseDir := t.TempDir()
	archivePath := filepath.Join(baseDir, "go-pro.tar.gz")
	writeTarGzArchive(t, archivePath, map[string]string{"skills/go-pro/SKILL.md": "# Go Pro"})
	targetDir := filepath.Join(baseDir, "cache")

	_, err := extractArchiveDownload(archiveDownloadReq{URL: archivePath, TargetDir: targetDir})

	assert.NoErr(t, err)
	assert.FileExists(t, filepath.Join(targetDir, "skills", "go-pro", "SKILL.md"))
}

func TestArchiveMaterializer_RejectsZipPathTraversal(t *testing.T) {
	baseDir := t.TempDir()
	archivePath := filepath.Join(baseDir, "evil.zip")
	writeZipArchive(t, archivePath, map[string]string{"../outside.txt": "bad"})

	_, err := extractArchiveDownload(archiveDownloadReq{URL: archivePath, TargetDir: filepath.Join(baseDir, "cache")})

	assert.Err(t, err)
	assert.Contains(t, err.Error(), "unsafe archive path")
	_, statErr := os.Stat(filepath.Join(baseDir, "outside.txt"))
	assert.True(t, os.IsNotExist(statErr))
}

func TestArchiveMaterializer_RejectsTarPathTraversal(t *testing.T) {
	baseDir := t.TempDir()
	archivePath := filepath.Join(baseDir, "evil.tgz")
	writeTarGzArchive(t, archivePath, map[string]string{"../outside.txt": "bad"})

	_, err := extractArchiveDownload(archiveDownloadReq{URL: archivePath, TargetDir: filepath.Join(baseDir, "cache")})

	assert.Err(t, err)
	assert.Contains(t, err.Error(), "unsafe archive path")
	_, statErr := os.Stat(filepath.Join(baseDir, "outside.txt"))
	assert.True(t, os.IsNotExist(statErr))
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

func writeTarGzArchive(t *testing.T, path string, files map[string]string) {
	t.Helper()
	out, err := os.Create(path)
	assert.Require(t, assert.NoErr(t, err))
	defer out.Close()
	gz := gzip.NewWriter(out)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		data := []byte(body)
		err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(data)), Typeflag: tar.TypeReg})
		assert.Require(t, assert.NoErr(t, err))
		_, err = tw.Write(data)
		assert.Require(t, assert.NoErr(t, err))
	}
	assert.Require(t, assert.NoErr(t, tw.Close()))
	assert.Require(t, assert.NoErr(t, gz.Close()))
}

func sha256FileHex(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	assert.Require(t, assert.NoErr(t, err))
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
