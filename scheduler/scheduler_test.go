package scheduler

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestSchedulerSubmitAndTick(t *testing.T) {
	s := New()
	defer s.Stop()

	var executed bool
	s.SubmitRealtime(func() { executed = true })

	if s.Pending() != 1 {
		t.Fatalf("pending = %d, want 1", s.Pending())
	}

	n := s.Tick()
	if n != 1 {
		t.Fatalf("tick executed %d, want 1", n)
	}
	if !executed {
		t.Fatal("task not executed")
	}
	if s.Pending() != 0 {
		t.Fatalf("pending after = %d, want 0", s.Pending())
	}
}

func TestSchedulerPriority(t *testing.T) {
	s := New()
	defer s.Stop()

	var order []int
	s.SubmitRealtime(func() { order = append(order, 1) })
	s.SubmitBackground(func() { order = append(order, 2) })
	s.SubmitStream(func() { order = append(order, 3) })

	s.Tick()

	// Realtime should run first, then stream, then background
	if len(order) != 3 {
		t.Fatalf("executed %d tasks, want 3", len(order))
	}
	if order[0] != 1 {
		t.Fatalf("first task = %d, want 1 (realtime)", order[0])
	}
}

func TestSchedulerLaneStats(t *testing.T) {
	s := New()
	defer s.Stop()

	s.SubmitRealtime(func() {})
	s.SubmitRealtime(func() {})
	s.SubmitStream(func() {})

	s.Tick()

	stats := s.Stats()
	if stats[LaneRealtime].Executed != 2 {
		t.Fatalf("realtime executed = %d, want 2", stats[LaneRealtime].Executed)
	}
	if stats[LaneStream].Executed != 1 {
		t.Fatalf("stream executed = %d, want 1", stats[LaneStream].Executed)
	}
}

func TestSchedulerConvenienceMethods(t *testing.T) {
	s := New()
	defer s.Stop()

	var count atomic.Int64
	s.SubmitRealtime(func() { count.Add(1) })
	s.SubmitStream(func() { count.Add(1) })
	s.SubmitCompute(func() { count.Add(1) })
	s.SubmitBackground(func() { count.Add(1) })

	s.Tick()

	if count.Load() != 4 {
		t.Fatalf("count = %d, want 4", count.Load())
	}
}

func TestSchedulerBatch(t *testing.T) {
	s := New()
	defer s.Stop()

	var count atomic.Int64
	s.Batch(LaneRealtime, 0,
		func() { count.Add(1) },
		func() { count.Add(1) },
		func() { count.Add(1) },
	)

	if s.PendingLane(LaneRealtime) != 3 {
		t.Fatalf("pending = %d, want 3", s.PendingLane(LaneRealtime))
	}

	s.Tick()

	if count.Load() != 3 {
		t.Fatalf("count = %d, want 3", count.Load())
	}
}

func TestSchedulerClear(t *testing.T) {
	s := New()
	defer s.Stop()

	s.SubmitRealtime(func() {})
	s.SubmitStream(func() {})
	s.SubmitCompute(func() {})

	s.Clear()

	if s.Pending() != 0 {
		t.Fatalf("pending after clear = %d, want 0", s.Pending())
	}
}

func TestSchedulerClearLane(t *testing.T) {
	s := New()
	defer s.Stop()

	s.SubmitRealtime(func() {})
	s.SubmitCompute(func() {})

	s.ClearLane(LaneCompute)

	if s.Pending() != 1 {
		t.Fatalf("pending = %d, want 1", s.Pending())
	}
	if s.PendingLane(LaneCompute) != 0 {
		t.Fatalf("compute pending = %d, want 0", s.PendingLane(LaneCompute))
	}
}

func TestSchedulerFrameCounter(t *testing.T) {
	s := New()
	defer s.Stop()

	if s.Frame() != 0 {
		t.Fatalf("initial frame = %d, want 0", s.Frame())
	}

	s.SubmitRealtime(func() {})
	s.Tick()

	if s.Frame() != 1 {
		t.Fatalf("frame after tick = %d, want 1", s.Frame())
	}
}

func TestSchedulerBudgetCustom(t *testing.T) {
	s := NewWithBudgets([4]time.Duration{
		10 * time.Millisecond,
		5 * time.Millisecond,
		2 * time.Millisecond,
		1 * time.Millisecond,
	})
	defer s.Stop()

	// Just verify it doesn't panic
	s.SubmitRealtime(func() {})
	s.Tick()
}

func TestSchedulerSetBudget(t *testing.T) {
	s := New()
	defer s.Stop()

	s.SetBudget(LaneRealtime, 16*time.Millisecond)
	s.SetAdaptiveBudget(false)

	s.SubmitRealtime(func() {})
	s.Tick()
}

func TestSchedulerStop(t *testing.T) {
	s := New()
	s.Stop()
	// Should not panic
}

func TestLaneString(t *testing.T) {
	cases := []struct {
		lane Lane
		want string
	}{
		{LaneRealtime, "realtime"},
		{LaneStream, "stream"},
		{LaneCompute, "compute"},
		{LaneBackground, "background"},
		{Lane(99), "unknown"},
	}
	for _, c := range cases {
		if got := c.lane.String(); got != c.want {
			t.Errorf("Lane(%d).String() = %q, want %q", c.lane, got, c.want)
		}
	}
}

func TestSchedulerHighVolume(t *testing.T) {
	s := New()
	defer s.Stop()

	var count atomic.Int64
	for i := 0; i < 1000; i++ {
		s.SubmitCompute(func() { count.Add(1) })
	}

	for s.Pending() > 0 {
		s.Tick()
	}

	if count.Load() != 1000 {
		t.Fatalf("count = %d, want 1000", count.Load())
	}
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func BenchmarkSchedulerSubmit(b *testing.B) {
	s := New()
	defer s.Stop()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.SubmitRealtime(func() {})
	}
}

func BenchmarkSchedulerTick(b *testing.B) {
	s := New()
	defer s.Stop()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.SubmitRealtime(func() {})
		s.Tick()
	}
}

func BenchmarkSchedulerTick100(b *testing.B) {
	s := New()
	defer s.Stop()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 100; j++ {
			s.SubmitCompute(func() {})
		}
		s.Tick()
	}
}
