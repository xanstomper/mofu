package widgets

import (
	"strings"

	"github.com/anomalyco/mofu"
)

// ViewportNode provides virtual scrolling over a body of text.
//
// Production reference: charmbracelet/bubbles/viewport.
// Mofu extension: content is a single string; lazy line slicing
// keeps large logs/streams cheap to render.
type ViewportNode struct {
	mofu.BaseNode
	content     string
	contentLines []string
	scrollY     int
}

func NewViewport() *ViewportNode {
	return &ViewportNode{}
}

// SetContent replaces the viewport body and resets scroll position.
func (v *ViewportNode) SetContent(s string) {
	if v.content == s {
		return
	}
	v.content = s
	v.contentLines = strings.Split(s, "\n")
	if v.scrollY > len(v.contentLines) {
		v.scrollY = len(v.contentLines)
	}
	v.SetDirty()
}

func (v *ViewportNode) Content() string { return v.content }

// ScrollTo places the first visible line at the given index, clamped to valid range.
func (v *ViewportNode) ScrollTo(y int) {
	if y < 0 {
		y = 0
	}
	if y > len(v.contentLines) {
		y = len(v.contentLines)
	}
	if y != v.scrollY {
		v.scrollY = y
		v.SetDirty()
	}
}

func (v *ViewportNode) ScrollY() int { return v.scrollY }

func (v *ViewportNode) ScrollUp(n int)   { v.ScrollTo(v.scrollY - n) }
func (v *ViewportNode) ScrollDown(n int) { v.ScrollTo(v.scrollY + n) }

func (v *ViewportNode) PageUp()   { v.ScrollUp(v.viewportHeight()) }
func (v *ViewportNode) PageDown() { v.ScrollDown(v.viewportHeight()) }
func (v *ViewportNode) LineUp()   { v.ScrollUp(1) }
func (v *ViewportNode) LineDown() { v.ScrollDown(1) }

func (v *ViewportNode) Top()    { v.ScrollTo(0) }
func (v *ViewportNode) Bottom() { v.ScrollTo(len(v.contentLines)) }

func (v *ViewportNode) viewportHeight() int {
	b := v.BaseNode.Bounds()
	if b.Height <= 0 {
		return 1
	}
	return b.Height
}

func (v *ViewportNode) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	if r.Width <= 0 || r.Height <= 0 {
		return
	}

	lines := v.contentLines
	height := v.viewportHeight()
	start := v.scrollY
	end := start + height
	if start < 0 {
		start = 0
	}
	if end > len(lines) {
		end = len(lines)
	}
	if start >= end && len(lines) == 0 {
		// nothing to draw
		return
	}
	if start >= end {
		start = 0
		if height > len(lines) {
			end = len(lines)
		} else {
			end = height
		}
	}

	sx := r.X
	sy := r.Y
	for i := start; i < end; i++ {
		line := lines[i]
		ctx.Renderer.WriteString(line, sx, sy, v.BaseNode.Style().Foreground, v.BaseNode.Style().Background, v.BaseNode.Style().Attrs)
		sy++
	}
	v.BaseNode.SetDirty()
}

func (v *ViewportNode) HandleEvent(event mofu.Event) mofu.Cmd {
	if event.Type != mofu.EventKeyPress {
		return nil
	}

	ke, ok := event.Data.(mofu.KeyEvent)
	if !ok {
		return nil
	}

	for _, r := range ke.Runes {
		switch r {
		case 'k':
			v.LineUp()
			return nil
		case 'j':
			v.LineDown()
			return nil
		case 'g':
			v.Top()
			return nil
		case 'G':
			v.Bottom()
			return nil
		}
	}

	switch ke.Key {
	case mofu.KeyUp:
		v.LineUp()
	case mofu.KeyDown:
		v.LineDown()
	case mofu.KeyPgUp:
		v.PageUp()
	case mofu.KeyPgDn:
		v.PageDown()
	case mofu.KeyHome:
		v.Top()
	case mofu.KeyEnd:
		v.Bottom()
	}

	return nil
}

func (v *ViewportNode) Mount() mofu.Cmd      { return nil }
func (v *ViewportNode) Unmount()             {}
func (v *ViewportNode) Children() []mofu.Node { return nil }
func (v *ViewportNode) AddChild(mofu.Node)   {}
func (v *ViewportNode) RemoveChild(mofu.Node) {}
func (v *ViewportNode) Bounds() mofu.Rect    { return v.BaseNode.Bounds() }
func (v *ViewportNode) SetBounds(r mofu.Rect) { v.BaseNode.SetBounds(r) }
func (v *ViewportNode) Style() *mofu.Style   { return v.BaseNode.Style() }
