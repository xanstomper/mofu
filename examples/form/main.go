package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/xanstomper/mofu"
)

type Form struct {
	mofu.Minimal
	name      string
	email     string
	password  string
	agree     bool
	field     int
	submitted bool
}

func (f *Form) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds

	titleStyle := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
	ctx.Renderer.WriteString(" Registration Form", r.X, r.Y, titleStyle.Foreground, titleStyle.Background, titleStyle.Attrs)
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, r.Y+1, mofu.Hex("444444"), mofu.ColorBlack, 0)

	fields := []struct {
		label string
		value string
	}{
		{"Name", f.name},
		{"Email", f.email},
		{"Password", f.password},
	}

	for i, field := range fields {
		y := r.Y + 3 + i*2
		prefix := "  "
		if i == f.field {
			prefix = "▸ "
		}
		ctx.Renderer.WriteString(fmt.Sprintf("%s%s:", prefix, field.label), r.X+1, y, mofu.Hex("89b4fa"), mofu.ColorBlack, 0)

		val := field.value
		if field.label == "Password" {
			val = strings.Repeat("*", len(val))
		}
		if i == f.field {
			val += "_"
		}
		ctx.Renderer.WriteString(val, r.X+15, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
	}

	y := r.Y + 3 + len(fields)*2
	agreeIcon := "○"
	if f.agree {
		agreeIcon = "●"
	}
	agreePrefix := "  "
	if f.field == 3 {
		agreePrefix = "▸ "
	}
	ctx.Renderer.WriteString(fmt.Sprintf("%s%s I agree to the terms", agreePrefix, agreeIcon), r.X+1, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)

	y += 2
	if f.submitted {
		ctx.Renderer.WriteString(" Form submitted!", r.X+1, y, mofu.Hex("a6e3a1"), mofu.ColorBlack, mofu.AttrBold)
	}

	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, r.Y+r.Height-2, mofu.Hex("444444"), mofu.ColorBlack, 0)
	ctx.Renderer.WriteString(" Tab/Shift+Tab:field Enter:submit q:quit", r.X, r.Y+r.Height-1, mofu.Hex("585b70"), mofu.Hex("1e1e2e"), 0)
}

func (f *Form) HandleEvent(event mofu.Event) mofu.Cmd {
	if event.Type != mofu.EventKeyPress {
		return nil
	}
	ke := event.Data.(mofu.KeyEvent)

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()
	case ke.Key == mofu.KeyTab:
		f.field = (f.field + 1) % 4
	case ke.Key == mofu.KeyEnter:
		f.submitted = true
	case ke.Key == mofu.KeyBack:
		switch f.field {
		case 0:
			if len(f.name) > 0 {
				f.name = f.name[:len(f.name)-1]
			}
		case 1:
			if len(f.email) > 0 {
				f.email = f.email[:len(f.email)-1]
			}
		case 2:
			if len(f.password) > 0 {
				f.password = f.password[:len(f.password)-1]
			}
		}
	case f.field == 3 && ke.Key == mofu.KeySpace:
		f.agree = !f.agree
	default:
		if len(ke.Runes) > 0 {
			switch f.field {
			case 0:
				f.name += string(ke.Runes)
			case 1:
				f.email += string(ke.Runes)
			case 2:
				f.password += string(ke.Runes)
			}
		}
	}
	return nil
}

func main() {
	app := &Form{}
	if err := mofu.Run(app); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
