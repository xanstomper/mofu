package widgets

import (
	"strings"

	"github.com/xanstomper/mofu"
)

type TextAlign int

const (
	TextLeft TextAlign = iota
	TextCenter
	TextRight
	TextJustify
)

type Text struct {
	mofu.BaseNode
	Content  string
	Align    TextAlign
	Wrap     bool
	ScrollY  int
	ScrollX  int
	Ellipsis bool
}

func NewLabel(content string) *Text {
	return &Text{
		Content:  content,
		Align:    TextLeft,
		Wrap:     true,
		Ellipsis: true,
	}
}

func (t *Text) SetContent(s string) {
	t.Content = s
	t.SetDirty()
}

func (t *Text) Render(ctx *mofu.RenderContext) {
	b := t.BaseNode.Bounds()
	r := ctx.Renderer
	if b.Width <= 0 || b.Height <= 0 {
		return
	}

	lines := t.layoutLines(b.Width)
	start := t.ScrollY
	if start < 0 {
		start = 0
	}
	if start >= len(lines) {
		return
	}
	end := start + b.Height
	if end > len(lines) {
		end = len(lines)
	}
	style := t.BaseNode.Style()

	for i := start; i < end; i++ {
		rowY := b.Y + i - start
		line := t.trimHorizontal(lines[i], t.ScrollX)
		if t.Ellipsis {
			line = mofu.Truncate(line, b.Width, true)
		} else {
			line = mofu.Truncate(line, b.Width, false)
		}
		switch t.Align {
		case TextCenter:
			line = mofu.PadCenter(line, b.Width)
		case TextRight:
			line = mofu.PadLeft(line, b.Width)
		case TextJustify:
			line = justifyText(line, b.Width)
		default:
			line = mofu.PadRight(line, b.Width)
		}
		r.WriteStyledString(line, b.X, rowY, *style)
	}
}

func (t *Text) layoutLines(width int) []string {
	if width <= 0 {
		return strings.Split(t.Content, "\n")
	}
	var out []string
	for _, line := range strings.Split(t.Content, "\n") {
		if t.Wrap {
			if line == "" {
				out = append(out, "")
				continue
			}
			out = append(out, mofu.WordWrap(line, width)...)
			continue
		}
		out = append(out, line)
	}
	return out
}

func (t *Text) trimHorizontal(line string, width int) string {
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

func wrapText(s string, width int) string {
	if width <= 0 {
		return s
	}
	var result strings.Builder
	for _, line := range strings.Split(s, "\n") {
		words := strings.Fields(line)
		if len(words) == 0 {
			result.WriteString(strings.Repeat(" ", width))
			continue
		}
		lineLen := 0
		for _, word := range words {
			wordLen := mofu.MeasureWidth(word)
			if lineLen > 0 && lineLen+1+wordLen > width {
				result.WriteString("\n")
				lineLen = 0
			}
			if lineLen > 0 {
				result.WriteString(" ")
				lineLen++
			}
			result.WriteString(word)
			lineLen += wordLen
		}
		result.WriteString("\n")
	}
	return strings.TrimRight(result.String(), "\n")
}

func justifyText(s string, width int) string {
	words := strings.Fields(s)
	if len(words) <= 1 || width <= len(s) {
		return s
	}
	totalChars := 0
	for _, w := range words {
		totalChars += mofu.MeasureWidth(w)
	}
	spaces := width - totalChars
	gaps := len(words) - 1
	if gaps <= 0 {
		return s
	}
	spacePerGap := spaces / gaps
	extra := spaces % gaps
	var result strings.Builder
	for i, w := range words {
		if i > 0 {
			n := spacePerGap
			if i <= extra {
				n++
			}
			result.WriteString(strings.Repeat(" ", n))
		}
		result.WriteString(w)
	}
	return result.String()
}

func (t *Text) HandleEvent(event mofu.Event) mofu.Cmd {
	if event.Type == mofu.EventKeyPress {
		ke, ok := event.Data.(mofu.KeyEvent)
		if ok {
			switch ke.Key {
			case mofu.KeyUp:
				if t.ScrollY > 0 {
					t.ScrollY--
					t.SetDirty()
				}
			case mofu.KeyDown:
				t.ScrollY++
				t.SetDirty()
			}
		}
	}
	return nil
}

func (t *Text) Children() []mofu.Node { return nil }
func (t *Text) Mount() mofu.Cmd       { return nil }
func (t *Text) Unmount()              {}
