package gadgets

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/xanstomper/mofu"
)

// =========================================================================
// BATCH 3: Advanced Data & Interactive Gadgets (10 gadgets)
// =========================================================================

// RealPieChart renders proportional data as ASCII pie segments.
type RealPieChart struct {
	Base
	Title   string
	Segments []PieSegment
	mu      sync.RWMutex
}

type PieSegment struct {
	Label string
	Value float64
	Color mofu.Color
}

func NewRealPieChart(id string) *RealPieChart {
	return &RealPieChart{Base: *NewBase(id)}
}

func (g *RealPieChart) SetSegments(segs []PieSegment) {
	g.mu.Lock()
	g.Segments = segs
	g.mu.Unlock()
}

func (g *RealPieChart) GetTotal() float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	total := 0.0
	for _, s := range g.Segments {
		total += s.Value
	}
	return total
}

func (g *RealPieChart) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	if g.Title != "" {
		titleStyle := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
		ctx.Renderer.WriteString(g.Title, r.X, y, titleStyle.Foreground, titleStyle.Background, titleStyle.Attrs)
		y++
	}

	total := 0.0
	for _, s := range g.Segments {
		total += s.Value
	}
	if total == 0 {
		return
	}

	arc := []rune(" ◢◣◤◥")
	for i, seg := range g.Segments {
		pct := seg.Value / total
		barW := int(pct * float64(r.Width-20))
		if barW < 1 {
			barW = 1
		}

		bar := strings.Repeat(string(arc[i%len(arc)]), barW)
		pctStr := fmt.Sprintf("%.1f%%", pct*100)

		ctx.Renderer.WriteString(fmt.Sprintf(" %-12s %s %s", seg.Label, bar, pctStr), r.X, y, seg.Color, mofu.ColorBlack, 0)
		y++
	}
}

func (g *RealPieChart) HandleEvent(e mofu.Event) mofu.Cmd { return nil }

// RealMiniMap shows a minimap overview of a scrollable document.
type RealMiniMap struct {
	Base
	Lines       []string
	ViewportY   int
	ViewportH   int
	TotalHeight int
	mu          sync.RWMutex
}

func NewRealMiniMap(id string) *RealMiniMap {
	return &RealMiniMap{Base: *NewBase(id)}
}

func (g *RealMiniMap) SetContent(lines []string, vpY, vpH int) {
	g.mu.Lock()
	g.Lines = lines
	g.ViewportY = vpY
	g.ViewportH = vpH
	g.TotalHeight = len(lines)
	g.mu.Unlock()
}

func (g *RealMiniMap) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	h := r.Height

	if g.TotalHeight == 0 {
		return
	}

	scale := float64(h) / float64(g.TotalHeight)
	vpStart := int(float64(g.ViewportY) * scale)
	vpEnd := int(float64(g.ViewportY+g.ViewportH) * scale)
	if vpEnd-vpStart < 1 {
		vpEnd = vpStart + 1
	}

	for y := 0; y < h; y++ {
		docLine := int(float64(y) / scale)
		if docLine >= len(g.Lines) {
			break
		}

		line := g.Lines[docLine]
		trimmed := strings.TrimSpace(line)
		density := float64(len(trimmed)) / float64(len(line)+1)

		var chars string
		n := int(density * float64(r.Width))
		if n < 1 {
			n = 1
		}
		indent := 0
		for _, ch := range line {
			if ch == ' ' || ch == '\t' {
				indent++
			} else {
				break
			}
		}
		indentW := indent * r.Width / (len(line) + 1)

		chars = strings.Repeat(" ", indentW) + strings.Repeat("█", n)
		if len(chars) > r.Width {
			chars = chars[:r.Width]
		}

		if y >= vpStart && y < vpEnd {
			ctx.Renderer.WriteString(chars, r.X, r.Y+y, mofu.Hex("89b4fa"), mofu.Hex("1e1e2e"), 0)
		} else {
			ctx.Renderer.WriteString(chars, r.X, r.Y+y, mofu.Hex("585b70"), mofu.ColorBlack, 0)
		}
	}
}

func (g *RealMiniMap) HandleEvent(e mofu.Event) mofu.Cmd { return nil }

// RealTagCloud displays weighted tags with proportional sizing.
type RealChatInput struct {
	Base
	Messages   []ChatMsg
	Input      string
	CursorPos  int
	MaxHistory int
	mu         sync.RWMutex
	OnSubmit   func(text string) mofu.Cmd
}

type ChatMsg struct {
	Author  string
	Content string
	Time    time.Time
	Color   mofu.Color
}

func NewRealChatInput(id string) *RealChatInput {
	return &RealChatInput{Base: *NewBase(id), MaxHistory: 100}
}

func (g *RealChatInput) AddMessage(author, content string, color mofu.Color) {
	g.mu.Lock()
	g.Messages = append(g.Messages, ChatMsg{
		Author:  author,
		Content: content,
		Time:    time.Now(),
		Color:   color,
	})
	if len(g.Messages) > g.MaxHistory {
		g.Messages = g.Messages[len(g.Messages)-g.MaxHistory:]
	}
	g.mu.Unlock()
}

func (g *RealChatInput) GetMessages() []ChatMsg {
	g.mu.RLock()
	defer g.mu.RUnlock()
	cp := make([]ChatMsg, len(g.Messages))
	copy(cp, g.Messages)
	return cp
}

func (g *RealChatInput) ClearHistory() {
	g.mu.Lock()
	g.Messages = nil
	g.mu.Unlock()
}

func (g *RealChatInput) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	chatH := r.Height - 3
	if chatH < 1 {
		chatH = 1
	}

	start := len(g.Messages) - chatH
	if start < 0 {
		start = 0
	}

	y := r.Y
	for i := start; i < len(g.Messages); i++ {
		msg := g.Messages[i]
		ts := msg.Time.Format("15:04")
		line := fmt.Sprintf("[%s] %s: %s", ts, msg.Author, msg.Content)
		if len(line) > r.Width-1 {
			line = line[:r.Width-4] + "..."
		}
		ctx.Renderer.WriteString(line, r.X, y, msg.Color, mofu.ColorBlack, 0)
		y++
	}

	if y < r.Y+r.Height-2 {
		ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, r.Y+r.Height-2, mofu.Hex("444444"), mofu.ColorBlack, 0)
	}

	input := "> " + g.Input
	if len(input) > r.Width-1 {
		input = input[:r.Width-1]
	}
	ctx.Renderer.WriteString(input, r.X, r.Y+r.Height-1, mofu.Hex("cdd6f4"), mofu.Hex("1e1e2e"), 0)
}

func (g *RealChatInput) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	g.mu.Lock()
	defer g.mu.Unlock()

	switch {
	case ke.Key == mofu.KeyEnter && len(g.Input) > 0:
		text := g.Input
		g.Input = ""
		g.CursorPos = 0
		if g.OnSubmit != nil {
			return g.OnSubmit(text)
		}

	case ke.Key == mofu.KeyBack && len(g.Input) > 0:
		if g.CursorPos > 0 {
			g.Input = g.Input[:g.CursorPos-1] + g.Input[g.CursorPos:]
			g.CursorPos--
		}

	case ke.Key == mofu.KeyLeft:
		if g.CursorPos > 0 {
			g.CursorPos--
		}

	case ke.Key == mofu.KeyRight:
		if g.CursorPos < len(g.Input) {
			g.CursorPos++
		}

	default:
		if len(ke.Runes) > 0 {
			ch := string(ke.Runes)
			g.Input = g.Input[:g.CursorPos] + ch + g.Input[g.CursorPos:]
			g.CursorPos += len(ch)
		}
	}
	return nil
}

// RealNotificationPanel is a notification center with read/unread state.
type RealNotificationPanel struct {
	Base
	Notifications []NotifItem
	Selected      int
	ShowUnreadOnly bool
	mu            sync.RWMutex
	OnDismiss     func(idx int)
	OnRead        func(idx int)
}

type NotifItem struct {
	Title    string
	Message  string
	Time     time.Time
	Level    string
	Read     bool
	Color    mofu.Color
}

func NewRealNotificationPanel(id string) *RealNotificationPanel {
	return &RealNotificationPanel{Base: *NewBase(id)}
}

func (g *RealNotificationPanel) Add(title, message, level string, color mofu.Color) {
	g.mu.Lock()
	g.Notifications = append([]NotifItem{{
		Title:   title,
		Message: message,
		Time:    time.Now(),
		Level:   level,
		Color:   color,
	}}, g.Notifications...)
	g.mu.Unlock()
}

func (g *RealNotificationPanel) GetUnreadCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	n := 0
	for _, notif := range g.Notifications {
		if !notif.Read {
			n++
		}
	}
	return n
}

func (g *RealNotificationPanel) MarkAllRead() {
	g.mu.Lock()
	for i := range g.Notifications {
		g.Notifications[i].Read = true
	}
	g.mu.Unlock()
}

func (g *RealNotificationPanel) Dismiss(idx int) {
	g.mu.Lock()
	if idx >= 0 && idx < len(g.Notifications) {
		g.Notifications = append(g.Notifications[:idx], g.Notifications[idx+1:]...)
		if g.Selected >= len(g.Notifications) {
			g.Selected = len(g.Notifications) - 1
		}
	}
	g.mu.Unlock()
}

func (g *RealNotificationPanel) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	unread := 0
	for _, n := range g.Notifications {
		if !n.Read {
			unread++
		}
	}
	ctx.Renderer.WriteString(fmt.Sprintf(" Notifications (%d unread)", unread), r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	for i, notif := range g.Notifications {
		if y >= r.Y+r.Height {
			break
		}

		if g.ShowUnreadOnly && notif.Read {
			continue
		}

		icon := "●"
		if notif.Read {
			icon = "○"
		}

		ts := notif.Time.Format("15:04")
		title := notif.Title
		if len(title) > r.Width-12 {
			title = title[:r.Width-15] + "..."
		}

		line := fmt.Sprintf("%s [%s] %s %s", icon, ts, title, notif.Level)
		if len(line) > r.Width-1 {
			line = line[:r.Width-1]
		}

		color := notif.Color
		if notif.Read {
			color = mofu.Hex("585b70")
		}

		if i == g.Selected {
			ctx.Renderer.WriteString(strings.Repeat(" ", r.Width-2), r.X+1, y, mofu.ColorWhite, mofu.Hex("2a2a3e"), 0)
		}
		ctx.Renderer.WriteString(line, r.X, y, color, mofu.ColorBlack, 0)
		y++

		if !notif.Read && y < r.Y+r.Height {
			msg := "  " + notif.Message
			if len(msg) > r.Width-2 {
				msg = msg[:r.Width-5] + "..."
			}
			ctx.Renderer.WriteString(msg, r.X+1, y, mofu.Hex("6c7086"), mofu.ColorBlack, 0)
			y++
		}
	}
}

func (g *RealNotificationPanel) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	g.mu.Lock()
	defer g.mu.Unlock()

	switch {
	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		if g.Selected < len(g.Notifications)-1 {
			g.Selected++
		}
	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if g.Selected > 0 {
			g.Selected--
		}
	case ke.Key == mofu.KeyEnter:
		if g.Selected < len(g.Notifications) && g.OnRead != nil {
			g.Notifications[g.Selected].Read = true
			g.OnRead(g.Selected)
		}
	case ke.Key == mofu.KeyBack || (len(ke.Runes) > 0 && ke.Runes[0] == 'd'):
		if g.Selected < len(g.Notifications) {
			idx := g.Selected
			g.Dismiss(idx)
			if g.OnDismiss != nil {
				g.OnDismiss(idx)
			}
		}
	}
	return nil
}

// RealResourceMonitor displays live CPU/memory/disk resource bars.
type RealResourceMonitor struct {
	Base
	Resources []ResourceItem
	mu        sync.RWMutex
}

type ResourceItem struct {
	Name    string
	Current float64
	Max     float64
	Unit    string
	Color   mofu.Color
}

func NewRealResourceMonitor(id string) *RealResourceMonitor {
	return &RealResourceMonitor{Base: *NewBase(id)}
}

func (g *RealResourceMonitor) Set(name string, current, max float64, unit string, color mofu.Color) {
	g.mu.Lock()
	for i, r := range g.Resources {
		if r.Name == name {
			g.Resources[i].Current = current
			g.Resources[i].Max = max
			g.Resources[i].Unit = unit
			g.Resources[i].Color = color
			g.mu.Unlock()
			return
		}
	}
	g.Resources = append(g.Resources, ResourceItem{Name: name, Current: current, Max: max, Unit: unit, Color: color})
	g.mu.Unlock()
}

func (g *RealResourceMonitor) Get(name string) (float64, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	for _, r := range g.Resources {
		if r.Name == name {
			return r.Current, true
		}
	}
	return 0, false
}

func (g *RealResourceMonitor) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	for _, res := range g.Resources {
		if y >= r.Y+r.Height {
			break
		}

		pct := 0.0
		if res.Max > 0 {
			pct = res.Current / res.Max * 100
		}

		barW := r.Width - 28
		if barW < 5 {
			barW = 5
		}
		filled := int(pct / 100 * float64(barW))
		if filled > barW {
			filled = barW
		}
		empty := barW - filled

		barColor := res.Color
		if pct > 90 {
			barColor = mofu.Hex("f38ba8")
		} else if pct > 70 {
			barColor = mofu.Hex("fab387")
		}

		bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
		label := fmt.Sprintf(" %-10s", res.Name)
		if len(label) > 12 {
			label = label[:12]
		}
		val := fmt.Sprintf(" %6.1f%%", pct)

		line := label + bar + val
		ctx.Renderer.WriteString(line, r.X, y, barColor, mofu.ColorBlack, 0)
		y++
	}
}

func (g *RealResourceMonitor) HandleEvent(e mofu.Event) mofu.Cmd { return nil }

// RealQueryBuilder builds SQL-like query conditions visually.
type RealQueryBuilder struct {
	Base
	Fields    []string
	Conditions []QueryCondition
	Selected   int
	mu         sync.RWMutex
	OnQuery    func(conds []QueryCondition)
}

type QueryCondition struct {
	Field    string
	Operator string
	Value    string
	Logic    string
}

var queryOps = []string{"=", "!=", ">", "<", ">=", "<=", "LIKE", "IN", "NOT IN", "IS NULL", "IS NOT NULL"}
var queryLogics = []string{"AND", "OR"}

func NewRealQueryBuilder(id string, fields []string) *RealQueryBuilder {
	return &RealQueryBuilder{Base: *NewBase(id), Fields: fields}
}

func (g *RealQueryBuilder) AddCondition(field, op, value, logic string) {
	g.mu.Lock()
	g.Conditions = append(g.Conditions, QueryCondition{Field: field, Operator: op, Value: value, Logic: logic})
	g.mu.Unlock()
}

func (g *RealQueryBuilder) RemoveCondition(idx int) {
	g.mu.Lock()
	if idx >= 0 && idx < len(g.Conditions) {
		g.Conditions = append(g.Conditions[:idx], g.Conditions[idx+1:]...)
		if g.Selected >= len(g.Conditions) {
			g.Selected = len(g.Conditions) - 1
		}
	}
	g.mu.Unlock()
}

func (g *RealQueryBuilder) BuildQuery() string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if len(g.Conditions) == 0 {
		return "SELECT * FROM table"
	}

	query := "SELECT * FROM table WHERE "
	for i, c := range g.Conditions {
		if i > 0 {
			query += " " + c.Logic + " "
		}
		query += c.Field + " " + c.Operator
		if c.Value != "" {
			query += " '" + c.Value + "'"
		}
	}
	return query
}

func (g *RealQueryBuilder) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(" Query Builder", r.X, y, mofu.Hex("89b4fa"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	for i, cond := range g.Conditions {
		if y >= r.Y+r.Height-3 {
			break
		}

		logic := cond.Logic
		if i == 0 {
			logic = "IF"
		}

		line := fmt.Sprintf(" %s %s %s '%s'", logic, cond.Field, cond.Operator, cond.Value)
		if len(line) > r.Width-2 {
			line = line[:r.Width-5] + "..."
		}

		if i == g.Selected {
			ctx.Renderer.WriteString(">"+line[1:], r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, 0)
		} else {
			ctx.Renderer.WriteString(line, r.X, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
		}
		y++
	}

	if y < r.Y+r.Height-1 {
		ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
		y++
	}

	query := g.BuildQuery()
	if len(query) > r.Width-2 {
		query = query[:r.Width-5] + "..."
	}
	ctx.Renderer.WriteString(" "+query, r.X, y, mofu.Hex("a6e3a1"), mofu.ColorBlack, 0)
}

func (g *RealQueryBuilder) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	g.mu.Lock()
	defer g.mu.Unlock()

	switch {
	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		if g.Selected < len(g.Conditions)-1 {
			g.Selected++
		}
	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if g.Selected > 0 {
			g.Selected--
		}
	case (ke.Key == mofu.KeyBack || ke.Key == mofu.KeyDelete) && len(g.Conditions) > 0:
		g.RemoveCondition(g.Selected)
	case ke.Key == mofu.KeyEnter && g.OnQuery != nil:
		g.OnQuery(g.Conditions)
	}
	return nil
}

// RealSyntaxHighlighter renders code with keyword highlighting.
type RealSyntaxHighlighter struct {
	Base
	Lines     []string
	ScrollY   int
	Language  string
	Highlight map[string]mofu.Color
	mu        sync.RWMutex
}

func NewRealSyntaxHighlighter(id string) *RealSyntaxHighlighter {
	g := &RealSyntaxHighlighter{Base: *NewBase(id)}
	g.Highlight = map[string]mofu.Color{
		"keyword": mofu.Hex("ff69b4"),
		"string":  mofu.Hex("a6e3a1"),
		"number":  mofu.Hex("fab387"),
		"comment": mofu.Hex("6c7086"),
		"func":    mofu.Hex("89b4fa"),
		"type":    mofu.Hex("f9e2af"),
	}
	return g
}

func (g *RealSyntaxHighlighter) SetCode(lines []string, lang string) {
	g.mu.Lock()
	g.Lines = lines
	g.Language = lang
	g.mu.Unlock()
}

var goKeywords = map[string]bool{
	"func": true, "return": true, "if": true, "else": true, "for": true,
	"range": true, "type": true, "struct": true, "interface": true, "package": true,
	"import": true, "var": true, "const": true, "map": true, "chan": true,
	"go": true, "defer": true, "select": true, "case": true, "switch": true,
	"break": true, "continue": true, "nil": true, "true": true, "false": true,
}

func (g *RealSyntaxHighlighter) highlightLine(line string) []struct {
	Text  string
	Color mofu.Color
} {
	var segments []struct {
		Text  string
		Color mofu.Color
	}

	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "--") {
		return append(segments, struct {
			Text  string
			Color mofu.Color
		}{Text: line, Color: g.Highlight["comment"]})
	}

	words := strings.Fields(line)
	current := ""
	for _, word := range words {
		if goKeywords[word] {
			if current != "" {
				segments = append(segments, struct {
					Text  string
					Color mofu.Color
				}{Text: current, Color: mofu.Hex("cdd6f4")})
				current = ""
			}
			segments = append(segments, struct {
				Text  string
				Color mofu.Color
			}{Text: word, Color: g.Highlight["keyword"]})
		} else if strings.HasPrefix(word, "\"") || strings.HasPrefix(word, "'") {
			if current != "" {
				segments = append(segments, struct {
					Text  string
					Color mofu.Color
				}{Text: current, Color: mofu.Hex("cdd6f4")})
				current = ""
			}
			segments = append(segments, struct {
				Text  string
				Color mofu.Color
			}{Text: word, Color: g.Highlight["string"]})
		} else if len(word) > 0 && word[0] >= '0' && word[0] <= '9' {
			if current != "" {
				segments = append(segments, struct {
					Text  string
					Color mofu.Color
				}{Text: current, Color: mofu.Hex("cdd6f4")})
				current = ""
			}
			segments = append(segments, struct {
				Text  string
				Color mofu.Color
			}{Text: word, Color: g.Highlight["number"]})
		} else {
			current += word + " "
		}
	}
	if current != "" {
		segments = append(segments, struct {
			Text  string
			Color mofu.Color
		}{Text: current, Color: mofu.Hex("cdd6f4")})
	}

	return segments
}

func (g *RealSyntaxHighlighter) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	lineNumW := len(fmt.Sprintf("%d", len(g.Lines))) + 1

	for i := g.ScrollY; i < len(g.Lines) && y < r.Y+r.Height; i++ {
		line := g.Lines[i]
		num := fmt.Sprintf("%*d", lineNumW, i+1)
		ctx.Renderer.WriteString(num+" │ ", r.X, y, mofu.Hex("585b70"), mofu.ColorBlack, 0)

		segments := g.highlightLine(line)
		x := r.X + lineNumW + 3
		for _, seg := range segments {
			if x >= r.X+r.Width {
				break
			}
			text := seg.Text
			if x+len(text) > r.X+r.Width {
				text = text[:r.X+r.Width-x]
			}
			ctx.Renderer.WriteString(text, x, y, seg.Color, mofu.ColorBlack, 0)
			x += len(text)
		}
		y++
	}
}

func (g *RealSyntaxHighlighter) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	g.mu.Lock()
	defer g.mu.Unlock()

	switch {
	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		if g.ScrollY < len(g.Lines)-1 {
			g.ScrollY++
		}
	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if g.ScrollY > 0 {
			g.ScrollY--
		}
	case ke.Key == mofu.KeyPgDn:
		g.ScrollY += 20
		if g.ScrollY > len(g.Lines)-1 {
			g.ScrollY = len(g.Lines) - 1
		}
	case ke.Key == mofu.KeyPgUp:
		g.ScrollY -= 20
		if g.ScrollY < 0 {
			g.ScrollY = 0
		}
	case ke.Key == mofu.KeyHome:
		g.ScrollY = 0
	case ke.Key == mofu.KeyEnd:
		g.ScrollY = len(g.Lines) - 1
		if g.ScrollY < 0 {
			g.ScrollY = 0
		}
	}
	return nil
}

// RealColorPicker allows picking colors from a palette.
type RealCalendarView struct {
	Base
	Year      int
	Month     time.Month
	Today     int
	Events    map[int][]string
	Selected  int
	mu        sync.RWMutex
	OnDaySelect func(day int)
}

func NewRealCalendarView(id string) *RealCalendarView {
	now := time.Now()
	return &RealCalendarView{
		Base:   *NewBase(id),
		Year:   now.Year(),
		Month:  now.Month(),
		Today:  now.Day(),
		Events: make(map[int][]string),
	}
}

func (g *RealCalendarView) AddEvent(day int, event string) {
	g.mu.Lock()
	g.Events[day] = append(g.Events[day], event)
	g.mu.Unlock()
}

func (g *RealCalendarView) ClearEvents(day int) {
	g.mu.Lock()
	delete(g.Events, day)
	g.mu.Unlock()
}

func (g *RealCalendarView) SetMonth(year int, month time.Month) {
	g.mu.Lock()
	g.Year = year
	g.Month = month
	g.mu.Unlock()
}

func (g *RealCalendarView) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	title := fmt.Sprintf(" %s %d", g.Month, g.Year)
	ctx.Renderer.WriteString(title, r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++

	headers := " Su Mo Tu We Th Fr Sa"
	ctx.Renderer.WriteString(headers, r.X, y, mofu.Hex("89b4fa"), mofu.ColorBlack, 0)
	y++

	firstDay := time.Date(g.Year, g.Month, 1, 0, 0, 0, 0, time.Local).Weekday()
	daysInMonth := time.Date(g.Year, g.Month+1, 0, 0, 0, 0, 0, time.Local).Day()

	line := strings.Repeat("    ", int(firstDay))
	for day := 1; day <= daysInMonth; day++ {
		cell := fmt.Sprintf("%3d", day)
		if day == g.Today {
			cell = fmt.Sprintf("\033[7m%3d\033[0m", day)
		} else if day == g.Selected {
			cell = fmt.Sprintf("[%2d]", day)
		} else {
			cell = fmt.Sprintf(" %2d", day)
		}

		if len(line)+4 > r.Width {
			ctx.Renderer.WriteString(line, r.X, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
			y++
			line = ""
		}
		line += cell
	}

	if line != "" {
		ctx.Renderer.WriteString(line, r.X, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
		y++
	}

	y++
	eventCount := len(g.Events[g.Selected])
	if eventCount > 0 {
		ctx.Renderer.WriteString(fmt.Sprintf(" Day %d: %d events", g.Selected, eventCount), r.X, y, mofu.Hex("a6e3a1"), mofu.ColorBlack, 0)
	}
}

func (g *RealCalendarView) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	g.mu.Lock()
	defer g.mu.Unlock()

	daysInMonth := time.Date(g.Year, g.Month+1, 0, 0, 0, 0, 0, time.Local).Day()

	switch {
	case ke.Key == mofu.KeyRight || (len(ke.Runes) > 0 && ke.Runes[0] == 'l'):
		if g.Selected < daysInMonth {
			g.Selected++
		}
	case ke.Key == mofu.KeyLeft || (len(ke.Runes) > 0 && ke.Runes[0] == 'h'):
		if g.Selected > 1 {
			g.Selected--
		}
	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		g.Selected += 7
		if g.Selected > daysInMonth {
			g.Selected = daysInMonth
		}
	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		g.Selected -= 7
		if g.Selected < 1 {
			g.Selected = 1
		}
	case ke.Key == mofu.KeyEnter && g.OnDaySelect != nil:
		g.OnDaySelect(g.Selected)
	}

	return nil
}
