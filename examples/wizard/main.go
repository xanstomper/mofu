package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/xanstomper/mofu"
	"github.com/xanstomper/mofu/widgets"
)

// Wizard — a multi-step setup wizard example.

type Wizard struct {
	mofu.Minimal
	step    int
	width   int
	height  int
	// Step 1: Welcome
	// Step 2: User info
	// Step 3: Preferences
	// Step 4: Complete
	name    *widgets.InputNode
	email   *widgets.InputNode
	theme   *widgets.Select
	lang    *widgets.Select
	focus   int
}

func NewWizard() *Wizard {
	w := &Wizard{
		step:  0,
		name:  widgets.NewInput(),
		email: widgets.NewInput(),
		theme: widgets.NewSelect([]string{"Mochi", "Catppuccin", "Tokyo Night"}),
		lang:  widgets.NewSelect([]string{"Go", "Python", "Rust", "TypeScript"}),
	}
	w.name.Placeholder = "Enter your name"
	w.email.Placeholder = "Enter your email"
	return w
}

func (w *Wizard) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	w.width = r.Width
	w.height = r.Height

	titleStyle := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
	ctx.Renderer.WriteString(" Setup Wizard", r.X, r.Y, titleStyle.Foreground, titleStyle.Background, titleStyle.Attrs)

	// Progress bar
	sep := strings.Repeat("─", r.Width-2)
	ctx.Renderer.WriteString(sep, r.X+1, r.Y+1, mofu.Hex("444444"), mofu.ColorBlack, 0)

	progress := fmt.Sprintf(" Step %d/4 ", w.step+1)
	ctx.Renderer.WriteString(progress, r.X+r.Width-len(progress)-1, r.Y, mofu.Hex("666666"), mofu.ColorBlack, 0)

	// Progress dots
	dotY := r.Y + 2
	for i := 0; i < 4; i++ {
		dot := "○"
		if i <= w.step {
			dot = "●"
		}
		ctx.Renderer.WriteString(dot, r.X+2+i*3, dotY, mofu.Hex("ff69b4"), mofu.ColorBlack, 0)
	}

	y := r.Y + 4
	labelStyle := mofu.DefaultStyle().Fg(mofu.Hex("89b4fa"))

	switch w.step {
	case 0: // Welcome
		ctx.Renderer.WriteString("Welcome to the Setup Wizard!", r.X+2, y, mofu.Hex("e0e0e0"), mofu.ColorBlack, 0)
		y += 2
		ctx.Renderer.WriteString("This wizard will guide you through", r.X+2, y, mofu.Hex("666666"), mofu.ColorBlack, 0)
		y++
		ctx.Renderer.WriteString("setting up your MOFU application.", r.X+2, y, mofu.Hex("666666"), mofu.ColorBlack, 0)

	case 1: // User info
		ctx.Renderer.WriteString("User Information", r.X+2, y, labelStyle.Foreground, labelStyle.Background, labelStyle.Attrs)
		y += 2

		ctx.Renderer.WriteString("Name:", r.X+2, y, labelStyle.Foreground, labelStyle.Background, labelStyle.Attrs)
		y++
		w.name.SetBounds(mofu.Rect{X: r.X + 2, Y: y, Width: r.Width - 4, Height: 1})
		w.name.Render(ctx)
		if w.focus == 0 {
			w.name.Focus()
		} else {
			w.name.Blur()
		}
		y += 2

		ctx.Renderer.WriteString("Email:", r.X+2, y, labelStyle.Foreground, labelStyle.Background, labelStyle.Attrs)
		y++
		w.email.SetBounds(mofu.Rect{X: r.X + 2, Y: y, Width: r.Width - 4, Height: 1})
		w.email.Render(ctx)
		if w.focus == 1 {
			w.email.Focus()
		} else {
			w.email.Blur()
		}

	case 2: // Preferences
		ctx.Renderer.WriteString("Preferences", r.X+2, y, labelStyle.Foreground, labelStyle.Background, labelStyle.Attrs)
		y += 2

		ctx.Renderer.WriteString("Theme:", r.X+2, y, labelStyle.Foreground, labelStyle.Background, labelStyle.Attrs)
		y++
		w.theme.SetBounds(mofu.Rect{X: r.X + 2, Y: y, Width: r.Width - 4, Height: 1})
		w.theme.Render(ctx)
		if w.focus == 0 {
			w.theme.Focus()
		} else {
			w.theme.Blur()
		}
		y += 2

		ctx.Renderer.WriteString("Language:", r.X+2, y, labelStyle.Foreground, labelStyle.Background, labelStyle.Attrs)
		y++
		w.lang.SetBounds(mofu.Rect{X: r.X + 2, Y: y, Width: r.Width - 4, Height: 1})
		w.lang.Render(ctx)
		if w.focus == 1 {
			w.lang.Focus()
		} else {
			w.lang.Blur()
		}

	case 3: // Complete
		successStyle := mofu.DefaultStyle().Fg(mofu.Hex("a6e3a1")).WithAttrs(mofu.AttrBold)
		ctx.Renderer.WriteString("Setup Complete!", r.X+2, y, successStyle.Foreground, successStyle.Background, successStyle.Attrs)
		y += 2
		ctx.Renderer.WriteString("Name: "+w.name.Value, r.X+2, y, mofu.Hex("e0e0e0"), mofu.ColorBlack, 0)
		y++
		ctx.Renderer.WriteString("Email: "+w.email.Value, r.X+2, y, mofu.Hex("e0e0e0"), mofu.ColorBlack, 0)
		y++
		ctx.Renderer.WriteString("Theme: "+w.theme.Options[w.theme.Selected], r.X+2, y, mofu.Hex("e0e0e0"), mofu.ColorBlack, 0)
		y++
		ctx.Renderer.WriteString("Language: "+w.lang.Options[w.lang.Selected], r.X+2, y, mofu.Hex("e0e0e0"), mofu.ColorBlack, 0)
	}

	// Footer
	ctx.Renderer.WriteString(" ←: Back │ →: Next │ Enter: Confirm │ q: Quit ", r.X, r.Y+r.Height-1, mofu.Hex("666666"), mofu.ColorBlack, 0)
}

func (w *Wizard) HandleEvent(event mofu.Event) mofu.Cmd {
	if event.Type != mofu.EventKeyPress {
		return nil
	}
	ke, ok := event.Data.(mofu.KeyEvent)
	if !ok {
		return nil
	}

	if ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && (ke.Runes[0] == 'q' || ke.Runes[0] == 'Q')) {
		return mofu.QuitCmd()
	}

	// Tab between fields
	if ke.Key == mofu.KeyTab {
		w.focus = (w.focus + 1) % 2
		return nil
	}

	// Back
	if ke.Key == mofu.KeyLeft || (len(ke.Runes) > 0 && ke.Runes[0] == 'h') {
		if w.step > 0 {
			w.step--
			w.focus = 0
		}
		return nil
	}

	// Next
	if ke.Key == mofu.KeyRight || (len(ke.Runes) > 0 && ke.Runes[0] == 'l') {
		if w.step < 3 {
			w.step++
			w.focus = 0
		}
		return nil
	}

	// Route to current step
	switch w.step {
	case 1:
		return w.name.HandleEvent(event)
	case 2:
		if w.focus == 0 {
			return w.theme.HandleEvent(event)
		}
		return w.lang.HandleEvent(event)
	}
	return nil
}

func main() {
	app := NewWizard()
	if err := mofu.Run(app); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
