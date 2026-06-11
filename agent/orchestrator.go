package agent

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/xanstomper/mofu"
)

// =========================================================================
// Multi-Agent Orchestration Display
// =========================================================================

// Orchestrator manages and displays multiple agents working in parallel.
type Orchestrator struct {
	mofu.Minimal
	Agents    []*Agent
	Selected  int
	Layout    string // "tabs", "grid", "focus"
	mu        sync.RWMutex
}

func NewOrchestrator(layout string) *Orchestrator {
	return &Orchestrator{Layout: layout}
}

func (o *Orchestrator) AddAgent(name string) *Agent {
	agent := NewAgent(name)
	o.mu.Lock()
	o.Agents = append(o.Agents, agent)
	o.mu.Unlock()
	return agent
}

func (o *Orchestrator) RemoveAgent(name string) {
	o.mu.Lock()
	for i, a := range o.Agents {
		if a.Name == name {
			o.Agents = append(o.Agents[:i], o.Agents[i+1:]...)
			if o.Selected >= len(o.Agents) && o.Selected > 0 {
				o.Selected--
			}
			break
		}
	}
	o.mu.Unlock()
}

func (o *Orchestrator) GetAgent(name string) *Agent {
	o.mu.RLock()
	defer o.mu.RUnlock()
	for _, a := range o.Agents {
		if a.Name == name {
			return a
		}
	}
	return nil
}

func (o *Orchestrator) ActiveCount() int {
	o.mu.RLock()
	defer o.mu.RUnlock()
	count := 0
	for _, a := range o.Agents {
		if a.State != StateIdle && a.State != StateDone {
			count++
		}
	}
	return count
}

func (o *Orchestrator) Render(ctx *mofu.RenderContext) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	// Header with agent tabs
	active := o.ActiveCount()
	ctx.Renderer.WriteString(fmt.Sprintf(" Agents (%d/%d active)", active, len(o.Agents)), r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++

	// Tab bar
	tabLine := ""
	for i, a := range o.Agents {
		icon := "○"
		switch a.State {
		case StateThinking, StateToolCall, StateStreaming:
			icon = "●"
		case StateError:
			icon = "✗"
		case StateDone:
			icon = "✓"
		}

		name := a.Name
		if len(name) > 12 {
			name = name[:10] + ".."
		}

		tab := fmt.Sprintf(" %s%s ", icon, name)
		if i == o.Selected {
			tab = "[" + tab + "]"
		}

		if len(tabLine)+len(tab) > r.Width-2 {
			break
		}
		tabLine += tab
	}
	ctx.Renderer.WriteString(tabLine, r.X, y, mofu.Hex("89b4fa"), mofu.Hex("1e1e2e"), 0)
	y++

	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	// Render selected agent
	if o.Selected < len(o.Agents) {
		agentCtx := &mofu.RenderContext{
			Bounds:   mofu.Rect{X: r.X, Y: y, Width: r.Width, Height: r.Height - (y - r.Y) - 1},
			Renderer: ctx.Renderer,
		}
		o.Agents[o.Selected].Render(agentCtx)
	}

	// Status bar
	status := fmt.Sprintf(" ←→:switch q:quit")
	ctx.Renderer.WriteString(status, r.X, r.Y+r.Height-1, mofu.Hex("585b70"), mofu.Hex("1e1e2e"), 0)
}

func (o *Orchestrator) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	o.mu.Lock()
	defer o.mu.Unlock()

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()
	case ke.Key == mofu.KeyLeft || (len(ke.Runes) > 0 && ke.Runes[0] == 'h'):
		if o.Selected > 0 {
			o.Selected--
		}
	case ke.Key == mofu.KeyRight || (len(ke.Runes) > 0 && ke.Runes[0] == 'l'):
		if o.Selected < len(o.Agents)-1 {
			o.Selected++
		}
	case ke.Key == mofu.KeyCtrlC:
		return mofu.QuitCmd()
	}
	return nil
}

// =========================================================================
// EventTimeline — chronological event log for agents
// =========================================================================

type EventTimeline struct {
	mofu.Minimal
	Events    []TimelineEvent
	MaxEvents int
	Filter    string
	Selected  int
	mu        sync.RWMutex
}

type TimelineEvent struct {
	Timestamp time.Time
	Agent     string
	Type      string
	Content   string
	Color     mofu.Color
}

func NewEventTimeline(maxEvents int) *EventTimeline {
	return &EventTimeline{MaxEvents: maxEvents}
}

func (et *EventTimeline) Add(agentName, eventType, content string, color mofu.Color) {
	et.mu.Lock()
	et.Events = append(et.Events, TimelineEvent{
		Timestamp: time.Now(),
		Agent:     agentName,
		Type:      eventType,
		Content:   content,
		Color:     color,
	})
	if len(et.Events) > et.MaxEvents {
		et.Events = et.Events[len(et.Events)-et.MaxEvents:]
	}
	et.mu.Unlock()
}

func (et *EventTimeline) Clear() {
	et.mu.Lock()
	et.Events = nil
	et.mu.Unlock()
}

func (et *EventTimeline) Render(ctx *mofu.RenderContext) {
	et.mu.RLock()
	defer et.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(fmt.Sprintf(" Event Log (%d events)", len(et.Events)), r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	for i, ev := range et.Events {
		if y >= r.Y+r.Height-1 {
			break
		}

		if et.Filter != "" && !strings.Contains(strings.ToLower(ev.Content), strings.ToLower(et.Filter)) &&
			!strings.Contains(strings.ToLower(ev.Agent), strings.ToLower(et.Filter)) {
			continue
		}

		ts := ev.Timestamp.Format("15:04:05.000")
		icon := "●"
		switch ev.Type {
		case "tool_start":
			icon = "⚡"
		case "tool_end":
			icon = "✓"
		case "thinking":
			icon = "◆"
		case "error":
			icon = "✗"
		case "stream":
			icon = "▶"
		}

		line := fmt.Sprintf(" %s %s %-12s %s %s", ts, icon, ev.Agent, ev.Type, ev.Content)
		if len(line) > r.Width-2 {
			line = line[:r.Width-5] + "..."
		}

		if i == et.Selected {
			ctx.Renderer.WriteString(strings.Repeat(" ", r.Width-2), r.X+1, y, mofu.ColorWhite, mofu.Hex("2a2a3e"), 0)
		}
		ctx.Renderer.WriteString(line, r.X, y, ev.Color, mofu.ColorBlack, 0)
		y++
	}
}

func (et *EventTimeline) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	et.mu.Lock()
	defer et.mu.Unlock()

	switch {
	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		if et.Selected < len(et.Events)-1 {
			et.Selected++
		}
	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if et.Selected > 0 {
			et.Selected--
		}
	}
	return nil
}

// =========================================================================
// AgentDashboard — full-screen agent monitoring dashboard
// =========================================================================

type AgentDashboard struct {
	mofu.Minimal
	Orchestrator *Orchestrator
	Timeline     *EventTimeline
	Costs        *CostBar
	width        int
	height       int
	mu           sync.RWMutex
}

func NewAgentDashboard() *AgentDashboard {
	return &AgentDashboard{
		Orchestrator: NewOrchestrator("tabs"),
		Timeline:     NewEventTimeline(100),
		Costs:        NewCostBar(128000),
	}
}

func (d *AgentDashboard) AddAgent(name string) *Agent {
	return d.Orchestrator.AddAgent(name)
}

func (d *AgentDashboard) LogEvent(agentName, eventType, content string) {
	colors := map[string]mofu.Color{
		"tool_start": mofu.Hex("89b4fa"),
		"tool_end":   mofu.Hex("a6e3a1"),
		"thinking":   mofu.Hex("f9e2af"),
		"error":      mofu.Hex("f38ba8"),
		"stream":     mofu.Hex("cdd6f4"),
		"info":       mofu.Hex("6c7086"),
	}
	color := colors[eventType]
	if color == (mofu.Color{}) {
		color = mofu.Hex("cdd6f4")
	}
	d.Timeline.Add(agentName, eventType, content, color)
}

func (d *AgentDashboard) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	d.mu.RLock()
	defer d.mu.RUnlock()
	d.width = r.Width
	d.height = r.Height

	leftW := r.Width * 3 / 5
	rightW := r.Width - leftW

	// Left: Orchestrator (agents)
	orchCtx := &mofu.RenderContext{
		Bounds:   mofu.Rect{X: r.X, Y: r.Y, Width: leftW - 1, Height: r.Height - 2},
		Renderer: ctx.Renderer,
	}
	d.Orchestrator.Render(orchCtx)

	// Separator
	ctx.Renderer.WriteString("│", r.X+leftW-1, r.Y, mofu.Hex("444444"), mofu.ColorBlack, 0)

	// Right top: Timeline
	timelineH := r.Height * 2 / 3
	timelineCtx := &mofu.RenderContext{
		Bounds:   mofu.Rect{X: r.X + leftW, Y: r.Y, Width: rightW, Height: timelineH - 1},
		Renderer: ctx.Renderer,
	}
	d.Timeline.Render(timelineCtx)

	// Separator
	if timelineH < r.Height-2 {
		ctx.Renderer.WriteString(strings.Repeat("─", rightW-1), r.X+leftW, r.Y+timelineH, mofu.Hex("444444"), mofu.ColorBlack, 0)
	}

	// Right bottom: Costs
	costH := r.Height - timelineH - 2
	if costH > 0 {
		costCtx := &mofu.RenderContext{
			Bounds:   mofu.Rect{X: r.X + leftW, Y: r.Y + timelineH + 1, Width: rightW, Height: costH},
			Renderer: ctx.Renderer,
		}
		d.Costs.Render(costCtx)
	}

	// Bottom status bar
	status := fmt.Sprintf(" %d agents | %d events | $%.4f total", len(d.Orchestrator.Agents), len(d.Timeline.Events), d.Costs.CostIn+d.Costs.CostOut)
	if len(status) > r.Width {
		status = status[:r.Width]
	}
	ctx.Renderer.WriteString(status, r.X, r.Y+r.Height-1, mofu.Hex("585b70"), mofu.Hex("1e1e2e"), 0)
}

func (d *AgentDashboard) HandleEvent(e mofu.Event) mofu.Cmd {
	return d.Orchestrator.HandleEvent(e)
}
