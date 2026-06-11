package widgets

import (
	"strings"

	"github.com/xanstomper/mofu"
)

// Modal is an overlay dialog that blocks input to background.
type Modal struct {
	mofu.BaseNode
	Title   string
	Child   mofu.Node
	Width   int
	Height  int
	OnClose func() mofu.Cmd
	Style   mofu.Style
}

// NewModal creates a modal dialog.
func NewModal(title string, child mofu.Node) *Modal {
	return &Modal{
		Title: title,
		Child: child,
		Width: 40,
		Height: 10,
		Style: mofu.DefaultStyle().Fg(mofu.Hex("e0e0e0")).Bg(mofu.Hex("1a1a2e")),
	}
}

func (m *Modal) Children() []mofu.Node {
	if m.Child == nil {
		return nil
	}
	return []mofu.Node{m.Child}
}

func (m *Modal) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds

	// Center the modal
	modalW := m.Width
	modalH := m.Height
	if modalW > r.Width-4 {
		modalW = r.Width - 4
	}
	if modalH > r.Height-4 {
		modalH = r.Height - 4
	}
	x := r.X + (r.Width-modalW)/2
	y := r.Y + (r.Height-modalH)/2

	// Dim background
	for dy := 0; dy < r.Height; dy++ {
		for dx := 0; dx < r.Width; dx++ {
			ctx.Renderer.WriteString(" ", r.X+dx, r.Y+dy, mofu.ColorBlack, mofu.Hex("000000"), 0)
		}
	}

	// Draw border
	bs := mofu.BorderRounded
	bsStyle := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))
	ctx.Renderer.WriteString(
		string(bs.TopLeft)+strings.Repeat(string(bs.Top), modalW-2)+string(bs.TopRight),
		x, y, bsStyle.Foreground, bsStyle.Background, bsStyle.Attrs,
	)
	for dy := 1; dy < modalH-1; dy++ {
		ctx.Renderer.WriteString(string(bs.Left), x, y+dy, bsStyle.Foreground, bsStyle.Background, bsStyle.Attrs)
		ctx.Renderer.WriteString(strings.Repeat(" ", modalW-2), x+1, y+dy, m.Style.Foreground, m.Style.Background, m.Style.Attrs)
		ctx.Renderer.WriteString(string(bs.Right), x+modalW-1, y+dy, bsStyle.Foreground, bsStyle.Background, bsStyle.Attrs)
	}
	ctx.Renderer.WriteString(
		string(bs.BottomLeft)+strings.Repeat(string(bs.Bottom), modalW-2)+string(bs.BottomRight),
		x, y+modalH-1, bsStyle.Foreground, bsStyle.Background, bsStyle.Attrs,
	)

	// Title
	titleStyle := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
	title := m.Title
	if len(title) > modalW-4 {
		title = title[:modalW-7] + "..."
	}
	titleX := x + (modalW-len(title))/2
	ctx.Renderer.WriteString(" "+title+" ", titleX, y, titleStyle.Foreground, titleStyle.Background, titleStyle.Attrs)

	// Child
	if m.Child != nil {
		inner := mofu.Rect{X: x + 2, Y: y + 2, Width: modalW - 4, Height: modalH - 4}
		m.Child.SetBounds(inner)
		childCtx := *ctx
		childCtx.Bounds = inner
		m.Child.Render(&childCtx)
	}

	// Footer
	footer := "Esc: Close"
	footerX := x + (modalW-len(footer))/2
	ctx.Renderer.WriteString(footer, footerX, y+modalH-2, mofu.Hex("666666"), mofu.ColorBlack, 0)
}

func (m *Modal) HandleEvent(event mofu.Event) mofu.Cmd {
	if event.Type == mofu.EventKeyPress {
		ke, ok := event.Data.(mofu.KeyEvent)
		if ok && ke.Key == mofu.KeyEsc {
			if m.OnClose != nil {
				return m.OnClose()
			}
			return nil
		}
	}
	if m.Child != nil {
		return m.Child.HandleEvent(event)
	}
	return nil
}

func (m *Modal) Mount() mofu.Cmd {
	if m.Child != nil {
		return m.Child.Mount()
	}
	return nil
}

func (m *Modal) Unmount() {
	if m.Child != nil {
		m.Child.Unmount()
	}
}
