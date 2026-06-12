<p align="center">
  <img src="banner.png" alt="MOFU — ターミナルの、その先へ。" width="100%">
</p>

<p align="center">
  <strong>The reactive terminal UI framework & runtime for Go</strong><br>
  <em>Build beautiful, animated, streaming terminal apps with zero-allocation rendering.</em><br>
  <em>Complete TUI framework — not just a runtime.</em>
</p>

<p align="center">
  <a href="https://pkg.go.dev/github.com/xanstomper/mofu"><img src="https://img.shields.io/badge/pkg.go.dev-reference-007d9c?style=flat-square&logo=go&logoColor=white" alt="pkg.go.dev"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-00ff00?style=flat-square" alt="MIT License"></a>
  <a href="https://github.com/xanstomper/mofu/releases"><img src="https://img.shields.io/badge/version-1.0.0-ff69b4?style=flat-square" alt="v1.0.0"></a>
  <a href="#benchmarks"><img src="https://img.shields.io/badge/perf-0%20allocs%20hot%20path-brightgreen?style=flat-square" alt="Zero Allocs"></a>
  <img src="https://img.shields.io/badge/tests-207%2B_passing-00dd00?style=flat-square" alt="Tests">
  <img src="https://img.shields.io/badge/framework-complete-ff69b4?style=flat-square" alt="Framework">
  <img src="https://img.shields.io/badge/widgets-6%20core%20%2B%2018%20basic-blueviolet?style=flat-square" alt="Widgets">
  <img src="https://img.shields.io/badge/examples-24-orange?style=flat-square" alt="Examples">
  <a href="https://github.com/xanstomper/mofu"><img src="https://img.shields.io/github/stars/xanstomper/mofu?style=flat-square&color=yellow" alt="Stars"></a>
</p>

---

## Quick Start

```bash
go get github.com/xanstomper/mofu@latest
```

```go
package main

import (
    "fmt"
    "github.com/xanstomper/mofu"
)

type counter struct {
    mofu.Minimal
    count int
}

func (c *counter) Render(ctx *mofu.RenderContext) {
    ctx.Renderer.WriteString(
        fmt.Sprintf("  Count: %d   (↑/↓ change · q quit)  ", c.count),
        0, 0, mofu.Hex("cdd6f4"), mofu.Hex("1e1e2e"), 0,
    )
}

func (c *counter) HandleEvent(e mofu.Event) mofu.Cmd {
    if e.Type != mofu.EventKeyPress { return nil }
    switch ke := e.Data.(mofu.KeyEvent); ke.Key {
    case mofu.KeyUp:   c.count++
    case mofu.KeyDown: c.count--
    case mofu.KeyEsc:  return mofu.QuitCmd()
    }
    return nil
}

func main() { mofu.Run(&counter{}) }
```

```
$ go run main.go
  Count: 42   (↑/↓ change · q quit)
```

---

## Why MOFU?

MOFU is not another TUI framework. It is a **reactive terminal runtime** — a cell-level diff renderer backed by a dirty-bit state graph, purpose-built for AI agents, streaming data, and apps that need to feel alive.

### Architecture

```
┌───────────────────────────────────────────────────────────────┐
│                        MOFU Runtime                           │
│                                                               │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────────┐   │
│  │   Kernel     │  │  State Graph │  │   Diff Renderer    │   │
│  │  input →     │──│  dirty-bit   │──│   cell-level        │   │
│  │  state →     │  │  DAG prop.   │  │   SGR cache         │   │
│  │  render      │  │              │  │   zero-alloc flush  │   │
│  └─────────────┘  └──────────────┘  └────────────────────┘   │
│                                                               │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────────┐   │
│  │  Animator    │  │  Event Bus   │  │  Layout Engine      │   │
│  │  tweens      │──│  1ms batch   │──│  flex + cache       │   │
│  │  springs     │  │  Ctrl+key    │  │  dirty-wash         │   │
│  │  groups      │  │  mouse SGR   │  │                     │   │
│  └─────────────┘  └──────────────┘  └────────────────────┘   │
└───────────────────────────────────────────────────────────────┘
         │               │                    │
         ▼               ▼                    ▼
┌───────────────────────────────────────────────────────────────┐
│                     Package Ecosystem                         │
│                                                               │
│  gadgets/   112 production-ready UI components                │
│  widgets/   18 focused UI primitives                          │
│  agent/     AI workflow display framework                     │
│  cuddles/   Semantic themes (Mochi · Sakura · Catppuccin)     │
│  meow/      Schema-driven forms with validators               │
│  render/    Cell-level diff renderer, scene buffer             │
│  state/     Reactive state graph with DAG propagation         │
│  kernel/    Event loop, input parsing, scheduling             │
│  message/   Type-safe pub/sub message bus                     │
│  effect/    Async effect dispatch for plugins & IO             │
│  ascii/     ASCII art scene rendering                         │
└───────────────────────────────────────────────────────────────┘
```

---

## Features

| | Feature | Details |
|--|---------|---------|
| **Rendering** | Cell-level diff | Only changed cells written to terminal — not full redraws |
| | Zero allocations | Hot path allocates nothing — preallocated framebuffers + SGR cache |
| | TrueColor | Full 24-bit RGB with ANSI 256 fallback |
| | Synchronized output | CSI 2026 sync — no flicker on supported terminals |
| **Animation** | 16 easing functions | Quad, cubic, bounce, elastic, back, expo, quint |
| | Spring physics | Damped springs with stiffness/damping/mass |
| | Animation groups | Parallel + sequence + stagger composition |
| | Enter/exit transitions | Declarative widget lifecycle animations |
| **Input** | Custom parser | No VT500 dependency — handles CSI, SS3, mouse SGR, bracketed paste |
| | 1ms batch window | Coalesces rapid keystrokes — fewer renders, same latency |
| | Full key coverage | Arrows, F1-F12, Ctrl+A-Z, Alt+key, mouse drag, scroll |
| **State** | Reactive graph | Dirty-bit DAG propagation — no full recomputation |
| | Computed values | Derived state that auto-updates when dependencies change |
| | Stream-first | All inputs are streams — no global Update() loop |
| **Layout** | Flexbox model | Row, column, flex-grow, flex-shrink, alignment, gaps |
| | Layout cache | Skips layout when width/height/stateHash unchanged |
| | Auto-sizing | Min/max constraints with overflow control |
| **Theming** | 3 built-in themes | Catppuccin Mocha · Mochi · Sakura |
| | Semantic tokens | Primary, secondary, accent, success, warning, error, info, muted |
| | Widget themes | Per-widget normal/hover/pressed/disabled/focused states |

---

## Performance

```
BenchmarkRingBufferWrite1K      90 ns/op    0 B/op    0 allocs
BenchmarkRingBufferRead1K      126 ns/op    0 B/op    0 allocs
BenchmarkVirtualScrollScroll    70 ns/op    0 B/op    0 allocs
BenchmarkVirtualScrollAppend   349 ns/op    0 B/op    0 allocs
BenchmarkStreamingBuffer       123 ns/op    0 B/op    0 allocs
BenchmarkSSEParser               3 µs/op    4 B/op    4 allocs
```

Run benchmarks yourself:

```bash
go test -bench=. -benchmem ./render/... ./state/... ./message/... ./kernel/...
```

---

## Themes

```
┌─────────────────────────────────────────────────────────────┐
│  Catppuccin Mocha (default)                                 │
│  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━  │
│  bg: #1e1e2e  surface: #313244  text: #cdd6f4               │
│  primary: #89b4fa  accent: #f5c2e7  success: #a6e3a1        │
│  error: #f38ba8  warning: #f9e2af  info: #7dcfff             │
├─────────────────────────────────────────────────────────────┤
│  Mochi                                                       │
│  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━  │
│  bg: #0a0a0a  surface: #1a1a2e  text: #e0e0e0               │
│  primary: #ff69b4  accent: #ff1493  success: #00ff88         │
│  error: #ff3355  warning: #ffaa00  info: #3399ff             │
├─────────────────────────────────────────────────────────────┤
│  Sakura                                                      │
│  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━  │
│  bg: #1a1020  surface: #2a1a30  text: #f0d0e0               │
│  primary: #ffb7d5  accent: #ff69b4  success: #a0f0c0         │
│  error: #ff6080  warning: #ffd080  info: #a0d0ff             │
└─────────────────────────────────────────────────────────────┘
```

---

## Examples (24)

| App | Description |
|-----|-------------|
| **counter** | Minimal counter — starter template |
| **dashboard** | Multi-panel system dashboard |
| **chat** | Chat interface with messages |
| **email** | Email client with folders and preview |
| **filemanager** | Directory browser with tree navigation |
| **form** | Registration form with validation |
| **settings** | Settings panel with toggles |
| **logviewer** | Log filtering and search |
| **logmonitor** | Real-time log file watcher |
| **wizard** | Setup wizard with steps |
| **monitor** | System metrics with sparklines |
| **gitui** | Git interface (branches, diff) |
| **dockerui** | Docker container dashboard |
| **kanban** | Kanban board |
| **taskmanager** | Task CRUD with filter/sort |
| **markdown** | Markdown viewer with scroll |
| **csvviewer** | CSV browser with sort/filter |
| **stocktracker** | Stock tracker with sparklines |
| **musicplayer** | Music player with playlists |
| **notepad** | Multi-tab text editor |
| **pomodoro** | Pomodoro timer with sessions |
| **budget** | Budget tracker with categories |
| **aiworkflow** | AI agent workflow display |
| **notepad** | Multi-tab text editor |

Run any example:

```bash
cd examples/dashboard && go run .
```

---

## Gadgets (112)

All gadgets are **real product-building tools** — mutex-protected state, data manipulation methods, event handling, and styled rendering. No thin wrappers.

### Data & Visualization

```
  HeatMap          Sparkline        ProgressBar      Donut
  Gauge            Timer            PieChart         MiniMap
  BoxPlot          RadarChart       WaterfallChart   FunnelChart
  TreemapChart     HeatCalendar     DotPlot          Candlestick
```

### Dev Tools

```
  APIClient        ProcessViewer    PortScanner      GitBranches
  GitLog           FileExplorer     DiffViewer       HexViewer
  CodeBlock        EnvConfig        CronScheduler    AICodeReview
  DependencyGraph  JSONViewer
```

### System & Monitoring

```
  SystemMonitor    DiskUsage        NetworkStats     ServiceHealth
  IncidentTracker  DeploymentTracker  AuditLog       LogAggregator
  ResourceMonitor  AlertBanner
```

### Interactive

```
  CRUDTable        SearchBox        DropDown         QueryBuilder
  FormField        FeatureFlags     ToolPanel        PipelineRunner
  DBSchema
```

### Display & Text

```
  MarkdownPreview  SyntaxHighlighter  StatusPage      KeyValueEditor
  LogFilter        Accordion          Tabs            Breadcrumb
  Badge            Toast              NotificationPanel  WordCounter
  TextTransform    ProgressBarSteps   ProgressBarAnimated
```

### AI & Agent

```
  DiffViewerPro    JSONViewer        StatusPage       MarkdownPreview
  AICodeReview     DependencyGraph   MetricGauge      FileWatcher
  StreamDisplay    AgentDashboard
```

### Terminal Tools

```
  TerminalOutput   ProgressBarDual   TimelineCompact  KeyValueEditor
  LogFilter        AsciiTable        DonutChart       GitLog
  SSHSession       NetworkPing
```

---

## Agent Package

Built for AI agent workflows — streaming, tool calls, cost tracking, multi-agent orchestration.

```go
a := agent.NewInstantAgent("my-agent", apiURL, apiKey, model)
a.SetSystemPrompt("You are a helpful assistant.")

a.OnToken(func(token string) {
    // renders instantly to terminal
})

a.Send("Explain this code")

a.RegisterTool("bash", func(input string) (string, error) {
    return exec.Command("bash", "-c", input).Output()
})
```

| Component | Purpose |
|-----------|---------|
| `Agent` | Core state machine with tool calls, streaming, thinking |
| `InstantAgent` | Production agent with live API streaming |
| `APIStream` | HTTP client for OpenAI/Anthropic/Ollama SSE |
| `ToolPanel` | Side panel showing active/completed tool calls |
| `CostBar` | Token usage and cost tracking |
| `VirtualScroll` | O(1) scroll through millions of lines |
| `MarkdownRenderer` | Terminal-native markdown rendering |
| `Orchestrator` | Multi-agent tab display |
| `EventTimeline` | Chronological event log with filtering |
| `AgentDashboard` | Full-screen monitoring dashboard |
| `WorkflowView` | Complete multi-panel layout |
| `StreamDisplay` | Instant terminal rendering of streamed tokens |

---

## Widgets (18)

| Widget | Description |
|--------|-------------|
| `Input` | Text input with cursor, validator, password mode |
| `Button` | Clickable button with label and callback |
| `List` | Navigable list with items |
| `Table` | Data table with columns and rows |
| `Select` | Dropdown select with options |
| `Checkbox` | Toggle checkbox |
| `Modal` | Modal dialog overlay |
| `Tabs` | Tab bar with active state |
| `Toast` | Temporary notification popup |
| `Tooltip` | Hover tooltip |
| `Tree` | Hierarchical tree view |
| `Viewport` | Scrollable viewport |
| `Text` | Styled text display |
| `ProgressBar` | Progress indicator |
| `Menu` | Menu with items |

---

## Framework Components

MOFU is a **complete TUI framework** — not just a runtime. Every feature you need to build production terminal apps is built in.

### Program Options

```go
p := mofu.New(model,
    mofu.WithAltScreen(),          // alternate screen buffer
    mofu.WithMouseCellMotion(),    // SGR mouse tracking
    mofu.WithBracketedPaste(),     // bracketed paste mode
    mofu.WithSyncOutput(),         // CSI 2026 synchronized output (no flicker)
    mofu.WithReportFocus(),        // focus in/out events
    mofu.WithFPS(60),              // frame rate cap
    mofu.WithTheme(mofu.MochiTheme()),
    mofu.WithMiddleware(mw1, mw2), // event middleware chain
    mofu.WithEventFilter(func(e Event) Event { return e }),
)
p.Run()
```

### Key Bindings

Declarative key maps with contextual help display:

```go
km := mofu.NewKeyMap()
km.Set("up", mofu.NewBinding(mofu.KeyUp, mofu.HelpText{Key: "↑", Desc: "up"}))
km.Set("down", mofu.NewBinding(mofu.KeyDown, mofu.HelpText{Key: "↓", Desc: "down"}))
km.Set("quit", mofu.NewBinding(mofu.KeyEsc, mofu.HelpText{Key: "esc", Desc: "quit"}))

name, ok := km.Matches(event)  // returns binding name if matched
help := km.Help()               // formatted help text
short := km.ShortHelp()         // first 3 bindings
full := km.FullHelp()           // all bindings grouped
```

### Middleware

Composable event filter chain:

```go
func myMiddleware(next mofu.EventFilter) mofu.EventFilter {
    return func(ev mofu.Event) mofu.Event {
        // pre-process event
        ev = next(ev)
        // post-process event
        return ev
    }
}

chain := mofu.Chain(mw1, mw2, mw3)
```

Built-in middleware: `ColorProfileMiddleware`, `FPSMiddleware`, `PasteFilterMiddleware`, `FocusMiddleware`.

### Core Widgets (6)

Production-grade widgets built into the framework:

| Widget | Description |
|--------|-------------|
| `Spinner` | 8 styles (Dot, Line, Dot2, Minidot, Pulse, Globe, Monkey, Points), start/stop/pause |
| `Progress` | 4 modes (Bar, Dots, Spinner, Percent), Incr/Set, percentage tracking |
| `Viewport` | Scrollable content with keyboard nav, GotoTop/Bottom, HalfPage, scroll percentage |
| `Textarea` | Multi-line text input with cursor, insert/delete, line navigation |
| `List` | Filterable list with delegate pattern, selection, pagination |
| `Table` | Sortable table with columns, alignment, header styling, selection |

```go
// Spinner
spinner := mofu.NewSpinner(mofu.SpinnerDot)
spinner.Title("Loading...")
spinner.Start()

// Progress
bar := mofu.NewProgress(100, 40)
bar.Set(75)      // or bar.Incr(10)
fmt.Println(bar.Render())  // ████████████████████░░░░░░░░ 75%

// Viewport
vp := mofu.NewViewport(80, 20)
vp.SetContent(longString)
vp.ScrollDown(5)

// Textarea
ta := mofu.NewTextarea()
ta.SetPlaceholder("Type here...")
ta.Focus()

// List
items := []mofu.ListItem{myItem1, myItem2}
l := mofu.NewList(items)
l.SetSize(40, 10)
l.OnSelect(func(i int, item mofu.ListItem) { /* selected */ })

// Table
cols := []mofu.TableColumn{{Title: "Name", Width: 20}, {Title: "Age", Width: 10}}
rows := [][]string{{"Alice", "30"}, {"Bob", "25"}}
t := mofu.NewTable(cols, rows)
t.SetSize(80, 20)
t.SortBy(1, true)  // sort by age ascending
```

---

## Style API

Fluent builder pattern for composable styles:

```go
style := mofu.DefaultStyle().
    Fg(mofu.Hex("cdd6f4")).
    Bg(mofu.Hex("1e1e2e")).
    Bold().
    WithRoundedBorder().
    PaddingHorizontal(2).
    MarginVertical(1).
    AlignCenter()
```

### Color utilities

```go
pink := mofu.Hex("ff69b4")
light := pink.Lighten(0.3)      // blend toward white
dark := pink.Darken(0.2)         // blend toward black
mixed := mofu.Blend(a, b, 0.5)  // 50/50 mix
lerped := mofu.Lerp(a, b, t)    // parametric interpolation
saturated := pink.Saturate(0.5) // boost saturation
```

### Animation API

```go
anim := mofu.NewAnimation(
    mofu.QuickSpec(300*time.Millisecond, mofu.EaseOutBack),
    0, 100,
)
anim.OnChange(func(v float64) {
    // v goes from 0 → 100 with overshoot easing
})

spring := mofu.NewSpring(0)
spring.SetTarget(100)
// spring physics auto-advance each tick

group := mofu.Parallel(anim1, anim2, anim3)
stagger := mofu.Stagger(spec, 50*time.Millisecond, fromTos)
```

---

## Architecture Comparison

| | MOFU | Elm-style (Bubble Tea) | Immediate-mode |
|--|------|----------------------|----------------|
| **Type** | Full TUI framework + runtime | Runtime only | Runtime only |
| **Render model** | Cell-level differential | Full string rebuild | Full buffer copy |
| **Allocs/frame** | 0 (hot path) | N (string concat) | N (Vec growth) |
| **State** | Reactive graph + dirty bits | Manual Msg returning | Global mutable |
| **Input latency** | <1ms (1ms batch) | Per-keystroke | Per-keystroke |
| **Layout** | Flexbox with cache | Manual positioning | Immediate |
| **Streaming** | Built-in SSE + ring buffer | Manual | None |
| **Animation** | Built-in tweens + springs + 16 easings | Manual | Manual |
| **Key bindings** | Declarative KeyMap + help | Manual | Manual |
| **Middleware** | EventMiddleware chain | None | None |
| **Program options** | AltScreen, mouse, paste, sync, focus | Basic | Basic |
| **Built-in widgets** | Spinner, Progress, Viewport, Textarea, List, Table | None (separate pkg) | None |
| **AI agent display** | Native `agent/` package | None | None |
| **Virtual scroll** | O(1) for millions of lines | None | Optional |
| **Themes** | 3 built-in + semantic tokens | Manual | Manual |

---

## Packages

| Package | Import | Description |
|---------|--------|-------------|
| `mofu` | `github.com/xanstomper/mofu` | Core runtime — kernel, state graph, renderer, input, events, layout, themes |
| `agent` | `github.com/xanstomper/mofu/agent` | AI agent display — API streaming, tool calls, virtual scroll, orchestration |
| `gadgets` | `github.com/xanstomper/mofu/gadgets` | 112 production-ready UI components |
| `widgets` | `github.com/xanstomper/mofu/widgets` | 18 focused UI primitives |
| `cuddles` | `github.com/xanstomper/mofu/cuddles` | Semantic themes — Mochi, Sakura, Catppuccin |
| `meow` | `github.com/xanstomper/mofu/meow` | Schema-driven forms with validators and computed fields |
| `kernel` | `github.com/xanstomper/mofu/kernel` | Event loop, input parsing, scheduling |
| `state` | `github.com/xanstomper/mofu/state` | Reactive state graph with dirty-bit DAG propagation |
| `render` | `github.com/xanstomper/mofu/render` | Diff renderer with preallocated framebuffer and SGR cache |
| `message` | `github.com/xanstomper/mofu/message` | Type-safe pub/sub message bus |
| `effect` | `github.com/xanstomper/mofu/effect` | Async effect dispatch for plugins and IO |
| `ascii` | `github.com/xanstomper/mofu/ascii` | ASCII art scene rendering |

---

## Tutorials

| Tutorial | Description |
|----------|-------------|
| [Log Monitor](tutorials/01-log-monitor.md) | Build a real-time log monitor from scratch |
| [AI Agent Display](tutorials/02-ai-agent-display.md) | Connect to an API and stream responses |
| [Data Dashboard](tutorials/03-data-dashboard.md) | Compose gadgets into a live dashboard |

---

## Documentation

| Guide | Description |
|-------|-------------|
| [Architecture](docs/guides/architecture.md) | MOFU's reactive graph architecture |
| [Getting Started](docs/guides/getting-started.md) | First steps tutorial |
| [Styling](docs/guides/styling.md) | Colors, themes, and attributes |
| [Gadgets](docs/guides/gadgets.md) | Using the 112 gadget library |
| [Forms](docs/guides/forms.md) | Building forms with Meow |
| [Testing](docs/guides/testing.md) | Testing MOFU applications |
| [Performance](docs/guides/performance.md) | Optimization guide |

---

<p align="center">
  <sub>Built with care by <a href="https://github.com/xanstomper">xanstomper</a> · MIT License</sub>
</p>
