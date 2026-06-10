# Mofu Development Contract

## Context
Mofu is a Go TUI framework using UI AST (Node interface), incremental dirty-rect rendering, flex layout, three-loop runtime, tween animation, reactive data store, multi-channel output pipeline, multi-pane workspace, data visualization charts, plugin architecture, and complete theming.

## Intent
Make Mofu the most powerful TUI framework across all ecosystems (Go, Rust, Python) by surpassing Bubble Tea, ratatui, and Textual on feature completeness, performance, developer experience, and production readiness.

## Contract Clauses

### C1: Mouse Support
- **Deliverable**: Full mouse protocol support: click, drag, release, scroll wheel, SGR extended mode (1000-1006)
- **Acceptance**: Click on a button triggers its handler. Drag selects text. Scroll wheel scrolls lists. Mouse enter/leave events on widgets.
- **Files**: `mouse.go`

### C2: Canvas Widget
- **Deliverable**: Pixel-level drawing surface using Braille (2x4 sub-cells), half-blocks (2 vertical sub-cells), and quarter-blocks (2x2 sub-cells). Shape primitives: line, rect, circle, ellipse, text. Path rendering.
- **Acceptance**: Render a line chart, filled circle, and text annotation on a Canvas. All visible at correct proportions.
- **Files**: `canvas.go`

### C3: Table Widget
- **Deliverable**: Sortable, scrollable, resizable-column Table widget. Header row, alternating row colors, column alignment, selection, sticky header.
- **Acceptance**: 100-row table renders at 60fps. Click column header to sort. Resize column with mouse drag. Scroll with keys/mouse.
- **Files**: `widgets/table.go`

### C4: Clipboard Integration
- **Deliverable**: Cross-platform clipboard read/write (Windows via syscall, macOS via pbpaste/pbcopy, Linux via xclip/xsel or wl-clipboard). Bracketed paste support.
- **Acceptance**: Copy text from Mofu, paste into notepad. Copy from browser, paste into Mofu input. Bracketed paste correctly buffers multi-line pasted content.
- **Files**: `clipboard.go`

### C5: Hot-Reload Dev Mode
- **Deliverable**: `mofu dev` command watches .go files, rebuilds and re-launches the TUI on change. Fallback to previous binary on build failure.
- **Acceptance**: Edit a widget color, save, see the TUI restart with new color in <500ms.
- **Files**: `cmd/mofu/main.go`, `dev.go`

### C6: Async Worker System
- **Deliverable**: `Worker` pool with context cancellation, progress reporting (0-100%), task grouping, priority queue. Integrated with the three-loop runtime.
- **Acceptance**: Fire 10 concurrent workers, see progress bars update independently, cancel mid-way, verify cleanup.
- **Files**: `worker.go`

### C7: Paragraph/WordWrap Widget
- **Deliverable**: Text widget that wraps at word boundaries, respects alignment (left/center/right/justify), supports ellipsis truncation, scrollable overflow.
- **Acceptance**: Long text wraps correctly at any terminal width. Justified text has even spacing. Truncation shows "...".
- **Files**: `widgets/text.go`

### C8: SSH Server Mode
- **Deliverable**: Mofu app serves over SSH (no client install needed). Built on Wish-style middleware chain (logging, auth, rate-limit, handler).
- **Acceptance**: `ssh localhost -p 23234` renders the TUI remotely. Middleware logs connections. Rate-limit blocks >10 conns/min.
- **Files**: `ssh.go`

### C9: Inspector/DevTools
- **Deliverable**: Ctrl+D toggles inspector overlay showing: widget tree, frame time, FPS, dirty rect count, memory stats. Clickable widget tree to highlight bounds.
- **Acceptance**: Press Ctrl+D, see debug overlay. Scroll through widget tree. Selected widget shows red border.
- **Files**: `inspector.go`, `debug.go` (update)

### C10: Performance Benchmarks
- **Deliverable**: Go benchmark suite comparing render throughput, layout time, event dispatch latency against Bubble Tea (where comparable). Target: 2x faster render, 5x lower allocation.
- **Acceptance**: `go test -bench=. -benchmem` shows stats. README includes benchmark table.
- **Files**: `*_test.go`

## Constraints
- **Zero breaking changes** to existing Node interface, runtime, or widget API
- **`go build ./...` and `go vet ./...` must pass** after every clause
- **No new external dependencies** beyond what's already in go.mod (golang.org/x/term, go-runewidth)
- **Windows + Unix compatibility** for all OS-level operations (clipboard, mouse, raw mode)
- **Documentation**: every new exported type and function gets a godoc comment

## Verification
Each clause verified by:
1. `go build ./...` - compiles
2. `go vet ./...` - no issues
3. Manual test described in Acceptance criteria
4. Example in `examples/` directory demonstrating the feature

## Termination
A clause is complete when all Acceptance criteria pass, build/vet pass, and an example exists. Clauses can be delivered in any order. Priority: C1 > C2 > C3 > C7 > C4 > C6 > C5 > C9 > C8 > C10.
