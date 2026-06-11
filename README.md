# MOFU

**A streaming-first reactive runtime for terminal applications.**

MOFU is not a TUI framework. It's a terminal-native application runtime with a reactive visual layer, designed for AI workloads, long-running processes, and high-throughput interfaces.

## Quick Start

```go
package main

import (
    "fmt"
    "os"
    "github.com/xanstomper/mofu"
)

type Counter struct {
    mofu.Minimal
    count int
}

func (c *Counter) Render(ctx *mofu.RenderContext) {
    r := ctx.Bounds
    style := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))
    text := fmt.Sprintf("Count: %d\n\nPress j/k to change, q to quit", c.count)
    ctx.Renderer.WriteStyledString(text, r.X, r.Y, style)
}

func (c *Counter) HandleEvent(event mofu.Event) mofu.Cmd {
    if event.Type != mofu.EventKeyPress {
        return nil
    }
    ke := event.Data.(mofu.KeyEvent)
    switch {
    case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
        return mofu.QuitCmd()
    case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
        c.count++
    case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
        c.count--
    }
    return nil
}

func main() {
    if err := mofu.Run(&Counter{}); err != nil {
        fmt.Fprintf(os.Stderr, "error: %v\n", err)
        os.Exit(1)
    }
}
```

## Architecture

MOFU uses a streaming-first reactive architecture:

```
Input Streams → Router → State Graph → Compute → Render Diff → Terminal
```

### Core Systems

| System | Description |
|--------|-------------|
| Reactive State Graph | O(1) dirty tracking, dependency-aware updates |
| Incremental Diff Renderer | Cell-level diff, Synchronized Output (CSI 2026) |
| Scheduler Lanes | 5 priority lanes: Realtime, Stream, Compute, Render, Background |
| Stream Engine | First-class streaming for AI, logs, events |
| Animation Graph | Declarative spring physics + 12 easing functions |
| Layout Engine | Constraint-based layout with flex/grid support |
| Plugin System | Full gadget runtime with state isolation |

## Ecosystem

MOFU ships as layered capability systems:

### 🧠 MOFU (Core Runtime)
- Reactive state graph with O(1) dirty tracking
- Incremental diff renderer with SGR cache
- Spring physics animations
- Constraint-based layout engine
- Full input parser (arrows, F-keys, Ctrl, Alt, mouse)

### 🎛️ Gadgets (50 Reactive UI Systems)

Gadgets are NOT widgets. They are runtime-aware, data-driven reactive systems.

**Data & Table Systems (10)**
- LiveTable — virtualized streaming table
- DiffTable — state change highlighting
- HeatTable — density visualization
- PagedTable — lazy loading + pagination
- TreeTable — expandable hierarchical
- StreamingGrid — real-time grid
- FilterTable — reactive filtering
- SortTable — multi-key sorting
- PivotTableLite — grouped aggregation
- SparseTable — 10k+ row optimization

**Navigation & Layout (10)**
- SmartSidebar — auto-collapsing nav
- AdaptiveSplit — layout balancing
- WorkspaceGrid — multi-panel grid
- InspectorPane — contextual inspector
- FocusNavigator — graph-based navigation
- CommandDock — persistent action bar
- ContextOverlay — floating UI layer
- DockingSystem — draggable panels
- ViewportManager — visible region only
- ResponsiveLayoutCore — terminal-aware layouts

**Input & Interaction (10)**
- SmartForm — schema-driven forms
- InlineEditor — editable text blocks
- KeyChordRouter — advanced shortcuts
- MultiCursorInput — multiple text inputs
- AutoCompleteEngine — context-aware suggestions
- ValidatedInputField — live validation
- CommandPalette — fuzzy search + actions
- InputStreamRouter — event routing
- GestureInputLayer — mouse abstraction
- FocusTrapManager — input boundaries

**Real-Time Data (10)**
- LogStream — zero-copy streaming logs
- MetricBoard — real-time metrics
- EventFeed — live event timeline
- ProcessTreeView — OS process visualization
- NetworkMonitor — live network visualization
- FileWatcherView — reactive filesystem
- StreamConsole — continuous CLI output
- TraceViewer — execution tracing
- PipelineVisualizer — data flow visualization
- StateInspector — live state graph debugger

**Visual & ASCII (10)**
- ASCIIScene — full scene graph
- ParticleField — terminal particle system
- SplashComposer — animated boot sequences
- WaveVisualizer — waveform renderer
- DensityMapRenderer — heat/flow visualization
- ProceduralArtEngine — generative ASCII
- MotionBanner — animated headers
- GlyphMorpher — character morph animations
- TerminalCanvas — pixel-like drawing
- SDFRendererLite — signed-distance-field ASCII

### 🎀 Cuddles (Semantic Styling)

```go
import "github.com/xanstomper/mofu/cuddles"

// Semantic tokens, not raw colors
theme := cuddles.Mochi()
style := theme.Style(cuddles.Primary)
style := theme.Style(cuddles.Error)

// Theme switching
manager := cuddles.NewManager(theme)
manager.Apply("catppuccin")
```

### 🐱 Meow (Schema Forms)

```go
import "github.com/xanstomper/mofu/meow"

// Declarative form schema
form := meow.NewForm(
    meow.Input("name", "Name").SetRequired(),
    meow.Input("email", "Email").Validate(meow.ValidateEmail),
    meow.Select("role", "Role", []string{"Admin", "User"}),
    meow.Checkbox("agree", "I agree to terms"),
)

form.OnSubmit(func(values map[string]any) mofu.Cmd {
    return nil
})
```

### 🎨 Widgets (Traditional)

```go
import "github.com/xanstomper/mofu/widgets"

// 15 pre-built widgets
input := widgets.NewInput()
list := widgets.NewList(items)
btn := widgets.NewButton("Click", nil)
table := widgets.NewTable(columns, rows)
```

## Examples

| Example | Description |
|---------|-------------|
| `examples/counter/` | Minimal counter — the "hello world" |
| `examples/dashboard/` | Multi-panel dashboard with navigation |
| `examples/chat/` | Chat interface with input widget |
| `examples/filemanager/` | File browser with directory navigation |
| `examples/form/` | Registration form with inputs, checkbox, button |
| `examples/settings/` | Settings panel with checkboxes and selects |
| `examples/logviewer/` | Log viewer with filtering and scrolling |
| `examples/wizard/` | Multi-step setup wizard |
| `examples/monitor/` | Real-time system monitor with live bars |

Run any example:

```bash
cd examples/counter && go run main.go
cd examples/dashboard && go run main.go
cd examples/wizard && go run main.go
```

## Features

### Input Handling
- Arrow keys, Home/End, PgUp/PgDn
- Function keys F1-F12
- Ctrl+key combinations (Ctrl+C, Ctrl+Z, etc.)
- Alt+key combinations
- Mouse events (SGR mode)
- Unicode input

### Rendering
- Double-buffered diff renderer
- Synchronized Output (CSI 2026) for flicker-free updates
- SGR cache for zero-allocation style lookups
- Dirty rect consolidation for minimal cursor movement

### Animation
- Spring physics with configurable stiffness/damping
- 12 easing functions (linear, quad, cubic, elastic, bounce, back)
- Timeline sequencing
- Staggered animations

### Accessibility
- Full ARIA-like semantic roles
- Focus management
- Screen reader hooks
- High contrast mode
- Reduced motion support

### Persistence
- JSON state store with auto-save
- File-backed state persistence
- LRU cache with TTL expiry
- State migration support

## Performance

| Metric | Value |
|--------|-------|
| State update | 185ns, 0 allocs |
| Dirty tracking (1000 nodes) | 106μs, 1 alloc |
| Input parse | <100ns |
| Diff render | Cell-level, minimal ANSI |

## Testing

```bash
go test ./...
```

86 tests passing across all packages.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT License - see [LICENSE](LICENSE)

## Community

- [CONTRIBUTING.md](CONTRIBUTING.md) — Contribution guidelines
- [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) — Community standards
- [SECURITY.md](SECURITY.md) — Security policy
- [CHANGELOG.md](CHANGELOG.md) — Version history
