package registryapp

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gookit/goutil/x/assert"
	"github.com/inhere/skillc/internal/domain/registry"
)

func TestSearchSkillsMPMapsGitHubTreeURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Eq(t, "/api/v1/skills/search", r.URL.Path)
		assert.Eq(t, "go", r.URL.Query().Get("q"))
		_, _ = w.Write([]byte(`{"success":true,"data":{"skills":[{"id":"owner-repo-skills-go-skill-md","name":"go","author":"Owner","description":"Go helper","githubUrl":"https://github.com/Owner/Repo/tree/main/skills/go","skillUrl":"https://skillsmp.com/creators/owner/repo/skills-go","stars":3,"updatedAt":"1781679284"}]}}`))
	}))
	defer server.Close()

	got, err := searchSkillsMP(&http.Client{Timeout: time.Second}, registry.Registry{ID: "skillsmp", Type: registry.TypeProvider, Provider: "skillsmp", URL: server.URL}, "go")

	assert.NoErr(t, err)
	assert.Len(t, got, 1)
	assert.Eq(t, "owner-repo-skills-go-skill-md", got[0].ID)
	assert.Eq(t, "go", got[0].Name)
	assert.Eq(t, "Go helper", got[0].Description)
	assert.Eq(t, "https://github.com/Owner/Repo.git", got[0].SourceURL)
	assert.Eq(t, "main", got[0].SourceRef)
	assert.Eq(t, "skills/go", got[0].InstallEntry)
	assert.Eq(t, "https://skillsmp.com/creators/owner/repo/skills-go", got[0].Homepage)
	assert.Eq(t, "skillsmp", got[0].RegistryID)
	assert.Contains(t, got[0].Tags, "skillsmp")
	assert.Contains(t, got[0].Tags, "author:Owner")
}

func TestSearchSkillsMPSkipsUninstallableResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"success":true,"data":{"skills":[{"id":"bad","name":"bad","githubUrl":"https://github.com/Owner/Repo"}]}}`))
	}))
	defer server.Close()

	_, err := searchSkillsMP(&http.Client{Timeout: time.Second}, registry.Registry{ID: "skillsmp", Type: registry.TypeProvider, Provider: "skillsmp", URL: server.URL}, "go")

	assert.Err(t, err)
	assert.Contains(t, err.Error(), "no installable skillsmp results")
}
