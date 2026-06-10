package mofu

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type DataCallback func(oldVal, newVal any)

type DataNode struct {
	ID        string
	Value     any
	Source    string
	Version   int64
	Updated   time.Time
	mu        sync.RWMutex
	listeners map[string][]DataCallback
}

func NewDataNode(id string, val any) *DataNode {
	return &DataNode{
		ID:        id,
		Value:     val,
		Source:    "local",
		Version:   1,
		Updated:   time.Now(),
		listeners: make(map[string][]DataCallback),
	}
}

func (dn *DataNode) Get() any {
	dn.mu.RLock()
	defer dn.mu.RUnlock()
	return dn.Value
}

func (dn *DataNode) Set(val any) {
	dn.mu.Lock()
	old := dn.Value
	dn.Value = val
	dn.Version++
	dn.Updated = time.Now()
	listeners := make(map[string][]DataCallback)
	for k, v := range dn.listeners {
		clist := make([]DataCallback, len(v))
		copy(clist, v)
		listeners[k] = clist
	}
	dn.mu.Unlock()
	for _, cbs := range listeners {
		for _, cb := range cbs {
			if cb != nil {
				cb(old, val)
			}
		}
	}
}

func (dn *DataNode) Subscribe(id string, fn DataCallback) {
	dn.mu.Lock()
	defer dn.mu.Unlock()
	dn.listeners[id] = append(dn.listeners[id], fn)
}

func (dn *DataNode) Unsubscribe(id string, fn DataCallback) {
	dn.mu.Lock()
	defer dn.mu.Unlock()
	cbs := dn.listeners[id]
	for i, cb := range cbs {
		if fmt.Sprintf("%p", cb) == fmt.Sprintf("%p", fn) {
			dn.listeners[id] = append(cbs[:i], cbs[i+1:]...)
			return
		}
	}
}

type DataSource interface {
	Read() (any, error)
	Watch(ctx context.Context, onData func(any)) error
	Close() error
}

type DataStore struct {
	nodes map[string]*DataNode
	mu    sync.RWMutex
}

func NewDataStore() *DataStore {
	return &DataStore{
		nodes: make(map[string]*DataNode),
	}
}

func (ds *DataStore) Get(id string) *DataNode {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.nodes[id]
}

func (ds *DataStore) Set(id string, val any) *DataNode {
	ds.mu.Lock()
	node, ok := ds.nodes[id]
	if !ok {
		node = NewDataNode(id, val)
		ds.nodes[id] = node
		ds.mu.Unlock()
		return node
	}
	ds.mu.Unlock()
	node.Set(val)
	return node
}

func (ds *DataStore) Delete(id string) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	delete(ds.nodes, id)
}

func (ds *DataStore) Subscribe(id string, fn DataCallback) {
	ds.mu.RLock()
	node := ds.nodes[id]
	ds.mu.RUnlock()
	if node != nil {
		node.Subscribe("datastore", fn)
	}
}

func (ds *DataStore) Unsubscribe(id string, fn DataCallback) {
	ds.mu.RLock()
	node := ds.nodes[id]
	ds.mu.RUnlock()
	if node != nil {
		node.Unsubscribe("datastore", fn)
	}
}

func (ds *DataStore) Snapshot() map[string]any {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	snap := make(map[string]any, len(ds.nodes))
	for id, node := range ds.nodes {
		snap[id] = node.Get()
	}
	return snap
}

func (ds *DataStore) Restore(snap map[string]any) {
	for id, val := range snap {
		ds.Set(id, val)
	}
}
