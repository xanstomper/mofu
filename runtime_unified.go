package mofu

import (
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Unified MOFU Runtime — single entry point for all subsystems
// ---------------------------------------------------------------------------

// UnifiedConfig configures the unified MOFU runtime.
type UnifiedConfig struct {
	// Terminal
	Width  int
	Height int

	// Render mode
	RenderMode RenderMode

	// Performance
	TargetFPS      int
	MaxMemory      uint64
	Backpressure   BackpressureConfig

	// Features
	EnableAnimations bool
	EnablePlugins    bool
	EnableStreams    bool
}

// DefaultUnifiedConfig returns sensible defaults.
func DefaultUnifiedConfig() UnifiedConfig {
	return UnifiedConfig{
		Width:            80,
		Height:           24,
		RenderMode:       RenderFullscreen,
		TargetFPS:        60,
		MaxMemory:        512 * 1024 * 1024,
		Backpressure:     DefaultBackpressureConfig(),
		EnableAnimations: true,
		EnablePlugins:    true,
		EnableStreams:    true,
	}
}

// MofuRuntime is the unified runtime that owns all subsystems.
type MofuRuntime struct {
	mu     sync.Mutex
	config UnifiedConfig

	// Core systems
	State       *StateGraph
	Events      *EventPropagator
	Focus       *FocusManager
	Layout      *DualRenderer

	// Performance
	Profiler    *FrameProfiler
	FrameRate   *AdaptiveFrameRate
	Memory      *MemoryPressure
	Arena       *Arena
	StringPool  *StringInterner

	// Content systems
	Animations  *Animator

	// Test support
	Recorder   *TestRecorder
	Stepper    *FrameStepper

	running    bool
}

// NewMofuRuntime creates a unified runtime with all subsystems initialized.
func NewMofuRuntime(config UnifiedConfig) *MofuRuntime {
	r := &MofuRuntime{
		config:      config,
		State:       NewStateGraph(),
		Events:      NewEventPropagator(),
		Focus:       NewFocusManager(),
		Layout:      NewDualRenderer(config.Width, config.Height),
		Profiler:    NewFrameProfiler(240),
		FrameRate:   NewAdaptiveFrameRate(15, config.TargetFPS),
		Memory:      NewMemoryPressure(config.MaxMemory),
		Arena:       NewArena(4096),
		StringPool:  NewStringInterner(),
		Animations:  NewAnimator(),
		Recorder:    NewTestRecorder(),
		Stepper:     NewFrameStepper(FPSDuration(config.TargetFPS)),
	}
	return r
}

// FPSDuration converts FPS to frame duration.
func FPSDuration(fps int) time.Duration {
	if fps <= 0 {
		fps = 60
	}
	return time.Second / time.Duration(fps)
}

// Config returns the runtime configuration.
func (r *MofuRuntime) Config() UnifiedConfig {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.config
}

// Resize updates terminal dimensions across all subsystems.
func (r *MofuRuntime) Resize(width, height int) {
	r.mu.Lock()
	r.config.Width = width
	r.config.Height = height
	r.Layout.Resize(width, height)
	r.mu.Unlock()
}

// SwitchRenderMode switches between inline and fullscreen rendering.
func (r *MofuRuntime) SwitchRenderMode(mode RenderMode) string {
	return r.Layout.SwitchMode(mode)
}

// SetState writes a value to the state graph.
func (r *MofuRuntime) SetState(path string, val any) bool {
	return r.State.Set(path, val)
}

// GetState reads a value from the state graph.
func (r *MofuRuntime) GetState(path string) (any, bool) {
	return r.State.Get(path)
}

// Snapshot captures a full runtime state snapshot.
func (r *MofuRuntime) Snapshot() RuntimeSnapshot2 {
	r.mu.Lock()
	defer r.mu.Unlock()

	return RuntimeSnapshot2{
		State:     r.State.Snapshot(),
		Width:     r.config.Width,
		Height:    r.config.Height,
		Frame:     r.Stepper.Frame(),
		Profiler:  r.Profiler.Snapshot(),
		Memory:    ReadMemoryStats(),
	}
}

// RuntimeSnapshot2 captures the complete runtime state.
type RuntimeSnapshot2 struct {
	State    map[string]any
	Width    int
	Height   int
	Frame    int64
	Profiler FrameProfileSnapshot
	Memory   MemoryStats
}

// Stats returns comprehensive runtime statistics.
func (r *MofuRuntime) Stats() RuntimeStats2 {
	return RuntimeStats2{
		FPS:          r.Profiler.FPS(),
		FrameTimeP95: r.Profiler.FrameTimeP95(),
		Memory:       ReadMemoryStats(),
		Frame:        r.Stepper.Frame(),
		Arena:        r.Arena.Stats(),
		Strings:      r.StringPool.Stats(),
	}
}

// RuntimeStats2 aggregates all runtime statistics.
type RuntimeStats2 struct {
	FPS          float64
	FrameTimeP95 time.Duration
	Memory       MemoryStats
	Frame        int64
	Arena        ArenaStats
	Strings      InternerStats
}

// ResetFrame resets frame-local resources (arena, etc.).
func (r *MofuRuntime) ResetFrame() {
	r.Arena.Reset()
}

// ---------------------------------------------------------------------------
// Convenience constructors
// ---------------------------------------------------------------------------

// QuickRun creates and runs a minimal MOFU application.
func QuickRun(root Node, opts ...Option) error {
	p := New(root, opts...)
	return p.Run()
}

// WithRuntimeOptions returns Program options from a UnifiedConfig.
func WithRuntimeOptions(config UnifiedConfig) []Option {
	var opts []Option
	if config.Width > 0 && config.Height > 0 {
		opts = append(opts, WithSize(config.Width, config.Height))
	}
	if config.TargetFPS > 0 {
		opts = append(opts, WithFPS(config.TargetFPS))
	}
	return opts
}
