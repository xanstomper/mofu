package widgets

import (
	"strings"

	"github.com/xanstomper/mofu"
)

type InputNode struct {
	mofu.BaseNode
	Value       string
	Placeholder string
	CursorPos   int
	Focused     bool
	MaxLen      int
	Password    bool
	Validator   func(string) bool
	OnSubmit    func(string) mofu.Cmd
	invalid     bool
}

func NewInput() *InputNode {
	return &InputNode{}
}

func (i *InputNode) Focus() {
	if !i.Focused {
		i.Focused = true
		i.SetDirty()
	}
}

func (i *InputNode) Blur() {
	if i.Focused {
		i.Focused = false
		i.SetDirty()
	}
}

func (i *InputNode) SetValue(value string) {
	i.Value = value
	i.clampCursor()
	i.invalid = i.Validator != nil && !i.Validator(i.Value)
	i.SetDirty()
}

func (i *InputNode) SetCursor(pos int) {
	i.CursorPos = pos
	i.clampCursor()
	i.SetDirty()
}

func (i *InputNode) MoveCursor(delta int) {
	i.CursorPos += delta
	i.clampCursor()
	i.SetDirty()
}

func (i *InputNode) DeleteBefore() {
	if i.CursorPos <= 0 {
		return
	}
	runes := []rune(i.Value)
	if i.CursorPos > len(runes) {
		i.CursorPos = len(runes)
	}
	i.Value = string(runes[:i.CursorPos-1]) + string(runes[i.CursorPos:])
	i.CursorPos--
	i.invalid = i.Validator != nil && !i.Validator(i.Value)
	i.SetDirty()
}

func (i *InputNode) DeleteAfter() {
	runes := []rune(i.Value)
	if i.CursorPos >= len(runes) {
		return
	}
	i.Value = string(runes[:i.CursorPos]) + string(runes[i.CursorPos+1:])
	i.invalid = i.Validator != nil && !i.Validator(i.Value)
	i.SetDirty()
}

func (i *InputNode) InsertRune(r rune) {
	if r < 32 {
		return
	}
	runes := []rune(i.Value)
	if i.MaxLen > 0 && len(runes) >= i.MaxLen {
		return
	}
	if i.CursorPos < 0 || i.CursorPos > len(runes) {
		i.CursorPos = len(runes)
	}
	out := make([]rune, 0, len(runes)+1)
	out = append(out, runes[:i.CursorPos]...)
	out = append(out, r)
	out = append(out, runes[i.CursorPos:]...)
	i.Value = string(out)
	i.CursorPos++
	i.invalid = i.Validator != nil && !i.Validator(i.Value)
	i.SetDirty()
}

func (i *InputNode) Valid() bool { return !i.invalid }

func (i *InputNode) clampCursor() {
	runes := []rune(i.Value)
	if i.CursorPos < 0 {
		i.CursorPos = 0
	}
	if i.CursorPos > len(runes) {
		i.CursorPos = len(runes)
	}
}

func (i *InputNode) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	if r.Width <= 2 {
		return
	}
	i.clampCursor()

	innerW := r.Width - 2
	runes := []rune(i.displayValue())
	start := 0
	if len(runes) > innerW {
		if i.CursorPos >= len(runes) {
			start = len(runes) - innerW
		} else if i.CursorPos > innerW {
			start = i.CursorPos - innerW + 1
		}
	}
	if start < 0 {
		start = 0
	}
	end := start + innerW
	if end > len(runes) {
		end = len(runes)
	}
	visible := string(runes[start:end])

	x := r.X
	y := r.Y
	style := *i.Style()
	ctx.Renderer.WriteString("[", x, y, style.Foreground, style.Background, style.Attrs)
	ctx.Renderer.WriteString(visible, x+1, y, style.Foreground, style.Background, style.Attrs)
	if i.Focused && innerW > 0 {
		cursor := i.CursorPos - start
		if cursor > innerW-1 {
			cursor = innerW - 1
		}
		if cursor >= 0 {
			ctx.Renderer.WriteString("█", x+1+cursor, y, style.Foreground, style.Background, style.Attrs)
		}
	}
	ctx.Renderer.WriteString("]", x+r.Width-1, y, style.Foreground, style.Background, style.Attrs)
}

func (i *InputNode) displayValue() string {
	if i.Value != "" {
		if i.Password {
			return strings.Repeat("•", len([]rune(i.Value)))
		}
		return i.Value
	}
	return i.Placeholder
}

func (i *InputNode) HandleEvent(event mofu.Event) mofu.Cmd {
	if !i.Focused || event.Type != mofu.EventKeyPress {
		return nil
	}
	ke, ok := event.Data.(mofu.KeyEvent)
	if !ok {
		return nil
	}

	switch ke.Key {
	case mofu.KeyEnter:
		if i.OnSubmit != nil && i.Validator == nil && !i.invalid {
			return i.OnSubmit(i.Value)
		}
	case mofu.KeyEsc:
		i.Blur()
	case mofu.KeyBack:
		i.DeleteBefore()
	case mofu.KeyLeft:
		i.MoveCursor(-1)
	case mofu.KeyRight:
		i.MoveCursor(1)
	case mofu.KeyHome:
		i.SetCursor(0)
	case mofu.KeyEnd:
		i.SetCursor(len([]rune(i.Value)))
	}

	for _, b := range ke.Runes {
		i.InsertRune(rune(b))
	}
	return nil
}

func (i *InputNode) Mount() mofu.Cmd { return nil }
func (i *InputNode) Unmount()        {}
