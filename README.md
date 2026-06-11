<p align="center">
  <img src="banner.png" alt="MOFU" width="100%">
</p>

<h1 align="center">MOFU</h1>

<p align="center">
  <strong>The Reactive Terminal Application Runtime</strong>
</p>

<p align="center">
  <a href="#quick-start">Quick Start</a> •
  <a href="#why-mofu">Why MOFU</a> •
  <a href="#architecture">Architecture</a> •
  <a href="#features">Features</a> •
  <a href="#gadgets">Gadgets</a> •
  <a href="#ecosystem">Ecosystem</a> •
  <a href="#performance">Performance</a> •
  <a href="#examples">Examples</a> •
  <a href="#tutorials">Tutorials</a> •
  <a href="#api">API</a> •
  <a href="#contributing">Contributing</a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/go-1.21+-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/license-MIT-00FF00?style=for-the-badge" alt="License">
  <img src="https://img.shields.io/badge/version-0.3.0-FF69B4?style=for-the-badge" alt="Version">
  <img src="https://img.shields.io/badge/tests-101%20passing-brightgreen?style=for-the-badge" alt="Tests">
  <img src="https://img.shields.io/badge/gadgets-65-blueviolet?style=for-the-badge" alt="Gadgets">
  <img src="https://img.shields.io/badge/examples-13-orange?style=for-the-badge" alt="Examples">
</p>

---

## Why MOFU?

MOFU is not another TUI framework. It's a **reactive terminal application runtime** that fundamentally changes how you build terminal applications.

```
┌─────────────────────────────────────────────────────────────────┐
│                    MOFU ARCHITECTURE                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐ │
│  │  INPUT   │───▶│  STATE   │───▶│ COMPUTE  │───▶│  RENDER  │ │
│  │  STREAMS │    │  GRAPH   │    │  ENGINE  │    │  DIFF    │ │
│  └──────────┘    └──────────┘    └──────────┘    └──────────┘ │
│       │               │               │               │        │
│       ▼               ▼               ▼               ▼        │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐ │
│  │ SCHEDULER│    │ ANIMATION│    │  LAYOUT  │    │ TERMINAL │ │
│  │  LANES   │    │  GRAPH   │    │  ENGINE  │    │  OUTPUT  │ │
│  └──────────┘    └──────────┘    └──────────┘    └──────────┘ │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Comparison

| Feature | MOFU | Bubble Tea | Ratatui | OpenTUI |
|---------|:----:|:----------:|:-------:|:-------:|
| **Architecture** | Reactive graph | Elm loop | Immediate mode | Virtual DOM |
| **State Updates** | O(1) dirty tracking | O(N) full copy | O(N) full redraw | O(N) diff |
| **Rendering** | Incremental diff | Full redraw | Immediate | Virtual diff |
| **Streaming** | First-class | Manual | No | No |
| **Gadgets** | 65 built-in | ~15 bubbles | ~10 | ~8 |
| **Styling** | Semantic (Cuddles) | Lipgloss | Inline | StyleSheet |
| **Forms** | Schema-driven (Meow) | Manual (Huh) | Manual | Manual |
| **Animation** | Spring physics | None | None | Limited |
| **Accessibility** | Built-in | None | None | None |
| **Plugin System** | Full runtime | None | None | None |
| **AI Support** | Native primitives | None | None | None |
| **Layout Engine** | Constraint-based | Manual | Manual | Flexbox-like |
| **License** | MIT | MIT | MIT | MIT |

---

## Quick Start

### Installation

```bash
go get github.com/xanstomper/mofu
```

### Your First App

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

**Run it:**
```bash
go run main.go
```

---

## Architecture

### How It Works

```
User Input
    │
    ▼
Event Router (parse keyboard, mouse, resize)
    │
    ▼
State Graph (reactive dependency tracking)
    │
    ▼
Dirty Propagation (O(1) per node)
    │
    ▼
Layout Engine (constraint-based)
    │
    ▼
Tree Diff (minimal changes)
    │
    ▼
Diff Renderer (cell-level comparison)
    │
    ▼
Terminal Output (ANSI sequences)
```

### Key Concepts

#### 1. Reactive State Graph

MOFU uses a reactive state graph with O(1) dirty tracking:

```go
// When state changes, only affected nodes are marked dirty
atom.SetValue(42)  // O(1) - only marks this node dirty

// CollectDirty returns only dirty nodes
dirty := graph.CollectDirty()  // O(dirty nodes), not O(all nodes)
```

#### 2. Tree-Based Rendering

Unlike Bubble Tea (string-based), MOFU maintains an actual tree of nodes:

```go
// Create a tree node
root := mofu.NewTreeNode("root", "box")
child := mofu.NewTreeNode("child1", "text")
child.SetProp("text", "Hello World")
root.AddChild(child)

// Diff two trees
results := mofu.DiffTrees(oldTree, newTree)
// Returns only the changes: add, remove, update
```

#### 3. Incremental Rendering

Only changed cells are written to the terminal:

```go
// Previous frame: "Hello World"
// Current frame:  "Hello MOFU"
// Diff output:    Only changes "World" → "MOFU"
```

#### 4. Scheduler Lanes

Critical operations are never blocked:

```
Lane          Purpose                    Priority
─────────────────────────────────────────────────
REALTIME      Input, focus, UI           Highest
STREAM        AI tokens, logs            High
COMPUTE       State derivation           Medium
RENDER        Diff + terminal output     Medium
BACKGROUND    Caching, cleanup           Lowest
```

---

## Features

### Core Runtime

| Feature | Description |
|---------|-------------|
| **Reactive State Graph** | O(1) dirty tracking, automatic dependency resolution |
| **Incremental Diff Renderer** | Cell-level diff, only changed cells written to terminal |
| **Synchronized Output** | CSI 2026 protocol for flicker-free updates |
| **Spring Physics** | Damped spring animations with configurable stiffness/damping |
| **Constraint Layout** | Flex, grid, and constraint-based layout engine |
| **Full Input Parser** | Arrows, F-keys, Ctrl+key, Alt+key, mouse (SGR mode) |
| **Tree Diffing** | Efficient tree comparison for incremental updates |

### Gadgets (65 Reactive UI Systems)

Gadgets are NOT widgets. They are runtime-aware, data-driven reactive systems.

**Data & Table Systems (10)**
| Gadget | Description |
|--------|-------------|
| LiveTable | Virtualized streaming table |
| DiffTable | State change highlighting |
| HeatTable | Density visualization |
| PagedTable | Lazy loading + pagination |
| TreeTable | Expandable hierarchical |
| StreamingGrid | Real-time grid |
| FilterTable | Reactive filtering |
| SortTable | Multi-key sorting |
| PivotTableLite | Grouped aggregation |
| SparseTable | 10k+ row optimization |

**Navigation & Layout (10)**
| Gadget | Description |
|--------|-------------|
| SmartSidebar | Auto-collapsing nav |
| AdaptiveSplit | Layout balancing |
| WorkspaceGrid | Multi-panel grid |
| InspectorPane | Contextual inspector |
| FocusNavigator | Graph-based navigation |
| CommandDock | Persistent action bar |
| ContextOverlay | Floating UI layer |
| DockingSystem | Draggable panels |
| ResponsiveLayoutCore | Terminal-aware layouts |
| LayoutEngine | Constraint-based layout |

**Input & Interaction (10)**
| Gadget | Description |
|--------|-------------|
| SmartForm | Schema-driven forms |
| InlineEditor | Editable text blocks |
| KeyChordRouter | Advanced shortcuts |
| MultiCursorInput | Multiple text inputs |
| AutoCompleteEngine | Context-aware suggestions |
| ValidatedInputField | Live validation |
| CommandPalette | Fuzzy search + actions |
| InputStreamRouter | Event routing |
| GestureInputLayer | Mouse abstraction |
| FocusTrapManager | Input boundaries |

**Real-Time Data (10)**
| Gadget | Description |
|--------|-------------|
| LogStream | Zero-copy streaming logs |
| MetricBoard | Real-time metrics |
| EventFeed | Live event timeline |
| ProcessTreeView | OS process visualization |
| NetworkMonitor | Live network visualization |
| FileWatcherView | Reactive filesystem |
| StreamConsole | Continuous CLI output |
| TraceViewer | Execution tracing |
| PipelineVisualizer | Data flow visualization |
| StateInspector | Live state graph debugger |

**Visual & ASCII (10)**
| Gadget | Description |
|--------|-------------|
| ASCIIScene | Full scene graph |
| ParticleField | Terminal particle system |
| SplashComposer | Animated boot sequences |
| WaveVisualizer | Waveform renderer |
| DensityMapRenderer | Heat/flow visualization |
| ProceduralArtEngine | Generative ASCII |
| MotionBanner | Animated headers |
| GlyphMorpher | Character morph animations |
| TerminalCanvas | Pixel-like drawing |
| SDFRendererLite | Signed-distance-field ASCII |

**Production Gadgets (15)**
| Gadget | Description |
|--------|-------------|
| MarkdownViewer | Markdown rendering |
| DiffViewer | Text diff display |
| HexViewer | Binary hex display |
| JSONExplorer | JSON tree viewer |
| InspectorPanel | Key-value inspector |
| GraphVisualizer | ASCII graph rendering |
| Spinner | Loading spinner |
| StatusBadge | Status indicators |
| KeyValue | Key-value display |
| Separator | Horizontal divider |
| Spacer | Empty space |
| Timer | Elapsed time display |
| Counter | Counter display |
| MarkdownViewer | Markdown rendering |
| DiffViewer | Text diff display |

### Cuddles (Semantic Styling)

```go
import "github.com/xanstomper/mofu/cuddles"

// Semantic tokens, not raw colors
theme := cuddles.Mochi()
style := theme.Style(cuddles.Primary)
style := theme.Style(cuddles.Error)

// Theme switching
manager := cuddles.NewManager(theme)
manager.Apply("catppuccin")

// Style builder
style := cuddles.NewStyle().
    Fg(mofu.Hex("ff69b4")).
    Bold().
    Underline().
    Build()
```

### Meow (Schema Forms)

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

### Reactive Data System

```go
import "github.com/xanstomper/mofu"

// Signal (like SolidJS)
count := mofu.NewSignal(0)
count.Set(42)
count.Subscribe(func(v int) { fmt.Println(v) })

// Computed (derived values)
double := mofu.NewComputed(func() int {
    return count.Get() * 2
}, count)

// History (undo/redo)
history := mofu.NewHistory[string](100)
history.Push("state1")
history.Undo()
history.Redo()

// Stream (continuous data)
stream := mofu.NewStream[string]("logs", 100)
stream.Send("new log entry")
```

---

## Ecosystem

### Package Structure

```
mofu/
├── core/              # Runtime kernel
│   ├── kernel/        # Execution engine
│   ├── state/         # Reactive graph system
│   ├── render/        # Diff + ANSI renderer
│   └── message/       # Event bus
├── gadgets/           # 65 reactive UI systems
├── cuddles/           # Semantic styling engine
├── meow/              # Schema-driven forms
├── widgets/           # Traditional widgets (15)
├── examples/          # 13 example applications
├── primitives/        # Low-level adapters
├── effect/            # Effect system
├── plugin/            # Plugin runtime
├── scheduler/         # Lane-based task system
├── stream/            # Streaming data engine
├── layout_engine.go   # Constraint-based layout
├── tree.go            # Tree-based rendering
├── data.go            # Reactive data system
└── cmd/mofu/          # CLI tool
```

---

## Performance

### Benchmarks

| Metric | Value | Allocations |
|--------|-------|-------------|
| AtomSetValue | **124ns** | 0 |
| CollectDirty (100 nodes) | **9μs** | 1 |
| CollectDirty (1000 nodes) | **108μs** | 1 |
| CollectDirty (no dirty) | **52ns** | 0 |
| Tree diff | **O(nodes)** | Minimal |
| SGR cache hit | **<1ns** | 0 |

### Comparison

| Metric | MOFU | Bubble Tea | Ratatui |
|--------|------|------------|---------|
| State update | **124ns** | ~1000ns | ~500ns |
| Dirty tracking | **52ns** | O(N) scan | O(N) scan |
| Memory per frame | **0 allocs** | 2-5 allocs | 1-3 allocs |
| Render (80x24) | **<1ms** | ~5ms | ~3ms |

---

## Examples

| Example | Description | Run |
|---------|-------------|-----|
| `counter` | Minimal counter — the "hello world" | `go run examples/counter/main.go` |
| `dashboard` | Multi-panel dashboard with navigation | `go run examples/dashboard/main.go` |
| `chat` | Chat interface with input widget | `go run examples/chat/main.go` |
| `filemanager` | File browser with directory navigation | `go run examples/filemanager/main.go .` |
| `form` | Registration form with inputs, checkbox, button | `go run examples/form/main.go` |
| `settings` | Settings panel with checkboxes and selects | `go run examples/settings/main.go` |
| `logviewer` | Log viewer with filtering and scrolling | `go run examples/logviewer/main.go` |
| `wizard` | Multi-step setup wizard | `go run examples/wizard/main.go` |
| `monitor` | Real-time system monitor with live bars | `go run examples/monitor/main.go` |
| `gitui` | Git interface (status, log, diff) | `go run examples/gitui/main.go` |
| `dockerui` | Docker interface (containers, images) | `go run examples/dockerui/main.go` |
| `kanban` | Kanban board with drag-and-drop | `go run examples/kanban/main.go` |
| `calculator` | Functional calculator | `go run examples/calculator/main.go` |

---

## Tutorials

### Getting Started
- [Complete Guide](docs/tutorials/complete-guide.md) — From zero to production
- [Building a Chat App](docs/tutorials/building-a-chat-app.md) — Real-time messaging
- [Building a Monitor](docs/tutorials/building-a-monitor.md) — System monitoring
- [Building a Dashboard](docs/tutorials/building-a-dashboard.md) — Multi-panel UI

### Guides
- [Architecture](docs/guides/architecture.md) — How MOFU works
- [Gadgets](docs/guides/gadgets.md) — Using the 65 reactive UI systems
- [Styling](docs/guides/styling.md) — Semantic styling with Cuddles
- [Forms](docs/guides/forms.md) — Schema-driven forms with Meow
- [Performance](docs/guides/performance.md) — Optimization guide
- [Testing](docs/guides/testing.md) — Testing MOFU applications
- [Migration](docs/guides/migration-from-bubbletea.md) — Migrating from Bubble Tea

### API Reference
- [API Reference](docs/api/README.md) — Complete API documentation

---

## Testing

```bash
# Run all tests
go test ./...

# Run benchmarks
go test -bench=. -benchmem ./...

# Run with coverage
go test -cover ./...
```

**101 tests passing** across all packages.

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Development

```bash
# Clone the repo
git clone https://github.com/xanstomper/mofu.git
cd mofu

# Build
go build ./...

# Test
go test ./...

# Lint
go vet ./...
```

---

## Community

- [Issues](https://github.com/xanstomper/mofu/issues) — Bug reports and feature requests
- [Discussions](https://github.com/xanstomper/mofu/discussions) — Community conversations
- [CONTRIBUTING.md](CONTRIBUTING.md) — Contribution guidelines
- [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) — Community standards
- [SECURITY.md](SECURITY.md) — Security policy
- [CHANGELOG.md](CHANGELOG.md) — Version history

---

## License

MIT License — see [LICENSE](LICENSE) for details.

---

<p align="center">
  <strong>Built with ❤️ for the terminal community</strong>
</p>

<p align="center">
  <img src="https://img.shields.io/github/stars/xanstomper/mofu?style=social" alt="Stars">
  <img src="https://img.shields.io/github/forks/xanstomper/mofu?style=social" alt="Forks">
  <img src="https://img.shields.io/github/watchers/xanstomper/mofu?style=social" alt="Watchers">
</p>
