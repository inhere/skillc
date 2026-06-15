package webapp

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"time"
)

type HistoryRecord struct {
	Time    string `json:"time"`
	Action  string `json:"action"`
	Agent   string `json:"agent,omitempty"`
	Scope   string `json:"scope,omitempty"`
	WorkDir string `json:"workdir,omitempty"`
	Status  string `json:"status"`
	Request any    `json:"request,omitempty"`
	Result  any    `json:"result,omitempty"`
	Error   string `json:"error,omitempty"`
}

type historyStore struct {
	path string
	now  func() time.Time
}

func newHistoryStore(path string) *historyStore {
	return &historyStore{path: path, now: time.Now}
}

func (s *historyStore) Append(record HistoryRecord) error {
	if record.Time == "" {
		record.Time = s.now().UTC().Format(time.RFC3339)
	}
	if record.Status == "" {
		record.Status = "ok"
	}
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(s.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(data, '\n'))
	return err
}

func (s *historyStore) List(limit int) ([]HistoryRecord, error) {
	f, err := os.Open(s.path)
	if os.IsNotExist(err) {
		return []HistoryRecord{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	items := make([]HistoryRecord, 0)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var item HistoryRecord
		if err := json.Unmarshal(scanner.Bytes(), &item); err != nil {
			continue
		}
		items = append(items, item)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	slices.Reverse(items)
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}
