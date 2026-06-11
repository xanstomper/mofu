package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/xanstomper/mofu"
)

// LogMonitor watches log files and displays them in real-time.
// Features: multi-file monitoring, level filtering, search, auto-scroll.

type LogEntry struct {
	Timestamp time.Time
	Level     string
	Source    string
	Message   string
}

type LogMonitor struct {
	mofu.Minimal
	lines      []LogEntry
	scrollY    int
	filter     string
	level      string
	width      int
	height     int
	focus      int // 0=main, 1=sidebar
	sources    []string
	active     map[string]bool
	totalLines int
	mu         sync.RWMutex
	watchers   []*fileWatcher
}

type fileWatcher struct {
	path   string
	file   *os.File
	reader *bufio.Reader
	done   chan struct{}
}

func NewLogMonitor(dir string) *LogMonitor {
	m := &LogMonitor{
		active:  make(map[string]bool),
		sources: []string{"all"},
	}

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".log") {
			m.sources = append(m.sources, e.Name())
			m.active[e.Name()] = true
			m.watchFile(filepath.Join(dir, e.Name()))
		}
	}

	// Simulated log generator if no log files found
	if len(m.sources) <= 1 {
		m.sources = []string{"all", "api-gateway", "payment-svc", "user-svc", "worker"}
		m.active = map[string]bool{"api-gateway": true, "payment-svc": true, "user-svc": true, "worker": true}
		go m.generateLogs()
	}

	return m
}

func (m *LogMonitor) watchFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}

	fw := &fileWatcher{
		path:   filepath.Base(path),
		file:   file,
		reader: bufio.NewReader(file),
		done:   make(chan struct{}),
	}

	go func() {
		scanner := bufio.NewScanner(fw.reader)
		for scanner.Scan() {
			line := scanner.Text()
			level := "INFO"
			upper := strings.ToUpper(line)
			if strings.Contains(upper, "ERROR") || strings.Contains(upper, "FATAL") {
				level = "ERROR"
			} else if strings.Contains(upper, "WARN") {
				level = "WARN"
			} else if strings.Contains(upper, "DEBUG") {
				level = "DEBUG"
			}

			m.mu.Lock()
			m.lines = append(m.lines, LogEntry{
				Timestamp: time.Now(),
				Level:     level,
				Source:    fw.path,
				Message:   line,
			})
			m.totalLines++
			if len(m.lines) > 10000 {
				m.lines = m.lines[len(m.lines)-10000:]
			}
			// Auto-scroll to bottom
			m.scrollY = len(m.lines) - (m.height - 4)
			if m.scrollY < 0 {
				m.scrollY = 0
			}
			m.mu.Unlock()
		}
	}()

	m.watchers = append(m.watchers, fw)
}

func (m *LogMonitor) generateLogs() {
	sources := []string{"api-gateway", "payment-svc", "user-svc", "worker"}
	levelTemplates := []struct {
		level   string
		message string
	}{
		{"INFO", "GET /api/v1/users 200 OK 12ms"},
		{"INFO", "POST /api/v1/orders 201 Created 45ms"},
		{"WARN", "Slow query detected: SELECT * FROM users WHERE active=true (2340ms)"},
		{"ERROR", "Connection refused: redis:6379 — retrying in 5s"},
		{"INFO", "Cache hit for key: user:1234"},
		{"DEBUG", "Request middleware took 2ms"},
		{"INFO", "Background job completed: send_welcome_email"},
		{"ERROR", "Payment gateway timeout: stripe/v1/charges"},
		{"WARN", "Memory usage at 82% — consider scaling"},
		{"INFO", "Health check: all services healthy"},
	}

	for {
		time.Sleep(time.Duration(100+rand.Intn(400)) * time.Millisecond)

		src := sources[rand.Intn(len(sources))]
		if !m.active[src] {
			continue
		}

		tmpl := levelTemplates[rand.Intn(len(levelTemplates))]

		m.mu.Lock()
		m.lines = append(m.lines, LogEntry{
			Timestamp: time.Now(),
			Level:     tmpl.level,
			Source:    src,
			Message:   tmpl.message,
		})
		m.totalLines++
		if len(m.lines) > 10000 {
			m.lines = m.lines[len(m.lines)-10000:]
		}
		m.mu.Unlock()
	}
}

func (m *LogMonitor) filteredLines() []LogEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []LogEntry
	for _, entry := range m.lines {
		if m.level != "" && entry.Level != m.level {
			continue
		}
		if m.filter != "" && !strings.Contains(strings.ToLower(entry.Message), strings.ToLower(m.filter)) {
			continue
		}
		result = append(result, entry)
	}
	return result
}

func (m *LogMonitor) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	m.mu.RLock()
	m.width = r.Width
	m.height = r.Height
	m.mu.RUnlock()

	leftW := 20
	rightW := r.Width - leftW

	// Sidebar
	sidebarStyle := mofu.DefaultStyle().Fg(mofu.Hex("89b4fa")).WithAttrs(mofu.AttrBold)
	ctx.Renderer.WriteString(" Sources", r.X, r.Y, sidebarStyle.Foreground, sidebarStyle.Background, sidebarStyle.Attrs)

	for i, src := range m.sources {
		y := r.Y + 2 + i
		if y >= r.Y+r.Height-2 {
			break
		}
		icon := "○"
		color := mofu.Hex("585b70")
		if m.active[src] {
			icon = "●"
			color = mofu.Hex("a6e3a1")
		}
		ctx.Renderer.WriteString(fmt.Sprintf(" %s %s", icon, src), r.X, y, color, mofu.ColorBlack, 0)
	}

	// Filter line
	ctx.Renderer.WriteString("│", r.X+leftW-1, r.Y, mofu.Hex("444444"), mofu.ColorBlack, 0)

	// Header
	levelColor := mofu.Hex("cdd6f4")
	if m.level != "" {
		switch m.level {
		case "ERROR":
			levelColor = mofu.Hex("f38ba8")
		case "WARN":
			levelColor = mofu.Hex("fab387")
		case "INFO":
			levelColor = mofu.Hex("a6e3a1")
		case "DEBUG":
			levelColor = mofu.Hex("6c7086")
		}
	}
	header := fmt.Sprintf(" Level: %s  Filter: %s", m.level, m.filter)
	ctx.Renderer.WriteString(header, r.X+leftW, r.Y, levelColor, mofu.ColorBlack, 0)
	ctx.Renderer.WriteString(strings.Repeat("─", rightW-1), r.X+leftW, r.Y+1, mofu.Hex("444444"), mofu.ColorBlack, 0)

	// Log entries
	filtered := m.filteredLines()
	start := m.scrollY
	if start > len(filtered) {
		start = len(filtered)
	}

	y := r.Y + 2
	viewH := r.Height - 4

	for i := start; i < len(filtered) && i-start < viewH; i++ {
		entry := filtered[i]
		ts := entry.Timestamp.Format("15:04:05")

		levelColor := mofu.Hex("cdd6f4")
		switch entry.Level {
		case "ERROR":
			levelColor = mofu.Hex("f38ba8")
		case "WARN":
			levelColor = mofu.Hex("fab387")
		case "INFO":
			levelColor = mofu.Hex("a6e3a1")
		case "DEBUG":
			levelColor = mofu.Hex("6c7086")
		}

		src := fmt.Sprintf("%-12s", entry.Source)
		msg := entry.Message
		if len(msg) > rightW-35 {
			msg = msg[:rightW-38] + "..."
		}

		line := fmt.Sprintf(" %s %-6s %s %s", ts, entry.Level, src, msg)
		if len(line) > rightW-1 {
			line = line[:rightW-1]
		}

		ctx.Renderer.WriteString(line, r.X+leftW, y, levelColor, mofu.ColorBlack, 0)
		y++
	}

	// Stats
	stats := fmt.Sprintf(" %d lines | %d shown | %d sources", m.totalLines, len(filtered), len(m.sources)-1)
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, r.Y+r.Height-2, mofu.Hex("444444"), mofu.ColorBlack, 0)
	ctx.Renderer.WriteString(stats, r.X, r.Y+r.Height-1, mofu.Hex("585b70"), mofu.Hex("1e1e2e"), 0)
}

func (m *LogMonitor) HandleEvent(event mofu.Event) mofu.Cmd {
	if event.Type != mofu.EventKeyPress {
		return nil
	}
	ke := event.Data.(mofu.KeyEvent)

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		for _, fw := range m.watchers {
			close(fw.done)
			fw.file.Close()
		}
		return mofu.QuitCmd()

	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		m.mu.Lock()
		filtered := m.filteredLines()
		if m.scrollY < len(filtered)-(m.height-4) {
			m.scrollY++
		}
		m.mu.Unlock()

	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		m.mu.Lock()
		if m.scrollY > 0 {
			m.scrollY--
		}
		m.mu.Unlock()

	case ke.Key == mofu.KeyPgDn:
		m.mu.Lock()
		m.scrollY += m.height - 4
		m.mu.Unlock()

	case ke.Key == mofu.KeyPgUp:
		m.mu.Lock()
		m.scrollY -= m.height - 4
		if m.scrollY < 0 {
			m.scrollY = 0
		}
		m.mu.Unlock()

	case ke.Key == mofu.KeyHome:
		m.mu.Lock()
		m.scrollY = 0
		m.mu.Unlock()

	case ke.Key == mofu.KeyEnd:
		m.mu.Lock()
		m.scrollY = len(m.filteredLines()) - m.height + 4
		m.mu.Unlock()

	case len(ke.Runes) > 0 && ke.Runes[0] == '1':
		m.mu.Lock()
		m.level = ""
		m.mu.Unlock()

	case len(ke.Runes) > 0 && ke.Runes[0] == '2':
		m.mu.Lock()
		m.level = "ERROR"
		m.mu.Unlock()

	case len(ke.Runes) > 0 && ke.Runes[0] == '3':
		m.mu.Lock()
		m.level = "WARN"
		m.mu.Unlock()

	case len(ke.Runes) > 0 && ke.Runes[0] == '4':
		m.mu.Lock()
		m.level = "INFO"
		m.mu.Unlock()

	case len(ke.Runes) > 0 && ke.Runes[0] == '5':
		m.mu.Lock()
		m.level = "DEBUG"
		m.mu.Unlock()

	case len(ke.Runes) > 0 && ke.Runes[0] == 'c':
		m.mu.Lock()
		m.lines = nil
		m.totalLines = 0
		m.scrollY = 0
		m.mu.Unlock()

	case ke.Key == mofu.KeyCtrlG:
		// Go to bottom
		m.mu.Lock()
		filtered := m.filteredLines()
		m.scrollY = len(filtered) - m.height + 4
		if m.scrollY < 0 {
			m.scrollY = 0
		}
		m.mu.Unlock()
	}
	return nil
}

func main() {
	dir := "."
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}

	app := NewLogMonitor(dir)
	if err := mofu.Run(app); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
