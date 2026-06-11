package mofu

import (
	"fmt"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// AI Agent Visualization (Anthology Ch.12)
// ---------------------------------------------------------------------------

// AgentStateKind identifies the high-level state of an AI agent.
type AgentStateKind uint8

const (
	AgentIdle AgentStateKind = iota
	AgentInitializing
	AgentThinking
	AgentExecuting
	AgentAwaitingInput
	AgentError
	AgentSuccess
	AgentStreaming
)

// AgentState represents an agent state machine snapshot.
type AgentState struct {
	Kind               AgentStateKind
	Progress           float64
	Step               string
	Prompt             string
	ElapsedMs          uint64
	Confidence         float64
	Command            string
	ExecStatus         ExecStatus
	Logs               []string
	Options            []string
	Error              string
	Retryable          bool
	StackTrace         string
	TokensReceived     int
	EstimatedTotal     int
	Metadata           map[string]string
	TransitionHistory  []AgentTransition
	LastTransitionTime time.Time
}

// ExecStatus describes command execution status.
type ExecStatus uint8

const (
	ExecPending ExecStatus = iota
	ExecRunning
	ExecDone
	ExecFailed
	ExecCancelled
)

// AgentTransition records one state transition.
type AgentTransition struct {
	From AgentState
	To   AgentState
	At   time.Time
}

// AgentStateMachine owns transitions, history, and validation.
type AgentStateMachine struct {
	mu              sync.Mutex
	State           AgentState
	MaxHistory      int
	OnTransition    func(old, next AgentState)
	transitionCount uint64
}

// NewAgentStateMachine returns an idle state machine.
func NewAgentStateMachine() *AgentStateMachine {
	return &AgentStateMachine{State: AgentState{Kind: AgentIdle}, MaxHistory: 256}
}

// Transition changes the current state after validating allowed transitions.
func (m *AgentStateMachine) Transition(next AgentState) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.CanTransitionTo(next.Kind) {
		return false
	}
	old := m.State
	next.TransitionHistory = append([]AgentTransition(nil), old.TransitionHistory...)
	if len(next.TransitionHistory) >= m.MaxHistory {
		next.TransitionHistory = next.TransitionHistory[len(next.TransitionHistory)-m.MaxHistory:]
	}
	next.TransitionHistory = append(next.TransitionHistory, AgentTransition{From: old, To: next, At: time.Now()})
	next.LastTransitionTime = time.Now()
	m.transitionCount++
	m.State = next
	if m.OnTransition != nil {
		m.OnTransition(old, next)
	}
	return true
}

// CanTransitionTo returns whether target is reachable from the current state.
func (m *AgentStateMachine) CanTransitionTo(target AgentStateKind) bool {
	switch m.State.Kind {
	case AgentIdle:
		return target == AgentInitializing
	case AgentInitializing:
		return target == AgentThinking
	case AgentThinking:
		return target == AgentExecuting || target == AgentAwaitingInput || target == AgentStreaming
	case AgentExecuting:
		return target == AgentAwaitingInput || target == AgentError || target == AgentSuccess || target == AgentStreaming
	case AgentAwaitingInput:
		return target == AgentThinking || target == AgentExecuting
	case AgentError:
		return target == AgentIdle || target == AgentThinking || target == AgentExecuting
	case AgentSuccess:
		return target == AgentIdle
	case AgentStreaming:
		return target == AgentSuccess || target == AgentError || target == AgentThinking
	default:
		return target == AgentError
	}
}

// DurationInCurrentState returns elapsed time since the last transition.
func (m *AgentStateMachine) DurationInCurrentState() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.State.LastTransitionTime.IsZero() {
		return 0
	}
	return time.Since(m.State.LastTransitionTime)
}

// Snapshot returns a copy of the current state.
func (m *AgentStateMachine) Snapshot() AgentState {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.State
}

// ---------------------------------------------------------------------------
// Behavior logging
// ---------------------------------------------------------------------------

// BehaviorEventKind identifies an agent behavior event.
type BehaviorEventKind uint8

const (
	BehaviorPromptSubmitted BehaviorEventKind = iota
	BehaviorToolUsed
	BehaviorTokenGenerated
	BehaviorStateTransition
	BehaviorErrorEncountered
	BehaviorUserFeedback
	BehaviorResourceUsed
)

// FeedbackCategory classifies user feedback.
type FeedbackCategory uint8

const (
	FeedbackPositive FeedbackCategory = iota
	FeedbackNegative
	FeedbackCorrection
	FeedbackInterrupt
)

// Sentiment captures simple sentiment polarity.
type Sentiment uint8

const (
	SentimentPositive Sentiment = iota
	SentimentNeutral
	SentimentNegative
)

// ResourceType identifies tracked resource categories.
type ResourceType uint8

const (
	ResourceTokens ResourceType = iota
	ResourceTime
	ResourceMemory
	ResourceNetwork
)

// BehaviorEvent records an agent action or observation.
type BehaviorEvent struct {
	Kind        BehaviorEventKind
	Timestamp   time.Time
	Text        string
	Tool        string
	Params      map[string]string
	DurationMs  uint64
	Token       string
	Complete    bool
	Model       string
	FromState   string
	ToState     string
	Trigger     string
	Error       string
	Recoverable bool
	Category    FeedbackCategory
	Sentiment   Sentiment
	Resource    ResourceType
	Amount      uint64
}

// BehaviorLogger stores append-only behavior events in memory.
type BehaviorLogger struct {
	mu      sync.Mutex
	events  []BehaviorEvent
	enabled bool
}

// NewBehaviorLogger returns a disabled logger by default.
func NewBehaviorLogger() *BehaviorLogger { return &BehaviorLogger{} }

// Enable turns event recording on.
func (bl *BehaviorLogger) Enable() { bl.mu.Lock(); bl.enabled = true; bl.mu.Unlock() }

// Disable turns event recording off.
func (bl *BehaviorLogger) Disable() { bl.mu.Lock(); bl.enabled = false; bl.mu.Unlock() }

// Record appends an event if enabled.
func (bl *BehaviorLogger) Record(event BehaviorEvent) {
	bl.mu.Lock()
	defer bl.mu.Unlock()
	if !bl.enabled {
		return
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	bl.events = append(bl.events, event)
}

// Export returns a copy of recorded events.
func (bl *BehaviorLogger) Export() []BehaviorEvent {
	bl.mu.Lock()
	defer bl.mu.Unlock()
	out := make([]BehaviorEvent, len(bl.events))
	copy(out, bl.events)
	return out
}

// Analyze returns aggregate behavior statistics.
func (bl *BehaviorLogger) Analyze() BehaviorSummary {
	events := bl.Export()
	s := BehaviorSummary{Events: len(events)}
	for _, e := range events {
		switch e.Kind {
		case BehaviorPromptSubmitted:
			s.TotalPrompts++
		case BehaviorErrorEncountered:
			s.TotalErrors++
		case BehaviorToolUsed:
			s.ToolUses++
		case BehaviorTokenGenerated:
			s.TokensGenerated += 1
		case BehaviorResourceUsed:
			s.ResourcesUsed[e.Resource] += e.Amount
		}
		if e.DurationMs > 0 {
			s.TotalDurationMs += e.DurationMs
		}
	}
	if s.Events > 0 {
		s.AvgDurationMs = s.TotalDurationMs / uint64(s.Events)
	}
	return s
}

// BehaviorSummary is the aggregate of a behavior log.
type BehaviorSummary struct {
	Events          int
	TotalPrompts    int
	TotalErrors     int
	ToolUses        int
	TokensGenerated int
	TotalDurationMs uint64
	AvgDurationMs   uint64
	ResourcesUsed   map[ResourceType]uint64
}

// ---------------------------------------------------------------------------
// Agent analytics dashboard model
// ---------------------------------------------------------------------------

// AgentSession captures one agent session.
type AgentSession struct {
	ID          string
	StartTime   time.Time
	EndTime     time.Time
	PromptCount int
	TokenCount  int
	ErrorCount  int
}

// AgentMetric stores a named metric.
type AgentMetric struct {
	Name  string
	Value float64
	Unit  string
}

// AgentAnalytics aggregates sessions and metrics.
type AgentAnalytics struct {
	Sessions []AgentSession
	Metrics  map[string]AgentMetric
}

// AgentVisualizer renders a compact textual tree for an agent state.
type AgentVisualizer struct{}

// Render returns a simple text tree for the given state.
func (AgentVisualizer) Render(state AgentState) []string {
	out := []string{"Agent", "├─ state: " + agentStateName(state.Kind)}
	if state.Step != "" {
		out = append(out, "├─ step: "+state.Step)
	}
	if state.Progress > 0 {
		out = append(out, "├─ progress: "+formatPercent(state.Progress))
	}
	if state.Command != "" {
		out = append(out, "├─ command: "+state.Command)
	}
	if state.Error != "" {
		out = append(out, "└─ error: "+state.Error)
	}
	return out
}

func agentStateName(kind AgentStateKind) string {
	switch kind {
	case AgentIdle:
		return "idle"
	case AgentInitializing:
		return "initializing"
	case AgentThinking:
		return "thinking"
	case AgentExecuting:
		return "executing"
	case AgentAwaitingInput:
		return "awaiting-input"
	case AgentError:
		return "error"
	case AgentSuccess:
		return "success"
	case AgentStreaming:
		return "streaming"
	default:
		return "unknown"
	}
}

func formatPercent(v float64) string {
	if v > 1 {
		v = v / 100
	}
	return fmt.Sprintf("%.1f%%", v*100)
}
