package mofu

import (
	"fmt"
	"sync"
	"time"
)

// ProgramState represents the lifecycle state of a Program.
type ProgramState int

const (
	StateInit ProgramState = iota
	StateMounting
	StateRunning
	StatePaused
	StateUnmounting
	StateDone
	StateFailed
)

func (s ProgramState) String() string {
	switch s {
	case StateInit:
		return "init"
	case StateMounting:
		return "mounting"
	case StateRunning:
		return "running"
	case StatePaused:
		return "paused"
	case StateUnmounting:
		return "unmounting"
	case StateDone:
		return "done"
	case StateFailed:
		return "failed"
	default:
		return fmt.Sprintf("unknown(%d)", s)
	}
}

type StateTransition struct {
	From      ProgramState
	To        ProgramState
	Timestamp time.Time
	Reason    string
}

type StateMachine struct {
	state    ProgramState
	history  []StateTransition
	maxHist  int
	mu       sync.Mutex
	onChange []func(ProgramState, ProgramState)
}

func NewStateMachine(initial ProgramState) *StateMachine {
	return &StateMachine{
		state:   initial,
		maxHist: 256,
		history: []StateTransition{{From: StateInit, To: initial, Timestamp: time.Now(), Reason: "initial"}},
	}
}

func (sm *StateMachine) State() ProgramState {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.state
}

func (sm *StateMachine) Transition(to ProgramState, reason string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if sm.state == to {
		return false
	}
	from := sm.state
	sm.state = to
	sm.history = append(sm.history, StateTransition{From: from, To: to, Timestamp: time.Now(), Reason: reason})
	if len(sm.history) > sm.maxHist {
		sm.history = sm.history[1:]
	}
	for _, fn := range sm.onChange {
		go fn(from, to)
	}
	return true
}

func (sm *StateMachine) History() []StateTransition {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	out := make([]StateTransition, len(sm.history))
	copy(out, sm.history)
	return out
}

func (sm *StateMachine) OnChange(fn func(from, to ProgramState)) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.onChange = append(sm.onChange, fn)
}

// TimeMachine stores state snapshots for rewind / audit.
type TimeMachine struct {
	snapshots []ProgramSnapshot
	maxSize   int
	mu        sync.Mutex
}

type ProgramSnapshot struct {
	Version   int            `json:"version"`
	Timestamp time.Time      `json:"timestamp"`
	State     ProgramState   `json:"state"`
	Meta      map[string]any `json:"meta,omitempty"`
}

func NewTimeMachine(maxSize int) *TimeMachine {
	if maxSize <= 0 {
		maxSize = 128
	}
	return &TimeMachine{maxSize: maxSize}
}

func (tm *TimeMachine) Record(s ProgramSnapshot) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	s.Version = 1
	s.Timestamp = time.Now()
	tm.snapshots = append(tm.snapshots, s)
	for len(tm.snapshots) > tm.maxSize {
		tm.snapshots = tm.snapshots[1:]
	}
}

func (tm *TimeMachine) Snapshots() []ProgramSnapshot {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	out := make([]ProgramSnapshot, len(tm.snapshots))
	copy(out, tm.snapshots)
	return out
}

func (tm *TimeMachine) Last() (ProgramSnapshot, bool) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if len(tm.snapshots) == 0 {
		return ProgramSnapshot{}, false
	}
	return tm.snapshots[len(tm.snapshots)-1], true
}
