package mofu

import (
	"fmt"
	"math"
	"strings"
	"sync"
)

type ProgressMode int

const (
	ProgressBar ProgressMode = iota
	ProgressDots
	ProgressSpinner
	ProgressPercent
)

type ProgressStyle struct {
	Empty    rune
	Full     rune
	Head     rune
	Mode     ProgressMode
	Filled   Color
	Unfilled Color
	ShowPct  bool
}

func DefaultProgressStyle() ProgressStyle {
	return ProgressStyle{
		Empty:    '░',
		Full:     '█',
		Head:     '█',
		Mode:     ProgressBar,
		Filled:   Hex("89b4fa"),
		Unfilled: Hex("313244"),
		ShowPct:  true,
	}
}

type Progress struct {
	mu       sync.Mutex
	total    float64
	current  float64
	width    int
	style    ProgressStyle
	title    string
	finished bool
}

func NewProgress(total float64, width int) *Progress {
	return &Progress{
		total: total,
		width: width,
		style: DefaultProgressStyle(),
	}
}

func (p *Progress) SetStyle(s ProgressStyle) {
	p.mu.Lock()
	p.style = s
	p.mu.Unlock()
}

func (p *Progress) Title(t string) {
	p.mu.Lock()
	p.title = t
	p.mu.Unlock()
}

func (p *Progress) Set(v float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if v > p.total {
		v = p.total
	}
	if v < 0 {
		v = 0
	}
	p.current = v
	if v >= p.total {
		p.finished = true
	}
}

func (p *Progress) Incr(v float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.current += v
	if p.current > p.total {
		p.current = p.total
	}
	if p.current >= p.total {
		p.finished = true
	}
}

func (p *Progress) Percent() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.total == 0 {
		return 0
	}
	return math.Min(100, (p.current/p.total)*100)
}

func (p *Progress) IsFinished() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.finished
}

func (p *Progress) Render() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch p.style.Mode {
	case ProgressDots:
		return p.renderDots()
	case ProgressSpinner:
		return p.renderSpinner()
	case ProgressPercent:
		return p.renderPercent()
	default:
		return p.renderBar()
	}
}

func (p *Progress) renderBar() string {
	pct := p.Percent()
	filled := int(pct / 100 * float64(p.width))
	if filled > p.width {
		filled = p.width
	}

	bar := DefaultStyle().Fg(p.style.Filled).Apply(strings.Repeat(string(p.style.Full), filled))
	empty := DefaultStyle().Fg(p.style.Unfilled).Apply(strings.Repeat(string(p.style.Empty), p.width-filled))

	label := fmt.Sprintf(" %s%.0f%%", bar+empty, pct)
	if p.title != "" {
		label = p.title + " " + label
	}
	return label
}

func (p *Progress) renderDots() string {
	pct := p.Percent()
	totalDots := p.width
	filledDots := int(pct / 100 * float64(totalDots))
	if filledDots > totalDots {
		filledDots = totalDots
	}

	dots := DefaultStyle().Fg(p.style.Filled).Apply(strings.Repeat("●", filledDots))
	empty := DefaultStyle().Fg(p.style.Unfilled).Apply(strings.Repeat("○", totalDots-filledDots))

	return fmt.Sprintf("%s%s %.0f%%", dots, empty, pct)
}

func (p *Progress) renderSpinner() string {
	if p.finished {
		return DefaultStyle().Fg(Hex("a6e3a1")).Apply(" ✓ Done")
	}
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	idx := int(p.current) % len(frames)
	return DefaultStyle().Fg(p.style.Filled).Apply(frames[idx]) + fmt.Sprintf(" %.0f%%", p.Percent())
}

func (p *Progress) renderPercent() string {
	pct := p.Percent()
	return fmt.Sprintf("%.1f%%", pct)
}
