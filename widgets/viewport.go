package widgets

import (
	"strings"

	"github.com/xanstomper/mofu"
)

// ViewportNode provides virtual scrolling over a body of text.
//
// Content is a single string; lazy line slicing
// keeps large logs/streams cheap to render.
type ViewportNode struct {
	mofu.BaseNode
	content       string
	contentLines  []string
	scrollY       int
	scrollX       int
	wrap          bool
	showScrollbar bool
	lineCache     map[int][]string
	DirtyReason   string
}

func NewViewport() *ViewportNode {
	return &ViewportNode{wrap: true, showScrollbar: true, lineCache: make(map[int][]string)}
}

// SetContent replaces the viewport body and resets scroll position.
func (v *ViewportNode) SetContent(s string) {
	if v.content == s {
		return
	}
	v.content = s
	v.contentLines = strings.Split(s, "\n")
	v.lineCache = make(map[int][]string, min(len(v.contentLines), v.viewportHeight()*2))
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

func (v *ViewportNode) SetScrollX(x int) {
	if x < 0 {
		x = 0
	}
	if x != v.scrollX {
		v.scrollX = x
		v.SetDirty()
	}
}

func (v *ViewportNode) ScrollX() int { return v.scrollX }

func (v *ViewportNode) ScrollLeft(n int)  { v.SetScrollX(v.scrollX - n) }
func (v *ViewportNode) ScrollRight(n int) { v.SetScrollX(v.scrollX + n) }

func (v *ViewportNode) SetWrap(wrap bool) {
	if v.wrap != wrap {
		v.wrap = wrap
		v.lineCache = make(map[int][]string, min(len(v.contentLines), v.viewportHeight()*2))
		v.SetDirty()
	}
}

func (v *ViewportNode) Wrap() bool { return v.wrap }

func (v *ViewportNode) SetScrollbar(show bool) {
	if v.showScrollbar != show {
		v.showScrollbar = show
		v.SetDirty()
	}
}

func (v *ViewportNode) Scrollbar() bool { return v.showScrollbar }

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

	height := v.viewportHeight()
	if height > r.Height {
		height = r.Height
	}
	contentWidth := r.Width
	if v.showScrollbar && contentWidth > 1 {
		contentWidth--
	}
	if contentWidth <= 0 {
		return
	}

	lines := v.visibleLines(contentWidth)
	start := v.scrollY
	if start < 0 {
		start = 0
	}
	if start >= len(lines) {
		v.drawScrollbar(ctx, r, height, len(lines), 0)
		return
	}
	end := start + height
	if end > len(lines) {
		end = len(lines)
	}

	sx := r.X
	sy := r.Y
	fg := v.BaseNode.Style().Foreground
	bg := v.BaseNode.Style().Background
	attrs := v.BaseNode.Style().Attrs
	for i := start; i < end; i++ {
		ctx.Renderer.WriteString(lines[i], sx, sy, fg, bg, attrs)
		sy++
	}
	v.drawScrollbar(ctx, r, height, len(lines), start)
	v.BaseNode.SetDirty()
}

func (v *ViewportNode) visibleLines(width int) []string {
	if v.wrap {
		if v.lineCache == nil {
			v.lineCache = make(map[int][]string, min(len(v.contentLines), v.viewportHeight()*2))
		}
		if cached, ok := v.lineCache[width]; ok {
			return cached
		}
		var out []string
		for _, line := range v.contentLines {
			if mofu.MeasureWidth(line) <= width {
				out = append(out, line)
				continue
			}
			out = append(out, mofu.CharWrap(line, width)...)
		}
		if len(v.contentLines) > 0 && v.contentLines[len(v.contentLines)-1] == "" {
			out = append(out, "")
		}
		v.lineCache[width] = out
		return out
	}

	if v.lineCache == nil {
		v.lineCache = make(map[int][]string, min(len(v.contentLines), v.viewportHeight()*2))
	}
	key := -width - 1
	if cached, ok := v.lineCache[key]; ok {
		return cached
	}
	out := make([]string, len(v.contentLines))
	for i, line := range v.contentLines {
		out[i] = mofu.Truncate(v.trimLeading(line, v.scrollX), width, true)
	}
	v.lineCache[key] = out
	return out
}

func (v *ViewportNode) trimLeading(line string, width int) string {
	if width <= 0 || mofu.MeasureWidth(line) <= width {
		return line
	}
	var out strings.Builder
	w := 0
	for _, r := range line {
		cw := mofu.RuneWidth(r)
		if w+cw > width {
			break
		}
		out.WriteRune(r)
		w += cw
	}
	return out.String()
}

func (v *ViewportNode) drawScrollbar(ctx *mofu.RenderContext, r mofu.Rect, viewport, total, start int) {
	if !v.showScrollbar || r.Width <= 1 || total <= viewport {
		return
	}
	x := r.X + r.Width - 1
	fg := v.BaseNode.Style().Foreground
	bg := v.BaseNode.Style().Background
	for y := r.Y; y < r.Y+r.Height; y++ {
		ctx.Renderer.WriteString(" ", x, y, bg, fg, 0)
	}
	thumb := max(1, viewport*viewport/total)
	if thumb > viewport {
		thumb = viewport
	}
	maxStart := max(0, viewport-thumb)
	pos := 0
	if total > viewport {
		pos = start * maxStart / (total - viewport)
	}
	for i := 0; i < thumb; i++ {
		ctx.Renderer.WriteString("│", x, r.Y+pos+i, fg, bg, 0)
	}
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
		case 'h':
			v.ScrollLeft(1)
			return nil
		case 'l':
			v.ScrollRight(1)
			return nil
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
	case mofu.KeyLeft:
		v.ScrollLeft(1)
	case mofu.KeyRight:
		v.ScrollRight(1)
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

func (v *ViewportNode) Mount() mofu.Cmd             { return nil }
func (v *ViewportNode) Unmount()                    {}
func (v *ViewportNode) Children() []mofu.Node       { return nil }
func (v *ViewportNode) AddChild(child mofu.Node)    { v.BaseNode.AddChild(child) }
func (v *ViewportNode) RemoveChild(child mofu.Node) { v.BaseNode.RemoveChild(child) }
func (v *ViewportNode) Bounds() mofu.Rect           { return v.BaseNode.Bounds() }
func (v *ViewportNode) SetBounds(r mofu.Rect)       { v.BaseNode.SetBounds(r) }
func (v *ViewportNode) Style() *mofu.Style          { return v.BaseNode.Style() }
