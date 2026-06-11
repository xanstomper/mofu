package mofu

import (
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// ---------------------------------------------------------------------------
// Performance Engineering (Anthology Ch.18)
// ---------------------------------------------------------------------------

// FrameProfiler tracks frame timing, FPS, dirty ratio, and output size.
type FrameProfiler struct {
	mu            sync.Mutex
	frameTimes    []time.Duration
	maxSamples    int
	dirtyCells    int
	totalCells    int
	escapeBytes   int
	lastFrameTime time.Time
}

// NewFrameProfiler returns a profiler retaining maxSamples frames.
func NewFrameProfiler(maxSamples int) *FrameProfiler {
	if maxSamples <= 0 {
		maxSamples = 240
	}
	return &FrameProfiler{frameTimes: make([]time.Duration, 0, maxSamples), maxSamples: maxSamples}
}

// RecordFrame records a frame duration.
func (p *FrameProfiler) RecordFrame(d time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.frameTimes = append(p.frameTimes, d)
	if len(p.frameTimes) > p.maxSamples {
		p.frameTimes = p.frameTimes[len(p.frameTimes)-p.maxSamples:]
	}
	p.lastFrameTime = time.Now()
}

// FPS returns average FPS over recorded samples.
func (p *FrameProfiler) FPS() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.frameTimes) == 0 {
		return 0
	}
	var total time.Duration
	for _, d := range p.frameTimes {
		total += d
	}
	avg := total / time.Duration(len(p.frameTimes))
	if avg <= 0 {
		return 0
	}
	return 1000 / avg.Seconds()
}

// FrameTimeP95 returns p95 frame duration.
func (p *FrameProfiler) FrameTimeP95() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.frameTimes) == 0 {
		return 0
	}
	cp := append([]time.Duration(nil), p.frameTimes...)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	idx := int(float64(len(cp)-1) * 0.95)
	return cp[idx]
}

// DirtyRatio returns changed cells / total cells.
func (p *FrameProfiler) DirtyRatio(dirty, total int) float64 {
	if total <= 0 {
		return 0
	}
	p.mu.Lock()
	p.dirtyCells = dirty
	p.totalCells = total
	p.mu.Unlock()
	return float64(dirty) / float64(total)
}

// EscapeBytesPerFrame records output byte count.
func (p *FrameProfiler) EscapeBytesPerFrame(bytes int) int {
	p.mu.Lock()
	p.escapeBytes = bytes
	p.mu.Unlock()
	return bytes
}

// Snapshot returns profiler stats.
func (p *FrameProfiler) Snapshot() FrameProfileSnapshot {
	p.mu.Lock()
	defer p.mu.Unlock()
	return FrameProfileSnapshot{
		DirtyCells:  p.dirtyCells,
		TotalCells:  p.totalCells,
		EscapeBytes: p.escapeBytes,
		FPS:         computeFPSLocked(p.frameTimes),
		P95:         percentileDuration(p.frameTimes, 0.95),
	}
}

// FrameProfileSnapshot is a serializable performance snapshot.
type FrameProfileSnapshot struct {
	FPS         float64
	P95         time.Duration
	DirtyCells  int
	TotalCells  int
	EscapeBytes int
}

// ProfileGuard measures scoped durations.
type ProfileGuard struct {
	Name  string
	start time.Time
}

// ProfileScope returns a guard that logs slow scopes.
func ProfileScope(name string) ProfileGuard { return ProfileGuard{Name: name, start: time.Now()} }

// Done finalizes the guard.
func (p ProfileGuard) Done() time.Duration {
	d := time.Since(p.start)
	if d > 100*time.Microsecond {
		// Intentionally side-channel-free: caller may expose metrics.
	}
	return d
}

// MemoryStats wraps runtime.MemStats.
type MemoryStats struct {
	Alloc        uint64
	TotalAlloc   uint64
	Sys          uint64
	NumGC        uint32
	GCPauseTotal time.Duration
	HeapAlloc    uint64
	StackInuse   uint64
}

// ReadMemoryStats returns current process memory stats.
func ReadMemoryStats() MemoryStats {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return MemoryStats{
		Alloc:        ms.Alloc,
		TotalAlloc:   ms.TotalAlloc,
		Sys:          ms.Sys,
		NumGC:        ms.NumGC,
		GCPauseTotal: time.Duration(ms.PauseTotalNs),
		HeapAlloc:    ms.HeapAlloc,
		StackInuse:   ms.StackInuse,
	}
}

// PerfCounter tracks events per second.
type PerfCounter struct {
	count atomic.Int64
	start time.Time
}

// Inc increments the counter.
func (c *PerfCounter) Inc(n int64) { c.count.Add(n) }

// Rate returns events per second since start.
func (c *PerfCounter) Rate() float64 {
	elapsed := time.Since(c.start).Seconds()
	if elapsed <= 0 {
		return 0
	}
	return float64(c.count.Load()) / elapsed
}

// AllocationTracker observes heap allocations.
type AllocationTracker struct {
	start runtime.MemStats
}

// NewAllocationTracker starts tracking allocations.
func NewAllocationTracker() AllocationTracker {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return AllocationTracker{start: ms}
}

// Delta returns allocation delta since construction.
func (a AllocationTracker) Delta() AllocationDelta {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return AllocationDelta{
		Alloc:   ms.Alloc - a.start.Alloc,
		Objects: int64(ms.Mallocs - a.start.Mallocs),
	}
}

// AllocationDelta is the allocation change.
type AllocationDelta struct {
	Alloc   uint64
	Objects int64
}

// FrameBudget tracks time budget usage.
type FrameBudget struct {
	Budget time.Duration
	Used   time.Duration
	Events []FrameBudgetEvent
}

// FrameBudgetEvent records budgeted work.
type FrameBudgetEvent struct {
	Name     string
	Duration time.Duration
}

// Begin records work and returns whether budget remains.
func (b *FrameBudget) Begin(name string, fn func()) {
	start := time.Now()
	fn()
	d := time.Since(start)
	b.Used += d
	b.Events = append(b.Events, FrameBudgetEvent{Name: name, Duration: d})
}

// Remaining returns unused budget.
func (b *FrameBudget) Remaining() time.Duration { return b.Budget - b.Used }

func computeFPSLocked(frames []time.Duration) float64 {
	if len(frames) == 0 {
		return 0
	}
	var total time.Duration
	for _, d := range frames {
		total += d
	}
	avg := total / time.Duration(len(frames))
	if avg <= 0 {
		return 0
	}
	return 1000 / avg.Seconds()
}

func percentileDuration(frames []time.Duration, p float64) time.Duration {
	if len(frames) == 0 {
		return 0
	}
	cp := append([]time.Duration(nil), frames...)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	idx := int(float64(len(cp)-1) * p)
	return cp[idx]
}
