package mofu

import (
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Event Propagation — capture + bubble model
// ---------------------------------------------------------------------------

// PropagationPhase identifies the current phase of event propagation.
type PropagationPhase int

const (
	PhaseCapture PropagationPhase = iota // root → target
	PhaseTarget                          // at target
	PhaseBubble                          // target → root
)

// PropagatedEvent wraps an event with propagation metadata.
type PropagatedEvent struct {
	Event
	Target    Node         // the original target node
	Current   Node         // current node being processed
	Phase     PropagationPhase
	stopped   bool
	prevented bool
	mu        sync.Mutex
}

// StopPropagation stops further propagation.
func (pe *PropagatedEvent) StopPropagation() {
	pe.mu.Lock()
	pe.stopped = true
	pe.mu.Unlock()
}

// PreventDefault prevents the default action.
func (pe *PropagatedEvent) PreventDefault() {
	pe.mu.Lock()
	pe.prevented = true
	pe.mu.Unlock()
}

// IsStopped reports whether propagation was stopped.
func (pe *PropagatedEvent) IsStopped() bool {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	return pe.stopped
}

// IsDefaultPrevented reports whether the default action was prevented.
func (pe *PropagatedEvent) IsDefaultPrevented() bool {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	return pe.prevented
}

// ---------------------------------------------------------------------------
// Event Propagator
// ---------------------------------------------------------------------------

// EventPropagator implements the capture + bubble event model.
type EventPropagator struct {
	mu             sync.RWMutex
	captureHandlers map[EventType][]EventHandler
	bubbleHandlers  map[EventType][]EventHandler
	delegates       map[EventType][]EventHandler
}

// NewEventPropagator creates a new event propagator.
func NewEventPropagator() *EventPropagator {
	return &EventPropagator{
		captureHandlers: make(map[EventType][]EventHandler),
		bubbleHandlers:  make(map[EventType][]EventHandler),
		delegates:       make(map[EventType][]EventHandler),
	}
}

// AddCapture registers a capture-phase handler (fires root → target).
func (ep *EventPropagator) AddCapture(eventType EventType, handler EventHandler) {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	ep.captureHandlers[eventType] = append(ep.captureHandlers[eventType], handler)
}

// AddBubble registers a bubble-phase handler (fires target → root).
func (ep *EventPropagator) AddBubble(eventType EventType, handler EventHandler) {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	ep.bubbleHandlers[eventType] = append(ep.bubbleHandlers[eventType], handler)
}

// AddDelegate registers a delegated event handler.
func (ep *EventPropagator) AddDelegate(eventType EventType, handler EventHandler) {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	ep.delegates[eventType] = append(ep.delegates[eventType], handler)
}

// Propagate dispatches an event through the capture → target → bubble phases.
func (ep *EventPropagator) Propagate(event *PropagatedEvent, ancestors []Node) {
	ep.mu.RLock()
	captures := ep.captureHandlers[event.Type]
	bubbles := ep.bubbleHandlers[event.Type]
	delegates := ep.delegates[event.Type]
	ep.mu.RUnlock()

	// Phase 1: Capture (root → target)
	event.Phase = PhaseCapture
	for i := len(ancestors) - 1; i >= 0; i-- {
		if event.IsStopped() {
			return
		}
		event.Current = ancestors[i]
		for _, handler := range captures {
			handler(event.Event)
		}
	}

	// Phase 2: Target
	if !event.IsStopped() {
		event.Phase = PhaseTarget
		event.Current = event.Target
		for _, handler := range delegates {
			handler(event.Event)
		}
	}

	// Phase 3: Bubble (target → root)
	if !event.IsStopped() {
		event.Phase = PhaseBubble
		for i := 0; i < len(ancestors); i++ {
			if event.IsStopped() {
				return
			}
			event.Current = ancestors[i]
			for _, handler := range bubbles {
				handler(event.Event)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Focus Management
// ---------------------------------------------------------------------------

// FocusManager tracks keyboard focus across the node tree.
type FocusManager struct {
	mu       sync.Mutex
	focused  Node
	focusable []Node
	onChange []func(old, new Node)
}

// NewFocusManager creates a new focus manager.
func NewFocusManager() *FocusManager {
	return &FocusManager{}
}

// SetFocus moves focus to the given node.
func (fm *FocusManager) SetFocus(node Node) {
	fm.mu.Lock()
	old := fm.focused
	fm.focused = node
	fm.mu.Unlock()

	if old != node {
		for _, fn := range fm.onChange {
			fn(old, node)
		}
	}
}

// Focused returns the currently focused node.
func (fm *FocusManager) Focused() Node {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	return fm.focused
}

// RegisterFocusable adds a node to the focusable list.
func (fm *FocusManager) RegisterFocusable(node Node) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.focusable = append(fm.focusable, node)
}

// UnregisterFocusable removes a node from the focusable list.
func (fm *FocusManager) UnregisterFocusable(node Node) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	for i, n := range fm.focusable {
		if n == node {
			fm.focusable = append(fm.focusable[:i], fm.focusable[i+1:]...)
			return
		}
	}
}

// FocusNext moves focus to the next focusable node.
func (fm *FocusManager) FocusNext() {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if len(fm.focusable) == 0 {
		return
	}

	current := -1
	for i, n := range fm.focusable {
		if n == fm.focused {
			current = i
			break
		}
	}

	next := (current + 1) % len(fm.focusable)
	fm.focused = fm.focusable[next]
}

// FocusPrev moves focus to the previous focusable node.
func (fm *FocusManager) FocusPrev() {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if len(fm.focusable) == 0 {
		return
	}

	current := -1
	for i, n := range fm.focusable {
		if n == fm.focused {
			current = i
			break
		}
	}

	prev := current - 1
	if prev < 0 {
		prev = len(fm.focusable) - 1
	}
	fm.focused = fm.focusable[prev]
}

// OnFocusChange registers a callback for focus changes.
func (fm *FocusManager) OnFocusChange(fn func(old, new Node)) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.onChange = append(fm.onChange, fn)
}

// ---------------------------------------------------------------------------
// Event Delegation
// ---------------------------------------------------------------------------

// EventDelegate matches events by type and optional filter.
type EventDelegate struct {
	EventType EventType
	Filter    func(event *PropagatedEvent) bool
	Handler   func(event *PropagatedEvent)
	Priority  int
}

// DelegationTree manages event delegation across a node hierarchy.
type DelegationTree struct {
	mu        sync.RWMutex
	delegates []EventDelegate
}

// NewDelegationTree creates a new delegation tree.
func NewDelegationTree() *DelegationTree {
	return &DelegationTree{}
}

// Register adds a delegate.
func (dt *DelegationTree) Register(delegate EventDelegate) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.delegates = append(dt.delegates, delegate)
}

// Dispatch sends an event to matching delegates.
func (dt *DelegationTree) Dispatch(event *PropagatedEvent) {
	dt.mu.RLock()
	delegates := make([]EventDelegate, len(dt.delegates))
	copy(delegates, dt.delegates)
	dt.mu.RUnlock()

	for _, d := range delegates {
		if d.EventType == event.Type {
			if d.Filter == nil || d.Filter(event) {
				d.Handler(event)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Event Throttling
// ---------------------------------------------------------------------------

// ThrottledHandler wraps a handler with throttling.
type ThrottledHandler struct {
	mu       sync.Mutex
	handler  EventHandler
	interval time.Duration
	lastFire time.Time
}

// NewThrottledHandler creates a throttled event handler.
func NewThrottledHandler(handler EventHandler, interval time.Duration) *ThrottledHandler {
	return &ThrottledHandler{
		handler:  handler,
		interval: interval,
	}
}

// Handle fires the handler if enough time has passed.
func (th *ThrottledHandler) Handle(event Event) {
	th.mu.Lock()
	now := time.Now()
	if now.Sub(th.lastFire) < th.interval {
		th.mu.Unlock()
		return
	}
	th.lastFire = now
	th.mu.Unlock()

	th.handler(event)
}

// ---------------------------------------------------------------------------
// Event Batching
// ---------------------------------------------------------------------------

// EventBatch collects events and flushes them in batches.
type EventBatch struct {
	mu       sync.Mutex
	events   []Event
	maxSize  int
	maxWait  time.Duration
	onFlush  func([]Event)
	lastFlush time.Time
}

// NewEventBatch creates a new event batcher.
func NewEventBatch(maxSize int, maxWait time.Duration, onFlush func([]Event)) *EventBatch {
	return &EventBatch{
		maxSize:  maxSize,
		maxWait:  maxWait,
		onFlush:  onFlush,
	}
}

// Add appends an event to the batch.
func (eb *EventBatch) Add(event Event) {
	eb.mu.Lock()
	eb.events = append(eb.events, event)
	shouldFlush := len(eb.events) >= eb.maxSize
	eb.mu.Unlock()

	if shouldFlush {
		eb.Flush()
	}
}

// Flush sends all batched events.
func (eb *EventBatch) Flush() {
	eb.mu.Lock()
	if len(eb.events) == 0 {
		eb.mu.Unlock()
		return
	}
	events := eb.events
	eb.events = nil
	eb.lastFlush = time.Now()
	eb.mu.Unlock()

	if eb.onFlush != nil {
		eb.onFlush(events)
	}
}

// CheckFlush flushes if the max wait time has elapsed.
func (eb *EventBatch) CheckFlush() {
	eb.mu.Lock()
	if len(eb.events) == 0 {
		eb.mu.Unlock()
		return
	}
	elapsed := time.Since(eb.lastFlush)
	eb.mu.Unlock()

	if elapsed >= eb.maxWait {
		eb.Flush()
	}
}
