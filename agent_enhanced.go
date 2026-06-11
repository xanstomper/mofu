package mofu

import (
	"fmt"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Tool Call Graph — visualize tool call chains
// ---------------------------------------------------------------------------

// ToolCall represents a single tool invocation.
type ToolCall struct {
	ID         string
	Name       string
	Params     map[string]any
	Result     any
	Error      string
	StartTime  time.Time
	EndTime    time.Time
	ParentID   string // for nested tool calls
	Children   []string
	Status     ToolCallStatus
	TokenCount int
}

// ToolCallStatus tracks tool call lifecycle.
type ToolCallStatus int

const (
	ToolCallPending ToolCallStatus = iota
	ToolCallRunning
	ToolCallSuccess
	ToolCallFailed
)

func (s ToolCallStatus) String() string {
	return [...]string{"pending", "running", "success", "failed"}[s]
}

// ToolCallGraph tracks a tree of tool calls.
type ToolCallGraph struct {
	mu    sync.Mutex
	calls map[string]*ToolCall
	order []string
	root  string
}

// NewToolCallGraph creates an empty tool call graph.
func NewToolCallGraph() *ToolCallGraph {
	return &ToolCallGraph{
		calls: make(map[string]*ToolCall),
	}
}

// Start records a new tool call.
func (g *ToolCallGraph) Start(id, name string, params map[string]any, parentID string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	call := &ToolCall{
		ID:        id,
		Name:      name,
		Params:    params,
		ParentID:  parentID,
		StartTime: time.Now(),
		Status:    ToolCallRunning,
	}
	g.calls[id] = call
	g.order = append(g.order, id)

	if parentID == "" {
		g.root = id
	} else if parent, ok := g.calls[parentID]; ok {
		parent.Children = append(parent.Children, id)
	}
}

// Complete marks a tool call as successful.
func (g *ToolCallGraph) Complete(id string, result any, tokenCount int) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if call, ok := g.calls[id]; ok {
		call.Result = result
		call.EndTime = time.Now()
		call.Status = ToolCallSuccess
		call.TokenCount = tokenCount
	}
}

// Fail marks a tool call as failed.
func (g *ToolCallGraph) Fail(id string, err string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if call, ok := g.calls[id]; ok {
		call.Error = err
		call.EndTime = time.Now()
		call.Status = ToolCallFailed
	}
}

// Get returns a tool call by ID.
func (g *ToolCallGraph) Get(id string) *ToolCall {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.calls[id]
}

// All returns all tool calls in order.
func (g *ToolCallGraph) All() []*ToolCall {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := make([]*ToolCall, 0, len(g.order))
	for _, id := range g.order {
		out = append(out, g.calls[id])
	}
	return out
}

// Stats returns aggregate statistics.
func (g *ToolCallGraph) Stats() ToolCallStats {
	g.mu.Lock()
	defer g.mu.Unlock()

	stats := ToolCallStats{}
	for _, call := range g.calls {
		stats.Total++
		stats.TotalTokens += call.TokenCount
		if !call.EndTime.IsZero() && !call.StartTime.IsZero() {
			d := call.EndTime.Sub(call.StartTime)
			stats.TotalDuration += d
			if d > stats.MaxDuration {
				stats.MaxDuration = d
			}
		}
		switch call.Status {
		case ToolCallSuccess:
			stats.Succeeded++
		case ToolCallFailed:
			stats.Failed++
		case ToolCallRunning:
			stats.Running++
		}
		stats.ByTool[call.Name]++
	}
	if stats.Total > 0 {
		stats.AvgDuration = stats.TotalDuration / time.Duration(stats.Total)
	}
	return stats
}

// ToolCallStats aggregates tool call statistics.
type ToolCallStats struct {
	Total         int
	Succeeded     int
	Failed        int
	Running       int
	TotalTokens   int
	TotalDuration time.Duration
	AvgDuration   time.Duration
	MaxDuration   time.Duration
	ByTool        map[string]int
}

// RenderTree returns a text representation of the tool call tree.
func (g *ToolCallGraph) RenderTree() []string {
	g.mu.Lock()
	defer g.mu.Unlock()

	var lines []string
	var render func(id string, prefix string, isLast bool)
	render = func(id string, prefix string, isLast bool) {
		call, ok := g.calls[id]
		if !ok {
			return
		}

		connector := "├─"
		if isLast {
			connector = "└─"
		}

		duration := ""
		if !call.EndTime.IsZero() {
			duration = fmt.Sprintf(" (%s)", call.EndTime.Sub(call.StartTime).Round(time.Millisecond))
		}

		status := call.Status.String()
		if call.Error != "" {
			status = "error: " + call.Error
		}

		lines = append(lines, fmt.Sprintf("%s%s %s [%s]%s", prefix, connector, call.Name, status, duration))

		childPrefix := prefix + "│ "
		if isLast {
			childPrefix = prefix + "  "
		}

		for i, childID := range call.Children {
			render(childID, childPrefix, i == len(call.Children)-1)
		}
	}

	if g.root != "" {
		lines = append(lines, g.calls[g.root].Name)
		for i, childID := range g.calls[g.root].Children {
			render(childID, "", i == len(g.calls[g.root].Children)-1)
		}
	}

	return lines
}

// ---------------------------------------------------------------------------
// Token Counter — track token usage across models
// ---------------------------------------------------------------------------

// TokenUsage tracks token consumption.
type TokenUsage struct {
	mu           sync.Mutex
	prompt       int
	completion   int
	total        int
	byModel      map[string]*ModelUsage
	sessionStart time.Time
}

// ModelUsage tracks per-model token usage.
type ModelUsage struct {
	Prompt     int
	Completion int
	Total      int
	Cost       float64
	Requests   int
}

// NewTokenUsage creates a new token counter.
func NewTokenUsage() *TokenUsage {
	return &TokenUsage{
		byModel:      make(map[string]*ModelUsage),
		sessionStart: time.Now(),
	}
}

// Record records token usage for a model.
func (tu *TokenUsage) Record(model string, prompt, completion int, cost float64) {
	tu.mu.Lock()
	defer tu.mu.Unlock()

	tu.prompt += prompt
	tu.completion += completion
	tu.total += prompt + completion

	m, ok := tu.byModel[model]
	if !ok {
		m = &ModelUsage{}
		tu.byModel[model] = m
	}
	m.Prompt += prompt
	m.Completion += completion
	m.Total += prompt + completion
	m.Cost += cost
	m.Requests++
}

// Total returns total tokens used.
func (tu *TokenUsage) Total() int {
	tu.mu.Lock()
	defer tu.mu.Unlock()
	return tu.total
}

// ByModel returns per-model usage.
func (tu *TokenUsage) ByModel() map[string]*ModelUsage {
	tu.mu.Lock()
	defer tu.mu.Unlock()
	out := make(map[string]*ModelUsage, len(tu.byModel))
	for k, v := range tu.byModel {
		cp := *v
		out[k] = &cp
	}
	return out
}

// TotalCost returns the total cost across all models.
func (tu *TokenUsage) TotalCost() float64 {
	tu.mu.Lock()
	defer tu.mu.Unlock()
	var cost float64
	for _, m := range tu.byModel {
		cost += m.Cost
	}
	return cost
}

// Duration returns the session duration.
func (tu *TokenUsage) Duration() time.Duration {
	tu.mu.Lock()
	defer tu.mu.Unlock()
	return time.Since(tu.sessionStart)
}

// TokensPerSecond returns the average token rate.
func (tu *TokenUsage) TokensPerSecond() float64 {
	d := tu.Duration()
	if d == 0 {
		return 0
	}
	return float64(tu.Total()) / d.Seconds()
}

// RenderSummary returns a text summary of token usage.
func (tu *TokenUsage) RenderSummary() []string {
	tu.mu.Lock()
	defer tu.mu.Unlock()

	var lines []string
	lines = append(lines, fmt.Sprintf("Tokens: %d (prompt: %d, completion: %d)", tu.total, tu.prompt, tu.completion))
	lines = append(lines, fmt.Sprintf("Cost: $%.4f", tu.TotalCost()))
	lines = append(lines, fmt.Sprintf("Duration: %s", tu.Duration().Round(time.Second)))

	for model, m := range tu.byModel {
		lines = append(lines, fmt.Sprintf("  %s: %d tokens, $%.4f (%d requests)", model, m.Total, m.Cost, m.Requests))
	}

	return lines
}
