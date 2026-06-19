package registry

import (
	"fmt"
	"path/filepath"
	"strings"
)

type Type string

const (
	TypeLocal    Type = "local"
	TypeHTTP     Type = "http"
	TypeProvider Type = "provider"
)

type Registry struct {
	ID           string `yaml:"id" json:"id"`
	Name         string `yaml:"name,omitempty" json:"name,omitempty"`
	Type         Type   `yaml:"type" json:"type"`
	Provider     string `yaml:"provider,omitempty" json:"provider,omitempty"`
	Path         string `yaml:"path,omitempty" json:"path,omitempty"`
	URL          string `yaml:"url,omitempty" json:"url,omitempty"`
	LastSyncAt   string `yaml:"last_sync_at,omitempty" json:"last_sync_at,omitempty"`
	Status       string `yaml:"status,omitempty" json:"status,omitempty"`
	ErrorMessage string `yaml:"error_message,omitempty" json:"error_message,omitempty"`
}

type Entry struct {
	ID          string   `json:"id"`
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
	Type        string   `json:"type"`
	URL         string   `json:"url,omitempty"`
	Path        string   `json:"path,omitempty"`
	Ref         string   `json:"ref,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	RegistryID  string   `json:"registry_id,omitempty"`
}

type Catalog struct {
	Skills  []SkillEntry `json:"skills,omitempty"`
	Sources []Entry      `json:"sources,omitempty"`
}

type SkillEntry struct {
	ID              string   `json:"id"`
	Name            string   `json:"name,omitempty"`
	Description     string   `json:"description,omitempty"`
	Version         string   `json:"version,omitempty"`
	SupportedAgents []string `json:"supported_agents,omitempty"`
	SourceURL       string   `json:"source_url,omitempty"`
	SourceRef       string   `json:"source_ref,omitempty"`
	DownloadURL     string   `json:"download_url,omitempty"`
	InstallEntry    string   `json:"install_entry,omitempty"`
	Checksum         string   `json:"checksum,omitempty"`
	Tags            []string `json:"tags,omitempty"`
	Homepage         string   `json:"homepage,omitempty"`
	RegistryID       string   `json:"registry_id,omitempty"`
	RegistryURL      string   `json:"registry_url,omitempty"`
}

func New(id string, name string, value string) (Registry, error) {
	return NewWithProvider(id, name, value, "")
}

func NewWithProvider(id string, name string, value string, provider string) (Registry, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return Registry{}, fmt.Errorf("registry path or url is required")
	}
	if name == "" {
		name = strings.TrimSpace(id)
	}
	if name == "" {
		name = registryNameFromValue(value)
	}
	id = NormalizeID(firstNonEmpty(id, name))
	if id == "" {
		return Registry{}, fmt.Errorf("registry id is required")
	}

	provider = NormalizeID(provider)
	if provider != "" {
		if provider != "skillsmp" {
			return Registry{}, fmt.Errorf("unsupported registry provider: %s", provider)
		}
		if !IsHTTPURL(value) {
			return Registry{}, fmt.Errorf("provider registry url must be http URL")
		}
		return Registry{ID: id, Name: name, Type: TypeProvider, Provider: provider, URL: strings.TrimRight(value, "/")}, nil
	}

	if IsHTTPURL(value) {
		return Registry{ID: id, Name: name, Type: TypeHTTP, URL: value}, nil
	}

	absPath, err := filepath.Abs(value)
	if err != nil {
		return Registry{}, err
	}
	return Registry{ID: id, Name: name, Type: TypeLocal, Path: filepath.Clean(absPath)}, nil
}

func (e Entry) Validate() error {
	if NormalizeID(e.ID) == "" {
		return fmt.Errorf("registry entry id is required")
	}
	switch strings.TrimSpace(strings.ToLower(e.Type)) {
	case "git":
		if strings.TrimSpace(e.URL) == "" {
			return fmt.Errorf("registry entry git url is required")
		}
	case "local":
		if strings.TrimSpace(e.Path) == "" {
			return fmt.Errorf("registry entry local path is required")
		}
	default:
		return fmt.Errorf("registry entry type is required")
	}
	return nil
}

func (e SkillEntry) Validate() error {
	if NormalizeID(e.ID) == "" {
		return fmt.Errorf("registry skill id is required")
	}
	if strings.TrimSpace(e.SourceURL) == "" && strings.TrimSpace(e.DownloadURL) == "" {
		return fmt.Errorf("registry skill source_url or download_url is required")
	}
	return nil
}

func NormalizeSkillEntry(entry SkillEntry, registryID string) (SkillEntry, error) {
	entry.ID = NormalizeID(entry.ID)
	entry.Name = strings.TrimSpace(entry.Name)
	if entry.Name == "" {
		entry.Name = entry.ID
	}
	entry.SourceURL = strings.TrimSpace(entry.SourceURL)
	entry.SourceRef = strings.TrimSpace(entry.SourceRef)
	entry.DownloadURL = strings.TrimSpace(entry.DownloadURL)
	entry.InstallEntry = strings.TrimSpace(entry.InstallEntry)
	if entry.InstallEntry == "" {
		entry.InstallEntry = "."
	}
	entry.Checksum = strings.TrimSpace(entry.Checksum)
	entry.RegistryURL = strings.TrimSpace(entry.RegistryURL)
	entry.RegistryID = registryID
	if err := entry.Validate(); err != nil {
		return SkillEntry{}, err
	}
	return entry, nil
}

func NormalizeID(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == '_' || r == '-':
			b.WriteRune(r)
			lastDash = r == '-'
		case !lastDash:
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-_")
}

func IsHTTPURL(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")
}

func registryNameFromValue(value string) string {
	if IsHTTPURL(value) {
		value = strings.TrimSuffix(value, "/")
	}
	base := filepath.Base(value)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	if base == "." || base == string(filepath.Separator) || base == "" {
		return "registry"
	}
	return base
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
