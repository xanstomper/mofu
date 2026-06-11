package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/xanstomper/mofu"
)

// ---------------------------------------------------------------------------
// Dashboard — a real-world MOFU example using the simplified API
// ---------------------------------------------------------------------------

type Dashboard struct {
	mofu.Minimal
	width, height int
	selected      int
	items         []string
	status        string
	clock         string
}

func NewDashboard() *Dashboard {
	d := &Dashboard{
		items: []string{
			"System Overview",
			"Network Status",
			"Disk Usage",
			"Process List",
			"Logs",
		},
		status: "Ready",
	}
	return d
}

func (d *Dashboard) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	d.width = r.Width
	d.height = r.Height

	if d.width < 20 || d.height < 10 {
		ctx.Renderer.WriteString("Terminal too small", r.X, r.Y, mofu.ColorWhite, mofu.ColorBlack, 0)
		return
	}

	// Header
	headerStyle := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
	ctx.Renderer.WriteString(" MOFU Dashboard", r.X, r.Y, headerStyle.Foreground, headerStyle.Background, headerStyle.Attrs)

	// Clock in top right
	d.clock = time.Now().Format("15:04:05")
	clockX := r.X + r.Width - len(d.clock) - 2
	ctx.Renderer.WriteString(d.clock, clockX, r.Y, mofu.Hex("666666"), mofu.ColorBlack, 0)

	// Separator
	sep := strings.Repeat("─", r.Width-2)
	ctx.Renderer.WriteString(sep, r.X+1, r.Y+1, mofu.Hex("444444"), mofu.ColorBlack, 0)

	// Sidebar
	sidebarWidth := 22
	if sidebarWidth > r.Width/3 {
		sidebarWidth = r.Width / 3
	}

	// Sidebar background
	for y := r.Y + 2; y < r.Y+r.Height-2; y++ {
		ctx.Renderer.WriteString(strings.Repeat(" ", sidebarWidth), r.X+1, y, mofu.ColorWhite, mofu.Hex("1a1a2e"), 0)
	}

	// Sidebar title
	titleStyle := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
	ctx.Renderer.WriteString(" Navigation", r.X+1, r.Y+2, titleStyle.Foreground, titleStyle.Background, titleStyle.Attrs)

	// Menu items
	for i, item := range d.items {
		y := r.Y + 4 + i
		if y >= r.Y+r.Height-2 {
			break
		}
		prefix := "  "
		itemStyle := mofu.DefaultStyle().Fg(mofu.Hex("e0e0e0"))
		if i == d.selected {
			prefix = "▸ "
			itemStyle = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
			// Highlight background
			ctx.Renderer.WriteString(strings.Repeat(" ", sidebarWidth), r.X+1, y, mofu.ColorWhite, mofu.Hex("2a2a3e"), 0)
		}
		ctx.Renderer.WriteString(prefix+item, r.X+1, y, itemStyle.Foreground, itemStyle.Background, itemStyle.Attrs)
	}

	// Main content area
	contentX := r.X + sidebarWidth + 2
	contentW := r.Width - sidebarWidth - 3

	// Content border
	borderStyle := mofu.DefaultStyle().Fg(mofu.Hex("444444"))
	ctx.Renderer.WriteString("┌"+strings.Repeat("─", contentW-2)+"┐", contentX, r.Y+2, borderStyle.Foreground, borderStyle.Background, borderStyle.Attrs)
	for y := r.Y + 3; y < r.Y+r.Height-3; y++ {
		ctx.Renderer.WriteString("│", contentX, y, borderStyle.Foreground, borderStyle.Background, borderStyle.Attrs)
		ctx.Renderer.WriteString(strings.Repeat(" ", contentW-2), contentX+1, y, mofu.ColorWhite, mofu.ColorBlack, 0)
		ctx.Renderer.WriteString("│", contentX+contentW-1, y, borderStyle.Foreground, borderStyle.Background, borderStyle.Attrs)
	}
	ctx.Renderer.WriteString("└"+strings.Repeat("─", contentW-2)+"┘", contentX, r.Y+r.Height-3, borderStyle.Foreground, borderStyle.Background, borderStyle.Attrs)

	// Content
	contentStyle := mofu.DefaultStyle().Fg(mofu.Hex("e0e0e0"))
	switch d.selected {
	case 0:
		d.renderOverview(ctx, contentX+2, r.Y+4, contentW-4)
	case 1:
		d.renderNetwork(ctx, contentX+2, r.Y+4, contentW-4)
	case 2:
		d.renderDisk(ctx, contentX+2, r.Y+4, contentW-4)
	case 3:
		d.renderProcesses(ctx, contentX+2, r.Y+4, contentW-4)
	case 4:
		d.renderLogs(ctx, contentX+2, r.Y+4, contentW-4)
	}
	_ = contentStyle

	// Status bar
	statusBar := fmt.Sprintf(" ↑↓ Navigate │ Enter Select │ q Quit │ %s", d.status)
	ctx.Renderer.WriteString(statusBar, r.X+1, r.Y+r.Height-1, mofu.Hex("666666"), mofu.ColorBlack, 0)
}

func (d *Dashboard) renderOverview(ctx *mofu.RenderContext, x, y, w int) {
	titleStyle := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
	ctx.Renderer.WriteString("System Overview", x, y, titleStyle.Foreground, titleStyle.Background, titleStyle.Attrs)

	y += 2
	dimStyle := mofu.DefaultStyle().Fg(mofu.Hex("666666"))

	metrics := []struct {
		label string
		value string
		color string
	}{
		{"CPU Usage", "23%", "a6e3a1"},
		{"Memory", "4.2 GB / 16 GB", "89b4fa"},
		{"Disk", "120 GB / 500 GB", "f9e2af"},
		{"Uptime", "3d 14h 22m", "e0e0e0"},
		{"Processes", "142", "e0e0e0"},
		{"Goroutines", "24", "e0e0e0"},
	}

	for i, m := range metrics {
		if y+i*2 >= d.height-4 {
			break
		}
		ctx.Renderer.WriteString(m.label+":", x, y+i*2, dimStyle.Foreground, dimStyle.Background, dimStyle.Attrs)
		ctx.Renderer.WriteString(m.value, x+len(m.label)+2, y+i*2, mofu.Hex(m.color), mofu.ColorBlack, 0)
	}
}

func (d *Dashboard) renderNetwork(ctx *mofu.RenderContext, x, y, w int) {
	titleStyle := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
	ctx.Renderer.WriteString("Network Status", x, y, titleStyle.Foreground, titleStyle.Background, titleStyle.Attrs)

	y += 2
	dimStyle := mofu.DefaultStyle().Fg(mofu.Hex("666666"))
	goodStyle := mofu.DefaultStyle().Fg(mofu.Hex("a6e3a1"))

	connections := []string{
		"eth0:     192.168.1.100  ● Connected",
		"wifi:     10.0.0.42     ● Connected",
		"vpn:      10.8.0.1      ● Disconnected",
		"docker0:  172.17.0.1     ● Active",
	}

	for i, c := range connections {
		if y+i >= d.height-4 {
			break
		}
		style := dimStyle
		if strings.Contains(c, "Connected") || strings.Contains(c, "Active") {
			style = goodStyle
		}
		ctx.Renderer.WriteString(c, x, y+i, style.Foreground, style.Background, style.Attrs)
	}
}

func (d *Dashboard) renderDisk(ctx *mofu.RenderContext, x, y, w int) {
	titleStyle := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
	ctx.Renderer.WriteString("Disk Usage", x, y, titleStyle.Foreground, titleStyle.Background, titleStyle.Attrs)

	y += 2
	dimStyle := mofu.DefaultStyle().Fg(mofu.Hex("666666"))

	disks := []struct {
		mount string
		used  int
		total int
	}{
		{"/", 120, 500},
		{"/home", 45, 200},
		{"/tmp", 2, 10},
	}

	for i, disk := range disks {
		if y+i*2 >= d.height-4 {
			break
		}
		ctx.Renderer.WriteString(disk.mount, x, y+i*2, dimStyle.Foreground, dimStyle.Background, dimStyle.Attrs)

		// Progress bar
		barW := w - len(disk.mount) - 15
		if barW < 10 {
			barW = 10
		}
		filled := disk.used * barW / disk.total
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barW-filled)
		barColor := "a6e3a1"
		if disk.used*100/disk.total > 80 {
			barColor = "f38ba8"
		} else if disk.used*100/disk.total > 60 {
			barColor = "f9e2af"
		}
		ctx.Renderer.WriteString(bar, x+len(disk.mount)+1, y+i*2, mofu.Hex(barColor), mofu.ColorBlack, 0)

		pct := fmt.Sprintf("%d%%", disk.used*100/disk.total)
		ctx.Renderer.WriteString(pct, x+len(disk.mount)+barW+2, y+i*2, dimStyle.Foreground, dimStyle.Background, dimStyle.Attrs)
	}
}

func (d *Dashboard) renderProcesses(ctx *mofu.RenderContext, x, y, w int) {
	titleStyle := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
	ctx.Renderer.WriteString("Process List", x, y, titleStyle.Foreground, titleStyle.Background, titleStyle.Attrs)

	y += 2
	headerStyle := mofu.DefaultStyle().Fg(mofu.Hex("89b4fa")).WithAttrs(mofu.AttrBold)
	ctx.Renderer.WriteString("PID   NAME                CPU   MEM", x, y, headerStyle.Foreground, headerStyle.Background, headerStyle.Attrs)
	y++

	dimStyle := mofu.DefaultStyle().Fg(mofu.Hex("666666"))
	processes := []string{
		"1     systemd           0.1%  12M",
		"234   mofu-dashboard    2.3%  45M",
		"567   sshd              0.0%  8M",
		"890   nginx             0.2%  24M",
		"1234  postgres          1.1%  128M",
		"2345  redis-server      0.3%  32M",
	}

	for i, p := range processes {
		if y+i >= d.height-4 {
			break
		}
		ctx.Renderer.WriteString(p, x, y+i, dimStyle.Foreground, dimStyle.Background, dimStyle.Attrs)
	}
}

func (d *Dashboard) renderLogs(ctx *mofu.RenderContext, x, y, w int) {
	titleStyle := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
	ctx.Renderer.WriteString("Recent Logs", x, y, titleStyle.Foreground, titleStyle.Background, titleStyle.Attrs)

	y += 2
	infoStyle := mofu.DefaultStyle().Fg(mofu.Hex("7dcfff"))
	warnStyle := mofu.DefaultStyle().Fg(mofu.Hex("f9e2af"))
	errStyle := mofu.DefaultStyle().Fg(mofu.Hex("f38ba8"))

	logs := []struct {
		level string
		msg   string
		style mofu.Style
	}{
		{"INFO", "System started successfully", infoStyle},
		{"INFO", "Connected to database", infoStyle},
		{"WARN", "High memory usage detected", warnStyle},
		{"INFO", "Request processed in 45ms", infoStyle},
		{"ERROR", "Connection timeout to upstream", errStyle},
		{"INFO", "Retry successful", infoStyle},
	}

	for i, log := range logs {
		if y+i >= d.height-4 {
			break
		}
		ctx.Renderer.WriteString(fmt.Sprintf("[%s] %s", log.level, log.msg), x, y+i, log.style.Foreground, log.style.Background, log.style.Attrs)
	}
}

func (d *Dashboard) HandleEvent(event mofu.Event) mofu.Cmd {
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

	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		d.selected--
		if d.selected < 0 {
			d.selected = len(d.items) - 1
		}
		d.status = fmt.Sprintf("Selected: %s", d.items[d.selected])

	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		d.selected++
		if d.selected >= len(d.items) {
			d.selected = 0
		}
		d.status = fmt.Sprintf("Selected: %s", d.items[d.selected])

	case ke.Key == mofu.KeyEnter || (len(ke.Runes) > 0 && ke.Runes[0] == 'l'):
		d.status = fmt.Sprintf("Opened: %s", d.items[d.selected])
	}

	return nil
}

func main() {
	app := NewDashboard()
	if err := mofu.Run(app); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
