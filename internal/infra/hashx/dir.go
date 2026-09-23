package hashx

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// dirFilter 控制目录树哈希时忽略的内容。
type dirFilter struct {
	dirs      map[string]bool
	ignoreExt []string
	ignoreIn  map[string]bool
}

// sourceFilter 用于 source 目录扫描，只忽略 VCS 元数据。
var sourceFilter = dirFilter{dirs: map[string]bool{".git": true}}

// deployedFilter 用于已安装目录的部署指纹。
// 解释器缓存等运行时产物不属于技能内容，若参与哈希会把它们误判成"本地改动"。
var deployedFilter = dirFilter{
	dirs: map[string]bool{
		".git":         true,
		"__pycache__":  true,
		"node_modules": true,
	},
	ignoreExt: []string{".pyc", ".pyo"},
	ignoreIn:  map[string]bool{".DS_Store": true, "Thumbs.db": true},
}

// DeployedInfo 描述一个安装目录的部署内容。
type DeployedInfo struct {
	// Sum 是目录级指纹，与 SumDeployed 结果一致。
	Sum string
	// Files 是相对路径 → 文件内容哈希。
	Files map[string]string
}

// SumDir 计算目录内容哈希（相对路径 + 文件内容），忽略 .git 目录。
func SumDir(root string) (string, error) {
	return sumDir(root, sourceFilter)
}

// SumDeployed 计算已安装目录的部署指纹，用于判断部署内容是否被本地改动。
// 相比 SumDir 额外忽略 __pycache__、node_modules、*.pyc/*.pyo/.DS_Store/Thumbs.db。
func SumDeployed(root string) (string, error) {
	info, err := Deployed(root)
	if err != nil {
		return "", err
	}
	return info.Sum, nil
}

// Deployed 一次遍历得到部署指纹与逐文件哈希（忽略规则同 SumDeployed）。
// 逐文件哈希用于按文件三方合并，判断哪些文件被本地改动。
func Deployed(root string) (DeployedInfo, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if deployedFilter.dirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if deployedFilter.ignored(d.Name()) {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return DeployedInfo{}, err
	}

	sort.Strings(files)
	info := DeployedInfo{Files: make(map[string]string, len(files))}
	hash := sha256.New()
	for _, path := range files {
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return DeployedInfo{}, err
		}
		rel = filepath.ToSlash(rel)
		data, err := os.ReadFile(path)
		if err != nil {
			return DeployedInfo{}, err
		}
		info.Files[rel] = SumBytes(data)
		hash.Write([]byte(rel))
		hash.Write([]byte{0})
		hash.Write(data)
		hash.Write([]byte{0})
	}
	info.Sum = hex.EncodeToString(hash.Sum(nil))
	return info, nil
}

func sumDir(root string, filter dirFilter) (string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if filter.dirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if filter.ignored(d.Name()) {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return "", err
	}

	sort.Strings(files)
	hash := sha256.New()
	for _, path := range files {
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return "", err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		hash.Write([]byte(filepath.ToSlash(rel)))
		hash.Write([]byte{0})
		hash.Write(data)
		hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (f dirFilter) ignored(name string) bool {
	if f.ignoreIn[name] {
		return true
	}
	for _, ext := range f.ignoreExt {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	return false
}
