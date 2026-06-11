package gadgets

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/xanstomper/mofu"
)

// =========================================================================
// BATCH 10: AI, Data & Interactive Gadgets (10 gadgets)
// =========================================================================

type RealDiffViewerPro struct {
	Base
	Lines    []DiffLine
	Selected int
	mu       sync.RWMutex
}

type DiffLine struct {
	Type  string
	Old   string
	New   string
	Num   int
}

func NewRealDiffViewerPro(id string) *RealDiffViewerPro {
	return &RealDiffViewerPro{Base: *NewBase(id)}
}

func (g *RealDiffViewerPro) SetDiff(lines []DiffLine) {
	g.mu.Lock()
	g.Lines = lines
	g.mu.Unlock()
}

func (g *RealDiffViewerPro) ComputeDiff(old, new []string) {
	g.mu.Lock()
	g.Lines = nil
	lineNum := 0

	maxLen := len(old)
	if len(new) > maxLen {
		maxLen = len(new)
	}

	for i := 0; i < maxLen; i++ {
		oldLine := ""
		newLine := ""
		if i < len(old) {
			oldLine = old[i]
		}
		if i < len(new) {
			newLine = new[i]
		}

		lineNum++
		if oldLine == newLine {
			g.Lines = append(g.Lines, DiffLine{Type: "context", Old: oldLine, New: newLine, Num: lineNum})
		} else {
			if oldLine != "" {
				g.Lines = append(g.Lines, DiffLine{Type: "removed", Old: oldLine, Num: lineNum})
			}
			if newLine != "" {
				g.Lines = append(g.Lines, DiffLine{Type: "added", New: newLine, Num: lineNum})
			}
		}
	}
	g.mu.Unlock()
}

func (g *RealDiffViewerPro) Stats() (added, removed, context int) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	for _, l := range g.Lines {
		switch l.Type {
		case "added":
			added++
		case "removed":
			removed++
		default:
			context++
		}
	}
	return
}

func (g *RealDiffViewerPro) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	added, removed, _ := g.Stats()
	ctx.Renderer.WriteString(fmt.Sprintf(" Diff (+%d -%d)", added, removed), r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	for i, line := range g.Lines {
		if y >= r.Y+r.Height-1 {
			break
		}

		var prefix string
		var color mofu.Color

		switch line.Type {
		case "added":
			prefix = "+ "
			color = mofu.Hex("a6e3a1")
		case "removed":
			prefix = "- "
			color = mofu.Hex("f38ba8")
		default:
			prefix = "  "
			color = mofu.Hex("cdd6f4")
		}

		text := line.Old
		if text == "" {
			text = line.New
		}
		if len(text) > r.Width-6 {
			text = text[:r.Width-9] + "..."
		}

		displayLine := fmt.Sprintf("%s%s", prefix, text)

		if i == g.Selected {
			ctx.Renderer.WriteString(strings.Repeat(" ", r.Width-2), r.X+1, y, mofu.ColorWhite, mofu.Hex("2a2a3e"), 0)
		}

		ctx.Renderer.WriteString(displayLine, r.X, y, color, mofu.ColorBlack, 0)
		y++
	}
}

func (g *RealDiffViewerPro) HandleEvent(e mofu.Event) mofu.Cmd {
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
		if g.Selected < len(g.Lines)-1 {
			g.Selected++
		}
	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if g.Selected > 0 {
			g.Selected--
		}
	}
	return nil
}

type RealJSONViewer struct {
	Base
	Raw       string
	Expanded  map[int]bool
	Selected  int
	mu        sync.RWMutex
}

func NewRealJSONViewer(id string) *RealJSONViewer {
	return &RealJSONViewer{Base: *NewBase(id), Expanded: map[int]bool{0: true}}
}

func (g *RealJSONViewer) SetJSON(raw string) {
	g.mu.Lock()
	g.Raw = raw
	g.mu.Unlock()
}

type jsonLine struct {
	indent int
	key    string
	value  string
}

func (g *RealJSONViewer) parseLines() []jsonLine {
	lines := strings.Split(g.Raw, "\n")
	var result []jsonLine
	indent := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		for _, ch := range trimmed {
			if ch == '}' || ch == ']' {
				indent--
			}
		}

		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) == 2 {
			result = append(result, jsonLine{indent: indent, key: strings.Trim(parts[0], "\" "), value: strings.TrimSpace(parts[1])})
		} else {
			result = append(result, jsonLine{indent: indent, key: "", value: trimmed})
		}

		for _, ch := range trimmed {
			if ch == '{' || ch == '[' {
				indent++
			}
		}
	}
	return result
}

func (g *RealJSONViewer) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(" JSON Viewer", r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	lines := g.parseLines()
	for _, line := range lines {
		if y >= r.Y+r.Height-1 {
			break
		}

		indent := strings.Repeat("  ", line.indent)
		text := ""
		if line.key != "" {
			text = fmt.Sprintf("%s\"%s\": %s", indent, line.key, line.value)
		} else {
			text = fmt.Sprintf("%s%s", indent, line.value)
		}

		if len(text) > r.Width-2 {
			text = text[:r.Width-5] + "..."
		}

		color := mofu.Hex("cdd6f4")
		if strings.Contains(line.value, "\"") {
			color = mofu.Hex("a6e3a1")
		} else if strings.Contains(line.value, "true") || strings.Contains(line.value, "false") {
			color = mofu.Hex("fab387")
		} else if strings.Contains(line.value, "null") {
			color = mofu.Hex("585b70")
		}

		if line.key != "" {
			ctx.Renderer.WriteString(fmt.Sprintf(" %s", line.key), r.X, y, mofu.Hex("89b4fa"), mofu.ColorBlack, mofu.AttrBold)
			keyW := len(line.key) + 3
			if keyW > r.Width-10 {
				keyW = r.Width - 10
			}
			ctx.Renderer.WriteString(fmt.Sprintf(": %s", line.value), r.X+keyW, y, color, mofu.ColorBlack, 0)
		} else {
			ctx.Renderer.WriteString(text, r.X, y, color, mofu.ColorBlack, 0)
		}
		y++
	}
}

func (g *RealJSONViewer) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)
	if ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q') {
		return mofu.QuitCmd()
	}
	return nil
}

type RealStatusPage struct {
	Base
	Title    string
	Sections []StatusSection
	mu       sync.RWMutex
}

type StatusSection struct {
	Title string
	Items []StatusItem
}

type StatusItem struct {
	Label  string
	Value  string
	Status string
	Color  mofu.Color
}

func NewRealStatusPage(id, title string) *RealStatusPage {
	return &RealStatusPage{Base: *NewBase(id), Title: title}
}

func (g *RealStatusPage) AddSection(title string, items []StatusItem) {
	g.mu.Lock()
	g.Sections = append(g.Sections, StatusSection{Title: title, Items: items})
	g.mu.Unlock()
}

func (g *RealStatusPage) UpdateItem(section, label, value, status string) {
	g.mu.Lock()
	for i, s := range g.Sections {
		if s.Title == section {
			for j, item := range s.Items {
				if item.Label == label {
					g.Sections[i].Items[j].Value = value
					g.Sections[i].Items[j].Status = status
					switch status {
					case "ok", "healthy":
						g.Sections[i].Items[j].Color = mofu.Hex("a6e3a1")
					case "warning":
						g.Sections[i].Items[j].Color = mofu.Hex("fab387")
					case "error", "critical":
						g.Sections[i].Items[j].Color = mofu.Hex("f38ba8")
					default:
						g.Sections[i].Items[j].Color = mofu.Hex("cdd6f4")
					}
				}
			}
		}
	}
	g.mu.Unlock()
}

func (g *RealStatusPage) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(fmt.Sprintf(" %s", g.Title), r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	for _, section := range g.Sections {
		if y >= r.Y+r.Height-2 {
			break
		}
		ctx.Renderer.WriteString(fmt.Sprintf(" %s", section.Title), r.X, y, mofu.Hex("89b4fa"), mofu.ColorBlack, mofu.AttrBold)
		y++

		for _, item := range section.Items {
			if y >= r.Y+r.Height-1 {
				break
			}
			statusIcon := "●"
			switch item.Status {
			case "ok", "healthy":
				statusIcon = "✓"
			case "warning":
				statusIcon = "⚠"
			case "error", "critical":
				statusIcon = "✗"
			default:
				statusIcon = "○"
			}
			ctx.Renderer.WriteString(fmt.Sprintf("   %s %-20s %s", statusIcon, item.Label, item.Value), r.X, y, item.Color, mofu.ColorBlack, 0)
			y++
		}
	}
}

func (g *RealStatusPage) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)
	if ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q') {
		return mofu.QuitCmd()
	}
	return nil
}

type RealMarkdownPreview struct {
	Base
	Content string
	ScrollY int
	mu      sync.RWMutex
}

func NewRealMarkdownPreview(id string) *RealMarkdownPreview {
	return &RealMarkdownPreview{Base: *NewBase(id)}
}

func (g *RealMarkdownPreview) SetContent(content string) {
	g.mu.Lock()
	g.Content = content
	g.mu.Unlock()
}

func (g *RealMarkdownPreview) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(" Markdown Preview", r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	lines := strings.Split(g.Content, "\n")
	for i := g.ScrollY; i < len(lines) && y < r.Y+r.Height-2; i++ {
		line := lines[i]
		style := mofu.DefaultStyle()

		if strings.HasPrefix(line, "# ") {
			style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
			line = line[2:]
		} else if strings.HasPrefix(line, "## ") {
			style = mofu.DefaultStyle().Fg(mofu.Hex("89b4fa")).WithAttrs(mofu.AttrBold)
			line = line[3:]
		} else if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			line = "  • " + line[2:]
		} else if strings.HasPrefix(line, "> ") {
			style = mofu.DefaultStyle().Fg(mofu.Hex("6c7086"))
			line = "  │ " + line[2:]
		} else if strings.HasPrefix(line, "```") {
			style = mofu.DefaultStyle().Fg(mofu.Hex("a6e3a1"))
		}

		if len(line) > r.Width-2 {
			line = line[:r.Width-5] + "..."
		}
		ctx.Renderer.WriteString(line, r.X+1, y, style.Foreground, style.Background, style.Attrs)
		y++
	}
}

func (g *RealMarkdownPreview) HandleEvent(e mofu.Event) mofu.Cmd {
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
		g.ScrollY++
	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if g.ScrollY > 0 {
			g.ScrollY--
		}
	}
	return nil
}

type RealAICodeReview struct {
	Base
	File      string
	Issues    []CodeIssue
	Selected  int
	mu        sync.RWMutex
	OnResolve func(idx int)
}

type CodeIssue struct {
	Line     int
	Severity string
	Message  string
	Rule     string
}

func NewRealAICodeReview(id string) *RealAICodeReview {
	return &RealAICodeReview{
		Base: *NewBase(id),
		Issues: []CodeIssue{
			{Line: 42, Severity: "error", Message: "Potential nil dereference", Rule: "nil-check"},
			{Line: 78, Severity: "warning", Message: "Unused variable 'result'", Rule: "unused-var"},
			{Line: 103, Severity: "info", Message: "Consider using context.WithTimeout", Rule: "best-practice"},
			{Line: 156, Severity: "warning", Message: "Missing error check", Rule: "error-handling"},
			{Line: 201, Severity: "error", Message: "Race condition on shared map", Rule: "concurrency"},
		},
	}
}

func (g *RealAICodeReview) AddIssue(line int, severity, message, rule string) {
	g.mu.Lock()
	g.Issues = append(g.Issues, CodeIssue{Line: line, Severity: severity, Message: message, Rule: rule})
	g.mu.Unlock()
}

func (g *RealAICodeReview) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	errors := 0
	warnings := 0
	for _, issue := range g.Issues {
		if issue.Severity == "error" {
			errors++
		} else if issue.Severity == "warning" {
			warnings++
		}
	}

	titleColor := mofu.Hex("a6e3a1")
	if errors > 0 {
		titleColor = mofu.Hex("f38ba8")
	}
	ctx.Renderer.WriteString(fmt.Sprintf(" Code Review — %d errors, %d warnings", errors, warnings), r.X, y, titleColor, mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	for i, issue := range g.Issues {
		if y >= r.Y+r.Height-1 {
			break
		}

		severityIcon := "●"
		color := mofu.Hex("cdd6f4")
		switch issue.Severity {
		case "error":
			severityIcon = "✗"
			color = mofu.Hex("f38ba8")
		case "warning":
			severityIcon = "⚠"
			color = mofu.Hex("fab387")
		case "info":
			severityIcon = "ℹ"
			color = mofu.Hex("89b4fa")
		}

		if i == g.Selected {
			ctx.Renderer.WriteString(strings.Repeat(" ", r.Width-2), r.X+1, y, mofu.ColorWhite, mofu.Hex("2a2a3e"), 0)
		}

		line := fmt.Sprintf(" %s L%d: %s [%s]", severityIcon, issue.Line, issue.Message, issue.Rule)
		if len(line) > r.Width-2 {
			line = line[:r.Width-5] + "..."
		}
		ctx.Renderer.WriteString(line, r.X, y, color, mofu.ColorBlack, 0)
		y++
	}
}

func (g *RealAICodeReview) HandleEvent(e mofu.Event) mofu.Cmd {
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
		if g.Selected < len(g.Issues)-1 {
			g.Selected++
		}
	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if g.Selected > 0 {
			g.Selected--
		}
	case (len(ke.Runes) > 0 && ke.Runes[0] == 'd') && g.OnResolve != nil:
		g.OnResolve(g.Selected)
	}
	return nil
}

type RealDependencyGraph struct {
	Base
	Title  string
	Nodes  []string
	Edges  [][2]int
	Layout string
	mu     sync.RWMutex
}

func NewRealDependencyGraph(id, title string) *RealDependencyGraph {
	return &RealDependencyGraph{
		Base:  *NewBase(id),
		Title: title,
		Nodes: []string{"core", "state", "render", "kernel", "input", "event", "layout", "diff"},
		Edges: [][2]int{
			{0, 1}, {0, 2}, {0, 3}, {3, 4}, {3, 5},
			{1, 6}, {2, 7}, {6, 7}, {4, 5},
		},
	}
}

func (g *RealDependencyGraph) AddNode(name string) {
	g.mu.Lock()
	g.Nodes = append(g.Nodes, name)
	g.mu.Unlock()
}

func (g *RealDependencyGraph) AddEdge(from, to int) {
	g.mu.Lock()
	g.Edges = append(g.Edges, [2]int{from, to})
	g.mu.Unlock()
}

func (g *RealDependencyGraph) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(fmt.Sprintf(" %s", g.Title), r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	nodeW := 12
	cols := r.Width / nodeW
	if cols < 1 {
		cols = 1
	}

	for i, node := range g.Nodes {
		if y >= r.Y+r.Height-2 {
			break
		}

		col := i % cols
		row := i / cols
		nx := r.X + col*nodeW
		ny := y + row*2

		if ny >= r.Y+r.Height-1 {
			break
		}

		color := mofu.Hex("89b4fa")
		ctx.Renderer.WriteString(fmt.Sprintf("[%-10s]", node), nx, ny, color, mofu.ColorBlack, 0)

		for _, edge := range g.Edges {
			if edge[0] == i && edge[1] < len(g.Nodes) {
				toCol := edge[1] % cols
				toRow := edge[1] / cols
				if toRow == row && toCol > col {
					arrowX := nx + nodeW - 1
					if arrowX < r.X+r.Width {
						ctx.Renderer.WriteString("→", arrowX, ny, mofu.Hex("585b70"), mofu.ColorBlack, 0)
					}
				}
			}
		}
	}
}

func (g *RealDependencyGraph) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)
	if ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q') {
		return mofu.QuitCmd()
	}
	return nil
}

type RealMetricGauge struct {
	Base
	Label   string
	Value   float64
	Max     float64
	Unit    string
	Warning float64
	Critical float64
	mu      sync.RWMutex
}

func NewRealMetricGauge(id, label, unit string, max float64) *RealMetricGauge {
	return &RealMetricGauge{Base: *NewBase(id), Label: label, Max: max, Unit: unit, Warning: 70, Critical: 90}
}

func (g *RealMetricGauge) SetValue(v float64) {
	g.mu.Lock()
	g.Value = v
	g.mu.Unlock()
}

func (g *RealMetricGauge) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	pct := 0.0
	if g.Max > 0 {
		pct = g.Value / g.Max * 100
	}

	color := mofu.Hex("a6e3a1")
	if pct > g.Critical {
		color = mofu.Hex("f38ba8")
	} else if pct > g.Warning {
		color = mofu.Hex("fab387")
	}

	ctx.Renderer.WriteString(fmt.Sprintf(" %s", g.Label), r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++

	bigBar := r.Width - 4

	bar := ""
	for i := 0; i < bigBar; i++ {
		frac := (float64(i) / float64(bigBar)) * 100
		if frac < pct {
			bar += "█"
		} else {
			bar += " "
		}
	}
	ctx.Renderer.WriteString(" "+bar, r.X, y, color, mofu.ColorBlack, 0)
	y++

	ctx.Renderer.WriteString(fmt.Sprintf(" %.1f %s (%.1f%%)", g.Value, g.Unit, pct), r.X, y, color, mofu.ColorBlack, 0)
}

func (g *RealMetricGauge) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)
	if ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q') {
		return mofu.QuitCmd()
	}
	return nil
}

type RealFileWatcher struct {
	Base
	Files    []WatchedFile
	mu       sync.RWMutex
	OnChange func(path string)
}

type WatchedFile struct {
	Path     string
	Size     int64
	Modified time.Time
	Changed  bool
}

func NewRealFileWatcher(id string) *RealFileWatcher {
	return &RealFileWatcher{Base: *NewBase(id)}
}

func (g *RealFileWatcher) Add(path string, size int64) {
	g.mu.Lock()
	g.Files = append(g.Files, WatchedFile{Path: path, Size: size, Modified: time.Now()})
	g.mu.Unlock()
}

func (g *RealFileWatcher) SimulateChange() {
	g.mu.Lock()
	if len(g.Files) > 0 {
		idx := rand.Intn(len(g.Files))
		g.Files[idx].Modified = time.Now()
		g.Files[idx].Changed = true
		g.Files[idx].Size += int64(rand.Intn(1000))
	}
	g.mu.Unlock()
}

func (g *RealFileWatcher) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(fmt.Sprintf(" File Watcher (%d files)", len(g.Files)), r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	for _, f := range g.Files {
		if y >= r.Y+r.Height-2 {
			break
		}
		icon := "○"
		color := mofu.Hex("cdd6f4")
		if f.Changed {
			icon = "●"
			color = mofu.Hex("fab387")
		}
		ctx.Renderer.WriteString(fmt.Sprintf(" %s %-30s %s", icon, f.Path, f.Modified.Format("15:04:05")), r.X, y, color, mofu.ColorBlack, 0)
		y++
	}
}

func (g *RealFileWatcher) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()
	case len(ke.Runes) > 0 && ke.Runes[0] == 's':
		g.SimulateChange()
	}
	return nil
}
