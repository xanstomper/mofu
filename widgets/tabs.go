package widgets

import (
	"github.com/xanstomper/mofu"
)

// Tab represents a single tab.
type Tab struct {
	Label string
	Data  any
}

// Tabs displays a horizontal tab bar with selection.
type Tabs struct {
	mofu.BaseNode
	Tabs     []Tab
	Selected int
	OnChange func(index int) mofu.Cmd
	Style    mofu.Style
	ActiveStyle mofu.Style
}

// NewTabs creates a tab bar.
func NewTabs(tabs []Tab) *Tabs {
	return &Tabs{
		Tabs: tabs,
		Style: mofu.DefaultStyle().Fg(mofu.Hex("666666")),
		ActiveStyle: mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold),
	}
}

func (t *Tabs) SetSelected(i int) {
	if i < 0 || i >= len(t.Tabs) {
		return
	}
	t.Selected = i
	t.SetDirty()
}

func (t *Tabs) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	if r.Width <= 0 || r.Height <= 0 {
		return
	}

	x := r.X
	for i, tab := range t.Tabs {
		if x >= r.X+r.Width {
			break
		}

		style := t.Style
		if i == t.Selected {
			style = t.ActiveStyle
		}

		label := tab.Label
		if len(label) > 10 {
			label = label[:7] + "..."
		}

		text := " " + label + " "
		if x+len(text) > r.X+r.Width {
			text = text[:r.X+r.Width-x]
		}

		ctx.Renderer.WriteString(text, x, r.Y, style.Foreground, style.Background, style.Attrs)
		x += len(text)
	}
}

func (t *Tabs) HandleEvent(event mofu.Event) mofu.Cmd {
	if event.Type != mofu.EventKeyPress {
		return nil
	}
	ke, ok := event.Data.(mofu.KeyEvent)
	if !ok {
		return nil
	}

	switch ke.Key {
	case mofu.KeyLeft:
		if t.Selected > 0 {
			t.Selected--
			t.SetDirty()
			if t.OnChange != nil {
				return t.OnChange(t.Selected)
			}
		}
	case mofu.KeyRight:
		if t.Selected < len(t.Tabs)-1 {
			t.Selected++
			t.SetDirty()
			if t.OnChange != nil {
				return t.OnChange(t.Selected)
			}
		}
	}
	return nil
}

func (t *Tabs) Mount() mofu.Cmd { return nil }
func (t *Tabs) Unmount()        {}

// Truncate truncates a string to the given width, adding "..." if truncated.
func Truncate(s string, width int, ellipsis bool) string {
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	if ellipsis && width > 3 {
		return string(runes[:width-3]) + "..."
	}
	return string(runes[:width])
}
