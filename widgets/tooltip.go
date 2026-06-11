package widgets

import (
	"strings"

	"github.com/xanstomper/mofu"
)

// Tooltip shows help text near an element.
type Tooltip struct {
	mofu.BaseNode
	Text    string
	Visible bool
	Style   mofu.Style
}

// NewTooltip creates a tooltip with the given text.
func NewTooltip(text string) *Tooltip {
	return &Tooltip{
		Text:  text,
		Style: mofu.DefaultStyle().Fg(mofu.Hex("e0e0e0")).Bg(mofu.Hex("333333")),
	}
}

func (t *Tooltip) Show()  { t.Visible = true; t.SetDirty() }
func (t *Tooltip) Hide()  { t.Visible = false; t.SetDirty() }
func (t *Tooltip) Toggle() { t.Visible = !t.Visible; t.SetDirty() }

func (t *Tooltip) Render(ctx *mofu.RenderContext) {
	if !t.Visible {
		return
	}

	r := ctx.Bounds
	if r.Width <= 0 || r.Height <= 0 {
		return
	}

	// Word wrap
	lines := t.wrapText(t.Text, r.Width-2)
	if len(lines) == 0 {
		return
	}

	// Find max width
	maxW := 0
	for _, line := range lines {
		if len(line) > maxW {
			maxW = len(line)
		}
	}
	maxW += 2 // padding

	// Position tooltip
	x := r.X
	y := r.Y
	if x+maxW > r.X+r.Width {
		x = r.X + r.Width - maxW
	}

	// Draw tooltip
	bs := mofu.BorderNormal
	bsStyle := mofu.DefaultStyle().Fg(mofu.Hex("666666"))

	// Top border
	ctx.Renderer.WriteString(
		string(bs.TopLeft)+strings.Repeat(string(bs.Top), maxW-2)+string(bs.TopRight),
		x, y, bsStyle.Foreground, bsStyle.Background, bsStyle.Attrs,
	)

	// Content
	for i, line := range lines {
		row := y + 1 + i
		if row >= r.Y+r.Height {
			break
		}
		padded := " " + line + strings.Repeat(" ", maxW-2-len(line)) + " "
		ctx.Renderer.WriteString(padded, x, row, t.Style.Foreground, t.Style.Background, t.Style.Attrs)
	}

	// Bottom border
	ctx.Renderer.WriteString(
		string(bs.BottomLeft)+strings.Repeat(string(bs.Bottom), maxW-2)+string(bs.BottomRight),
		x, y+len(lines)+1, bsStyle.Foreground, bsStyle.Background, bsStyle.Attrs,
	)
}

func (t *Tooltip) wrapText(text string, width int) []string {
	if width <= 0 {
		return nil
	}
	var lines []string
	for _, paragraph := range strings.Split(text, "\n") {
		if len(paragraph) <= width {
			lines = append(lines, paragraph)
			continue
		}
		for len(paragraph) > width {
			// Find last space before width
			split := width
			for i := width - 1; i > 0; i-- {
				if paragraph[i] == ' ' {
					split = i
					break
				}
			}
			lines = append(lines, paragraph[:split])
			paragraph = strings.TrimLeft(paragraph[split:], " ")
		}
		if paragraph != "" {
			lines = append(lines, paragraph)
		}
	}
	return lines
}

func (t *Tooltip) HandleEvent(event mofu.Event) mofu.Cmd { return nil }
func (t *Tooltip) Mount() mofu.Cmd                      { return nil }
func (t *Tooltip) Unmount()                             {}
