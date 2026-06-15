package webapp

import "github.com/inhere/skillc/internal/app/installapp"

type uninstallActionReq struct {
	Confirm bool     `json:"confirm,omitempty"`
	Skills  []string `json:"skills"`
	Agent   string   `json:"agent,omitempty"`
	Scope   string   `json:"scope,omitempty"`
}

type uninstallActionResult struct {
	Error   string                         `json:"error,omitempty"`
	Plan    installapp.UninstallPlan       `json:"plan"`
	Removed []installapp.UninstallPlanItem `json:"removed"`
	Failed  []actionErrorItem              `json:"failed,omitempty"`
}

func (m *Manager) PlanUninstall(req uninstallActionReq) (installapp.UninstallPlan, error) {
	config, err := m.config()
	if err != nil {
		return installapp.UninstallPlan{}, err
	}
	return installapp.NewService(config.LockFile).WithRuntime(config, m.baseDir).PlanUninstall(installapp.UninstallReq{
		Skills:  req.Skills,
		Agent:   req.Agent,
		Scope:   req.Scope,
		WorkDir: m.baseDir,
	})
}

func (m *Manager) RunUninstall(req uninstallActionReq) (uninstallActionResult, error) {
	config, err := m.config()
	if err != nil {
		return uninstallActionResult{}, err
	}
	result, err := installapp.NewService(config.LockFile).WithRuntime(config, m.baseDir).RunUninstall(installapp.UninstallReq{
		Skills:  req.Skills,
		Agent:   req.Agent,
		Scope:   req.Scope,
		WorkDir: m.baseDir,
	})
	out := uninstallActionResult{
		Plan:    result.Plan,
		Removed: result.Removed,
		Failed:  installErrors(result.Failed),
	}
	if err != nil {
		if len(out.Plan.Items) > 0 || len(out.Failed) > 0 {
			out.Error = err.Error()
			return out, nil
		}
		return out, err
	}
	return out, nil
}
