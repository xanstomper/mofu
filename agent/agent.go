// Package agent provides AI-native TUI components for agentic workflows.
//
// It includes components for displaying agent state, tool calls, streaming output,
// multi-agent orchestration, and cost tracking. The framework handles API streaming
// (OpenAI, Anthropic, Ollama), virtual scrolling for massive datasets, and provides
// a polished panel-based layout system.
//
// Basic usage:
//
//	a := agent.NewAgent("my-agent")
//	a.BeginToolCall("bash", "ls -la")
//	a.EndToolCall("file1.go file2.go", nil)
//	a.AppendStream("Here are the files...")
//	a.FinishStep(42, 0.001)
//
// For real API streaming:
//
//	a := agent.NewInstantAgent("assistant", apiURL, apiKey, model)
//	a.OnToken(func(token string) { /* render instantly */ })
//	a.Send("What is 2+2?")
package agent

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/xanstomper/mofu"
)

// =========================================================================
// Agent Display Framework
// AI-native TUI components for agentic workflows.
// Superior to OpenTUI/Bubble Tea for agent output display.
// =========================================================================

// AgentState represents the current state of an AI agent.
type AgentState int

const (
	StateIdle AgentState = iota
	StateThinking
	StateToolCall
	StateStreaming
	StateWaiting
	StateError
	StateDone
)

func (s AgentState) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StateThinking:
		return "thinking"
	case StateToolCall:
		return "tool_call"
	case StateStreaming:
		return "streaming"
	case StateWaiting:
		return "waiting"
	case StateError:
		return "error"
	case StateDone:
		return "done"
	default:
		return "unknown"
	}
}

// Step represents a single agent execution step.
type Step struct {
	ID        string
	Type      string
	Content   string
	State     AgentState
	ToolName  string
	ToolInput string
	ToolOutput string
	Error     string
	StartedAt time.Time
	EndedAt   time.Time
	Tokens    int
	Cost      float64
}

func (s Step) Duration() time.Duration {
	if s.EndedAt.IsZero() {
		return time.Since(s.StartedAt)
	}
	return s.EndedAt.Sub(s.StartedAt)
}

// Agent is the main AI agent display component.
type Agent struct {
	mofu.Minimal
	Name      string
	State     AgentState
	Steps     []Step
	Current   string
	Thinking  string
	CursorPos int
	Width     int
	Height    int
	mu        sync.RWMutex

	// Callbacks
	OnToolCall  func(name, input string)
	OnInterrupt func()

	// Metrics
	TotalTokens  int
	TotalCost    float64
	TotalSteps   int
	StartTime    time.Time
}

func NewAgent(name string) *Agent {
	return &Agent{
		Name:      name,
		State:     StateIdle,
		StartTime: time.Now(),
	}
}

func (a *Agent) BeginThinking(content string) {
	a.mu.Lock()
	a.State = StateThinking
	a.Thinking = content
	a.mu.Unlock()
}

func (a *Agent) EndThinking() {
	a.mu.Lock()
	a.Thinking = ""
	a.mu.Unlock()
}

func (a *Agent) BeginToolCall(name, input string) {
	a.mu.Lock()
	a.State = StateToolCall
	a.Steps = append(a.Steps, Step{
		ID:        fmt.Sprintf("step-%d", len(a.Steps)+1),
		Type:      "tool",
		ToolName:  name,
		ToolInput: input,
		State:     StateToolCall,
		StartedAt: time.Now(),
	})
	a.mu.Unlock()
}

func (a *Agent) EndToolCall(output string, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.Steps) > 0 {
		last := &a.Steps[len(a.Steps)-1]
		last.ToolOutput = output
		last.EndedAt = time.Now()
		if err != nil {
			last.Error = err.Error()
			last.State = StateError
		} else {
			last.State = StateDone
		}
	}
	a.State = StateStreaming
}

func (a *Agent) AppendStream(token string) {
	a.mu.Lock()
	a.Current += token
	a.TotalSteps++
	a.mu.Unlock()
}

func (a *Agent) FinishStep(tokens int, cost float64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.TotalTokens += tokens
	a.TotalCost += cost
	a.Steps = append(a.Steps, Step{
		ID:        fmt.Sprintf("step-%d", len(a.Steps)+1),
		Type:      "output",
		Content:   a.Current,
		State:     StateDone,
		StartedAt: time.Now(),
		EndedAt:   time.Now(),
		Tokens:    tokens,
		Cost:      cost,
	})
	a.Current = ""
	a.State = StateIdle
}

func (a *Agent) SetError(err string) {
	a.mu.Lock()
	a.State = StateError
	a.mu.Unlock()
}

func (a *Agent) Render(ctx *mofu.RenderContext) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	r := ctx.Bounds
	a.Width = r.Width
	a.Height = r.Height
	y := r.Y

	// Agent header with status
	header := a.renderHeader(r)
	ctx.Renderer.WriteString(header, r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++

	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	// Render past steps
	for _, step := range a.Steps {
		if y >= r.Y+r.Height-4 {
			break
		}
		y = a.renderStep(ctx, step, y)
	}

	// Render current thinking
	if a.State == StateThinking && a.Thinking != "" {
		y = a.renderThinking(ctx, y)
	}

	// Render current streaming output
	if a.State == StateStreaming && a.Current != "" {
		y = a.renderCurrent(ctx, y)
	}

	// Render current tool call
	if a.State == StateToolCall {
		y = a.renderCurrentTool(ctx, y)
	}

	// Status bar
	status := fmt.Sprintf(" %d tok | $%.4f | %d steps | i:interrupt q:quit", a.TotalTokens, a.TotalCost, len(a.Steps))
	if len(status) > a.Width {
		status = status[:a.Width]
	}
	ctx.Renderer.WriteString(status, r.X, r.Y+r.Height-1, mofu.Hex("585b70"), mofu.Hex("1e1e2e"), 0)
}

func (a *Agent) renderHeader(r mofu.Rect) string {
	statusIcon := "○"
	statusText := "idle"

	switch a.State {
	case StateThinking:
		statusIcon = "◆"
		statusText = "thinking"
	case StateToolCall:
		statusIcon = "⚡"
		statusText = "tool call"
	case StateStreaming:
		statusIcon = "▶"
		statusText = "streaming"
	case StateError:
		statusIcon = "✗"
		statusText = "error"
	case StateDone:
		statusIcon = "✓"
		statusText = "done"
	}

	elapsed := time.Since(a.StartTime).Round(time.Second)
	return fmt.Sprintf(" %s %s [%s] %s", statusIcon, a.Name, statusText, elapsed)
}

func (a *Agent) renderStep(ctx *mofu.RenderContext, step Step, y int) int {
	r := ctx.Bounds
	if y >= r.Y+r.Height-4 {
		return y
	}

	switch step.Type {
	case "tool":
		icon := "⚡"
		color := mofu.Hex("89b4fa")
		if step.State == StateError {
			icon = "✗"
			color = mofu.Hex("f38ba8")
		} else if step.State == StateDone {
			icon = "✓"
			color = mofu.Hex("a6e3a1")
		}

		header := fmt.Sprintf(" %s %s", icon, step.ToolName)
		ctx.Renderer.WriteString(header, r.X, y, color, mofu.ColorBlack, mofu.AttrBold)
		y++

		if step.ToolInput != "" {
			input := step.ToolInput
			if len(input) > r.Width-4 {
				input = input[:r.Width-7] + "..."
			}
			ctx.Renderer.WriteString(" │ "+input, r.X, y, mofu.Hex("6c7086"), mofu.ColorBlack, 0)
			y++
		}

		if step.ToolOutput != "" {
			output := step.ToolOutput
			if len(output) > r.Width-4 {
				output = output[:r.Width-7] + "..."
			}
			ctx.Renderer.WriteString(" │ "+output, r.X, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
			y++
		}

		if step.Error != "" {
			errText := step.Error
			if len(errText) > r.Width-4 {
				errText = errText[:r.Width-7] + "..."
			}
			ctx.Renderer.WriteString(" │ ERROR: "+errText, r.X, y, mofu.Hex("f38ba8"), mofu.ColorBlack, 0)
			y++
		}

	case "output":
		lines := strings.Split(step.Content, "\n")
		for _, line := range lines {
			if y >= r.Y+r.Height-4 {
				break
			}
			display := line
			if len(display) > r.Width-2 {
				display = display[:r.Width-5] + "..."
			}
			ctx.Renderer.WriteString(" "+display, r.X, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
			y++
		}
	}

	return y
}

func (a *Agent) renderThinking(ctx *mofu.RenderContext, y int) int {
	r := ctx.Bounds
	if y >= r.Y+r.Height-4 {
		return y
	}

	ctx.Renderer.WriteString(" ◆ thinking...", r.X, y, mofu.Hex("f9e2af"), mofu.ColorBlack, mofu.AttrBold)
	y++

	if a.Thinking != "" {
		lines := strings.Split(a.Thinking, "\n")
		for _, line := range lines {
			if y >= r.Y+r.Height-4 {
				break
			}
			display := line
			if len(display) > r.Width-4 {
				display = display[:r.Width-7] + "..."
			}
			ctx.Renderer.WriteString(" │ "+display, r.X, y, mofu.Hex("6c7086"), mofu.ColorBlack, 0)
			y++
		}
	}

	return y
}

func (a *Agent) renderCurrent(ctx *mofu.RenderContext, y int) int {
	r := ctx.Bounds
	if y >= r.Y+r.Height-4 {
		return y
	}

	lines := strings.Split(a.Current, "\n")
	for _, line := range lines {
		if y >= r.Y+r.Height-4 {
			break
		}
		display := line
		if len(display) > r.Width-2 {
			display = display[:r.Width-5] + "..."
		}
		ctx.Renderer.WriteString(" "+display, r.X, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
		y++
	}

	return y
}

func (a *Agent) renderCurrentTool(ctx *mofu.RenderContext, y int) int {
	r := ctx.Bounds
	if y >= r.Y+r.Height-4 || len(a.Steps) == 0 {
		return y
	}

	last := a.Steps[len(a.Steps)-1]
	ctx.Renderer.WriteString(fmt.Sprintf(" ⚡ %s (running...)", last.ToolName), r.X, y, mofu.Hex("f9e2af"), mofu.ColorBlack, mofu.AttrBold)
	y++

	if last.ToolInput != "" {
		input := last.ToolInput
		if len(input) > r.Width-4 {
			input = input[:r.Width-7] + "..."
		}
		ctx.Renderer.WriteString(" │ "+input, r.X, y, mofu.Hex("6c7086"), mofu.ColorBlack, 0)
		y++
	}

	return y
}

func (a *Agent) HandleEvent(event mofu.Event) mofu.Cmd {
	if event.Type != mofu.EventKeyPress {
		return nil
	}
	ke := event.Data.(mofu.KeyEvent)

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()
	case ke.Key == mofu.KeyCtrlC:
		if a.OnInterrupt != nil {
			a.OnInterrupt()
		}
		return mofu.QuitCmd()
	}
	return nil
}

// =========================================================================
// ToolCall — standalone tool call display
// =========================================================================

type ToolCall struct {
	mofu.Minimal
	Name    string
	Input   string
	Output  string
	Error   string
	Status  string
	Elapsed time.Duration
	mu      sync.RWMutex
}

func NewToolCall(name string) *ToolCall {
	return &ToolCall{Name: name, Status: "running"}
}

func (t *ToolCall) SetOutput(output string) {
	t.mu.Lock()
	t.Output = output
	t.Status = "done"
	t.mu.Unlock()
}

func (t *ToolCall) SetError(err string) {
	t.mu.Lock()
	t.Error = err
	t.Status = "error"
	t.mu.Unlock()
}

func (t *ToolCall) Render(ctx *mofu.RenderContext) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	icon := "⚡"
	color := mofu.Hex("89b4fa")
	switch t.Status {
	case "done":
		icon = "✓"
		color = mofu.Hex("a6e3a1")
	case "error":
		icon = "✗"
		color = mofu.Hex("f38ba8")
	case "running":
		icon = "●"
		color = mofu.Hex("f9e2af")
	}

	ctx.Renderer.WriteString(fmt.Sprintf(" %s %s", icon, t.Name), r.X, y, color, mofu.ColorBlack, mofu.AttrBold)
	y++

	if t.Input != "" {
		input := t.Input
		if len(input) > r.Width-4 {
			input = input[:r.Width-7] + "..."
		}
		ctx.Renderer.WriteString(" │ input: "+input, r.X, y, mofu.Hex("6c7086"), mofu.ColorBlack, 0)
		y++
	}

	if t.Output != "" {
		lines := strings.Split(t.Output, "\n")
		for _, line := range lines {
			if y >= r.Y+r.Height-1 {
				break
			}
			display := line
			if len(display) > r.Width-4 {
				display = display[:r.Width-7] + "..."
			}
			ctx.Renderer.WriteString(" │ "+display, r.X, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
			y++
		}
	}

	if t.Error != "" {
		ctx.Renderer.WriteString(" │ ERROR: "+t.Error, r.X, y, mofu.Hex("f38ba8"), mofu.ColorBlack, 0)
	}
}

func (t *ToolCall) HandleEvent(e mofu.Event) mofu.Cmd { return nil }

// =========================================================================
// TokenStream — streaming token display with typing effect
// =========================================================================

type TokenStream struct {
	mofu.Minimal
	Buffer     []string
	MaxLines   int
	Tokens     int
	Cost       float64
	Rendering  bool
	mu         sync.RWMutex
}

func NewTokenStream(maxLines int) *TokenStream {
	return &TokenStream{MaxLines: maxLines}
}

func (ts *TokenStream) Write(token string) {
	ts.mu.Lock()
	ts.Tokens++

	// Split token by newlines, appending to last line or creating new
	if len(ts.Buffer) == 0 {
		ts.Buffer = append(ts.Buffer, "")
	}

	for _, ch := range token {
		if ch == '\n' {
			ts.Buffer = append(ts.Buffer, "")
		} else {
			ts.Buffer[len(ts.Buffer)-1] += string(ch)
		}
	}

	if len(ts.Buffer) > ts.MaxLines {
		ts.Buffer = ts.Buffer[len(ts.Buffer)-ts.MaxLines:]
	}
	ts.mu.Unlock()
}

func (ts *TokenStream) Clear() {
	ts.mu.Lock()
	ts.Buffer = nil
	ts.Tokens = 0
	ts.Cost = 0
	ts.mu.Unlock()
}

func (ts *TokenStream) Render(ctx *mofu.RenderContext) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	r := ctx.Bounds
	start := 0
	if len(ts.Buffer) > r.Height {
		start = len(ts.Buffer) - r.Height
	}

	y := r.Y
	for i := start; i < len(ts.Buffer); i++ {
		if y >= r.Y+r.Height {
			break
		}
		line := ts.Buffer[i]
		if len(line) > r.Width-1 {
			line = line[:r.Width-4] + "..."
		}
		ctx.Renderer.WriteString(line, r.X, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
		y++
	}
}

func (ts *TokenStream) HandleEvent(e mofu.Event) mofu.Cmd { return nil }

// =========================================================================
// ToolPanel — side panel showing active tool calls
// =========================================================================

type ToolPanel struct {
	mofu.Minimal
	Calls    []ToolPanelEntry
	Selected int
	mu       sync.RWMutex
}

type ToolPanelEntry struct {
	Name     string
	Status   string
	Started  time.Time
	Input    string
	Output   string
}

func NewToolPanel() *ToolPanel {
	return &ToolPanel{}
}

func (tp *ToolPanel) Begin(name, input string) {
	tp.mu.Lock()
	tp.Calls = append(tp.Calls, ToolPanelEntry{
		Name:    name,
		Status:  "running",
		Started: time.Now(),
		Input:   input,
	})
	tp.mu.Unlock()
}

func (tp *ToolPanel) End(name, output string, err bool) {
	tp.mu.Lock()
	for i := len(tp.Calls) - 1; i >= 0; i-- {
		if tp.Calls[i].Name == name && tp.Calls[i].Status == "running" {
			tp.Calls[i].Output = output
			if err {
				tp.Calls[i].Status = "error"
			} else {
				tp.Calls[i].Status = "done"
			}
			break
		}
	}
	tp.mu.Unlock()
}

func (tp *ToolPanel) Render(ctx *mofu.RenderContext) {
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(" Tool Calls", r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	for i, call := range tp.Calls {
		if y >= r.Y+r.Height-1 {
			break
		}

		icon := "○"
		color := mofu.Hex("585b70")
		switch call.Status {
		case "done":
			icon = "✓"
			color = mofu.Hex("a6e3a1")
		case "running":
			icon = "●"
			color = mofu.Hex("f9e2af")
		case "error":
			icon = "✗"
			color = mofu.Hex("f38ba8")
		}

		elapsed := time.Since(call.Started).Round(time.Millisecond)
		name := call.Name
		if len(name) > r.Width-15 {
			name = name[:r.Width-18] + "..."
		}

		line := fmt.Sprintf(" %s %s %s", icon, name, elapsed)
		if len(line) > r.Width-2 {
			line = line[:r.Width-2]
		}

		if i == tp.Selected {
			ctx.Renderer.WriteString(strings.Repeat(" ", r.Width-2), r.X+1, y, mofu.ColorWhite, mofu.Hex("2a2a3e"), 0)
		}
		ctx.Renderer.WriteString(line, r.X, y, color, mofu.ColorBlack, 0)
		y++
	}
}

func (tp *ToolPanel) HandleEvent(e mofu.Event) mofu.Cmd { return nil }

// =========================================================================
// CostBar — token usage and cost display bar
// =========================================================================

type CostBar struct {
	mofu.Minimal
	TokensIn   int
	TokensOut  int
	CostIn     float64
	CostOut    float64
	MaxTokens  int
	mu         sync.RWMutex
}

func NewCostBar(maxTokens int) *CostBar {
	return &CostBar{MaxTokens: maxTokens}
}

func (cb *CostBar) AddTokens(in, out int, costIn, costOut float64) {
	cb.mu.Lock()
	cb.TokensIn += in
	cb.TokensOut += out
	cb.CostIn += costIn
	cb.CostOut += costOut
	cb.mu.Unlock()
}

func (cb *CostBar) Reset() {
	cb.mu.Lock()
	cb.TokensIn = 0
	cb.TokensOut = 0
	cb.CostIn = 0
	cb.CostOut = 0
	cb.mu.Unlock()
}

func (cb *CostBar) Render(ctx *mofu.RenderContext) {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	totalTokens := cb.TokensIn + cb.TokensOut
	totalCost := cb.CostIn + cb.CostOut

	// Context usage bar
	barW := r.Width - 30
	if barW < 10 {
		barW = 10
	}
	filled := 0
	if cb.MaxTokens > 0 {
		filled = totalTokens * barW / cb.MaxTokens
	}
	if filled > barW {
		filled = barW
	}

	barColor := mofu.Hex("a6e3a1")
	if filled*100/barW > 80 {
		barColor = mofu.Hex("fab387")
	}
	if filled*100/barW > 95 {
		barColor = mofu.Hex("f38ba8")
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barW-filled)
	ctx.Renderer.WriteString(fmt.Sprintf(" %s %d/%d", bar, totalTokens, cb.MaxTokens), r.X, y, barColor, mofu.ColorBlack, 0)
	y++

	// Cost line
	ctx.Renderer.WriteString(fmt.Sprintf(" in:%d ($%.4f)  out:%d ($%.4f)  total: $%.4f",
		cb.TokensIn, cb.CostIn, cb.TokensOut, cb.CostOut, totalCost), r.X, y, mofu.Hex("89b4fa"), mofu.ColorBlack, 0)
}

func (cb *CostBar) HandleEvent(e mofu.Event) mofu.Cmd { return nil }

// =========================================================================
// AgentLayout — multi-panel layout for agent workflows
// =========================================================================

type AgentLayout struct {
	mofu.Minimal
	Agent    *Agent
	Tools    *ToolPanel
	Costs    *CostBar
	Stream   *TokenStream
	LeftW    int
	RightW   int
	mu       sync.RWMutex
}

func NewAgentLayout(agent *Agent) *AgentLayout {
	return &AgentLayout{
		Agent:  agent,
		Tools:  NewToolPanel(),
		Costs:  NewCostBar(128000),
		Stream: NewTokenStream(200),
	}
}

func (al *AgentLayout) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	al.mu.RLock()
	defer al.mu.RUnlock()

	leftW := al.LeftW
	if leftW == 0 {
		leftW = r.Width * 2 / 3
	}
	rightW := r.Width - leftW
	if rightW < 20 {
		rightW = 20
		leftW = r.Width - rightW
	}

	// Left panel: Agent + Stream
	leftCtx := &mofu.RenderContext{
		Bounds:   mofu.Rect{X: r.X, Y: r.Y, Width: leftW, Height: r.Height - 2},
		Renderer: ctx.Renderer,
	}
	al.Agent.Render(leftCtx)

	// Right panel: Tools + Costs
	rightX := r.X + leftW
	rightCtx := &mofu.RenderContext{
		Bounds:   mofu.Rect{X: rightX, Y: r.Y, Width: rightW, Height: r.Height - 2},
		Renderer: ctx.Renderer,
	}
	al.Tools.Render(rightCtx)

	// Bottom bar: Costs
	costCtx := &mofu.RenderContext{
		Bounds:   mofu.Rect{X: r.X, Y: r.Y + r.Height - 2, Width: r.Width, Height: 2},
		Renderer: ctx.Renderer,
	}
	al.Costs.Render(costCtx)

	// Separator
	ctx.Renderer.WriteString("│", r.X+leftW-1, r.Y, mofu.Hex("444444"), mofu.ColorBlack, 0)
}

func (al *AgentLayout) HandleEvent(e mofu.Event) mofu.Cmd {
	return al.Agent.HandleEvent(e)
}
