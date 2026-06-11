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

## Getting Started (3 lines)

```go
package main

import "github.com/xanstomper/mofu"

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

## Full Demo

Run the interactive showcase with live-updating dashboard, widget demos, todo list, and more:

```bash
cd demo && go run .
```

Features: tabbed interface, live metrics, process list, log stream, todo tracker, widget showcase.

## Why MOFU?

MOFU is not another TUI framework. It's a **reactive terminal runtime** built for the age of AI agents and streaming data.

### vs Bubble Tea / Ratatui / OpenTUI

| Feature | MOFU | Bubble Tea | Ratatui | OpenTUI |
|---------|------|------------|---------|---------|
| **Architecture** | Reactive graph + diff | Elm loop + full rebuild | Immediate mode | React-like |
| **Render model** | Cell-level differential | Full string rebuild | Full buffer copy | Virtual DOM |
| **Allocations/frame** | 0 (hot path) | N (string concat) | N (Vec growth) | N |
| **Input latency** | <1ms (batched) | Per-keystroke | Per-keystroke | Per-keystroke |
| **Streaming support** | Built-in SSE + ring buffer | Manual | None | Manual |
| **AI agent display** | Native (agent/) | None | None | Basic |
| **Gadgets** | 112 production-ready | 0 (manual) | 0 (manual) | 0 |
| **Virtual scroll** | O(1) for millions of lines | None | Optional | None |
| **Multi-agent** | Tab orchestration | None | None | None |
| **API streaming** | OpenAI/Anthropic/Ollama | None | None | None |
| **Cost tracking** | Built-in token/cost | None | None | None |
| **Markdown** | Terminal-native renderer | None | None | None |

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
│  gadgets/   → 112 UI components (tables, charts, │
│                forms, monitors, dev tools)       │
│  agent/     → AI workflow display (streaming,    │
│                tool calls, multi-agent, SSE)     │
│  cuddles/   → Semantic themes (Mochi, Catppuccin,│
│                Tokyo Night)                      │
│  meow/      → Schema-driven forms               │
│  stream/    → Reactive streams                  │
│  render/    → Diff renderer, scene buffer        │
│  state/     → Reactive state graph              │
│  kernel/    → Event loop, input parsing          │
└─────────────────────────────────────────────────┘
```

## Gadgets (112)

All gadgets have real functionality — mutex-protected state, data manipulation, event handling, styled rendering. No thin wrappers.

**Data & Visualization**: HeatMap, Sparkline, ProgressBar, Donut, Gauge, Timer, PieChart, MiniMap, BoxPlot, RadarChart, WaterfallChart, FunnelChart, TreemapChart, HeatCalendar, DotPlot, Candlestick

**Dev Tools**: APIClient, ProcessViewer, PortScanner, GitBranches, GitLog, FileExplorer, DiffViewer, HexViewer, CodeBlock, EnvConfig, CronScheduler, AICodeReview, DependencyGraph, JSONViewer

**System**: SystemMonitor, DiskUsage, NetworkStats, ServiceHealth, IncidentTracker, DeploymentTracker, AuditLog, LogAggregator, ResourceMonitor, AlertBanner

**Interactive**: CRUDTable, SearchBox, DropDown, QueryBuilder, FormField, FeatureFlags, ToolPanel, PipelineRunner, DBSchema

**Display**: MarkdownPreview, SyntaxHighlighter, StatusPage, KeyValueEditor, LogFilter, Accordion, Tabs, Breadcrumb, Badge, Toast, NotificationPanel

## Agent Package

Built for AI agent workflows. Streaming, tool calls, cost tracking, multi-agent orchestration.

```go
agent := agent.NewInstantAgent("my-agent", apiURL, apiKey, model)
agent.SetSystemPrompt("You are helpful.")
agent.OnToken(func(token string) { /* instant render */ })
agent.Send("What is the capital of France?")
```

Components: `Agent`, `InstantAgent`, `APIStream`, `ToolPanel`, `CostBar`, `VirtualScroll`, `MarkdownRenderer`, `Orchestrator`, `EventTimeline`, `AgentDashboard`, `WorkflowView`, `StreamDisplay`

## Examples (23)

| App | Description | Features Used |
|-----|-------------|---------------|
| **counter** | Minimal counter | Core API, events |
| **dashboard** | Multi-panel dashboard | Layout, panels |
| **chat** | Chat interface | Input widget, messages |
| **filemanager** | Directory browser | File tree, navigation |
| **form** | Registration form | Input, checkbox, button |
| **settings** | Settings panel | Select, checkbox |
| **logviewer** | Log filtering | Virtual scroll, search |
| **wizard** | Setup wizard | Multi-step flow |
| **monitor** | System monitor | Metrics, sparklines |
| **gitui** | Git interface | Branches, diff |
| **dockerui** | Docker dashboard | Containers, status |
| **kanban** | Kanban board | Drag columns |
| **calculator** | Calculator | Input, math |
| **taskmanager** | Task CRUD | Table, filter, sort |
| **markdown** | Markdown viewer | Parse, scroll |
| **csvviewer** | CSV browser | Sort, filter, select |
| **email** | Email client | Folders, preview |
| **stocktracker** | Stock tracker | Sparklines, data |
| **musicplayer** | Music player | Playlists, controls |
| **notepad** | Multi-tab editor | Tabs, text input |
| **pomodoro** | Pomodoro timer | Timer, sessions |
| **budget** | Budget tracker | Categories, bars |
| **aiworkflow** | AI agent display | agent/ package |

## Quick Start

```bash
# Install
go get github.com/xanstomper/mofu

# Run an example
cd examples/counter && go run main.go
```

## API

```go
// Embed Minimal for default implementations — just add Render + HandleEvent
type myApp struct {
    mofu.Minimal
    count int
}

func (a *myApp) Render(ctx *mofu.RenderContext) {
    // Write directly to terminal — zero intermediate strings
    ctx.Renderer.WriteString("...", x, y, fg, bg, attrs)
}

func (a *myApp) HandleEvent(e mofu.Event) mofu.Cmd {
    // Handle keyboard/mouse, return Cmd for async ops
    return nil
}

// Run it
mofu.Run(&myApp{})
```

## License

MIT
