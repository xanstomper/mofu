package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/xanstomper/mofu"
	"github.com/xanstomper/mofu/widgets"
)

// Form — a form with multiple input types.

type Form struct {
	mofu.Minimal
	name    *widgets.InputNode
	email   *widgets.InputNode
	agree   *widgets.Checkbox
	send    *widgets.Button
	focus   int
	width   int
	height  int
	sent    bool
}

func NewForm() *Form {
	f := &Form{
		name:  widgets.NewInput(),
		email: widgets.NewInput(),
		agree: widgets.NewCheckbox("I agree to the terms", false),
		send:  widgets.NewButton("Submit", nil),
	}
	f.name.Placeholder = "Enter your name"
	f.email.Placeholder = "Enter your email"
	f.send.OnPress = func() mofu.Cmd {
		f.sent = true
		return nil
	}
	return f
}

func (f *Form) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	f.width = r.Width
	f.height = r.Height

	titleStyle := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
	ctx.Renderer.WriteString(" Registration Form", r.X, r.Y, titleStyle.Foreground, titleStyle.Background, titleStyle.Attrs)

	sep := strings.Repeat("─", r.Width-2)
	ctx.Renderer.WriteString(sep, r.X+1, r.Y+1, mofu.Hex("444444"), mofu.ColorBlack, 0)

	if f.sent {
		successStyle := mofu.DefaultStyle().Fg(mofu.Hex("a6e3a1")).WithAttrs(mofu.AttrBold)
		ctx.Renderer.WriteString("  Form submitted successfully!", r.X+2, r.Y+4, successStyle.Foreground, successStyle.Background, successStyle.Attrs)
		ctx.Renderer.WriteString("  Name: "+f.name.Value, r.X+2, r.Y+6, mofu.Hex("e0e0e0"), mofu.ColorBlack, 0)
		ctx.Renderer.WriteString("  Email: "+f.email.Value, r.X+2, r.Y+7, mofu.Hex("e0e0e0"), mofu.ColorBlack, 0)
		ctx.Renderer.WriteString("  Press q to quit", r.X+2, r.Y+9, mofu.Hex("666666"), mofu.ColorBlack, 0)
		return
	}

	y := r.Y + 3
	labelStyle := mofu.DefaultStyle().Fg(mofu.Hex("89b4fa"))

	// Name field
	ctx.Renderer.WriteString("Name:", r.X+2, y, labelStyle.Foreground, labelStyle.Background, labelStyle.Attrs)
	y++
	f.name.SetBounds(mofu.Rect{X: r.X + 2, Y: y, Width: r.Width - 4, Height: 1})
	f.name.Render(ctx)
	if f.focus == 0 {
		f.name.Focus()
	} else {
		f.name.Blur()
	}
	y += 2

	// Email field
	ctx.Renderer.WriteString("Email:", r.X+2, y, labelStyle.Foreground, labelStyle.Background, labelStyle.Attrs)
	y++
	f.email.SetBounds(mofu.Rect{X: r.X + 2, Y: y, Width: r.Width - 4, Height: 1})
	f.email.Render(ctx)
	if f.focus == 1 {
		f.email.Focus()
	} else {
		f.email.Blur()
	}
	y += 2

	// Checkbox
	f.agree.SetBounds(mofu.Rect{X: r.X + 2, Y: y, Width: r.Width - 4, Height: 1})
	f.agree.Render(ctx)
	if f.focus == 2 {
		f.agree.Focus()
	} else {
		f.agree.Blur()
	}
	y += 2

	// Submit button
	f.send.SetBounds(mofu.Rect{X: r.X + 2, Y: y, Width: 20, Height: 1})
	f.send.Render(ctx)
	if f.focus == 3 {
		f.send.Focus()
	} else {
		f.send.Blur()
	}

	// Status
	ctx.Renderer.WriteString(" Tab: Next field │ Enter: Submit │ q: Quit", r.X, r.Y+r.Height-1, mofu.Hex("666666"), mofu.ColorBlack, 0)
}

func (f *Form) HandleEvent(event mofu.Event) mofu.Cmd {
	if event.Type != mofu.EventKeyPress {
		return nil
	}
	ke, ok := event.Data.(mofu.KeyEvent)
	if !ok {
		return nil
	}

	// Global keys
	if ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && (ke.Runes[0] == 'q' || ke.Runes[0] == 'Q')) {
		return mofu.QuitCmd()
	}

	// Tab to next field
	if ke.Key == mofu.KeyTab {
		f.focus = (f.focus + 1) % 4
		return nil
	}

	// Route to current field
	switch f.focus {
	case 0:
		return f.name.HandleEvent(event)
	case 1:
		return f.email.HandleEvent(event)
	case 2:
		return f.agree.HandleEvent(event)
	case 3:
		return f.send.HandleEvent(event)
	}
	return nil
}

func main() {
	app := NewForm()
	if err := mofu.Run(app); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
