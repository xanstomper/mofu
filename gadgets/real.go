package gadgets

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/xanstomper/mofu"
)

// ---------------------------------------------------------------------------
// REAL Gadgets — Not thin wrappers, actual functionality
// ---------------------------------------------------------------------------

// RealLiveTable is a live-updating table with actual data management.
type RealLiveTable struct {
	Base
	Columns    []string
	Rows       [][]string
	Selected   int
	Offset     int
	SortCol    int
	SortAsc    bool
	Filter     string
	OnSelect   func(row int)
	OnSort     func(col int, asc bool)
	mu         sync.RWMutex
}

func NewRealLiveTable(id string, cols []string) *RealLiveTable {
	return &RealLiveTable{Base: *NewBase(id), Columns: cols}
}

func (g *RealLiveTable) AddRow(row []string) {
	g.mu.Lock()
	g.Rows = append(g.Rows, row)
	g.mu.Unlock()
}

func (g *RealLiveTable) SetRows(rows [][]string) {
	g.mu.Lock()
	g.Rows = rows
	g.mu.Unlock()
}

func (g *RealLiveTable) GetSelectedRow() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.Selected >= 0 && g.Selected < len(g.Rows) {
		return g.Rows[g.Selected]
	}
	return nil
}

func (g *RealLiveTable) GetRows() [][]string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.Rows
}

func (g *RealLiveTable) FilteredRows() [][]string {
	return g.filteredRows()
}

func (g *RealLiveTable) Sort(col int) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if col == g.SortCol {
		g.SortAsc = !g.SortAsc
	} else {
		g.SortCol = col
		g.SortAsc = true
	}

	sort.SliceStable(g.Rows, func(i, j int) bool {
		a, b := g.Rows[i][col], g.Rows[j][col]
		if g.SortAsc {
			return a < b
		}
		return a > b
	})

	if g.OnSort != nil {
		g.OnSort(col, g.SortAsc)
	}
}

func (g *RealLiveTable) SetFilter(filter string) {
	g.mu.Lock()
	g.Filter = strings.ToLower(filter)
	g.Selected = 0
	g.mu.Unlock()
}

func (g *RealLiveTable) filteredRows() [][]string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.Filter == "" {
		return g.Rows
	}

	var filtered [][]string
	for _, row := range g.Rows {
		for _, cell := range row {
			if strings.Contains(strings.ToLower(cell), g.Filter) {
				filtered = append(filtered, row)
				break
			}
		}
	}
	return filtered
}

func (g *RealLiveTable) Render(state StateView) []RenderNode {
	var nodes []RenderNode

	// Header
	header := ""
	for i, col := range g.Columns {
		if i == g.SortCol {
			if g.SortAsc {
				header += col + " ▲ "
			} else {
				header += col + " ▼ "
			}
		} else {
			header += col + "   "
		}
	}
	nodes = append(nodes, RenderNode{
		Type:    "text",
		Content: header,
		Style:   mofu.DefaultStyle().WithAttrs(mofu.AttrBold),
	})

	// Separator
	nodes = append(nodes, RenderNode{
		Type:    "text",
		Content: strings.Repeat("─", 50),
		Style:   mofu.DefaultStyle().Fg(mofu.Hex("444444")),
	})

	// Rows
	rows := g.filteredRows()
	visible := 20
	start := g.Offset
	if start+visible > len(rows) {
		start = len(rows) - visible
	}
	if start < 0 {
		start = 0
	}

	for i := start; i < len(rows) && i < start+visible; i++ {
		row := rows[i]
		text := strings.Join(row, " | ")
		style := mofu.DefaultStyle()
		if i == g.Selected {
			style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))
		}
		nodes = append(nodes, RenderNode{Type: "text", Content: text, Style: style})
	}

	return nodes
}

func (g *RealLiveTable) OnEvent(e Event) {
	if e.Type == "keypress" {
		if ke, ok := e.Payload.(mofu.KeyEvent); ok {
			switch ke.Key {
			case mofu.KeyDown:
				g.mu.Lock()
				if g.Selected < len(g.filteredRows())-1 {
					g.Selected++
				}
				g.mu.Unlock()
			case mofu.KeyUp:
				g.mu.Lock()
				if g.Selected > 0 {
					g.Selected--
				}
				g.mu.Unlock()
			case mofu.KeyEnter:
				if g.OnSelect != nil {
					g.OnSelect(g.Selected)
				}
			}
		}
	}
}

// ---------------------------------------------------------------------------
// RealMetricBoard — Actually tracks and displays metrics
// ---------------------------------------------------------------------------

type RealMetricBoard struct {
	Base
	metrics map[string]*Metric
	mu      sync.RWMutex
}

type Metric struct {
	Value     float64
	Unit      string
	Min       float64
	Max       float64
	History   []float64
	Threshold float64
}

func NewRealMetricBoard(id string) *RealMetricBoard {
	return &RealMetricBoard{
		Base:    *NewBase(id),
		metrics: make(map[string]*Metric),
	}
}

func (g *RealMetricBoard) Set(name string, value float64, unit string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	m, exists := g.metrics[name]
	if !exists {
		m = &Metric{Unit: unit, History: make([]float64, 0, 100)}
		g.metrics[name] = m
	}

	m.Value = value
	m.History = append(m.History, value)
	if len(m.History) > 100 {
		m.History = m.History[1:]
	}

	if value < m.Min || m.Min == 0 {
		m.Min = value
	}
	if value > m.Max {
		m.Max = value
	}
}

func (g *RealMetricBoard) SetThreshold(name string, threshold float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if m, exists := g.metrics[name]; exists {
		m.Threshold = threshold
	}
}

func (g *RealMetricBoard) Get(name string) float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if m, exists := g.metrics[name]; exists {
		return m.Value
	}
	return 0
}

func (g *RealMetricBoard) Render(state StateView) []RenderNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var nodes []RenderNode
	for name, m := range g.metrics {
		// Value with unit
		text := fmt.Sprintf("%-20s %8.2f %s", name+":", m.Value, m.Unit)

		// Color based on threshold
		style := mofu.DefaultStyle()
		if m.Threshold > 0 && m.Value > m.Threshold {
			style = mofu.DefaultStyle().Fg(mofu.Hex("f38ba8"))
		} else if m.Value > m.Max*0.8 {
			style = mofu.DefaultStyle().Fg(mofu.Hex("f9e2af"))
		} else {
			style = mofu.DefaultStyle().Fg(mofu.Hex("a6e3a1"))
		}

		nodes = append(nodes, RenderNode{Type: "text", Content: text, Style: style})

		// Mini sparkline
		if len(m.History) > 1 {
			sparkline := g.renderSparkline(m.History, 20)
			nodes = append(nodes, RenderNode{Type: "text", Content: "  " + sparkline, Style: mofu.DefaultStyle().Fg(mofu.Hex("666666"))})
		}
	}
	return nodes
}

func (g *RealMetricBoard) renderSparkline(data []float64, width int) string {
	if len(data) == 0 {
		return ""
	}

	min, max := data[0], data[0]
	for _, v := range data {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	if max == min {
		max = min + 1
	}

	blocks := []string{" ", "▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}
	result := ""

	// Sample data to fit width
	step := len(data) / width
	if step < 1 {
		step = 1
	}

	for i := 0; i < width && i*step < len(data); i++ {
		idx := i * step
		val := data[idx]
		level := int((val - min) / (max - min) * 7)
		if level < 0 {
			level = 0
		}
		if level > 7 {
			level = 7
		}
		result += blocks[level]
	}

	return result
}

// ---------------------------------------------------------------------------
// RealCommandPalette — Actually searches and executes commands
// ---------------------------------------------------------------------------

type RealCommandPalette struct {
	Base
	commands  []CommandItem
	filtered  []CommandItem
	query     string
	selected  int
	visible   bool
	OnExecute func(cmd CommandItem)
	mu        sync.RWMutex
}

type CommandItem struct {
	Name     string
	Shortcut string
	Category string
	Action   func()
	Disabled bool
}

func NewRealCommandPalette(id string) *RealCommandPalette {
	return &RealCommandPalette{Base: *NewBase(id)}
}

func (g *RealCommandPalette) AddCommand(cmd CommandItem) {
	g.mu.Lock()
	g.commands = append(g.commands, cmd)
	g.mu.Unlock()
}

func (g *RealCommandPalette) Toggle() {
	g.mu.Lock()
	g.visible = !g.visible
	if g.visible {
		g.query = ""
		g.selected = 0
		g.filter()
	}
	g.mu.Unlock()
}

func (g *RealCommandPalette) Show() {
	g.mu.Lock()
	g.visible = true
	g.query = ""
	g.selected = 0
	g.filter()
	g.mu.Unlock()
}

func (g *RealCommandPalette) Hide() {
	g.mu.Lock()
	g.visible = false
	g.mu.Unlock()
}

func (g *RealCommandPalette) IsVisible() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.visible
}

func (g *RealCommandPalette) GetFiltered() []CommandItem {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.filtered
}

func (g *RealCommandPalette) Search(query string) {
	g.mu.Lock()
	g.query = strings.ToLower(query)
	g.selected = 0
	g.filter()
	g.mu.Unlock()
}

func (g *RealCommandPalette) filter() {
	g.filtered = nil
	for _, cmd := range g.commands {
		if cmd.Disabled {
			continue
		}
		if g.query == "" || strings.Contains(strings.ToLower(cmd.Name), g.query) ||
			strings.Contains(strings.ToLower(cmd.Category), g.query) {
			g.filtered = append(g.filtered, cmd)
		}
	}
}

func (g *RealCommandPalette) Execute() {
	g.mu.RLock()
	if g.selected >= 0 && g.selected < len(g.filtered) {
		cmd := g.filtered[g.selected]
		g.mu.RUnlock()

		if cmd.Action != nil {
			cmd.Action()
		}
		if g.OnExecute != nil {
			g.OnExecute(cmd)
		}
		g.Hide()
	} else {
		g.mu.RUnlock()
	}
}

func (g *RealCommandPalette) Render(state StateView) []RenderNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if !g.visible {
		return nil
	}

	var nodes []RenderNode

	// Search box
	nodes = append(nodes, RenderNode{
		Type:    "text",
		Content: " 🔍 " + g.query + "_",
		Style:   mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")),
		ZIndex:  100,
	})

	// Results
	for i, cmd := range g.filtered {
		style := mofu.DefaultStyle()
		if i == g.selected {
			style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))
		}
		text := fmt.Sprintf(" %s %s", cmd.Name, cmd.Shortcut)
		nodes = append(nodes, RenderNode{
			Type:    "text",
			Content: text,
			Style:   style,
			ZIndex:  100,
		})
	}

	return nodes
}

func (g *RealCommandPalette) OnEvent(e Event) {
	if !g.visible {
		return
	}

	if e.Type == "keypress" {
		if ke, ok := e.Payload.(mofu.KeyEvent); ok {
			switch ke.Key {
			case mofu.KeyEsc:
				g.Hide()
			case mofu.KeyDown:
				g.mu.Lock()
				if g.selected < len(g.filtered)-1 {
					g.selected++
				}
				g.mu.Unlock()
			case mofu.KeyUp:
				g.mu.Lock()
				if g.selected > 0 {
					g.selected--
				}
				g.mu.Unlock()
			case mofu.KeyEnter:
				g.Execute()
			default:
				for _, r := range ke.Runes {
					if r == ' ' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
						g.mu.Lock()
						g.query += string(r)
						g.filter()
						g.mu.Unlock()
					}
				}
			}
		}
	}
}

// ---------------------------------------------------------------------------
// RealLogStream — Actually buffers and filters logs
// ---------------------------------------------------------------------------

type RealLogStream struct {
	Base
	lines    []string
	maxLines int
	filter   string
	level    string
	mu       sync.RWMutex
}

func NewRealLogStream(id string) *RealLogStream {
	return &RealLogStream{
		Base:     *NewBase(id),
		maxLines: 1000,
	}
}

func (g *RealLogStream) Append(line string) {
	g.mu.Lock()
	g.lines = append(g.lines, line)
	if len(g.lines) > g.maxLines {
		g.lines = g.lines[len(g.lines)-g.maxLines:]
	}
	g.mu.Unlock()
}

func (g *RealLogStream) SetFilter(filter string) {
	g.mu.Lock()
	g.filter = strings.ToLower(filter)
	g.mu.Unlock()
}

func (g *RealLogStream) SetLevel(level string) {
	g.mu.Lock()
	g.level = strings.ToUpper(level)
	g.mu.Unlock()
}

func (g *RealLogStream) Clear() {
	g.mu.Lock()
	g.lines = nil
	g.mu.Unlock()
}

func (g *RealLogStream) Count() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.lines)
}

func (g *RealLogStream) FilteredLines() []string {
	return g.filteredLines()
}

func (g *RealLogStream) filteredLines() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var result []string
	for _, line := range g.lines {
		if g.filter != "" && !strings.Contains(strings.ToLower(line), g.filter) {
			continue
		}
		if g.level != "" && !strings.Contains(strings.ToUpper(line), g.level) {
			continue
		}
		result = append(result, line)
	}
	return result
}

func (g *RealLogStream) Render(state StateView) []RenderNode {
	lines := g.filteredLines()
	var nodes []RenderNode

	for i := len(lines) - 1; i >= 0 && i >= len(lines)-50; i-- {
		line := lines[i]
		style := mofu.DefaultStyle()

		if strings.Contains(line, "ERROR") || strings.Contains(line, "error") {
			style = mofu.DefaultStyle().Fg(mofu.Hex("f38ba8"))
		} else if strings.Contains(line, "WARN") || strings.Contains(line, "warn") {
			style = mofu.DefaultStyle().Fg(mofu.Hex("f9e2af"))
		} else if strings.Contains(line, "DEBUG") || strings.Contains(line, "debug") {
			style = mofu.DefaultStyle().Fg(mofu.Hex("6c7086"))
		}

		nodes = append(nodes, RenderNode{Type: "text", Content: line, Style: style})
	}
	return nodes
}
