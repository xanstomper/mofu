package gadgets

import (
	"fmt"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Gadget Lifecycle — state machine for gadget lifecycle management
// ---------------------------------------------------------------------------

// GadgetState represents the lifecycle state of a gadget.
type GadgetState int

const (
	GadgetDiscovered  GadgetState = iota // just registered, not initialized
	GadgetResolved                       // dependencies resolved
	GadgetMounted                        // initialized and bound
	GadgetActive                         // actively rendering and handling events
	GadgetSuspended                      // paused, not rendering
	GadgetUnmounted                      // disposed, resources freed
	GadgetFailed                         // error state
)

func (s GadgetState) String() string {
	switch s {
	case GadgetDiscovered:
		return "discovered"
	case GadgetResolved:
		return "resolved"
	case GadgetMounted:
		return "mounted"
	case GadgetActive:
		return "active"
	case GadgetSuspended:
		return "suspended"
	case GadgetUnmounted:
		return "unmounted"
	case GadgetFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// ValidGadgetTransition returns true if the transition is allowed.
func ValidGadgetTransition(from, to GadgetState) bool {
	switch from {
	case GadgetDiscovered:
		return to == GadgetResolved || to == GadgetFailed
	case GadgetResolved:
		return to == GadgetMounted || to == GadgetFailed
	case GadgetMounted:
		return to == GadgetActive || to == GadgetSuspended || to == GadgetUnmounted || to == GadgetFailed
	case GadgetActive:
		return to == GadgetSuspended || to == GadgetUnmounted || to == GadgetFailed
	case GadgetSuspended:
		return to == GadgetActive || to == GadgetUnmounted || to == GadgetFailed
	case GadgetFailed:
		return to == GadgetDiscovered || to == GadgetUnmounted
	default:
		return false
	}
}

// ---------------------------------------------------------------------------
// GadgetInstance — managed gadget with lifecycle
// ---------------------------------------------------------------------------

// GadgetInstance wraps a Gadget with lifecycle management, state tracking,
// and hot-reload support.
type GadgetInstance struct {
	mu        sync.Mutex
	gadget    Gadget
	id        string
	state     GadgetState
	history   []GadgetStateTransition
	createdAt time.Time
	mountedAt time.Time
	lastTick  time.Time
	errorMsg  string
	scope     *GadgetScope
}

// GadgetStateTransition records a lifecycle state change.
type GadgetStateTransition struct {
	From      GadgetState
	To        GadgetState
	Timestamp time.Time
	Error     string
}

// NewGadgetInstance creates a new managed gadget instance.
func NewGadgetInstance(gadget Gadget) *GadgetInstance {
	return &GadgetInstance{
		gadget:    gadget,
		id:        gadget.ID(),
		state:     GadgetDiscovered,
		createdAt: time.Now(),
		history:   make([]GadgetStateTransition, 0, 8),
		scope:     NewGadgetScope(gadget.ID()),
	}
}

// ID returns the gadget ID.
func (gi *GadgetInstance) ID() string { return gi.id }

// State returns the current lifecycle state.
func (gi *GadgetInstance) State() GadgetState {
	gi.mu.Lock()
	defer gi.mu.Unlock()
	return gi.state
}

// Gadget returns the underlying gadget.
func (gi *GadgetInstance) Gadget() Gadget { return gi.gadget }

// Scope returns the gadget's isolated state scope.
func (gi *GadgetInstance) Scope() *GadgetScope { return gi.scope }

// Error returns the error message if in failed state.
func (gi *GadgetInstance) Error() string {
	gi.mu.Lock()
	defer gi.mu.Unlock()
	return gi.errorMsg
}

// History returns a copy of the lifecycle transition history.
func (gi *GadgetInstance) History() []GadgetStateTransition {
	gi.mu.Lock()
	defer gi.mu.Unlock()
	out := make([]GadgetStateTransition, len(gi.history))
	copy(out, gi.history)
	return out
}

// transitionTo attempts to move to a new state. Returns false if invalid.
func (gi *GadgetInstance) transitionTo(to GadgetState) bool {
	gi.mu.Lock()
	defer gi.mu.Unlock()

	if !ValidGadgetTransition(gi.state, to) {
		return false
	}

	gi.history = append(gi.history, GadgetStateTransition{
		From:      gi.state,
		To:        to,
		Timestamp: time.Now(),
	})
	gi.state = to
	return true
}

// ---------------------------------------------------------------------------
// GadgetManager — manages gadget lifecycle and hot-reload
// ---------------------------------------------------------------------------

// GadgetManager manages the lifecycle of all gadgets.
type GadgetManager struct {
	mu        sync.Mutex
	gadgets   map[string]*GadgetInstance
	order     []string // insertion order
	onChange  []func(id string, state GadgetState)
	ctx       GadgetContext
	running   bool
}

// NewGadgetManager creates a new gadget manager.
func NewGadgetManager(ctx GadgetContext) *GadgetManager {
	return &GadgetManager{
		gadgets: make(map[string]*GadgetInstance),
		ctx:     ctx,
	}
}

// Register adds a gadget and resolves it.
func (gm *GadgetManager) Register(gadget Gadget) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	id := gadget.ID()
	if _, exists := gm.gadgets[id]; exists {
		return fmt.Errorf("gadget %s already registered", id)
	}

	inst := NewGadgetInstance(gadget)
	gm.gadgets[id] = inst
	gm.order = append(gm.order, id)

	// Resolve
	inst.transitionTo(GadgetResolved)
	gm.fireChange(id, GadgetResolved)

	return nil
}

// Mount initializes and mounts a gadget.
func (gm *GadgetManager) Mount(id string) error {
	gm.mu.Lock()
	inst, ok := gm.gadgets[id]
	gm.mu.Unlock()

	if !ok {
		return fmt.Errorf("gadget %s not found", id)
	}

	// Initialize
	if err := inst.gadget.Init(gm.ctx); err != nil {
		inst.transitionTo(GadgetFailed)
		gi := inst
		gi.mu.Lock()
		gi.errorMsg = err.Error()
		gi.mu.Unlock()
		gm.fireChange(id, GadgetFailed)
		return fmt.Errorf("gadget %s init failed: %w", id, err)
	}

	// Bind
	if gm.ctx.Binder != nil {
		inst.gadget.Bind(gm.ctx.Binder)
	}

	inst.transitionTo(GadgetMounted)
	inst.mu.Lock()
	inst.mountedAt = time.Now()
	inst.mu.Unlock()
	gm.fireChange(id, GadgetMounted)

	// Auto-activate
	inst.transitionTo(GadgetActive)
	gm.fireChange(id, GadgetActive)

	return nil
}

// Suspend pauses a gadget.
func (gm *GadgetManager) Suspend(id string) error {
	gm.mu.Lock()
	inst, ok := gm.gadgets[id]
	gm.mu.Unlock()

	if !ok {
		return fmt.Errorf("gadget %s not found", id)
	}

	if !inst.transitionTo(GadgetSuspended) {
		return fmt.Errorf("gadget %s cannot be suspended from %s", id, inst.State())
	}

	gm.fireChange(id, GadgetSuspended)
	return nil
}

// Resume reactivates a suspended gadget.
func (gm *GadgetManager) Resume(id string) error {
	gm.mu.Lock()
	inst, ok := gm.gadgets[id]
	gm.mu.Unlock()

	if !ok {
		return fmt.Errorf("gadget %s not found", id)
	}

	if !inst.transitionTo(GadgetActive) {
		return fmt.Errorf("gadget %s cannot be resumed from %s", id, inst.State())
	}

	gm.fireChange(id, GadgetActive)
	return nil
}

// Unmount disposes and removes a gadget.
func (gm *GadgetManager) Unmount(id string) error {
	gm.mu.Lock()
	inst, ok := gm.gadgets[id]
	gm.mu.Unlock()

	if !ok {
		return fmt.Errorf("gadget %s not found", id)
	}

	if err := inst.gadget.Dispose(); err != nil {
		gm.mu.Lock()
		inst.errorMsg = err.Error()
		gm.mu.Unlock()
	}

	if !inst.transitionTo(GadgetUnmounted) {
		return fmt.Errorf("gadget %s cannot be unmounted from %s", id, inst.State())
	}

	gm.fireChange(id, GadgetUnmounted)
	return nil
}

// Reload hot-reloads a gadget (unmount then remount).
func (gm *GadgetManager) Reload(id string, newGadget Gadget) error {
	gm.mu.Lock()
	oldInst, ok := gm.gadgets[id]
	gm.mu.Unlock()

	if !ok {
		return fmt.Errorf("gadget %s not found", id)
	}

	// Dispose old
	if oldInst.State() == GadgetActive || oldInst.State() == GadgetMounted {
		oldInst.gadget.Dispose()
		oldInst.transitionTo(GadgetUnmounted)
	}

	// Register new
	gm.mu.Lock()
	inst := NewGadgetInstance(newGadget)
	gm.gadgets[id] = inst
	gm.mu.Unlock()

	inst.transitionTo(GadgetResolved)

	// Mount new
	return gm.Mount(id)
}

// Get returns a gadget instance by ID.
func (gm *GadgetManager) Get(id string) *GadgetInstance {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	return gm.gadgets[id]
}

// Active returns all active gadget instances.
func (gm *GadgetManager) Active() []*GadgetInstance {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	var out []*GadgetInstance
	for _, inst := range gm.gadgets {
		if inst.State() == GadgetActive {
			out = append(out, inst)
		}
	}
	return out
}

// All returns all gadget instances in registration order.
func (gm *GadgetManager) All() []*GadgetInstance {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	out := make([]*GadgetInstance, 0, len(gm.order))
	for _, id := range gm.order {
		out = append(out, gm.gadgets[id])
	}
	return out
}

// OnChange registers a callback for lifecycle state changes.
func (gm *GadgetManager) OnChange(fn func(id string, state GadgetState)) {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	gm.onChange = append(gm.onChange, fn)
}

func (gm *GadgetManager) fireChange(id string, state GadgetState) {
	for _, fn := range gm.onChange {
		fn(id, state)
	}
}

// Tick calls OnTick on all active gadgets.
func (gm *GadgetManager) Tick(dt int64) {
	gm.mu.Lock()
	active := make([]*GadgetInstance, 0)
	for _, inst := range gm.gadgets {
		if inst.State() == GadgetActive {
			active = append(active, inst)
		}
	}
	gm.mu.Unlock()

	for _, inst := range active {
		inst.gadget.OnTick(dt)
		inst.mu.Lock()
		inst.lastTick = time.Now()
		inst.mu.Unlock()
	}
}

// Render calls Render on all active gadgets and returns combined render nodes.
func (gm *GadgetManager) Render(state StateView) []RenderNode {
	gm.mu.Lock()
	active := make([]*GadgetInstance, 0)
	for _, inst := range gm.gadgets {
		if inst.State() == GadgetActive {
			active = append(active, inst)
		}
	}
	gm.mu.Unlock()

	var nodes []RenderNode
	for _, inst := range active {
		nodes = append(nodes, inst.gadget.Render(state)...)
	}
	return nodes
}

// MountAll mounts all registered gadgets.
func (gm *GadgetManager) MountAll() []error {
	gm.mu.Lock()
	ids := make([]string, len(gm.order))
	copy(ids, gm.order)
	gm.mu.Unlock()

	var errs []error
	for _, id := range ids {
		if err := gm.Mount(id); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// UnmountAll unmounts all gadgets.
func (gm *GadgetManager) UnmountAll() {
	gm.mu.Lock()
	ids := make([]string, len(gm.order))
	copy(ids, gm.order)
	gm.mu.Unlock()

	for _, id := range ids {
		gm.Unmount(id)
	}
}

// ---------------------------------------------------------------------------
// GadgetScope — isolated state for a gadget
// ---------------------------------------------------------------------------

// GadgetScope provides isolated state storage for a gadget.
type GadgetScope struct {
	mu       sync.RWMutex
	id       string
	state    map[string]any
	persist  map[string]any
}

// NewGadgetScope creates a new gadget scope.
func NewGadgetScope(id string) *GadgetScope {
	return &GadgetScope{
		id:      id,
		state:   make(map[string]any),
		persist: make(map[string]any),
	}
}

// Get reads a value from the scope.
func (gs *GadgetScope) Get(key string) (any, bool) {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	v, ok := gs.state[key]
	return v, ok
}

// Set writes a value to the scope.
func (gs *GadgetScope) Set(key string, value any) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.state[key] = value
}

// Delete removes a value from the scope.
func (gs *GadgetScope) Delete(key string) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	delete(gs.state, key)
}

// Snapshot returns a copy of the scope state.
func (gs *GadgetScope) Snapshot() map[string]any {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	out := make(map[string]any, len(gs.state))
	for k, v := range gs.state {
		out[k] = v
	}
	return out
}

// Restore restores the scope from a snapshot.
func (gs *GadgetScope) Restore(snap map[string]any) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.state = snap
}

// SetPersistent marks a value for persistence across reloads.
func (gs *GadgetScope) SetPersistent(key string, value any) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.persist[key] = value
}

// GetPersistent reads a persistent value.
func (gs *GadgetScope) GetPersistent(key string) (any, bool) {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	v, ok := gs.persist[key]
	return v, ok
}

// PersistentSnapshot returns a copy of persistent state.
func (gs *GadgetScope) PersistentSnapshot() map[string]any {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	out := make(map[string]any, len(gs.persist))
	for k, v := range gs.persist {
		out[k] = v
	}
	return out
}
