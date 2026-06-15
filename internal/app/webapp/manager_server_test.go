package webapp

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestManagerServerSummaryEndpoint(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeWebManagerFixture(t, baseDir)
	server := NewManagerServer(configFile, baseDir)

	rec := performManagerRequest(server, http.MethodGet, "/api/summary?agent=universal&scope=project")

	assert.Eq(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")
	assert.Contains(t, rec.Body.String(), `"source_count":1`)
	assert.Contains(t, rec.Body.String(), `"profile_count":1`)
	assert.Contains(t, rec.Body.String(), `"skill_count":2`)
	assert.Contains(t, rec.Body.String(), `"outdated":1`)
}

func TestManagerServerProfilesEndpoint(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeWebManagerFixture(t, baseDir)
	server := NewManagerServer(configFile, baseDir)

	rec := performManagerRequest(server, http.MethodGet, "/api/profiles")

	assert.Eq(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"name":"go-dev"`)
	assert.Contains(t, rec.Body.String(), `"description":"Go dev"`)
}

func TestManagerServerInstallMapEndpoint(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeWebManagerFixture(t, baseDir)
	server := NewManagerServer(configFile, baseDir)

	rec := performManagerRequest(server, http.MethodGet, "/api/install-map")

	assert.Eq(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"skill_id":"go-pro"`)
	assert.Contains(t, rec.Body.String(), `"agent":"universal"`)
	assert.Contains(t, rec.Body.String(), `"profile":"go-dev"`)
}

func TestManagerServerStatusEndpointUsesWebJSONModel(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeWebManagerFixture(t, baseDir)
	server := NewManagerServer(configFile, baseDir)

	rec := performManagerRequest(server, http.MethodGet, "/api/status?agent=universal&scope=project")

	assert.Eq(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"items"`)
	assert.Contains(t, rec.Body.String(), `"summary"`)
	assert.Contains(t, rec.Body.String(), `"outdated":1`)
	assert.NotContains(t, rec.Body.String(), `"Items"`)
	assert.NotContains(t, rec.Body.String(), `"Summary"`)
}

func TestManagerServerProfilePlanEndpoint(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeWebManagerFixture(t, baseDir)
	server := NewManagerServer(configFile, baseDir)

	rec := performManagerRequest(server, http.MethodPost, "/api/profiles/go-dev/plan?agent=universal&scope=project")

	assert.Eq(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"profile":"go-dev"`)
	assert.Contains(t, rec.Body.String(), `"action":"skip"`)
}

func TestManagerServerProfileApplyRequiresConfirmation(t *testing.T) {
	baseDir := t.TempDir()
	configFile := writeWebActionFixture(t, baseDir)
	server := NewManagerServer(configFile, baseDir)

	rec := performManagerRequestWithBody(server, http.MethodPost, "/api/profiles/go-dev/apply", strings.NewReader(`{}`))

	assert.Eq(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), `"confirmation required"`)
}

func TestManagerServerProfileApplyEndpointExecutesConfirmedApply(t *testing.T) {
	baseDir := t.TempDir()
	configFile := writeWebActionFixture(t, baseDir)
	server := NewManagerServer(configFile, baseDir)

	rec := performManagerRequestWithBody(server, http.MethodPost, "/api/profiles/go-dev/apply?agent=universal&scope=project", strings.NewReader(`{"confirm":true}`))

	assert.Eq(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"installed"`)
	assert.Contains(t, rec.Body.String(), `"skill_id":"review"`)
}

func TestManagerServerUpdateRunRequiresConfirmation(t *testing.T) {
	baseDir := t.TempDir()
	configFile := writeWebActionFixture(t, baseDir)
	server := NewManagerServer(configFile, baseDir)

	rec := performManagerRequestWithBody(server, http.MethodPost, "/api/update/run", strings.NewReader(`{"confirm":false}`))

	assert.Eq(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), `"confirmation required"`)
}

func TestManagerServerUpdateRunRejectsInvalidJSONBody(t *testing.T) {
	baseDir := t.TempDir()
	configFile := writeWebActionFixture(t, baseDir)
	server := NewManagerServer(configFile, baseDir)

	rec := performManagerRequestWithBody(server, http.MethodPost, "/api/update/run", strings.NewReader(`{`))

	assert.Eq(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), `"invalid json body"`)
}

func TestManagerServerUpdateRunEndpointExecutesConfirmedUpdate(t *testing.T) {
	baseDir := t.TempDir()
	configFile := writeWebActionFixture(t, baseDir)
	server := NewManagerServer(configFile, baseDir)

	rec := performManagerRequestWithBody(server, http.MethodPost, "/api/update/run?agent=universal&scope=project", strings.NewReader(`{"confirm":true,"target":"go-pro"}`))

	assert.Eq(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"updated"`)
	assert.Contains(t, rec.Body.String(), `"skill_id":"go-pro"`)
	assert.Contains(t, rec.Body.String(), `"version":"2.0.0"`)
}

func TestManagerServerRejectsInvalidProfileActionPath(t *testing.T) {
	baseDir := t.TempDir()
	configFile := writeWebActionFixture(t, baseDir)
	server := NewManagerServer(configFile, baseDir)

	tests := []struct {
		name string
		path string
	}{
		{name: "unknown action", path: "/api/profiles/go-dev/delete"},
		{name: "nested action", path: "/api/profiles/go-dev/extra/plan"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := performManagerRequestWithBody(server, http.MethodPost, tt.path, strings.NewReader(`{"confirm":true}`))

			assert.Eq(t, http.StatusNotFound, rec.Code)
		})
	}
}

func TestManagerServerRejectsInvalidMethods(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeWebManagerFixture(t, baseDir)
	server := NewManagerServer(configFile, baseDir)

	tests := []struct {
		name   string
		method string
		path   string
		code   int
	}{
		{name: "read endpoint rejects post", method: http.MethodPost, path: "/api/summary", code: http.StatusMethodNotAllowed},
		{name: "profile plan rejects get", method: http.MethodGet, path: "/api/profiles/go-dev/plan", code: http.StatusMethodNotAllowed},
		{name: "profile apply rejects get", method: http.MethodGet, path: "/api/profiles/go-dev/apply", code: http.StatusMethodNotAllowed},
		{name: "update run rejects get", method: http.MethodGet, path: "/api/update/run", code: http.StatusMethodNotAllowed},
		{name: "invalid profile plan path returns not found", method: http.MethodPost, path: "/api/profiles/go-dev/extra/plan", code: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := performManagerRequest(server, tt.method, tt.path)

			assert.Eq(t, tt.code, rec.Code)
		})
	}
}

func TestManagerServerIndexPageContainsAppShell(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeWebManagerFixture(t, baseDir)
	server := NewManagerServer(configFile, baseDir)

	rec := performManagerRequest(server, http.MethodGet, "/")
	body := rec.Body.String()

	assert.Eq(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")
	assert.Contains(t, body, "Skillc")
	assert.Contains(t, body, "Dashboard")
	assert.Contains(t, body, "Sources")
	assert.Contains(t, body, "Profiles")
	assert.Contains(t, body, "Version Drift")
}

func TestManagerServerStaticPageDoesNotUseExternalAssets(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeWebManagerFixture(t, baseDir)
	server := NewManagerServer(configFile, baseDir)

	rec := performManagerRequest(server, http.MethodGet, "/")
	body := rec.Body.String()

	assert.Eq(t, http.StatusOK, rec.Code)
	assert.NotContains(t, body, "https://")
	assert.NotContains(t, body, "http://")
}

func performManagerRequest(server *ManagerServer, method string, path string) *httptest.ResponseRecorder {
	return performManagerRequestWithBody(server, method, path, nil)
}

func performManagerRequestWithBody(server *ManagerServer, method string, path string, body io.Reader) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, body)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	return rec
}
