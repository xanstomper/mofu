# Tutorial 1: Build a Real-Time Log Monitor

This tutorial builds a production-quality log monitor from scratch using MOFU. You'll learn the core API, event handling, and real-time data display.

## What We're Building

A log monitor that watches files, displays colored log output, supports filtering by level, and auto-scrolls to the latest entries.

## Step 1: Skeleton

```go
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/xanstomper/mofu"
)

type LogMonitor struct {
	mofu.Minimal
	lines    []LogEntry
	scrollY  int
	level    string
	width    int
	height   int
}

type LogEntry struct {
	Timestamp time.Time
	Level     string
	Message   string
}

func main() {
	app := &LogMonitor{}
	if err := mofu.Run(app); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
```

Every MOFU app embeds `mofu.Minimal` and implements two methods: `Render` and `HandleEvent`.

## Step 2: Rendering

The `Render` method writes directly to the terminal — no string building, no View functions:

```go
func (m *LogMonitor) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	m.width = r.Width
	m.height = r.Height

	// Header
	titleStyle := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
	ctx.Renderer.WriteString(" Log Monitor", r.X, r.Y, titleStyle.Foreground, titleStyle.Background, titleStyle.Attrs)
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, r.Y+1, mofu.Hex("444444"), mofu.ColorBlack, 0)

	// Log entries
	y := r.Y + 2
	for i := m.scrollY; i < len(m.lines) && y < r.Y+r.Height-1; i++ {
		entry := m.lines[i]
		ts := entry.Timestamp.Format("15:04:05")

		color := mofu.Hex("cdd6f4")
		switch entry.Level {
		case "ERROR": color = mofu.Hex("f38ba8")
		case "WARN":  color = mofu.Hex("fab387")
		case "INFO":  color = mofu.Hex("a6e3a1")
		case "DEBUG": color = mofu.Hex("6c7086")
		}

		line := fmt.Sprintf(" %s %-6s %s", ts, entry.Level, entry.Message)
		if len(line) > r.Width-1 {
			line = line[:r.Width-4] + "..."
		}
		ctx.Renderer.WriteString(line, r.X, y, color, mofu.ColorBlack, 0)
		y++
	}

	// Status bar
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, r.Y+r.Height-2, mofu.Hex("444444"), mofu.ColorBlack, 0)
	status := fmt.Sprintf(" %d lines | Level: %s | j/k:scroll 1-4:filter c:clear q:quit", len(m.lines), m.level)
	ctx.Renderer.WriteString(status, r.X, r.Y+r.Height-1, mofu.Hex("585b70"), mofu.Hex("1e1e2e"), 0)
}
```

Key point: `ctx.Renderer.WriteString(text, x, y, fg, bg, attrs)` writes directly to the terminal. No intermediate strings, no View() methods.

## Step 3: Event Handling

`HandleEvent` receives keyboard/mouse events and returns `nil` (no side effect) or a `Cmd` (async operation):

```go
func (m *LogMonitor) HandleEvent(event mofu.Event) mofu.Cmd {
	if event.Type != mofu.EventKeyPress {
		return nil
	}
	ke := event.Data.(mofu.KeyEvent)

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()
	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		if m.scrollY < len(m.lines)-m.height+2 {
			m.scrollY++
		}
	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if m.scrollY > 0 {
			m.scrollY--
		}
	case len(ke.Runes) > 0 && ke.Runes[0] == '1':
		m.level = ""  // show all
	case len(ke.Runes) > 0 && ke.Runes[0] == '2':
		m.level = "ERROR"
	case len(ke.Runes) > 0 && ke.Runes[0] == '3':
		m.level = "WARN"
	case len(ke.Runes) > 0 && ke.Runes[0] == '4':
		m.level = "INFO"
	case len(ke.Runes) > 0 && ke.Runes[0] == 'c':
		m.lines = nil
		m.scrollY = 0
	}
	return nil
}
```

## Step 4: Adding Data

Add a method to receive log entries. In a real app, this would be a goroutine watching files:

```go
func (m *LogMonitor) AddLine(level, message string) {
	m.lines = append(m.lines, LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
	})
	// Auto-scroll to bottom
	m.scrollY = len(m.lines) - m.height + 2
	if m.scrollY < 0 {
		m.scrollY = 0
	}
}
```

## Step 5: Running

```bash
go run main.go
```

Press `1-4` to filter by level, `j/k` to scroll, `c` to clear, `q` to quit.

## Complete Code

See `examples/logmonitor/main.go` for the full implementation with file watching.

## What You Learned

1. **Minimal embedding** — embed `mofu.Minimal` for default implementations
2. **Direct rendering** — `ctx.Renderer.WriteString()` writes to terminal
3. **Event handling** — `HandleEvent` receives keyboard events
4. **Color styling** — `mofu.Hex("ff69b4")` for true-color, `mofu.AttrBold` for attributes
5. **Quit** — return `mofu.QuitCmd()` from HandleEvent to exit
