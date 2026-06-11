package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/xanstomper/mofu"
)

type Wizard struct {
	mofu.Minimal
	step     int
	name     string
	email    string
	language int
	finished bool
}

var langs = []string{"Go", "Python", "Rust", "TypeScript"}

func (w *Wizard) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds

	titleStyle := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
	ctx.Renderer.WriteString(fmt.Sprintf(" Setup Wizard — Step %d/3", w.step+1), r.X, r.Y, titleStyle.Foreground, titleStyle.Background, titleStyle.Attrs)
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, r.Y+1, mofu.Hex("444444"), mofu.ColorBlack, 0)

	// Progress bar
	progress := fmt.Sprintf(" %s%s%s ", strings.Repeat("█", w.step+1), strings.Repeat("░", 2-w.step), "")
	ctx.Renderer.WriteString(progress, r.X+1, r.Y+3, mofu.Hex("89b4fa"), mofu.ColorBlack, 0)

	y := r.Y + 5

	switch w.step {
	case 0:
		ctx.Renderer.WriteString(" What is your name?", r.X+1, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
		y += 2
		ctx.Renderer.WriteString(" Name: "+w.name+"_", r.X+1, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)

	case 1:
		ctx.Renderer.WriteString(" What is your email?", r.X+1, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
		y += 2
		ctx.Renderer.WriteString(" Email: "+w.email+"_", r.X+1, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)

	case 2:
		ctx.Renderer.WriteString(" Choose your language:", r.X+1, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
		y += 2
		for i, lang := range langs {
			prefix := "  "
			if i == w.language {
				prefix = "▸ "
			}
			ctx.Renderer.WriteString(fmt.Sprintf("%s%s", prefix, lang), r.X+1, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
			y++
		}
	}

	if w.finished {
		y = r.Y + r.Height/2
		ctx.Renderer.WriteString(" Setup complete!", r.X+1, y, mofu.Hex("a6e3a1"), mofu.ColorBlack, mofu.AttrBold)
		ctx.Renderer.WriteString(fmt.Sprintf(" Name: %s", w.name), r.X+1, y+2, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
		ctx.Renderer.WriteString(fmt.Sprintf(" Email: %s", w.email), r.X+1, y+3, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
		ctx.Renderer.WriteString(fmt.Sprintf(" Language: %s", langs[w.language]), r.X+1, y+4, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
	}

	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, r.Y+r.Height-2, mofu.Hex("444444"), mofu.ColorBlack, 0)
	ctx.Renderer.WriteString(" Enter:next/finish ←→:change q:quit", r.X, r.Y+r.Height-1, mofu.Hex("585b70"), mofu.Hex("1e1e2e"), 0)
}

func (w *Wizard) HandleEvent(event mofu.Event) mofu.Cmd {
	if event.Type != mofu.EventKeyPress {
		return nil
	}
	ke := event.Data.(mofu.KeyEvent)

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()

	case ke.Key == mofu.KeyEnter:
		if w.finished {
			return mofu.QuitCmd()
		}
		w.step++
		if w.step > 2 {
			w.finished = true
		}

	case w.step == 0 && ke.Key == mofu.KeyBack && len(w.name) > 0:
		w.name = w.name[:len(w.name)-1]
	case w.step == 0 && len(ke.Runes) > 0:
		w.name += string(ke.Runes)

	case w.step == 1 && ke.Key == mofu.KeyBack && len(w.email) > 0:
		w.email = w.email[:len(w.email)-1]
	case w.step == 1 && len(ke.Runes) > 0:
		w.email += string(ke.Runes)

	case w.step == 2 && (ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j')):
		w.language = (w.language + 1) % len(langs)
	case w.step == 2 && (ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k')):
		w.language--
		if w.language < 0 {
			w.language = len(langs) - 1
		}
	}
	return nil
}

func main() {
	app := &Wizard{}
	if err := mofu.Run(app); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
