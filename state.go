package mofu

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/xanstomper/mofu/message"
)

// State is the lifecycle state of the Program.
type State int

const (
	StateInit State = iota
	StateReady
	StateRunning
	StatePaused
	StateStopping
	StateDone
	StateError
)

func (s State) String() string {
	switch s {
	case StateInit:
		return "Init"
	case StateReady:
		return "Ready"
	case StateRunning:
		return "Running"
	case StatePaused:
		return "Paused"
	case StateStopping:
		return "Stopping"
	case StateDone:
		return "Done"
	case StateError:
		return "Error"
	default:
		return "Unknown"
	}
}

// Transition records a state change.
type Transition struct {
	From State
	To   State
}

func (t Transition) String() string { return t.From.String() + " -> " + t.To.String() }

// Valid returns true if the transition is allowed.
func (t Transition) Valid() bool { return validTransition(t.From, t.To) }

func validTransition(from, to State) bool {
	switch from {
	case StateInit:
		return to == StateReady || to == StateError
	case StateReady:
		return to == StateRunning || to == StateError || to == StateDone
	case StateRunning:
		return to == StatePaused || to == StateStopping || to == StateError
	case StatePaused:
		return to == StateRunning || to == StateStopping || to == StateError
	case StateStopping:
		return to == StateDone || to == StateError
	default:
		return false
	}
}

// ---------------------------------------------------------------------------
// StateMachine
// ---------------------------------------------------------------------------

// StateMachine tracks lifecycle state transitions and notifies listeners on changes.
type StateMachine struct {
	state    State
	history  []Transition
	mu       sync.RWMutex
	onChange []func(from, to State)
}

func newStateMachine(initial State) *StateMachine {
	return &StateMachine{
		state:   initial,
		history: make([]Transition, 0, 16),
	}
}

// State returns the current state.
func (sm *StateMachine) State() State {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state
}

// TransitionTo attempts to change state. It returns whether the transition was accepted.
func (sm *StateMachine) TransitionTo(to State) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !validTransition(sm.state, to) {
		return false
	}

	from := sm.state
	sm.history = append(sm.history, Transition{From: from, To: to})
	sm.state = to

	for _, hook := range sm.onChange {
		hook(from, to)
	}
	return true
}

// OnChange registers a lifecycle hook.
func (sm *StateMachine) OnChange(fn func(from, to State)) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.onChange = append(sm.onChange, fn)
}

// History returns a copy of recorded transitions.
func (sm *StateMachine) History() []Transition {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	out := make([]Transition, len(sm.history))
	copy(out, sm.history)
	return out
}

// ---------------------------------------------------------------------------
// Runtime
// ---------------------------------------------------------------------------

// RuntimeConfig captures the canonical execution configuration for a Program.
type RuntimeConfig struct {
	Source string
	Thread string
	Type   string
}

// Runtime is the canonical execution state for a Program.
type Runtime struct {
	ID           string
	Type         string
	State        string
	Config       RuntimeConfig
	Mounted      bool
	mountDelayed bool
	UpdateHook   func()
	mu           sync.Mutex
}

// NewRuntime builds a Runtime from configuration.
func NewRuntime(id, typ string, cfg RuntimeConfig) *Runtime {
	return &Runtime{
		ID:     id,
		Type:   typ,
		State:  "init",
		Config: cfg,
	}
}

// Update transitions the runtime to a new state.
func (r *Runtime) Update(state string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.State = state
	if r.UpdateHook != nil {
		r.UpdateHook()
	}
}

// Mount marks the runtime as mounted.
func (r *Runtime) Mount() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Mounted = true
	r.State = "mounted"
}

// MountDelay defers mount completion by delaying the mounted state.
func (r *Runtime) markMountDelayed() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.mountDelayed = true
	r.State = "delayed"
}

// DirtyRectangles returns rendered dirty rectangles. This is a no-op placeholder
// in this revision; the renderer exposes its own dirty-region source of truth.
func (r *Runtime) DirtyRectangles() []Rect {
	return nil
}

// RequestRender triggers a render. This is a no-op placeholder; runtime
// callers should use the owning Program's render scheduling.
func (r *Runtime) RequestRender() {}

// ---------------------------------------------------------------------------
// Messaging
// ---------------------------------------------------------------------------

// RoutedEvent carries a message plus routing metadata for tenant dispatch.
type RoutedEvent struct {
	Dest   string
	Msg    Msg
	Source string
}

// MessageRouter dispatches string messages to destinations.
type MessageRouter struct {
	dispatch func(RoutedEvent)
}

// NewMessageRouter constructs a router with an optional initial dispatcher.
func NewMessageRouter(dispatch func(RoutedEvent)) *MessageRouter {
	return &MessageRouter{dispatch: dispatch}
}

// SetDispatch configures the downstream dispatch function.
func (r *MessageRouter) SetDispatch(dispatch func(RoutedEvent)) {
	r.dispatch = dispatch
}

// SendMessage delivers a string message to a destination.
// Destinations follow routing rules used by the Runtime system:
// literal channel name for local broadcast, "to:<addr>" for a single
// recipient, or comma-separated list for fan-out.
func SendMessage(dest, message string) error {
	if globalMessageRouter == nil {
		return nil
	}
	globalMessageRouter.dispatch(RoutedEvent{
		Dest:   dest,
		Msg:    message,
		Source: "SendMessage",
	})
	return nil
}

var globalMessageRouter = NewMessageRouter(nil)

// WithProgramMessageRouter connects SendMessage to a Program-level dispatch.
func WithProgramMessageRouter(p *Program) {
	if p == nil {
		return
	}
	globalMessageRouter.SetDispatch(func(ev RoutedEvent) {
		msg := message.Message{
			Type:    message.TypeCommand,
			Payload: ev.Msg,
			Source:  ev.Source,
		}
		p.kern.Bus.Publish(msg)
	})
}

// ---------------------------------------------------------------------------
// Program state query
// ---------------------------------------------------------------------------

// ProgramState returns the current Program lifecycle state from the active Program instance.
func ProgramState(p *Program) State {
	if p == nil {
		return StateInit
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running.Load() {
		return StateInit
	}
	return StateRunning
}

// ---------------------------------------------------------------------------
// StateGraph — reactive state tree with path-based subscriptions
// ---------------------------------------------------------------------------

// DataCallback is the legacy callback signature kept for backward compatibility.
type DataCallback func(oldVal, newVal any)

// SubscribePattern describes how a path subscription matches DataNode IDs.
type SubscribePattern struct {
	Exact    string
	Prefix   string
	Suffix   string
	Contains string
}

// Matches reports whether id satisfies the pattern.
func (p SubscribePattern) Matches(id string) bool {
	switch {
	case p.Exact != "" && id == p.Exact:
		return true
	case p.Prefix != "" && (id == p.Prefix || strings.HasPrefix(id, p.Prefix+"/")):
		return true
	case p.Suffix != "" && strings.HasSuffix(id, p.Suffix):
		return true
	case p.Contains != "" && strings.Contains(id, p.Contains):
		return true
	default:
		return false
	}
}

func (p SubscribePattern) String() string {
	switch {
	case p.Exact != "":
		return p.Exact
	case p.Prefix != "":
		return p.Prefix + ".*"
	case p.Suffix != "":
		return "*" + p.Suffix
	case p.Contains != "":
		return "*" + p.Contains + "*"
	default:
		return "*"
	}
}

// DataNode is a typed state leaf in the graph.
type DataNode struct {
	ID        string
	Value     any
	Source    string
	Version   int64
	Updated   time.Time
	mu        sync.RWMutex
	listeners map[string][]DataCallback
	patterns  map[uint64]SubscribePattern
}

// NewDataNode constructs a state leaf.
func NewDataNode(id string, val any) *DataNode {
	return &DataNode{
		ID:        id,
		Value:     val,
		Source:    "local",
		Version:   1,
		Updated:   time.Now(),
		listeners: make(map[string][]DataCallback),
		patterns:  make(map[uint64]SubscribePattern),
	}
}

// Get returns the current value under the node.
func (dn *DataNode) Get() any {
	dn.mu.RLock()
	defer dn.mu.RUnlock()
	return dn.Value
}

// Set updates the node value and fans out to exact subscribers.
func (dn *DataNode) Set(val any) {
	dn.mu.Lock()
	old := dn.Value
	dn.Value = val
	dn.Version++
	dn.Updated = time.Now()

	listeners := make(map[string][]DataCallback, len(dn.listeners))
	for owner, cbs := range dn.listeners {
		clist := make([]DataCallback, len(cbs))
		copy(clist, cbs)
		listeners[owner] = clist
	}
	dn.mu.Unlock()

	for _, cbs := range listeners {
		for _, cb := range cbs {
			if cb == nil {
				continue
			}
			cb(old, val)
		}
	}
}

// Subscribe registers a callback keyed by owner id.
func (dn *DataNode) Subscribe(id string, fn DataCallback) {
	dn.mu.Lock()
	defer dn.mu.Unlock()
	dn.listeners[id] = append(dn.listeners[id], fn)
}

// SubscribePattern records a wildcard subscription id for later notification.
func (dn *DataNode) SubscribePattern(id uint64, pattern SubscribePattern) {
	dn.mu.Lock()
	defer dn.mu.Unlock()
	dn.patterns[id] = pattern
}

// Unsubscribe removes an exact listener.
func (dn *DataNode) Unsubscribe(id string, fn DataCallback) {
	dn.mu.Lock()
	defer dn.mu.Unlock()
	if fn == nil {
		delete(dn.listeners, id)
		return
	}
	cbs := dn.listeners[id]
	for i, cb := range cbs {
		if fmt.Sprintf("%p", cb) == fmt.Sprintf("%p", fn) {
			dn.listeners[id] = append(cbs[:i], cbs[i+1:]...)
			return
		}
	}
}

// UnsubscribePattern removes a pattern listener.
func (dn *DataNode) UnsubscribePattern(id uint64) {
	dn.mu.Lock()
	defer dn.mu.Unlock()
	delete(dn.patterns, id)
}

// ---------------------------------------------------------------------------
// StateGraph
// ---------------------------------------------------------------------------

// StateChangeListener is called with the path that changed and the new value.
type StateChangeListener func(path string, oldVal, newVal any)

// StateGraph holds the application state as a path-addressed reactive tree.
// It is the core differentiator from other TUI frameworks: widgets only
// redraw when their subscribed paths change.
type StateGraph struct {
	mu       sync.RWMutex
	nodes    map[string]*DataNode
	nextID   uint64
	version  uint64
	onChange map[string][]StateChangeListener
}

// NewStateGraph builds an empty reactive state graph.
func NewStateGraph() *StateGraph {
	return &StateGraph{
		nodes:    make(map[string]*DataNode),
		onChange: make(map[string][]StateChangeListener),
	}
}

// Set writes a value at path, creating the node when missing.
// Bubbles the change to exact and pattern subscribers.
func (sg *StateGraph) Set(path string, val any) bool {
	sg.mu.Lock()
	node, ok := sg.nodes[path]
	if !ok {
		node = NewDataNode(path, val)
		sg.nodes[path] = node
		sg.mu.Unlock()
		sg.version++
		sg.broadcast(path, nil, val)
		return true
	}
	old := node.Get()
	if old == val {
		sg.mu.Unlock()
		return false
	}
	sg.mu.Unlock()
	node.Set(val)
	sg.version++
	sg.broadcast(path, old, val)
	return true
}

// Get reads the value at path.
func (sg *StateGraph) Get(path string) (any, bool) {
	sg.mu.RLock()
	node, ok := sg.nodes[path]
	sg.mu.RUnlock()
	if !ok {
		return nil, false
	}
	return node.Get(), true
}

// GetNode exposes the underlying leaf for advanced callers.
func (sg *StateGraph) GetNode(path string) *DataNode {
	sg.mu.RLock()
	defer sg.mu.RUnlock()
	return sg.nodes[path]
}

// SubscribePath registers fn for changes matching pattern and returns a handle
// that can be passed to Unsubscribe.
func (sg *StateGraph) SubscribePath(pattern SubscribePattern, fn StateChangeListener) uint64 {
	sg.mu.Lock()
	defer sg.mu.Unlock()
	id := sg.nextID
	sg.nextID++
	for path, node := range sg.nodes {
		if pattern.Matches(path) {
			node.SubscribePattern(id, pattern)
		}
	}
	key := pattern.String()
	sg.onChange[key] = append(sg.onChange[key], fn)
	return id
}

// Unsubscribe removes a path listener and its pattern bindings.
func (sg *StateGraph) Unsubscribe(id uint64) {
	sg.mu.Lock()
	defer sg.mu.Unlock()
	for _, node := range sg.nodes {
		node.UnsubscribePattern(id)
	}
}

// Snapshot returns a copy of the entire graph.
func (sg *StateGraph) Snapshot() map[string]any {
	sg.mu.RLock()
	defer sg.mu.RUnlock()
	out := make(map[string]any, len(sg.nodes))
	for k, v := range sg.nodes {
		out[k] = v.Get()
	}
	return out
}

// Restore hydrates the graph from a snapshot, broadcasting changed paths.
func (sg *StateGraph) Restore(snap map[string]any) {
	for k, v := range snap {
		sg.Set(k, v)
	}
}

// Version returns the monotonically increasing graph version.
func (sg *StateGraph) Version() uint64 {
	sg.mu.RLock()
	defer sg.mu.RUnlock()
	return sg.version
}

// DirtyPaths returns paths that changed since sinceVersion, or nil when unchanged.
func (sg *StateGraph) DirtyPaths(sinceVersion uint64) []string {
	sg.mu.RLock()
	defer sg.mu.RUnlock()
	if sg.version <= sinceVersion {
		return nil
	}
	var dirty []string
	for path, node := range sg.nodes {
		if node.Version > 1 {
			dirty = append(dirty, path)
		}
	}
	return dirty
}

// broadcast fans out path changes to exact and prefix subscribers.
func (sg *StateGraph) broadcast(path string, oldVal, newVal any) {
	sg.mu.RLock()
	exact := sg.onChange[path]
	prefix := sg.onChange[SubscribePattern{Prefix: path}.String()]
	listeners := make([]StateChangeListener, 0, len(exact)+len(prefix))
	listeners = append(listeners, exact...)
	listeners = append(listeners, prefix...)
	sg.mu.RUnlock()

	for _, fn := range listeners {
		fn(path, oldVal, newVal)
	}
}
