# MOFU Development Contract
## Modular Orchestrated Flow Utility

## Context

MOFU is a reactive terminal runtime + UI compiler — not a TUI framework. It is a deterministic terminal application execution system with a compiler, state graph, and rendering kernel.

**Architecture:**
- streams → message bus → state DAG → effect system → UI compiler → diff renderer
- 6 core subsystems: kernel, state, message, effect, uicompile, render + plugin
- Fast path (90-95% of operations): input → state mutation → dirty propagation → UI diff → render
- Slow path (complex operations): plugins + async jobs + external IO + heavy computations

**Positioning:**
| System | What it actually is |
|--------|-------------------|
| Bubble Tea | Event loop UI framework |
| Ratatui | Immediate-mode terminal renderer |
| OpenTUI | Component-based TUI system |
| **MOFU** | **Reactive terminal runtime + UI compiler** |

## Intent

Make MOFU the most powerful terminal application runtime across all ecosystems (Go, Rust, Python) by surpassing Bubble Tea, ratatui, and Textual on architecture, performance, developer experience, and production readiness.

## Performance Targets

| Metric | Target | How |
|--------|--------|-----|
| Input latency | <1-5ms perceived | Fast path bypass, no DAG/plugin overhead |
| Frame rate | Stable 60fps | Kernel tick + render notify, no wasted frames |
| Flicker | Zero | Synchronized Output protocol (CSI 2026) |
| Rendering | O(Δ) only changed cells | Cell-level diff, dirty rect consolidation |
| GC pressure | Near-zero | Preallocated frame buffer, no per-frame alloc |
| Layout | Cached with hash fingerprinting | Skip recomputation when state unchanged |

## Contract Clauses

### C1: Mouse Support ✅
- **Deliverable**: Full mouse protocol: click, drag, release, scroll wheel, SGR extended mode (1000-1006)
- **Acceptance**: Click triggers handler. Drag selects text. Scroll wheel scrolls lists. Mouse enter/leave events on widgets.
- **Files**: `mouse.go`
- **Status**: Complete

### C2: Canvas Widget ✅
- **Deliverable**: Pixel-level drawing using Braille (2x4 sub-cells), half-blocks, quarter-blocks. Shape primitives: line, rect, circle, ellipse, text.
- **Acceptance**: Render line chart, filled circle, text annotation on Canvas. All visible at correct proportions.
- **Files**: `canvas.go`
- **Status**: Complete

### C3: Table Widget ✅
- **Deliverable**: Sortable, scrollable, resizable-column Table. Header row, alternating row colors, column alignment, selection, sticky header.
- **Acceptance**: 100-row table at 60fps. Click header to sort. Resize column. Scroll with keys/mouse.
- **Files**: `widgets/table.go`
- **Status**: Complete

### C4: Clipboard Integration ✅
- **Deliverable**: Cross-platform clipboard read/write (Windows/macOS/Linux). Bracketed paste support.
- **Acceptance**: Copy from Mofu → notepad. Copy from browser → Mofu input. Bracketed paste buffers multi-line content.
- **Files**: `clipboard.go`
- **Status**: Complete

### C5: Hot-Reload Dev Mode ✅
- **Deliverable**: `mofu dev` watches .go files, rebuilds and re-launches on change. Fallback to previous binary on build failure.
- **Acceptance**: Edit widget color, save, see TUI restart with new color in <500ms.
- **Files**: `cmd/mofu/main.go`, `dev.go`
- **Status**: Complete

### C6: Async Worker System ✅
- **Deliverable**: `Worker` pool with context cancellation, progress reporting (0-100%), task grouping, priority queue.
- **Acceptance**: Fire 10 concurrent workers, see progress bars update independently, cancel mid-way, verify cleanup.
- **Files**: `worker.go`
- **Status**: Complete

### C7: Paragraph/WordWrap Widget ✅
- **Deliverable**: Text widget with word-wrap, alignment (left/center/right/justify), ellipsis truncation, scrollable overflow.
- **Acceptance**: Long text wraps at any terminal width. Justified text has even spacing. Truncation shows "...".
- **Files**: `widgets/text.go`
- **Status**: Complete

### C8: SSH Server Mode
- **Deliverable**: MOFU app serves over SSH. Wish-style middleware chain (logging, auth, rate-limit, handler).
- **Acceptance**: `ssh localhost -p 23234` renders TUI remotely. Middleware logs connections. Rate-limit blocks >10 conns/min.
- **Files**: `ssh.go`
- **Status**: Pending

### C9: Inspector/DevTools ✅
- **Deliverable**: Ctrl+D toggles debug overlay: widget tree, frame time, FPS, dirty rect count, memory stats. Per-node render cost tracking.
- **Acceptance**: Press Ctrl+D, see profiler overlay. Scroll through widget tree. Selected widget shows red border.
- **Files**: `render/profiler.go`
- **Status**: Complete (profiler implemented)

### C10: Performance Benchmarks
- **Deliverable**: Go benchmark suite comparing render throughput, layout time, event dispatch latency vs Bubble Tea. Target: 2x faster render, 5x lower allocation.
- **Acceptance**: `go test -bench=. -benchmem` shows stats. README includes benchmark table.
- **Files**: `*_test.go`
- **Status**: Pending

### C11: Zero-Allocation Diff Renderer ✅
- **Deliverable**: Cell-level diff engine with preallocated frame buffer, Synchronized Output (CSI 2026), SGR cache, dirty rect consolidation.
- **Acceptance**: No heap allocations per frame. Flicker-free animation. Only changed cells emitted.
- **Files**: `render/diff.go`
- **Status**: Complete

### C12: Fast-Path Kernel ✅
- **Deliverable**: Two execution paths: fast path (input→state→render) for 90% of ops, slow path (plugins+IO) for complex ops.
- **Acceptance**: Input latency <5ms. No plugin overhead on keystroke handling. Kernel tick + render notify for instant response.
- **Files**: `kernel/kernel.go`
- **Status**: Complete

### C13: Viewport-Aware Computation ✅
- **Deliverable**: Only compute/render visible content. Viewport with scroll, scroll-into-view, scrollbar thumb calculation.
- **Acceptance**: 10,000-row table renders only visible rows. Scroll is smooth. Scrollbar position accurate.
- **Files**: `render/viewport.go`
- **Status**: Complete

### C14: Tiered Memory System ✅
- **Deliverable**: 4-tier LRU cache (L1 hot / L2 cached / L3 streamed / L4 persisted) with TTL expiry and tier demotion.
- **Acceptance**: Hot data stays in L1. Cold data evicted to L4. Hit rate >80% for typical workloads.
- **Files**: `render/memory.go`
- **Status**: Complete

## Constraints
- **Zero breaking changes** to existing Node interface, runtime, or widget API
- **`go build ./...` and `go vet ./...` must pass** after every clause
- **No new external dependencies** beyond go.mod (golang.org/x/term, go-runewidth)
- **Windows + Unix compatibility** for all OS-level operations
- **Documentation**: every new exported type and function gets a godoc comment

## Architecture Rules
1. **State is the only source of truth** — UI is a derived projection
2. **Rendering is pure** — deterministic, stateless, diff-only
3. **Effects are isolated** — state engine never performs IO
4. **Plugins never bypass state engine** — all mutations go through graph
5. **Fast path for 90% of operations** — no plugin/DAG overhead on input handling
6. **Synchronized Output** — all frame updates wrapped in CSI 2026

## Verification
Each clause verified by:
1. `go build ./...` — compiles
2. `go vet ./...` — no issues
3. Manual test described in Acceptance criteria
4. Example in `examples/` directory demonstrating the feature

## Termination
A clause is complete when all Acceptance criteria pass, build/vet pass, and an example exists. Clauses can be delivered in any order.
