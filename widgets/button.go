package widgets

import (
	"strings"

	"github.com/xanstomper/mofu"
)

// Button is a clickable button widget.
type Button struct {
	mofu.BaseNode
	Label    string
	Focused  bool
	Pressed  bool
	OnPress  func() mofu.Cmd
	Style    mofu.Style
	FocusStyle mofu.Style
}

// NewButton creates a button with the given label.
func NewButton(label string, onPress func() mofu.Cmd) *Button {
	return &Button{
		Label: label,
		OnPress: onPress,
		Style: mofu.DefaultStyle().Fg(mofu.Hex("e0e0e0")).Bg(mofu.Hex("333333")),
		FocusStyle: mofu.DefaultStyle().Fg(mofu.Hex("ffffff")).Bg(mofu.Hex("ff69b4")),
	}
}

func (b *Button) Focus()   { b.Focused = true; b.SetDirty() }
func (b *Button) Blur()    { b.Focused = false; b.SetDirty() }
func (b *Button) IsFocused() bool { return b.Focused }

func (b *Button) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	if r.Width <= 0 || r.Height <= 0 {
		return
	}

	style := b.Style
	if b.Focused {
		style = b.FocusStyle
	}

	label := b.Label
	if len(label) > r.Width-4 {
		label = label[:r.Width-7] + "..."
	}
	padding := (r.Width - len(label) - 2) / 2
	if padding < 0 {
		padding = 0
	}

	text := strings.Repeat(" ", padding) + label + strings.Repeat(" ", r.Width-2-padding-len(label))
	ctx.Renderer.WriteString("[", r.X, r.Y, style.Foreground, style.Background, style.Attrs)
	ctx.Renderer.WriteString(text, r.X+1, r.Y, style.Foreground, style.Background, style.Attrs)
	ctx.Renderer.WriteString("]", r.X+r.Width-1, r.Y, style.Foreground, style.Background, style.Attrs)
}

func (b *Button) HandleEvent(event mofu.Event) mofu.Cmd {
	if !b.Focused || event.Type != mofu.EventKeyPress {
		return nil
	}
	ke, ok := event.Data.(mofu.KeyEvent)
	if !ok {
		return nil
	}
	if ke.Key == mofu.KeyEnter || (len(ke.Runes) > 0 && ke.Runes[0] == ' ') {
		b.Pressed = true
		b.SetDirty()
		if b.OnPress != nil {
			cmd := b.OnPress()
			b.Pressed = false
			b.SetDirty()
			return cmd
		}
		b.Pressed = false
		b.SetDirty()
	}
	return nil
}

func (b *Button) Mount() mofu.Cmd { return nil }
func (b *Button) Unmount()        {}
