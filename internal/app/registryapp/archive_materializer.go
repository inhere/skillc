package registryapp

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const ArchiveChecksumMissingWarning = "checksum is missing; archive integrity is not verified"

type archiveDownloadReq struct {
	URL       string
	Checksum  string
	TargetDir string
	Client    *http.Client
}

func extractArchiveDownload(req archiveDownloadReq) (string, error) {
	data, err := readArchiveBytes(req)
	if err != nil {
		return "", err
	}
	if err := verifyArchiveChecksum(data, req.Checksum); err != nil {
		return "", err
	}
	if err := os.RemoveAll(req.TargetDir); err != nil {
		return "", err
	}
	if err := os.MkdirAll(req.TargetDir, 0o755); err != nil {
		return "", err
	}
	lowerURL := strings.ToLower(req.URL)
	var extractErr error
	switch {
	case strings.HasSuffix(lowerURL, ".zip"):
		extractErr = extractZipBytes(data, req.TargetDir)
	case strings.HasSuffix(lowerURL, ".tar.gz"), strings.HasSuffix(lowerURL, ".tgz"):
		extractErr = extractTarGzBytes(data, req.TargetDir)
	default:
		extractErr = fmt.Errorf("unsupported registry archive format: %s", req.URL)
	}
	if extractErr != nil {
		return "", extractErr
	}
	if strings.TrimSpace(req.Checksum) == "" {
		return ArchiveChecksumMissingWarning, nil
	}
	return "", nil
}

func verifyArchiveChecksum(data []byte, checksum string) error {
	checksum = strings.TrimSpace(strings.ToLower(checksum))
	if checksum == "" {
		return nil
	}
	checksum = strings.TrimPrefix(checksum, "sha256:")
	sum := sha256.Sum256(data)
	actual := hex.EncodeToString(sum[:])
	if checksum != actual {
		return fmt.Errorf("registry archive checksum mismatch: expected %s, got %s", checksum, actual)
	}
	return nil
}

func readArchiveBytes(req archiveDownloadReq) ([]byte, error) {
	lowerURL := strings.ToLower(req.URL)
	if strings.HasPrefix(lowerURL, "http://") || strings.HasPrefix(lowerURL, "https://") {
		client := req.Client
		if client == nil {
			client = http.DefaultClient
		}
		resp, err := client.Get(req.URL)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("registry archive http status: %d", resp.StatusCode)
		}
		return io.ReadAll(resp.Body)
	}
	return os.ReadFile(req.URL)
}

func extractZipBytes(data []byte, targetDir string) error {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	for _, file := range reader.File {
		targetPath, err := safeArchiveTarget(targetDir, file.Name)
		if err != nil {
			return err
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}
		src, err := file.Open()
		if err != nil {
			return err
		}
		err = writeFileFromReader(targetPath, src)
		closeErr := src.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func extractTarGzBytes(data []byte, targetDir string) error {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		targetPath, err := safeArchiveTarget(targetDir, header.Name)
		if err != nil {
			return err
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return err
			}
			if err := writeFileFromReader(targetPath, tr); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported archive entry type for %s", header.Name)
		}
	}
}

func safeArchiveTarget(root string, name string) (string, error) {
	cleanName := filepath.Clean(filepath.FromSlash(name))
	if cleanName == "." || filepath.IsAbs(cleanName) || strings.HasPrefix(cleanName, ".."+string(filepath.Separator)) || cleanName == ".." {
		return "", fmt.Errorf("unsafe archive path: %s", name)
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	targetAbs, err := filepath.Abs(filepath.Join(rootAbs, cleanName))
	if err != nil {
		return "", err
	}
	if targetAbs != rootAbs && !strings.HasPrefix(targetAbs, rootAbs+string(filepath.Separator)) {
		return "", fmt.Errorf("unsafe archive path: %s", name)
	}
	return targetAbs, nil
}

func writeFileFromReader(path string, src io.Reader) error {
	dst, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	return err
}
