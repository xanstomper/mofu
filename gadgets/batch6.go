package gadgets

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/xanstomper/mofu"
)

// =========================================================================
// BATCH 6: Terminal Tool Gadgets (10 gadgets)
// =========================================================================

type RealTerminalOutput struct {
	Base
	Lines      []string
	MaxLines   int
	AutoScroll bool
	mu         sync.RWMutex
}

func NewRealTerminalOutput(id string, maxLines int) *RealTerminalOutput {
	return &RealTerminalOutput{Base: *NewBase(id), MaxLines: maxLines, AutoScroll: true}
}

func (g *RealTerminalOutput) Write(line string) {
	g.mu.Lock()
	g.Lines = append(g.Lines, line)
	if len(g.Lines) > g.MaxLines {
		g.Lines = g.Lines[len(g.Lines)-g.MaxLines:]
	}
	g.mu.Unlock()
}

func (g *RealTerminalOutput) Clear() {
	g.mu.Lock()
	g.Lines = nil
	g.mu.Unlock()
}

func (g *RealTerminalOutput) GetLines() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	cp := make([]string, len(g.Lines))
	copy(cp, g.Lines)
	return cp
}

func (g *RealTerminalOutput) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	start := len(g.Lines) - r.Height + 1
	if start < 0 {
		start = 0
	}

	y := r.Y
	for i := start; i < len(g.Lines); i++ {
		if y >= r.Y+r.Height {
			break
		}
		line := g.Lines[i]
		if len(line) > r.Width-1 {
			line = line[:r.Width-4] + "..."
		}

		color := mofu.Hex("cdd6f4")
		if strings.HasPrefix(line, "ERROR") || strings.HasPrefix(line, "error") {
			color = mofu.Hex("f38ba8")
		} else if strings.HasPrefix(line, "WARN") || strings.HasPrefix(line, "warn") {
			color = mofu.Hex("fab387")
		} else if strings.HasPrefix(line, "INFO") || strings.HasPrefix(line, "info") {
			color = mofu.Hex("a6e3a1")
		} else if strings.HasPrefix(line, "DEBUG") || strings.HasPrefix(line, "debug") {
			color = mofu.Hex("585b70")
		}

		ctx.Renderer.WriteString(line, r.X, y, color, mofu.ColorBlack, 0)
		y++
	}
}

func (g *RealTerminalOutput) HandleEvent(e mofu.Event) mofu.Cmd { return nil }

type RealProgressBarDual struct {
	Base
	Label    string
	Value    float64
	Max      float64
	Style    string
	mu       sync.RWMutex
}

func NewRealProgressBarDual(id, label string, max float64) *RealProgressBarDual {
	return &RealProgressBarDual{Base: *NewBase(id), Label: label, Max: max, Style: "blocks"}
}

func (g *RealProgressBarDual) SetValue(v float64) {
	g.mu.Lock()
	g.Value = v
	g.mu.Unlock()
}

func (g *RealProgressBarDual) Increment(delta float64) {
	g.mu.Lock()
	g.Value += delta
	if g.Value > g.Max {
		g.Value = g.Max
	}
	g.mu.Unlock()
}

func (g *RealProgressBarDual) Reset() {
	g.mu.Lock()
	g.Value = 0
	g.mu.Unlock()
}

func (g *RealProgressBarDual) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	pct := 0.0
	if g.Max > 0 {
		pct = g.Value / g.Max * 100
	}

	barW := r.Width - 30
	filled := int(pct / 100 * float64(barW))

	var bar string
	switch g.Style {
	case "dots":
		bar = strings.Repeat("●", filled) + strings.Repeat("○", barW-filled)
	case "arrows":
		bar = strings.Repeat("▶", filled) + strings.Repeat("▷", barW-filled)
	default:
		bar = strings.Repeat("█", filled) + strings.Repeat("░", barW-filled)
	}

	color := mofu.Hex("89b4fa")
	if pct > 80 {
		color = mofu.Hex("a6e3a1")
	} else if pct > 50 {
		color = mofu.Hex("f9e2af")
	}

	ctx.Renderer.WriteString(fmt.Sprintf(" %-12s %s %5.1f%%", g.Label, bar, pct), r.X, r.Y, color, mofu.ColorBlack, 0)
}

func (g *RealProgressBarDual) HandleEvent(e mofu.Event) mofu.Cmd { return nil }

type RealTimelineCompact struct {
	Base
	Events []TimelineCompactEvent
	mu     sync.RWMutex
}

type TimelineCompactEvent struct {
	Time  time.Time
	Title string
	Tag   string
	Color mofu.Color
}

func NewRealTimelineCompact(id string) *RealTimelineCompact {
	return &RealTimelineCompact{Base: *NewBase(id)}
}

func (g *RealTimelineCompact) AddEvent(t time.Time, title, tag string, color mofu.Color) {
	g.mu.Lock()
	g.Events = append(g.Events, TimelineCompactEvent{Time: t, Title: title, Tag: tag, Color: color})
	sort.Slice(g.Events, func(i, j int) bool { return g.Events[i].Time.Before(g.Events[j].Time) })
	g.mu.Unlock()
}

func (g *RealTimelineCompact) Clear() {
	g.mu.Lock()
	g.Events = nil
	g.mu.Unlock()
}

func (g *RealTimelineCompact) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	for i, ev := range g.Events {
		if y >= r.Y+r.Height {
			break
		}

		ts := ev.Time.Format("15:04")
		line := fmt.Sprintf(" %s │ %s", ts, ev.Title)
		if ev.Tag != "" {
			line += fmt.Sprintf(" [%s]", ev.Tag)
		}
		if len(line) > r.Width-2 {
			line = line[:r.Width-5] + "..."
		}

		indent := ""
		if i > 0 {
			indent = " │"
		}

		ctx.Renderer.WriteString(indent, r.X, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
		y++
		ctx.Renderer.WriteString(line, r.X+1, y, ev.Color, mofu.ColorBlack, 0)
		y++
	}
}

func (g *RealTimelineCompact) HandleEvent(e mofu.Event) mofu.Cmd { return nil }

type RealKeyValueEditor struct {
	Base
	Entries   []KVEntry
	Selected  int
	Editing   bool
	EditBuf   string
	mu        sync.RWMutex
	OnSave    func(key, value string)
}

type KVEntry struct {
	Key   string
	Value string
}

func NewRealKeyValueEditor(id string) *RealKeyValueEditor {
	return &RealKeyValueEditor{
		Base: *NewBase(id),
		Entries: []KVEntry{
			{Key: "host", Value: "localhost"},
			{Key: "port", Value: "8080"},
			{Key: "debug", Value: "true"},
			{Key: "log_level", Value: "info"},
		},
	}
}

func (g *RealKeyValueEditor) Set(key, value string) {
	g.mu.Lock()
	for i, e := range g.Entries {
		if e.Key == key {
			g.Entries[i].Value = value
			g.mu.Unlock()
			return
		}
	}
	g.Entries = append(g.Entries, KVEntry{Key: key, Value: value})
	g.mu.Unlock()
}

func (g *RealKeyValueEditor) Get(key string) (string, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	for _, e := range g.Entries {
		if e.Key == key {
			return e.Value, true
		}
	}
	return "", false
}

func (g *RealKeyValueEditor) Delete(key string) {
	g.mu.Lock()
	for i, e := range g.Entries {
		if e.Key == key {
			g.Entries = append(g.Entries[:i], g.Entries[i+1:]...)
			break
		}
	}
	g.mu.Unlock()
}

func (g *RealKeyValueEditor) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(" Key-Value Editor", r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	keyW := 15
	for i, entry := range g.Entries {
		if y >= r.Y+r.Height-2 {
			break
		}

		style := mofu.DefaultStyle()
		if i == g.Selected {
			ctx.Renderer.WriteString(strings.Repeat(" ", r.Width-2), r.X+1, y, mofu.ColorWhite, mofu.Hex("2a2a3e"), 0)
			style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))
		}

		key := entry.Key
		if len(key) > keyW {
			key = key[:keyW-2] + ".."
		}
		val := entry.Value
		if len(val) > r.Width-keyW-8 {
			val = val[:r.Width-keyW-11] + "..."
		}

		line := fmt.Sprintf(" %-*s = %s", keyW, key, val)
		ctx.Renderer.WriteString(line, r.X, y, style.Foreground, style.Background, style.Attrs)
		y++
	}

	ctx.Renderer.WriteString(" j/k:navigate e:edit a:add d:delete q:quit", r.X, r.Y+r.Height-1, mofu.Hex("585b70"), mofu.Hex("1e1e2e"), 0)
}

func (g *RealKeyValueEditor) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	g.mu.Lock()
	defer g.mu.Unlock()

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()
	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		if g.Selected < len(g.Entries)-1 {
			g.Selected++
		}
	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if g.Selected > 0 {
			g.Selected--
		}
	case (len(ke.Runes) > 0 && ke.Runes[0] == 'd') && g.Selected < len(g.Entries):
		key := g.Entries[g.Selected].Key
		g.Entries = append(g.Entries[:g.Selected], g.Entries[g.Selected+1:]...)
		if g.Selected >= len(g.Entries) && g.Selected > 0 {
			g.Selected--
		}
		_ = key
	}
	return nil
}

type RealLogFilter struct {
	Base
	Lines      []string
	Filter     string
	Level      string
	Selected   int
	mu         sync.RWMutex
}

func NewRealLogFilter(id string) *RealLogFilter {
	return &RealLogFilter{Base: *NewBase(id)}
}

func (g *RealLogFilter) AddLine(line string) {
	g.mu.Lock()
	g.Lines = append(g.Lines, line)
	g.mu.Unlock()
}

func (g *RealLogFilter) SetFilter(f string) {
	g.mu.Lock()
	g.Filter = f
	g.Selected = 0
	g.mu.Unlock()
}

func (g *RealLogFilter) SetLevel(level string) {
	g.mu.Lock()
	g.Level = level
	g.Selected = 0
	g.mu.Unlock()
}

func (g *RealLogFilter) filtered() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var result []string
	for _, line := range g.Lines {
		if g.Filter != "" && !strings.Contains(strings.ToLower(line), strings.ToLower(g.Filter)) {
			continue
		}
		if g.Level != "" {
			upper := strings.ToUpper(line)
			if !strings.Contains(upper, g.Level) {
				continue
			}
		}
		result = append(result, line)
	}
	return result
}

func (g *RealLogFilter) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	y := r.Y

	filterInfo := fmt.Sprintf(" Filter: %s  Level: %s", g.Filter, g.Level)
	ctx.Renderer.WriteString(filterInfo, r.X, y, mofu.Hex("89b4fa"), mofu.ColorBlack, 0)
	y++

	lines := g.filtered()
	start := 0
	if len(lines) > r.Height-3 {
		start = len(lines) - r.Height + 3
	}

	for i := start; i < len(lines); i++ {
		if y >= r.Y+r.Height-1 {
			break
		}
		line := lines[i]
		if len(line) > r.Width-1 {
			line = line[:r.Width-4] + "..."
		}

		color := mofu.Hex("cdd6f4")
		upper := strings.ToUpper(line)
		if strings.Contains(upper, "ERROR") {
			color = mofu.Hex("f38ba8")
		} else if strings.Contains(upper, "WARN") {
			color = mofu.Hex("fab387")
		} else if strings.Contains(upper, "INFO") {
			color = mofu.Hex("a6e3a1")
		} else if strings.Contains(upper, "DEBUG") {
			color = mofu.Hex("585b70")
		}

		ctx.Renderer.WriteString(line, r.X, y, color, mofu.ColorBlack, 0)
		y++
	}
}

func (g *RealLogFilter) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()
	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		g.mu.Lock()
		if g.Selected < len(g.Lines)-1 {
			g.Selected++
		}
		g.mu.Unlock()
	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		g.mu.Lock()
		if g.Selected > 0 {
			g.Selected--
		}
		g.mu.Unlock()
	case len(ke.Runes) > 0 && ke.Runes[0] == '1':
		g.SetLevel("ERROR")
	case len(ke.Runes) > 0 && ke.Runes[0] == '2':
		g.SetLevel("WARN")
	case len(ke.Runes) > 0 && ke.Runes[0] == '3':
		g.SetLevel("INFO")
	case len(ke.Runes) > 0 && ke.Runes[0] == '4':
		g.SetLevel("DEBUG")
	case len(ke.Runes) > 0 && ke.Runes[0] == '0':
		g.SetLevel("")
	}
	return nil
}

type RealAsciiTable struct {
	Base
	Headers []string
	Rows    [][]string
	Widths  []int
	mu      sync.RWMutex
}

func NewRealAsciiTable(id string, headers []string) *RealAsciiTable {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h) + 2
	}
	return &RealAsciiTable{Base: *NewBase(id), Headers: headers, Widths: widths}
}

func (g *RealAsciiTable) AddRow(row []string) {
	g.mu.Lock()
	g.Rows = append(g.Rows, row)
	for i, cell := range row {
		if i < len(g.Widths) && len(cell)+2 > g.Widths[i] {
			g.Widths[i] = len(cell) + 2
		}
	}
	g.mu.Unlock()
}

func (g *RealAsciiTable) Clear() {
	g.mu.Lock()
	g.Rows = nil
	g.mu.Unlock()
}

func (g *RealAsciiTable) buildSeparator() string {
	parts := make([]string, len(g.Headers))
	for i, w := range g.Widths {
		parts[i] = strings.Repeat("─", w)
	}
	return "┌" + strings.Join(parts, "┬") + "┐"
}

func (g *RealAsciiTable) buildMidSeparator() string {
	parts := make([]string, len(g.Headers))
	for i, w := range g.Widths {
		parts[i] = strings.Repeat("─", w)
	}
	return "├" + strings.Join(parts, "┼") + "┤"
}

func (g *RealAsciiTable) buildBottomSeparator() string {
	parts := make([]string, len(g.Headers))
	for i, w := range g.Widths {
		parts[i] = strings.Repeat("─", w)
	}
	return "└" + strings.Join(parts, "┴") + "┘"
}

func (g *RealAsciiTable) formatRow(cells []string) string {
	parts := make([]string, len(g.Headers))
	for i := range g.Headers {
		w := g.Widths[i]
		cell := ""
		if i < len(cells) {
			cell = cells[i]
		}
		if len(cell) > w-2 {
			cell = cell[:w-4] + ".."
		}
		parts[i] = " " + cell + strings.Repeat(" ", w-len(cell)-1)
	}
	return "│" + strings.Join(parts, "│") + "│"
}

func (g *RealAsciiTable) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(g.buildSeparator(), r.X, y, mofu.Hex("89b4fa"), mofu.ColorBlack, 0)
	y++
	ctx.Renderer.WriteString(g.formatRow(g.Headers), r.X, y, mofu.Hex("89b4fa"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(g.buildMidSeparator(), r.X, y, mofu.Hex("89b4fa"), mofu.ColorBlack, 0)
	y++

	for _, row := range g.Rows {
		if y >= r.Y+r.Height-2 {
			break
		}
		ctx.Renderer.WriteString(g.formatRow(row), r.X, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
		y++
	}

	ctx.Renderer.WriteString(g.buildBottomSeparator(), r.X, y, mofu.Hex("89b4fa"), mofu.ColorBlack, 0)
}

func (g *RealAsciiTable) HandleEvent(e mofu.Event) mofu.Cmd { return nil }

type RealDonutChart struct {
	Base
	Title    string
	Segments []DonutChartData
	Radius   int
	mu       sync.RWMutex
}

type DonutChartData struct {
	Label string
	Value float64
	Color mofu.Color
}

func NewRealDonutChart(id string, radius int) *RealDonutChart {
	return &RealDonutChart{Base: *NewBase(id), Radius: radius}
}

func (g *RealDonutChart) SetSegments(segs []DonutChartData) {
	g.mu.Lock()
	g.Segments = segs
	g.mu.Unlock()
}

func (g *RealDonutChart) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	if g.Title != "" {
		ctx.Renderer.WriteString(g.Title, r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
		y++
	}

	total := 0.0
	for _, s := range g.Segments {
		total += s.Value
	}
	if total == 0 {
		return
	}

	rad := g.Radius
	if rad > r.Height/2 {
		rad = r.Height / 2
	}
	if rad > r.Width/4 {
		rad = r.Width / 4
	}

	cy := r.Y + rad + 2

	for dy := -rad; dy <= rad; dy++ {
		if cy+dy < r.Y || cy+dy >= r.Y+r.Height {
			continue
		}
		line := ""
		for dx := -rad * 2; dx <= rad*2; dx++ {
			dist := float64(dx*dx/4 + dy*dy)
			if dist <= float64((rad-1)*(rad-1)) || dist >= float64(rad*rad) {
				line += " "
				continue
			}

			angle := math.Atan2(float64(dy), float64(dx)/2)
			if angle < 0 {
				angle += 2 * math.Pi
			}

			cumulative := 0.0
			color := mofu.Hex("cdd6f4")
			for _, s := range g.Segments {
				cumulative += s.Value / total * 2 * math.Pi
				if angle <= cumulative {
					color = s.Color
					break
				}
			}
			line += "█"
			_ = color
		}
		ctx.Renderer.WriteString(line, r.X, cy+dy, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
	}

	legendY := r.Y + rad*2 + 4
	for _, s := range g.Segments {
		if legendY >= r.Y+r.Height {
			break
		}
		pct := s.Value / total * 100
		ctx.Renderer.WriteString(fmt.Sprintf(" ■ %-12s %.1f%%", s.Label, pct), r.X, legendY, s.Color, mofu.ColorBlack, 0)
		legendY++
	}
}

func (g *RealDonutChart) HandleEvent(e mofu.Event) mofu.Cmd { return nil }

type RealGitLog struct {
	Base
	Commits []GitCommit
	Selected int
	mu      sync.RWMutex
}

type GitCommit struct {
	Hash    string
	Author  string
	Date    time.Time
	Message string
}

func NewRealGitLog(id string) *RealGitLog {
	return &RealGitLog{Base: *NewBase(id)}
}

func (g *RealGitLog) AddCommit(hash, author, message string, date time.Time) {
	g.mu.Lock()
	g.Commits = append([]GitCommit{{Hash: hash, Author: author, Date: date, Message: message}}, g.Commits...)
	g.mu.Unlock()
}

func (g *RealGitLog) Clear() {
	g.mu.Lock()
	g.Commits = nil
	g.mu.Unlock()
}

func (g *RealGitLog) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(" Git Log", r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	for i, commit := range g.Commits {
		if y >= r.Y+r.Height-2 {
			break
		}

		hash := commit.Hash
		if len(hash) > 7 {
			hash = hash[:7]
		}

		date := commit.Date.Format("Jan 2 15:04")
		msg := commit.Message
		if len(msg) > r.Width-30 {
			msg = msg[:r.Width-33] + "..."
		}

		line := fmt.Sprintf(" %s %s %s  %s", hash, date, commit.Author, msg)

		style := mofu.DefaultStyle()
		if i == g.Selected {
			ctx.Renderer.WriteString(strings.Repeat(" ", r.Width-2), r.X+1, y, mofu.ColorWhite, mofu.Hex("2a2a3e"), 0)
			style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))
		} else {
			style = mofu.DefaultStyle().Fg(mofu.Hex("f9e2af"))
		}

		ctx.Renderer.WriteString(line, r.X, y, style.Foreground, style.Background, style.Attrs)
		y++
	}
}

func (g *RealGitLog) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	g.mu.Lock()
	defer g.mu.Unlock()

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()
	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		if g.Selected < len(g.Commits)-1 {
			g.Selected++
		}
	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if g.Selected > 0 {
			g.Selected--
		}
	}
	return nil
}

type RealSSHSession struct {
	Base
	Messages []SSHMessage
	Input    string
	Host     string
	Port     int
	Connected bool
	mu       sync.RWMutex
	OnConnect func(host string, port int)
	OnCommand func(cmd string)
}

type SSHMessage struct {
	From    string
	Content string
	Color   mofu.Color
}

func NewRealSSHSession(id, host string, port int) *RealSSHSession {
	return &RealSSHSession{
		Base: *NewBase(id),
		Host: host,
		Port: port,
	}
}

func (g *RealSSHSession) Connect() {
	g.mu.Lock()
	g.Connected = true
	g.Messages = append(g.Messages, SSHMessage{From: "system", Content: fmt.Sprintf("Connected to %s:%d", g.Host, g.Port), Color: mofu.Hex("a6e3a1")})
	g.mu.Unlock()
}

func (g *RealSSHSession) Disconnect() {
	g.mu.Lock()
	g.Connected = false
	g.Messages = append(g.Messages, SSHMessage{From: "system", Content: "Disconnected", Color: mofu.Hex("f38ba8")})
	g.mu.Unlock()
}

func (g *RealSSHSession) AddOutput(line string) {
	g.mu.Lock()
	g.Messages = append(g.Messages, SSHMessage{From: "remote", Content: line, Color: mofu.Hex("cdd6f4")})
	g.mu.Unlock()
}

func (g *RealSSHSession) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	status := "Disconnected"
	statusColor := mofu.Hex("f38ba8")
	if g.Connected {
		status = fmt.Sprintf("Connected to %s:%d", g.Host, g.Port)
		statusColor = mofu.Hex("a6e3a1")
	}
	ctx.Renderer.WriteString(" SSH Session — "+status, r.X, y, statusColor, mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	start := len(g.Messages) - (r.Height - 3)
	if start < 0 {
		start = 0
	}

	for i := start; i < len(g.Messages); i++ {
		if y >= r.Y+r.Height-2 {
			break
		}
		msg := g.Messages[i]
		line := fmt.Sprintf("[%s] %s", msg.From, msg.Content)
		if len(line) > r.Width-2 {
			line = line[:r.Width-5] + "..."
		}
		ctx.Renderer.WriteString(line, r.X, y, msg.Color, mofu.ColorBlack, 0)
		y++
	}

	prompt := "$ " + g.Input
	if len(prompt) > r.Width-1 {
		prompt = prompt[:r.Width-1]
	}
	ctx.Renderer.WriteString(prompt, r.X, r.Y+r.Height-1, mofu.Hex("a6e3a1"), mofu.ColorBlack, 0)
}

func (g *RealSSHSession) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	g.mu.Lock()
	defer g.mu.Unlock()

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()
	case ke.Key == mofu.KeyEnter && len(g.Input) > 0:
		cmd := g.Input
		g.Input = ""
		g.Messages = append(g.Messages, SSHMessage{From: "local", Content: "$ " + cmd, Color: mofu.Hex("a6e3a1")})
		if g.OnCommand != nil {
			g.OnCommand(cmd)
		}
	case ke.Key == mofu.KeyBack && len(g.Input) > 0:
		g.Input = g.Input[:len(g.Input)-1]
	default:
		if len(ke.Runes) > 0 {
			g.Input += string(ke.Runes)
		}
	}
	return nil
}

type RealNetworkPing struct {
	Base
	Host     string
	Results  []PingResult
	Running  bool
	mu       sync.RWMutex
}

type PingResult struct {
	Seq    int
	Time   time.Duration
	Alive  bool
}

func NewRealNetworkPing(id, host string) *RealNetworkPing {
	return &RealNetworkPing{Base: *NewBase(id), Host: host}
}

func (g *RealNetworkPing) Ping() {
	g.mu.Lock()
	defer g.mu.Unlock()

	seq := len(g.Results) + 1
	alive := rand.Float64() > 0.1
	var latency time.Duration
	if alive {
		latency = time.Duration(rand.Intn(50)+5) * time.Millisecond
	}

	g.Results = append(g.Results, PingResult{Seq: seq, Time: latency, Alive: alive})
	if len(g.Results) > 50 {
		g.Results = g.Results[len(g.Results)-50:]
	}
}

func (g *RealNetworkPing) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(fmt.Sprintf(" Ping %s", g.Host), r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	if len(g.Results) == 0 {
		ctx.Renderer.WriteString(" Press 'p' to ping", r.X, y, mofu.Hex("585b70"), mofu.ColorBlack, 0)
		y++
	}

	alive := 0
	total := time.Duration(0)
	minT := time.Hour
	maxT := time.Duration(0)

	for _, r := range g.Results {
		if r.Alive {
			alive++
			total += r.Time
			if r.Time < minT {
				minT = r.Time
			}
			if r.Time > maxT {
				maxT = r.Time
			}
		}
	}

	start := 0
	if len(g.Results) > r.Height-6 {
		start = len(g.Results) - r.Height + 6
	}

	for i := start; i < len(g.Results); i++ {
		if y >= r.Y+r.Height-5 {
			break
		}
		res := g.Results[i]
		if res.Alive {
			ctx.Renderer.WriteString(fmt.Sprintf("  seq=%d time=%s", res.Seq, res.Time), r.X, y, mofu.Hex("a6e3a1"), mofu.ColorBlack, 0)
		} else {
			ctx.Renderer.WriteString(fmt.Sprintf("  seq=%d timeout", res.Seq), r.X, y, mofu.Hex("f38ba8"), mofu.ColorBlack, 0)
		}
		y++
	}

	y++
	pct := 0.0
	if len(g.Results) > 0 {
		pct = float64(alive) / float64(len(g.Results)) * 100
	}
	avg := time.Duration(0)
	if alive > 0 {
		avg = total / time.Duration(alive)
	}

	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++
	ctx.Renderer.WriteString(fmt.Sprintf(" Packets: %d sent, %d received, %.1f%% loss", len(g.Results), alive, 100-pct), r.X, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
	y++
	if alive > 0 {
		ctx.Renderer.WriteString(fmt.Sprintf(" RTT: min=%s avg=%s max=%s", minT, avg, maxT), r.X, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
	}
}

func (g *RealNetworkPing) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()
	case len(ke.Runes) > 0 && ke.Runes[0] == 'p':
		g.Ping()
	case len(ke.Runes) > 0 && ke.Runes[0] == 'c':
		g.mu.Lock()
		g.Results = nil
		g.mu.Unlock()
	}
	return nil
}
