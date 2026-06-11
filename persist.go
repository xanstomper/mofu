package mofu

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Persistence (Anthology Ch.16)
// ---------------------------------------------------------------------------

// SerializationFormat identifies supported persistence encodings.
type SerializationFormat uint8

const (
	FormatJSON SerializationFormat = iota
	FormatBinary
	FormatGzipJSON
)

// StateSerializer saves and loads typed state snapshots.
type StateSerializer struct {
	Format      SerializationFormat
	PrettyPrint bool
}

// Save writes state to path.
func (s *StateSerializer) Save(state any, path string) error {
	var data []byte
	var err error
	switch s.Format {
	case FormatBinary:
		data, err = json.Marshal(state)
	case FormatGzipJSON:
		var raw []byte
		raw, err = json.Marshal(state)
		if err != nil {
			return err
		}
		data, err = gzipData(raw)
	default:
		if s.PrettyPrint {
			data, err = json.MarshalIndent(state, "", "  ")
		} else {
			data, err = json.Marshal(state)
		}
	}
	if err != nil {
		return err
	}
	return atomicWrite(path, data)
}

// Load reads state from path.
func (s *StateSerializer) Load(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	switch s.Format {
	case FormatGzipJSON:
		data, err = gunzipData(data)
		if err != nil {
			return err
		}
		fallthrough
	case FormatJSON, FormatBinary:
		return json.Unmarshal(data, out)
	default:
		return json.Unmarshal(data, out)
	}
}

// StateStore persists state snapshots by key.
type StateStore interface {
	Save(key string, value any) error
	Load(key string, out any) error
	List() ([]string, error)
	Delete(key string) error
	Exists(key string) (bool, error)
}

// FileStateStore stores snapshots as files in a directory.
type FileStateStore struct {
	mu         sync.Mutex
	Dir        string
	Serializer StateSerializer
}

// NewFileStateStore returns a filesystem-backed StateStore.
func NewFileStateStore(dir string, format SerializationFormat) (*FileStateStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &FileStateStore{Dir: dir, Serializer: StateSerializer{Format: format}}, nil
}

func (s *FileStateStore) path(key string) string {
	return filepath.Join(s.Dir, safeFileName(key)+".json")
}

func (s *FileStateStore) Save(key string, value any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Serializer.Save(value, s.path(key))
}

func (s *FileStateStore) Load(key string, out any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Serializer.Load(s.path(key), out)
}

func (s *FileStateStore) List() ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if len(name) > 5 && name[len(name)-5:] == ".json" {
			out = append(out, name[:len(name)-5])
		}
	}
	sort.Strings(out)
	return out, nil
}

func (s *FileStateStore) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return os.Remove(s.path(key))
}

func (s *FileStateStore) Exists(key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := os.Stat(s.path(key))
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

// AppStateSnapshot is a versioned serializable state snapshot.
type AppStateSnapshot struct {
	Version int
	Time    time.Time
	Nodes   map[string]any
}

// StateMigrator applies ordered migrations to saved state.
type StateMigrator struct {
	migrations []Migration
	current    int
}

// Migration upgrades state from one version to another.
type Migration interface {
	Version() int
	Migrate(*AppStateSnapshot) error
}

// AddMigration registers a migration.
func (sm *StateMigrator) AddMigration(m Migration) {
	sm.migrations = append(sm.migrations, m)
	sort.Slice(sm.migrations, func(i, j int) bool { return sm.migrations[i].Version() < sm.migrations[j].Version() })
}

// Migrate applies all migrations newer than current.
func (sm *StateMigrator) Migrate(snap *AppStateSnapshot) error {
	for _, m := range sm.migrations {
		if m.Version() <= sm.current || m.Version() <= snap.Version {
			continue
		}
		if err := m.Migrate(snap); err != nil {
			return err
		}
		sm.current = m.Version()
	}
	return nil
}

// SnapshotManager periodically saves snapshots.
type SnapshotManager struct {
	mu       sync.Mutex
	store    StateStore
	interval time.Duration
	stop     chan struct{}
	stopped  chan struct{}
}

// NewSnapshotManager creates a manager that saves every interval.
func NewSnapshotManager(store StateStore, interval time.Duration) *SnapshotManager {
	return &SnapshotManager{
		store:    store,
		interval: interval,
		stop:     make(chan struct{}),
		stopped:  make(chan struct{}),
	}
}

// Start begins periodic saves.
func (sm *SnapshotManager) Start(snap AppStateSnapshot) {
	go func() {
		defer close(sm.stopped)
		ticker := time.NewTicker(sm.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				sm.mu.Lock()
				_ = sm.store.Save("latest", snap)
				sm.mu.Unlock()
			case <-sm.stop:
				return
			}
		}
	}()
}

// Stop stops the periodic saver.
func (sm *SnapshotManager) Stop() {
	close(sm.stop)
	<-sm.stopped
}

// LRUStore caches values with TTL.
type LRUStore struct {
	mu       sync.Mutex
	items    map[string]lruItem
	order    []string
	capacity int
}

type lruItem struct {
	Value any
	At    time.Time
}

// NewLRUStore returns a cache with capacity items.
func NewLRUStore(capacity int) *LRUStore {
	if capacity <= 0 {
		capacity = 128
	}
	return &LRUStore{items: make(map[string]lruItem), capacity: capacity}
}

// Get returns a cached value and refreshes recency.
func (l *LRUStore) Get(key string) (any, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	item, ok := l.items[key]
	if !ok {
		return nil, false
	}
	l.touch(key)
	return item.Value, true
}

// Put stores or refreshes a value.
func (l *LRUStore) Put(key string, value any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, ok := l.items[key]; ok {
		l.items[key] = lruItem{Value: value, At: time.Now()}
		l.touch(key)
		return
	}
	if len(l.items) >= l.capacity {
		oldest := l.order[0]
		delete(l.items, oldest)
		l.order = append(l.order[:0], l.order[1:]...)
	}
	l.items[key] = lruItem{Value: value, At: time.Now()}
	l.order = append(l.order, key)
}

// Delete removes a value.
func (l *LRUStore) Delete(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.items, key)
	for i, k := range l.order {
		if k == key {
			l.order = append(l.order[:i], l.order[i+1:]...)
			return
		}
	}
}

func (l *LRUStore) touch(key string) {
	for i, k := range l.order {
		if k == key {
			l.order = append(l.order[:i], l.order[i+1:]...)
			break
		}
	}
	l.order = append(l.order, key)
}

func atomicWrite(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func gzipData(in []byte) ([]byte, error) {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	if _, err := w.Write(in); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func gunzipData(in []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(in))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

func safeFileName(name string) string {
	var b bytes.Buffer
	for _, r := range name {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			b.WriteByte('_')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
