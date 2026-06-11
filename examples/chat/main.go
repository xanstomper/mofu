package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/xanstomper/mofu"
)

type Chat struct {
	mofu.Minimal
	messages []string
	input    string
	width    int
	height   int
}

func NewChat() *Chat {
	return &Chat{
		messages: []string{
			"Welcome to MOFU Chat!",
			"Type a message and press Enter.",
			"",
		},
	}
}

func (c *Chat) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	c.width = r.Width
	c.height = r.Height

	titleStyle := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
	ctx.Renderer.WriteString(" Chat", r.X, r.Y, titleStyle.Foreground, titleStyle.Background, titleStyle.Attrs)
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, r.Y+1, mofu.Hex("444444"), mofu.ColorBlack, 0)

	y := r.Y + 2
	for _, msg := range c.messages {
		if y >= r.Y+r.Height-2 {
			break
		}
		color := mofu.Hex("a6e3a1")
		if strings.HasPrefix(msg, "Bot:") {
			color = mofu.Hex("89b4fa")
		}
		ctx.Renderer.WriteString(msg, r.X+1, y, color, mofu.ColorBlack, 0)
		y++
	}

	input := "> " + c.input
	if len(input) > r.Width-1 {
		input = input[:r.Width-1]
	}
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, r.Y+r.Height-2, mofu.Hex("444444"), mofu.ColorBlack, 0)
	ctx.Renderer.WriteString(input, r.X, r.Y+r.Height-1, mofu.Hex("cdd6f4"), mofu.Hex("1e1e2e"), 0)
}

func (c *Chat) HandleEvent(event mofu.Event) mofu.Cmd {
	if event.Type != mofu.EventKeyPress {
		return nil
	}
	ke := event.Data.(mofu.KeyEvent)

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()

	case ke.Key == mofu.KeyEnter && len(c.input) > 0:
		c.messages = append(c.messages, "You: "+c.input)
		c.messages = append(c.messages, "Bot: I received your message!")
		c.input = ""

	case ke.Key == mofu.KeyBack && len(c.input) > 0:
		c.input = c.input[:len(c.input)-1]

	default:
		if len(ke.Runes) > 0 {
			c.input += string(ke.Runes)
		}
	}
	return nil
}

func main() {
	app := NewChat()
	if err := mofu.Run(app); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
