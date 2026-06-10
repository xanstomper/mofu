// Package kernel provides the execution core for MOFU (Modular Orchestrated Flow Utility).
//
// The kernel runs a hybrid execution model with two paths:
//
//	Fast Path (90-95% of operations):
//	  input → state mutation → dirty propagation → UI diff → render
//	  No plugins, no full DAG recompute, no heavy scheduling.
//
//	Slow Path (complex operations):
//	  plugins + async jobs + external IO + heavy computations
//
// The fast path ensures sub-frame input latency (<1-5ms perceived) by
// bypassing everything except the minimal state→render pipeline.
package kernel

import (
	"context"
	"hash/fnv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/anomalyco/mofu/effect"
	"github.com/anomalyco/mofu/message"
	"github.com/anomalyco/mofu/state"
)

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

// Config holds kernel execution parameters.
type Config struct {
	FPSCap        int
	EventBufSize  int
	EffectBufSize int
	MaxTasks      int
	FastPathOnly  bool // When true, bypass slow path entirely
}

// DefaultConfig returns a configuration tuned for interactive TUIs.
func DefaultConfig() Config {
	return Config{
		FPSCap:        60,
		EventBufSize:  64,
		EffectBufSize: 32,
		MaxTasks:      100,
		FastPathOnly:  false,
	}
}

// ---------------------------------------------------------------------------
// Callback types
// ---------------------------------------------------------------------------

// RenderFunc is called each frame with the delta time since last frame.
type RenderFunc func(dt time.Duration)

// LayoutFunc is called each frame before render to compute layout.
type LayoutFunc func()

// UIFunc is called each frame to materialize the UI tree.
type UIFunc func() any

// StateChangeFunc is called when a state node changes value.
type StateChangeFunc func(id state.NodeID, oldVal, newVal any)

// ---------------------------------------------------------------------------
// Kernel — the deterministic execution engine
// ---------------------------------------------------------------------------

// Kernel is the deterministic execution engine for MOFU.
// It owns the event loop, state graph propagation, effect dispatch,
// and render scheduling. The kernel never knows about UI directly.
type Kernel struct {
	config  Config
	running atomic.Bool
	mu      sync.Mutex

	// Core subsystems
	State   *state.Graph
	Bus     *message.Bus
	Effects *effect.System

	// Callbacks
	onRender      RenderFunc
	onLayout      LayoutFunc
	onUI          UIFunc
	onStateChange StateChangeFunc

	// Frame timing
	tickRate     time.Duration
	lastTick     time.Time
	frameNum     atomic.Int64
	renderNotify chan struct{}

	// Fast path state
	lastSnapshot map[state.NodeID]any
	dirtyCount   atomic.Int64

	// Layout cache
	layoutCache    *LayoutCache
	lastTermWidth  int
	lastTermHieght int

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// New creates a new Kernel with the given configuration.
func New(cfg Config) *Kernel {
	ctx, cancel := context.WithCancel(context.Background())
	return &Kernel{
		config:       cfg,
		State:        state.NewGraph(),
		Bus:          message.NewBus(cfg.EventBufSize),
		Effects:      effect.NewSystem(cfg.EffectBufSize),
		tickRate:     time.Second / time.Duration(cfg.FPSCap),
		renderNotify: make(chan struct{}, 1),
		lastSnapshot: make(map[state.NodeID]any),
		layoutCache:  NewLayoutCache(),
		ctx:          ctx,
		cancel:       cancel,
	}
}

// OnRender registers the render callback.
func (k *Kernel) OnRender(fn RenderFunc) { k.onRender = fn }

// OnLayout registers the layout callback.
func (k *Kernel) OnLayout(fn LayoutFunc) { k.onLayout = fn }

// OnUI registers the UI materialization callback.
func (k *Kernel) OnUI(fn UIFunc) { k.onUI = fn }

// OnStateChange registers a callback for state node changes.
func (k *Kernel) OnStateChange(fn StateChangeFunc) { k.onStateChange = fn }

// ---------------------------------------------------------------------------
// Lifecycle
// ---------------------------------------------------------------------------

// Init initializes the kernel subsystems.
func (k *Kernel) Init() {
	k.Effects.RegisterDefaults()
	k.Effects.Start()
}

// Run starts the kernel event loop and render loop.
// This call blocks until Stop is called.
func (k *Kernel) Run() {
	k.running.Store(true)
	k.lastTick = time.Now()

	k.wg.Add(2)
	go k.eventLoop()
	go k.kernelLoop()

	k.wg.Wait()
}

// Stop halts the kernel and cleans up resources.
func (k *Kernel) Stop() {
	k.running.Store(false)
	k.cancel()
	k.Effects.Stop()
	k.Bus.Stop()
}

// FrameCount returns the total number of rendered frames.
func (k *Kernel) FrameCount() int64 {
	return k.frameNum.Load()
}

// Running reports whether the kernel is currently executing.
func (k *Kernel) Running() bool {
	return k.running.Load()
}

// DirtyCount returns the number of dirty state nodes since last frame.
func (k *Kernel) DirtyCount() int64 {
	return k.dirtyCount.Load()
}

// ---------------------------------------------------------------------------
// Event loop — message pump
// ---------------------------------------------------------------------------

func (k *Kernel) eventLoop() {
	defer k.wg.Done()

	for k.running.Load() {
		select {
		case <-k.ctx.Done():
			return
		case msg := <-k.Bus.Channel():
			k.handleMessage(msg)
		}
	}
}

func (k *Kernel) handleMessage(msg message.Message) {
	switch msg.Type {
	case message.TypeInput:
		k.fastPathDispatch(msg)
	case message.TypeCommand:
		k.fastPathDispatch(msg)
	case message.TypePlugin:
		if k.config.FastPathOnly {
			return
		}
		k.slowPathDispatch(msg)
	case message.TypeStream:
		k.fastPathDispatch(msg)
	case message.TypeTimer:
		k.fastPathDispatch(msg)
	case message.TypeResize:
		k.handleResize(msg)
	default:
		k.fastPathDispatch(msg)
	}
}

// fastPathDispatch bypasses plugins and heavy scheduling.
// This is the hot path for 90-95% of operations.
func (k *Kernel) fastPathDispatch(msg message.Message) {
	k.Bus.Dispatch(msg)
	k.requestRender()
}

// slowPathDispatch goes through the full effect system.
func (k *Kernel) slowPathDispatch(msg message.Message) {
	k.Bus.Dispatch(msg)
	k.requestRender()
}

func (k *Kernel) handleResize(msg message.Message) {
	k.Bus.Dispatch(msg)
	k.layoutCache.Invalidate()
	k.requestRender()
}

// ---------------------------------------------------------------------------
// Kernel loop — tick-based state propagation + render
// ---------------------------------------------------------------------------

func (k *Kernel) kernelLoop() {
	defer k.wg.Done()

	ticker := time.NewTicker(k.tickRate)
	defer ticker.Stop()

	for k.running.Load() {
		select {
		case <-k.ctx.Done():
			return
		case <-ticker.C:
			k.tick()
		case <-k.renderNotify:
			k.tick()
		}
	}
}

func (k *Kernel) tick() {
	now := time.Now()
	dt := now.Sub(k.lastTick)
	k.lastTick = now
	k.frameNum.Add(1)

	k.mu.Lock()
	defer k.mu.Unlock()

	// 1. Collect dirty nodes
	dirty := k.State.CollectDirty()
	if len(dirty) == 0 {
		return
	}
	k.dirtyCount.Store(int64(len(dirty)))

	// 2. Propagate dirty bits (incremental, not full recompute)
	for _, node := range dirty {
		k.State.Propagate(node.ID())
	}

	// 3. Notify state changes
	if k.onStateChange != nil {
		for _, node := range dirty {
			id := node.ID()
			oldVal := k.lastSnapshot[id]
			newVal := node.Value()
			k.onStateChange(id, oldVal, newVal)
			k.lastSnapshot[id] = newVal
		}
	}

	// 4. Layout
	if k.onLayout != nil {
		k.onLayout()
	}

	// 5. UI materialization
	if k.onUI != nil {
		_ = k.onUI()
	}

	// 6. Render
	if k.onRender != nil {
		k.onRender(dt)
	}
}

// requestRender signals the kernel loop to wake up and tick.
func (k *Kernel) requestRender() {
	select {
	case k.renderNotify <- struct{}{}:
	default:
	}
}

// ---------------------------------------------------------------------------
// Layout Cache — hash-based layout skip
// ---------------------------------------------------------------------------

// LayoutCache caches layout results and skips recomputation when
// terminal dimensions and state hash haven't changed.
type LayoutCache struct {
	mu         sync.RWMutex
	finger     uint64
	valid      bool
	lastWidth  int
	lastHeight int
}

// NewLayoutCache creates a new layout cache.
func NewLayoutCache() *LayoutCache {
	return &LayoutCache{}
}

// Check returns true if layout needs recomputation.
func (lc *LayoutCache) Check(w, h int, stateHash uint64) bool {
	lc.mu.RLock()
	defer lc.mu.RUnlock()
	if !lc.valid {
		return true
	}
	return lc.lastWidth != w || lc.lastHeight != h || lc.finger != stateHash
}

// Update records a successful layout computation.
func (lc *LayoutCache) Update(w, h int, stateHash uint64) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.lastWidth = w
	lc.lastHeight = h
	lc.finger = stateHash
	lc.valid = true
}

// Invalidate forces the next layout to recompute.
func (lc *LayoutCache) Invalidate() {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.valid = false
}

// HashState computes a fast hash of the current state graph for layout caching.
func HashState(g *state.Graph) uint64 {
	snap := g.Snapshot()
	h := fnv.New64a()
	for id, val := range snap {
		h.Write([]byte{byte(id >> 56), byte(id >> 48), byte(id >> 40), byte(id >> 32),
			byte(id >> 24), byte(id >> 16), byte(id >> 8), byte(id)})
		if s, ok := val.(string); ok {
			h.Write([]byte(s))
		}
	}
	return h.Sum64()
}
