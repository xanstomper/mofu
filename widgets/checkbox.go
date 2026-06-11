package widgets

import (
	"github.com/xanstomper/mofu"
)

// Checkbox is a toggle checkbox widget.
type Checkbox struct {
	mofu.BaseNode
	Label    string
	Checked  bool
	Focused  bool
	OnToggle func(checked bool) mofu.Cmd
	Style    mofu.Style
	FocusStyle mofu.Style
}

// NewCheckbox creates a checkbox with the given label.
func NewCheckbox(label string, checked bool) *Checkbox {
	return &Checkbox{
		Label: label,
		Checked: checked,
		Style: mofu.DefaultStyle().Fg(mofu.Hex("e0e0e0")),
		FocusStyle: mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")),
	}
}

func (c *Checkbox) Focus()   { c.Focused = true; c.SetDirty() }
func (c *Checkbox) Blur()    { c.Focused = false; c.SetDirty() }
func (c *Checkbox) IsFocused() bool { return c.Focused }

func (c *Checkbox) Toggle() {
	c.Checked = !c.Checked
	c.SetDirty()
	if c.OnToggle != nil {
		c.OnToggle(c.Checked)
	}
}

func (c *Checkbox) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	if r.Width <= 0 || r.Height <= 0 {
		return
	}

	style := c.Style
	if c.Focused {
		style = c.FocusStyle
	}

	marker := "[ ]"
	if c.Checked {
		marker = "[x]"
	}

	text := marker + " " + c.Label
	if len(text) > r.Width {
		text = text[:r.Width-3] + "..."
	}

	ctx.Renderer.WriteString(text, r.X, r.Y, style.Foreground, style.Background, style.Attrs)
}

func (c *Checkbox) HandleEvent(event mofu.Event) mofu.Cmd {
	if !c.Focused || event.Type != mofu.EventKeyPress {
		return nil
	}
	ke, ok := event.Data.(mofu.KeyEvent)
	if !ok {
		return nil
	}

	if ke.Key == mofu.KeySpace || ke.Key == mofu.KeyEnter || (len(ke.Runes) > 0 && ke.Runes[0] == ' ') {
		c.Toggle()
	}
	return nil
}

func (c *Checkbox) Mount() mofu.Cmd { return nil }
func (c *Checkbox) Unmount()        {}
