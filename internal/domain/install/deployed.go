package install

import (
	"os"

	"github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/infra/agentfs"
	"github.com/inhere/skillc/internal/infra/hashx"
)

// Drift 描述已安装目录内容相对 lock 记录里的部署指纹的偏离情况。
type Drift struct {
	// Tracked 表示记录里存在可比的部署指纹（copy 模式安装且已记录）。
	Tracked bool
	// Modified 表示当前目录内容与部署指纹不一致，即被本地改动或被其它内容覆盖过。
	Modified bool
	// Current 是当前目录的部署指纹，路径不存在或不可比时为空。
	Current string
}

// TracksDeployedChecksum 判断记录是否能跟踪部署内容。
// 只有 copy 模式会把内容复制到目标目录；link 模式下目标目录就是源目录，比对指纹会产生误报。
func TracksDeployedChecksum(record lock.Record) bool {
	return record.InstallMode == string(agentfs.ModeCopy) && record.InstalledChecksum != ""
}

// DetectDrift 比较目标目录当前内容与记录中的部署指纹。
// 未跟踪（旧记录、link 模式、目录不存在、目录已被换成链接）时返回零值 Drift。
func DetectDrift(record lock.Record, targetPath string) (Drift, error) {
	if !TracksDeployedChecksum(record) {
		return Drift{}, nil
	}
	current, ok, err := Fingerprint(targetPath)
	if err != nil || !ok {
		return Drift{}, err
	}
	return Drift{Tracked: true, Modified: current != record.InstalledChecksum, Current: current}, nil
}

// Fingerprint 计算已安装目录的部署指纹。
// 目录不存在、不是普通目录（已被替换为 symlink/junction）时返回 ok=false。
func Fingerprint(targetPath string) (string, bool, error) {
	info, err := os.Lstat(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	if !info.IsDir() || info.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 {
		return "", false, nil
	}
	sum, err := hashx.SumDeployed(targetPath)
	if err != nil {
		return "", false, err
	}
	return sum, true, nil
}
