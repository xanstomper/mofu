package gadgets

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/xanstomper/mofu"
)

// =========================================================================
// BATCH 1: Data & Visualization Gadgets (15 gadgets)
// =========================================================================

// Gadget 1: RealHeatMap — Density-based value visualization
type RealHeatMap struct {
	Base
	Data    [][]float64
	Min     float64
	Max     float64
	Width   int
	Height  int
	Labels  []string
	OnCell  func(x, y int, value float64)
	mu      sync.RWMutex
}

func NewRealHeatMap(id string, w, h int) *RealHeatMap {
	return &RealHeatMap{Base: *NewBase(id), Width: w, Height: h, Min: 0, Max: 1}
}

func (g *RealHeatMap) SetData(data [][]float64) {
	g.mu.Lock()
	g.Data = data
	if len(data) > 0 && len(data[0]) > 0 {
		g.Min = data[0][0]
		g.Max = data[0][0]
		for _, row := range data {
			for _, v := range row {
				if v < g.Min {
					g.Min = v
				}
				if v > g.Max {
					g.Max = v
				}
			}
		}
	}
	g.mu.Unlock()
}

func (g *RealHeatMap) SetValue(x, y int, value float64) {
	g.mu.Lock()
	if y >= 0 && y < len(g.Data) && x >= 0 && x < len(g.Data[y]) {
		g.Data[y][x] = value
		if value < g.Min {
			g.Min = value
		}
		if value > g.Max {
			g.Max = value
		}
	}
	g.mu.Unlock()
}

func (g *RealHeatMap) GetStats() (min, max, avg float64, count int) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if len(g.Data) == 0 {
		return
	}
	min = g.Data[0][0]
	max = g.Data[0][0]
	sum := 0.0
	count = 0
	for _, row := range g.Data {
		for _, v := range row {
			sum += v
			count++
			if v < min {
				min = v
			}
			if v > max {
				max = v
			}
		}
	}
	if count > 0 {
		avg = sum / float64(count)
	}
	return
}

func (g *RealHeatMap) Render(state StateView) []RenderNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var nodes []RenderNode
	density := " ·░▒▓█"

	for _, row := range g.Data {
		line := ""
		for _, v := range row {
			idx := 0
			if g.Max > g.Min {
				idx = int((v - g.Min) / (g.Max - g.Min) * 7)
			}
			if idx < 0 {
				idx = 0
			}
			if idx >= len(density) {
				idx = len(density) - 1
			}
			line += string(density[idx])
		}
		nodes = append(nodes, RenderNode{Type: "text", Content: line})
	}
	return nodes
}

// Gadget 2: RealSparkline — Sparkline chart
type RealSparkline struct {
	Base
	Values    []float64
	Width     int
	Height    int
	Color     string
	Label     string
	Min       float64
	Max       float64
	ShowStats bool
	mu        sync.RWMutex
}

func NewRealSparkline(id string, w int) *RealSparkline {
	return &RealSparkline{Base: *NewBase(id), Width: w, Height: 8, Color: "a6e3a1"}
}

func (g *RealSparkline) AddValue(value float64) {
	g.mu.Lock()
	g.Values = append(g.Values, value)
	if len(g.Values) > g.Width {
		g.Values = g.Values[len(g.Values)-g.Width:]
	}
	if len(g.Values) == 0 || value < g.Min {
		g.Min = value
	}
	if len(g.Values) == 0 || value > g.Max {
		g.Max = value
	}
	g.mu.Unlock()
}

func (g *RealSparkline) SetValues(values []float64) {
	g.mu.Lock()
	g.Values = values
	if len(values) > 0 {
		g.Min = values[0]
		g.Max = values[0]
		for _, v := range values {
			if v < g.Min {
				g.Min = v
			}
			if v > g.Max {
				g.Max = v
			}
		}
	}
	g.mu.Unlock()
}

func (g *RealSparkline) GetStats() (min, max, avg, current float64) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if len(g.Values) == 0 {
		return
	}
	min = g.Values[0]
	max = g.Values[0]
	sum := 0.0
	for _, v := range g.Values {
		sum += v
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	avg = sum / float64(len(g.Values))
	current = g.Values[len(g.Values)-1]
	return
}

func (g *RealSparkline) Render(state StateView) []RenderNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var nodes []RenderNode
	if g.Label != "" {
		nodes = append(nodes, RenderNode{Type: "text", Content: g.Label, Style: mofu.DefaultStyle().WithAttrs(mofu.AttrBold)})
	}

	if len(g.Values) == 0 {
		nodes = append(nodes, RenderNode{Type: "text", Content: "No data"})
		return nodes
	}

	blocks := []string{" ", "▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}
	line := ""
	for _, v := range g.Values {
		idx := 0
		if g.Max > g.Min {
			idx = int((v - g.Min) / (g.Max - g.Min) * 7)
		}
		if idx < 0 {
			idx = 0
		}
		if idx >= len(blocks) {
			idx = len(blocks) - 1
		}
		line += blocks[idx]
	}
	nodes = append(nodes, RenderNode{Type: "text", Content: line, Style: mofu.DefaultStyle().Fg(mofu.Hex(g.Color))})

	if g.ShowStats {
		min, max, avg, current := g.GetStats()
		stats := fmt.Sprintf("Min: %.2f | Max: %.2f | Avg: %.2f | Now: %.2f", min, max, avg, current)
		nodes = append(nodes, RenderNode{Type: "text", Content: stats, Style: mofu.DefaultStyle().Fg(mofu.Hex("666666"))})
	}
	return nodes
}

// Gadget 3: RealProgressBar — Multi-segment progress bar
type RealProgressBar struct {
	Base
	Value     float64
	Max       float64
	Width     int
	Label     string
	ShowPct   bool
	ShowValue bool
	FillChar  string
	EmptyChar string
	mu        sync.RWMutex
}

func NewRealProgressBar(id string) *RealProgressBar {
	return &RealProgressBar{Base: *NewBase(id), Max: 100, Width: 30, FillChar: "█", EmptyChar: "░"}
}

func (g *RealProgressBar) SetValue(value float64) {
	g.mu.Lock()
	g.Value = value
	if g.Value > g.Max {
		g.Value = g.Max
	}
	if g.Value < 0 {
		g.Value = 0
	}
	g.mu.Unlock()
}

func (g *RealProgressBar) Increment(amount float64) {
	g.mu.Lock()
	g.Value += amount
	if g.Value > g.Max {
		g.Value = g.Max
	}
	g.mu.Unlock()
}

func (g *RealProgressBar) Reset() {
	g.mu.Lock()
	g.Value = 0
	g.mu.Unlock()
}

func (g *RealProgressBar) GetPercentage() float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.Max == 0 {
		return 0
	}
	return (g.Value / g.Max) * 100
}

func (g *RealProgressBar) Render(state StateView) []RenderNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var nodes []RenderNode
	if g.Label != "" {
		nodes = append(nodes, RenderNode{Type: "text", Content: g.Label, Style: mofu.DefaultStyle().WithAttrs(mofu.AttrBold)})
	}

	pct := g.GetPercentage()
	filled := int(pct / 100 * float64(g.Width))
	empty := g.Width - filled

	bar := strings.Repeat(g.FillChar, filled) + strings.Repeat(g.EmptyChar, empty)

	// Color based on percentage
	color := "a6e3a1" // green
	if pct > 80 {
		color = "f38ba8" // red
	} else if pct > 60 {
		color = "f9e2af" // yellow
	}

	nodes = append(nodes, RenderNode{Type: "text", Content: bar, Style: mofu.DefaultStyle().Fg(mofu.Hex(color))})

	if g.ShowPct || g.ShowValue {
		text := ""
		if g.ShowPct {
			text += fmt.Sprintf(" %.0f%%", pct)
		}
		if g.ShowValue {
			text += fmt.Sprintf(" (%.0f/%.0f)", g.Value, g.Max)
		}
		nodes = append(nodes, RenderNode{Type: "text", Content: text, Style: mofu.DefaultStyle().Fg(mofu.Hex("666666"))})
	}
	return nodes
}

// Gadget 4: RealDonut — ASCII donut chart
type RealDonut struct {
	Base
	Segments []DonutSegment
	Size     int
	Label    string
	mu       sync.RWMutex
}

type DonutSegment struct {
	Label  string
	Value  float64
	Color  string
}

func NewRealDonut(id string, size int) *RealDonut {
	return &RealDonut{Base: *NewBase(id), Size: size, Segments: make([]DonutSegment, 0)}
}

func (g *RealDonut) AddSegment(segment DonutSegment) {
	g.mu.Lock()
	g.Segments = append(g.Segments, segment)
	g.mu.Unlock()
}

func (g *RealDonut) Clear() {
	g.mu.Lock()
	g.Segments = nil
	g.mu.Unlock()
}

func (g *RealDonut) GetTotal() float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	total := 0.0
	for _, seg := range g.Segments {
		total += seg.Value
	}
	return total
}

func (g *RealDonut) Render(state StateView) []RenderNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var nodes []RenderNode
	if g.Label != "" {
		nodes = append(nodes, RenderNode{Type: "text", Content: g.Label, Style: mofu.DefaultStyle().WithAttrs(mofu.AttrBold)})
	}

	if len(g.Segments) == 0 {
		nodes = append(nodes, RenderNode{Type: "text", Content: "No data"})
		return nodes
	}

	total := g.GetTotal()
	if total == 0 {
		nodes = append(nodes, RenderNode{Type: "text", Content: "No data"})
		return nodes
	}

	// Simple ASCII donut
	size := g.Size
	if size < 3 {
		size = 3
	}
	center := size / 2

	for y := 0; y < size; y++ {
		line := ""
		for x := 0; x < size; x++ {
			dx := float64(x - center)
			dy := float64(y - center)
			dist := math.Sqrt(dx*dx + dy*dy)
			radius := float64(size) / 2.0

			if dist > radius*0.3 && dist < radius*0.9 {
				// In the ring
				angle := math.Atan2(dy, dx) + math.Pi
				pct := angle / (2 * math.Pi)
				acc := 0.0
				for _, seg := range g.Segments {
					acc += seg.Value / total
					if pct <= acc {
						line += "█"
						break
					}
				}
				if len(line) == 0 || line[len(line)-1:] != "█" {
					line += " "
				}
			} else {
				line += " "
			}
		}
		nodes = append(nodes, RenderNode{Type: "text", Content: line})
	}

	// Legend
	for _, seg := range g.Segments {
		pct := seg.Value / total * 100
		text := fmt.Sprintf("  %s: %.1f%%", seg.Label, pct)
		nodes = append(nodes, RenderNode{Type: "text", Content: text})
	}
	return nodes
}

// Gadget 5: RealGauge — Circular gauge display
type RealGauge struct {
	Base
	Value    float64
	Min      float64
	Max      float64
	Label    string
	Unit     string
	Size     int
	mu       sync.RWMutex
}

func NewRealGauge(id string) *RealGauge {
	return &RealGauge{Base: *NewBase(id), Max: 100, Size: 10}
}

func (g *RealGauge) SetValue(value float64) {
	g.mu.Lock()
	g.Value = value
	if g.Value > g.Max {
		g.Value = g.Max
	}
	if g.Value < g.Min {
		g.Value = g.Min
	}
	g.mu.Unlock()
}

func (g *RealGauge) GetPercentage() float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.Max == g.Min {
		return 0
	}
	return (g.Value - g.Min) / (g.Max - g.Min) * 100
}

func (g *RealGauge) Render(state StateView) []RenderNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var nodes []RenderNode
	pct := g.GetPercentage()

	// Simple bar gauge
	barWidth := 20
	filled := int(pct / 100 * float64(barWidth))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	text := fmt.Sprintf("%s [%s] %.1f%%%s", g.Label, bar, pct, g.Unit)
	nodes = append(nodes, RenderNode{Type: "text", Content: text})
	return nodes
}

// Gadget 6: RealTimer — Stopwatch/timer display
type RealTimer struct {
	Base
	StartTime time.Time
	Running   bool
	LapTimes  []time.Duration
	Label     string
	Elapsed   time.Duration
	mu        sync.RWMutex
}

func NewRealTimer(id string) *RealTimer {
	return &RealTimer{Base: *NewBase(id)}
}

func (g *RealTimer) Start() {
	g.mu.Lock()
	g.StartTime = time.Now()
	g.Running = true
	g.mu.Unlock()
}

func (g *RealTimer) Stop() {
	g.mu.Lock()
	if g.Running {
		g.Elapsed = time.Since(g.StartTime)
		g.Running = false
	}
	g.mu.Unlock()
}

func (g *RealTimer) Reset() {
	g.mu.Lock()
	g.StartTime = time.Time{}
	g.Running = false
	g.Elapsed = 0
	g.LapTimes = nil
	g.mu.Unlock()
}

func (g *RealTimer) Lap() {
	g.mu.Lock()
	if g.Running {
		g.LapTimes = append(g.LapTimes, time.Since(g.StartTime))
	}
	g.mu.Unlock()
}

func (g *RealTimer) GetElapsed() time.Duration {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.Running {
		return time.Since(g.StartTime)
	}
	return g.Elapsed
}

func (g *RealTimer) Render(state StateView) []RenderNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var nodes []RenderNode
	elapsed := g.GetElapsed()
	text := fmt.Sprintf("%s: %s", g.Label, elapsed.Round(time.Millisecond))
	nodes = append(nodes, RenderNode{Type: "text", Content: text, Style: mofu.DefaultStyle().Fg(mofu.Hex("a6e3a1"))})

	for i, lap := range g.LapTimes {
		nodes = append(nodes, RenderNode{
			Type:    "text",
			Content: fmt.Sprintf("  Lap %d: %s", i+1, lap.Round(time.Millisecond)),
			Style:   mofu.DefaultStyle().Fg(mofu.Hex("666666")),
		})
	}
	return nodes
}

// Gadget 7: RealNotification — Toast notification system
type RealNotification struct {
	Base
	Notifications []NotificationItem
	MaxItems      int
	mu            sync.RWMutex
}

type NotificationItem struct {
	ID        string
	Message   string
	Level     string // "info", "success", "warning", "error"
	Timestamp time.Time
	Read      bool
}

func NewRealNotification(id string) *RealNotification {
	return &RealNotification{Base: *NewBase(id), MaxItems: 50}
}

func (g *RealNotification) Add(message, level string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.Notifications = append(g.Notifications, NotificationItem{
		ID:        fmt.Sprintf("notif_%d", len(g.Notifications)),
		Message:   message,
		Level:     level,
		Timestamp: time.Now(),
	})
	if len(g.Notifications) > g.MaxItems {
		g.Notifications = g.Notifications[len(g.Notifications)-g.MaxItems:]
	}
}

func (g *RealNotification) MarkRead(id string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for i := range g.Notifications {
		if g.Notifications[i].ID == id {
			g.Notifications[i].Read = true
			return
		}
	}
}

func (g *RealNotification) Clear() {
	g.mu.Lock()
	g.Notifications = nil
	g.mu.Unlock()
}

func (g *RealNotification) UnreadCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	count := 0
	for _, n := range g.Notifications {
		if !n.Read {
			count++
		}
	}
	return count
}

func (g *RealNotification) Render(state StateView) []RenderNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var nodes []RenderNode
	for _, n := range g.Notifications {
		timeStr := n.Timestamp.Format("15:04")
		style := mofu.DefaultStyle()
		switch n.Level {
		case "error":
			style = mofu.DefaultStyle().Fg(mofu.Hex("f38ba8"))
		case "warning":
			style = mofu.DefaultStyle().Fg(mofu.Hex("f9e2af"))
		case "success":
			style = mofu.DefaultStyle().Fg(mofu.Hex("a6e3a1"))
		}
		if n.Read {
			style = mofu.DefaultStyle().Fg(mofu.Hex("666666"))
		}
		text := fmt.Sprintf("[%s] %s", timeStr, n.Message)
		nodes = append(nodes, RenderNode{Type: "text", Content: text, Style: style})
	}
	return nodes
}

// Gadget 8: RealAccordion — Collapsible sections
type RealAccordion struct {
	Base
	Sections    []AccordionSection
	Expanded    map[int]bool
	Selected    int
	mu          sync.RWMutex
}

type AccordionSection struct {
	Title   string
	Content string
}

func NewRealAccordion(id string) *RealAccordion {
	return &RealAccordion{Base: *NewBase(id), Expanded: make(map[int]bool)}
}

func (g *RealAccordion) AddSection(section AccordionSection) {
	g.mu.Lock()
	g.Sections = append(g.Sections, section)
	g.mu.Unlock()
}

func (g *RealAccordion) Toggle(index int) {
	g.mu.Lock()
	g.Expanded[index] = !g.Expanded[index]
	g.mu.Unlock()
}

func (g *RealAccordion) Expand(index int) {
	g.mu.Lock()
	g.Expanded[index] = true
	g.mu.Unlock()
}

func (g *RealAccordion) Collapse(index int) {
	g.mu.Lock()
	g.Expanded[index] = false
	g.mu.Unlock()
}

func (g *RealAccordion) Render(state StateView) []RenderNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var nodes []RenderNode
	for i, section := range g.Sections {
		icon := "▶"
		if g.Expanded[i] {
			icon = "▼"
		}
		text := fmt.Sprintf("%s %s", icon, section.Title)
		style := mofu.DefaultStyle()
		if i == g.Selected {
			style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))
		}
		nodes = append(nodes, RenderNode{Type: "text", Content: text, Style: style})

		if g.Expanded[i] && section.Content != "" {
			lines := strings.Split(section.Content, "\n")
			for _, line := range lines {
				nodes = append(nodes, RenderNode{Type: "text", Content: "  " + line, Style: mofu.DefaultStyle().Fg(mofu.Hex("666666"))})
			}
		}
	}
	return nodes
}

func (g *RealAccordion) OnEvent(e Event) {
	if e.Type == "keypress" {
		if ke, ok := e.Payload.(mofu.KeyEvent); ok {
			g.mu.Lock()
			defer g.mu.Unlock()
			switch ke.Key {
			case mofu.KeyDown:
				if g.Selected < len(g.Sections)-1 {
					g.Selected++
				}
			case mofu.KeyUp:
				if g.Selected > 0 {
					g.Selected--
				}
			case mofu.KeyEnter:
				g.Expanded[g.Selected] = !g.Expanded[g.Selected]
			}
		}
	}
}

// Gadget 9: RealTabs — Tab navigation
type RealTabs struct {
	Base
	Tabs     []TabItem
	Selected int
	OnChange func(index int)
	mu       sync.RWMutex
}

type TabItem struct {
	Label    string
	Content  string
	Disabled bool
}

func NewRealTabs(id string) *RealTabs {
	return &RealTabs{Base: *NewBase(id)}
}

func (g *RealTabs) AddTab(tab TabItem) {
	g.mu.Lock()
	g.Tabs = append(g.Tabs, tab)
	g.mu.Unlock()
}

func (g *RealTabs) SetSelected(index int) {
	g.mu.Lock()
	if index >= 0 && index < len(g.Tabs) && !g.Tabs[index].Disabled {
		g.Selected = index
		if g.OnChange != nil {
			g.OnChange(index)
		}
	}
	g.mu.Unlock()
}

func (g *RealTabs) GetContent() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.Selected >= 0 && g.Selected < len(g.Tabs) {
		return g.Tabs[g.Selected].Content
	}
	return ""
}

func (g *RealTabs) Render(state StateView) []RenderNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var nodes []RenderNode
	// Tab bar
	tabBar := ""
	for i, tab := range g.Tabs {
		if i == g.Selected {
			tabBar += " [" + tab.Label + "] "
		} else {
			tabBar += " " + tab.Label + " "
		}
	}
	nodes = append(nodes, RenderNode{Type: "text", Content: tabBar, Style: mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))})

	// Content
	content := g.GetContent()
	if content != "" {
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			nodes = append(nodes, RenderNode{Type: "text", Content: line})
		}
	}
	return nodes
}

func (g *RealTabs) OnEvent(e Event) {
	if e.Type == "keypress" {
		if ke, ok := e.Payload.(mofu.KeyEvent); ok {
			g.mu.Lock()
			defer g.mu.Unlock()
			switch ke.Key {
			case mofu.KeyLeft:
				if g.Selected > 0 {
					g.Selected--
				}
			case mofu.KeyRight:
				if g.Selected < len(g.Tabs)-1 {
					g.Selected++
				}
			}
		}
	}
}

// Gadget 10: RealBreadcrumb — Navigation breadcrumb
type RealBreadcrumb struct {
	Base
	Items   []BreadcrumbItem
	Sep     string
	OnClick func(index int)
	mu      sync.RWMutex
}

type BreadcrumbItem struct {
	Label    string
	Active   bool
	Disabled bool
}

func NewRealBreadcrumb(id string) *RealBreadcrumb {
	return &RealBreadcrumb{Base: *NewBase(id), Sep: " / "}
}

func (g *RealBreadcrumb) AddItem(item BreadcrumbItem) {
	g.mu.Lock()
	g.Items = append(g.Items, item)
	g.mu.Unlock()
}

func (g *RealBreadcrumb) SetItems(items []BreadcrumbItem) {
	g.mu.Lock()
	g.Items = items
	g.mu.Unlock()
}

func (g *RealBreadcrumb) NavigateTo(index int) {
	g.mu.Lock()
	for i := range g.Items {
		g.Items[i].Active = (i == index)
	}
	if g.OnClick != nil {
		g.OnClick(index)
	}
	g.mu.Unlock()
}

func (g *RealBreadcrumb) Render(state StateView) []RenderNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var nodes []RenderNode
	parts := []string{}
	for i, item := range g.Items {
		text := item.Label
		if item.Active {
			text = "▸ " + text
		}
		parts = append(parts, text)
		_ = i
	}
	nodes = append(nodes, RenderNode{Type: "text", Content: strings.Join(parts, g.Sep)})
	return nodes
}

// Gadget 11: RealBadge — Count badge
type RealBadge struct {
	Base
	Count   int
	Label   string
	Max     int
	Style_  string // "default", "success", "warning", "error"
	mu      sync.RWMutex
}

func NewRealBadge(id string) *RealBadge {
	return &RealBadge{Base: *NewBase(id), Max: 99, Style_: "default"}
}

func (g *RealBadge) SetCount(count int) {
	g.mu.Lock()
	g.Count = count
	g.mu.Unlock()
}

func (g *RealBadge) Increment() {
	g.mu.Lock()
	g.Count++
	g.mu.Unlock()
}

func (g *RealBadge) Decrement() {
	g.mu.Lock()
	if g.Count > 0 {
		g.Count--
	}
	g.mu.Unlock()
}

func (g *RealBadge) Reset() {
	g.mu.Lock()
	g.Count = 0
	g.mu.Unlock()
}

func (g *RealBadge) Render(state StateView) []RenderNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	text := g.Label
	if g.Count > 0 {
		display := g.Count
		if display > g.Max {
			display = g.Max
			text += fmt.Sprintf("%d+", display)
		} else {
			text += fmt.Sprintf("%d", display)
		}
	}

	style := mofu.DefaultStyle()
	switch g.Style_ {
	case "success":
		style = mofu.DefaultStyle().Fg(mofu.Hex("a6e3a1"))
	case "warning":
		style = mofu.DefaultStyle().Fg(mofu.Hex("f9e2af"))
	case "error":
		style = mofu.DefaultStyle().Fg(mofu.Hex("f38ba8"))
	}

	return []RenderNode{{Type: "text", Content: text, Style: style}}
}

// Gadget 12: RealTooltip — Hover tooltip
type RealTooltip struct {
	Base
	Text    string
	Visible bool
	X, Y    int
	mu      sync.RWMutex
}

func NewRealTooltip(id string) *RealTooltip {
	return &RealTooltip{Base: *NewBase(id)}
}

func (g *RealTooltip) Show(text string) {
	g.mu.Lock()
	g.Text = text
	g.Visible = true
	g.mu.Unlock()
}

func (g *RealTooltip) Hide() {
	g.mu.Lock()
	g.Visible = false
	g.mu.Unlock()
}

func (g *RealTooltip) Toggle(text string) {
	g.mu.Lock()
	if g.Visible {
		g.Visible = false
	} else {
		g.Text = text
		g.Visible = true
	}
	g.mu.Unlock()
}

func (g *RealTooltip) Render(state StateView) []RenderNode {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if !g.Visible || g.Text == "" {
		return nil
	}
	return []RenderNode{
		{Type: "text", Content: "╭─" + g.Text + "─╮", Style: mofu.DefaultStyle().Fg(mofu.Hex("666666"))},
		{Type: "text", Content: "╰" + strings.Repeat("─", len(g.Text)+2) + "╯", Style: mofu.DefaultStyle().Fg(mofu.Hex("666666"))},
	}
}

// Gadget 13: RealContextMenu — Right-click context menu
type RealContextMenu struct {
	Base
	Items     []ContextMenuItem
	Visible   bool
	Selected  int
	X, Y      int
	OnSelect  func(index int)
	mu        sync.RWMutex
}

type ContextMenuItem struct {
	Label    string
	Icon     string
	Shortcut string
	Disabled bool
	Separator bool
}

func NewRealContextMenu(id string) *RealContextMenu {
	return &RealContextMenu{Base: *NewBase(id)}
}

func (g *RealContextMenu) Show(x, y int) {
	g.mu.Lock()
	g.Visible = true
	g.X = x
	g.Y = y
	g.Selected = 0
	g.mu.Unlock()
}

func (g *RealContextMenu) Hide() {
	g.mu.Lock()
	g.Visible = false
	g.mu.Unlock()
}

func (g *RealContextMenu) AddItem(item ContextMenuItem) {
	g.mu.Lock()
	g.Items = append(g.Items, item)
	g.mu.Unlock()
}

func (g *RealContextMenu) Render(state StateView) []RenderNode {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if !g.Visible {
		return nil
	}

	var nodes []RenderNode
	for i, item := range g.Items {
		if item.Separator {
			nodes = append(nodes, RenderNode{Type: "text", Content: strings.Repeat("─", 20)})
			continue
		}
		text := fmt.Sprintf(" %s %s", item.Icon, item.Label)
		if item.Shortcut != "" {
			text += "  " + item.Shortcut
		}
		style := mofu.DefaultStyle()
		if i == g.Selected {
			style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))
		}
		if item.Disabled {
			style = mofu.DefaultStyle().Fg(mofu.Hex("666666"))
		}
		nodes = append(nodes, RenderNode{Type: "text", Content: text, Style: style})
	}
	return nodes
}

func (g *RealContextMenu) OnEvent(e Event) {
	if e.Type == "keypress" {
		if ke, ok := e.Payload.(mofu.KeyEvent); ok {
			g.mu.Lock()
			defer g.mu.Unlock()
			switch ke.Key {
			case mofu.KeyDown:
				g.Selected++
				if g.Selected >= len(g.Items) {
					g.Selected = 0
				}
			case mofu.KeyUp:
				g.Selected--
				if g.Selected < 0 {
					g.Selected = len(g.Items) - 1
				}
			case mofu.KeyEnter:
				if g.OnSelect != nil {
					g.OnSelect(g.Selected)
				}
				g.Visible = false
			case mofu.KeyEsc:
				g.Visible = false
			}
		}
	}
}

// Gadget 14: RealToast — Auto-dismiss notification
type RealToast struct {
	Base
	Messages  []ToastMessage
	MaxItems  int
	mu        sync.RWMutex
}

type ToastMessage struct {
	Text      string
	Level     string
	Timestamp time.Time
}

func NewRealToast(id string) *RealToast {
	return &RealToast{Base: *NewBase(id), MaxItems: 5}
}

func (g *RealToast) Show(text, level string) {
	g.mu.Lock()
	g.Messages = append(g.Messages, ToastMessage{
		Text:      text,
		Level:     level,
		Timestamp: time.Now(),
	})
	if len(g.Messages) > g.MaxItems {
		g.Messages = g.Messages[len(g.Messages)-g.MaxItems:]
	}
	g.mu.Unlock()
}

func (g *RealToast) Clear() {
	g.mu.Lock()
	g.Messages = nil
	g.mu.Unlock()
}

func (g *RealToast) Render(state StateView) []RenderNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var nodes []RenderNode
	for _, msg := range g.Messages {
		style := mofu.DefaultStyle()
		switch msg.Level {
		case "error":
			style = mofu.DefaultStyle().Fg(mofu.Hex("f38ba8"))
		case "warning":
			style = mofu.DefaultStyle().Fg(mofu.Hex("f9e2af"))
		case "success":
			style = mofu.DefaultStyle().Fg(mofu.Hex("a6e3a1"))
		}
		nodes = append(nodes, RenderNode{Type: "text", Content: msg.Text, Style: style})
	}
	return nodes
}

// Gadget 15: RealChip — Removable tag/chip
type RealChip struct {
	Base
	Label    string
	Color    string
	Removable bool
	OnRemove func()
	mu       sync.RWMutex
}

func NewRealChip(id, label string) *RealChip {
	return &RealChip{Base: *NewBase(id), Label: label, Color: "89b4fa", Removable: true}
}

func (g *RealChip) SetLabel(label string) {
	g.mu.Lock()
	g.Label = label
	g.mu.Unlock()
}

func (g *RealChip) Remove() {
	g.mu.Lock()
	if g.OnRemove != nil {
		g.OnRemove()
	}
	g.mu.Unlock()
}

func (g *RealChip) Render(state StateView) []RenderNode {
	g.mu.RLock()
	defer g.mu.RUnlock()
	text := g.Label
	if g.Removable {
		text += " ✕"
	}
	return []RenderNode{{Type: "text", Content: "[" + text + "]", Style: mofu.DefaultStyle().Fg(mofu.Hex(g.Color))}}
}
