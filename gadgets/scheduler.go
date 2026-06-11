package gadgets

import (
	"sync"
	"sync/atomic"
	"time"
)

// ---------------------------------------------------------------------------
// Scheduler Lane System
// ---------------------------------------------------------------------------
//
// Split execution into priority lanes:
//   REALTIME  → input + focus + UI interaction
//   STREAM    → AI/log/event ingestion
//   COMPUTE   → derived state updates
//   RENDER    → diff + terminal output
//   BACKGROUND → caching + indexing + cleanup
//
// Rule: No lane may block another.

// Lane identifies a scheduler lane.
type Lane int

const (
	// LaneRealtime handles input and UI interaction.
	LaneRealtime Lane = iota
	// LaneStream handles streaming data (AI, logs, events).
	LaneStream
	// LaneCompute handles state computation.
	LaneCompute
	// LaneRender handles rendering.
	LaneRender
	// LaneBackground handles background tasks.
	LaneBackground
)

// LaneName returns the name of a lane.
func (l Lane) Name() string {
	return [...]string{"realtime", "stream", "compute", "render", "background"}[l]
}

// Task is a unit of work in a lane.
type Task struct {
	ID       uint64
	Lane     Lane
	Priority int
	Fn       func()
	Done     chan struct{}
	Cancel   chan struct{}
}

// Scheduler manages task execution across lanes.
type Scheduler struct {
	mu       sync.Mutex
	queues   map[Lane][]*Task
	workers  map[Lane]int
	running  bool
	nextID   uint64
	stats    SchedulerStats
	metrics  []LaneMetrics
}

// SchedulerStats tracks overall scheduler statistics.
type SchedulerStats struct {
	TasksExecuted int64
	TasksDropped  int64
	AvgLatency    time.Duration
}

// LaneMetrics tracks per-lane statistics.
type LaneMetrics struct {
	Lane        Lane
	QueueDepth  int
	TasksRun    int64
	AvgDuration time.Duration
}

// NewScheduler creates a new scheduler.
func NewScheduler() *Scheduler {
	s := &Scheduler{
		queues:  make(map[Lane][]*Task),
		workers: make(map[Lane]int),
		metrics: make([]LaneMetrics, 5),
	}
	for i := range s.metrics {
		s.metrics[i].Lane = Lane(i)
	}
	return s
}

// Submit adds a task to a lane.
func (s *Scheduler) Submit(lane Lane, priority int, fn func()) *Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := atomic.AddUint64(&s.nextID, 1)
	task := &Task{
		ID:       id,
		Lane:     lane,
		Priority: priority,
		Fn:       fn,
		Done:     make(chan struct{}),
		Cancel:   make(chan struct{}),
	}

	s.queues[lane] = append(s.queues[lane], task)
	s.metrics[lane].QueueDepth = len(s.queues[lane])

	return task
}

// SubmitRealtime submits a task to the realtime lane.
func (s *Scheduler) SubmitRealtime(fn func()) *Task {
	return s.Submit(LaneRealtime, 100, fn)
}

// SubmitStream submits a task to the stream lane.
func (s *Scheduler) SubmitStream(fn func()) *Task {
	return s.Submit(LaneStream, 50, fn)
}

// SubmitCompute submits a task to the compute lane.
func (s *Scheduler) SubmitCompute(fn func()) *Task {
	return s.Submit(LaneCompute, 30, fn)
}

// SubmitRender submits a task to the render lane.
func (s *Scheduler) SubmitRender(fn func()) *Task {
	return s.Submit(LaneRender, 40, fn)
}

// SubmitBackground submits a task to the background lane.
func (s *Scheduler) SubmitBackground(fn func()) *Task {
	return s.Submit(LaneBackground, 10, fn)
}

// Start begins processing tasks in all lanes.
func (s *Scheduler) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	// Start workers for each lane
	for lane := LaneRealtime; lane <= LaneBackground; lane++ {
		go s.processLane(lane)
	}
}

// Stop stops all lane processing.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	s.running = false
	s.mu.Unlock()
}

// QueueDepth returns the number of pending tasks in a lane.
func (s *Scheduler) QueueDepth(lane Lane) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.queues[lane])
}

// Metrics returns metrics for a lane.
func (s *Scheduler) Metrics(lane Lane) LaneMetrics {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.metrics[lane]
}

// Stats returns overall scheduler statistics.
func (s *Scheduler) Stats() SchedulerStats {
	return s.stats
}

func (s *Scheduler) processLane(lane Lane) {
	for {
		s.mu.Lock()
		if !s.running {
			s.mu.Unlock()
			return
		}

		// Get highest priority task
		if len(s.queues[lane]) == 0 {
			s.mu.Unlock()
			time.Sleep(time.Millisecond)
			continue
		}

		// Find highest priority task
		bestIdx := 0
		for i, task := range s.queues[lane] {
			if task.Priority > s.queues[lane][bestIdx].Priority {
				bestIdx = i
			}
		}

		task := s.queues[lane][bestIdx]
		s.queues[lane] = append(s.queues[lane][:bestIdx], s.queues[lane][bestIdx+1:]...)
		s.metrics[lane].QueueDepth = len(s.queues[lane])
		s.mu.Unlock()

		// Execute task
		start := time.Now()
		task.Fn()
		elapsed := time.Since(start)

		atomic.AddInt64(&s.stats.TasksExecuted, 1)
		s.metrics[lane].TasksRun++
		s.metrics[lane].AvgDuration = elapsed

		close(task.Done)
	}
}
