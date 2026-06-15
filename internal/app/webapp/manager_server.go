package webapp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gookit/goutil/x/ccolor"
	"github.com/inhere/skillc/internal/app/statusapp"
)

type ManagerServer struct {
	configFile string
	baseDir    string
	manager    *Manager
}

type updatePlan struct {
	Items []statusapp.Item `json:"items"`
}

type errorResp struct {
	Error string `json:"error"`
}

func NewManagerServer(configFile string, baseDir string) *ManagerServer {
	return &ManagerServer{
		configFile: configFile,
		baseDir:    baseDir,
		manager:    NewManager(configFile, baseDir),
	}
}

func (s *ManagerServer) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/summary", s.handleSummary)
	mux.HandleFunc("/api/sources", s.handleSources)
	mux.HandleFunc("/api/collections", s.handleCollections)
	mux.HandleFunc("/api/skills", s.handleSkills)
	mux.HandleFunc("/api/profiles", s.handleProfiles)
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/install-map", s.handleInstallMap)
	mux.HandleFunc("/api/version-drift", s.handleVersionDrift)
	mux.HandleFunc("/api/profiles/", s.handleProfileAction)
	mux.HandleFunc("/api/update/plan", s.handleUpdatePlan)
	return mux
}

func (s *ManagerServer) Serve(host string, port int) error {
	addr := fmt.Sprintf("%s:%d", host, port)
	url := fmt.Sprintf("http://%s", addr)
	ccolor.Infof("Skillc web manager started: %s\n", url)
	ccolor.Infof("Project: %s\n", s.baseDir)
	ccolor.Println("Press Ctrl+C to stop")
	return http.ListenAndServe(addr, s.Handler())
}

func (s *ManagerServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(managerHTML))
}

func (s *ManagerServer) handleSummary(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	result, err := s.manager.Summary(managerReqFromQuery(r))
	writeResult(w, result, err)
}

func (s *ManagerServer) handleSources(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	result, err := s.manager.Sources()
	writeResult(w, result, err)
}

func (s *ManagerServer) handleCollections(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	result, err := s.manager.Collections(r.URL.Query().Get("source"))
	writeResult(w, result, err)
}

func (s *ManagerServer) handleSkills(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	result, err := s.manager.Skills(r.URL.Query().Get("keyword"))
	writeResult(w, result, err)
}

func (s *ManagerServer) handleProfiles(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	result, err := s.manager.Profiles()
	writeResult(w, result, err)
}

func (s *ManagerServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	result, err := s.manager.Status(managerReqFromQuery(r))
	writeResult(w, result, err)
}

func (s *ManagerServer) handleInstallMap(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	result, err := s.manager.InstallMap()
	writeResult(w, result, err)
}

func (s *ManagerServer) handleVersionDrift(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	result, err := s.manager.VersionDrift()
	writeResult(w, result, err)
}

func (s *ManagerServer) handleProfileAction(w http.ResponseWriter, r *http.Request) {
	name, ok := profilePlanName(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	result, err := s.manager.PlanProfileApply(name, managerReqFromQuery(r))
	writeResult(w, result, err)
}

func (s *ManagerServer) handleUpdatePlan(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	statusResult, err := s.manager.Status(managerReqFromQuery(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	items := make([]statusapp.Item, 0)
	for _, item := range statusResult.Items {
		if item.Status == statusapp.StatusOutdated || item.Status == statusapp.StatusMissing {
			items = append(items, item)
		}
	}
	writeJSON(w, http.StatusOK, updatePlan{Items: items})
}

func managerReqFromQuery(r *http.Request) ManagerReq {
	q := r.URL.Query()
	return ManagerReq{
		Agent:   q.Get("agent"),
		Scope:   q.Get("scope"),
		WorkDir: q.Get("workdir"),
	}
}

func profilePlanName(path string) (string, bool) {
	rest := strings.TrimPrefix(path, "/api/profiles/")
	if rest == path || rest == "" {
		return "", false
	}
	name, tail, ok := strings.Cut(rest, "/")
	if !ok || tail != "plan" || name == "" || strings.Contains(name, "/") {
		return "", false
	}
	return name, true
}

func allowMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method == method {
		return true
	}
	w.Header().Set("Allow", method)
	writeJSON(w, http.StatusMethodNotAllowed, errorResp{Error: "method not allowed"})
	return false
}

func writeResult(w http.ResponseWriter, result any, err error) {
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, errorResp{Error: err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
