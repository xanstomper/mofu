package mofu

import (
	"sync"
	"sync/atomic"
	"time"
)

// ---------------------------------------------------------------------------
// Backpressure Control — batching, compression, drop policies
// ---------------------------------------------------------------------------

// DropPolicy controls what happens when a channel is full.
type DropPolicy int

const (
	DropOldest  DropPolicy = iota // drop oldest item in buffer
	DropNewest                    // drop incoming item
	DropCoalesce                  // merge items together
	DropBlock                     // block until space available
)

// BackpressureConfig configures backpressure behavior.
type BackpressureConfig struct {
	MaxBuffer    int           // maximum buffered items
	BatchSize    int           // items per batch
	BatchTimeout time.Duration // max time between batches
	DropPolicy  DropPolicy
	Compress    bool // enable output compression
}

// DefaultBackpressureConfig returns sensible defaults.
func DefaultBackpressureConfig() BackpressureConfig {
	return BackpressureConfig{
		MaxBuffer:    1000,
		BatchSize:    10,
		BatchTimeout: 16 * time.Millisecond, // ~1 frame
		DropPolicy:   DropOldest,
		Compress:     false,
	}
}

// ---------------------------------------------------------------------------
// BackpressureChannel — buffered channel with backpressure
// ---------------------------------------------------------------------------

// BackpressureChannel is a buffered channel with configurable backpressure.
type BackpressureChannel[T any] struct {
	mu       sync.Mutex
	buf      []T
	config   BackpressureConfig
	onDrop   func(T)
	onBatch  func([]T)
	dropped  atomic.Int64
	flushed  atomic.Int64
}

// NewBackpressureChannel creates a new backpressure-aware channel.
func NewBackpressureChannel[T any](config BackpressureConfig) *BackpressureChannel[T] {
	return &BackpressureChannel[T]{
		buf:    make([]T, 0, config.MaxBuffer),
		config: config,
	}
}

// OnDrop registers a callback for dropped items.
func (ch *BackpressureChannel[T]) OnDrop(fn func(T)) {
	ch.mu.Lock()
	ch.onDrop = fn
	ch.mu.Unlock()
}

// OnBatch registers a callback for batched items.
func (ch *BackpressureChannel[T]) OnBatch(fn func([]T)) {
	ch.mu.Lock()
	ch.onBatch = fn
	ch.mu.Unlock()
}

// Push adds an item to the channel. Returns false if dropped.
func (ch *BackpressureChannel[T]) Push(item T) bool {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	if len(ch.buf) >= ch.config.MaxBuffer {
		switch ch.config.DropPolicy {
		case DropOldest:
			dropped := ch.buf[0]
			ch.buf = ch.buf[1:]
			ch.dropped.Add(1)
			if ch.onDrop != nil {
				ch.onDrop(dropped)
			}
		case DropNewest:
			ch.dropped.Add(1)
			if ch.onDrop != nil {
				ch.onDrop(item)
			}
			return false
		case DropCoalesce:
			// Caller should handle coalescing externally
			ch.dropped.Add(1)
			return false
		case DropBlock:
			// Handled externally via Wait()
			return false
		}
	}

	ch.buf = append(ch.buf, item)

	// Auto-flush if batch size reached
	if len(ch.buf) >= ch.config.BatchSize {
		ch.flushLocked()
	}

	return true
}

// Flush sends all buffered items to the batch callback.
func (ch *BackpressureChannel[T]) Flush() {
	ch.mu.Lock()
	ch.flushLocked()
	ch.mu.Unlock()
}

func (ch *BackpressureChannel[T]) flushLocked() {
	if len(ch.buf) == 0 {
		return
	}
	batch := ch.buf
	ch.buf = make([]T, 0, ch.config.MaxBuffer)
	ch.flushed.Add(int64(len(batch)))
	if ch.onBatch != nil {
		ch.onBatch(batch)
	}
}

// Len returns the current buffer size.
func (ch *BackpressureChannel[T]) Len() int {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	return len(ch.buf)
}

// Stats returns channel statistics.
func (ch *BackpressureChannel[T]) Stats() BackpressureStats {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	return BackpressureStats{
		Buffered: len(ch.buf),
		Dropped:  int(ch.dropped.Load()),
		Flushed:  int(ch.flushed.Load()),
	}
}

// BackpressureStats tracks channel statistics.
type BackpressureStats struct {
	Buffered int
	Dropped  int
	Flushed  int
}

// ---------------------------------------------------------------------------
// Output Compressor — merge rapid updates
// ---------------------------------------------------------------------------

// OutputCompressor merges rapid sequential updates into single outputs.
type OutputCompressor[T any] struct {
	mu         sync.Mutex
	last       T
	count      int
	interval   time.Duration
	lastSend   time.Time
	merge      func(T, T) T
	onEmit     func(T)
}

// NewOutputCompressor creates a compressor that merges updates within an interval.
func NewOutputCompressor[T any](interval time.Duration, merge func(T, T) T) *OutputCompressor[T] {
	return &OutputCompressor[T]{
		interval: interval,
		merge:    merge,
	}
}

// OnEmit registers the output callback.
func (c *OutputCompressor[T]) OnEmit(fn func(T)) {
	c.mu.Lock()
	c.onEmit = fn
	c.mu.Unlock()
}

// Push submits an update. If within the compression interval, it merges
// with the previous update. Otherwise, it emits immediately.
func (c *OutputCompressor[T]) Push(item T) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	if c.count > 0 && now.Sub(c.lastSend) < c.interval && c.merge != nil {
		// Merge with previous
		c.last = c.merge(c.last, item)
		c.count++
		return
	}

	// Emit previous if pending
	if c.count > 0 && c.onEmit != nil {
		c.onEmit(c.last)
	}

	c.last = item
	c.count = 1
	c.lastSend = now
}

// Flush forces emission of any pending update.
func (c *OutputCompressor[T]) Flush() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.count > 0 && c.onEmit != nil {
		c.onEmit(c.last)
		c.count = 0
	}
}

// ---------------------------------------------------------------------------
// Adaptive Frame Rate — drop to lower FPS under load
// ---------------------------------------------------------------------------

// AdaptiveFrameRate adjusts the frame rate based on system load.
type AdaptiveFrameRate struct {
	mu           sync.Mutex
	targetFPS    int
	minFPS       int
	maxFPS       int
	currentFPS   int
	frameTime    time.Duration
	threshold    time.Duration // frame time threshold to trigger downgrade
	upgradeDelay time.Duration // time before upgrading back
	lastDowngrade time.Time
}

// NewAdaptiveFrameRate creates an adaptive frame rate controller.
func NewAdaptiveFrameRate(minFPS, maxFPS int) *AdaptiveFrameRate {
	return &AdaptiveFrameRate{
		targetFPS:    maxFPS,
		minFPS:       minFPS,
		maxFPS:       maxFPS,
		currentFPS:   maxFPS,
		threshold:    time.Second / time.Duration(maxFPS) * 2, // 2x target frame time
		upgradeDelay: 2 * time.Second,
	}
}

// Adjust updates the frame rate based on actual frame time.
func (a *AdaptiveFrameRate) Adjust(frameTime time.Duration) int {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.frameTime = frameTime

	if frameTime > a.threshold {
		// Downgrade
		newFPS := a.currentFPS - 5
		if newFPS < a.minFPS {
			newFPS = a.minFPS
		}
		if newFPS != a.currentFPS {
			a.currentFPS = newFPS
			a.lastDowngrade = time.Now()
		}
	} else if time.Since(a.lastDowngrade) > a.upgradeDelay {
		// Upgrade
		newFPS := a.currentFPS + 1
		if newFPS > a.maxFPS {
			newFPS = a.maxFPS
		}
		a.currentFPS = newFPS
	}

	return a.currentFPS
}

// CurrentFPS returns the current target FPS.
func (a *AdaptiveFrameRate) CurrentFPS() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.currentFPS
}

// TickInterval returns the current tick interval.
func (a *AdaptiveFrameRate) TickInterval() time.Duration {
	fps := a.CurrentFPS()
	if fps <= 0 {
		return time.Second
	}
	return time.Second / time.Duration(fps)
}

// ---------------------------------------------------------------------------
// Memory Pressure Detector
// ---------------------------------------------------------------------------

// MemoryPressureLevel indicates system memory pressure.
type MemoryPressureLevel int

const (
	MemoryPressureNone     MemoryPressureLevel = iota
	MemoryPressureLow                         // >70% heap
	MemoryPressureMedium                      // >80% heap
	MemoryPressureHigh                        // >90% heap
	MemoryPressureCritical                    // >95% heap
)

// MemoryPressure monitors memory usage and signals pressure.
type MemoryPressure struct {
	mu           sync.Mutex
	thresholds   [4]float64 // low, medium, high, critical (0-1 of heap limit)
	heapLimit    uint64
	onPressure   func(MemoryPressureLevel)
	lastLevel    MemoryPressureLevel
}

// NewMemoryPressure creates a memory pressure detector.
func NewMemoryPressure(heapLimit uint64) *MemoryPressure {
	return &MemoryPressure{
		thresholds: [4]float64{0.70, 0.80, 0.90, 0.95},
		heapLimit:  heapLimit,
	}
}

// OnPressure registers a callback for pressure changes.
func (mp *MemoryPressure) OnPressure(fn func(MemoryPressureLevel)) {
	mp.mu.Lock()
	mp.onPressure = fn
	mp.mu.Unlock()
}

// Check evaluates current memory pressure.
func (mp *MemoryPressure) Check(stats MemoryStats) MemoryPressureLevel {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	if mp.heapLimit == 0 {
		return MemoryPressureNone
	}

	ratio := float64(stats.HeapAlloc) / float64(mp.heapLimit)

	var level MemoryPressureLevel
	switch {
	case ratio >= mp.thresholds[3]:
		level = MemoryPressureCritical
	case ratio >= mp.thresholds[2]:
		level = MemoryPressureHigh
	case ratio >= mp.thresholds[1]:
		level = MemoryPressureMedium
	case ratio >= mp.thresholds[0]:
		level = MemoryPressureLow
	default:
		level = MemoryPressureNone
	}

	if level != mp.lastLevel {
		mp.lastLevel = level
		if mp.onPressure != nil {
			mp.onPressure(level)
		}
	}

	return level
}

// Level returns the current pressure level.
func (mp *MemoryPressure) Level() MemoryPressureLevel {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	return mp.lastLevel
}
