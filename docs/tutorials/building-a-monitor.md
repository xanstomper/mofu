# Tutorial: Building a System Monitor

Build a real-time system monitor with MOFU.

## Step 1: Setup

```bash
mkdir monitor && cd monitor
go mod init monitor
go get github.com/xanstomper/mofu
```

## Step 2: Create the App

```go
package main

import (
    "fmt"
    "os"
    "strings"
    "time"
    "github.com/xanstomper/mofu"
)

type Monitor struct {
    mofu.Minimal
    cpu, mem, disk, net float64
    uptime              time.Duration
    start               time.Time
    width, height       int
}

func NewMonitor() *Monitor {
    return &Monitor{start: time.Now()}
}
```

## Step 3: Implement Render

```go
func (m *Monitor) Render(ctx *mofu.RenderContext) {
    r := ctx.Bounds
    m.width = r.Width
    m.height = r.Height
    m.uptime = time.Since(m.start)

    // Simulate changing values
    m.cpu = 20 + 30*float64(time.Since(m.start).Milliseconds()%1000)/1000
    m.mem = 40 + 10*float64(time.Since(m.start).Milliseconds()%2000)/2000
    m.net = 10 + 20*float64(time.Since(m.start).Milliseconds()%3000)/3000

    // Title
    titleStyle := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
    ctx.Renderer.WriteString(" System Monitor", r.X, r.Y, titleStyle.Foreground, titleStyle.Background, titleStyle.Attrs)

    sep := strings.Repeat("─", r.Width-2)
    ctx.Renderer.WriteString(sep, r.X+1, r.Y+1, mofu.Hex("444444"), mofu.ColorBlack, 0)

    y := r.Y + 3
    labelStyle := mofu.DefaultStyle().Fg(mofu.Hex("89b4fa"))

    // CPU
    ctx.Renderer.WriteString("CPU Usage", r.X+2, y, labelStyle.Foreground, labelStyle.Background, labelStyle.Attrs)
    y++
    m.renderBar(ctx, r.X+2, y, r.Width-4, m.cpu/100)
    y += 2

    // Memory
    ctx.Renderer.WriteString("Memory", r.X+2, y, labelStyle.Foreground, labelStyle.Background, labelStyle.Attrs)
    y++
    m.renderBar(ctx, r.X+2, y, r.Width-4, m.mem/100)
    y += 2

    // Disk
    ctx.Renderer.WriteString("Disk", r.X+2, y, labelStyle.Foreground, labelStyle.Background, labelStyle.Attrs)
    y++
    m.renderBar(ctx, r.X+2, y, r.Width-4, 0.55)
    y += 2

    // Network
    ctx.Renderer.WriteString("Network", r.X+2, y, labelStyle.Foreground, labelStyle.Background, labelStyle.Attrs)
    y++
    m.renderBar(ctx, r.X+2, y, r.Width-4, m.net/100)
    y += 2

    // Stats
    ctx.Renderer.WriteString(fmt.Sprintf("Uptime: %s", m.uptime.Round(time.Second)), r.X+2, y, mofu.Hex("e0e0e0"), mofu.ColorBlack, 0)

    ctx.Renderer.WriteString(" q: Quit ", r.X, r.Y+r.Height-1, mofu.Hex("666666"), mofu.ColorBlack, 0)
}

func (m *Monitor) renderBar(ctx *mofu.RenderContext, x, y, width int, pct float64) {
    if pct < 0 { pct = 0 }
    if pct > 1 { pct = 1 }

    filled := int(pct * float64(width))
    empty := width - filled

    bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

    color := "a6e3a1"
    if pct > 0.8 {
        color = "f38ba8"
    } else if pct > 0.6 {
        color = "f9e2af"
    }

    ctx.Renderer.WriteString(bar, x, y, mofu.Hex(color), mofu.ColorBlack, 0)

    pctStr := fmt.Sprintf(" %.0f%%", pct*100)
    ctx.Renderer.WriteString(pctStr, x+width-len(pctStr), y, mofu.Hex("666666"), mofu.ColorBlack, 0)
}
```

## Step 4: Handle Events

```go
func (m *Monitor) HandleEvent(event mofu.Event) mofu.Cmd {
    if event.Type != mofu.EventKeyPress {
        return nil
    }
    ke := event.Data.(mofu.KeyEvent)
    if ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q') {
        return mofu.QuitCmd()
    }
    return nil
}
```

## Step 5: Run

```go
func main() {
    app := NewMonitor()
    if err := mofu.Run(app); err != nil {
        fmt.Fprintf(os.Stderr, "error: %v\n", err)
        os.Exit(1)
    }
}
```

## What You Learned

- Creating real-time updating displays
- Rendering progress bars with colors
- Handling percentage-based metrics
- Formatting time durations
- Color-coding based on thresholds

## Next Steps

- Read actual system metrics (using runtime package)
- Add process list
- Add network connections
- Add disk I/O
- Add historical charts
