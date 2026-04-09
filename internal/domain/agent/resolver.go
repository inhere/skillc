package agent

import (
	"fmt"
	"path/filepath"

	cfg "github.com/inhere/skillc/internal/domain/config"
	"github.com/inhere/skillc/internal/infra/fsx"
)

func ResolveInstallPath(config cfg.Config, baseDir string, agentName string, scope Scope) (string, error) {
	_, tool, ok := config.ResolveAgentTool(agentName)
	if !ok {
		return "", fmt.Errorf("unsupported agent: %s", agentName)
	}

	path := tool.GetUserDir()
	if scope == ScopeProject {
		path = tool.GetProjectDir()
	}

	expanded, err := fsx.ExpandPath(path, baseDir)
	if err != nil {
		return "", err
	}
	return filepath.Join(expanded, "skills"), nil
}
