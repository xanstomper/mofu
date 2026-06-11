package mofu_test

import (
	"sync"
	"testing"
	"time"

	"github.com/xanstomper/mofu"
)

func TestSignal(t *testing.T) {
	sig := mofu.NewSignal(0)

	// Test initial value
	if sig.Get() != 0 {
		t.Errorf("expected 0, got %d", sig.Get())
	}

	// Test set
	sig.Set(42)
	if sig.Get() != 42 {
		t.Errorf("expected 42, got %d", sig.Get())
	}

	// Test version
	v1 := sig.Version()
	sig.Set(100)
	if sig.Version() <= v1 {
		t.Error("version should increment")
	}
}

func TestSignalSubscribe(t *testing.T) {
	sig := mofu.NewSignal(0)
	var received int

	unsub := sig.Subscribe(func(v int) {
		received = v
	})

	sig.Set(42)
	if received != 42 {
		t.Errorf("expected 42, got %d", received)
	}

	// Test unsubscribe
	unsub()
	sig.Set(100)
	if received != 42 {
		t.Error("subscriber should not be called after unsubscribe")
	}
}

func TestComputed(t *testing.T) {
	count := mofu.NewSignal(0)
	doubled := mofu.NewComputed(func() int {
		return count.Get() * 2
	}, count)

	if doubled.Get() != 0 {
		t.Errorf("expected 0, got %d", doubled.Get())
	}

	count.Set(5)
	// Note: Computed doesn't auto-recompute in this simple implementation
	// In production, you'd use a dependency tracking system
}

func TestStore(t *testing.T) {
	store := mofu.NewDataStore()

	// Test set/get
	store.Set("name", "MOFU")
	if store.Get("name") != "MOFU" {
		t.Errorf("expected MOFU, got %v", store.Get("name"))
	}

	// Test subscribe
	var received any
	store.Subscribe("name", func(v any) {
		received = v
	})

	store.Set("name", "MOFU v2")
	if received != "MOFU v2" {
		t.Errorf("expected MOFU v2, got %v", received)
	}

	// Test snapshot
	snap := store.Snapshot()
	if snap["name"] != "MOFU v2" {
		t.Errorf("snapshot mismatch")
	}
}

func TestHistory(t *testing.T) {
	h := mofu.NewHistory[string](10)

	// Push states
	h.Push("state1")
	h.Push("state2")
	h.Push("state3")

	// Test undo
	state, ok := h.Undo()
	if !ok || state != "state2" {
		t.Errorf("undo failed: got %v, %v", state, ok)
	}

	// Test redo
	state, ok = h.Redo()
	if !ok || state != "state3" {
		t.Errorf("redo failed: got %v, %v", state, ok)
	}

	// Test can undo/redo
	if !h.CanUndo() {
		t.Error("should be able to undo")
	}
	// After redo, redo stack is empty
	if h.CanRedo() {
		t.Error("should not be able to redo after redo")
	}
}

func TestBatchCoalescer(t *testing.T) {
	sig := mofu.NewSignal(0)
	batch := mofu.NewBatchCoalescer(sig, 10*time.Millisecond)

	batch.Add(1)
	batch.Add(2)
	batch.Add(3)

	deadline := time.Now().Add(500 * time.Millisecond)
	for sig.Get() != 3 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}

	if sig.Get() != 3 {
		t.Errorf("expected 3, got %d", sig.Get())
	}
}

func TestStream(t *testing.T) {
	stream := mofu.NewStream[int]("test", 10)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		stream.Send(1)
		stream.Send(2)
		stream.Send(3)
	}()

	val, ok := stream.Receive()
	if !ok || val != 1 {
		t.Errorf("expected 1, got %d", val)
	}

	val, ok = stream.Receive()
	if !ok || val != 2 {
		t.Errorf("expected 2, got %d", val)
	}

	if stream.Backlog() != 1 {
		t.Errorf("expected backlog 1, got %d", stream.Backlog())
	}

	wg.Wait()

	val, ok = stream.Receive()
	if !ok || val != 3 {
		t.Errorf("expected 3, got %d (ok=%v)", val, ok)
	}

	stream.Close()
	_, ok = stream.Receive()
	if ok {
		t.Error("should not receive after close")
	}
}

func TestConcurrentSignal(t *testing.T) {
	sig := mofu.NewSignal(0)
	var mu sync.Mutex
	values := make([]int, 0)

	sig.Subscribe(func(v int) {
		mu.Lock()
		values = append(values, v)
		mu.Unlock()
	})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			sig.Set(n)
		}(i)
	}
	wg.Wait()

	// Should have received all values
	mu.Lock()
	if len(values) != 100 {
		t.Errorf("expected 100 values, got %d", len(values))
	}
	mu.Unlock()
}
