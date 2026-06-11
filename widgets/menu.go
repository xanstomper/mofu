package widgets

import (
	"strings"

	"github.com/xanstomper/mofu"
)

// MenuItem represents a single menu item.
type MenuItem struct {
	Label    string
	Shortcut string
	Disabled bool
	OnSelect func() mofu.Cmd
}

// Menu displays a vertical list of actions.
type Menu struct {
	mofu.BaseNode
	Items    []MenuItem
	Selected int
	Focused  bool
	OnSelect func(index int) mofu.Cmd
	Style    mofu.Style
	FocusStyle mofu.Style
}

// NewMenu creates a menu with items.
func NewMenu(items []MenuItem) *Menu {
	return &Menu{
		Items: items,
		Style: mofu.DefaultStyle().Fg(mofu.Hex("e0e0e0")),
		FocusStyle: mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")),
	}
}

func (m *Menu) Focus()   { m.Focused = true; m.SetDirty() }
func (m *Menu) Blur()    { m.Focused = false; m.SetDirty() }
func (m *Menu) IsFocused() bool { return m.Focused }

func (m *Menu) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	if r.Width <= 0 || r.Height <= 0 {
		return
	}

	y := r.Y
	for i, item := range m.Items {
		if y >= r.Y+r.Height {
			break
		}

		style := m.Style
		if i == m.Selected && m.Focused {
			style = m.FocusStyle
			ctx.Renderer.WriteString(strings.Repeat(" ", r.Width-2), r.X+1, y, mofu.ColorWhite, mofu.Hex("2a2a3e"), 0)
		}

		if item.Disabled {
			style = mofu.DefaultStyle().Fg(mofu.Hex("555555"))
		}

		// Label + shortcut
		text := " " + item.Label
		if item.Shortcut != "" {
			padding := r.Width - len(text) - len(item.Shortcut) - 3
			if padding > 0 {
				text += strings.Repeat(" ", padding)
			}
			text += item.Shortcut + " "
		} else {
			text += " "
		}

		if len(text) > r.Width-2 {
			text = text[:r.Width-5] + "..."
		}

		ctx.Renderer.WriteString(text, r.X+1, y, style.Foreground, style.Background, style.Attrs)
		y++
	}
}

func (m *Menu) HandleEvent(event mofu.Event) mofu.Cmd {
	if !m.Focused || event.Type != mofu.EventKeyPress {
		return nil
	}
	ke, ok := event.Data.(mofu.KeyEvent)
	if !ok {
		return nil
	}

	switch ke.Key {
	case mofu.KeyDown:
		if m.Selected < len(m.Items)-1 {
			m.Selected++
			m.SetDirty()
		}
	case mofu.KeyUp:
		if m.Selected > 0 {
			m.Selected--
			m.SetDirty()
		}
	case mofu.KeyEnter:
		if m.Selected >= 0 && m.Selected < len(m.Items) {
			item := m.Items[m.Selected]
			if !item.Disabled {
				if item.OnSelect != nil {
					cmd := item.OnSelect()
					m.SetDirty()
					return cmd
				}
				if m.OnSelect != nil {
					cmd := m.OnSelect(m.Selected)
					m.SetDirty()
					return cmd
				}
			}
		}
	}

	// vim bindings
	for _, b := range ke.Runes {
		switch b {
		case 'j':
			if m.Selected < len(m.Items)-1 {
				m.Selected++
				m.SetDirty()
			}
		case 'k':
			if m.Selected > 0 {
				m.Selected--
				m.SetDirty()
			}
		}
	}

	return nil
}

func (m *Menu) Mount() mofu.Cmd { return nil }
func (m *Menu) Unmount()        {}
