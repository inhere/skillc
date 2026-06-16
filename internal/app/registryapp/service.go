package registryapp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/inhere/skillc/internal/app/sourceapp"
	cfg "github.com/inhere/skillc/internal/domain/config"
	"github.com/inhere/skillc/internal/domain/registry"
	"github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/configstore"
	"github.com/inhere/skillc/internal/infra/registrystore"
)

type AddReq struct {
	ID    string
	Name  string
	Value string
}

type AddSourceReq struct {
	EntryID string
	ID      string
	Name    string
	Sync    bool
}

type Service struct {
	configFile string
	baseDir    string
	store      *configstore.YAMLStore
	cache      *registrystore.Store
	client     *http.Client
	now        func() time.Time
}

func NewService(configFile string, baseDir string) *Service {
	return &Service{
		configFile: configFile,
		baseDir:    baseDir,
		store:      configstore.NewYAMLStore(),
		cache:      registrystore.NewStore(),
		client:     &http.Client{Timeout: 30 * time.Second},
		now:        time.Now,
	}
}

func (s *Service) List() ([]registry.Registry, error) {
	data, err := s.load()
	if err != nil {
		return nil, err
	}
	items := append([]registry.Registry(nil), data.Registries...)
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}

func (s *Service) Add(req AddReq) (registry.Registry, error) {
	data, err := s.load()
	if err != nil {
		return registry.Registry{}, err
	}
	item, err := registry.New(req.ID, req.Name, req.Value)
	if err != nil {
		return registry.Registry{}, err
	}
	if registryIDExists(data.Registries, item.ID) {
		return registry.Registry{}, fmt.Errorf("registry id already registered: %s", item.ID)
	}
	if item.Type == registry.TypeLocal {
		if err := ensureLocalCatalogFile(item.Path); err != nil {
			return registry.Registry{}, err
		}
	}
	data.Registries = append(data.Registries, item)
	if err := s.save(data); err != nil {
		return registry.Registry{}, err
	}
	return item, nil
}

func (s *Service) Remove(id string) error {
	data, err := s.load()
	if err != nil {
		return err
	}
	out := data.Registries[:0]
	removed := false
	for _, item := range data.Registries {
		if item.ID == id {
			removed = true
			continue
		}
		out = append(out, item)
	}
	if !removed {
		return fmt.Errorf("registry not found: %s", id)
	}
	data.Registries = out
	if err := s.save(data); err != nil {
		return err
	}
	return s.removeCachedEntries(data, id)
}

func (s *Service) Sync(id string) error {
	data, err := s.load()
	if err != nil {
		return err
	}
	idx := -1
	for i, item := range data.Registries {
		if item.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("registry not found: %s", id)
	}

	item := data.Registries[idx]
	entries, err := s.fetchEntries(item)
	if err != nil {
		data.Registries[idx].Status = "error"
		data.Registries[idx].ErrorMessage = err.Error()
		_ = s.save(data)
		return err
	}
	data.Registries[idx].Status = "ready"
	data.Registries[idx].ErrorMessage = ""
	data.Registries[idx].LastSyncAt = s.now().UTC().Format(time.RFC3339)
	if err := s.save(data); err != nil {
		return err
	}
	return s.replaceCachedEntries(data, item.ID, entries)
}

func (s *Service) SyncAll() error {
	items, err := s.List()
	if err != nil {
		return err
	}
	for _, item := range items {
		if err := s.Sync(item.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) Search(keyword string) ([]registry.Entry, error) {
	entries, err := s.loadCachedEntries()
	if err != nil {
		return nil, err
	}
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	if keyword == "" {
		return entries, nil
	}
	var out []registry.Entry
	for _, entry := range entries {
		if entryMatches(entry, keyword) {
			out = append(out, entry)
		}
	}
	return out, nil
}

func (s *Service) Info(entryID string) (registry.Entry, error) {
	entries, err := s.loadCachedEntries()
	if err != nil {
		return registry.Entry{}, err
	}
	matches := matchEntries(entries, entryID)
	switch len(matches) {
	case 0:
		return registry.Entry{}, fmt.Errorf("registry entry not found: %s", entryID)
	case 1:
		return matches[0], nil
	default:
		return registry.Entry{}, fmt.Errorf("ambiguous registry entry: %s", entryID)
	}
}

func (s *Service) AddSource(req AddSourceReq) (source.Source, error) {
	entry, err := s.Info(req.EntryID)
	if err != nil {
		return source.Source{}, err
	}
	sourceType := source.Type(strings.TrimSpace(strings.ToLower(entry.Type)))
	value := entry.URL
	if sourceType == source.TypeLocal {
		value = entry.Path
	}
	if sourceType != source.TypeGit && sourceType != source.TypeLocal {
		return source.Source{}, fmt.Errorf("unsupported registry entry type: %s", entry.Type)
	}
	src, err := sourceapp.NewService(s.configFile, s.baseDir).Add(sourceapp.AddReq{
		Value: value,
		Type:  sourceType,
		ID:    firstNonEmpty(req.ID, entry.ID),
		Name:  firstNonEmpty(req.Name, entry.Name),
		Ref:   entry.Ref,
	})
	if err != nil {
		return source.Source{}, err
	}
	if req.Sync {
		if err := sourceapp.NewService(s.configFile, s.baseDir).Sync(src.ID); err != nil {
			return source.Source{}, err
		}
	}
	return src, nil
}

func (s *Service) fetchEntries(item registry.Registry) ([]registry.Entry, error) {
	switch item.Type {
	case registry.TypeLocal:
		return s.fetchLocalEntries(item)
	case registry.TypeHTTP:
		return s.fetchHTTPEntries(item)
	default:
		return nil, fmt.Errorf("unsupported registry type: %s", item.Type)
	}
}

func (s *Service) fetchLocalEntries(item registry.Registry) ([]registry.Entry, error) {
	data, err := os.ReadFile(item.Path)
	if err != nil {
		return nil, err
	}
	var catalog registry.Catalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, err
	}
	return normalizeEntries(catalog.Sources, item.ID, filepath.Dir(item.Path), false)
}

func (s *Service) fetchHTTPEntries(item registry.Registry) ([]registry.Entry, error) {
	resp, err := s.client.Get(item.URL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("registry http status: %d", resp.StatusCode)
	}
	var catalog registry.Catalog
	if err := json.NewDecoder(resp.Body).Decode(&catalog); err != nil {
		return nil, err
	}
	return normalizeEntries(catalog.Sources, item.ID, "", true)
}

func normalizeEntries(entries []registry.Entry, registryID string, catalogDir string, remote bool) ([]registry.Entry, error) {
	out := make([]registry.Entry, 0, len(entries))
	for _, entry := range entries {
		entry.ID = registry.NormalizeID(entry.ID)
		entry.Type = strings.TrimSpace(strings.ToLower(entry.Type))
		entry.URL = strings.TrimSpace(entry.URL)
		entry.Path = strings.TrimSpace(entry.Path)
		entry.Ref = strings.TrimSpace(entry.Ref)
		entry.RegistryID = registryID
		if entry.Name == "" {
			entry.Name = entry.ID
		}
		if entry.Type == string(source.TypeLocal) {
			if remote && !filepath.IsAbs(entry.Path) {
				return nil, fmt.Errorf("registry entry local path must be absolute: %s", entry.ID)
			}
			if !filepath.IsAbs(entry.Path) {
				entry.Path = filepath.Join(catalogDir, entry.Path)
			}
			entry.Path = filepath.Clean(entry.Path)
		}
		if err := entry.Validate(); err != nil {
			return nil, err
		}
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].RegistryID == out[j].RegistryID {
			return out[i].ID < out[j].ID
		}
		return out[i].RegistryID < out[j].RegistryID
	})
	return out, nil
}

func (s *Service) replaceCachedEntries(data cfg.Config, registryID string, entries []registry.Entry) error {
	current, err := s.cache.Load(registryCachePath(data))
	if err != nil {
		return err
	}
	filtered := make([]registry.Entry, 0, len(current)+len(entries))
	for _, entry := range current {
		if entry.RegistryID != registryID {
			filtered = append(filtered, entry)
		}
	}
	filtered = append(filtered, entries...)
	sortEntries(filtered)
	return s.cache.Save(registryCachePath(data), filtered)
}

func (s *Service) removeCachedEntries(data cfg.Config, registryID string) error {
	current, err := s.cache.Load(registryCachePath(data))
	if err != nil {
		return err
	}
	filtered := current[:0]
	for _, entry := range current {
		if entry.RegistryID != registryID {
			filtered = append(filtered, entry)
		}
	}
	return s.cache.Save(registryCachePath(data), filtered)
}

func (s *Service) loadCachedEntries() ([]registry.Entry, error) {
	data, err := s.load()
	if err != nil {
		return nil, err
	}
	entries, err := s.cache.Load(registryCachePath(data))
	if err != nil {
		return nil, err
	}
	sortEntries(entries)
	return entries, nil
}

func registryCachePath(data cfg.Config) string {
	return filepath.Join(data.RegistryCacheDir, "registry-index.json")
}

func (s *Service) load() (cfg.Config, error) {
	return s.store.Load(s.configFile, s.baseDir)
}

func (s *Service) save(data cfg.Config) error {
	return s.store.Save(s.configFile, data, s.baseDir)
}

func ensureLocalCatalogFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("registry path is a directory: %s", path)
	}
	return nil
}

func registryIDExists(items []registry.Registry, id string) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}

func entryMatches(entry registry.Entry, keyword string) bool {
	fields := []string{entry.ID, entry.Name, entry.Description, entry.Type, entry.URL, entry.Path, entry.RegistryID}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), keyword) {
			return true
		}
	}
	for _, tag := range entry.Tags {
		if strings.Contains(strings.ToLower(tag), keyword) {
			return true
		}
	}
	return false
}

func matchEntries(entries []registry.Entry, selector string) []registry.Entry {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return nil
	}
	if registryID, entryID, ok := strings.Cut(selector, "/"); ok {
		registryID = strings.ToLower(strings.TrimSpace(registryID))
		entryID = strings.ToLower(strings.TrimSpace(entryID))
		var matches []registry.Entry
		for _, entry := range entries {
			if strings.ToLower(entry.RegistryID) == registryID && strings.ToLower(entry.ID) == entryID {
				matches = append(matches, entry)
			}
		}
		return matches
	}

	lower := strings.ToLower(selector)
	var exact []registry.Entry
	for _, entry := range entries {
		if strings.ToLower(entry.ID) == lower {
			exact = append(exact, entry)
		}
	}
	if len(exact) > 0 {
		return exact
	}

	var partial []registry.Entry
	for _, entry := range entries {
		if strings.Contains(strings.ToLower(entry.ID), lower) {
			partial = append(partial, entry)
		}
	}
	return partial
}

func sortEntries(entries []registry.Entry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].RegistryID == entries[j].RegistryID {
			return entries[i].ID < entries[j].ID
		}
		return entries[i].RegistryID < entries[j].RegistryID
	})
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
