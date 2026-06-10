// Package state provides the reactive state graph for MOFU (Modular Orchestrated Flow Utility).
//
// The state graph is a Directed Acyclic Graph (DAG) where nodes represent
// reactive state atoms, computed selectors, or live streams. Changes propagate
// automatically through the dependency graph, marking dependents as dirty.
//
// This eliminates manual update logic and reducer bottlenecks found in
// traditional TUI frameworks like Bubble Tea.
//
// Usage:
//
//	g := state.NewGraph()
//	count := state.NewAtom(0)
//	g.Add(count)
//	doubled := state.NewComputed([]state.StateNode{count}, func(deps []any) any {
//	    return deps[0].(int) * 2
//	})
//	g.Add(doubled)
//	count.SetValue(5) // doubled auto-recomputes to 10
package state

import (
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
	ev := ChangeEvent{ID: n.id, Value: v, Timestamp: time.Now(), Source: "set"}
	n.mu.Unlock()
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
	n.mu.Unlock()
}

func (n *BaseNode) MarkClean() {
	n.mu.Lock()
	n.dirty = false
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

// Graph is the reactive state graph that manages all state nodes.
// It handles dependency tracking, dirty propagation, and snapshotting.
type Graph struct {
	mu    sync.RWMutex
	nodes map[NodeID]StateNode
}

// NewGraph creates an empty state graph.
func NewGraph() *Graph {
	return &Graph{
		nodes: make(map[NodeID]StateNode),
	}
}

// Add registers a state node in the graph.
func (g *Graph) Add(node StateNode) {
	g.mu.Lock()
	g.nodes[node.ID()] = node
	g.mu.Unlock()
}

// Get retrieves a state node by its ID. Returns nil if not found.
func (g *Graph) Get(id NodeID) StateNode {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.nodes[id]
}

// Propagate marks all dependents of the given node as dirty, then recomputes
// any Computed nodes in the dependency chain. This is the core of the reactive system.
func (g *Graph) Propagate(id NodeID) {
	visited := make(map[NodeID]bool)
	g.propagate(id, visited)
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

	if c, ok := node.(*Computed); ok {
		c.Recompute()
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

// CollectDirty returns all nodes that are currently marked dirty.
func (g *Graph) CollectDirty() []StateNode {
	g.mu.RLock()
	defer g.mu.RUnlock()
	var dirty []StateNode
	for _, node := range g.nodes {
		if node.IsDirty() {
			dirty = append(dirty, node)
		}
	}
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
