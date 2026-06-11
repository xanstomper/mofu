package agent

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/xanstomper/mofu"
)

// =========================================================================
// 4. Sleek Panel System — polished terminal UI like OpenTUI/OpenCode but better
// =========================================================================

// Panel is a styled border container with title, focus management.
type Panel struct {
	Title     string
	Focused   bool
	BorderFg  mofu.Color
	TitleFg   mofu.Color
	BgColor   mofu.Color
	Content   func(ctx *mofu.RenderContext, x, y, w, h int)
	mu        sync.RWMutex
}

func NewPanel(title string) *Panel {
	return &Panel{
		Title:    title,
		BorderFg: mofu.Hex("45475a"),
		TitleFg:  mofu.Hex("89b4fa"),
		BgColor:  mofu.ColorBlack,
	}
}

func (p *Panel) SetFocused(focused bool) {
	p.mu.Lock()
	p.Focused = focused
	if focused {
		p.BorderFg = mofu.Hex("89b4fa")
		p.TitleFg = mofu.Hex("89b4fa")
	} else {
		p.BorderFg = mofu.Hex("45475a")
		p.TitleFg = mofu.Hex("6c7086")
	}
	p.mu.Unlock()
}

func (p *Panel) RenderFrame(ctx *mofu.RenderContext, x, y, w, h int) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if w < 2 || h < 2 {
		return
	}

	borderFg := p.BorderFg
	if p.Focused {
		borderFg = mofu.Hex("89b4fa")
	}

	// Top border with title
	if p.Title != "" {
		title := fmt.Sprintf(" %s ", p.Title)
		maxTitle := w - 2
		if len(title) > maxTitle {
			title = title[:maxTitle-1] + " "
		}

		gap := w - 2 - len(title)
		leftGap := gap / 2
		rightGap := gap - leftGap

		topLine := "╭" + strings.Repeat("─", leftGap) + title + strings.Repeat("─", rightGap) + "╮"
		if len(topLine) > w {
			topLine = topLine[:w]
		}
		ctx.Renderer.WriteString(topLine, x, y, borderFg, mofu.ColorBlack, 0)
	} else {
		ctx.Renderer.WriteString("╭"+strings.Repeat("─", w-2)+"╮", x, y, borderFg, mofu.ColorBlack, 0)
	}

	// Side borders
	for i := 1; i < h-1; i++ {
		if y+i >= ctx.Bounds.Y+ctx.Bounds.Height {
			break
		}
		ctx.Renderer.WriteString("│", x, y+i, borderFg, mofu.ColorBlack, 0)
		ctx.Renderer.WriteString("│", x+w-1, y+i, borderFg, mofu.ColorBlack, 0)
	}

	// Bottom border
	if y+h-1 < ctx.Bounds.Y+ctx.Bounds.Height {
		ctx.Renderer.WriteString("╰"+strings.Repeat("─", w-2)+"╯", x, y+h-1, borderFg, mofu.ColorBlack, 0)
	}
}

// =========================================================================
// StatusBar — sleek status bar with sections
// =========================================================================

type StatusBar struct {
	Sections []StatusSection
	mu       sync.RWMutex
}

type StatusSection struct {
	Label string
	Value string
	Icon  string
	Color mofu.Color
}

func NewStatusBar() *StatusBar {
	return &StatusBar{}
}

func (sb *StatusBar) Set(label, value, icon string, color mofu.Color) {
	sb.mu.Lock()
	for i, s := range sb.Sections {
		if s.Label == label {
			sb.Sections[i] = StatusSection{Label: label, Value: value, Icon: icon, Color: color}
			sb.mu.Unlock()
			return
		}
	}
	sb.Sections = append(sb.Sections, StatusSection{Label: label, Value: value, Icon: icon, Color: color})
	sb.mu.Unlock()
}

func (sb *StatusBar) Render(ctx *mofu.RenderContext, y int) {
	sb.mu.RLock()
	defer sb.mu.RUnlock()

	r := ctx.Bounds
	bg := mofu.Hex("1e1e2e")

	// Fill background
	ctx.Renderer.WriteString(strings.Repeat(" ", r.Width), r.X, y, mofu.Hex("cdd6f4"), bg, 0)

	x := r.X + 1
	for _, section := range sb.Sections {
		text := fmt.Sprintf(" %s %s: %s ", section.Icon, section.Label, section.Value)
		if x+len(text) > r.X+r.Width-1 {
			break
		}
		ctx.Renderer.WriteString(text, x, y, section.Color, bg, 0)
		x += len(text)
	}
}

// =========================================================================
// NotificationBar — toast-style notifications
// =========================================================================

type Notification struct {
	Message   string
	Level     string
	Timestamp time.Time
	Color     mofu.Color
}

type NotificationBar struct {
	notifications []Notification
	maxVisible    int
	mu            sync.RWMutex
}

func NewNotificationBar(maxVisible int) *NotificationBar {
	return &NotificationBar{maxVisible: maxVisible}
}

func (nb *NotificationBar) Push(message, level string) {
	color := mofu.Hex("cdd6f4")
	switch level {
	case "error":
		color = mofu.Hex("f38ba8")
	case "warning":
		color = mofu.Hex("fab387")
	case "success":
		color = mofu.Hex("a6e3a1")
	case "info":
		color = mofu.Hex("89b4fa")
	}

	nb.mu.Lock()
	nb.notifications = append(nb.notifications, Notification{
		Message:   message,
		Level:     level,
		Timestamp: time.Now(),
		Color:     color,
	})
	if len(nb.notifications) > 50 {
		nb.notifications = nb.notifications[len(nb.notifications)-50:]
	}
	nb.mu.Unlock()
}

func (nb *NotificationBar) Render(ctx *mofu.RenderContext, y int) {
	nb.mu.RLock()
	defer nb.mu.RUnlock()

	r := ctx.Bounds
	start := len(nb.notifications) - nb.maxVisible
	if start < 0 {
		start = 0
	}

	for i := start; i < len(nb.notifications); i++ {
		if y >= ctx.Bounds.Y+ctx.Bounds.Height {
			break
		}

		n := nb.notifications[i]
		icon := "●"
		switch n.Level {
		case "error":
			icon = "✗"
		case "warning":
			icon = "⚠"
		case "success":
			icon = "✓"
		case "info":
			icon = "ℹ"
		}

		age := time.Since(n.Timestamp).Round(time.Second)
		line := fmt.Sprintf(" %s %s (%s ago)", icon, n.Message, age)
		if len(line) > r.Width-2 {
			line = line[:r.Width-5] + "..."
		}
		ctx.Renderer.WriteString(line, r.X, y, n.Color, mofu.ColorBlack, 0)
		y++
	}
}

// =========================================================================
// AppLayout — complete polished terminal app layout
// =========================================================================

type AppLayout struct {
	Panels      []*Panel
	StatusBar   *StatusBar
	NotifBar    *NotificationBar
	FocusedIdx  int
	Width       int
	Height      int
	mu          sync.RWMutex
}

func NewAppLayout() *AppLayout {
	return &AppLayout{
		StatusBar: NewStatusBar(),
		NotifBar:  NewNotificationBar(3),
	}
}

func (al *AppLayout) AddPanel(title string) *Panel {
	p := NewPanel(title)
	al.Panels = append(al.Panels, p)
	return p
}

func (al *AppLayout) SetFocus(idx int) {
	al.mu.Lock()
	for i, p := range al.Panels {
		p.SetFocused(i == idx)
	}
	al.FocusedIdx = idx
	al.mu.Unlock()
}

func (al *AppLayout) NextFocus() {
	al.mu.Lock()
	if len(al.Panels) > 0 {
		al.Panels[al.FocusedIdx].SetFocused(false)
		al.FocusedIdx = (al.FocusedIdx + 1) % len(al.Panels)
		al.Panels[al.FocusedIdx].SetFocused(true)
	}
	al.mu.Unlock()
}

func (al *AppLayout) PrevFocus() {
	al.mu.Lock()
	if len(al.Panels) > 0 {
		al.Panels[al.FocusedIdx].SetFocused(false)
		al.FocusedIdx--
		if al.FocusedIdx < 0 {
			al.FocusedIdx = len(al.Panels) - 1
		}
		al.Panels[al.FocusedIdx].SetFocused(true)
	}
	al.mu.Unlock()
}

func (al *AppLayout) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	if ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q') {
		return mofu.QuitCmd()
	}
	if ke.Key == mofu.KeyTab {
		al.NextFocus()
	}
	return nil
}

// =========================================================================
// StreamDisplay — complete streaming output display with all the bells
// =========================================================================

type StreamDisplay struct {
	mofu.Minimal
	agent      *InstantAgent
	buffer     *StreamDisplayBuffer
	panels     *AppLayout
	outputPanel *Panel
	toolPanel  *Panel
	statusBar  *StatusBar
	notifier   *NotificationBar
	width      int
	height     int
	mu         sync.RWMutex
}

type StreamDisplayBuffer struct {
	lines    []string
	maxLines int
	scrollY  int
	mu       sync.RWMutex
}

func NewStreamDisplayBuffer(maxLines int) *StreamDisplayBuffer {
	return &StreamDisplayBuffer{maxLines: maxLines}
}

func (b *StreamDisplayBuffer) Append(text string) {
	b.mu.Lock()
	b.lines = append(b.lines, text)
	if len(b.lines) > b.maxLines {
		b.lines = b.lines[len(b.lines)-b.maxLines:]
	}
	b.mu.Unlock()
}

func (b *StreamDisplayBuffer) Clear() {
	b.mu.Lock()
	b.lines = nil
	b.mu.Unlock()
}

func (b *StreamDisplayBuffer) Lines() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	cp := make([]string, len(b.lines))
	copy(cp, b.lines)
	return cp
}

func (b *StreamDisplayBuffer) ScrollDown(n int) {
	b.mu.Lock()
	b.scrollY += n
	maxScroll := len(b.lines) - 10
	if maxScroll < 0 {
		maxScroll = 0
	}
	if b.scrollY > maxScroll {
		b.scrollY = maxScroll
	}
	b.mu.Unlock()
}

func (b *StreamDisplayBuffer) ScrollUp(n int) {
	b.mu.Lock()
	b.scrollY -= n
	if b.scrollY < 0 {
		b.scrollY = 0
	}
	b.mu.Unlock()
}

func NewStreamDisplay(agent *InstantAgent) *StreamDisplay {
	sd := &StreamDisplay{
		agent:   agent,
		buffer:  NewStreamDisplayBuffer(10000),
		panels:  NewAppLayout(),
		statusBar: NewStatusBar(),
		notifier:  NewNotificationBar(3),
	}

	sd.outputPanel = sd.panels.AddPanel("Output")
	sd.toolPanel = sd.panels.AddPanel("Tools")
	sd.panels.SetFocus(0)

	sd.statusBar.Set("model", agent.api.Model, "🤖", mofu.Hex("89b4fa"))
	sd.statusBar.Set("status", "ready", "●", mofu.Hex("a6e3a1"))

	return sd
}

func (sd *StreamDisplay) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	sd.mu.RLock()
	sd.width = r.Width
	sd.height = r.Height
	sd.mu.RUnlock()

	// Main output area
	outputH := r.Height - 2
	outputW := r.Width

	sd.outputPanel.RenderFrame(ctx, r.X, r.Y, outputW, outputH)

	// Render buffer content inside panel
	sd.buffer.mu.RLock()
	lines := sd.buffer.lines
	scrollY := sd.buffer.scrollY
	sd.buffer.mu.RUnlock()

	viewH := outputH - 2
	start := scrollY
	if start > len(lines)-viewH {
		start = len(lines) - viewH
	}
	if start < 0 {
		start = 0
	}

	for i := start; i < len(lines) && i-start < viewH; i++ {
		y := r.Y + 1 + (i - start)
		line := lines[i]
		if len(line) > outputW-3 {
			line = line[:outputW-6] + "..."
		}

		fg := mofu.Hex("cdd6f4")
		if strings.HasPrefix(line, "# ") {
			fg = mofu.Hex("ff69b4")
			line = line[2:]
		} else if strings.HasPrefix(line, "## ") {
			fg = mofu.Hex("89b4fa")
			line = line[3:]
		} else if strings.HasPrefix(line, "> ") {
			fg = mofu.Hex("6c7086")
			line = line[2:]
		} else if strings.HasPrefix(line, "```") {
			fg = mofu.Hex("a6e3a1")
		}

		ctx.Renderer.WriteString(" "+line, r.X+1, y, fg, mofu.ColorBlack, 0)
	}

	// Scroll indicator
	if len(lines) > viewH {
		pct := float64(scrollY) / float64(len(lines)-viewH)
		thumbY := r.Y + 1 + int(pct*float64(viewH-1))
		ctx.Renderer.WriteString("█", r.X+outputW-2, thumbY, mofu.Hex("45475a"), mofu.ColorBlack, 0)
	}

	// Status bar at bottom
	sd.statusBar.Set("tokens", fmt.Sprintf("%d", sd.agent.totalTokens), "📊", mofu.Hex("f9e2af"))
	state := sd.agent.GetState()
	stateColor := mofu.Hex("585b70")
	switch state {
	case StateStreaming:
		stateColor = mofu.Hex("a6e3a1")
	case StateError:
		stateColor = mofu.Hex("f38ba8")
	case StateDone:
		stateColor = mofu.Hex("a6e3a1")
	}
	sd.statusBar.Set("state", state.String(), "●", stateColor)
	sd.statusBar.Render(ctx, r.Y+r.Height-1)
}

func (sd *StreamDisplay) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()
	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		sd.buffer.ScrollDown(1)
	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		sd.buffer.ScrollUp(1)
	case ke.Key == mofu.KeyPgDn:
		sd.buffer.ScrollDown(20)
	case ke.Key == mofu.KeyPgUp:
		sd.buffer.ScrollUp(20)
	case ke.Key == mofu.KeyHome:
		sd.buffer.mu.Lock()
		sd.buffer.scrollY = 0
		sd.buffer.mu.Unlock()
	case ke.Key == mofu.KeyEnd:
		sd.buffer.mu.Lock()
		sd.buffer.scrollY = len(sd.buffer.lines) - 10
		if sd.buffer.scrollY < 0 {
			sd.buffer.scrollY = 0
		}
		sd.buffer.mu.Unlock()
	}
	return nil
}
