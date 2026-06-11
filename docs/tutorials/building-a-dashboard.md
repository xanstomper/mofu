# Tutorial: Building a Real-Time Dashboard

This tutorial shows how to build a real-time system monitor with MOFU.

## Step 1: Project Setup

```bash
mkdir dashboard && cd dashboard
go mod init dashboard
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

type Dashboard struct {
    mofu.Minimal
    cpu, mem, disk float64
    width, height  int
    start          time.Time
}

func NewDashboard() *Dashboard {
    return &Dashboard{start: time.Now()}
}
```

## Step 3: Implement Render

```go
func (d *Dashboard) Render(ctx *mofu.RenderContext) {
    r := ctx.Bounds
    d.width = r.Width
    d.height = r.Height

    // Update simulated values
    d.cpu = 20 + 30*float64(time.Since(d.start).Milliseconds()%1000)/1000
    d.mem = 40 + 10*float64(time.Since(d.start).Milliseconds()%2000)/2000

    // Title
    titleStyle := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
    ctx.Renderer.WriteString(" System Monitor", r.X, r.Y, titleStyle.Foreground, titleStyle.Background, titleStyle.Attrs)

    // Separator
    sep := strings.Repeat("─", r.Width-2)
    ctx.Renderer.WriteString(sep, r.X+1, r.Y+1, mofu.Hex("444444"), mofu.ColorBlack, 0)

    // Metrics
    y := r.Y + 3
    d.renderBar(ctx, "CPU", r.X+2, y, r.Width-4, d.cpu/100)
    y += 2
    d.renderBar(ctx, "Memory", r.X+2, y, r.Width-4, d.mem/100)
    y += 2
    d.renderBar(ctx, "Disk", r.X+2, y, r.Width-4, 0.55)

    // Status
    ctx.Renderer.WriteString(" q: Quit ", r.X, r.Y+r.Height-1, mofu.Hex("666666"), mofu.ColorBlack, 0)
}

func (d *Dashboard) renderBar(ctx *mofu.RenderContext, label string, x, y, width int, pct float64) {
    labelStyle := mofu.DefaultStyle().Fg(mofu.Hex("89b4fa"))
    ctx.Renderer.WriteString(label+":", x, y, labelStyle.Foreground, labelStyle.Background, labelStyle.Attrs)

    barW := width - len(label) - 10
    filled := int(pct * float64(barW))
    empty := barW - filled

    color := "a6e3a1"
    if pct > 0.8 { color = "f38ba8" } else if pct > 0.6 { color = "f9e2af" }

    bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
    ctx.Renderer.WriteString(bar, x+len(label)+2, y, mofu.Hex(color), mofu.ColorBlack, 0)

    pctStr := fmt.Sprintf("%.0f%%", pct*100)
    ctx.Renderer.WriteString(pctStr, x+len(label)+2+barW-len(pctStr), y, mofu.Hex("666666"), mofu.ColorBlack, 0)
}
```

## Step 4: Handle Events

```go
func (d *Dashboard) HandleEvent(event mofu.Event) mofu.Cmd {
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
    if err := mofu.Run(NewDashboard()); err != nil {
        fmt.Fprintf(os.Stderr, "error: %v\n", err)
        os.Exit(1)
    }
}
```

## What You Learned

- Creating a MOFU app with `Minimal` embed
- Rendering text and progress bars
- Handling keyboard events
- Using colors and styles
- Creating a real-time updating UI

## Next Steps

- Add more metrics (network, processes)
- Implement sorting and filtering
- Add a settings panel
- Deploy as a system service
