// Package render provides the introspective debug overlay for MOFU.
//
// The profiler tracks per-node render cost, IO latency, frame timing,
// and memory usage. Toggle with Ctrl+D at runtime to see the overlay.
//
// This is MOFU's differentiator: a self-observing system that can
// visualize its own execution graph, render bottlenecks, and state
// propagation costs in real-time.
package render

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

// ProfileNode stores performance data for a single render node.
type ProfileNode struct {
	ID         string
	Name       string
	RenderTime time.Duration
	LayoutTime time.Duration
	FrameCount int64
	LastSeen   time.Time
}

// Profiler tracks runtime performance metrics for the MOFU kernel.
type Profiler struct {
	mu          sync.RWMutex
	nodes       map[string]*ProfileNode
	frameStart  time.Time
	frameEnd    time.Time
	frameTime   time.Duration
	fps         float64
	fpsCount    int
	fpsLastTime time.Time
	dirtyRects  int
	memStats    runtime.MemStats
	ioLatency   time.Duration
	enabled     bool
}

// NewProfiler creates a new Profiler.
func NewProfiler() *Profiler {
	return &Profiler{
		nodes:       make(map[string]*ProfileNode),
		fpsLastTime: time.Now(),
		enabled:     false,
	}
}

// Enable turns on profiling collection.
func (p *Profiler) Enable() { p.mu.Lock(); p.enabled = true; p.mu.Unlock() }

// Disable turns off profiling collection.
func (p *Profiler) Disable() { p.mu.Lock(); p.enabled = false; p.mu.Unlock() }

// Enabled reports whether profiling is active.
func (p *Profiler) Enabled() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.enabled
}

// BeginFrame marks the start of a frame.
func (p *Profiler) BeginFrame() {
	if !p.enabled {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.frameStart = time.Now()
}

// EndFrame marks the end of a frame and updates FPS.
func (p *Profiler) EndFrame() {
	if !p.enabled {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.frameEnd = time.Now()
	p.frameTime = p.frameEnd.Sub(p.frameStart)
	p.fpsCount++

	now := time.Now()
	elapsed := now.Sub(p.fpsLastTime)
	if elapsed >= time.Second {
		p.fps = float64(p.fpsCount) / elapsed.Seconds()
		p.fpsCount = 0
		p.fpsLastTime = now
	}
}

// BeginNode marks the start of rendering for a named node.
func (p *Profiler) BeginNode(id, name string) func() {
	if !p.enabled {
		return func() {}
	}
	start := time.Now()
	return func() {
		p.mu.Lock()
		defer p.mu.Unlock()
		node, ok := p.nodes[id]
		if !ok {
			node = &ProfileNode{ID: id, Name: name}
			p.nodes[id] = node
		}
		node.RenderTime = time.Since(start)
		node.FrameCount++
		node.LastSeen = time.Now()
	}
}

// SetDirtyRects records the number of dirty rects in the current frame.
func (p *Profiler) SetDirtyRects(n int) {
	if !p.enabled {
		return
	}
	p.mu.Lock()
	p.dirtyRects = n
	p.mu.Unlock()
}

// SetIOLatency records IO latency for the current frame.
func (p *Profiler) SetIOLatency(d time.Duration) {
	if !p.enabled {
		return
	}
	p.mu.Lock()
	p.ioLatency = d
	p.mu.Unlock()
}

// Snapshot returns a copy of all current profiling data.
func (p *Profiler) Snapshot() ProfileSnapshot {
	p.mu.RLock()
	defer p.mu.RUnlock()

	runtime.ReadMemStats(&p.memStats)

	snap := ProfileSnapshot{
		FPS:         p.fps,
		FrameTime:   p.frameTime,
		DirtyRects:  p.dirtyRects,
		IOLatency:   p.ioLatency,
		HeapAlloc:   p.memStats.HeapAlloc,
		HeapObjects: p.memStats.HeapObjects,
		NumGC:       p.memStats.NumGC,
		Nodes:       make([]ProfileNodeSnapshot, 0, len(p.nodes)),
	}

	for _, n := range p.nodes {
		snap.Nodes = append(snap.Nodes, ProfileNodeSnapshot{
			ID:         n.ID,
			Name:       n.Name,
			RenderTime: n.RenderTime,
			FrameCount: n.FrameCount,
		})
	}

	return snap
}

// ProfileSnapshot is an immutable snapshot of profiling data.
type ProfileSnapshot struct {
	FPS         float64
	FrameTime   time.Duration
	DirtyRects  int
	IOLatency   time.Duration
	HeapAlloc   uint64
	HeapObjects uint64
	NumGC       uint32
	Nodes       []ProfileNodeSnapshot
}

// ProfileNodeSnapshot is a snapshot of a single node's performance data.
type ProfileNodeSnapshot struct {
	ID         string
	Name       string
	RenderTime time.Duration
	FrameCount int64
}

// RenderOverlay renders the debug overlay as a list of strings.
// Call this from the render callback when profiler is enabled.
func (p *Profiler) RenderOverlay(width int) []string {
	snap := p.Snapshot()

	lines := []string{
		"╔═══ MOFU Profiler ═══╗",
		fmt.Sprintf("║ FPS:      %.1f", snap.FPS),
		fmt.Sprintf("║ Frame:    %v", snap.FrameTime.Round(time.Microsecond)),
		fmt.Sprintf("║ Dirty:    %d rects", snap.DirtyRects),
		fmt.Sprintf("║ IO:       %v", snap.IOLatency.Round(time.Microsecond)),
		fmt.Sprintf("║ Heap:     %.1f KB", float64(snap.HeapAlloc)/1024.0),
		fmt.Sprintf("║ Objects:  %d", snap.HeapObjects),
		fmt.Sprintf("║ GC:       %d", snap.NumGC),
		"╠═══ Nodes ═══╣",
	}

	for _, n := range snap.Nodes {
		if n.FrameCount == 0 {
			continue
		}
		lines = append(lines,
			fmt.Sprintf("║ %s: %v (%d frames)",
				n.Name, n.RenderTime.Round(time.Microsecond), n.FrameCount))
	}

	lines = append(lines, "╚═══════════════════╝")
	return lines
}
