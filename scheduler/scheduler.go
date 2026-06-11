// Package scheduler provides a lane-based task scheduler for MOFU.
//
// The scheduler divides work into lanes with different priorities and
// frame budgets, ensuring high-priority work (input, rendering) is
// never starved by background computation.
//
// Lanes:
//
//	Realtime (60fps)  — input handling, state mutations, render
//	Stream (event)    — AI tokens, log lines, system events
//	Compute (batch)   — heavy computation, search, indexing
//	Background (idle) — cleanup, persistence, analytics
package scheduler

import (
	"container/heap"
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// ---------------------------------------------------------------------------
// Lane types
// ---------------------------------------------------------------------------

// Lane identifies a scheduling lane with different priority and budget.
type Lane int

const (
	// LaneRealtime handles input, state mutations, and rendering at 60fps.
	// Budget: 8ms per frame (75% of 16.6ms frame time).
	LaneRealtime Lane = iota

	// LaneStream handles event-driven work: AI tokens, logs, system events.
	// Budget: 4ms per frame.
	LaneStream

	// LaneCompute handles batch computation: search, indexing, analysis.
	// Budget: 2ms per frame, can defer to next frame.
	LaneCompute

	// LaneBackground handles idle work: cleanup, persistence, analytics.
	// Budget: 1ms per frame, only runs when other lanes are idle.
	LaneBackground
)

func (l Lane) String() string {
	switch l {
	case LaneRealtime:
		return "realtime"
	case LaneStream:
		return "stream"
	case LaneCompute:
		return "compute"
	case LaneBackground:
		return "background"
	default:
		return "unknown"
	}
}

// DefaultBudget returns the default frame budget for a lane.
func (l Lane) DefaultBudget() time.Duration {
	switch l {
	case LaneRealtime:
		return 8 * time.Millisecond
	case LaneStream:
		return 4 * time.Millisecond
	case LaneCompute:
		return 2 * time.Millisecond
	case LaneBackground:
		return 1 * time.Millisecond
	default:
		return 1 * time.Millisecond
	}
}

// ---------------------------------------------------------------------------
// Task
// ---------------------------------------------------------------------------

// Task is a unit of work scheduled on a lane.
type Task struct {
	fn       func() // the work to do
	lane     Lane
	priority int       // higher = more urgent
	created  time.Time
	index    int // heap index
}

// taskHeap implements heap.Interface for priority ordering within a lane.
type taskHeap []*Task

func (h taskHeap) Len() int { return len(h) }

func (h taskHeap) Less(i, j int) bool {
	if h[i].priority != h[j].priority {
		return h[i].priority > h[j].priority // higher priority first
	}
	return h[i].created.Before(h[j].created) // FIFO within same priority
}

func (h taskHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *taskHeap) Push(x any) {
	task := x.(*Task)
	task.index = len(*h)
	*h = append(*h, task)
}

func (h *taskHeap) Pop() any {
	old := *h
	n := len(old)
	task := old[n-1]
	old[n-1] = nil
	task.index = -1
	*h = old[:n-1]
	return task
}

// ---------------------------------------------------------------------------
// LaneStats
// ---------------------------------------------------------------------------

// LaneStats tracks per-lane scheduling statistics.
type LaneStats struct {
	Submitted  int64
	Executed   int64
	Deferred   int64
	TotalTime  time.Duration
	MaxTime    time.Duration
	Queued     int
}

// ---------------------------------------------------------------------------
// Scheduler
// ---------------------------------------------------------------------------

// Scheduler is a lane-based task scheduler that ensures high-priority work
// is never starved by background computation.
type Scheduler struct {
	mu     sync.Mutex
	lanes  [4]*taskHeap
	stats  [4]LaneStats
	budget [4]time.Duration

	ctx    context.Context
	cancel context.CancelFunc

	running atomic.Bool
	frame   atomic.Int64

	// Adaptive budget: if a lane uses less than its budget, the remainder
	// is donated to the next lower-priority lane.
	adaptiveBudget bool
}

// New creates a new Scheduler with default lane budgets.
func New() *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Scheduler{
		ctx:            ctx,
		cancel:         cancel,
		adaptiveBudget: true,
	}
	for i := range s.lanes {
		h := &taskHeap{}
		heap.Init(h)
		s.lanes[i] = h
		s.budget[i] = Lane(i).DefaultBudget()
	}
	return s
}

// NewWithBudgets creates a Scheduler with custom lane budgets.
func NewWithBudgets(budgets [4]time.Duration) *Scheduler {
	s := New()
	s.budget = budgets
	return s
}

// Submit adds a task to the given lane. Returns immediately.
func (s *Scheduler) Submit(lane Lane, priority int, fn func()) {
	s.mu.Lock()
	task := &Task{
		fn:       fn,
		lane:     lane,
		priority: priority,
		created:  time.Now(),
	}
	heap.Push(s.lanes[lane], task)
	s.stats[lane].Submitted++
	s.mu.Unlock()
}

// SubmitRealtime is a convenience for LaneRealtime with normal priority.
func (s *Scheduler) SubmitRealtime(fn func()) {
	s.Submit(LaneRealtime, 0, fn)
}

// SubmitStream is a convenience for LaneStream with normal priority.
func (s *Scheduler) SubmitStream(fn func()) {
	s.Submit(LaneStream, 0, fn)
}

// SubmitCompute is a convenience for LaneCompute with normal priority.
func (s *Scheduler) SubmitCompute(fn func()) {
	s.Submit(LaneCompute, 0, fn)
}

// SubmitBackground is a convenience for LaneBackground with normal priority.
func (s *Scheduler) SubmitBackground(fn func()) {
	s.Submit(LaneBackground, 0, fn)
}

// Tick executes one scheduling frame. It processes tasks from highest to
// lowest priority lane, respecting per-lane frame budgets.
//
// Returns the number of tasks executed.
func (s *Scheduler) Tick() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	executed := 0
	remaining := [4]time.Duration{
		s.budget[0], s.budget[1], s.budget[2], s.budget[3],
	}

	// Process lanes in priority order
	for lane := LaneRealtime; lane <= LaneBackground; lane++ {
		laneStart := time.Now()
		for s.lanes[lane].Len() > 0 {
			// Check budget
			elapsed := time.Since(laneStart)
			if elapsed >= remaining[lane] {
				// If adaptive, donate remaining budget to next lane
				if s.adaptiveBudget && lane < LaneBackground {
					leftover := remaining[lane] - elapsed
					if leftover > 0 {
						remaining[lane+1] += leftover
					}
				}
				break
			}

			task := heap.Pop(s.lanes[lane]).(*Task)
			s.mu.Unlock()

			// Execute task
			taskStart := time.Now()
			task.fn()
			taskTime := time.Since(taskStart)

			s.mu.Lock()
			executed++

			// Update stats
			st := &s.stats[lane]
			st.Executed++
			st.TotalTime += taskTime
			if taskTime > st.MaxTime {
				st.MaxTime = taskTime
			}
		}
	}

	s.frame.Add(1)
	return executed
}

// Pending returns the total number of pending tasks across all lanes.
func (s *Scheduler) Pending() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	total := 0
	for _, h := range s.lanes {
		total += h.Len()
	}
	return total
}

// PendingLane returns the number of pending tasks in a specific lane.
func (s *Scheduler) PendingLane(lane Lane) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lanes[lane].Len()
}

// Stats returns a copy of the statistics for all lanes.
func (s *Scheduler) Stats() [4]LaneStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out [4]LaneStats
	copy(out[:], s.stats[:])
	return out
}

// LaneStatsFor returns stats for a specific lane.
func (s *Scheduler) LaneStatsFor(lane Lane) LaneStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stats[lane]
}

// Frame returns the current frame number.
func (s *Scheduler) Frame() int64 {
	return s.frame.Load()
}

// SetAdaptiveBudget enables or adaptive budget donation between lanes.
func (s *Scheduler) SetAdaptiveBudget(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.adaptiveBudget = enabled
}

// SetBudget sets the frame budget for a specific lane.
func (s *Scheduler) SetBudget(lane Lane, d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.budget[lane] = d
}

// Clear removes all pending tasks from all lanes.
func (s *Scheduler) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, h := range s.lanes {
		*h = (*h)[:0]
		heap.Init(h)
	}
}

// ClearLane removes all pending tasks from a specific lane.
func (s *Scheduler) ClearLane(lane Lane) {
	s.mu.Lock()
	defer s.mu.Unlock()
	h := s.lanes[lane]
	*h = (*h)[:0]
	heap.Init(h)
}

// Stop cancels the scheduler context.
func (s *Scheduler) Stop() {
	s.cancel()
}

// ---------------------------------------------------------------------------
// Batch helper
// ---------------------------------------------------------------------------

// Batch submits a slice of tasks to the same lane.
func (s *Scheduler) Batch(lane Lane, priority int, fns ...func()) {
	for _, fn := range fns {
		s.Submit(lane, priority, fn)
	}
}

// ---------------------------------------------------------------------------
// Deferred task — runs in Compute or Background lane
// ---------------------------------------------------------------------------

// DeferCompute schedules work on the compute lane and returns a channel
// that receives the result when the task executes.
func DeferCompute[T any](s *Scheduler, fn func() T) <-chan T {
	ch := make(chan T, 1)
	s.SubmitCompute(func() {
		ch <- fn()
	})
	return ch
}
