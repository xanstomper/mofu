package gadgets

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/xanstomper/mofu"
)

// DataColumn defines a table column.
type DataColumn struct {
	Name     string
	Width    int
	Align    int
	Sortable bool
}

// ---------------------------------------------------------------------------
// RealDataTable — Full-featured data table with sorting, filtering, pagination
// ---------------------------------------------------------------------------

type RealDataTable struct {
	Base
	Columns  []DataColumn
	Rows     [][]string
	Selected int
	Offset   int
	SortCol  int
	SortAsc  bool
	Filter   string
	OnSelect func(row int, data []string)
	OnSort   func(col int, asc bool)
	mu       sync.RWMutex
}

func NewRealDataTable(id string, cols []DataColumn) *RealDataTable {
	return &RealDataTable{Base: *NewBase(id), Columns: cols, Rows: make([][]string, 0)}
}

func (g *RealDataTable) AddRow(row []string) {
	g.mu.Lock()
	g.Rows = append(g.Rows, row)
	g.mu.Unlock()
}

func (g *RealDataTable) RemoveRow(index int) {
	g.mu.Lock()
	if index >= 0 && index < len(g.Rows) {
		g.Rows = append(g.Rows[:index], g.Rows[index+1:]...)
		if g.Selected >= len(g.Rows) {
			g.Selected = len(g.Rows) - 1
		}
	}
	g.mu.Unlock()
}

func (g *RealDataTable) UpdateRow(index int, row []string) {
	g.mu.Lock()
	if index >= 0 && index < len(g.Rows) {
		g.Rows[index] = row
	}
	g.mu.Unlock()
}

func (g *RealDataTable) GetRow(index int) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if index >= 0 && index < len(g.Rows) {
		return g.Rows[index]
	}
	return nil
}

func (g *RealDataTable) GetRows() [][]string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	result := make([][]string, len(g.Rows))
	copy(result, g.Rows)
	return result
}

func (g *RealDataTable) SetRows(rows [][]string) {
	g.mu.Lock()
	g.Rows = rows
	g.Selected = 0
	g.mu.Unlock()
}

func (g *RealDataTable) Sort(col int) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if col == g.SortCol {
		g.SortAsc = !g.SortAsc
	} else {
		g.SortCol = col
		g.SortAsc = true
	}

	sort.SliceStable(g.Rows, func(i, j int) bool {
		if col >= len(g.Rows[i]) || col >= len(g.Rows[j]) {
			return false
		}
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

func (g *RealDataTable) SetFilter(filter string) {
	g.mu.Lock()
	g.Filter = strings.ToLower(filter)
	g.Selected = 0
	g.mu.Unlock()
}

func (g *RealDataTable) filteredRows() [][]string {
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

func (g *RealDataTable) SelectRow(index int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	rows := g.filteredRows()
	if index >= 0 && index < len(rows) {
		g.Selected = index
		if g.OnSelect != nil {
			g.OnSelect(index, rows[index])
		}
	}
}

func (g *RealDataTable) Render(state StateView) []RenderNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var nodes []RenderNode
	header := ""
	for i, col := range g.Columns {
		name := col.Name
		if col.Sortable && i == g.SortCol {
			if g.SortAsc {
				name += " ▲"
			} else {
				name += " ▼"
			}
		}
		header += fmt.Sprintf("%-*s ", col.Width, name)
	}
	nodes = append(nodes, RenderNode{Type: "text", Content: header, Style: mofu.DefaultStyle().WithAttrs(mofu.AttrBold)})
	nodes = append(nodes, RenderNode{Type: "text", Content: strings.Repeat("─", 60)})

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
		text := ""
		for j, col := range g.Columns {
			cell := ""
			if j < len(row) {
				cell = row[j]
			}
			if len(cell) > col.Width {
				cell = cell[:col.Width-2] + ".."
			}
			text += fmt.Sprintf("%-*s ", col.Width, cell)
		}
		style := mofu.DefaultStyle()
		if i == g.Selected {
			style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))
		}
		nodes = append(nodes, RenderNode{Type: "text", Content: text, Style: style})
	}

	status := fmt.Sprintf(" %d rows | %d selected", len(rows), g.Selected+1)
	nodes = append(nodes, RenderNode{Type: "text", Content: status, Style: mofu.DefaultStyle().Fg(mofu.Hex("666666"))})
	return nodes
}

func (g *RealDataTable) OnEvent(e Event) {
	if e.Type == "keypress" {
		if ke, ok := e.Payload.(mofu.KeyEvent); ok {
			g.mu.Lock()
			defer g.mu.Unlock()
			rows := g.filteredRows()
			switch ke.Key {
			case mofu.KeyDown:
				if g.Selected < len(rows)-1 {
					g.Selected++
				}
			case mofu.KeyUp:
				if g.Selected > 0 {
					g.Selected--
				}
			case mofu.KeyHome:
				g.Selected = 0
				g.Offset = 0
			case mofu.KeyEnd:
				g.Selected = len(rows) - 1
			case mofu.KeyEnter:
				if g.OnSelect != nil && g.Selected < len(rows) {
					g.OnSelect(g.Selected, rows[g.Selected])
				}
			}
		}
	}
}

// ---------------------------------------------------------------------------
// RealTimeline — Event timeline with timestamps
// ---------------------------------------------------------------------------

type RealTimelineEvent struct {
	ID        string
	Timestamp time.Time
	Type      string
	Title     string
	Detail    string
}

type RealTimeline struct {
	Base
	Events    []RealTimelineEvent
	MaxEvents int
	Selected  int
	Filter    string
	mu        sync.RWMutex
}

func NewRealTimeline(id string) *RealTimeline {
	return &RealTimeline{Base: *NewBase(id), MaxEvents: 1000, Events: make([]RealTimelineEvent, 0)}
}

func (g *RealTimeline) AddEvent(event RealTimelineEvent) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	if event.ID == "" {
		event.ID = fmt.Sprintf("evt_%d", len(g.Events))
	}
	g.Events = append(g.Events, event)
	if len(g.Events) > g.MaxEvents {
		g.Events = g.Events[len(g.Events)-g.MaxEvents:]
	}
}

func (g *RealTimeline) RemoveEvent(id string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for i, event := range g.Events {
		if event.ID == id {
			g.Events = append(g.Events[:i], g.Events[i+1:]...)
			return
		}
	}
}

func (g *RealTimeline) Clear() {
	g.mu.Lock()
	g.Events = nil
	g.mu.Unlock()
}

func (g *RealTimeline) SetFilter(filter string) {
	g.mu.Lock()
	g.Filter = strings.ToLower(filter)
	g.Selected = 0
	g.mu.Unlock()
}

func (g *RealTimeline) filteredEvents() []RealTimelineEvent {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.Filter == "" {
		return g.Events
	}
	var filtered []RealTimelineEvent
	for _, event := range g.Events {
		if strings.Contains(strings.ToLower(event.Title), g.Filter) ||
			strings.Contains(strings.ToLower(event.Type), g.Filter) {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

func (g *RealTimeline) Render(state StateView) []RenderNode {
	events := g.filteredEvents()
	var nodes []RenderNode
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		timeStr := event.Timestamp.Format("15:04:05")
		style := mofu.DefaultStyle()
		switch strings.ToLower(event.Type) {
		case "error":
			style = mofu.DefaultStyle().Fg(mofu.Hex("f38ba8"))
		case "warn":
			style = mofu.DefaultStyle().Fg(mofu.Hex("f9e2af"))
		case "success":
			style = mofu.DefaultStyle().Fg(mofu.Hex("a6e3a1"))
		}
		text := fmt.Sprintf("[%s] %s: %s", timeStr, event.Title, event.Detail)
		nodes = append(nodes, RenderNode{Type: "text", Content: text, Style: style})
	}
	return nodes
}

func (g *RealTimeline) OnEvent(e Event) {
	if e.Type == "keypress" {
		if ke, ok := e.Payload.(mofu.KeyEvent); ok {
			g.mu.Lock()
			defer g.mu.Unlock()
			events := g.filteredEvents()
			switch ke.Key {
			case mofu.KeyDown:
				if g.Selected < len(events)-1 {
					g.Selected++
				}
			case mofu.KeyUp:
				if g.Selected > 0 {
					g.Selected--
				}
			}
		}
	}
}

// ---------------------------------------------------------------------------
// RealPropertyGrid — Key-value editor
// ---------------------------------------------------------------------------

type PropertyItem struct {
	Key      string
	Value    any
	Type     string
	Editable bool
}

type RealPropertyGrid struct {
	Base
	Properties []PropertyItem
	Selected   int
	Editing    bool
	OnChange   func(key string, old, new any)
	mu         sync.RWMutex
}

func NewRealPropertyGrid(id string) *RealPropertyGrid {
	return &RealPropertyGrid{Base: *NewBase(id), Properties: make([]PropertyItem, 0)}
}

func (g *RealPropertyGrid) SetProperty(key string, value any) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for i, prop := range g.Properties {
		if prop.Key == key {
			old := g.Properties[i].Value
			g.Properties[i].Value = value
			if g.OnChange != nil {
				g.OnChange(key, old, value)
			}
			return
		}
	}
	g.Properties = append(g.Properties, PropertyItem{Key: key, Value: value, Editable: true})
}

func (g *RealPropertyGrid) GetProperty(key string) any {
	g.mu.RLock()
	defer g.mu.RUnlock()
	for _, prop := range g.Properties {
		if prop.Key == key {
			return prop.Value
		}
	}
	return nil
}

func (g *RealPropertyGrid) Clear() {
	g.mu.Lock()
	g.Properties = nil
	g.mu.Unlock()
}

func (g *RealPropertyGrid) Render(state StateView) []RenderNode {
	g.mu.RLock()
	defer g.mu.RUnlock()
	var nodes []RenderNode
	for i, prop := range g.Properties {
		style := mofu.DefaultStyle()
		if i == g.Selected {
			style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))
		}
		text := fmt.Sprintf("%-20s %v", prop.Key+":", prop.Value)
		nodes = append(nodes, RenderNode{Type: "text", Content: text, Style: style})
	}
	return nodes
}

func (g *RealPropertyGrid) OnEvent(e Event) {
	if e.Type == "keypress" {
		if ke, ok := e.Payload.(mofu.KeyEvent); ok {
			g.mu.Lock()
			defer g.mu.Unlock()
			switch ke.Key {
			case mofu.KeyDown:
				if g.Selected < len(g.Properties)-1 {
					g.Selected++
				}
			case mofu.KeyUp:
				if g.Selected > 0 {
					g.Selected--
				}
			}
		}
	}
}

// ---------------------------------------------------------------------------
// RealStatusBar — Multi-section status bar
// ---------------------------------------------------------------------------

type StatusBarSection struct {
	Text  string
	Style mofu.Style
	Align int
}

type RealStatusBar struct {
	Base
	Sections []StatusBarSection
	Width    int
	mu       sync.RWMutex
}

func NewRealStatusBar(id string) *RealStatusBar {
	return &RealStatusBar{Base: *NewBase(id)}
}

func (g *RealStatusBar) AddSection(section StatusBarSection) {
	g.mu.Lock()
	g.Sections = append(g.Sections, section)
	g.mu.Unlock()
}

func (g *RealStatusBar) SetSection(index int, text string) {
	g.mu.Lock()
	if index >= 0 && index < len(g.Sections) {
		g.Sections[index].Text = text
	}
	g.mu.Unlock()
}

func (g *RealStatusBar) Render(state StateView) []RenderNode {
	g.mu.RLock()
	defer g.mu.RUnlock()
	var left, center, right []string
	for _, section := range g.Sections {
		text := section.Text
		switch section.Align {
		case 0:
			left = append(left, text)
		case 1:
			center = append(center, text)
		case 2:
			right = append(right, text)
		}
	}
	leftStr := strings.Join(left, " │ ")
	centerStr := strings.Join(center, " │ ")
	rightStr := strings.Join(right, " │ ")
	bar := leftStr + " " + centerStr + " " + rightStr
	return []RenderNode{{Type: "text", Content: bar, Style: mofu.DefaultStyle().Fg(mofu.Hex("666666"))}}
}

// ---------------------------------------------------------------------------
// RealPagedTable — Paginated data table
// ---------------------------------------------------------------------------

type RealPagedTable struct {
	Base
	Columns    []DataColumn
	AllRows    [][]string
	Page       int
	PageSize   int
	TotalPages int
	OnPageChange func(page int)
	mu         sync.RWMutex
}

func NewRealPagedTable(id string, cols []DataColumn, pageSize int) *RealPagedTable {
	return &RealPagedTable{Base: *NewBase(id), Columns: cols, PageSize: pageSize}
}

func (g *RealPagedTable) SetData(rows [][]string) {
	g.mu.Lock()
	g.AllRows = rows
	g.TotalPages = (len(rows) + g.PageSize - 1) / g.PageSize
	if g.TotalPages < 1 {
		g.TotalPages = 1
	}
	g.Page = 0
	g.mu.Unlock()
}

func (g *RealPagedTable) NextPage() {
	g.mu.Lock()
	if g.Page < g.TotalPages-1 {
		g.Page++
		if g.OnPageChange != nil {
			g.OnPageChange(g.Page)
		}
	}
	g.mu.Unlock()
}

func (g *RealPagedTable) PrevPage() {
	g.mu.Lock()
	if g.Page > 0 {
		g.Page--
		if g.OnPageChange != nil {
			g.OnPageChange(g.Page)
		}
	}
	g.mu.Unlock()
}

func (g *RealPagedTable) CurrentRows() [][]string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	start := g.Page * g.PageSize
	end := start + g.PageSize
	if end > len(g.AllRows) {
		end = len(g.AllRows)
	}
	return g.AllRows[start:end]
}

func (g *RealPagedTable) Render(state StateView) []RenderNode {
	g.mu.RLock()
	defer g.mu.RUnlock()
	var nodes []RenderNode
	header := ""
	for _, col := range g.Columns {
		header += fmt.Sprintf("%-*s ", col.Width, col.Name)
	}
	nodes = append(nodes, RenderNode{Type: "text", Content: header, Style: mofu.DefaultStyle().WithAttrs(mofu.AttrBold)})
	nodes = append(nodes, RenderNode{Type: "text", Content: strings.Repeat("─", 60)})

	rows := g.CurrentRows()
	for _, row := range rows {
		text := ""
		for j, col := range g.Columns {
			cell := ""
			if j < len(row) {
				cell = row[j]
			}
			text += fmt.Sprintf("%-*s ", col.Width, cell)
		}
		nodes = append(nodes, RenderNode{Type: "text", Content: text})
	}

	pageInfo := fmt.Sprintf("Page %d of %d", g.Page+1, g.TotalPages)
	nodes = append(nodes, RenderNode{Type: "text", Content: pageInfo, Style: mofu.DefaultStyle().Fg(mofu.Hex("666666"))})
	return nodes
}

func (g *RealPagedTable) OnEvent(e Event) {
	if e.Type == "keypress" {
		if ke, ok := e.Payload.(mofu.KeyEvent); ok {
			switch ke.Key {
			case mofu.KeyRight:
				g.NextPage()
			case mofu.KeyLeft:
				g.PrevPage()
			}
		}
	}
}
