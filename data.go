package mofu

import (
	"sync"
)

// DataStore retains ownership of the DataNode registry helpers that do not
// belong in state.go. Path-based subscriptions and the reactive graph live in
// state.go; DataStore remains the legacy flat store.
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
	node, ok := ds.nodes[id]
	ds.mu.RUnlock()
	if ok {
		node.Subscribe("datastore", fn)
	}
}

func (ds *DataStore) Unsubscribe(id string, fn DataCallback) {
	ds.mu.RLock()
	node, ok := ds.nodes[id]
	ds.mu.RUnlock()
	if ok {
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

func (ds *DataStore) ForEach(fn func(id string, node *DataNode)) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	for id, node := range ds.nodes {
		fn(id, node)
	}
}
