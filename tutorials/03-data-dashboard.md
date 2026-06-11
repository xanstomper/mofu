# Tutorial 3: Build a Data Dashboard with Gadgets

This tutorial builds a live dashboard using MOFU's gadget library. You'll learn how to compose gadgets into a complete application.

## What We're Building

A system dashboard with CPU/memory gauges, a process list, network stats, and a live log stream — all updating in real-time.

## Step 1: Set Up the App

```go
package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/xanstomper/mofu"
	"github.com/xanstomper/mofu/gadgets"
)

type Dashboard struct {
	mofu.Minimal
	cpu      *gadgets.RealMetricGauge
	mem      *gadgets.RealMetricGauge
	disk     *gadgets.RealResourceMonitor
	logs     *gadgets.RealLogStream
	proclist *gadgets.RealProcessList
	width    int
	height   int
}

func NewDashboard() *Dashboard {
	return &Dashboard{
		cpu:      gadgets.NewRealMetricGauge("cpu", "CPU Usage", "%", 100),
		mem:      gadgets.NewRealMetricGauge("mem", "Memory Usage", "%", 100),
		disk:     gadgets.NewRealResourceMonitor("disk"),
		logs:     gadgets.NewRealLogStream("logs"),
		proclist: gadgets.NewRealProcessList("procs"),
	}
}
```

## Step 2: Initialize Gadgets

```go
func (d *Dashboard) init() {
	// Set initial values
	d.cpu.SetValue(23.5)
	d.mem.SetValue(67.2)

	d.disk.Set("Root", 45, 100, "GB", mofu.Hex("89b4fa"))
	d.disk.Set("Data", 180, 500, "GB", mofu.Hex("a6e3a1"))
	d.disk.Set("Backup", 320, 500, "GB", mofu.Hex("fab387"))

	d.proclist.AddProcess(1234, "mofu-app", 12.5, 8.3, "R", "ben")
	d.proclist.AddProcess(5678, "postgres", 3.2, 15.6, "S", "postgres")
	d.proclist.AddProcess(9012, "nginx", 0.8, 2.1, "S", "www-data")
}
```

## Step 3: Render with Gadgets

Gadgets implement `Render(state)` — compose them into your layout:

```go
func (d *Dashboard) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	d.width = r.Width
	d.height = r.Height

	// Title
	titleStyle := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
	ctx.Renderer.WriteString(" System Dashboard", r.X, r.Y, titleStyle.Foreground, titleStyle.Background, titleStyle.Attrs)
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, r.Y+1, mofu.Hex("444444"), mofu.ColorBlack, 0)

	// CPU + Memory gauges (top row)
	gaugeW := r.Width / 2
	cpuCtx := &mofu.RenderContext{
		Bounds:   mofu.Rect{X: r.X, Y: r.Y + 2, Width: gaugeW - 1, Height: 2},
		Renderer: ctx.Renderer,
	}
	d.cpu.Render(cpuCtx)

	memCtx := &mofu.RenderContext{
		Bounds:   mofu.Rect{X: r.X + gaugeW, Y: r.Y + 2, Width: gaugeW, Height: 2},
		Renderer: ctx.Renderer,
	}
	d.mem.Render(memCtx)

	// Disk usage (middle)
	diskCtx := &mofu.RenderContext{
		Bounds:   mofu.Rect{X: r.X, Y: r.Y + 5, Width: r.Width, Height: 5},
		Renderer: ctx.Renderer,
	}
	d.disk.Render(diskCtx)

	// Process list (bottom)
	procCtx := &mofu.RenderContext{
		Bounds:   mofu.Rect{X: r.X, Y: r.Y + 11, Width: r.Width, Height: r.Height - 13},
		Renderer: ctx.Renderer,
	}
	d.proclist.Render(procCtx)
}
```

## Step 4: Simulate Live Data

In a real app, data comes from system calls. Here we simulate:

```go
func (d *Dashboard) updateData() {
	for {
		time.Sleep(2 * time.Second)

		// Simulate CPU/memory changes
		d.cpu.SetValue(20 + rand.Float64()*60)
		d.mem.SetValue(50 + rand.Float64()*40)

		// Add log entries
		levels := []string{"INFO", "WARN", "ERROR", "DEBUG"}
		messages := []string{
			"Request handled in 12ms",
			"Cache miss for key: user:1234",
			"Slow query: 2340ms",
			"Connection pool at 80%",
		}
		i := rand.Intn(len(levels))
		d.logs.AddEntry(levels[i], messages[i])
	}
}
```

## Step 5: Handle Events

Forward events to focused gadgets:

```go
func (d *Dashboard) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()
	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		d.proclist.HandleEvent(e)
	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		d.proclist.HandleEvent(e)
	}
	return nil
}

func main() {
	app := NewDashboard()
	app.init()
	go app.updateData()
	mofu.Run(app)
}
```

## Step 6: Run It

```bash
go run main.go
```

You'll see a live-updating dashboard with gauges, disk usage bars, and a process list.

## Available Gadgets

The `gadgets/` package has 112 production-ready components:

| Category | Examples |
|----------|---------|
| **Data Viz** | HeatMap, Sparkline, ProgressBar, Donut, Gauge, PieChart, BoxPlot, RadarChart |
| **Dev Tools** | APIClient, ProcessViewer, PortScanner, GitBranches, GitLog, DiffViewer |
| **System** | SystemMonitor, DiskUsage, NetworkStats, ServiceHealth, AuditLog |
| **Interactive** | CRUDTable, SearchBox, FeatureFlags, PipelineRunner, DBSchema |
| **Display** | MarkdownPreview, SyntaxHighlighter, StatusPage, Accordion, Tabs |
| **AI/Agent** | Agent, ToolPanel, CostBar, Orchestrator, EventTimeline |

All gadgets have:
- Mutex-protected state
- Data manipulation methods
- `Render()` with styled output
- `OnEvent()` for keyboard/mouse handling
- Getter methods for testing

## What You Learned

1. **Gadget composition** — compose gadgets into layouts by creating render contexts
2. **Real-time updates** — goroutines push data, MOFU re-renders changed cells
3. **Event routing** — forward events to the focused gadget
4. **Render contexts** — `mofu.Rect` defines where each gadget draws
5. **Styling** — `mofu.Hex()` for true-color, `mofu.AttrBold` for attributes
