package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/xanstomper/mofu"
)

type Settings struct {
	mofu.Minimal
	darkMode       bool
	notifications  bool
	language       int
	theme          int
	field          int
}

var languages = []string{"English", "Spanish", "French", "German", "Japanese"}
var themes = []string{"Mochi", "Catppuccin", "Tokyo Night", "Dracula"}

func (s *Settings) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds

	titleStyle := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
	ctx.Renderer.WriteString(" Settings", r.X, r.Y, titleStyle.Foreground, titleStyle.Background, titleStyle.Attrs)
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, r.Y+1, mofu.Hex("444444"), mofu.ColorBlack, 0)

	fields := []struct {
		label string
		value string
	}{
		{"Dark Mode", onOff(s.darkMode)},
		{"Notifications", onOff(s.notifications)},
		{"Language", languages[s.language]},
		{"Theme", themes[s.theme]},
	}

	for i, field := range fields {
		y := r.Y + 3 + i*2
		prefix := "  "
		if i == s.field {
			prefix = "▸ "
		}
		ctx.Renderer.WriteString(fmt.Sprintf("%s%s:", prefix, field.label), r.X+1, y, mofu.Hex("89b4fa"), mofu.ColorBlack, 0)
		ctx.Renderer.WriteString(field.value, r.X+25, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
	}

	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, r.Y+r.Height-2, mofu.Hex("444444"), mofu.ColorBlack, 0)
	ctx.Renderer.WriteString(" ↑↓:select ←→:change q:quit", r.X, r.Y+r.Height-1, mofu.Hex("585b70"), mofu.Hex("1e1e2e"), 0)
}

func onOff(v bool) string {
	if v {
		return "ON"
	}
	return "OFF"
}

func (s *Settings) HandleEvent(event mofu.Event) mofu.Cmd {
	if event.Type != mofu.EventKeyPress {
		return nil
	}
	ke := event.Data.(mofu.KeyEvent)

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()
	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		s.field = (s.field + 1) % 4
	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		s.field--
		if s.field < 0 {
			s.field = 3
		}
	case ke.Key == mofu.KeyRight || (len(ke.Runes) > 0 && ke.Runes[0] == 'l'):
		switch s.field {
		case 0:
			s.darkMode = true
		case 1:
			s.notifications = true
		case 2:
			s.language = (s.language + 1) % len(languages)
		case 3:
			s.theme = (s.theme + 1) % len(themes)
		}
	case ke.Key == mofu.KeyLeft || (len(ke.Runes) > 0 && ke.Runes[0] == 'h'):
		switch s.field {
		case 0:
			s.darkMode = false
		case 1:
			s.notifications = false
		case 2:
			s.language--
			if s.language < 0 {
				s.language = len(languages) - 1
			}
		case 3:
			s.theme--
			if s.theme < 0 {
				s.theme = len(themes) - 1
			}
		}
	}
	return nil
}

func main() {
	app := &Settings{darkMode: true, notifications: true}
	if err := mofu.Run(app); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
