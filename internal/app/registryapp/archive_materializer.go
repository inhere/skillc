package registryapp

import (
	"archive/zip"
	"bytes"
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
	if err := os.RemoveAll(req.TargetDir); err != nil {
		return "", err
	}
	if err := os.MkdirAll(req.TargetDir, 0o755); err != nil {
		return "", err
	}
	if strings.HasSuffix(strings.ToLower(req.URL), ".zip") {
		if err := extractZipBytes(data, req.TargetDir); err != nil {
			return "", err
		}
	} else {
		return "", fmt.Errorf("unsupported registry archive format: %s", req.URL)
	}
	if strings.TrimSpace(req.Checksum) == "" {
		return ArchiveChecksumMissingWarning, nil
	}
	return "", nil
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
