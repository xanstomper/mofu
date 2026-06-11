package mofu

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// ---------------------------------------------------------------------------
// Data Engine — Better than Bubble Tea's manual state management
// ---------------------------------------------------------------------------
// MOFU's data engine provides:
// - Reactive signals (like SolidJS)
// - Computed values (like Vue's computed)
// - Effects (like React's useEffect)
// - Streaming data (first-class support)
// - Backpressure control
// - State snapshots (for undo/redo)

// Signal is a reactive value that notifies subscribers when changed.
type Signal[T any] struct {
	value    T
	mu       sync.RWMutex
	subs     []func(T)
	id       uint64
	version  uint64
}

// NewSignal creates a new reactive signal.
func NewSignal[T any](initial T) *Signal[T] {
	return &Signal[T]{
		value: initial,
		subs:  make([]func(T), 0),
	}
}

// Get returns the current value.
func (s *Signal[T]) Get() T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.value
}

// Set updates the value and notifies subscribers.
func (s *Signal[T]) Set(value T) {
	s.mu.Lock()
	old := s.value
	s.value = value
	s.version++
	subs := make([]func(T), len(s.subs))
	copy(subs, s.subs)
	s.mu.Unlock()

	// Notify subscribers
	for _, sub := range subs {
		if sub != nil {
			sub(value)
		}
	}
	_ = old // Available for diffing if needed
}

// Subscribe registers a callback for value changes.
func (s *Signal[T]) Subscribe(fn func(T)) func() {
	s.mu.Lock()
	idx := len(s.subs)
	s.subs = append(s.subs, fn)
	s.mu.Unlock()

	return func() {
		s.mu.Lock()
		if idx < len(s.subs) {
			s.subs[idx] = nil // Mark as removed
		}
		s.mu.Unlock()
	}
}

// Version returns the current version (increments on each change).
func (s *Signal[T]) Version() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.version
}

// ---------------------------------------------------------------------------
// Computed — Derived values that auto-recompute
// ---------------------------------------------------------------------------

// Computed is a value derived from signals.
type Computed[T any] struct {
	value    T
	mu       sync.RWMutex
	subs     []func(T)
	compute  func() T
	signals  []any // *Signal[any]
	dirty    bool
	id       uint64
	version  uint64
}

// NewComputed creates a computed value.
func NewComputed[T any](compute func() T, signals ...any) *Computed[T] {
	c := &Computed[T]{
		compute: compute,
		signals: signals,
		dirty:   true,
		value:   compute(), // Initial compute
	}

	// Subscribe to all signals
	for _, sig := range signals {
		switch s := sig.(type) {
		case interface{ Subscribe(func(any)) func() }:
			s.Subscribe(func(any) {
				c.mu.Lock()
				c.dirty = true
				c.mu.Unlock()
				c.recompute()
			})
		}
	}

	return c
}

// Get returns the current computed value.
func (c *Computed[T]) Get() T {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.value
}

// Subscribe registers a callback for value changes.
func (c *Computed[T]) Subscribe(fn func(T)) func() {
	c.mu.Lock()
	idx := len(c.subs)
	c.subs = append(c.subs, fn)
	c.mu.Unlock()

	return func() {
		c.mu.Lock()
		if idx < len(c.subs) {
			c.subs[idx] = nil
		}
		c.mu.Unlock()
	}
}

func (c *Computed[T]) recompute() {
	c.mu.Lock()
	if !c.dirty {
		c.mu.Unlock()
		return
	}
	newValue := c.compute()
	c.value = newValue
	c.dirty = false
	c.version++
	subs := make([]func(T), len(c.subs))
	copy(subs, c.subs)
	c.mu.Unlock()

	for _, sub := range subs {
		sub(newValue)
	}
}

// ---------------------------------------------------------------------------
// EffectRunner — Side effects that run when dependencies change
// ---------------------------------------------------------------------------

// EffectRunner runs a side effect when dependencies change.
type EffectRunner struct {
	fn       func()
	cleanups []func()
	mu       sync.Mutex
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewEffectRunner creates a new effect runner.
func NewEffectRunner(fn func()) *EffectRunner {
	ctx, cancel := context.WithCancel(context.Background())
	return &EffectRunner{
		fn:     fn,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Run runs the effect and tracks dependencies.
func (e *EffectRunner) Run() {
	e.mu.Lock()
	// Run cleanups from previous run
	for _, cleanup := range e.cleanups {
		cleanup()
	}
	e.cleanups = nil
	e.mu.Unlock()

	// Run the effect
	e.fn()
}

// OnCleanup registers a cleanup function.
func (e *EffectRunner) OnCleanup(fn func()) {
	e.mu.Lock()
	e.cleanups = append(e.cleanups, fn)
	e.mu.Unlock()
}

// Stop stops the effect and runs cleanups.
func (e *EffectRunner) Stop() {
	e.cancel()
	e.mu.Lock()
	for _, cleanup := range e.cleanups {
		cleanup()
	}
	e.cleanups = nil
	e.mu.Unlock()
}

// ---------------------------------------------------------------------------
// DataStore — Centralized state management (renamed from Store to avoid conflict)
// ---------------------------------------------------------------------------

// DataStore is a centralized state container with subscriptions.
type DataStore struct {
	state   map[string]any
	mu      sync.RWMutex
	subs    map[string][]func(any)
	version uint64
}

// NewDataStore creates a new store.
func NewDataStore() *DataStore {
	return &DataStore{
		state: make(map[string]any),
		subs:  make(map[string][]func(any)),
	}
}

// Get returns a value from the store.
func (s *DataStore) Get(key string) any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state[key]
}

// Set sets a value in the store and notifies subscribers.
func (s *DataStore) Set(key string, value any) {
	s.mu.Lock()
	s.state[key] = value
	s.version++
	subs := make([]func(any), len(s.subs[key]))
	copy(subs, s.subs[key])
	s.mu.Unlock()

	for _, sub := range subs {
		sub(value)
	}
}

// Subscribe subscribes to changes for a key.
func (s *DataStore) Subscribe(key string, fn func(any)) func() {
	s.mu.Lock()
	idx := len(s.subs[key])
	s.subs[key] = append(s.subs[key], fn)
	s.mu.Unlock()

	return func() {
		s.mu.Lock()
		if idx < len(s.subs[key]) {
			s.subs[key][idx] = nil
		}
		s.mu.Unlock()
	}
}

// Snapshot returns a copy of all state.
func (s *DataStore) Snapshot() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snap := make(map[string]any, len(s.state))
	for k, v := range s.state {
		snap[k] = v
	}
	return snap
}

// Restore hydrates the store from a snapshot.
func (s *DataStore) Restore(snap map[string]any) {
	s.mu.Lock()
	for k, v := range snap {
		s.state[k] = v
	}
	s.mu.Unlock()
}

// Version returns the current version.
func (s *DataStore) Version() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.version
}

// ---------------------------------------------------------------------------
// Stream — Continuous data source
// ---------------------------------------------------------------------------

// Stream is a continuous data source with backpressure.
type Stream[T any] struct {
	ch       chan T
	done     chan struct{}
	backlog  int32
	name     string
	closed   bool
	mu       sync.Mutex
}

// NewStream creates a new stream.
func NewStream[T any](name string, buffer int) *Stream[T] {
	return &Stream[T]{
		ch:   make(chan T, buffer),
		done: make(chan struct{}),
		name: name,
	}
}

// Send sends data to the stream (non-blocking).
func (s *Stream[T]) Send(value T) bool {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return false
	}
	s.mu.Unlock()

	select {
	case s.ch <- value:
		atomic.AddInt32(&s.backlog, 1)
		return true
	default:
		// Drop on backpressure
		return false
	}
}

// Receive receives data from the stream.
func (s *Stream[T]) Receive() (T, bool) {
	select {
	case v, ok := <-s.ch:
		if ok {
			atomic.AddInt32(&s.backlog, -1)
		}
		return v, ok
	case <-s.done:
		var zero T
		return zero, false
	}
}

// Close closes the stream.
func (s *Stream[T]) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.closed {
		s.closed = true
		close(s.done)
		close(s.ch)
	}
}

// Done returns a channel that's closed when the stream is done.
func (s *Stream[T]) Done() <-chan struct{} {
	return s.done
}

// Backlog returns the number of pending items.
func (s *Stream[T]) Backlog() int32 {
	return atomic.LoadInt32(&s.backlog)
}

// Name returns the stream name.
func (s *Stream[T]) Name() string {
	return s.name
}

// ---------------------------------------------------------------------------
// BatchCoalescer — Coalesce rapid updates
// ---------------------------------------------------------------------------

// BatchCoalescer coalesces rapid updates into a single update.
type BatchCoalescer[T any] struct {
	signal   *Signal[T]
	pending  []T
	timer    *time.Timer
	interval time.Duration
	mu       sync.Mutex
}

// NewBatchCoalescer creates a new batch coalescer.
func NewBatchCoalescer[T any](signal *Signal[T], interval time.Duration) *BatchCoalescer[T] {
	return &BatchCoalescer[T]{
		signal:   signal,
		interval: interval,
	}
}

// Add adds a value to the batch.
func (b *BatchCoalescer[T]) Add(value T) {
	b.mu.Lock()
	b.pending = append(b.pending, value)
	if b.timer == nil {
		b.timer = time.AfterFunc(b.interval, b.flush)
	}
	b.mu.Unlock()
}

func (b *BatchCoalescer[T]) flush() {
	b.mu.Lock()
	if len(b.pending) == 0 {
		b.mu.Unlock()
		return
	}

	// Take the last value
	last := b.pending[len(b.pending)-1]
	b.pending = nil
	b.mu.Unlock()

	b.signal.Set(last)
}

// ---------------------------------------------------------------------------
// Undo/Redo — State history
// ---------------------------------------------------------------------------

// History manages state snapshots for undo/redo.
type History[T any] struct {
	undoStack []T
	redoStack []T
	maxSize   int
	mu        sync.Mutex
}

// NewHistory creates a new history manager.
func NewHistory[T any](maxSize int) *History[T] {
	return &History[T]{
		undoStack: make([]T, 0),
		redoStack: make([]T, 0),
		maxSize:   maxSize,
	}
}

// Push pushes a state to the undo stack.
func (h *History[T]) Push(state T) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.undoStack = append(h.undoStack, state)
	if len(h.undoStack) > h.maxSize {
		h.undoStack = h.undoStack[1:]
	}
	h.redoStack = nil // Clear redo on new action
}

// Undo undoes the last action.
func (h *History[T]) Undo() (T, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.undoStack) == 0 {
		var zero T
		return zero, false
	}

	// Pop from undo stack
	state := h.undoStack[len(h.undoStack)-1]
	h.undoStack = h.undoStack[:len(h.undoStack)-1]

	// Push to redo stack
	h.redoStack = append(h.redoStack, state)

	// Return the previous state (if any)
	if len(h.undoStack) > 0 {
		return h.undoStack[len(h.undoStack)-1], true
	}
	var zero T
	return zero, false
}

// Redo redoes the last undone action.
func (h *History[T]) Redo() (T, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.redoStack) == 0 {
		var zero T
		return zero, false
	}

	// Pop from redo stack
	state := h.redoStack[len(h.redoStack)-1]
	h.redoStack = h.redoStack[:len(h.redoStack)-1]

	// Push to undo stack
	h.undoStack = append(h.undoStack, state)

	return state, true
}

// CanUndo returns whether undo is possible.
func (h *History[T]) CanUndo() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.undoStack) > 1
}

// CanRedo returns whether redo is possible.
func (h *History[T]) CanRedo() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.redoStack) > 0
}

// Clear clears all history.
func (h *History[T]) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.undoStack = nil
	h.redoStack = nil
}
