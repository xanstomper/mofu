package gadgets

import (
	"fmt"
	"sync"
	"time"

	"github.com/xanstomper/mofu"
)

// ---------------------------------------------------------------------------
// AITrace — AI token stream visualizer
// ---------------------------------------------------------------------------

// TokenEvent represents an AI token event.
type TokenEvent struct {
	Timestamp time.Time
	Model     string
	Token     string
	Type      TokenType
	Cost      float64
	Latency   time.Duration
}

// TokenType classifies AI tokens.
type TokenType int

const (
	TokenText TokenType = iota
	TokenToolCall
	TokenToolResult
	TokenThinking
	TokenError
)

func (t TokenType) String() string {
	return [...]string{"text", "tool_call", "tool_result", "thinking", "error"}[t]
}

// AITrace visualizes AI token streams.
type AITrace struct {
	Base
	mu        sync.Mutex
	tokens    []TokenEvent
	maxTokens int
	totalCost float64
	totalTokens int
	models    map[string]int
}

// NewAITrace creates a new AI trace gadget.
func NewAITrace(id string) *AITrace {
	return &AITrace{
		Base:       *NewBase(id),
		maxTokens:  500,
		models:     make(map[string]int),
	}
}

// AddToken appends a token event.
func (at *AITrace) AddToken(event TokenEvent) {
	at.mu.Lock()
	defer at.mu.Unlock()

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	at.tokens = append(at.tokens, event)
	if len(at.tokens) > at.maxTokens {
		at.tokens = at.tokens[1:]
	}

	at.totalCost += event.Cost
	at.totalTokens++
	at.models[event.Model]++
}

// Stats returns summary statistics.
func (at *AITrace) Stats() AITraceStats {
	at.mu.Lock()
	defer at.mu.Unlock()

	return AITraceStats{
		TotalTokens: at.totalTokens,
		TotalCost:   at.totalCost,
		Models:      copyMap(at.models),
	}
}

// AITraceStats summarizes AI usage.
type AITraceStats struct {
	TotalTokens int
	TotalCost   float64
	Models      map[string]int
}

func copyMap(m map[string]int) map[string]int {
	out := make(map[string]int, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func (at *AITrace) Render(state StateView) []RenderNode {
	at.mu.Lock()
	defer at.mu.Unlock()

	var nodes []RenderNode

	header := fmt.Sprintf("AI Trace — %d tokens, $%.4f", at.totalTokens, at.totalCost)
	nodes = append(nodes, RenderNode{
		Type:    "text",
		Content: header,
		Style:   mofu.DefaultStyle().WithAttrs(mofu.AttrBold),
	})

	for model, count := range at.models {
		nodes = append(nodes, RenderNode{
			Type:    "text",
			Content: fmt.Sprintf("  %s: %d tokens", model, count),
			Style:   mofu.DefaultStyle().Fg(mofu.Hex("6c7086")),
		})
	}

	nodes = append(nodes, RenderNode{
		Type:    "text",
		Content: "---",
		Style:   mofu.DefaultStyle().Fg(mofu.Hex("45475a")),
	})

	visible := 10
	start := len(at.tokens) - visible
	if start < 0 {
		start = 0
	}

	for i := start; i < len(at.tokens); i++ {
		t := at.tokens[i]
		ts := t.Timestamp.Format("15:04:05")
		line := fmt.Sprintf("%s [%s] %q", ts, t.Type, t.Token)

		color := mofu.Hex("cdd6f4")
		switch t.Type {
		case TokenToolCall:
			color = mofu.Hex("89b4fa")
		case TokenToolResult:
			color = mofu.Hex("a6e3a1")
		case TokenThinking:
			color = mofu.Hex("6c7086")
		case TokenError:
			color = mofu.Hex("f38ba8")
		}

		nodes = append(nodes, RenderNode{
			Type:    "text",
			Content: line,
			Style:   mofu.DefaultStyle().Fg(color),
		})
	}

	return nodes
}

// ---------------------------------------------------------------------------
// Timeline — event timeline with zoom/pan
// ---------------------------------------------------------------------------

// TimelineEvent is an event on the timeline.
type TimelineEvent struct {
	Timestamp time.Time
	Label     string
	Category  string
	Color     mofu.Color
}

// Timeline visualizes events over time.
type Timeline struct {
	Base
	mu        sync.Mutex
	events    []TimelineEvent
	maxEvents int
	window    time.Duration
}

// NewTimeline creates a new timeline gadget.
func NewTimeline(id string) *Timeline {
	return &Timeline{
		Base:      *NewBase(id),
		maxEvents: 200,
		window:    5 * time.Minute,
	}
}

// AddEvent adds an event to the timeline.
func (tl *Timeline) AddEvent(event TimelineEvent) {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	if event.Color == (mofu.Color{}) {
		event.Color = mofu.Hex("89b4fa")
	}

	tl.events = append(tl.events, event)
	if len(tl.events) > tl.maxEvents {
		tl.events = tl.events[1:]
	}
}

// SetWindow sets the visible time window.
func (tl *Timeline) SetWindow(d time.Duration) {
	tl.mu.Lock()
	tl.window = d
	tl.mu.Unlock()
}

func (tl *Timeline) Render(state StateView) []RenderNode {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	var nodes []RenderNode
	now := time.Now()
	cutoff := now.Add(-tl.window)

	var visible []TimelineEvent
	for _, e := range tl.events {
		if e.Timestamp.After(cutoff) {
			visible = append(visible, e)
		}
	}

	header := fmt.Sprintf("Timeline (%d events in %s)", len(visible), tl.window)
	nodes = append(nodes, RenderNode{
		Type:    "text",
		Content: header,
		Style:   mofu.DefaultStyle().WithAttrs(mofu.AttrBold),
	})

	show := 15
	start := len(visible) - show
	if start < 0 {
		start = 0
	}

	for i := start; i < len(visible); i++ {
		e := visible[i]
		ts := e.Timestamp.Format("15:04:05")
		cat := ""
		if e.Category != "" {
			cat = fmt.Sprintf("[%s] ", e.Category)
		}
		line := fmt.Sprintf("  %s %s%s", ts, cat, e.Label)
		nodes = append(nodes, RenderNode{
			Type:    "text",
			Content: line,
			Style:   mofu.DefaultStyle().Fg(e.Color),
		})
	}

	return nodes
}
