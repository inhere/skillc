// Package apputil 提供 app 层跨模块共享的工具函数。
package apputil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/inhere/skillc/internal/domain/agent"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
)

// ParseScope 将字符串解析为 agent.Scope，不合法时返回错误。
func ParseScope(value string) (agent.Scope, error) {
	scope := agent.Scope(value)
	switch scope {
	case agent.ScopeUser, agent.ScopeProject:
		return scope, nil
	default:
		return "", fmt.Errorf("unsupported scope: %s", value)
	}
}

// ResolveScopeKey 根据 scope 和工作目录生成 lock file 中的 scope key。
func ResolveScopeKey(scope agent.Scope, workDir string) (string, error) {
	if scope == agent.ScopeUser {
		return lockpkg.GlobalKey, nil
	}
	if strings.TrimSpace(workDir) == "" {
		return "", fmt.Errorf("work dir is required for project scope")
	}
	absPath, err := filepath.Abs(workDir)
	if err != nil {
		return "", err
	}
	return filepath.Clean(absPath), nil
}

// ScopeFromKey 由 scope key 推断 agent.Scope。
func ScopeFromKey(scopeKey string) agent.Scope {
	if scopeKey == lockpkg.GlobalKey {
		return agent.ScopeUser
	}
	return agent.ScopeProject
}

// LegacyInstallDir 计算历史兼容安装目录名。
func LegacyInstallDir(skillID, sourceQualifiedName, sourceID string) string {
	if sourceQualifiedName != "" {
		return strings.ReplaceAll(sourceQualifiedName, "/", "--")
	}
	if sourceID != "" {
		return sourceID + "--" + skillID
	}
	return skillID
}

// PreferExistingInstallPath 优先返回磁盘上已存在的安装路径（flat 优先于 legacy）。
func PreferExistingInstallPath(flatPath, legacyPath string) (string, error) {
	if _, err := os.Stat(flatPath); err == nil {
		return flatPath, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if legacyPath == flatPath {
		return flatPath, nil
	}
	if _, err := os.Stat(legacyPath); err == nil {
		return legacyPath, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	return flatPath, nil
}
