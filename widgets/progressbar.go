package widgets

import (
	"fmt"
	"strings"

	"github.com/xanstomper/mofu"
)

// ProgressBar displays progress as a filled bar.
type ProgressBar struct {
	mofu.BaseNode
	Value    float64 // 0.0 to 1.0
	Width    int
	ShowPct  bool
	Style    mofu.Style
	FillStyle mofu.Style
}

// NewProgressBar creates a progress bar.
func NewProgressBar(value float64) *ProgressBar {
	return &ProgressBar{
		Value: value,
		Style: mofu.DefaultStyle().Fg(mofu.Hex("666666")),
		FillStyle: mofu.DefaultStyle().Fg(mofu.Hex("a6e3a1")),
	}
}

func (p *ProgressBar) SetValue(v float64) {
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	p.Value = v
	p.SetDirty()
}

func (p *ProgressBar) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	if r.Width <= 0 || r.Height <= 0 {
		return
	}

	barW := r.Width
	if p.ShowPct {
		barW -= 6
	}
	if barW < 2 {
		barW = 2
	}

	filled := int(p.Value * float64(barW))
	empty := barW - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	ctx.Renderer.WriteString(bar, r.X, r.Y, p.FillStyle.Foreground, p.FillStyle.Background, p.FillStyle.Attrs)

	if p.ShowPct {
		pct := fmt.Sprintf(" %3.0f%%", p.Value*100)
		ctx.Renderer.WriteString(pct, r.X+barW, r.Y, p.Style.Foreground, p.Style.Background, p.Style.Attrs)
	}
}

func (p *ProgressBar) HandleEvent(event mofu.Event) mofu.Cmd { return nil }
func (p *ProgressBar) Mount() mofu.Cmd                       { return nil }
func (p *ProgressBar) Unmount()                              {}
