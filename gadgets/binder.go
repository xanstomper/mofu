package gadgets

import (
	"sync"
)

// ---------------------------------------------------------------------------
// Binder Implementation
// ---------------------------------------------------------------------------

// binder is the default Binder implementation.
type binder struct {
	mu       sync.RWMutex
	state    map[NodeID]any
	streams  map[string][]func(data any)
	events   []func(Event)
}

// NewBinder creates a new binder.
func NewBinder() *binder {
	return &binder{
		state:   make(map[NodeID]any),
		streams: make(map[string][]func(data any)),
	}
}

func (b *binder) Subscribe(node NodeID) {
	// Subscription is handled by the state graph
}

func (b *binder) SubscribeStream(name string) {
	// Stream subscription is handled by the stream engine
}

func (b *binder) Get(node NodeID) any {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.state[node]
}

func (b *binder) Set(node NodeID, value any) {
	b.mu.Lock()
	b.state[node] = value
	b.mu.Unlock()
}

func (b *binder) Emit(event Event) {
	b.mu.RLock()
	events := b.events
	b.mu.RUnlock()

	for _, handler := range events {
		handler(event)
	}
}

func (b *binder) EmitStream(name string, data any) {
	b.mu.RLock()
	handlers := b.streams[name]
	b.mu.RUnlock()

	for _, handler := range handlers {
		handler(data)
	}
}

// OnEvent registers an event handler.
func (b *binder) OnEvent(handler func(Event)) {
	b.mu.Lock()
	b.events = append(b.events, handler)
	b.mu.Unlock()
}

// OnStream registers a stream handler.
func (b *binder) OnStream(name string, handler func(data any)) {
	b.mu.Lock()
	b.streams[name] = append(b.streams[name], handler)
	b.mu.Unlock()
}

// ---------------------------------------------------------------------------
// Stream Engine
// ---------------------------------------------------------------------------

// Stream represents a continuous data source.
type Stream struct {
	Name    string
	Channel chan any
	done    chan struct{}
}

// NewStream creates a new stream.
func NewStream(name string, buffer int) *Stream {
	return &Stream{
		Name:    name,
		Channel: make(chan any, buffer),
		done:    make(chan struct{}),
	}
}

// Send sends data to the stream.
func (s *Stream) Send(data any) {
	select {
	case s.Channel <- data:
	default:
		// Drop if buffer full (backpressure)
	}
}

// Close closes the stream.
func (s *Stream) Close() {
	close(s.Channel)
	close(s.done)
}

// Done returns a channel that's closed when the stream is done.
func (s *Stream) Done() <-chan struct{} {
	return s.done
}

// ---------------------------------------------------------------------------
// Stream Router
// ---------------------------------------------------------------------------

// StreamRouter routes stream events to gadgets.
type StreamRouter struct {
	mu      sync.RWMutex
	streams map[string]*Stream
	subs    map[string][]func(data any)
}

// NewStreamRouter creates a new stream router.
func NewStreamRouter() *StreamRouter {
	return &StreamRouter{
		streams: make(map[string]*Stream),
		subs:    make(map[string][]func(data any)),
	}
}

// Create creates a new named stream.
func (r *StreamRouter) Create(name string, buffer int) *Stream {
	r.mu.Lock()
	defer r.mu.Unlock()

	stream := NewStream(name, buffer)
	r.streams[name] = stream

	// Route stream data to subscribers
	go func() {
		for data := range stream.Channel {
			r.mu.RLock()
			subs := r.subs[name]
			r.mu.RUnlock()

			for _, sub := range subs {
				sub(data)
			}
		}
	}()

	return stream
}

// Subscribe subscribes to a named stream.
func (r *StreamRouter) Subscribe(name string, handler func(data any)) {
	r.mu.Lock()
	r.subs[name] = append(r.subs[name], handler)
	r.mu.Unlock()
}

// Get returns a stream by name.
func (r *StreamRouter) Get(name string) *Stream {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.streams[name]
}

// Close closes all streams.
func (r *StreamRouter) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, stream := range r.streams {
		stream.Close()
	}
}
