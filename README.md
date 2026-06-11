<h1 align="center">MOFU</h1>

<p align="center">
  <strong>The Reactive Terminal Application Runtime</strong>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/go-1.21+-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/license-MIT-00FF00?style=flat-square" alt="License">
  <img src="https://img.shields.io/badge/version-0.5.0-FF69B4?style=flat-square" alt="Version">
  <img src="https://img.shields.io/badge/tests-207%20passing-brightgreen?style=flat-square" alt="Tests">
  <img src="https://img.shields.io/badge/gadgets-112-blueviolet?style=flat-square" alt="Gadgets">
  <img src="https://img.shields.io/badge/examples-25-orange?style=flat-square" alt="Examples">
</p>

---

## Quick Start

```bash
go get github.com/xanstomper/mofu
```

```go
package main

import (
    "fmt"
    "github.com/xanstomper/mofu"
)

func main() {
    mofu.Run(&counter{})
}

type counter struct {
    mofu.Minimal
    n int
}

func (c *counter) Render(ctx *mofu.RenderContext) {
    ctx.Renderer.WriteString(
        fmt.Sprintf("Count: %d  (↑/↓ to change, q to quit)", c.n),
        0, 0, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0,
    )
}

func (c *counter) HandleEvent(e mofu.Event) mofu.Cmd {
    if e.Type != mofu.EventKeyPress { return nil }
    ke := e.Data.(mofu.KeyEvent)
    switch {
    case ke.Key == mofu.KeyUp: c.n++
    case ke.Key == mofu.KeyDown: c.n--
    case ke.Key == mofu.KeyEsc: return mofu.QuitCmd()
    }
    return nil
}
```

```bash
go run main.go
```

## Demo

Run the interactive showcase:

```bash
cd demo && go run .
```

**5 tabs** with live-updating dashboard, process list, log stream, todo tracker, and widget showcase. Navigate with arrow keys, interact with each panel.

## Examples (25)

| App | Run | Description |
|-----|-----|-------------|
| **demo** | `cd demo && go run .` | Full interactive showcase |
| **counter** | `cd examples/counter && go run .` | Minimal counter — starter template |
| **dashboard** | `cd examples/dashboard && go run .` | Multi-panel system dashboard |
| **chat** | `cd examples/chat && go run .` | Chat interface with messages |
| **email** | `cd examples/email && go run .` | Email client with folders, preview |
| **filemanager** | `cd examples/filemanager && go run .` | Directory browser with tree navigation |
| **form** | `cd examples/form && go run .` | Registration form with validation |
| **settings** | `cd examples/settings && go run .` | Settings panel with toggles |
| **logviewer** | `cd examples/logviewer && go run .` | Log filtering and search |
| **logmonitor** | `cd examples/logmonitor && go run .` | Real-time log file watcher |
| **wizard** | `cd examples/wizard && go run .` | Setup wizard with steps |
| **monitor** | `cd examples/monitor && go run .` | System metrics with sparklines |
| **gitui** | `cd examples/gitui && go run .` | Git interface (branches, diff) |
| **dockerui** | `cd examples/dockerui && go run .` | Docker container dashboard |
| **kanban** | `cd examples/kanban && go run .` | Kanban board |
| **calculator** | `cd examples/calculator && go run .` | Calculator with input |
| **taskmanager** | `cd examples/taskmanager && go run .` | Task CRUD with filter/sort |
| **markdown** | `cd examples/markdown && go run .` | Markdown viewer with scroll |
| **csvviewer** | `cd examples/csvviewer && go run .` | CSV browser with sort/filter |
| **stocktracker** | `cd examples/stocktracker && go run .` | Stock tracker with sparklines |
| **musicplayer** | `cd examples/musicplayer && go run .` | Music player with playlists |
| **notepad** | `cd examples/notepad && go run .` | Multi-tab text editor |
| **pomodoro** | `cd examples/pomodoro && go run .` | Pomodoro timer with sessions |
| **budget** | `cd examples/budget && go run .` | Budget tracker with categories |
| **aiworkflow** | `cd examples/aiworkflow && go run .` | AI agent workflow display |

## Why MOFU?

MOFU is not another TUI framework. It's a **reactive terminal runtime** built for AI agents and streaming data.

### vs Bubble Tea / Ratatui / OpenTUI

| Feature | MOFU | Bubble Tea | Ratatui | OpenTUI |
|---------|------|------------|---------|---------|
| Architecture | Reactive graph + diff | Elm loop + full rebuild | Immediate mode | React-like |
| Render model | Cell-level differential | Full string rebuild | Full buffer copy | Virtual DOM |
| Allocations/frame | 0 (hot path) | N (string concat) | N (Vec growth) | N |
| Input latency | <1ms (batched) | Per-keystroke | Per-keystroke | Per-keystroke |
| Streaming | Built-in SSE + ring buffer | Manual | None | Manual |
| AI agent display | Native (`agent/`) | None | None | Basic |
| Gadgets | 112 production-ready | 0 (manual) | 0 (manual) | 0 |
| Virtual scroll | O(1) for millions of lines | None | Optional | None |
| Multi-agent | Tab orchestration | None | None | None |
| API streaming | OpenAI/Anthropic/Ollama | None | None | None |
| Cost tracking | Built-in token/cost | None | None | None |
| Markdown | Terminal-native renderer | None | None | None |

### Performance

```
RingBuffer write 1KB:    90ns   0 allocs
RingBuffer read 1KB:    126ns   0 allocs
VirtualScroll scroll:    70ns   0 allocs
VirtualScroll append:   349ns   0 allocs
StreamingBuffer:        123ns   0 allocs
SSEParser (1 event):   3.3µs   4 allocs
DiffRenderer:           cell-level differential — only changed cells written to terminal
```

## Architecture

```
┌─────────────────────────────────────────────────┐
│                  MOFU Runtime                    │
│  ┌──────────┐  ┌──────────┐  ┌──────────────┐  │
│  │  Kernel   │  │  State   │  │    Render     │  │
│  │ (input→   │  │  Graph   │  │  (diff+flush) │  │
│  │  state→   │  │ (dirty   │  │  cell-level   │  │
│  │  render)  │  │  DAG)    │  │  differential │  │
│  └──────────┘  └──────────┘  └──────────────┘  │
├─────────────────────────────────────────────────┤
│              Package Ecosystem                   │
│  gadgets/   → 112 UI components                  │
│  widgets/   → 18 basic UI primitives             │
│  agent/     → AI workflow display                │
│  cuddles/   → Semantic themes                    │
│  meow/      → Schema-driven forms                │
│  render/    → Diff renderer, scene buffer        │
│  state/     → Reactive state graph               │
│  kernel/    → Event loop, input parsing          │
└─────────────────────────────────────────────────┘
```

## Packages

| Package | Description |
|---------|-------------|
| `mofu` | Core runtime — kernel, state graph, renderer, input, events, layout |
| `agent` | AI agent display — API streaming, tool calls, virtual scroll, SSE parser, multi-agent orchestration |
| `gadgets` | 112 production-ready UI components — tables, charts, forms, monitors, dev tools |
| `widgets` | 18 basic UI primitives — Input, Button, List, Table, Select, Checkbox, Modal, Tabs, Toast, Tooltip, Tree, Viewport |
| `cuddles` | Semantic themes — Mochi, Catppuccin, Tokyo Night with dark/light variants |
| `meow` | Schema-driven forms with validators and computed fields |
| `kernel` | Event loop, input parsing (CSI, SS3, mouse SGR, Ctrl+key) |
| `state` | Reactive state graph with dirty-bit DAG propagation |
| `render` | Diff renderer with preallocated framebuffer and SGR cache |
| `message` | Type-safe message bus with pub/sub |
| `effect` | Async effect system for plugin/IO dispatch |
| `ascii` | ASCII art scene rendering |

## Gadgets (112)

All gadgets have real functionality — mutex-protected state, data manipulation, event handling, styled rendering.

**Data & Visualization (16)**
HeatMap, Sparkline, ProgressBar, Donut, Gauge, Timer, PieChart, MiniMap, BoxPlot, RadarChart, WaterfallChart, FunnelChart, TreemapChart, HeatCalendar, DotPlot, Candlestick

**Dev Tools (14)**
APIClient, ProcessViewer, PortScanner, GitBranches, GitLog, FileExplorer, DiffViewer, HexViewer, CodeBlock, EnvConfig, CronScheduler, AICodeReview, DependencyGraph, JSONViewer

**System (10)**
SystemMonitor, DiskUsage, NetworkStats, ServiceHealth, IncidentTracker, DeploymentTracker, AuditLog, LogAggregator, ResourceMonitor, AlertBanner

**Interactive (9)**
CRUDTable, SearchBox, DropDown, QueryBuilder, FormField, FeatureFlags, ToolPanel, PipelineRunner, DBSchema

**Display (15)**
MarkdownPreview, SyntaxHighlighter, StatusPage, KeyValueEditor, LogFilter, Accordion, Tabs, Breadcrumb, Badge, Toast, NotificationPanel, WordCounter, TextTransform, ProgressBarSteps, ProgressBarAnimated

**AI/Agent (10)**
DiffViewerPro, JSONViewer, StatusPage, MarkdownPreview, AICodeReview, DependencyGraph, MetricGauge, FileWatcher, StreamDisplay, AgentDashboard

**Terminal Tools (10)**
TerminalOutput, ProgressBarDual, TimelineCompact, KeyValueEditor, LogFilter, AsciiTable, DonutChart, GitLog, SSHSession, NetworkPing

**Text & Input (6)**
WordCounter, TextTransform, GrepViewer, CRUDTable, ProgressBarSteps, ProgressBarAnimated

## Agent Package

Built for AI agent workflows — streaming, tool calls, cost tracking, multi-agent orchestration.

```go
// Create an agent connected to any OpenAI-compatible API
a := agent.NewInstantAgent("my-agent", apiURL, apiKey, model)
a.SetSystemPrompt("You are a helpful assistant.")

// Stream responses token-by-token
a.OnToken(func(token string) {
    // Render instantly to terminal
})

// Send messages
a.Send("Explain this code")

// Use tools
a.RegisterTool("bash", func(input string) (string, error) {
    return exec.Command("bash", "-c", input).Output()
})
```

**Components:**

| Component | Purpose |
|-----------|---------|
| `Agent` | Core agent state machine with tool calls, streaming, thinking |
| `InstantAgent` | Production agent with live API streaming |
| `APIStream` | HTTP client for OpenAI/Anthropic/Ollama SSE endpoints |
| `ToolPanel` | Side panel showing active/completed tool calls |
| `CostBar` | Token usage and cost tracking bar |
| `VirtualScroll` | O(1) scroll through millions of log lines |
| `MarkdownRenderer` | Terminal-native markdown (headers, code blocks, lists) |
| `Orchestrator` | Multi-agent tab display |
| `EventTimeline` | Chronological event log with filtering |
| `AgentDashboard` | Full-screen monitoring dashboard |
| `WorkflowView` | Complete multi-panel layout |
| `StreamDisplay` | Instant terminal rendering of streamed tokens |

## Widgets (18)

Simple, focused UI primitives for building interactive TUIs:

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
| [Migration](docs/guides/migration-from-bubbletea.md) | Migrating from Bubble Tea |

## Tutorials

| Tutorial | Source |
|----------|--------|
| [Log Monitor](tutorials/01-log-monitor.md) | Build a real-time log monitor from scratch |
| [AI Agent Display](tutorials/02-ai-agent-display.md) | Connect to an API and stream responses |
| [Data Dashboard](tutorials/03-data-dashboard.md) | Compose gadgets into a live dashboard |

## License

MIT
