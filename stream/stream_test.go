package stream

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestStreamPushSubscribe(t *testing.T) {
	s := NewStream("test", TypeLog)
	var received []Item
	s.Subscribe(func(item Item) bool {
		received = append(received, item)
		return true
	})

	s.Push(Item{Data: "hello"})
	s.Push(Item{Data: "world"})

	if len(received) != 2 {
		t.Fatalf("received %d items, want 2", len(received))
	}
	if received[0].Data != "hello" || received[1].Data != "world" {
		t.Fatalf("wrong items: %v", received)
	}
}

func TestStreamUnsubscribe(t *testing.T) {
	s := NewStream("test", TypeLog)
	count := 0
	id := s.Subscribe(func(item Item) bool {
		count++
		return true
	})
	s.Push(Item{Data: "a"})
	s.Unsubscribe(id)
	s.Push(Item{Data: "b"})

	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
}

func TestStreamSelfUnsubscribe(t *testing.T) {
	s := NewStream("test", TypeLog)
	count := 0
	s.Subscribe(func(item Item) bool {
		count++
		return count < 2 // unsubscribe after 2 items
	})

	s.Push(Item{Data: "a"})
	s.Push(Item{Data: "b"})
	s.Push(Item{Data: "c"})

	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}
}

func TestStreamClose(t *testing.T) {
	s := NewStream("test", TypeLog)
	count := 0
	s.Subscribe(func(item Item) bool {
		count++
		return true
	})

	s.Push(Item{Data: "a"})
	s.Close()
	s.Push(Item{Data: "b"}) // should be dropped

	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
	if !s.Closed() {
		t.Fatal("stream should be closed")
	}
}

func TestStreamLast(t *testing.T) {
	s := NewBufferedStream("test", TypeLog, 3)
	s.Push(Item{Data: "a"})
	s.Push(Item{Data: "b"})
	s.Push(Item{Data: "c"})
	s.Push(Item{Data: "d"}) // should evict "a"

	last := s.Last(2)
	if len(last) != 2 {
		t.Fatalf("Last(2) returned %d items", len(last))
	}
	if last[0].Data != "c" || last[1].Data != "d" {
		t.Fatalf("Last(2) = %v", last)
	}
}

func TestStreamTimestamp(t *testing.T) {
	s := NewStream("test", TypeLog)
	var item Item
	s.Subscribe(func(i Item) bool {
		item = i
		return true
	})

	s.Push(Item{Data: "x"})
	if item.Timestamp.IsZero() {
		t.Fatal("timestamp should be auto-set")
	}
}

func TestStreamConcurrent(t *testing.T) {
	s := NewStream("test", TypeLog)
	var count atomic.Int64
	s.Subscribe(func(item Item) bool {
		count.Add(1)
		return true
	})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Push(Item{Data: i})
		}()
	}
	wg.Wait()

	if count.Load() != 100 {
		t.Fatalf("count = %d, want 100", count.Load())
	}
}

// ---------------------------------------------------------------------------
// Combinator tests
// ---------------------------------------------------------------------------

func TestMerge(t *testing.T) {
	s1 := NewStream("s1", TypeLog)
	s2 := NewStream("s2", TypeToken)
	merged := Merge("merged", s1, s2)

	var received []Item
	merged.Subscribe(func(item Item) bool {
		received = append(received, item)
		return true
	})

	s1.Push(Item{Data: "log1"})
	s2.Push(Item{Data: "token1"})
	s1.Push(Item{Data: "log2"})

	if len(received) != 3 {
		t.Fatalf("merged received %d, want 3", len(received))
	}
}

func TestFilter(t *testing.T) {
	s := NewStream("test", TypeLog)
	filtered := Filter(s, func(item Item) bool {
		return item.Type == TypeLog
	})

	var received []Item
	filtered.Subscribe(func(item Item) bool {
		received = append(received, item)
		return true
	})

	s.Push(Item{Data: "log", Type: TypeLog})
	s.Push(Item{Data: "token", Type: TypeToken})
	s.Push(Item{Data: "log2", Type: TypeLog})

	if len(received) != 2 {
		t.Fatalf("filtered received %d, want 2", len(received))
	}
}

func TestMap(t *testing.T) {
	s := NewStream("test", TypeLog)
	mapped := Map(s, func(item Item) Item {
		item.Data = "mapped:" + item.Data.(string)
		return item
	})

	var received []Item
	mapped.Subscribe(func(item Item) bool {
		received = append(received, item)
		return true
	})

	s.Push(Item{Data: "hello"})

	if len(received) != 1 || received[0].Data != "mapped:hello" {
		t.Fatalf("mapped = %v", received)
	}
}

func TestDebounce(t *testing.T) {
	s := NewStream("test", TypeLog)
	debounced := Debounce(s, 50*time.Millisecond)

	var received []Item
	debounced.Subscribe(func(item Item) bool {
		received = append(received, item)
		return true
	})

	s.Push(Item{Data: "a"})
	s.Push(Item{Data: "b"})
	s.Push(Item{Data: "c"})

	// Should not have received anything yet
	if len(received) != 0 {
		t.Fatalf("received %d before debounce, want 0", len(received))
	}

	time.Sleep(100 * time.Millisecond)

	if len(received) != 1 {
		t.Fatalf("received %d after debounce, want 1", len(received))
	}
	if received[0].Data != "c" {
		t.Fatalf("debounced = %v, want c", received[0].Data)
	}
}

func TestThrottle(t *testing.T) {
	s := NewStream("test", TypeLog)
	throttled := Throttle(s, 50*time.Millisecond)

	var count atomic.Int64
	throttled.Subscribe(func(item Item) bool {
		count.Add(1)
		return true
	})

	for i := 0; i < 10; i++ {
		s.Push(Item{Data: i})
		time.Sleep(5 * time.Millisecond)
	}

	// With 50ms throttle and 5ms between pushes, should get ~2 items
	if count.Load() < 1 || count.Load() > 3 {
		t.Fatalf("throttled count = %d, want 1-3", count.Load())
	}
}

func TestBuffer(t *testing.T) {
	s := NewStream("test", TypeLog)
	buffered := Buffer(s, 3, 1*time.Second)

	var batches int64
	buffered.Subscribe(func(item Item) bool {
		atomic.AddInt64(&batches, 1)
		return true
	})

	s.Push(Item{Data: 1})
	s.Push(Item{Data: 2})
	s.Push(Item{Data: 3}) // should trigger flush

	if atomic.LoadInt64(&batches) != 3 {
		t.Fatalf("batches = %d, want 3", atomic.LoadInt64(&batches))
	}
}

// ---------------------------------------------------------------------------
// Convenience constructor tests
// ---------------------------------------------------------------------------

func TestConvenienceConstructors(t *testing.T) {
	log := NewLogStream("app")
	if log.Name() != "log:app" || log.Type() != TypeLog {
		t.Fatal("NewLogStream wrong")
	}

	tok := NewTokenStream("gpt-4")
	if tok.Name() != "token:gpt-4" || tok.Type() != TypeToken {
		t.Fatal("NewTokenStream wrong")
	}

	inp := NewInputStream()
	if inp.Name() != "input" || inp.Type() != TypeInput {
		t.Fatal("NewInputStream wrong")
	}

	sys := NewSystemStream()
	if sys.Name() != "system" || sys.Type() != TypeSystem {
		t.Fatal("NewSystemStream wrong")
	}

	met := NewMetricStream("cpu")
	if met.Name() != "metric:cpu" || met.Type() != TypeMetric {
		t.Fatal("NewMetricStream wrong")
	}
}

// ---------------------------------------------------------------------------
// State bridge tests
// ---------------------------------------------------------------------------

type mockSG struct {
	values map[string]any
	mu     sync.Mutex
}

func newMockSG() *mockSG {
	return &mockSG{values: make(map[string]any)}
}

func (m *mockSG) Set(path string, val any) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.values[path] = val
	return true
}

func TestToState(t *testing.T) {
	sg := newMockSG()
	s := NewStream("test", TypeLog)
	ToState(sg, "logs", s, 5)

	s.Push(Item{Data: "a"})
	s.Push(Item{Data: "b"})

	sg.mu.Lock()
	val, ok := sg.values["logs"]
	sg.mu.Unlock()

	if !ok {
		t.Fatal("ToState did not set value")
	}
	items, ok := val.([]Item)
	if !ok {
		t.Fatalf("value is not []Item: %T", val)
	}
	if len(items) != 2 {
		t.Fatalf("items len = %d, want 2", len(items))
	}
}

func TestToStateLatest(t *testing.T) {
	sg := newMockSG()
	s := NewStream("test", TypeLog)
	ToStateLatest(sg, "latest", s)

	s.Push(Item{Data: "first"})
	s.Push(Item{Data: "second"})

	sg.mu.Lock()
	val := sg.values["latest"]
	sg.mu.Unlock()

	item, ok := val.(Item)
	if !ok {
		t.Fatalf("value is not Item: %T", val)
	}
	if item.Data != "second" {
		t.Fatalf("latest = %v, want second", item.Data)
	}
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func BenchmarkStreamPush(b *testing.B) {
	s := NewStream("bench", TypeLog)
	s.Subscribe(func(item Item) bool { return true })
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Push(Item{Data: i})
	}
}

func BenchmarkStreamPush10Subscribers(b *testing.B) {
	s := NewStream("bench", TypeLog)
	for i := 0; i < 10; i++ {
		s.Subscribe(func(item Item) bool { return true })
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Push(Item{Data: i})
	}
}

func BenchmarkMerge3(b *testing.B) {
	s1 := NewStream("s1", TypeLog)
	s2 := NewStream("s2", TypeToken)
	s3 := NewStream("s3", TypeSystem)
	merged := Merge("merged", s1, s2, s3)
	merged.Subscribe(func(item Item) bool { return true })
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s1.Push(Item{Data: i})
	}
}
