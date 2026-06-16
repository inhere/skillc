package webapp

import (
	"encoding/json"
	"errors"
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
	SkillID                  string `json:"skill_id"`
	QualifiedName            string `json:"qualified_name,omitempty"`
	SourceQualifiedName      string `json:"source_qualified_name,omitempty"`
	SourceID                 string `json:"source_id,omitempty"`
	Agent                    string `json:"agent"`
	Scope                    string `json:"scope"`
	Profile                  string `json:"profile,omitempty"`
	Status                   string `json:"status"`
	CurrentVersion           string `json:"current_version,omitempty"`
	LatestVersion            string `json:"latest_version,omitempty"`
	CurrentChecksum          string `json:"current_checksum,omitempty"`
	LatestChecksum           string `json:"latest_checksum,omitempty"`
	CurrentSourceResolvedRef string `json:"current_source_resolved_ref,omitempty"`
	LatestSourceResolvedRef  string `json:"latest_source_resolved_ref,omitempty"`
	InstalledPath            string `json:"installed_path,omitempty"`
	Reason                   string `json:"reason,omitempty"`
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
	mux.HandleFunc("/api/projects", s.handleProjects)
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/install-map", s.handleInstallMap)
	mux.HandleFunc("/api/version-drift", s.handleVersionDrift)
	mux.HandleFunc("/api/history", s.handleHistory)
	mux.HandleFunc("/api/profiles/", s.handleProfileAction)
	mux.HandleFunc("/api/profiles/save/plan", s.handleProfileSavePlan)
	mux.HandleFunc("/api/profiles/save/run", s.handleProfileSaveRun)
	mux.HandleFunc("/api/profiles/from-installed/plan", s.handleProfileFromInstalledPlan)
	mux.HandleFunc("/api/profiles/from-installed/run", s.handleProfileFromInstalledRun)
	mux.HandleFunc("/api/profiles/from-collection/plan", s.handleProfileFromCollectionPlan)
	mux.HandleFunc("/api/profiles/from-collection/run", s.handleProfileFromCollectionRun)
	mux.HandleFunc("/api/uninstall/plan", s.handleUninstallPlan)
	mux.HandleFunc("/api/uninstall/run", s.handleUninstallRun)
	mux.HandleFunc("/api/update/plan", s.handleUpdatePlan)
	mux.HandleFunc("/api/update/run", s.handleUpdateRun)
	mux.HandleFunc("/api/update/all/plan", s.handleUpdateAllPlan)
	mux.HandleFunc("/api/update/all/run", s.handleUpdateAllRun)
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

func (s *ManagerServer) handleProjects(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	result, err := s.manager.Projects()
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
		s.recordHistory(r, "profile.apply", map[string]string{"name": action.Name}, result, err)
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
	s.recordHistory(r, "update.run", WebUpdateReq{ManagerReq: req, Target: body.Target}, result, err)
	writeResult(w, result, err)
}

func (s *ManagerServer) handleUpdateAllPlan(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	body, ok := readJSONReq[updateAllProjectsReq](w, r)
	if !ok {
		return
	}
	req := managerReqFromQuery(r)
	result, err := s.manager.PlanAllProjectsUpdate(WebUpdateAllReq{ManagerReq: req, Target: body.Target, ProjectIDs: body.ProjectIDs})
	writeResult(w, result, err)
}

func (s *ManagerServer) handleUpdateAllRun(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	body, ok := requireConfirmedUpdateAllReq(w, r)
	if !ok {
		return
	}
	req := managerReqFromQuery(r)
	result, err := s.manager.RunAllProjectsUpdate(WebUpdateAllReq{ManagerReq: req, Target: body.Target, ProjectIDs: body.ProjectIDs})
	s.recordHistory(r, "update.all_projects", WebUpdateAllReq{ManagerReq: req, Target: body.Target, ProjectIDs: body.ProjectIDs}, result, err)
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
	s.recordHistory(r, "source.add", req, result, err)
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
	s.recordHistory(r, "source.sync", req, result, err)
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
	s.recordHistory(r, "source.remove", req, result, err)
	writeResult(w, result, err)
}

func (s *ManagerServer) handleHistory(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	result, err := s.manager.History(100)
	writeResult(w, result, err)
}

func (s *ManagerServer) handleProfileSavePlan(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	req, ok := readJSONReq[profileSaveReq](w, r)
	if !ok {
		return
	}
	result, err := s.manager.PlanProfileSave(req)
	writeResult(w, result, err)
}

func (s *ManagerServer) handleProfileSaveRun(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	req, ok := requireConfirmedProfileSaveReq(w, r)
	if !ok {
		return
	}
	result, err := s.manager.RunProfileSave(req)
	s.recordHistory(r, "profile.save", req, result, err)
	writeResult(w, result, err)
}

func (s *ManagerServer) handleProfileFromInstalledPlan(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	req, ok := readJSONReq[profileFromInstalledReq](w, r)
	if !ok {
		return
	}
	result, err := s.manager.PlanProfileFromInstalled(req)
	writeResult(w, result, err)
}

func (s *ManagerServer) handleProfileFromInstalledRun(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	req, ok := requireConfirmedProfileFromInstalledReq(w, r)
	if !ok {
		return
	}
	result, err := s.manager.RunProfileFromInstalled(req)
	s.recordHistory(r, "profile.from_installed", req, result, err)
	writeResult(w, result, err)
}

func (s *ManagerServer) handleProfileFromCollectionPlan(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	req, ok := readJSONReq[profileFromCollectionReq](w, r)
	if !ok {
		return
	}
	result, err := s.manager.PlanProfileFromCollection(req)
	writeResult(w, result, err)
}

func (s *ManagerServer) handleProfileFromCollectionRun(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	req, ok := requireConfirmedProfileFromCollectionReq(w, r)
	if !ok {
		return
	}
	result, err := s.manager.RunProfileFromCollection(req)
	s.recordHistory(r, "profile.from_collection", req, result, err)
	writeResult(w, result, err)
}

func (s *ManagerServer) handleUninstallPlan(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	req, ok := readJSONReq[uninstallActionReq](w, r)
	if !ok {
		return
	}
	result, err := s.manager.PlanUninstall(req)
	writeResult(w, result, err)
}

func (s *ManagerServer) handleUninstallRun(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	req, ok := requireConfirmedUninstallReq(w, r)
	if !ok {
		return
	}
	result, err := s.manager.RunUninstall(req)
	s.recordHistory(r, "uninstall.run", req, result, err)
	writeResult(w, result, err)
}

func (s *ManagerServer) recordHistory(r *http.Request, action string, req any, result any, err error) {
	status := "ok"
	errMsg := resultErrorMessage(result)
	if err != nil {
		status = "error"
		errMsg = err.Error()
	} else if errMsg != "" {
		status = "error"
	}
	managerReq := managerReqFromQuery(r)
	_ = newHistoryStore(s.manager.historyFile()).Append(HistoryRecord{
		Action:  action,
		Agent:   managerReq.Agent,
		Scope:   managerReq.Scope,
		WorkDir: managerReq.WorkDir,
		Status:  status,
		Request: req,
		Result:  result,
		Error:   errMsg,
	})
}

func resultErrorMessage(result any) string {
	switch item := result.(type) {
	case profileApplyActionResult:
		return item.Error
	case updateRunActionResult:
		return item.Error
	case updateAllProjectsActionResult:
		return item.Error
	case sourceActionResult:
		return item.Error
	case profileSaveResult:
		return item.Error
	case uninstallActionResult:
		return item.Error
	default:
		return ""
	}
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

func requireConfirmedProfileSaveReq(w http.ResponseWriter, r *http.Request) (profileSaveReq, bool) {
	req, ok := readJSONReq[profileSaveReq](w, r)
	if !ok {
		return profileSaveReq{}, false
	}
	if !req.Confirm {
		writeJSON(w, http.StatusBadRequest, errorResp{Error: "confirmation required"})
		return profileSaveReq{}, false
	}
	return req, true
}

func requireConfirmedProfileFromInstalledReq(w http.ResponseWriter, r *http.Request) (profileFromInstalledReq, bool) {
	req, ok := readJSONReq[profileFromInstalledReq](w, r)
	if !ok {
		return profileFromInstalledReq{}, false
	}
	if !req.Confirm {
		writeJSON(w, http.StatusBadRequest, errorResp{Error: "confirmation required"})
		return profileFromInstalledReq{}, false
	}
	return req, true
}

func requireConfirmedProfileFromCollectionReq(w http.ResponseWriter, r *http.Request) (profileFromCollectionReq, bool) {
	req, ok := readJSONReq[profileFromCollectionReq](w, r)
	if !ok {
		return profileFromCollectionReq{}, false
	}
	if !req.Confirm {
		writeJSON(w, http.StatusBadRequest, errorResp{Error: "confirmation required"})
		return profileFromCollectionReq{}, false
	}
	return req, true
}

func requireConfirmedUninstallReq(w http.ResponseWriter, r *http.Request) (uninstallActionReq, bool) {
	req, ok := readJSONReq[uninstallActionReq](w, r)
	if !ok {
		return uninstallActionReq{}, false
	}
	if !req.Confirm {
		writeJSON(w, http.StatusBadRequest, errorResp{Error: "confirmation required"})
		return uninstallActionReq{}, false
	}
	return req, true
}

func requireConfirmedUpdateAllReq(w http.ResponseWriter, r *http.Request) (updateAllProjectsReq, bool) {
	req, ok := readJSONReq[updateAllProjectsReq](w, r)
	if !ok {
		return updateAllProjectsReq{}, false
	}
	if !req.Confirm {
		writeJSON(w, http.StatusBadRequest, errorResp{Error: "confirmation required"})
		return updateAllProjectsReq{}, false
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
		var statusErr httpStatusError
		if errors.As(err, &statusErr) {
			writeJSON(w, statusErr.status, errorResp{Error: statusErr.msg})
			return
		}
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
		SkillID:                  item.SkillID,
		QualifiedName:            item.QualifiedName,
		SourceQualifiedName:      item.SourceQualifiedName,
		SourceID:                 item.SourceID,
		Agent:                    item.Agent,
		Scope:                    item.Scope,
		Profile:                  item.Profile,
		Status:                   item.Status,
		CurrentVersion:           item.CurrentVersion,
		LatestVersion:            item.LatestVersion,
		CurrentChecksum:          item.CurrentChecksum,
		LatestChecksum:           item.LatestChecksum,
		CurrentSourceResolvedRef: item.CurrentSourceResolvedRef,
		LatestSourceResolvedRef:  item.LatestSourceResolvedRef,
		InstalledPath:            item.InstalledPath,
		Reason:                   item.Reason,
	}
}
