// Package apputil 提供 app 层跨模块共享的工具函数。
package apputil

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	installpkg "github.com/inhere/skillc/internal/domain/install"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/infra/fsx"
)

// ErrLocalChanges 表示已安装目录存在本地改动，未开启 force 时拒绝覆盖或删除。
var ErrLocalChanges = errors.New("installed skill has local changes")

// OverwriteGuard 在覆盖已安装目录前保护本地改动。
type OverwriteGuard struct {
	// BackupRoot 备份根目录；为空表示不备份，此时无法判断是否有改动就拒绝覆盖。
	BackupRoot string
	// Force 为 true 时允许覆盖本地改动，覆盖前仍会先备份。
	Force bool
	// Now 便于测试注入时间，为空时使用 time.Now。
	Now func() time.Time
}

// GuardOverwrite 判断 targetPath 能否被覆盖，需要时先把旧目录快照到备份目录。
// 返回值是备份路径（未备份时为空）。
//
// 规则：
//   - 目标不存在或不是普通目录（link 安装）时不做任何处理；
//   - copy 模式且目录内容与部署指纹一致时直接覆盖；
//   - 内容与部署指纹不一致（本地改动）时默认返回 ErrLocalChanges，Force 时先备份再覆盖；
//   - 没有部署指纹（旧记录、未纳管的同名目录）时先备份再覆盖。
func (g OverwriteGuard) GuardOverwrite(record lockpkg.Record, scopeKey string, targetPath string) (string, error) {
	info, err := os.Lstat(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	if !info.IsDir() || info.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 {
		return "", nil
	}

	drift, err := installpkg.DetectDrift(record, targetPath)
	if err != nil {
		return "", err
	}
	if drift.Tracked && !drift.Modified {
		return "", nil
	}
	if drift.Tracked && !g.Force {
		return "", fmt.Errorf("%w: %s (rerun with --force to overwrite)", ErrLocalChanges, targetPath)
	}
	return g.Backup(scopeKey, record.SkillID, targetPath)
}

// Backup 把安装目录快照到 <BackupRoot>/<scope>/<skill>/<timestamp>，返回备份路径。
func (g OverwriteGuard) Backup(scopeKey string, skillID string, targetPath string) (string, error) {
	if strings.TrimSpace(g.BackupRoot) == "" {
		return "", fmt.Errorf("%w: %s (backup dir is not configured)", ErrLocalChanges, targetPath)
	}
	now := time.Now
	if g.Now != nil {
		now = g.Now
	}
	root := filepath.Join(g.BackupRoot, ScopeLabel(scopeKey), SanitizeName(skillID))
	dir, err := uniqueDir(root, now().Format("20060102-150405"))
	if err != nil {
		return "", err
	}
	if err := fsx.CopyDir(targetPath, dir); err != nil {
		return "", fmt.Errorf("backup %s: %w", targetPath, err)
	}
	return dir, nil
}

// ScopeLabel 把 lock 的 scope key 转成可读的备份目录名。
// 项目 scope 用「项目目录名-短哈希」，避免不同项目同名冲突。
func ScopeLabel(scopeKey string) string {
	if scopeKey == lockpkg.GlobalKey || strings.TrimSpace(scopeKey) == "" {
		return "user"
	}
	sum := sha256.Sum256([]byte(filepath.Clean(scopeKey)))
	base := SanitizeName(filepath.Base(filepath.Clean(scopeKey)))
	if base == "" || base == "." {
		base = "project"
	}
	return base + "-" + hex.EncodeToString(sum[:4])
}

// SanitizeName 去掉名称中的路径分隔符，保证可以作为单层目录名使用。
func SanitizeName(name string) string {
	replaced := strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return '_'
		}
		return r
	}, strings.TrimSpace(name))
	return strings.Trim(replaced, "._")
}

// uniqueDir 在 root 下创建 name 目录，重名时追加序号。
func uniqueDir(root string, name string) (string, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", err
	}
	candidate := filepath.Join(root, name)
	for i := 2; ; i++ {
		err := os.Mkdir(candidate, 0o755)
		if err == nil {
			return candidate, nil
		}
		if !os.IsExist(err) {
			return "", err
		}
		candidate = filepath.Join(root, fmt.Sprintf("%s-%d", name, i))
	}
}
