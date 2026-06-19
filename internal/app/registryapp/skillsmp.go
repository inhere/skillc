package registryapp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/inhere/skillc/internal/domain/registry"
)

type skillsMPResp struct {
	Success bool `json:"success"`
	Data    struct {
		Skills []skillsMPItem `json:"skills"`
	} `json:"data"`
	Message string `json:"message"`
}

type skillsMPItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Author      string `json:"author"`
	Description string `json:"description"`
	GitHubURL   string `json:"githubUrl"`
	SkillURL    string `json:"skillUrl"`
}

func searchSkillsMP(client *http.Client, item registry.Registry, keyword string) ([]registry.SkillEntry, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil, fmt.Errorf("skillsmp search keyword is required")
	}
	base := strings.TrimRight(item.URL, "/")
	reqURL := base + "/api/v1/skills/search?q=" + url.QueryEscape(keyword) + "&page=1&limit=50"
	resp, err := client.Get(reqURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("skillsmp search failed: HTTP %d", resp.StatusCode)
	}
	var payload skillsMPResp
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("skillsmp search response is invalid: %w", err)
	}
	if !payload.Success {
		if payload.Message != "" {
			return nil, fmt.Errorf("skillsmp search failed: %s", payload.Message)
		}
		return nil, fmt.Errorf("skillsmp search failed")
	}

	var out []registry.SkillEntry
	for _, row := range payload.Data.Skills {
		entry, ok := skillsMPEntry(row, item)
		if ok {
			out = append(out, entry)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no installable skillsmp results for %q", keyword)
	}
	return out, nil
}

func skillsMPEntry(row skillsMPItem, item registry.Registry) (registry.SkillEntry, bool) {
	sourceURL, ref, installEntry, ok := parseGitHubTree(row.GitHubURL)
	if !ok {
		return registry.SkillEntry{}, false
	}
	entry, err := registry.NormalizeSkillEntry(registry.SkillEntry{
		ID:           firstNonEmpty(row.ID, row.Name),
		Name:         row.Name,
		Description:  row.Description,
		SourceURL:    sourceURL,
		SourceRef:    ref,
		InstallEntry: installEntry,
		Homepage:     row.SkillURL,
		RegistryURL:  strings.TrimRight(item.URL, "/"),
		Tags:         skillsMPTags(row.Author),
	}, item.ID)
	return entry, err == nil
}

func parseGitHubTree(raw string) (sourceURL string, ref string, installEntry string, ok bool) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || strings.ToLower(u.Host) != "github.com" {
		return "", "", "", false
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 5 || parts[2] != "tree" {
		return "", "", "", false
	}
	ref = parts[3]
	installEntry = path.Join(parts[4:]...)
	if ref == "" || installEntry == "." || installEntry == "" {
		return "", "", "", false
	}
	return fmt.Sprintf("https://github.com/%s/%s.git", parts[0], parts[1]), ref, installEntry, true
}

func skillsMPTags(author string) []string {
	tags := []string{"skillsmp"}
	author = strings.TrimSpace(author)
	if author != "" {
		tags = append(tags, "author:"+author)
	}
	return tags
}
