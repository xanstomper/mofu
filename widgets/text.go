package widgets

import (
	"strings"
	"unicode/utf8"

	"github.com/anomalyco/mofu"
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

	lines := strings.Split(t.Content, "\n")
	visibleLines := lines
	if t.ScrollY > 0 {
		if t.ScrollY >= len(lines) {
			visibleLines = nil
		} else {
			visibleLines = lines[t.ScrollY:]
		}
	}
	style := t.BaseNode.Style()

	maxLines := b.Height
	if len(visibleLines) > maxLines {
		visibleLines = visibleLines[:maxLines]
	}

	for lineIdx, line := range visibleLines {
		rowY := b.Y + lineIdx
		if rowY >= b.Y+b.Height {
			break
		}

		if t.Wrap {
			line = wrapText(line, b.Width)
		}

		if t.ScrollX > 0 {
			if t.ScrollX < len(line) {
				line = line[t.ScrollX:]
			} else {
				line = ""
			}
		}

		if t.Ellipsis && utf8.RuneCountInString(line) > b.Width {
			line = truncate(line, b.Width)
		}

		switch t.Align {
		case TextCenter:
			pad := (b.Width - utf8.RuneCountInString(line)) / 2
			if pad > 0 {
				line = strings.Repeat(" ", pad) + line
			}
		case TextRight:
			pad := b.Width - utf8.RuneCountInString(line)
			if pad > 0 {
				line = strings.Repeat(" ", pad) + line
			}
		case TextJustify:
			line = justifyText(line, b.Width)
		}

		r.WriteStyledString(line, b.X, rowY, *style)
	}
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
			wordLen := utf8.RuneCountInString(word)
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
		totalChars += utf8.RuneCountInString(w)
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
