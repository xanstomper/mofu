package main

import (
	"fmt"
	"os"

	"github.com/xanstomper/mofu"
)

// Counter — the simplest MOFU example.
//
// Usage:
//
//	go run examples/counter/main.go
//
// Press j/k or arrow keys to change count, q to quit.

type Counter struct {
	mofu.Minimal
	count int
}

func (c *Counter) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	style := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))
	text := fmt.Sprintf("Count: %d\n\nPress j/k or arrows to change\nPress q to quit", c.count)
	ctx.Renderer.WriteStyledString(text, r.X, r.Y, style)
}

func (c *Counter) HandleEvent(event mofu.Event) mofu.Cmd {
	if event.Type != mofu.EventKeyPress {
		return nil
	}
	ke, ok := event.Data.(mofu.KeyEvent)
	if !ok {
		return nil
	}

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && (ke.Runes[0] == 'q' || ke.Runes[0] == 'Q')):
		return mofu.QuitCmd()
	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		c.count++
	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		c.count--
	}
	return nil
}

func main() {
	if err := mofu.Run(&Counter{}); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
