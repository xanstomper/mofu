<p align="center">
  <img src="banner.png" alt="MOFU Banner" width="800">
</p>

<h1 align="center">MOFU</h1>

<p align="center">
  <strong>A streaming-first reactive runtime for terminal applications</strong>
</p>

<p align="center">
  <a href="#quick-start">Quick Start</a> •
  <a href="#features">Features</a> •
  <a href="#ecosystem">Ecosystem</a> •
  <a href="#benchmarks">Benchmarks</a> •
  <a href="#examples">Examples</a> •
  <a href="#contributing">Contributing</a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/go-1.21+-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/license-MIT-00FF00?style=for-the-badge" alt="License">
  <img src="https://img.shields.io/badge/version-0.2.0-FF69B4?style=for-the-badge" alt="Version">
  <img src="https://img.shields.io/badge/tests-86%20passing-brightgreen?style=for-the-badge" alt="Tests">
  <img src="https://img.shields.io/badge/gadgets-50-blueviolet?style=for-the-badge" alt="Gadgets">
  <img src="https://img.shields.io/badge/examples-9-orange?style=for-the-badge" alt="Examples">
</p>

---

```
 ███╗   ███╗ ██████╗ ███████╗███████╗
 ████╗ ████║██╔═══██╗██╔════╝██╔════╝
 ██╔████╔██║██║   ██║███████╗███████╗
 ██║╚██╔╝██║██║   ██║╚════██║╚════██║
 ██║ ╚═╝ ██║╚██████╔╝███████║███████║
 ╚═╝     ╚═╝ ╚═════╝ ╚══════╝╚══════╝
```

<p align="center">
  <em>Not a TUI framework. A terminal-native application runtime with a reactive visual layer.</em>
</p>

---

## Why MOFU?

| | MOFU | Bubble Tea | Ratatui | Ink |
|---|:---:|:---:|:---:|:---:|
| **Architecture** | Reactive graph | Elm loop | Immediate mode | Virtual DOM |
| **State Updates** | O(1) dirty tracking | O(N) full copy | O(N) full redraw | O(N) diff |
| **Rendering** | Incremental diff | Full redraw | Immediate | Virtual diff |
| **Streaming** | First-class | Manual | No | No |
| **Widgets** | 50 gadgets | ~15 bubbles | ~10 | ~8 |
| **Styling** | Semantic (Cuddles) | Lipgloss | Inline | StyleSheet |
| **Forms** | Schema-driven | Manual (Huh) | Manual | Manual |
| **Animation** | Spring physics | None | None | Limited |
| **Accessibility** | Built-in | None | None | None |
| **Plugin System** | Full runtime | None | None | None |
| **AI Support** | Native primitives | None | None | None |
| **License** | MIT | MIT | MIT | MIT |

---

## Quick Start

```bash
go get github.com/xanstomper/mofu
```

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

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        MOFU RUNTIME                             │
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

### Core Systems

| System | Description | Performance |
|--------|-------------|-------------|
| **Reactive State Graph** | O(1) dirty tracking, dependency-aware updates | 185ns/update |
| **Incremental Diff Renderer** | Cell-level diff, Synchronized Output (CSI 2026) | Zero flicker |
| **Scheduler Lanes** | 5 priority lanes: Realtime, Stream, Compute, Render, Background | Non-blocking |
| **Stream Engine** | First-class streaming for AI, logs, events | Zero-copy |
| **Animation Graph** | Declarative spring physics + 12 easing functions | 60fps |
| **Layout Engine** | Constraint-based layout with flex/grid support | Cached |
| **Plugin System** | Full gadget runtime with state isolation | Sandboxed |

---

## Features

### Core Runtime

- **Reactive State Graph** — O(1) dirty tracking, automatic dependency resolution
- **Incremental Diff Renderer** — Cell-level diff, only changed cells written to terminal
- **Synchronized Output** — CSI 2026 protocol for flicker-free updates
- **Spring Physics** — Damped spring animations with configurable stiffness/damping
- **Constraint Layout** — Flex, grid, and constraint-based layout engine
- **Full Input Parser** — Arrows, F-keys, Ctrl+key, Alt+key, mouse (SGR mode)

### Gadgets (50 Reactive UI Systems)

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
| ViewportManager | Visible region only |
| ResponsiveLayoutCore | Terminal-aware layouts |

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

### Cuddles (Semantic Styling)

```go
theme := cuddles.Mochi()
style := theme.Style(cuddles.Primary)  // Not #ff69b4, but "primary"
style := theme.Style(cuddles.Error)    // Theme decides the color
```

### Meow (Schema Forms)

```go
form := meow.NewForm(
    meow.Input("name", "Name").SetRequired(),
    meow.Input("email", "Email").Validate(meow.ValidateEmail),
    meow.Select("role", "Role", []string{"Admin", "User"}),
    meow.Checkbox("agree", "I agree to terms"),
)
```

---

## Benchmarks

### State Update Performance

```
BenchmarkAtomSetValue-4       7041339    185ns/op    0 allocs/op
BenchmarkCollectDirty100-4     88650   13491ns/op    1 allocs/op
BenchmarkCollectDirty1000-4    10000  105703ns/op    1 allocs/op
BenchmarkCollectDirtyNoDirty-4 21M       90ns/op    0 allocs/op
```

### Comparison with Other Frameworks

| Metric | MOFU | Bubble Tea | Ratatui |
|--------|------|------------|---------|
| State update | **185ns** | ~1000ns | ~500ns |
| Dirty tracking (1000 nodes) | **106μs** | O(N) scan | O(N) scan |
| Input parse | **<100ns** | <100ns | N/A |
| Memory per frame | **0 allocs** | 2-5 allocs | 1-3 allocs |
| Render (80x24) | **<1ms** | ~5ms | ~3ms |

### Rendering Performance

| Metric | MOFU | Others |
|--------|------|--------|
| Flicker | **Zero** (CSI 2026) | Common |
| Cells per frame | **Only changed** | Full redraw |
| ANSI sequences | **Minimal** | Heavy |
| Terminal bandwidth | **Low** | High |

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

---

## Installation

```bash
# Get MOFU
go get github.com/xanstomper/mofu

# Run an example
cd examples/counter && go run main.go
```

### Requirements

- Go 1.21 or later
- Terminal with ANSI support
- Unix-like system or Windows Terminal

---

## Project Structure

```
mofu/
├── core/              # Runtime kernel
│   ├── kernel/        # Execution engine
│   ├── state/         # Reactive graph system
│   ├── render/        # Diff + ANSI renderer
│   └── message/       # Event bus
├── gadgets/           # 50 reactive UI systems
├── cuddles/           # Semantic styling engine
├── meow/              # Schema-driven forms
├── widgets/           # Traditional widgets (15)
├── examples/          # 9 example applications
├── primitives/        # Low-level adapters
├── effect/            # Effect system
├── plugin/            # Plugin runtime
└── cmd/mofu/          # CLI tool
```

---

## Testing

```bash
# Run all tests
go test ./...

# Run benchmarks
go test -bench=. -benchmem ./...

# Run specific test
go test -v ./gadgets/...
```

**86 tests passing** across all packages.

---

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

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
