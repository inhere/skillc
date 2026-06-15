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
	Items []managerStatusItem `json:"items"`
}

type managerStatusResp struct {
	Items      []managerStatusItem         `json:"items"`
	SyncFailed []statusapp.SourceSyncError `json:"sync_failed,omitempty"`
	Summary    StatusSummary               `json:"summary"`
}

type managerStatusItem struct {
	SkillID             string `json:"skill_id"`
	QualifiedName       string `json:"qualified_name,omitempty"`
	SourceQualifiedName string `json:"source_qualified_name,omitempty"`
	SourceID            string `json:"source_id,omitempty"`
	Agent               string `json:"agent"`
	Scope               string `json:"scope"`
	Profile             string `json:"profile,omitempty"`
	Status              string `json:"status"`
	CurrentVersion      string `json:"current_version,omitempty"`
	LatestVersion       string `json:"latest_version,omitempty"`
	InstalledPath       string `json:"installed_path,omitempty"`
	Reason              string `json:"reason,omitempty"`
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
	mux.HandleFunc("/api/update/run", s.handleUpdateRun)
	mux.HandleFunc("/api/sources/add/plan", s.handleSourceAddPlan)
	mux.HandleFunc("/api/sources/add/run", s.handleSourceAddRun)
	mux.HandleFunc("/api/sources/sync/plan", s.handleSourceSyncPlan)
	mux.HandleFunc("/api/sources/sync/run", s.handleSourceSyncRun)
	mux.HandleFunc("/api/sources/remove/plan", s.handleSourceRemovePlan)
	mux.HandleFunc("/api/sources/remove/run", s.handleSourceRemoveRun)
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
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, toManagerStatusResp(result))
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
	action, ok := parseProfileAction(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}

	switch action.Action {
	case "plan":
		if !allowMethod(w, r, http.MethodPost) {
			return
		}
		result, err := s.manager.PlanProfileApply(action.Name, managerReqFromQuery(r))
		writeResult(w, result, err)
	case "apply":
		if !allowMethod(w, r, http.MethodPost) {
			return
		}
		if _, ok := requireConfirm(w, r); !ok {
			return
		}
		result, err := s.manager.ApplyProfile(action.Name, managerReqFromQuery(r))
		writeResult(w, result, err)
	default:
		http.NotFound(w, r)
	}
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
	items := make([]managerStatusItem, 0)
	for _, item := range statusResult.Items {
		if item.Status == statusapp.StatusOutdated || item.Status == statusapp.StatusMissing {
			items = append(items, toManagerStatusItem(item))
		}
	}
	writeJSON(w, http.StatusOK, updatePlan{Items: items})
}

func (s *ManagerServer) handleUpdateRun(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	body, ok := requireConfirm(w, r)
	if !ok {
		return
	}
	req := managerReqFromQuery(r)
	result, err := s.manager.RunUpdate(WebUpdateReq{ManagerReq: req, Target: body.Target})
	writeResult(w, result, err)
}

func (s *ManagerServer) handleSourceAddPlan(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	req, ok := readJSONReq[sourceActionReq](w, r)
	if !ok {
		return
	}
	result, err := s.manager.PlanSourceAdd(req)
	writeResult(w, result, err)
}

func (s *ManagerServer) handleSourceAddRun(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	req, ok := requireConfirmedSourceReq(w, r)
	if !ok {
		return
	}
	result, err := s.manager.RunSourceAdd(req)
	writeResult(w, result, err)
}

func (s *ManagerServer) handleSourceSyncPlan(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	req, ok := readJSONReq[sourceActionReq](w, r)
	if !ok {
		return
	}
	result, err := s.manager.PlanSourceSync(req)
	writeResult(w, result, err)
}

func (s *ManagerServer) handleSourceSyncRun(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	req, ok := requireConfirmedSourceReq(w, r)
	if !ok {
		return
	}
	result, err := s.manager.RunSourceSync(req)
	writeResult(w, result, err)
}

func (s *ManagerServer) handleSourceRemovePlan(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	req, ok := readJSONReq[sourceActionReq](w, r)
	if !ok {
		return
	}
	result, err := s.manager.PlanSourceRemove(req)
	writeResult(w, result, err)
}

func (s *ManagerServer) handleSourceRemoveRun(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	req, ok := requireConfirmedSourceReq(w, r)
	if !ok {
		return
	}
	result, err := s.manager.RunSourceRemove(req)
	writeResult(w, result, err)
}

func managerReqFromQuery(r *http.Request) ManagerReq {
	q := r.URL.Query()
	return ManagerReq{
		Agent:   q.Get("agent"),
		Scope:   q.Get("scope"),
		WorkDir: q.Get("workdir"),
	}
}

type profileAction struct {
	Name   string
	Action string
}

func parseProfileAction(path string) (profileAction, bool) {
	rest := strings.TrimPrefix(path, "/api/profiles/")
	if rest == path || rest == "" {
		return profileAction{}, false
	}
	name, tail, ok := strings.Cut(rest, "/")
	if !ok || name == "" || strings.Contains(name, "/") {
		return profileAction{}, false
	}
	switch tail {
	case "plan", "apply":
		return profileAction{Name: name, Action: tail}, true
	default:
		return profileAction{}, false
	}
}

type actionConfirmReq struct {
	Confirm bool   `json:"confirm"`
	Target  string `json:"target,omitempty"`
}

func readActionConfirmReq(r *http.Request) (actionConfirmReq, error) {
	if r.Body == nil || r.Body == http.NoBody {
		return actionConfirmReq{}, nil
	}
	defer r.Body.Close()

	var req actionConfirmReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return actionConfirmReq{}, fmt.Errorf("invalid json body")
	}
	return req, nil
}

func requireConfirm(w http.ResponseWriter, r *http.Request) (actionConfirmReq, bool) {
	req, err := readActionConfirmReq(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return actionConfirmReq{}, false
	}
	if !req.Confirm {
		writeJSON(w, http.StatusBadRequest, errorResp{Error: "confirmation required"})
		return actionConfirmReq{}, false
	}
	return req, true
}

func readJSONReq[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var req T
	if r.Body == nil || r.Body == http.NoBody {
		return req, true
	}
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid json body"))
		return req, false
	}
	return req, true
}

func requireConfirmedSourceReq(w http.ResponseWriter, r *http.Request) (sourceActionReq, bool) {
	req, ok := readJSONReq[sourceActionReq](w, r)
	if !ok {
		return sourceActionReq{}, false
	}
	if !req.Confirm {
		writeJSON(w, http.StatusBadRequest, errorResp{Error: "confirmation required"})
		return sourceActionReq{}, false
	}
	return req, true
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

func toManagerStatusResp(result statusapp.Result) managerStatusResp {
	items := make([]managerStatusItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toManagerStatusItem(item))
	}
	return managerStatusResp{
		Items:      items,
		SyncFailed: result.SyncFailed,
		Summary:    toStatusSummary(result.Summary),
	}
}

func toManagerStatusItem(item statusapp.Item) managerStatusItem {
	return managerStatusItem{
		SkillID:             item.SkillID,
		QualifiedName:       item.QualifiedName,
		SourceQualifiedName: item.SourceQualifiedName,
		SourceID:            item.SourceID,
		Agent:               item.Agent,
		Scope:               item.Scope,
		Profile:             item.Profile,
		Status:              item.Status,
		CurrentVersion:      item.CurrentVersion,
		LatestVersion:       item.LatestVersion,
		InstalledPath:       item.InstalledPath,
		Reason:              item.Reason,
	}
}
