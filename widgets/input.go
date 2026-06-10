package widgets

import (
	"strings"

	"github.com/anomalyco/mofu"
)

type InputNode struct {
	mofu.BaseNode
	Value       string
	Placeholder string
	CursorPos   int
	Focused     bool
	OnSubmit    func(string) mofu.Cmd
}

func NewInput() *InputNode {
	return &InputNode{}
}

func (i *InputNode) Focus() { i.Focused = true }
func (i *InputNode) Blur()  { i.Focused = false }

func (i *InputNode) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	display := i.Value
	if display == "" && !i.Focused {
		display = i.Placeholder
	}
	w := r.Width - 4
	if w < 0 {
		w = 0
	}
	if len(display) > w {
		display = display[:w-3] + "..."
	}
	if i.Focused {
		display += strings.Repeat(" ", w-len(display)) + "█"
	}
	ctx.Renderer.WriteStyledString("[ "+display+" ]", r.X, r.Y, *i.Style())
}

func (i *InputNode) HandleEvent(event mofu.Event) mofu.Cmd {
	if !i.Focused {
		return nil
	}
	if event.Type != mofu.EventKeyPress {
		return nil
	}
	ke, ok := event.Data.(mofu.KeyEvent)
	if !ok {
		return nil
	}
	for _, b := range ke.Runes {
		switch b {
		case '\r', '\n':
			if i.OnSubmit != nil {
				return i.OnSubmit(i.Value)
			}
		case '\x7f', '\b':
			if len(i.Value) > 0 {
				i.Value = i.Value[:len(i.Value)-1]
			}
		default:
			if b >= 32 && b <= 126 {
				i.Value += string(b)
			}
		}
	}
	return nil
}

func (i *InputNode) Mount() mofu.Cmd { return nil }
func (i *InputNode) Unmount()        {}
