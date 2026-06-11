// Package stream provides a unified streaming engine for MOFU.
//
// Streams unify logs, AI tokens, user input, and system events into a
// single composable abstraction with backpressure control, buffering,
// and reactive state graph integration.
//
// Usage:
//
//	s := stream.NewStream("ai-tokens", stream.TypeToken)
//	s.Push(stream.Item{Data: "hello", Source: "gpt-4"})
//
//	merged := stream.Merge(s1, s2, s3)
//	filtered := stream.Filter(merged, func(item stream.Item) bool {
//	    return item.Type == stream.TypeLog
//	})
//
//	// Bridge to StateGraph
//	stream.ToState(sg, "logs/tail", filtered, 100)
package stream

import (
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// ItemType classifies stream items.
type ItemType int

const (
	TypeInput  ItemType = iota // User keyboard/mouse input
	TypeToken                  // AI token stream
	TypeLog                    // Log line
	TypeSystem                 // System event (resize, signal)
	TypeMetric                 // Metric data point
	TypeCustom                 // Application-defined
)

func (t ItemType) String() string {
	switch t {
	case TypeInput:
		return "input"
	case TypeToken:
		return "token"
	case TypeLog:
		return "log"
	case TypeSystem:
		return "system"
	case TypeMetric:
		return "metric"
	default:
		return "custom"
	}
}

// Item is a single element in a stream.
type Item struct {
	Data      any
	Type      ItemType
	Source    string
	Timestamp time.Time
}

// ---------------------------------------------------------------------------
// Subscriber
// ---------------------------------------------------------------------------

// Subscriber receives stream items. Return false from Handle to unsubscribe.
type Subscriber struct {
	id     uint64
	handle func(Item) bool
}

// ---------------------------------------------------------------------------
// Stream — the core reactive stream
// ---------------------------------------------------------------------------

// Stream is a named, typed, backpressure-aware event stream.
// Items are pushed in and fanned out to subscribers.
type Stream struct {
	mu          sync.RWMutex
	name        string
	itemType    ItemType
	subscribers map[uint64]*Subscriber
	nextID      uint64
	items       []Item       // ring buffer for replay
	bufSize     int
	closed      bool
	onPush      []func(Item) // hooks for combinators
}

// NewStream creates a new stream with the given name and type.
func NewStream(name string, itemType ItemType) *Stream {
	return &Stream{
		name:        name,
		itemType:    itemType,
		subscribers: make(map[uint64]*Subscriber),
		bufSize:     64,
	}
}

// NewBufferedStream creates a stream with a custom ring buffer size.
func NewBufferedStream(name string, itemType ItemType, bufSize int) *Stream {
	return &Stream{
		name:        name,
		itemType:    itemType,
		subscribers: make(map[uint64]*Subscriber),
		bufSize:     bufSize,
	}
}

// Name returns the stream name.
func (s *Stream) Name() string { return s.name }

// Type returns the stream item type.
func (s *Stream) Type() ItemType { return s.itemType }

// Push sends an item to all subscribers. If the stream is closed, the item is dropped.
// Push is safe for concurrent use.
func (s *Stream) Push(item Item) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	if item.Timestamp.IsZero() {
		item.Timestamp = time.Now()
	}
	if item.Source == "" {
		item.Source = s.name
	}

	// Ring buffer
	s.items = append(s.items, item)
	if len(s.items) > s.bufSize {
		s.items = s.items[1:]
	}

	// Copy subscribers to avoid holding lock during callbacks
	subs := make([]*Subscriber, 0, len(s.subscribers))
	for _, sub := range s.subscribers {
		subs = append(subs, sub)
	}

	// Fire combinators
	hooks := make([]func(Item), len(s.onPush))
	copy(hooks, s.onPush)
	s.mu.Unlock()

	// Fire hooks (for combinators)
	for _, hook := range hooks {
		hook(item)
	}

	// Deliver to subscribers
	var dead []uint64
	for _, sub := range subs {
		if !sub.handle(item) {
			dead = append(dead, sub.id)
		}
	}

	// Remove dead subscribers
	if len(dead) > 0 {
		s.mu.Lock()
		for _, id := range dead {
			delete(s.subscribers, id)
		}
		s.mu.Unlock()
	}
}

// Subscribe registers a handler that receives all future items.
// Return false from the handler to unsubscribe.
func (s *Stream) Subscribe(handler func(Item) bool) uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.nextID
	s.nextID++
	s.subscribers[id] = &Subscriber{id: id, handle: handler}
	return id
}

// Unsubscribe removes a subscriber by ID.
func (s *Stream) Unsubscribe(id uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.subscribers, id)
}

// Close marks the stream as closed. No more items can be pushed.
func (s *Stream) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
}

// Closed reports whether the stream is closed.
func (s *Stream) Closed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.closed
}

// Last returns the most recent N items from the ring buffer.
func (s *Stream) Last(n int) []Item {
	s.mu.RLock()
	defer s.mu.RUnlock()
	total := len(s.items)
	if n > total {
		n = total
	}
	out := make([]Item, n)
	copy(out, s.items[total-n:])
	return out
}

// Len returns the current number of buffered items.
func (s *Stream) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}

// SubscriberCount returns the number of active subscribers.
func (s *Stream) SubscriberCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.subscribers)
}

// ---------------------------------------------------------------------------
// Combinators
// ---------------------------------------------------------------------------

// Merge combines multiple streams into one. Items from all input streams
// are forwarded to the output stream in arrival order.
func Merge(name string, streams ...*Stream) *Stream {
	out := NewStream(name, TypeCustom)
	for _, s := range streams {
		s := s
		s.Subscribe(func(item Item) bool {
			out.Push(item)
			return true
		})
	}
	return out
}

// Filter creates a new stream that only forwards items matching the predicate.
func Filter(input *Stream, predicate func(Item) bool) *Stream {
	out := NewStream(input.name+"/filtered", input.itemType)
	input.Subscribe(func(item Item) bool {
		if predicate(item) {
			out.Push(item)
		}
		return true
	})
	return out
}

// Map creates a new stream that transforms each item.
func Map(input *Stream, transform func(Item) Item) *Stream {
	out := NewStream(input.name+"/mapped", input.itemType)
	input.Subscribe(func(item Item) bool {
		out.Push(transform(item))
		return true
	})
	return out
}

// Buffer collects items and flushes them in batches.
// Flushes when the batch reaches `size` items or `maxWait` elapses.
func Buffer(input *Stream, size int, maxWait time.Duration) *Stream {
	out := NewStream(input.name+"/buffered", input.itemType)

	var mu sync.Mutex
	var batch []Item
	var timer *time.Timer

	flush := func() {
		mu.Lock()
		if len(batch) == 0 {
			mu.Unlock()
			return
		}
		items := batch
		batch = nil
		mu.Unlock()

		for _, item := range items {
			out.Push(item)
		}
	}

	input.Subscribe(func(item Item) bool {
		mu.Lock()
		batch = append(batch, item)
		shouldFlush := len(batch) >= size
		if shouldFlush {
			mu.Unlock()
			flush()
		} else {
			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(maxWait, flush)
			mu.Unlock()
		}
		return true
	})

	return out
}

// Debounce creates a new stream that only emits the last item after
// a quiet period of `d` with no new items.
func Debounce(input *Stream, d time.Duration) *Stream {
	out := NewStream(input.name+"/debounced", input.itemType)

	var mu sync.Mutex
	var timer *time.Timer
	var lastItem *Item

	input.Subscribe(func(item Item) bool {
		mu.Lock()
		lastItem = &item
		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(d, func() {
			mu.Lock()
			if lastItem != nil {
				item := *lastItem
				lastItem = nil
				mu.Unlock()
				out.Push(item)
			} else {
				mu.Unlock()
			}
		})
		mu.Unlock()
		return true
	})

	return out
}

// Throttle creates a new stream that emits at most one item per duration `d`.
func Throttle(input *Stream, d time.Duration) *Stream {
	out := NewStream(input.name+"/throttled", input.itemType)

	var mu sync.Mutex
	var lastEmit time.Time

	input.Subscribe(func(item Item) bool {
		mu.Lock()
		now := time.Now()
		if now.Sub(lastEmit) >= d {
			lastEmit = now
			mu.Unlock()
			out.Push(item)
		} else {
			mu.Unlock()
		}
		return true
	})

	return out
}

// ---------------------------------------------------------------------------
// Stream-to-StateGraph bridge
// ---------------------------------------------------------------------------

// ToState bridges a stream into a StateGraph path. It maintains a ring buffer
// of the last `bufSize` items at the given path.
func ToState(sg StateGraphAccessor, path string, input *Stream, bufSize int) {
	var mu sync.Mutex
	var buf []Item

	input.Subscribe(func(item Item) bool {
		mu.Lock()
		buf = append(buf, item)
		if len(buf) > bufSize {
			buf = buf[1:]
		}
		snapshot := make([]Item, len(buf))
		copy(snapshot, buf)
		mu.Unlock()

		sg.Set(path, snapshot)
		return true
	})
}

// ToStateLatest bridges a stream to a StateGraph path, storing only the latest item.
func ToStateLatest(sg StateGraphAccessor, path string, input *Stream) {
	input.Subscribe(func(item Item) bool {
		sg.Set(path, item)
		return true
	})
}

// StateGraphAccessor is the interface for StateGraph operations needed by stream bridges.
type StateGraphAccessor interface {
	Set(path string, val any) bool
}

// ---------------------------------------------------------------------------
// Convenience constructors
// ---------------------------------------------------------------------------

// NewLogStream creates a stream for log lines.
func NewLogStream(source string) *Stream {
	return NewStream("log:"+source, TypeLog)
}

// NewTokenStream creates a stream for AI tokens.
func NewTokenStream(source string) *Stream {
	return NewStream("token:"+source, TypeToken)
}

// NewInputStream creates a stream for user input events.
func NewInputStream() *Stream {
	return NewStream("input", TypeInput)
}

// NewSystemStream creates a stream for system events.
func NewSystemStream() *Stream {
	return NewStream("system", TypeSystem)
}

// NewMetricStream creates a stream for metric data points.
func NewMetricStream(name string) *Stream {
	return NewStream("metric:"+name, TypeMetric)
}
