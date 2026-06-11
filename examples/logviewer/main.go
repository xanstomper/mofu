package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/xanstomper/mofu"
)

// LogViewer — a log viewer with filtering and scrolling.

type LogEntry struct {
	Time    time.Time
	Level   string
	Message string
}

type LogViewer struct {
	mofu.Minimal
	logs      []LogEntry
	Offset    int
	Selected  int
	Filter    string
	width     int
	height    int
}

func NewLogViewer() *LogViewer {
	lv := &LogViewer{
		logs: []LogEntry{
			{time.Now(), "INFO", "Application started"},
			{time.Now(), "INFO", "Connected to database"},
			{time.Now(), "WARN", "High memory usage detected"},
			{time.Now(), "INFO", "Request processed in 45ms"},
			{time.Now(), "ERROR", "Connection timeout to upstream"},
			{time.Now(), "INFO", "Retry successful"},
			{time.Now(), "DEBUG", "Cache hit for key user:123"},
			{time.Now(), "INFO", "User logged in"},
			{time.Now(), "WARN", "Rate limit approaching"},
			{time.Now(), "ERROR", "Failed to write to disk"},
			{time.Now(), "INFO", "Backup completed"},
			{time.Now(), "DEBUG", "GC cycle completed"},
		},
	}
	return lv
}

func (lv *LogViewer) filteredLogs() []LogEntry {
	if lv.Filter == "" {
		return lv.logs
	}
	var filtered []LogEntry
	for _, log := range lv.logs {
		if strings.Contains(strings.ToLower(log.Message), strings.ToLower(lv.Filter)) ||
			strings.Contains(strings.ToLower(log.Level), strings.ToLower(lv.Filter)) {
			filtered = append(filtered, log)
		}
	}
	return filtered
}

func (lv *LogViewer) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	lv.width = r.Width
	lv.height = r.Height

	titleStyle := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
	ctx.Renderer.WriteString(" Log Viewer", r.X, r.Y, titleStyle.Foreground, titleStyle.Background, titleStyle.Attrs)

	// Filter status
	filterText := fmt.Sprintf(" Filter: %s ", lv.Filter)
	ctx.Renderer.WriteString(filterText, r.X+r.Width-len(filterText)-1, r.Y, mofu.Hex("666666"), mofu.ColorBlack, 0)

	sep := strings.Repeat("─", r.Width-2)
	ctx.Renderer.WriteString(sep, r.X+1, r.Y+1, mofu.Hex("444444"), mofu.ColorBlack, 0)

	// Headers
	headerStyle := mofu.DefaultStyle().Fg(mofu.Hex("89b4fa")).WithAttrs(mofu.AttrBold)
	ctx.Renderer.WriteString(" Time      Level   Message", r.X+1, r.Y+2, headerStyle.Foreground, headerStyle.Background, headerStyle.Attrs)

	// Logs
	logs := lv.filteredLogs()
	y := r.Y + 3
	visible := r.Height - 5
	if visible <= 0 {
		visible = 1
	}

	start := lv.Offset
	if start < 0 {
		start = 0
	}
	if start > len(logs)-visible {
		start = len(logs) - visible
	}
	if start < 0 {
		start = 0
	}

	for i := start; i < len(logs) && y < r.Y+r.Height-2; i++ {
		log := logs[i]
		timeStr := log.Time.Format("15:04:05")

		// Color by level
		var levelColor mofu.Color
		switch log.Level {
		case "ERROR":
			levelColor = mofu.Hex("f38ba8")
		case "WARN":
			levelColor = mofu.Hex("f9e2af")
		case "DEBUG":
			levelColor = mofu.Hex("6c7086")
		default:
			levelColor = mofu.Hex("a6e3a1")
		}

		// Highlight selected
		if i == lv.Selected {
			ctx.Renderer.WriteString(strings.Repeat(" ", r.Width-2), r.X+1, y, mofu.ColorWhite, mofu.Hex("2a2a3e"), 0)
		}

		// Time
		ctx.Renderer.WriteString(timeStr+" ", r.X+1, y, mofu.Hex("666666"), mofu.ColorBlack, 0)

		// Level
		levelStr := fmt.Sprintf("%-5s ", log.Level)
		ctx.Renderer.WriteString(levelStr, r.X+11, y, levelColor, mofu.ColorBlack, 0)

		// Message
		msg := log.Message
		if len(msg) > r.Width-19 {
			msg = msg[:r.Width-22] + "..."
		}
		ctx.Renderer.WriteString(msg, r.X+18, y, mofu.Hex("e0e0e0"), mofu.ColorBlack, 0)

		y++
	}

	// Status bar
	status := fmt.Sprintf(" %d/%d logs │ j/k: Navigate │ /: Filter │ q: Quit", lv.Selected+1, len(logs))
	ctx.Renderer.WriteString(status, r.X, r.Y+r.Height-1, mofu.Hex("666666"), mofu.ColorBlack, 0)
}

func (lv *LogViewer) HandleEvent(event mofu.Event) mofu.Cmd {
	if event.Type != mofu.EventKeyPress {
		return nil
	}
	ke, ok := event.Data.(mofu.KeyEvent)
	if !ok {
		return nil
	}

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && (ke.Runes[0] == 'q' || ke.Runes[0] == 'Q')):
		return mofu.QuitCmd()

	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		logs := lv.filteredLogs()
		if lv.Selected < len(logs)-1 {
			lv.Selected++
			lv.clamp()
		}

	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if lv.Selected > 0 {
			lv.Selected--
			lv.clamp()
		}

	case ke.Key == mofu.KeyPgDn:
		lv.Selected += lv.height - 5
		lv.clamp()

	case ke.Key == mofu.KeyPgUp:
		lv.Selected -= lv.height - 5
		lv.clamp()

	case ke.Key == mofu.KeyHome || (len(ke.Runes) > 0 && ke.Runes[0] == 'g'):
		lv.Selected = 0
		lv.clamp()

	case ke.Key == mofu.KeyEnd || (len(ke.Runes) > 0 && ke.Runes[0] == 'G'):
		logs := lv.filteredLogs()
		lv.Selected = len(logs) - 1
		lv.clamp()

	case len(ke.Runes) > 0 && ke.Runes[0] == '/':
		lv.Filter = ""
		lv.Selected = 0
		lv.Offset = 0

	case len(ke.Runes) > 0 && ke.Runes[0] != '/' && ke.Key != mofu.KeyEsc:
		lv.Filter += string(ke.Runes)
		lv.Selected = 0
		lv.Offset = 0
	}

	return nil
}

func (lv *LogViewer) clamp() {
	logs := lv.filteredLogs()
	visible := lv.height - 5
	if visible <= 0 {
		visible = 10
	}
	if lv.Selected < 0 {
		lv.Selected = 0
	}
	if lv.Selected >= len(logs) {
		lv.Selected = len(logs) - 1
	}
	if lv.Selected < lv.Offset {
		lv.Offset = lv.Selected
	}
	if lv.Selected >= lv.Offset+visible {
		lv.Offset = lv.Selected - visible + 1
	}
	if lv.Offset < 0 {
		lv.Offset = 0
	}
}

func main() {
	app := NewLogViewer()
	if err := mofu.Run(app); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
