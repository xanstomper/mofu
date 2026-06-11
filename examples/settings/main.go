package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/xanstomper/mofu"
	"github.com/xanstomper/mofu/widgets"
)

// Settings — a settings panel with checkboxes and selects.

type Settings struct {
	mofu.Minimal
	darkMode  *widgets.Checkbox
	notifications *widgets.Checkbox
	language  *widgets.Select
	theme     *widgets.Select
	focus     int
	width     int
	height    int
	saved     bool
}

func NewSettings() *Settings {
	s := &Settings{
		darkMode:       widgets.NewCheckbox("Dark Mode", true),
		notifications:  widgets.NewCheckbox("Enable Notifications", true),
		language:       widgets.NewSelect([]string{"English", "Spanish", "French", "German", "Japanese"}),
		theme:          widgets.NewSelect([]string{"Mochi", "Catppuccin", "Tokyo Night", "Dracula"}),
	}
	s.language.SetSelected(0)
	s.theme.SetSelected(0)
	return s
}

func (s *Settings) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	s.width = r.Width
	s.height = r.Height

	titleStyle := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
	ctx.Renderer.WriteString(" Settings", r.X, r.Y, titleStyle.Foreground, titleStyle.Background, titleStyle.Attrs)

	sep := strings.Repeat("─", r.Width-2)
	ctx.Renderer.WriteString(sep, r.X+1, r.Y+1, mofu.Hex("444444"), mofu.ColorBlack, 0)

	y := r.Y + 3
	labelStyle := mofu.DefaultStyle().Fg(mofu.Hex("89b4fa"))

	// Appearance section
	ctx.Renderer.WriteString("Appearance", r.X+2, y, labelStyle.Foreground, labelStyle.Background, labelStyle.Attrs)
	y++

	s.darkMode.SetBounds(mofu.Rect{X: r.X + 4, Y: y, Width: r.Width - 6, Height: 1})
	s.darkMode.Render(ctx)
	if s.focus == 0 {
		s.darkMode.Focus()
	} else {
		s.darkMode.Blur()
	}
	y += 2

	// Theme select
	ctx.Renderer.WriteString("Theme:", r.X+2, y, labelStyle.Foreground, labelStyle.Background, labelStyle.Attrs)
	y++
	s.theme.SetBounds(mofu.Rect{X: r.X + 4, Y: y, Width: r.Width - 6, Height: 1})
	s.theme.Render(ctx)
	if s.focus == 1 {
		s.theme.Focus()
	} else {
		s.theme.Blur()
	}
	y += 3

	// Notifications section
	ctx.Renderer.WriteString("Notifications", r.X+2, y, labelStyle.Foreground, labelStyle.Background, labelStyle.Attrs)
	y++

	s.notifications.SetBounds(mofu.Rect{X: r.X + 4, Y: y, Width: r.Width - 6, Height: 1})
	s.notifications.Render(ctx)
	if s.focus == 2 {
		s.notifications.Focus()
	} else {
		s.notifications.Blur()
	}
	y += 3

	// Language section
	ctx.Renderer.WriteString("Language", r.X+2, y, labelStyle.Foreground, labelStyle.Background, labelStyle.Attrs)
	y++

	s.language.SetBounds(mofu.Rect{X: r.X + 4, Y: y, Width: r.Width - 6, Height: 1})
	s.language.Render(ctx)
	if s.focus == 3 {
		s.language.Focus()
	} else {
		s.language.Blur()
	}

	// Status
	if s.saved {
		ctx.Renderer.WriteString(" Settings saved! ", r.X+2, r.Y+r.Height-1, mofu.Hex("a6e3a1"), mofu.ColorBlack, 0)
	} else {
		ctx.Renderer.WriteString(" Tab: Next │ Enter: Save │ q: Quit", r.X, r.Y+r.Height-1, mofu.Hex("666666"), mofu.ColorBlack, 0)
	}
}

func (s *Settings) HandleEvent(event mofu.Event) mofu.Cmd {
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

	if ke.Key == mofu.KeyTab {
		s.focus = (s.focus + 1) % 4
		return nil
	}

	if ke.Key == mofu.KeyEnter {
		s.saved = true
		return nil
	}

	switch s.focus {
	case 0:
		return s.darkMode.HandleEvent(event)
	case 1:
		return s.theme.HandleEvent(event)
	case 2:
		return s.notifications.HandleEvent(event)
	case 3:
		return s.language.HandleEvent(event)
	}
	return nil
}

func main() {
	app := NewSettings()
	if err := mofu.Run(app); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
