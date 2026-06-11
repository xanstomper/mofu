package gadgets

import (
	"fmt"
	"strings"
	"time"

	"github.com/xanstomper/mofu"
)

// ---------------------------------------------------------------------------
// Additional Gadgets for Production Use
// ---------------------------------------------------------------------------

// MarkdownViewer renders markdown text.
type MarkdownViewer struct {
	Base
	Content string
	Width   int
}

func NewMarkdownViewer(id string) *MarkdownViewer {
	return &MarkdownViewer{Base: *NewBase(id)}
}

func (g *MarkdownViewer) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	lines := strings.Split(g.Content, "\n")
	for _, line := range lines {
		style := mofu.DefaultStyle()
		if strings.HasPrefix(line, "# ") {
			style = mofu.DefaultStyle().WithAttrs(mofu.AttrBold)
			line = line[2:]
		} else if strings.HasPrefix(line, "## ") {
			style = mofu.DefaultStyle().WithAttrs(mofu.AttrBold).Fg(mofu.Hex("89b4fa"))
			line = line[3:]
		} else if strings.HasPrefix(line, "- ") {
			line = "• " + line[2:]
		}
		nodes = append(nodes, RenderNode{Type: "text", Content: line, Style: style})
	}
	return nodes
}

// DiffViewer shows differences between two texts.
type DiffViewer struct {
	Base
	Old     string
	New     string
	Width   int
}

func NewDiffViewer(id string) *DiffViewer {
	return &DiffViewer{Base: *NewBase(id)}
}

func (g *DiffViewer) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	oldLines := strings.Split(g.Old, "\n")
	newLines := strings.Split(g.New, "\n")

	maxLines := len(oldLines)
	if len(newLines) > maxLines {
		maxLines = len(newLines)
	}

	for i := 0; i < maxLines; i++ {
		var oldLine, newLine string
		if i < len(oldLines) {
			oldLine = oldLines[i]
		}
		if i < len(newLines) {
			newLine = newLines[i]
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

// HexViewer displays binary data in hex format.
type HexViewer struct {
	Base
	Data     []byte
	Offset   int
	BytesPerLine int
}

func NewHexViewer(id string) *HexViewer {
	return &HexViewer{Base: *NewBase(id), BytesPerLine: 16}
}

func (g *HexViewer) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	for i := 0; i < len(g.Data); i += g.BytesPerLine {
		end := i + g.BytesPerLine
		if end > len(g.Data) {
			end = len(g.Data)
		}
		chunk := g.Data[i:end]

		// Offset
		offset := fmt.Sprintf("%08x", i)

		// Hex bytes
		hex := ""
		for _, b := range chunk {
			hex += fmt.Sprintf("%02x ", b)
		}
		for len(hex) < g.BytesPerLine*3 {
			hex += " "
		}

		// ASCII representation
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

// JSONExplorer displays formatted JSON.
type JSONExplorer struct {
	Base
	Data     string
	Expanded map[string]bool
	Width    int
}

func NewJSONExplorer(id string) *JSONExplorer {
	return &JSONExplorer{Base: *NewBase(id), Expanded: make(map[string]bool)}
}

func (g *JSONExplorer) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	lines := strings.Split(g.Data, "\n")
	indent := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "}") || strings.HasPrefix(trimmed, "]") {
			indent--
		}
		if indent < 0 {
			indent = 0
		}
		prefix := strings.Repeat("  ", indent)
		style := mofu.DefaultStyle()
		if strings.Contains(trimmed, ":") {
			style = mofu.DefaultStyle().Fg(mofu.Hex("89b4fa"))
		}
		nodes = append(nodes, RenderNode{Type: "text", Content: prefix + trimmed, Style: style})
		if strings.HasSuffix(trimmed, "{") || strings.HasSuffix(trimmed, "[") {
			indent++
		}
	}
	return nodes
}

// InspectorPanel shows key-value pairs.
type InspectorPanel struct {
	Base
	Title  string
	Items  map[string]any
	Width  int
}

func NewInspectorPanel(id, title string) *InspectorPanel {
	return &InspectorPanel{Base: *NewBase(id), Title: title, Items: make(map[string]any)}
}

func (g *InspectorPanel) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	nodes = append(nodes, RenderNode{Type: "text", Content: g.Title, Style: mofu.DefaultStyle().WithAttrs(mofu.AttrBold)})
	nodes = append(nodes, RenderNode{Type: "text", Content: strings.Repeat("─", g.Width-2)})

	for k, v := range g.Items {
		text := fmt.Sprintf("%-20s %v", k, v)
		nodes = append(nodes, RenderNode{Type: "text", Content: text})
	}
	return nodes
}

func (g *InspectorPanel) Set(key string, value any) { g.Items[key] = value }

// GraphVisualizer displays a simple ASCII graph.
type GraphVisualizer struct {
	Base
	Values []float64
	Width  int
	Height int
	Title  string
}

func NewGraphVisualizer(id string, w, h int) *GraphVisualizer {
	return &GraphVisualizer{Base: *NewBase(id), Width: w, Height: h}
}

func (g *GraphVisualizer) Render(state StateView) []RenderNode {
	if len(g.Values) == 0 {
		return nil
	}

	var nodes []RenderNode
	if g.Title != "" {
		nodes = append(nodes, RenderNode{Type: "text", Content: g.Title, Style: mofu.DefaultStyle().WithAttrs(mofu.AttrBold)})
	}

	// Find min/max
	min, max := g.Values[0], g.Values[0]
	for _, v := range g.Values {
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

	// Render graph rows
	for row := g.Height - 1; row >= 0; row-- {
		line := ""
		threshold := min + (max-min)*float64(row)/float64(g.Height-1)
		for x := 0; x < g.Width; x++ {
			idx := x * len(g.Values) / g.Width
			if idx >= len(g.Values) {
				idx = len(g.Values) - 1
			}
			if g.Values[idx] >= threshold {
				line += "█"
			} else {
				line += " "
			}
		}
		nodes = append(nodes, RenderNode{Type: "text", Content: line})
	}
	return nodes
}

// Spinner shows a loading spinner.
type Spinner struct {
	Base
	Frames []string
	Frame  int
	Label  string
}

func NewSpinner(id string) *Spinner {
	return &Spinner{
		Base:   *NewBase(id),
		Frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	}
}

func (g *Spinner) Render(state StateView) []RenderNode {
	frame := g.Frames[g.Frame%len(g.Frames)]
	text := fmt.Sprintf("%s %s", frame, g.Label)
	return []RenderNode{{Type: "text", Content: text, Style: mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))}}
}

func (g *Spinner) Advance() { g.Frame++ }

// StatusBadge shows a status with color.
type StatusBadge struct {
	Base
	Text   string
	Status string // "success", "warning", "error", "info"
}

func NewStatusBadge(id, text, status string) *StatusBadge {
	return &StatusBadge{Base: *NewBase(id), Text: text, Status: status}
}

func (g *StatusBadge) Render(state StateView) []RenderNode {
	style := mofu.DefaultStyle()
	switch g.Status {
	case "success":
		style = mofu.DefaultStyle().Fg(mofu.Hex("a6e3a1"))
	case "warning":
		style = mofu.DefaultStyle().Fg(mofu.Hex("f9e2af"))
	case "error":
		style = mofu.DefaultStyle().Fg(mofu.Hex("f38ba8"))
	case "info":
		style = mofu.DefaultStyle().Fg(mofu.Hex("7dcfff"))
	}
	return []RenderNode{{Type: "text", Content: fmt.Sprintf("[%s] %s", strings.ToUpper(g.Status), g.Text), Style: style}}
}

// KeyValue shows a key-value pair.
type KeyValue struct {
	Base
	Key   string
	Value any
	Style mofu.Style
}

func NewKeyValue(id, key string, value any) *KeyValue {
	return &KeyValue{Base: *NewBase(id), Key: key, Value: value, Style: mofu.DefaultStyle()}
}

func (g *KeyValue) Render(state StateView) []RenderNode {
	text := fmt.Sprintf("%-20s %v", g.Key+":", g.Value)
	return []RenderNode{{Type: "text", Content: text, Style: g.Style}}
}

// Separator is a horizontal line.
type Separator struct {
	Base
	Char  rune
	Width int
}

func NewSeparator(id string) *Separator {
	return &Separator{Base: *NewBase(id), Char: '─', Width: 40}
}

func (g *Separator) Render(state StateView) []RenderNode {
	line := strings.Repeat(string(g.Char), g.Width)
	return []RenderNode{{Type: "text", Content: line, Style: mofu.DefaultStyle().Fg(mofu.Hex("444444"))}}
}

// Spacer creates empty space.
type Spacer struct {
	Base
	Width  int
	Height int
}

func NewSpacer(id string, w, h int) *Spacer {
	return &Spacer{Base: *NewBase(id), Width: w, Height: h}
}

func (g *Spacer) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	for i := 0; i < g.Height; i++ {
		nodes = append(nodes, RenderNode{Type: "text", Content: strings.Repeat(" ", g.Width)})
	}
	return nodes
}

// Timer shows elapsed time.
type Timer struct {
	Base
	StartTime time.Time
	Label     string
}

func NewTimer(id, label string) *Timer {
	return &Timer{Base: *NewBase(id), StartTime: time.Now(), Label: label}
}

func (g *Timer) Render(state StateView) []RenderNode {
	elapsed := time.Since(g.StartTime)
	text := fmt.Sprintf("%s: %s", g.Label, elapsed.Round(time.Millisecond))
	return []RenderNode{{Type: "text", Content: text, Style: mofu.DefaultStyle().Fg(mofu.Hex("a6e3a1"))}}
}

// Counter shows a count with optional label.
type Counter struct {
	Base
	Value int
	Label string
}

func NewCounter(id, label string) *Counter {
	return &Counter{Base: *NewBase(id), Label: label}
}

func (g *Counter) Render(state StateView) []RenderNode {
	text := fmt.Sprintf("%s: %d", g.Label, g.Value)
	return []RenderNode{{Type: "text", Content: text}}
}

func (g *Counter) Increment() { g.Value++ }
func (g *Counter) Decrement() { g.Value-- }
func (g *Counter) Reset()     { g.Value = 0 }
