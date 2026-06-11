package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/xanstomper/mofu"
)

type Pomodoro struct {
	mofu.Minimal
	mode        string
	elapsed     time.Duration
	total       time.Duration
	running     bool
	sessions    int
	breaks      int
	longBreaks  int
	currentTask string
	width       int
	height      int
	lastTick    time.Time
}

func NewPomodoro() *Pomodoro {
	return &Pomodoro{
		mode:    "work",
		total:   25 * time.Minute,
		running: false,
	}
}

func (p *Pomodoro) tick() {
	if !p.running {
		return
	}

	now := time.Now()
	if p.lastTick.IsZero() {
		p.lastTick = now
		return
	}

	p.elapsed += now.Sub(p.lastTick)
	p.lastTick = now

	if p.elapsed >= p.total {
		p.running = false
		switch p.mode {
		case "work":
			p.sessions++
			if p.sessions%4 == 0 {
				p.mode = "longbreak"
				p.total = 15 * time.Minute
				p.longBreaks++
			} else {
				p.mode = "break"
				p.total = 5 * time.Minute
				p.breaks++
			}
		case "break", "longbreak":
			p.mode = "work"
			p.total = 25 * time.Minute
		}
		p.elapsed = 0
	}
}

func (p *Pomodoro) Render(ctx *mofu.RenderContext) {
	p.tick()
	r := ctx.Bounds
	p.width = r.Width
	p.height = r.Height

	y := r.Y

	// Title
	titleStyle := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
	ctx.Renderer.WriteString(" Pomodoro Timer", r.X, y, titleStyle.Foreground, titleStyle.Background, titleStyle.Attrs)
	y++

	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	// Mode indicator
	modeColor := mofu.Hex("a6e3a1")
	modeLabel := "WORK"
	switch p.mode {
	case "break":
		modeColor = mofu.Hex("89b4fa")
		modeLabel = "BREAK"
	case "longbreak":
		modeColor = mofu.Hex("cba6f7")
		modeLabel = "LONG BREAK"
	}

	ctx.Renderer.WriteString(fmt.Sprintf(" [%s] ", modeLabel), r.X+r.Width/2-7, y, modeColor, mofu.ColorBlack, mofu.AttrBold)
	y += 2

	// Big time display
	remaining := p.total - p.elapsed
	if remaining < 0 {
		remaining = 0
	}
	minutes := int(remaining.Minutes())
	seconds := int(remaining.Seconds()) % 60
	timeStr := fmt.Sprintf("%02d:%02d", minutes, seconds)

	timeStyle := mofu.DefaultStyle().Fg(modeColor).WithAttrs(mofu.AttrBold)
	ctx.Renderer.WriteString(timeStr, r.X+r.Width/2-5, y, timeStyle.Foreground, timeStyle.Background, timeStyle.Attrs)
	y += 2

	// Progress bar
	progress := float64(p.elapsed) / float64(p.total)
	if progress > 1 {
		progress = 1
	}
	barW := r.Width - 8
	filled := int(progress * float64(barW))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barW-filled)
	ctx.Renderer.WriteString("  "+bar+"  ", r.X, y, modeColor, mofu.ColorBlack, 0)
	y += 2

	// Status
	if p.running {
		ctx.Renderer.WriteString(" ▶ RUNNING", r.X, y, modeColor, mofu.ColorBlack, 0)
	} else {
		ctx.Renderer.WriteString(" ■ PAUSED", r.X, y, mofu.Hex("f38ba8"), mofu.ColorBlack, 0)
	}
	y += 2

	// Stats
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	ctx.Renderer.WriteString(fmt.Sprintf(" Sessions: %d", p.sessions), r.X, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
	y++
	ctx.Renderer.WriteString(fmt.Sprintf(" Breaks: %d", p.breaks), r.X, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
	y++
	ctx.Renderer.WriteString(fmt.Sprintf(" Long Breaks: %d", p.longBreaks), r.X, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
	y += 2

	// Session indicators
	indicators := ""
	for i := 0; i < 4; i++ {
		if i < p.sessions%4 {
			indicators += " ●"
		} else {
			indicators += " ○"
		}
	}
	ctx.Renderer.WriteString(indicators, r.X, y, modeColor, mofu.ColorBlack, 0)

	// Status bar
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, r.Y+r.Height-2, mofu.Hex("444444"), mofu.ColorBlack, 0)
	ctx.Renderer.WriteString(" space:start/stop r:reset m:mode w:work b:break q:quit", r.X, r.Y+r.Height-1, mofu.Hex("585b70"), mofu.Hex("1e1e2e"), 0)
}

func (p *Pomodoro) HandleEvent(event mofu.Event) mofu.Cmd {
	if event.Type != mofu.EventKeyPress {
		return nil
	}
	ke := event.Data.(mofu.KeyEvent)

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()

	case ke.Key == mofu.KeySpace:
		p.running = !p.running
		if p.running {
			p.lastTick = time.Now()
		}

	case len(ke.Runes) > 0 && ke.Runes[0] == 'r':
		p.elapsed = 0
		p.running = false

	case len(ke.Runes) > 0 && ke.Runes[0] == 'w':
		p.mode = "work"
		p.total = 25 * time.Minute
		p.elapsed = 0
		p.running = false

	case len(ke.Runes) > 0 && ke.Runes[0] == 'b':
		p.mode = "break"
		p.total = 5 * time.Minute
		p.elapsed = 0
		p.running = false
	}

	return nil
}

func main() {
	app := NewPomodoro()
	if err := mofu.Run(app); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
