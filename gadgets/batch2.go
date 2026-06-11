package gadgets

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/xanstomper/mofu"
)

// =========================================================================
// BATCH 2: System & Monitoring Gadgets (15 gadgets)
// =========================================================================

// Gadget 16: RealSystemMonitor — System metrics display
type RealSystemMonitor struct {
	Base
	Metrics   map[string]float64
	Labels    map[string]string
	Thresholds map[string]float64
	History   map[string][]float64
	mu        sync.RWMutex
}

func NewRealSystemMonitor(id string) *RealSystemMonitor {
	return &RealSystemMonitor{
		Base:       *NewBase(id),
		Metrics:    make(map[string]float64),
		Labels:     make(map[string]string),
		Thresholds: make(map[string]float64),
		History:    make(map[string][]float64),
	}
}

func (g *RealSystemMonitor) Set(name string, value float64) {
	g.mu.Lock()
	g.Metrics[name] = value
	g.History[name] = append(g.History[name], value)
	if len(g.History[name]) > 100 {
		g.History[name] = g.History[name][len(g.History[name])-100:]
	}
	g.mu.Unlock()
}

func (g *RealSystemMonitor) SetLabel(name, label string) {
	g.mu.Lock()
	g.Labels[name] = label
	g.mu.Unlock()
}

func (g *RealSystemMonitor) SetThreshold(name string, threshold float64) {
	g.mu.Lock()
	g.Thresholds[name] = threshold
	g.mu.Unlock()
}

func (g *RealSystemMonitor) Get(name string) float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.Metrics[name]
}

func (g *RealSystemMonitor) GetHistory(name string) []float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.History[name]
}

func (g *RealSystemMonitor) Render(state StateView) []RenderNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var nodes []RenderNode
	for name, value := range g.Metrics {
		label := name
		if l, ok := g.Labels[name]; ok {
			label = l
		}

		style := mofu.DefaultStyle()
		if threshold, ok := g.Thresholds[name]; ok && value > threshold {
			style = mofu.DefaultStyle().Fg(mofu.Hex("f38ba8"))
		} else if value > 80 {
			style = mofu.DefaultStyle().Fg(mofu.Hex("f9e2af"))
		} else {
			style = mofu.DefaultStyle().Fg(mofu.Hex("a6e3a1"))
		}

		text := fmt.Sprintf("%-20s %8.2f", label+":", value)
		nodes = append(nodes, RenderNode{Type: "text", Content: text, Style: style})

		// Sparkline
		if history, ok := g.History[name]; ok && len(history) > 1 {
			sparkline := g.renderSparkline(history, 20)
			nodes = append(nodes, RenderNode{Type: "text", Content: "  " + sparkline, Style: mofu.DefaultStyle().Fg(mofu.Hex("666666"))})
		}
	}
	return nodes
}

func (g *RealSystemMonitor) renderSparkline(data []float64, width int) string {
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
	step := len(data) / width
	if step < 1 {
		step = 1
	}
	for i := 0; i < width && i*step < len(data); i++ {
		idx := i * step
		level := int((data[idx] - min) / (max - min) * 7)
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

// Gadget 17: RealProcessList — Process list display
type RealProcessList struct {
	Base
	Processes []ProcessInfo2
	Selected  int
	SortBy    string
	mu        sync.RWMutex
}

type ProcessInfo2 struct {
	PID    int
	Name   string
	Status string
	CPU    float64
	Memory float64
	Ports  []int
}

func NewRealProcessList(id string) *RealProcessList {
	return &RealProcessList{Base: *NewBase(id)}
}

func (g *RealProcessList) SetProcesses(processes []ProcessInfo2) {
	g.mu.Lock()
	g.Processes = processes
	g.mu.Unlock()
}

func (g *RealProcessList) AddProcess(p ProcessInfo2) {
	g.mu.Lock()
	g.Processes = append(g.Processes, p)
	g.mu.Unlock()
}

func (g *RealProcessList) RemoveProcess(pid int) {
	g.mu.Lock()
	for i, p := range g.Processes {
		if p.PID == pid {
			g.Processes = append(g.Processes[:i], g.Processes[i+1:]...)
			break
		}
	}
	g.mu.Unlock()
}

func (g *RealProcessList) Sort(column string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.SortBy = column

	switch column {
	case "name":
		for i := 0; i < len(g.Processes); i++ {
			for j := i + 1; j < len(g.Processes); j++ {
				if g.Processes[i].Name > g.Processes[j].Name {
					g.Processes[i], g.Processes[j] = g.Processes[j], g.Processes[i]
				}
			}
		}
	case "cpu":
		for i := 0; i < len(g.Processes); i++ {
			for j := i + 1; j < len(g.Processes); j++ {
				if g.Processes[i].CPU < g.Processes[j].CPU {
					g.Processes[i], g.Processes[j] = g.Processes[j], g.Processes[i]
				}
			}
		}
	case "memory":
		for i := 0; i < len(g.Processes); i++ {
			for j := i + 1; j < len(g.Processes); j++ {
				if g.Processes[i].Memory < g.Processes[j].Memory {
					g.Processes[i], g.Processes[j] = g.Processes[j], g.Processes[i]
				}
			}
		}
	}
}

func (g *RealProcessList) Render(state StateView) []RenderNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var nodes []RenderNode
	header := fmt.Sprintf("%-8s %-20s %-10s %6s %8s", "PID", "NAME", "STATUS", "CPU%", "MEMORY")
	nodes = append(nodes, RenderNode{Type: "text", Content: header, Style: mofu.DefaultStyle().WithAttrs(mofu.AttrBold)})
	nodes = append(nodes, RenderNode{Type: "text", Content: strings.Repeat("─", 55)})

	for i, p := range g.Processes {
		style := mofu.DefaultStyle()
		if i == g.Selected {
			style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))
		}
		text := fmt.Sprintf("%-8d %-20s %-10s %5.1f%% %7.1fM", p.PID, p.Name, p.Status, p.CPU, p.Memory)
		nodes = append(nodes, RenderNode{Type: "text", Content: text, Style: style})
	}
	return nodes
}

func (g *RealProcessList) OnEvent(e Event) {
	if e.Type == "keypress" {
		if ke, ok := e.Payload.(mofu.KeyEvent); ok {
			g.mu.Lock()
			defer g.mu.Unlock()
			switch ke.Key {
			case mofu.KeyDown:
				if g.Selected < len(g.Processes)-1 {
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

// Gadget 18: RealDiskUsage — Disk usage visualization
type RealDiskUsage struct {
	Base
	Partitions []DiskPartition
	mu         sync.RWMutex
}

type DiskPartition struct {
	Name   string
	Mount  string
	Used   int64
	Total  int64
}

func NewRealDiskUsage(id string) *RealDiskUsage {
	return &RealDiskUsage{Base: *NewBase(id)}
}

func (g *RealDiskUsage) SetPartitions(partitions []DiskPartition) {
	g.mu.Lock()
	g.Partitions = partitions
	g.mu.Unlock()
}

func (g *RealDiskUsage) GetUsagePercent(mount string) float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	for _, p := range g.Partitions {
		if p.Mount == mount && p.Total > 0 {
			return float64(p.Used) / float64(p.Total) * 100
		}
	}
	return 0
}

func (g *RealDiskUsage) Render(state StateView) []RenderNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var nodes []RenderNode
	for _, p := range g.Partitions {
		pct := float64(p.Used) / float64(p.Total) * 100
		barWidth := 30
		filled := int(pct / 100 * float64(barWidth))
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

		color := "a6e3a1"
		if pct > 80 {
			color = "f38ba8"
		} else if pct > 60 {
			color = "f9e2af"
		}

		text := fmt.Sprintf("%-10s [%s] %.1f%%", p.Mount, bar, pct)
		nodes = append(nodes, RenderNode{Type: "text", Content: text, Style: mofu.DefaultStyle().Fg(mofu.Hex(color))})
	}
	return nodes
}

// Gadget 19: RealNetworkStats — Network statistics
type RealNetworkStats struct {
	Base
	Interfaces []NetworkInterface
	mu         sync.RWMutex
}

type NetworkInterface struct {
	Name      string
	IP        string
	Status    string
	RxBytes   int64
	TxBytes   int64
	RxPackets int64
	TxPackets int64
	Speed     int64
}

func NewRealNetworkStats(id string) *RealNetworkStats {
	return &RealNetworkStats{Base: *NewBase(id)}
}

func (g *RealNetworkStats) SetInterfaces(interfaces []NetworkInterface) {
	g.mu.Lock()
	g.Interfaces = interfaces
	g.mu.Unlock()
}

func (g *RealNetworkStats) GetInterface(name string) *NetworkInterface {
	g.mu.RLock()
	defer g.mu.RUnlock()
	for i := range g.Interfaces {
		if g.Interfaces[i].Name == name {
			return &g.Interfaces[i]
		}
	}
	return nil
}

func (g *RealNetworkStats) Render(state StateView) []RenderNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var nodes []RenderNode
	header := fmt.Sprintf("%-10s %-15s %-10s %12s %12s", "INTERFACE", "IP", "STATUS", "RX", "TX")
	nodes = append(nodes, RenderNode{Type: "text", Content: header, Style: mofu.DefaultStyle().WithAttrs(mofu.AttrBold)})
	nodes = append(nodes, RenderNode{Type: "text", Content: strings.Repeat("─", 65)})

	for _, iface := range g.Interfaces {
		style := mofu.DefaultStyle()
		if iface.Status == "up" {
			style = mofu.DefaultStyle().Fg(mofu.Hex("a6e3a1"))
		} else {
			style = mofu.DefaultStyle().Fg(mofu.Hex("f38ba8"))
		}
		text := fmt.Sprintf("%-10s %-15s %-10s %10s %10s",
			iface.Name, iface.IP, iface.Status,
			formatBytes(iface.RxBytes), formatBytes(iface.TxBytes))
		nodes = append(nodes, RenderNode{Type: "text", Content: text, Style: style})
	}
	return nodes
}

func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// Gadget 20: RealFileTree — File system tree
type RealFileTree struct {
	Base
	Root     *FileNode
	Selected string
	Expanded map[string]bool
	mu       sync.RWMutex
}

type FileNode struct {
	Name     string
	Path     string
	IsDir    bool
	Size     int64
	ModTime  time.Time
	Children []*FileNode
}

func NewRealFileTree(id string) *RealFileTree {
	return &RealFileTree{Base: *NewBase(id), Expanded: make(map[string]bool)}
}

func (g *RealFileTree) SetRoot(root *FileNode) {
	g.mu.Lock()
	g.Root = root
	g.mu.Unlock()
}

func (g *RealFileTree) Toggle(path string) {
	g.mu.Lock()
	g.Expanded[path] = !g.Expanded[path]
	g.mu.Unlock()
}

func (g *RealFileTree) Expand(path string) {
	g.mu.Lock()
	g.Expanded[path] = true
	g.mu.Unlock()
}

func (g *RealFileTree) Collapse(path string) {
	g.mu.Lock()
	g.Expanded[path] = false
	g.mu.Unlock()
}

func (g *RealFileTree) GetSelected() *FileNode {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.Root == nil {
		return nil
	}
	var find func(node *FileNode) *FileNode
	find = func(node *FileNode) *FileNode {
		if node.Path == g.Selected {
			return node
		}
		for _, child := range node.Children {
			if found := find(child); found != nil {
				return found
			}
		}
		return nil
	}
	return find(g.Root)
}

func (g *RealFileTree) Render(state StateView) []RenderNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.Root == nil {
		return []RenderNode{{Type: "text", Content: "No files"}}
	}

	var nodes []RenderNode
	var renderNode func(node *FileNode, depth int)
	renderNode = func(node *FileNode, depth int) {
		indent := strings.Repeat("  ", depth)
		icon := "📄"
		if node.IsDir {
			if g.Expanded[node.Path] {
				icon = "📂"
			} else {
				icon = "📁"
			}
		}

		text := fmt.Sprintf("%s%s %s", indent, icon, node.Name)
		style := mofu.DefaultStyle()
		if node.Path == g.Selected {
			style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))
		}

		nodes = append(nodes, RenderNode{Type: "text", Content: text, Style: style})

		if node.IsDir && g.Expanded[node.Path] {
			for _, child := range node.Children {
				renderNode(child, depth+1)
			}
		}
	}

	renderNode(g.Root, 0)
	return nodes
}

func (g *RealFileTree) OnEvent(e Event) {
	if e.Type == "keypress" {
		if ke, ok := e.Payload.(mofu.KeyEvent); ok {
			g.mu.Lock()
			defer g.mu.Unlock()
			switch ke.Key {
			case mofu.KeyEnter:
				if g.Selected != "" {
					g.Expanded[g.Selected] = !g.Expanded[g.Selected]
				}
			}
		}
	}
}

// Gadget 21: RealSearchBox — Search input with suggestions
type RealSearchBox struct {
	Base
	Query       string
	Suggestions []string
	Selected    int
	Placeholder string
	OnSearch    func(query string)
	OnSelect    func(suggestion string)
	mu          sync.RWMutex
}

func NewRealSearchBox(id string) *RealSearchBox {
	return &RealSearchBox{Base: *NewBase(id), Placeholder: "Search..."}
}

func (g *RealSearchBox) SetQuery(query string) {
	g.mu.Lock()
	g.Query = query
	g.mu.Unlock()
	if g.OnSearch != nil {
		g.OnSearch(query)
	}
}

func (g *RealSearchBox) SetSuggestions(suggestions []string) {
	g.mu.Lock()
	g.Suggestions = suggestions
	g.Selected = 0
	g.mu.Unlock()
}

func (g *RealSearchBox) GetQuery() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.Query
}

func (g *RealSearchBox) Clear() {
	g.mu.Lock()
	g.Query = ""
	g.Suggestions = nil
	g.mu.Unlock()
}

func (g *RealSearchBox) Render(state StateView) []RenderNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var nodes []RenderNode
	text := g.Query
	if text == "" {
		text = g.Placeholder
	}
	nodes = append(nodes, RenderNode{Type: "text", Content: "🔍 " + text + "_", Style: mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))})

	for i, s := range g.Suggestions {
		style := mofu.DefaultStyle()
		if i == g.Selected {
			style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))
		}
		nodes = append(nodes, RenderNode{Type: "text", Content: "  " + s, Style: style})
	}
	return nodes
}

func (g *RealSearchBox) OnEvent(e Event) {
	if e.Type == "keypress" {
		if ke, ok := e.Payload.(mofu.KeyEvent); ok {
			g.mu.Lock()
			defer g.mu.Unlock()
			switch ke.Key {
			case mofu.KeyDown:
				if g.Selected < len(g.Suggestions)-1 {
					g.Selected++
				}
			case mofu.KeyUp:
				if g.Selected > 0 {
					g.Selected--
				}
			case mofu.KeyEnter:
				if g.OnSelect != nil && g.Selected < len(g.Suggestions) {
					g.OnSelect(g.Suggestions[g.Selected])
				}
			case mofu.KeyBack:
				if len(g.Query) > 0 {
					g.Query = g.Query[:len(g.Query)-1]
				}
			default:
				for _, r := range ke.Runes {
					if r >= 32 && r < 127 {
						g.Query += string(r)
					}
				}
			}
		}
	}
}

// Gadget 22: RealDropDown — Dropdown select
type RealDropDown struct {
	Base
	Options   []string
	Selected  int
	Open      bool
	Placeholder string
	OnChange  func(index int, value string)
	mu        sync.RWMutex
}

func NewRealDropDown(id string, options []string) *RealDropDown {
	return &RealDropDown{Base: *NewBase(id), Options: options, Placeholder: "Select..."}
}

func (g *RealDropDown) SetSelected(index int) {
	g.mu.Lock()
	if index >= 0 && index < len(g.Options) {
		g.Selected = index
	}
	g.mu.Unlock()
}

func (g *RealDropDown) Toggle() {
	g.mu.Lock()
	g.Open = !g.Open
	g.mu.Unlock()
}

func (g *RealDropDown) GetSelected() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.Selected >= 0 && g.Selected < len(g.Options) {
		return g.Options[g.Selected]
	}
	return ""
}

func (g *RealDropDown) Render(state StateView) []RenderNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var nodes []RenderNode
	text := g.GetSelected()
	if text == "" {
		text = g.Placeholder
	}
	arrow := "▾"
	if g.Open {
		arrow = "▴"
	}
	nodes = append(nodes, RenderNode{Type: "text", Content: text + " " + arrow, Style: mofu.DefaultStyle()})

	if g.Open {
		for i, opt := range g.Options {
			style := mofu.DefaultStyle()
			if i == g.Selected {
				style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))
			}
			nodes = append(nodes, RenderNode{Type: "text", Content: "  " + opt, Style: style})
		}
	}
	return nodes
}

func (g *RealDropDown) OnEvent(e Event) {
	if e.Type == "keypress" {
		if ke, ok := e.Payload.(mofu.KeyEvent); ok {
			g.mu.Lock()
			defer g.mu.Unlock()
			if !g.Open {
				if ke.Key == mofu.KeyEnter {
					g.Open = true
				}
				return
			}
			switch ke.Key {
			case mofu.KeyDown:
				if g.Selected < len(g.Options)-1 {
					g.Selected++
				}
			case mofu.KeyUp:
				if g.Selected > 0 {
					g.Selected--
				}
			case mofu.KeyEnter:
				g.Open = false
				if g.OnChange != nil {
					g.OnChange(g.Selected, g.Options[g.Selected])
				}
			case mofu.KeyEsc:
				g.Open = false
			}
		}
	}
}

// Gadget 23: RealCalendar — Calendar display
type RealCalendar struct {
	Base
	Year    int
	Month   time.Month
	Selected time.Time
	mu      sync.RWMutex
}

func NewRealCalendar(id string) *RealCalendar {
	now := time.Now()
	return &RealCalendar{Base: *NewBase(id), Year: now.Year(), Month: now.Month(), Selected: now}
}

func (g *RealCalendar) SetDate(year int, month time.Month) {
	g.mu.Lock()
	g.Year = year
	g.Month = month
	g.mu.Unlock()
}

func (g *RealCalendar) NextMonth() {
	g.mu.Lock()
	if g.Month == time.December {
		g.Month = time.January
		g.Year++
	} else {
		g.Month++
	}
	g.mu.Unlock()
}

func (g *RealCalendar) PrevMonth() {
	g.mu.Lock()
	if g.Month == time.January {
		g.Month = time.December
		g.Year--
	} else {
		g.Month--
	}
	g.mu.Unlock()
}

func (g *RealCalendar) Render(state StateView) []RenderNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var nodes []RenderNode
	// Header
	header := fmt.Sprintf("     %s %d", g.Month.String(), g.Year)
	nodes = append(nodes, RenderNode{Type: "text", Content: header, Style: mofu.DefaultStyle().WithAttrs(mofu.AttrBold)})

	// Day names
	nodes = append(nodes, RenderNode{Type: "text", Content: "Su Mo Tu We Th Fr Sa"})

	// Calendar grid
	firstDay := time.Date(g.Year, g.Month, 1, 0, 0, 0, 0, time.Local)
	weekday := firstDay.Weekday()
	daysInMonth := time.Date(g.Year, g.Month+1, 0, 0, 0, 0, 0, time.Local).Day()

	line := strings.Repeat("   ", int(weekday))
	for day := 1; day <= daysInMonth; day++ {
		date := time.Date(g.Year, g.Month, day, 0, 0, 0, 0, time.Local)
		text := fmt.Sprintf("%2d", day)
		if date.Equal(g.Selected) {
			text = fmt.Sprintf("[%2d]", day)
		}
		line += text + " "
		if (weekday+time.Weekday(day))%7 == 0 {
			nodes = append(nodes, RenderNode{Type: "text", Content: line})
			line = ""
		}
		weekday++
	}
	if line != "" {
		nodes = append(nodes, RenderNode{Type: "text", Content: line})
	}
	return nodes
}

// Gadget 24: RealCodeBlock — Code display with syntax highlighting
type RealCodeBlock struct {
	Base
	Code      string
	Language  string
	ShowLineNo bool
	Highlight []int // lines to highlight
	mu        sync.RWMutex
}

func NewRealCodeBlock(id string) *RealCodeBlock {
	return &RealCodeBlock{Base: *NewBase(id), ShowLineNo: true}
}

func (g *RealCodeBlock) SetCode(code, language string) {
	g.mu.Lock()
	g.Code = code
	g.Language = language
	g.mu.Unlock()
}

func (g *RealCodeBlock) HighlightLines(lines []int) {
	g.mu.Lock()
	g.Highlight = lines
	g.mu.Unlock()
}

func (g *RealCodeBlock) Render(state StateView) []RenderNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var nodes []RenderNode
	lines := strings.Split(g.Code, "\n")
	for i, line := range lines {
		style := mofu.DefaultStyle()
		// Simple syntax highlighting
		if strings.HasPrefix(strings.TrimSpace(line), "//") || strings.HasPrefix(strings.TrimSpace(line), "#") {
			style = mofu.DefaultStyle().Fg(mofu.Hex("6c7086"))
		} else if strings.Contains(line, "func ") || strings.Contains(line, "func(") {
			style = mofu.DefaultStyle().Fg(mofu.Hex("89b4fa"))
		} else if strings.Contains(line, "return ") {
			style = mofu.DefaultStyle().Fg(mofu.Hex("f38ba8"))
		}

		// Highlight specific lines
		for _, hl := range g.Highlight {
			if i == hl {
				style = mofu.DefaultStyle().Bg(mofu.Hex("313244"))
			}
		}

		prefix := ""
		if g.ShowLineNo {
			prefix = fmt.Sprintf("%3d │ ", i+1)
		}
		nodes = append(nodes, RenderNode{Type: "text", Content: prefix + line, Style: style})
	}
	return nodes
}

// Gadget 25: RealDiffViewer — Side-by-side diff
type RealDiffViewer struct {
	Base
	OldLines  []string
	NewLines  []string
	Title     string
	mu        sync.RWMutex
}

func NewRealDiffViewer(id string) *RealDiffViewer {
	return &RealDiffViewer{Base: *NewBase(id)}
}

func (g *RealDiffViewer) SetDiff(old, new []string) {
	g.mu.Lock()
	g.OldLines = old
	g.NewLines = new
	g.mu.Unlock()
}

func (g *RealDiffViewer) Clear() {
	g.mu.Lock()
	g.OldLines = nil
	g.NewLines = nil
	g.mu.Unlock()
}

func (g *RealDiffViewer) Render(state StateView) []RenderNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var nodes []RenderNode
	if g.Title != "" {
		nodes = append(nodes, RenderNode{Type: "text", Content: g.Title, Style: mofu.DefaultStyle().WithAttrs(mofu.AttrBold)})
	}

	maxLines := len(g.OldLines)
	if len(g.NewLines) > maxLines {
		maxLines = len(g.NewLines)
	}

	for i := 0; i < maxLines; i++ {
		var oldLine, newLine string
		if i < len(g.OldLines) {
			oldLine = g.OldLines[i]
		}
		if i < len(g.NewLines) {
			newLine = g.NewLines[i]
		}

		if oldLine == newLine {
			nodes = append(nodes, RenderNode{Type: "text", Content: "  " + oldLine})
		} else {
			if oldLine != "" {
				nodes = append(nodes, RenderNode{
					Type:    "text",
					Content: "- " + oldLine,
					Style:   mofu.DefaultStyle().Fg(mofu.Hex("f38ba8")),
				})
			}
			if newLine != "" {
				nodes = append(nodes, RenderNode{
					Type:    "text",
					Content: "+ " + newLine,
					Style:   mofu.DefaultStyle().Fg(mofu.Hex("a6e3a1")),
				})
			}
		}
	}
	return nodes
}

// Gadget 26: RealHexViewer — Hex dump display
type RealHexViewer struct {
	Base
	Data        []byte
	Offset      int
	BytesPerLine int
	mu          sync.RWMutex
}

func NewRealHexViewer(id string) *RealHexViewer {
	return &RealHexViewer{Base: *NewBase(id), BytesPerLine: 16}
}

func (g *RealHexViewer) SetData(data []byte) {
	g.mu.Lock()
	g.Data = data
	g.mu.Unlock()
}

func (g *RealHexViewer) SetOffset(offset int) {
	g.mu.Lock()
	g.Offset = offset
	g.mu.Unlock()
}

func (g *RealHexViewer) Render(state StateView) []RenderNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var nodes []RenderNode
	for i := g.Offset; i < len(g.Data); i += g.BytesPerLine {
		end := i + g.BytesPerLine
		if end > len(g.Data) {
			end = len(g.Data)
		}
		chunk := g.Data[i:end]

		offset := fmt.Sprintf("%08x", i)
		hex := ""
		for _, b := range chunk {
			hex += fmt.Sprintf("%02x ", b)
		}
		for len(hex) < g.BytesPerLine*3 {
			hex += " "
		}

		ascii := ""
		for _, b := range chunk {
			if b >= 32 && b <= 126 {
				ascii += string(b)
			} else {
				ascii += "."
			}
		}

		line := fmt.Sprintf("%s  %s  %s", offset, hex, ascii)
		nodes = append(nodes, RenderNode{Type: "text", Content: line})
	}
	return nodes
}

// Gadget 27: RealProgressBarGroup — Multiple progress bars
type RealProgressBarGroup struct {
	Base
	Bars   []ProgressBarItem
	mu     sync.RWMutex
}

type ProgressBarItem struct {
	Label string
	Value float64
	Max   float64
	Color string
}

func NewRealProgressBarGroup(id string) *RealProgressBarGroup {
	return &RealProgressBarGroup{Base: *NewBase(id)}
}

func (g *RealProgressBarGroup) AddBar(label string, value, max float64) {
	g.mu.Lock()
	g.Bars = append(g.Bars, ProgressBarItem{Label: label, Value: value, Max: max, Color: "a6e3a1"})
	g.mu.Unlock()
}

func (g *RealProgressBarGroup) UpdateBar(label string, value float64) {
	g.mu.Lock()
	for i := range g.Bars {
		if g.Bars[i].Label == label {
			g.Bars[i].Value = value
			break
		}
	}
	g.mu.Unlock()
}

func (g *RealProgressBarGroup) Render(state StateView) []RenderNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var nodes []RenderNode
	for _, bar := range g.Bars {
		pct := bar.Value / bar.Max * 100
		barWidth := 20
		filled := int(pct / 100 * float64(barWidth))
		barStr := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

		color := bar.Color
		if pct > 80 {
			color = "f38ba8"
		} else if pct > 60 {
			color = "f9e2af"
		}

		text := fmt.Sprintf("%-15s [%s] %.1f%%", bar.Label, barStr, pct)
		nodes = append(nodes, RenderNode{Type: "text", Content: text, Style: mofu.DefaultStyle().Fg(mofu.Hex(color))})
	}
	return nodes
}

// Gadget 28: RealStatCard — Single stat display
type RealStatCard struct {
	Base
	Label  string
	Value  string
	Change float64
	Unit   string
	mu     sync.RWMutex
}

func NewRealStatCard(id, label string) *RealStatCard {
	return &RealStatCard{Base: *NewBase(id), Label: label}
}

func (g *RealStatCard) SetValue(value string) {
	g.mu.Lock()
	g.Value = value
	g.mu.Unlock()
}

func (g *RealStatCard) SetChange(change float64) {
	g.mu.Lock()
	g.Change = change
	g.mu.Unlock()
}

func (g *RealStatCard) Render(state StateView) []RenderNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var nodes []RenderNode
	nodes = append(nodes, RenderNode{Type: "text", Content: g.Label, Style: mofu.DefaultStyle().Fg(mofu.Hex("666666"))})
	nodes = append(nodes, RenderNode{Type: "text", Content: g.Value + g.Unit, Style: mofu.DefaultStyle().WithAttrs(mofu.AttrBold)})

	if g.Change != 0 {
		changeStr := fmt.Sprintf("%+.1f%%", g.Change)
		style := mofu.DefaultStyle().Fg(mofu.Hex("a6e3a1"))
		if g.Change < 0 {
			style = mofu.DefaultStyle().Fg(mofu.Hex("f38ba8"))
		}
		nodes = append(nodes, RenderNode{Type: "text", Content: changeStr, Style: style})
	}
	return nodes
}

// Gadget 29: RealAlertBanner — Alert banner
type RealAlertBanner struct {
	Base
	Message string
	Level   string // "info", "success", "warning", "error"
	Visible bool
	mu      sync.RWMutex
}

func NewRealAlertBanner(id string) *RealAlertBanner {
	return &RealAlertBanner{Base: *NewBase(id)}
}

func (g *RealAlertBanner) Show(message, level string) {
	g.mu.Lock()
	g.Message = message
	g.Level = level
	g.Visible = true
	g.mu.Unlock()
}

func (g *RealAlertBanner) Hide() {
	g.mu.Lock()
	g.Visible = false
	g.mu.Unlock()
}

func (g *RealAlertBanner) Render(state StateView) []RenderNode {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if !g.Visible {
		return nil
	}

	style := mofu.DefaultStyle()
	icon := "ℹ"
	switch g.Level {
	case "success":
		style = mofu.DefaultStyle().Fg(mofu.Hex("a6e3a1"))
		icon = "✓"
	case "warning":
		style = mofu.DefaultStyle().Fg(mofu.Hex("f9e2af"))
		icon = "⚠"
	case "error":
		style = mofu.DefaultStyle().Fg(mofu.Hex("f38ba8"))
		icon = "✗"
	}

	text := fmt.Sprintf("%s %s", icon, g.Message)
	return []RenderNode{{Type: "text", Content: text, Style: style}}
}

// Gadget 30: RealKeyValueList — Key-value list display
type RealKeyValueList struct {
	Base
	Items  []KeyValueItem
	mu     sync.RWMutex
}

type KeyValueItem struct {
	Key   string
	Value string
	Icon  string
}

func NewRealKeyValueList(id string) *RealKeyValueList {
	return &RealKeyValueList{Base: *NewBase(id)}
}

func (g *RealKeyValueList) SetItem(key, value string) {
	g.mu.Lock()
	for i, item := range g.Items {
		if item.Key == key {
			g.Items[i].Value = value
			g.mu.Unlock()
			return
		}
	}
	g.Items = append(g.Items, KeyValueItem{Key: key, Value: value})
	g.mu.Unlock()
}

func (g *RealKeyValueList) RemoveItem(key string) {
	g.mu.Lock()
	for i, item := range g.Items {
		if item.Key == key {
			g.Items = append(g.Items[:i], g.Items[i+1:]...)
			break
		}
	}
	g.mu.Unlock()
}

func (g *RealKeyValueList) Clear() {
	g.mu.Lock()
	g.Items = nil
	g.mu.Unlock()
}

func (g *RealKeyValueList) Render(state StateView) []RenderNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var nodes []RenderNode
	for _, item := range g.Items {
		icon := item.Icon
		if icon == "" {
			icon = "•"
		}
		text := fmt.Sprintf("%s %-20s %s", icon, item.Key+":", item.Value)
		nodes = append(nodes, RenderNode{Type: "text", Content: text})
	}
	return nodes
}
