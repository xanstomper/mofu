package main

import (
	"fmt"
	"os"

	"github.com/anomalyco/mofu"
	"github.com/anomalyco/mofu/widgets"
)

// Counter is a simple counter component.
type Counter struct {
	count int
}

func (c *Counter) Render() string {
	return fmt.Sprintf("Count: %d\n\nPress 'q' to quit, 'j'/'k' or up/down to change", c.count)
}

func (c *Counter) HandleEvent(msg mofu.Msg) mofu.Cmd {
	switch msg := msg.(type) {
	case mofu.KeyPressMsg:
		for _, b := range msg.Runes {
			switch {
			case b == 'q' || b == 'Q':
				return func() mofu.Msg {
					os.Exit(0)
					return nil
				}
			case b == 'j' || msg.Key == mofu.KeyDown:
				c.count++
			case b == 'k' || msg.Key == mofu.KeyUp:
				c.count--
			}
		}
	}
	return nil
}

func (c *Counter) Mount() mofu.Cmd { return nil }
func (c *Counter) Unmount()        {}

func main() {
	// Build a simple app with a boxed counter and a list
	items := []widgets.ListItem{
		{Title: "Item 1", Subtitle: "first"},
		{Title: "Item 2", Subtitle: "second"},
		{Title: "Item 3", Subtitle: "third"},
	}

	list := widgets.NewList(items)
	_ = list

	app := mofu.New(&Counter{}, mofu.WithTheme(mofu.MochiTheme()))
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
