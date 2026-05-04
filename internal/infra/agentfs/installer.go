package agentfs

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Mode 表示技能安装方式
type Mode string

const (
	// ModeSymlink 通过创建符号链接安装，便于多项目共享与同步更新
	ModeSymlink Mode = "symlink"
	// ModeCopy 通过递归复制文件树安装
	ModeCopy Mode = "copy"
	// ModeJunction 通过 Windows directory junction 安装目录，兼容部分不跟随 symlink 的工具
	ModeJunction Mode = "junction"
)

// FallbackNotifier 在 symlink 失败回退到 copy 时被回调，用于打印提示
type FallbackNotifier func(sourceDir, targetDir string, err error)

type Installer struct {
	// Mode 安装方式，空值使用 DefaultMode
	Mode Mode
	// OnSymlinkFallback 在 Windows 等环境下 symlink 失败回退 copy 时回调
	OnSymlinkFallback FallbackNotifier
}

func NewInstaller() *Installer {
	return &Installer{Mode: DefaultMode()}
}

// NewInstallerWithMode 创建指定安装模式的 Installer，空字符串视为平台默认模式。
func NewInstallerWithMode(mode Mode) *Installer {
	if mode == "" {
		mode = DefaultMode()
	}
	return &Installer{Mode: mode}
}

// DefaultMode 返回当前平台的默认安装模式：Windows 使用 junction，其他系统使用 symlink。
func DefaultMode() Mode {
	if runtime.GOOS == "windows" {
		return ModeJunction
	}
	return ModeSymlink
}

func IsValidMode(value string) bool {
	value = strings.TrimSpace(value)
	switch Mode(value) {
	case ModeCopy, ModeJunction, ModeSymlink:
		return true
	default:
		return false
	}
}

// NormalizeMode 将任意输入归一化为合法的 Mode；非法或空值返回平台默认模式。
func NormalizeMode(value string) Mode {
	value = strings.TrimSpace(value)
	switch Mode(value) {
	case ModeCopy:
		return ModeCopy
	case ModeJunction:
		return ModeJunction
	case ModeSymlink:
		return ModeSymlink
	default:
		return DefaultMode()
	}
}

func (i *Installer) Install(sourceDir string, targetDir string) error {
	mode := i.Mode
	if mode == "" {
		mode = DefaultMode()
	}
	if mode == ModeSymlink {
		err := installSymlink(sourceDir, targetDir)
		if err == nil {
			return nil
		}
		// Windows 下若因权限不足创建 symlink 失败，回退到拷贝模式
		if runtime.GOOS == "windows" {
			if i.OnSymlinkFallback != nil {
				i.OnSymlinkFallback(sourceDir, targetDir, err)
			}
			return installCopy(sourceDir, targetDir)
		}
		return err
	}
	if mode == ModeJunction {
		return installJunction(sourceDir, targetDir)
	}
	return installCopy(sourceDir, targetDir)
}

func (i *Installer) Remove(targetDir string) error {
	// os.RemoveAll 对符号链接只删除链接本身，不会破坏源目录
	return os.RemoveAll(targetDir)
}

// installSymlink 在 targetDir 处创建一个指向 sourceDir 的符号链接。
// 若 targetDir 已存在（无论是文件、目录还是旧 symlink）会先清理。
func installSymlink(sourceDir string, targetDir string) error {
	absSource, err := filepath.Abs(sourceDir)
	if err != nil {
		return err
	}
	if _, err := os.Stat(absSource); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(targetDir), 0o755); err != nil {
		return err
	}
	if _, err := os.Lstat(targetDir); err == nil {
		if err := os.RemoveAll(targetDir); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.Symlink(absSource, targetDir)
}

// installJunction 在 Windows 上创建目录联接点。相比目录 symlink，
// junction 对一些只识别传统目录 reparse point 的扫描器更兼容。
func installJunction(sourceDir string, targetDir string) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("junction install mode is only supported on Windows")
	}
	absSource, err := filepath.Abs(sourceDir)
	if err != nil {
		return err
	}
	if info, err := os.Stat(absSource); err != nil {
		return err
	} else if !info.IsDir() {
		return fmt.Errorf("junction source must be a directory: %s", absSource)
	}
	if err := os.MkdirAll(filepath.Dir(targetDir), 0o755); err != nil {
		return err
	}
	if _, err := os.Lstat(targetDir); err == nil {
		if err := os.RemoveAll(targetDir); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	out, err := exec.Command("cmd", "/c", "mklink", "/J", targetDir, absSource).CombinedOutput()
	if err != nil {
		return fmt.Errorf("create junction: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func installCopy(sourceDir string, targetDir string) error {
	if err := removeLinkLikeTarget(targetDir); err != nil {
		return err
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return err
	}
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		targetPath := filepath.Join(targetDir, rel)
		if info.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}
		return copyFile(path, targetPath)
	})
}

func removeLinkLikeTarget(targetDir string) error {
	info, err := os.Lstat(targetDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	// Windows junctions may not expose os.ModeSymlink, but still set ModeIrregular.
	if info.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 {
		return os.Remove(targetDir)
	}
	return nil
}

func copyFile(sourcePath string, targetPath string) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}
	src, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return err
	}
	return nil
}
