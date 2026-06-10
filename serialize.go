package mofu

import (
	"encoding/json"
	"io"
	"sync"
	"time"
)

type StateSnapshot struct {
	Version int
	Time    time.Time
	Nodes   map[string]any
}

type EventLogEntry struct {
	Time    time.Time
	Type    string
	Data    any
	Version int64
}

func (ds *DataStore) ExportJSON(w io.Writer) error {
	snap := ds.Snapshot()
	ss := StateSnapshot{
		Version: 1,
		Time:    time.Now(),
		Nodes:   snap,
	}
	return json.NewEncoder(w).Encode(ss)
}

func (ds *DataStore) ImportJSON(r io.Reader) error {
	var ss StateSnapshot
	if err := json.NewDecoder(r).Decode(&ss); err != nil {
		return err
	}
	ds.Restore(ss.Nodes)
	return nil
}

type EventLog struct {
	entries []EventLogEntry
	mu      sync.Mutex
}

func NewEventLog() *EventLog {
	return &EventLog{}
}

func (el *EventLog) Append(etype string, data any) {
	el.mu.Lock()
	defer el.mu.Unlock()
	el.entries = append(el.entries, EventLogEntry{
		Time: time.Now(),
		Type: etype,
		Data: data,
	})
}

func (el *EventLog) Replay(store *DataStore) {
	el.mu.Lock()
	entries := make([]EventLogEntry, len(el.entries))
	copy(entries, el.entries)
	el.mu.Unlock()
	for _, entry := range entries {
		if entry.Type == "state" {
			if m, ok := entry.Data.(map[string]any); ok {
				for k, v := range m {
					store.Set(k, v)
				}
			}
		}
	}
}

func (el *EventLog) Export(w io.Writer) error {
	el.mu.Lock()
	defer el.mu.Unlock()
	return json.NewEncoder(w).Encode(el.entries)
}
