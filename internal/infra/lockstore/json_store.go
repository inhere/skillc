package lockstore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/infra/fsx"
)

type Store struct {
	// agentResolver 把 agent 名称/别名统一为正式名称；为空时保持原样。
	agentResolver func(string) string
}

func NewStore() *Store {
	return &Store{}
}

// WithAgentResolver 注入 agent 名称归一化函数。
// Load/Save 会把记录里的 agent 统一为正式名称并去重，避免同一 agent 同时以别名
// 和正式名存在（例如 agents/universal、claude/claude-code）。
func (s *Store) WithAgentResolver(resolver func(string) string) *Store {
	clone := *s
	clone.agentResolver = resolver
	return &clone
}

func (s *Store) Save(path string, items lockpkg.File) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	items = s.normalize(items)
	data, err := json.Marshal(items)
	if err != nil {
		return err
	}
	return fsx.WriteFileAtomically(path, data, 0o644)
}

func (s *Store) Load(path string) (lockpkg.File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var items lockpkg.File
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return s.normalize(items), nil
}

// normalize 就地归一化记录中的 agent 名称（幂等）。
func (s *Store) normalize(items lockpkg.File) lockpkg.File {
	if s.agentResolver == nil || len(items) == 0 {
		return items
	}
	for key, records := range items {
		for i := range records {
			records[i].Agents = normalizeAgentNames(records[i].Agents, s.agentResolver)
		}
		items[key] = records
	}
	return items
}

// normalizeAgentNames 归一化并去重排序，保证同一组 agent 的存储结果稳定。
func normalizeAgentNames(agents []string, resolver func(string) string) []string {
	if len(agents) == 0 {
		return agents
	}
	seen := make(map[string]struct{}, len(agents))
	out := make([]string, 0, len(agents))
	for _, name := range agents {
		canonical := resolver(name)
		if canonical == "" {
			continue
		}
		if _, ok := seen[canonical]; ok {
			continue
		}
		seen[canonical] = struct{}{}
		out = append(out, canonical)
	}
	sort.Strings(out)
	return out
}
