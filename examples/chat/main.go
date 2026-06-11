package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/xanstomper/mofu"
	"github.com/xanstomper/mofu/widgets"
)

// Chat — a simple chat interface example.

type Chat struct {
	mofu.Minimal
	messages []string
	input    *widgets.InputNode
	width    int
	height   int
}

func NewChat() *Chat {
	c := &Chat{
		messages: []string{
			"Welcome to MOFU Chat!",
			"Type a message and press Enter.",
			"",
		},
		input: widgets.NewInput(),
	}
	c.input.Placeholder = "Type a message..."
	c.input.OnSubmit = func(value string) mofu.Cmd {
		if value == "" {
			return nil
		}
		c.messages = append(c.messages, "You: "+value)
		c.messages = append(c.messages, "Bot: I received your message!")
		c.input.SetValue("")
		return nil
	}
	return c
}

func (c *Chat) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	c.width = r.Width
	c.height = r.Height

	titleStyle := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
	ctx.Renderer.WriteString(" MOFU Chat", r.X, r.Y, titleStyle.Foreground, titleStyle.Background, titleStyle.Attrs)

	sep := strings.Repeat("─", r.Width-2)
	ctx.Renderer.WriteString(sep, r.X+1, r.Y+1, mofu.Hex("444444"), mofu.ColorBlack, 0)

	// Messages
	msgY := r.Y + 2
	msgH := r.Height - 5
	start := 0
	if len(c.messages) > msgH {
		start = len(c.messages) - msgH
	}
	for i := start; i < len(c.messages); i++ {
		if msgY >= r.Y+r.Height-3 {
			break
		}
		msg := c.messages[i]
		if len(msg) > r.Width-2 {
			msg = msg[:r.Width-5] + "..."
		}
		style := mofu.DefaultStyle().Fg(mofu.Hex("e0e0e0"))
		if strings.HasPrefix(msg, "You:") {
			style = mofu.DefaultStyle().Fg(mofu.Hex("89b4fa"))
		} else if strings.HasPrefix(msg, "Bot:") {
			style = mofu.DefaultStyle().Fg(mofu.Hex("a6e3a1"))
		}
		ctx.Renderer.WriteString(msg, r.X+1, msgY, style.Foreground, style.Background, style.Attrs)
		msgY++
	}

	// Input
	c.input.SetBounds(mofu.Rect{X: r.X, Y: r.Y + r.Height - 2, Width: r.Width, Height: 1})
	c.input.Render(ctx)

	// Status
	ctx.Renderer.WriteString(" Enter: Send │ Esc: Quit", r.X, r.Y+r.Height-1, mofu.Hex("666666"), mofu.ColorBlack, 0)
}

func (c *Chat) HandleEvent(event mofu.Event) mofu.Cmd {
	if event.Type == mofu.EventKeyPress {
		ke, ok := event.Data.(mofu.KeyEvent)
		if ok && ke.Key == mofu.KeyEsc {
			return mofu.QuitCmd()
		}
	}
	return c.input.HandleEvent(event)
}

func main() {
	app := NewChat()
	if err := mofu.Run(app); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
