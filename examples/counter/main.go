package main

import (
	"fmt"
	"os"

	"github.com/anomalyco/mofu"
	"github.com/anomalyco/mofu/widgets"
)

type App struct {
	mofu.BaseNode
	count int
}

func (a *App) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	text := fmt.Sprintf("Count: %d\n\nPress 'q' to quit, 'j'/'k' to change", a.count)
	ctx.Renderer.WriteStyledString(text, r.X, r.Y, *a.Style())
}

func (a *App) HandleEvent(event mofu.Event) mofu.Cmd {
	if event.Type != mofu.EventKeyPress {
		return nil
	}
	ke, ok := event.Data.(mofu.KeyEvent)
	if !ok {
		return nil
	}
	for _, b := range ke.Runes {
		switch {
		case b == 'q' || b == 'Q':
			return func() mofu.Msg {
				os.Exit(0)
				return nil
			}
		case b == 'j' || ke.Key == mofu.KeyDown:
			a.count++
		case b == 'k' || ke.Key == mofu.KeyUp:
			a.count--
		}
	}
	return nil
}

func main() {
	_ = widgets.NewText("")
	app := &App{}
	app.Style().Foreground = mofu.Hex("ff69b4")

	p := mofu.New(app, mofu.WithTheme(mofu.MochiTheme()))
	if err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
