package state

import (
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// NodeID is a unique identifier for a state node in the graph.
type NodeID uint64

var nodeCounter atomic.Uint64

// NewNodeID generates a globally unique node identifier.
func NewNodeID() NodeID {
	return NodeID(nodeCounter.Add(1))
}

// ChangeEvent is emitted when a state node's value changes.
type ChangeEvent struct {
	ID        NodeID
	Value     any
	Timestamp time.Time
	Source    string
}

// StateNode is the interface for all nodes in the reactive state graph.
// Implementations include Atom (primitive), Computed (derived), and Stream (live input).
type StateNode interface {
	ID() NodeID
	Value() any
	SetValue(v any)
	Dependencies() []NodeID
	AddDependent(id NodeID)
	Dependents() []NodeID
	IsDirty() bool
	MarkDirty()
	MarkClean()
}

// BaseNode provides the common implementation for all StateNode types.
type BaseNode struct {
	id       NodeID
	value    any
	dirty    bool
	deps     []NodeID
	depOf    []NodeID
	mu       sync.RWMutex
	onChange []func(ChangeEvent)
	onDirty  func(NodeID) // called when node becomes dirty (graph tracking)
	onClean  func(NodeID) // called when node becomes clean (graph tracking)
}

// NewBaseNode creates a BaseNode with the given initial value.
func NewBaseNode(initial any) BaseNode {
	return BaseNode{
		id:    NewNodeID(),
		value: initial,
	}
}

func (n *BaseNode) ID() NodeID { return n.id }

func (n *BaseNode) Value() any {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.value
}

func (n *BaseNode) SetValue(v any) {
	n.mu.Lock()
	n.value = v
	n.dirty = true
	cb := n.onDirty
	ev := ChangeEvent{ID: n.id, Value: v, Timestamp: time.Now(), Source: "set"}
	n.mu.Unlock()
	if cb != nil {
		cb(n.id)
	}
	n.fireChange(ev)
}

func (n *BaseNode) Dependencies() []NodeID {
	n.mu.RLock()
	defer n.mu.RUnlock()
	out := make([]NodeID, len(n.deps))
	copy(out, n.deps)
	return out
}

func (n *BaseNode) AddDependent(id NodeID) {
	n.mu.Lock()
	n.depOf = append(n.depOf, id)
	n.mu.Unlock()
}

func (n *BaseNode) Dependents() []NodeID {
	n.mu.RLock()
	defer n.mu.RUnlock()
	out := make([]NodeID, len(n.depOf))
	copy(out, n.depOf)
	return out
}

func (n *BaseNode) IsDirty() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.dirty
}

func (n *BaseNode) MarkDirty() {
	n.mu.Lock()
	n.dirty = true
	cb := n.onDirty
	n.mu.Unlock()
	if cb != nil {
		cb(n.id)
	}
}

func (n *BaseNode) MarkClean() {
	n.mu.Lock()
	n.dirty = false
	cb := n.onClean
	n.mu.Unlock()
	if cb != nil {
		cb(n.id)
	}
}

// SetDirtyCallbacks registers callbacks for dirty/clean transitions.
// Called by Graph.Add to maintain O(1) dirty tracking.
func (n *BaseNode) SetDirtyCallbacks(onDirty, onClean func(NodeID)) {
	n.mu.Lock()
	n.onDirty = onDirty
	n.onClean = onClean
	n.mu.Unlock()
}

func (n *BaseNode) OnChange(fn func(ChangeEvent)) {
	n.mu.Lock()
	n.onChange = append(n.onChange, fn)
	n.mu.Unlock()
}

func (n *BaseNode) fireChange(ev ChangeEvent) {
	n.mu.RLock()
	fns := make([]func(ChangeEvent), len(n.onChange))
	copy(fns, n.onChange)
	n.mu.RUnlock()
	for _, fn := range fns {
		fn(ev)
	}
}

// Atom is a primitive state node that holds a single value.
// When its value changes, all dependent Computed nodes are marked dirty.
type Atom struct {
	BaseNode
}

// NewAtom creates an Atom with the given initial value.
func NewAtom(initial any) *Atom {
	return &Atom{BaseNode: NewBaseNode(initial)}
}

// Computed is a derived state node that recomputes its value when dependencies change.
// It reads actual dependency values (not nil) and caches the result until marked dirty.
type Computed struct {
	BaseNode
	compute  func(deps []any) any
	depNodes []StateNode
}

// NewComputed creates a Computed node that depends on the given state nodes.
// It immediately computes its initial value from the dependency values.
func NewComputed(deps []StateNode, fn func(deps []any) any) *Computed {
	c := &Computed{
		BaseNode: NewBaseNode(nil),
		compute:  fn,
		depNodes: make([]StateNode, len(deps)),
	}
	for i, dep := range deps {
		c.depNodes[i] = dep
		c.deps = append(c.deps, dep.ID())
		dep.AddDependent(c.id)
	}
	c.Recompute()
	return c
}

// Recompute reads current values from all dependencies and recomputes the derived value.
func (c *Computed) Recompute() {
	vals := make([]any, len(c.depNodes))
	for i, dep := range c.depNodes {
		vals[i] = dep.Value()
	}
	c.BaseNode.mu.Lock()
	newVal := c.compute(vals)
	c.value = newVal
	c.dirty = false
	c.BaseNode.mu.Unlock()
}

// Stream is a state node backed by an external live input source (stdin, network, file).
// Call Push to update its value from any goroutine.
type Stream struct {
	BaseNode
	source string
}

// NewStream creates a Stream node with the given source identifier.
func NewStream(source string) *Stream {
	return &Stream{
		BaseNode: NewBaseNode(nil),
		source:   source,
	}
}

func (s *Stream) Push(v any) {
	s.SetValue(v)
}

// ---------------------------------------------------------------------------
// Graph — the reactive state graph
// ---------------------------------------------------------------------------

// Graph is the reactive state graph that manages all state nodes.
// It handles dependency tracking, dirty propagation, transactions, and snapshotting.
type Graph struct {
	mu      sync.RWMutex
	nodes   map[NodeID]StateNode
	dirty   map[NodeID]struct{} // O(1) dirty tracking
	visited map[NodeID]bool     // preallocated for Propagate

	// Transaction support
	txnMu     sync.Mutex
	txnActive bool
	txnDirty  map[NodeID]struct{}
	txnDepth  int

	// Snapshot stack for rollback
	snapshots []map[NodeID]any
}

// NewGraph creates an empty state graph.
func NewGraph() *Graph {
	return &Graph{
		nodes:     make(map[NodeID]StateNode),
		dirty:     make(map[NodeID]struct{}),
		visited:   make(map[NodeID]bool),
		txnDirty:  make(map[NodeID]struct{}),
		snapshots: make([]map[NodeID]any, 0, 4),
	}
}

// Add registers a state node in the graph.
func (g *Graph) Add(node StateNode) {
	g.mu.Lock()
	g.nodes[node.ID()] = node
	// Register dirty callback so the graph tracks dirty nodes in O(1)
	if dcs, ok := node.(interface{ SetDirtyCallbacks(func(NodeID), func(NodeID)) }); ok {
		dcs.SetDirtyCallbacks(
			func(id NodeID) {
				g.mu.Lock()
				g.dirty[id] = struct{}{}
				// Also accumulate in active transaction
				g.txnMu.Lock()
				if g.txnActive {
					g.txnDirty[id] = struct{}{}
				}
				g.txnMu.Unlock()
				g.mu.Unlock()
			},
			func(id NodeID) {
				g.mu.Lock()
				delete(g.dirty, id)
				g.mu.Unlock()
			},
		)
	}
	if node.IsDirty() {
		g.dirty[node.ID()] = struct{}{}
	}
	g.mu.Unlock()
}

// Get retrieves a state node by its ID. Returns nil if not found.
func (g *Graph) Get(id NodeID) StateNode {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.nodes[id]
}

// ---------------------------------------------------------------------------
// Propagation
// ---------------------------------------------------------------------------

// Propagate marks all dependents of the given node as dirty, then recomputes
// any Computed nodes in the dependency chain. Uses BFS in topological order.
func (g *Graph) Propagate(id NodeID) {
	// Reset visited map for this propagation pass
	for k := range g.visited {
		delete(g.visited, k)
	}
	g.propagate(id, g.visited)
}

// recomputer is implemented by nodes that need to recompute when dependencies change.
type recomputer interface {
	Recompute()
}

func (g *Graph) propagate(id NodeID, visited map[NodeID]bool) {
	if visited[id] {
		return
	}
	visited[id] = true

	node := g.Get(id)
	if node == nil {
		return
	}

	if r, ok := node.(recomputer); ok {
		r.Recompute()
	}

	node.MarkClean()

	for _, depOf := range node.Dependents() {
		depNode := g.Get(depOf)
		if depNode != nil {
			depNode.MarkDirty()
			g.propagate(depOf, visited)
		}
	}
}

// PropagateAll propagates all currently dirty nodes in topological order.
// This is the preferred method for batch updates — it ensures Computed nodes
// are recomputed after their dependencies.
func (g *Graph) PropagateAll() {
	g.mu.Lock()
	dirtyIDs := make([]NodeID, 0, len(g.dirty))
	for id := range g.dirty {
		dirtyIDs = append(dirtyIDs, id)
	}
	g.mu.Unlock()

	if len(dirtyIDs) == 0 {
		return
	}

	// Topological sort: nodes with fewer dependencies first
	// This ensures atoms propagate before their computed dependents
	sort.Slice(dirtyIDs, func(i, j int) bool {
		ni := g.Get(dirtyIDs[i])
		nj := g.Get(dirtyIDs[j])
		if ni == nil || nj == nil {
			return false
		}
		return len(ni.Dependencies()) < len(nj.Dependencies())
	})

	for k := range g.visited {
		delete(g.visited, k)
	}

	for _, id := range dirtyIDs {
		g.propagate(id, g.visited)
	}
}

// CollectDirty returns all nodes that are currently marked dirty.
// Uses the O(1) dirty set maintained by MarkDirty/MarkClean callbacks.
func (g *Graph) CollectDirty() []StateNode {
	g.mu.RLock()
	dirty := make([]StateNode, 0, len(g.dirty))
	for id := range g.dirty {
		if node, ok := g.nodes[id]; ok {
			dirty = append(dirty, node)
		}
	}
	g.mu.RUnlock()
	return dirty
}

// Snapshot returns a copy of all node values keyed by NodeID.
func (g *Graph) Snapshot() map[NodeID]any {
	g.mu.RLock()
	defer g.mu.RUnlock()
	snap := make(map[NodeID]any)
	for id, node := range g.nodes {
		snap[id] = node.Value()
	}
	return snap
}

// ---------------------------------------------------------------------------
// Transactions — batch multiple mutations, propagate once
// ---------------------------------------------------------------------------

// Transaction represents a batch of state mutations that are propagated
// atomically when committed. Nested transactions are supported — only
// the outermost commit triggers propagation.
type Transaction struct {
	graph *Graph
}

// BeginTransaction starts a new transaction. All state mutations within
// the transaction are batched and propagated together on Commit().
// Supports nesting — only the outermost commit triggers propagation.
func (g *Graph) BeginTransaction() *Transaction {
	g.txnMu.Lock()
	g.txnDepth++
	g.txnActive = true
	g.txnMu.Unlock()
	return &Transaction{graph: g}
}

// Commit ends the transaction and propagates all accumulated dirty nodes.
// For nested transactions, only the outermost commit triggers propagation.
func (t *Transaction) Commit() {
	g := t.graph
	g.txnMu.Lock()
	g.txnDepth--
	if g.txnDepth > 0 {
		// Nested transaction — don't propagate yet
		g.txnMu.Unlock()
		return
	}

	// Collect all dirty nodes accumulated during the transaction
	dirtyIDs := make([]NodeID, 0, len(g.txnDirty))
	for id := range g.txnDirty {
		dirtyIDs = append(dirtyIDs, id)
		delete(g.txnDirty, id)
	}
	g.txnActive = false
	g.txnMu.Unlock()

	if len(dirtyIDs) == 0 {
		return
	}

	// Propagate in topological order
	sort.Slice(dirtyIDs, func(i, j int) bool {
		ni := g.Get(dirtyIDs[i])
		nj := g.Get(dirtyIDs[j])
		if ni == nil || nj == nil {
			return false
		}
		return len(ni.Dependencies()) < len(nj.Dependencies())
	})

	for k := range g.visited {
		delete(g.visited, k)
	}

	for _, id := range dirtyIDs {
		g.propagate(id, g.visited)
	}
}

// Rollback discards the transaction without propagating any changes.
// Note: individual node values are already mutated; this just prevents
// propagation. Use SaveSnapshot/RestoreSnapshot for true rollback.
func (t *Transaction) Rollback() {
	g := t.graph
	g.txnMu.Lock()
	g.txnDepth--
	if g.txnDepth <= 0 {
		// Clear accumulated dirty nodes without propagating
		for id := range g.txnDirty {
			delete(g.txnDirty, id)
		}
		g.txnActive = false
	}
	g.txnMu.Unlock()
}

// ---------------------------------------------------------------------------
// Snapshot / Rollback — save and restore graph state
// ---------------------------------------------------------------------------

// SaveSnapshot captures the current state of all nodes and pushes it
// onto the snapshot stack. Use RestoreSnapshot to pop and restore.
func (g *Graph) SaveSnapshot() {
	snap := g.Snapshot()
	g.mu.Lock()
	g.snapshots = append(g.snapshots, snap)
	g.mu.Unlock()
}

// RestoreSnapshot pops the most recent snapshot and restores all node
// values. Returns false if no snapshots exist.
func (g *Graph) RestoreSnapshot() bool {
	g.mu.Lock()
	if len(g.snapshots) == 0 {
		g.mu.Unlock()
		return false
	}
	snap := g.snapshots[len(g.snapshots)-1]
	g.snapshots = g.snapshots[:len(g.snapshots)-1]
	g.mu.Unlock()

	// Restore values — this will trigger dirty callbacks
	for id, val := range snap {
		node := g.Get(id)
		if node != nil {
			node.SetValue(val)
		}
	}
	return true
}

// SnapshotDepth returns the number of saved snapshots on the stack.
func (g *Graph) SnapshotDepth() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.snapshots)
}

// ---------------------------------------------------------------------------
// Selector — memoized derived state
// ---------------------------------------------------------------------------

// MemoizedSelector is a Computed node that caches its result and only
// recomputes when dependencies change. Unlike plain Computed, it tracks
// whether its value actually changed (shallow equality).
type MemoizedSelector struct {
	*Computed
	lastValue any
	changed   bool
}

// NewMemoizedSelector creates a selector that only marks itself as changed
// when the computed value differs from the previous result (shallow comparison).
func NewMemoizedSelector(deps []StateNode, fn func(deps []any) any) *MemoizedSelector {
	ms := &MemoizedSelector{}
	ms.Computed = NewComputed(deps, func(deps []any) any {
		newVal := fn(deps)
		if shallowEqual(ms.lastValue, newVal) {
			ms.changed = false
			return ms.lastValue
		}
		ms.changed = true
		ms.lastValue = newVal
		return newVal
	})
	ms.lastValue = ms.Computed.Value()
	return ms
}

// Changed reports whether the last recomputation produced a different value.
func (ms *MemoizedSelector) Changed() bool {
	return ms.changed
}

// shallowEqual does a basic equality check for common types.
func shallowEqual(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	// Fast path for common types
	switch av := a.(type) {
	case int:
		if bv, ok := b.(int); ok {
			return av == bv
		}
	case int64:
		if bv, ok := b.(int64); ok {
			return av == bv
		}
	case float64:
		if bv, ok := b.(float64); ok {
			return av == bv
		}
	case string:
		if bv, ok := b.(string); ok {
			return av == bv
		}
	case bool:
		if bv, ok := b.(bool); ok {
			return av == bv
		}
	}
	// Fallback: compare string representations (allocates)
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

// ---------------------------------------------------------------------------
// Dependency introspection
// ---------------------------------------------------------------------------

// DependentsOf returns all node IDs that directly depend on the given node.
func (g *Graph) DependentsOf(id NodeID) []NodeID {
	node := g.Get(id)
	if node == nil {
		return nil
	}
	return node.Dependents()
}

// DependenciesOf returns all node IDs that the given node depends on.
func (g *Graph) DependenciesOf(id NodeID) []NodeID {
	node := g.Get(id)
	if node == nil {
		return nil
	}
	return node.Dependencies()
}

// TopologicalSort returns all node IDs in topological order (dependencies first).
func (g *Graph) TopologicalSort() []NodeID {
	g.mu.RLock()
	defer g.mu.RUnlock()

	visited := make(map[NodeID]bool)
	var order []NodeID

	var visit func(id NodeID)
	visit = func(id NodeID) {
		if visited[id] {
			return
		}
		visited[id] = true
		node, ok := g.nodes[id]
		if !ok {
			return
		}
		for _, dep := range node.Dependencies() {
			visit(dep)
		}
		order = append(order, id)
	}

	for id := range g.nodes {
		visit(id)
	}
	return order
}

// String returns a human-readable representation of the graph for debugging.
func (g *Graph) String() string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	s := fmt.Sprintf("Graph(%d nodes, %d dirty)\n", len(g.nodes), len(g.dirty))
	for id, node := range g.nodes {
		dirty := ""
		if node.IsDirty() {
			dirty = " [DIRTY]"
		}
		s += fmt.Sprintf("  %d: deps=%v depOf=%v%s\n", id, node.Dependencies(), node.Dependents(), dirty)
	}
	return s
}
