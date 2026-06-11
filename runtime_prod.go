package mofu

import (
	"context"
	"runtime"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Production Runtime (Anthology Ch.20)
// ---------------------------------------------------------------------------

// HealthStatusKind describes runtime health.
type HealthStatusKind uint8

const (
	HealthHealthy HealthStatusKind = iota
	HealthDegraded
	HealthFailing
)

// HealthStatus is a runtime health snapshot.
type HealthStatus struct {
	Kind   HealthStatusKind
	Reason string
	Uptime time.Duration
	FPS    float64
	Memory uint64
	Errors uint64
}

// HealthMonitor observes uptime, FPS, memory, and errors.
type HealthMonitor struct {
	mu          sync.RWMutex
	start       time.Time
	frameTimes  []time.Duration
	errorCount  uint64
	memoryUsage uint64
	interval    time.Duration
}

// NewHealthMonitor returns a monitor.
func NewHealthMonitor(interval time.Duration) *HealthMonitor {
	if interval <= 0 {
		interval = time.Second
	}
	return &HealthMonitor{start: time.Now(), frameTimes: make([]time.Duration, 0, 120), interval: interval}
}

// RecordFrame records a frame duration.
func (h *HealthMonitor) RecordFrame(d time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.frameTimes = append(h.frameTimes, d)
	if len(h.frameTimes) > 120 {
		h.frameTimes = h.frameTimes[len(h.frameTimes)-120:]
	}
}

// RecordError increments errors.
func (h *HealthMonitor) RecordError() { h.mu.Lock(); h.errorCount++; h.mu.Unlock() }

// RecordMemory records current memory usage.
func (h *HealthMonitor) RecordMemory(bytes uint64) { h.mu.Lock(); h.memoryUsage = bytes; h.mu.Unlock() }

// Check returns current health.
func (h *HealthMonitor) Check() HealthStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()
	fps := computeFPSLocked(h.frameTimes)
	kind := HealthHealthy
	reason := "healthy"
	if fps > 0 && fps < 30 {
		kind = HealthDegraded
		reason = "low FPS"
	}
	if h.memoryUsage > 512*1024*1024 {
		kind = HealthDegraded
		reason = "high memory"
	}
	if h.errorCount > 100 {
		kind = HealthFailing
		reason = "high error count"
	}
	return HealthStatus{Kind: kind, Reason: reason, Uptime: time.Since(h.start), FPS: fps, Memory: h.memoryUsage, Errors: h.errorCount}
}

// SelfHealingRuntime supervises components and restarts them after failures.
type SelfHealingRuntime struct {
	mu         sync.Mutex
	maxRetries map[string]int
	running    map[string]context.CancelFunc
}

// Component is a runtime component with lifecycle hooks.
type Component interface {
	Name() string
	Start(context.Context) error
	Stop() error
}

// NewSelfHealingRuntime returns an empty supervisor.
func NewSelfHealingRuntime() *SelfHealingRuntime {
	return &SelfHealingRuntime{maxRetries: make(map[string]int), running: make(map[string]context.CancelFunc)}
}

// Start launches a component with restart supervision.
func (s *SelfHealingRuntime) Start(ctx context.Context, component Component, maxRetries int) {
	s.mu.Lock()
	name := component.Name()
	s.maxRetries[name] = maxRetries
	s.mu.Unlock()
	child, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	s.running[name] = cancel
	s.mu.Unlock()
	go func() {
		for {
			if err := component.Start(child); err != nil {
				if !s.shouldRestart(name) {
					return
				}
				continue
			}
			return
		}
	}()
}

// Stop stops a supervised component.
func (s *SelfHealingRuntime) Stop(name string) error {
	s.mu.Lock()
	cancel, ok := s.running[name]
	delete(s.running, name)
	delete(s.maxRetries, name)
	s.mu.Unlock()
	if ok {
		cancel()
	}
	return nil
}

func (s *SelfHealingRuntime) shouldRestart(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.maxRetries[name]--
	return s.maxRetries[name] >= 0
}

// MetricsCollector stores counters and gauges.
type MetricsCollector struct {
	mu       sync.Mutex
	counters map[string]uint64
	gauges   map[string]float64
}

// NewMetricsCollector returns a collector.
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{counters: make(map[string]uint64), gauges: make(map[string]float64)}
}

// Inc increments a counter.
func (m *MetricsCollector) Inc(name string, n uint64) {
	m.mu.Lock()
	m.counters[name] += n
	m.mu.Unlock()
}

// Set sets a gauge.
func (m *MetricsCollector) Set(name string, v float64) {
	m.mu.Lock()
	m.gauges[name] = v
	m.mu.Unlock()
}

// Snapshot returns all metrics.
func (m *MetricsCollector) Snapshot() MetricsSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	return MetricsSnapshot{Counters: copyUint64Map(m.counters), Gauges: copyFloatMap(m.gauges)}
}

// MetricsSnapshot is a snapshot of metrics.
type MetricsSnapshot struct {
	Counters map[string]uint64
	Gauges   map[string]float64
}

// GracefulShutdown coordinates shutdown with timeout.
func GracefulShutdown(ctx context.Context, timeout time.Duration, fn func(context.Context) error) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return fn(shutdownCtx)
}

// RuntimeMemory returns current Go runtime memory usage.
func RuntimeMemory() uint64 {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return ms.Alloc
}

func copyUint64Map(in map[string]uint64) map[string]uint64 {
	out := make(map[string]uint64, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func copyFloatMap(in map[string]float64) map[string]float64 {
	out := make(map[string]float64, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
