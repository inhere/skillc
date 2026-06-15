package webapp

import (
	"fmt"

	"github.com/inhere/skillc/internal/app/profileapp"
	"github.com/inhere/skillc/internal/domain/profile"
)

type profileSaveReq struct {
	Confirm      bool             `json:"confirm,omitempty"`
	Name         string           `json:"name"`
	Description  string           `json:"description,omitempty"`
	DefaultAgent string           `json:"default_agent,omitempty"`
	DefaultScope string           `json:"default_scope,omitempty"`
	InstallMode  string           `json:"install_mode,omitempty"`
	Targets      []profile.Target `json:"targets"`
}

type profileSavePlan struct {
	Name    string           `json:"name"`
	Mode    string           `json:"mode"`
	Profile profile.Profile  `json:"profile"`
	Added   []profile.Target `json:"added,omitempty"`
	Removed []profile.Target `json:"removed,omitempty"`
	Kept    []profile.Target `json:"kept,omitempty"`
}

type profileSaveResult struct {
	Error string          `json:"error,omitempty"`
	Plan  profileSavePlan `json:"plan"`
	Saved bool            `json:"saved"`
}

type profileFromInstalledReq struct {
	Confirm bool   `json:"confirm,omitempty"`
	Name    string `json:"name"`
	Agent   string `json:"agent,omitempty"`
	Scope   string `json:"scope,omitempty"`
}

type profileFromCollectionReq struct {
	Confirm  bool   `json:"confirm,omitempty"`
	Name     string `json:"name"`
	Selector string `json:"selector"`
}

type httpStatusError struct {
	status int
	msg    string
}

func (e httpStatusError) Error() string {
	return e.msg
}

func conflictError(msg string) error {
	return httpStatusError{status: 409, msg: msg}
}

func (m *Manager) PlanProfileSave(req profileSaveReq) (profileSavePlan, error) {
	plan, err := profileapp.NewService(m.configFile, m.baseDir).PlanSave(req.Name, profile.Profile{
		Description:  req.Description,
		DefaultAgent: req.DefaultAgent,
		DefaultScope: req.DefaultScope,
		InstallMode:  req.InstallMode,
		Targets:      req.Targets,
	})
	return toProfileSavePlan(plan), err
}

func (m *Manager) RunProfileSave(req profileSaveReq) (profileSaveResult, error) {
	plan, err := m.PlanProfileSave(req)
	if err != nil {
		return profileSaveResult{}, err
	}
	err = profileapp.NewService(m.configFile, m.baseDir).Save(req.Name, plan.Profile)
	result := profileSaveResult{Plan: plan, Saved: err == nil}
	if err != nil {
		result.Error = err.Error()
		return result, nil
	}
	return result, nil
}

func (m *Manager) PlanProfileFromInstalled(req profileFromInstalledReq) (profileSavePlan, error) {
	item, err := profileapp.NewService(m.configFile, m.baseDir).BuildFromInstalled(profileapp.CreateFromInstalledReq{
		Agent:   req.Agent,
		Scope:   req.Scope,
		WorkDir: m.baseDir,
	})
	if err != nil {
		return profileSavePlan{}, err
	}
	plan, err := profileapp.NewService(m.configFile, m.baseDir).PlanSave(req.Name, item)
	return toProfileSavePlan(plan), err
}

func (m *Manager) RunProfileFromInstalled(req profileFromInstalledReq) (profileSaveResult, error) {
	plan, err := m.PlanProfileFromInstalled(req)
	if err != nil {
		return profileSaveResult{}, err
	}
	if plan.Mode != "create" {
		return profileSaveResult{}, conflictError(fmt.Sprintf("profile already exists: %s", req.Name))
	}
	err = profileapp.NewService(m.configFile, m.baseDir).Save(req.Name, plan.Profile)
	result := profileSaveResult{Plan: plan, Saved: err == nil}
	if err != nil {
		result.Error = err.Error()
		return result, nil
	}
	return result, nil
}

func (m *Manager) PlanProfileFromCollection(req profileFromCollectionReq) (profileSavePlan, error) {
	item, err := profileapp.NewService(m.configFile, m.baseDir).BuildFromCollection(req.Selector)
	if err != nil {
		return profileSavePlan{}, err
	}
	plan, err := profileapp.NewService(m.configFile, m.baseDir).PlanSave(req.Name, item)
	return toProfileSavePlan(plan), err
}

func (m *Manager) RunProfileFromCollection(req profileFromCollectionReq) (profileSaveResult, error) {
	plan, err := m.PlanProfileFromCollection(req)
	if err != nil {
		return profileSaveResult{}, err
	}
	if plan.Mode != "create" {
		return profileSaveResult{}, conflictError(fmt.Sprintf("profile already exists: %s", req.Name))
	}
	err = profileapp.NewService(m.configFile, m.baseDir).Save(req.Name, plan.Profile)
	result := profileSaveResult{Plan: plan, Saved: err == nil}
	if err != nil {
		result.Error = err.Error()
		return result, nil
	}
	return result, nil
}

func toProfileSavePlan(plan profileapp.SavePlan) profileSavePlan {
	return profileSavePlan{
		Name:    plan.Name,
		Mode:    plan.Mode,
		Profile: plan.Profile,
		Added:   plan.Added,
		Removed: plan.Removed,
		Kept:    plan.Kept,
	}
}
