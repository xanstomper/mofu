package main

import (
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/xanstomper/mofu"
	"github.com/xanstomper/mofu/widgets"
)

// MOFU Demo — a polished showcase of the framework's capabilities.
// Run: go run .

type Demo struct {
	mofu.Minimal
	width    int
	height   int
	tab      int
	focus    int
	tabs     []string

	// Dashboard data
	cpu      float64
	mem      float64
	disk     float64
	netIn    float64
	netOut   float64
	uptime   time.Duration
	start    time.Time

	// Process list
	procs    []Process

	// Log stream
	logs     []LogEntry
	logScroll int

	// Todo list
	todos    []Todo
	todoIdx  int
	todoText string
	todoMode bool

	// Notification panel
	notifs   []Notification

	mu sync.RWMutex
}

type Process struct {
	Name string
	CPU  float64
	Mem  float64
	PID  int
}

type LogEntry struct {
	Time    string
	Level   string
	Message string
}

type Todo struct {
	Text   string
	Done   bool
}

type Notification struct {
	Title   string
	Message string
	Time    time.Time
	Level   string
}

func NewDemo() *Demo {
	d := &Demo{
		start:  time.Now(),
		tabs:   []string{"Dashboard", "Processes", "Logs", "Todo", "Widgets"},
		procs: []Process{
			{Name: "mofu-demo", CPU: 2.3, Mem: 45.2, PID: 1234},
			{Name: "postgres", CPU: 8.1, Mem: 128.5, PID: 2345},
			{Name: "nginx", CPU: 0.8, Mem: 12.3, PID: 3456},
			{Name: "redis", CPU: 1.2, Mem: 8.9, PID: 4567},
			{Name: "node", CPU: 15.3, Mem: 256.8, PID: 5678},
			{Name: "docker", CPU: 3.5, Mem: 512.0, PID: 6789},
			{Name: "go", CPU: 22.1, Mem: 89.4, PID: 7890},
			{Name: "python", CPU: 5.6, Mem: 67.2, PID: 8901},
		},
		todos: []Todo{
			{Text: "Build MOFU framework", Done: true},
			{Text: "Add 112 gadgets", Done: true},
			{Text: "Create 24 examples", Done: true},
			{Text: "Write 3 tutorials", Done: true},
			{Text: "Security audit", Done: true},
			{Text: "Release v1.0", Done: false},
		},
		logs: []LogEntry{
			{Time: "12:00:01", Level: "INFO", Message: "Application started"},
			{Time: "12:00:02", Level: "INFO", Message: "Connected to database"},
			{Time: "12:00:03", Level: "INFO", Message: "Loaded 112 gadgets"},
			{Time: "12:00:04", Level: "WARN", Message: "High memory usage detected"},
			{Time: "12:00:05", Level: "INFO", Message: "Agent framework initialized"},
			{Time: "12:00:06", Level: "ERROR", Message: "Connection timeout to redis"},
			{Time: "12:00:07", Level: "INFO", Message: "Retrying connection..."},
			{Time: "12:00:08", Level: "INFO", Message: "Connection restored"},
		},
		notifs: []Notification{
			{Title: "Build Complete", Message: "All 207 tests passing", Time: time.Now().Add(-5 * time.Minute), Level: "success"},
			{Title: "Security Audit", Message: "4 vulnerabilities fixed", Time: time.Now().Add(-10 * time.Minute), Level: "success"},
			{Title: "Performance", Message: "0 allocs on hot path", Time: time.Now().Add(-15 * time.Minute), Level: "info"},
		},
	}

	go d.simulateData()
	go d.generateLogs()

	return d
}

func (d *Demo) simulateData() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		d.mu.Lock()
		d.cpu = 20 + rand.Float64()*60
		d.mem = 50 + rand.Float64()*40
		d.disk = 45 + rand.Float64()*10
		d.netIn = rand.Float64() * 100
		d.netOut = rand.Float64() * 50
		d.uptime = time.Since(d.start)

		for i := range d.procs {
			d.procs[i].CPU = 0.5 + rand.Float64()*30
			d.procs[i].Mem += (rand.Float64() - 0.5) * 5
			if d.procs[i].Mem < 1 {
				d.procs[i].Mem = 1
			}
		}
		d.mu.Unlock()
	}
}

func (d *Demo) generateLogs() {
	levels := []string{"INFO", "WARN", "ERROR", "DEBUG"}
	messages := []string{
		"GET /api/v1/users 200 OK 12ms",
		"POST /api/v1/orders 201 Created 45ms",
		"Cache hit for key: session:abc123",
		"Background job completed: sync_data",
		"Memory usage at 78% — monitoring",
		"Health check passed",
		"Request middleware took 3ms",
		"WebSocket connection established",
		"File upload completed: report.pdf",
		"Rate limit triggered for client: mobile",
	}

	for {
		time.Sleep(time.Duration(200+rand.Intn(800)) * time.Millisecond)

		d.mu.Lock()
		level := levels[rand.Intn(len(levels))]
		msg := messages[rand.Intn(len(messages))]
		d.logs = append(d.logs, LogEntry{
			Time:    time.Now().Format("15:04:05"),
			Level:   level,
			Message: msg,
		})
		if len(d.logs) > 200 {
			d.logs = d.logs[len(d.logs)-200:]
		}
		d.mu.Unlock()
	}
}

func (d *Demo) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	d.mu.Lock()
	d.width = r.Width
	d.height = r.Height
	d.mu.Unlock()

	// Top bar
	d.renderTopBar(ctx, r)

	// Tab content
	contentY := r.Y + 2
	contentH := r.Height - 4

	contentCtx := &mofu.RenderContext{
		Bounds:   mofu.Rect{X: r.X, Y: contentY, Width: r.Width, Height: contentH},
		Renderer: ctx.Renderer,
	}

	switch d.tab {
	case 0:
		d.renderDashboard(contentCtx)
	case 1:
		d.renderProcesses(contentCtx)
	case 2:
		d.renderLogs(contentCtx)
	case 3:
		d.renderTodo(contentCtx)
	case 4:
		d.renderWidgets(contentCtx)
	}

	// Bottom bar
	d.renderBottomBar(ctx, r)
}

func (d *Demo) renderTopBar(ctx *mofu.RenderContext, r mofu.Rect) {
	bg := mofu.Hex("1e1e2e")

	// Tab bar
	x := r.X
	for i, tab := range d.tabs {
		style := " "
		color := mofu.Hex("6c7086")
		if i == d.tab {
			style = "▸"
			color = mofu.Hex("ff69b4")
		}
		label := fmt.Sprintf(" %s %s ", style, tab)
		ctx.Renderer.WriteString(label, x, r.Y, color, bg, 0)
		x += len(label)
	}

	// Fill rest
	remaining := r.Width - (x - r.X)
	if remaining > 0 {
		ctx.Renderer.WriteString(strings.Repeat(" ", remaining), x, r.Y, mofu.Hex("6c7086"), bg, 0)
	}

	ctx.Renderer.WriteString(strings.Repeat("─", r.Width), r.X, r.Y+1, mofu.Hex("313244"), mofu.ColorBlack, 0)
}

func (d *Demo) renderBottomBar(ctx *mofu.RenderContext, r mofu.Rect) {
	bg := mofu.Hex("1e1e2e")
	y := r.Y + r.Height - 1

	// Left: navigation hint
	nav := " ← →:tabs"
	ctx.Renderer.WriteString(nav, r.X, y, mofu.Hex("6c7086"), bg, 0)

	// Center: tab-specific shortcuts
	var shortcuts string
	switch d.tab {
	case 0:
		shortcuts = " j/k:scroll"
	case 1:
		shortcuts = " ↑↓:select  s:sort"
	case 2:
		shortcuts = " j/k:scroll  c:clear"
	case 3:
		shortcuts = " ↑↓:select  a:add  d:done  x:delete"
	case 4:
		shortcuts = " ↑↓:select  Enter:interact"
	}
	ctx.Renderer.WriteString(shortcuts, r.X+len(nav)+2, y, mofu.Hex("585b70"), bg, 0)

	// Right: clock
	clock := time.Now().Format("15:04:05")
	ctx.Renderer.WriteString(clock, r.X+r.Width-len(clock)-1, y, mofu.Hex("a6e3a1"), bg, 0)
}

// =================== DASHBOARD ===================

func (d *Demo) renderDashboard(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	d.mu.RLock()
	defer d.mu.RUnlock()

	y := r.Y
	leftW := r.Width * 2 / 3
	rightW := r.Width - leftW

	// Left panel: metrics
	d.renderMetricGauge(ctx, r.X, y, leftW-1, "CPU Usage", d.cpu, "%", 100, d.cpuColor())
	y += 3

	d.renderMetricGauge(ctx, r.X, y, leftW-1, "Memory", d.mem, "%", 100, d.memColor())
	y += 3

	d.renderMetricGauge(ctx, r.X, y, leftW-1, "Disk", d.disk, "%", 100, d.diskColor())
	y += 3

	d.renderNetworkBar(ctx, r.X, y, leftW-1, "Network In", d.netIn, "MB/s", mofu.Hex("89b4fa"))
	y += 2
	d.renderNetworkBar(ctx, r.X, y, leftW-1, "Network Out", d.netOut, "MB/s", mofu.Hex("f38ba8"))
	y += 2

	// Uptime
	uptime := fmt.Sprintf(" Uptime: %s", d.uptime.Round(time.Second))
	ctx.Renderer.WriteString(uptime, r.X, y, mofu.Hex("a6e3a1"), mofu.ColorBlack, 0)

	// Separator
	ctx.Renderer.WriteString("│", r.X+leftW-1, r.Y, mofu.Hex("313244"), mofu.ColorBlack, 0)

	// Right panel: notifications
	ctx.Renderer.WriteString(" Notifications", r.X+leftW, r.Y, mofu.Hex("89b4fa"), mofu.ColorBlack, mofu.AttrBold)
	ctx.Renderer.WriteString(strings.Repeat("─", rightW-1), r.X+leftW, r.Y+1, mofu.Hex("313244"), mofu.ColorBlack, 0)

	notifY := r.Y + 2
	for _, n := range d.notifs {
		if notifY >= r.Y+r.Height-1 {
			break
		}
		icon := "●"
		color := mofu.Hex("a6e3a1")
		switch n.Level {
		case "warning":
			icon = "⚠"
			color = mofu.Hex("fab387")
		case "error":
			icon = "✗"
			color = mofu.Hex("f38ba8")
		case "info":
			icon = "ℹ"
			color = mofu.Hex("89b4fa")
		}

		title := n.Title
		if len(title) > rightW-5 {
			title = title[:rightW-8] + "..."
		}
		ctx.Renderer.WriteString(fmt.Sprintf(" %s %s", icon, title), r.X+leftW, notifY, color, mofu.ColorBlack, 0)
		notifY++

		msg := n.Message
		if len(msg) > rightW-5 {
			msg = msg[:rightW-8] + "..."
		}
		ctx.Renderer.WriteString("   "+msg, r.X+leftW, notifY, mofu.Hex("6c7086"), mofu.ColorBlack, 0)
		notifY++
	}
}

func (d *Demo) renderMetricGauge(ctx *mofu.RenderContext, x, y, w int, label string, value float64, unit string, max float64, color mofu.Color) {
	pct := value / max * 100
	barW := w - 25
	if barW < 5 {
		barW = 5
	}
	filled := int(pct / 100 * float64(barW))

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barW-filled)
	ctx.Renderer.WriteString(fmt.Sprintf(" %-10s %s %5.1f%s", label, bar, pct, unit), x, y, color, mofu.ColorBlack, 0)
}

func (d *Demo) renderNetworkBar(ctx *mofu.RenderContext, x, y, w int, label string, value float64, unit string, color mofu.Color) {
	barW := w - 25
	if barW < 5 {
		barW = 5
	}
	filled := int(value / 100 * float64(barW))
	if filled > barW {
		filled = barW
	}

	bar := strings.Repeat("▓", filled) + strings.Repeat("░", barW-filled)
	ctx.Renderer.WriteString(fmt.Sprintf(" %-10s %s %5.1f %s", label, bar, value, unit), x, y, color, mofu.ColorBlack, 0)
}

func (d *Demo) cpuColor() mofu.Color {
	if d.cpu > 80 {
		return mofu.Hex("f38ba8")
	}
	if d.cpu > 60 {
		return mofu.Hex("fab387")
	}
	return mofu.Hex("a6e3a1")
}

func (d *Demo) memColor() mofu.Color {
	if d.mem > 80 {
		return mofu.Hex("f38ba8")
	}
	if d.mem > 60 {
		return mofu.Hex("fab387")
	}
	return mofu.Hex("89b4fa")
}

func (d *Demo) diskColor() mofu.Color {
	if d.disk > 90 {
		return mofu.Hex("f38ba8")
	}
	return mofu.Hex("a6e3a1")
}

// =================== PROCESSES ===================

func (d *Demo) renderProcesses(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	d.mu.RLock()
	defer d.mu.RUnlock()

	y := r.Y

	// Header
	ctx.Renderer.WriteString(" Processes", r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("313244"), mofu.ColorBlack, 0)
	y++

	header := fmt.Sprintf("  %-4s %-16s %8s %8s", "PID", "Name", "CPU%", "Mem(MB)")
	ctx.Renderer.WriteString(header, r.X, y, mofu.Hex("89b4fa"), mofu.ColorBlack, mofu.AttrBold)
	y++

	for i, proc := range d.procs {
		if y >= r.Y+r.Height-1 {
			break
		}

		style := mofu.DefaultStyle()
		if i == d.focus {
			ctx.Renderer.WriteString(strings.Repeat(" ", r.Width-2), r.X+1, y, mofu.ColorWhite, mofu.Hex("1e1e2e"), 0)
			style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))
		}

		cpuColor := mofu.Hex("a6e3a1")
		if proc.CPU > 20 {
			cpuColor = mofu.Hex("fab387")
		}
		if proc.CPU > 30 {
			cpuColor = mofu.Hex("f38ba8")
		}

		line := fmt.Sprintf("  %-4d %-16s %7.1f%% %8.1f", proc.PID, proc.Name, proc.CPU, proc.Mem)
		if len(line) > r.Width-2 {
			line = line[:r.Width-2]
		}
		ctx.Renderer.WriteString(line, r.X, y, cpuColor, mofu.ColorBlack, style.Attrs)
		y++
	}
}

// =================== LOGS ===================

func (d *Demo) renderLogs(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	d.mu.RLock()
	defer d.mu.RUnlock()

	y := r.Y

	ctx.Renderer.WriteString(fmt.Sprintf(" Logs (%d entries)", len(d.logs)), r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("313244"), mofu.ColorBlack, 0)
	y++

	start := len(d.logs) - (r.Height - 3)
	if start < 0 {
		start = 0
	}

	for i := start; i < len(d.logs); i++ {
		if y >= r.Y+r.Height-1 {
			break
		}

		entry := d.logs[i]
		color := mofu.Hex("cdd6f4")
		switch entry.Level {
		case "ERROR":
			color = mofu.Hex("f38ba8")
		case "WARN":
			color = mofu.Hex("fab387")
		case "INFO":
			color = mofu.Hex("a6e3a1")
		case "DEBUG":
			color = mofu.Hex("6c7086")
		}

		line := fmt.Sprintf(" %s %-6s %s", entry.Time, entry.Level, entry.Message)
		if len(line) > r.Width-2 {
			line = line[:r.Width-5] + "..."
		}
		ctx.Renderer.WriteString(line, r.X, y, color, mofu.ColorBlack, 0)
		y++
	}
}

// =================== TODO ===================

func (d *Demo) renderTodo(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	d.mu.RLock()
	defer d.mu.RUnlock()

	y := r.Y

	ctx.Renderer.WriteString(" Todo List", r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("313244"), mofu.ColorBlack, 0)
	y++

	for i, todo := range d.todos {
		if y >= r.Y+r.Height-3 {
			break
		}

		icon := "○"
		if todo.Done {
			icon = "●"
		}

		prefix := "  "
		if i == d.todoIdx && !d.todoMode {
			prefix = "▸ "
		}

		color := mofu.Hex("cdd6f4")
		if todo.Done {
			color = mofu.Hex("6c7086")
		}

		text := todo.Text
		if len(text) > r.Width-10 {
			text = text[:r.Width-13] + "..."
		}
		ctx.Renderer.WriteString(fmt.Sprintf("%s%s %s", prefix, icon, text), r.X, y, color, mofu.ColorBlack, 0)
		y++
	}

	// Add todo input
	y++
	if d.todoMode {
		ctx.Renderer.WriteString(" > "+d.todoText+"_ ", r.X, y, mofu.Hex("cdd6f4"), mofu.Hex("1e1e2e"), 0)
	} else {
		ctx.Renderer.WriteString(" a:add todo  d:mark done  x:delete  Enter:submit", r.X, y, mofu.Hex("6c7086"), mofu.ColorBlack, 0)
	}
}

// =================== WIDGETS SHOWCASE ===================

func (d *Demo) renderWidgets(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(" Widgets Showcase", r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("313244"), mofu.ColorBlack, 0)
	y += 2

	// Progress bar
	ctx.Renderer.WriteString(" Progress Bar", r.X+1, y, mofu.Hex("89b4fa"), mofu.ColorBlack, mofu.AttrBold)
	y++
	pb := widgets.NewProgressBar(0.67)
	pbCtx := &mofu.RenderContext{
		Bounds:   mofu.Rect{X: r.X + 1, Y: y, Width: r.Width - 4, Height: 1},
		Renderer: ctx.Renderer,
	}
	pb.Render(pbCtx)
	y += 3

	// Button
	ctx.Renderer.WriteString(" Buttons", r.X+1, y, mofu.Hex("89b4fa"), mofu.ColorBlack, mofu.AttrBold)
	y++
	btn := widgets.NewButton("Click Me", nil)
	btnCtx := &mofu.RenderContext{
		Bounds:   mofu.Rect{X: r.X + 1, Y: y, Width: 20, Height: 1},
		Renderer: ctx.Renderer,
	}
	btn.Render(btnCtx)
	y += 3

	// Checkbox
	ctx.Renderer.WriteString(" Checkboxes", r.X+1, y, mofu.Hex("89b4fa"), mofu.ColorBlack, mofu.AttrBold)
	y++
	cb := widgets.NewCheckbox("Enable feature", true)
	cbCtx := &mofu.RenderContext{
		Bounds:   mofu.Rect{X: r.X + 1, Y: y, Width: r.Width - 4, Height: 1},
		Renderer: ctx.Renderer,
	}
	cb.Render(cbCtx)
	y += 2

	cb2 := widgets.NewCheckbox("Debug mode", false)
	cb2Ctx := &mofu.RenderContext{
		Bounds:   mofu.Rect{X: r.X + 1, Y: y, Width: r.Width - 4, Height: 1},
		Renderer: ctx.Renderer,
	}
	cb2.Render(cb2Ctx)
	y += 3

	// Text
	ctx.Renderer.WriteString(" Text Widget", r.X+1, y, mofu.Hex("89b4fa"), mofu.ColorBlack, mofu.AttrBold)
	y++
	textW := widgets.NewText("This is a styled text widget with MOFU.")
	textCtx := &mofu.RenderContext{
		Bounds:   mofu.Rect{X: r.X + 1, Y: y, Width: r.Width - 4, Height: 1},
		Renderer: ctx.Renderer,
	}
	textW.Render(textCtx)
	y += 2

	// Tabs widget
	ctx.Renderer.WriteString(" Tabs Widget", r.X+1, y, mofu.Hex("89b4fa"), mofu.ColorBlack, mofu.AttrBold)
	y++
	tabs := widgets.NewTabs([]widgets.Tab{
		{Label: "Overview"},
		{Label: "Details"},
		{Label: "Settings"},
	})
	tabsCtx := &mofu.RenderContext{
		Bounds:   mofu.Rect{X: r.X + 1, Y: y, Width: r.Width - 4, Height: 1},
		Renderer: ctx.Renderer,
	}
	tabs.Render(tabsCtx)
	y += 3

	// Toast
	ctx.Renderer.WriteString(" Toast Notification", r.X+1, y, mofu.Hex("89b4fa"), mofu.ColorBlack, mofu.AttrBold)
	y++
	toast := widgets.NewToast("Operation completed successfully!", widgets.ToastSuccess)
	toastCtx := &mofu.RenderContext{
		Bounds:   mofu.Rect{X: r.X + 1, Y: y, Width: r.Width - 4, Height: 1},
		Renderer: ctx.Renderer,
	}
	toast.Render(toastCtx)
}

// =================== EVENT HANDLING ===================

func (d *Demo) HandleEvent(event mofu.Event) mofu.Cmd {
	if event.Type != mofu.EventKeyPress {
		return nil
	}
	ke := event.Data.(mofu.KeyEvent)

	// Global: quit
	if ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q') {
		return mofu.QuitCmd()
	}

	// Global: tab switch
	if ke.Key == mofu.KeyRight || (len(ke.Runes) > 0 && ke.Runes[0] == 'l') {
		d.tab = (d.tab + 1) % len(d.tabs)
		d.focus = 0
		return nil
	}
	if ke.Key == mofu.KeyLeft || (len(ke.Runes) > 0 && ke.Runes[0] == 'h') {
		d.tab--
		if d.tab < 0 {
			d.tab = len(d.tabs) - 1
		}
		d.focus = 0
		return nil
	}

	// Tab-specific handling
	switch d.tab {
	case 1:
		d.handleProcesses(ke)
	case 2:
		d.handleLogs(ke)
	case 3:
		d.handleTodo(ke)
	}

	return nil
}

func (d *Demo) handleProcesses(ke mofu.KeyEvent) {
	d.mu.Lock()
	defer d.mu.Unlock()

	switch {
	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		if d.focus < len(d.procs)-1 {
			d.focus++
		}
	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if d.focus > 0 {
			d.focus--
		}
	case len(ke.Runes) > 0 && ke.Runes[0] == 's':
		sort.Slice(d.procs, func(i, j int) bool {
			return d.procs[i].CPU > d.procs[j].CPU
		})
	}
}

func (d *Demo) handleLogs(ke mofu.KeyEvent) {
	d.mu.Lock()
	defer d.mu.Unlock()

	switch {
	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		d.logScroll++
	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if d.logScroll > 0 {
			d.logScroll--
		}
	case len(ke.Runes) > 0 && ke.Runes[0] == 'c':
		d.logs = nil
		d.logScroll = 0
	}
}

func (d *Demo) handleTodo(ke mofu.KeyEvent) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.todoMode {
		// Text input mode
		switch {
		case ke.Key == mofu.KeyEnter:
			if len(d.todoText) > 0 {
				d.todos = append(d.todos, Todo{Text: d.todoText})
				d.todoText = ""
			}
			d.todoMode = false
		case ke.Key == mofu.KeyBack && len(d.todoText) > 0:
			d.todoText = d.todoText[:len(d.todoText)-1]
		case ke.Key == mofu.KeyEsc:
			d.todoMode = false
			d.todoText = ""
		default:
			if len(ke.Runes) > 0 {
				d.todoText += string(ke.Runes)
			}
		}
		return
	}

	// Navigation mode
	switch {
	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		if d.todoIdx < len(d.todos)-1 {
			d.todoIdx++
		}
	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if d.todoIdx > 0 {
			d.todoIdx--
		}
	case len(ke.Runes) > 0 && ke.Runes[0] == 'a':
		d.todoMode = true
		d.todoText = ""
	case len(ke.Runes) > 0 && ke.Runes[0] == 'd':
		if d.todoIdx < len(d.todos) {
			d.todos[d.todoIdx].Done = !d.todos[d.todoIdx].Done
		}
	case len(ke.Runes) > 0 && ke.Runes[0] == 'x':
		if d.todoIdx < len(d.todos) {
			d.todos = append(d.todos[:d.todoIdx], d.todos[d.todoIdx+1:]...)
			if d.todoIdx >= len(d.todos) && d.todoIdx > 0 {
				d.todoIdx--
			}
		}
	}
}

// =================== MAIN ===================

func main() {
	app := NewDemo()
	if err := mofu.Run(app); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
