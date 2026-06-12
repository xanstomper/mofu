package mofu

import (
	"fmt"
	"sync"
	"time"
)

type SpinnerStyle struct {
	Frames []string
	Fg     Color
	Bg     Color
	Attrs  AttrMask
}

var (
	SpinnerDot = SpinnerStyle{
		Frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		Fg:     Hex("89b4fa"),
	}
	SpinnerLine = SpinnerStyle{
		Frames: []string{"|", "/", "—", "\\"},
		Fg:     Hex("a6e3a1"),
	}
	SpinnerDot2 = SpinnerStyle{
		Frames: []string{"● ○", "○ ●"},
		Fg:     Hex("f5c2e7"),
	}
	SpinnerMinidot = SpinnerStyle{
		Frames: []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"},
		Fg:     Hex("f9e2af"),
	}
	SpinnerPulse = SpinnerStyle{
		Frames: []string{"◐", "◓", "◑", "◒"},
		Fg:     Hex("7dcfff"),
	}
	SpinnerGlobe = SpinnerStyle{
		Frames: []string{"🌍", "🌎", "🌏"},
		Fg:     Hex("a6e3a1"),
	}
	SpinnerMonkey = SpinnerStyle{
		Frames: []string{"🙈", "🙉", "🙊"},
		Fg:     Hex("f5c2e7"),
	}
	SpinnerPoints = SpinnerStyle{
		Frames: []string{"·   ", "··  ", "··· ", "····", " ···", "  ··", "   ·", "    "},
		Fg:     Hex("89b4fa"),
	}
)

type Spinner struct {
	mu         sync.Mutex
	style      SpinnerStyle
	index      int
	ticker     *time.Ticker
	updateCh   chan struct{}
	title      string
	startTime  time.Time
	paused     bool
	running    bool
	frame      int64
	onUpdate   func()
}

func NewSpinner(style SpinnerStyle) *Spinner {
	return &Spinner{
		style:    style,
		updateCh: make(chan struct{}, 1),
		startTime: time.Now(),
	}
}

func (s *Spinner) Title(title string) {
	s.mu.Lock()
	s.title = title
	s.mu.Unlock()
}

func (s *Spinner) Style(style SpinnerStyle) {
	s.mu.Lock()
	s.style = style
	s.mu.Unlock()
}

func (s *Spinner) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.startTime = time.Now()
	s.ticker = time.NewTicker(80 * time.Millisecond)
	s.mu.Unlock()
	go s.spin()
}

func (s *Spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ticker != nil {
		s.ticker.Stop()
		s.ticker = nil
	}
	s.running = false
}

func (s *Spinner) Pause() {
	s.mu.Lock()
	s.paused = true
	s.mu.Unlock()
}

func (s *Spinner) Resume() {
	s.mu.Lock()
	s.paused = false
	s.mu.Unlock()
}

func (s *Spinner) spin() {
	for {
		s.mu.Lock()
		if !s.running {
			s.mu.Unlock()
			return
		}
		ticker := s.ticker
		paused := s.paused
		s.mu.Unlock()

		if paused {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if ticker == nil {
			return
		}
		<-ticker.C
		s.mu.Lock()
		s.index = (s.index + 1) % len(s.style.Frames)
		s.frame++
		s.mu.Unlock()
		if s.onUpdate != nil {
			s.onUpdate()
		}
	}
}

func (s *Spinner) Elapsed() time.Duration {
	return time.Since(s.startTime)
}

func (s *Spinner) Frame() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.frame
}

func (s *Spinner) Render(w int) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.style.Frames) == 0 {
		return ""
	}
	frame := s.style.Frames[s.index%s.len()]
	title := s.title
	if title == "" {
		title = "Working"
	}
	label := fmt.Sprintf(" %s %s", frame, title)
	return DefaultStyle().Fg(s.style.Fg).Bg(s.style.Bg).WithAttrs(s.style.Attrs).Apply(label)
}

func (s *Spinner) len() int {
	if len(s.style.Frames) == 0 {
		return 1
	}
	return len(s.style.Frames)
}
