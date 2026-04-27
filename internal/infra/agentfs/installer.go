package agentfs

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
)

// Mode 表示技能安装方式
type Mode string

const (
	// ModeSymlink 通过创建符号链接安装（默认），便于多项目共享与同步更新
	ModeSymlink Mode = "symlink"
	// ModeCopy 通过递归复制文件树安装
	ModeCopy Mode = "copy"
)

// FallbackNotifier 在 symlink 失败回退到 copy 时被回调，用于打印提示
type FallbackNotifier func(sourceDir, targetDir string, err error)

type Installer struct {
	// Mode 安装方式，默认 ModeSymlink
	Mode Mode
	// OnSymlinkFallback 在 Windows 等环境下 symlink 失败回退 copy 时回调
	OnSymlinkFallback FallbackNotifier
}

func NewInstaller() *Installer {
	return &Installer{Mode: ModeSymlink}
}

// NewInstallerWithMode 创建指定安装模式的 Installer，空字符串视为默认 symlink
func NewInstallerWithMode(mode Mode) *Installer {
	if mode == "" {
		mode = ModeSymlink
	}
	return &Installer{Mode: mode}
}

// NormalizeMode 将任意输入归一化为合法的 Mode；非法或空值返回默认 ModeSymlink
func NormalizeMode(value string) Mode {
	switch Mode(value) {
	case ModeCopy:
		return ModeCopy
	case ModeSymlink:
		return ModeSymlink
	default:
		return ModeSymlink
	}
}

func (i *Installer) Install(sourceDir string, targetDir string) error {
	mode := i.Mode
	if mode == "" {
		mode = ModeSymlink
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

func installCopy(sourceDir string, targetDir string) error {
	// 若 targetDir 已是 symlink，先移除避免污染源目录
	if info, err := os.Lstat(targetDir); err == nil && info.Mode()&os.ModeSymlink != 0 {
		if err := os.Remove(targetDir); err != nil {
			return err
		}
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
