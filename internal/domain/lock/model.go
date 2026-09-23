package lock

import "time"

const GlobalKey = "__global__"

type File map[string][]Record

type Record struct {
	SkillID             string   `json:"skill_id"`
	QualifiedName       string   `json:"qualified_name"`
	SourceQualifiedName string   `json:"source_qualified_name"`
	Version             string   `json:"version"`
	SourceID            string   `json:"source_id"`
	SourceType          string   `json:"source_type"`
	SourceResolvedRef   string   `json:"source_resolved_ref"`
	Profile             string   `json:"profile,omitempty"`
	InstallEntry        string   `json:"install_entry"`
	Agents              []string `json:"agents"`
	Checksum            string   `json:"checksum"`
	// InstalledChecksum 是安装后目标目录的内容指纹，用于检测部署内容是否被本地改动。
	// 仅 copy 模式写入；link 模式下目标目录就是源目录，指纹无意义。
	InstalledChecksum string `json:"installed_checksum,omitempty"`
	// InstalledFiles 是安装时源内容的逐文件哈希（相对路径 → 内容哈希），仅 copy 模式写入。
	// 用于按文件三方合并：本地未改的文件跟随上游，本地改过的文件保留。
	InstalledFiles  map[string]string `json:"installed_files,omitempty"`
	RegistryEntryID string            `json:"registry_entry_id,omitempty"`
	RegistryURL     string            `json:"registry_url,omitempty"`
	DownloadURL     string            `json:"download_url,omitempty"`
	SourceURL       string            `json:"source_url,omitempty"`
	SourceRef       string            `json:"source_ref,omitempty"`
	InstalledAt     time.Time         `json:"installed_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	Pinned          bool              `json:"pinned"`
	InstallMode     string            `json:"install_mode,omitempty"`
}
