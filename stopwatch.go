package mofu

import (
	"sync"
	"sync/atomic"
	"time"
)

var lastStopwatchID int64

func nextStopwatchID() int {
	return int(atomic.AddInt64(&lastStopwatchID, 1))
}

type StopwatchMsg struct {
	ID      int
_elapsed time.Duration
}

type Stopwatch struct {
	mu        sync.Mutex
	id        int
	startTime time.Time
	paused    bool
	elapsed   time.Duration
	interval  time.Duration
	running   bool
}

func NewStopwatch(interval time.Duration) Stopwatch {
	return Stopwatch{
		id:       nextStopwatchID(),
		interval: interval,
		running:  true,
	}
}

func (s *Stopwatch) ID() int {
	return s.id
}

func (s *Stopwatch) Running() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

func (s *Stopwatch) Elapsed() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return time.Since(s.startTime)
	}
	return s.elapsed
}

func (s *Stopwatch) Start() Cmd {
	return func() Msg {
		return StartStopMsg{id: s.id, running: true}
	}
}

func (s *Stopwatch) Stop() Cmd {
	return func() Msg {
		return StartStopMsg{id: s.id, running: false}
	}
}

func (s *Stopwatch) Toggle() Cmd {
	s.mu.Lock()
	v := !s.running
	s.mu.Unlock()
	return func() Msg {
		return StartStopMsg{id: s.id, running: v}
	}
}

type StartStopMsg struct {
	id      int
	running bool
}

type ExecMsg struct {
	Cmd string
	Fn  func(string, error) Msg
}

func Exec(name string, args ...string) Cmd {
	return func() Msg {
		return ExecMsg{Cmd: name}
	}
}
