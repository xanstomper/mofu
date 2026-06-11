package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/xanstomper/mofu"
)

// Kanban — a kanban board example.

type Kanban struct {
	mofu.Minimal
	columns   []Column
	selected  int
	colIndex  int
	width     int
	height    int
}

type Column struct {
	Name  string
	Items []string
}

func NewKanban() *Kanban {
	return &Kanban{
		columns: []Column{
			{Name: "To Do", Items: []string{"Design UI", "Write docs", "Setup CI"}},
			{Name: "In Progress", Items: []string{"Build widgets", "Test API"}},
			{Name: "Done", Items: []string{"Setup repo", "Write README"}},
		},
	}
}

func (k *Kanban) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	k.width = r.Width
	k.height = r.Height

	titleStyle := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
	ctx.Renderer.WriteString(" Kanban Board", r.X, r.Y, titleStyle.Foreground, titleStyle.Background, titleStyle.Attrs)

	sep := strings.Repeat("─", r.Width-2)
	ctx.Renderer.WriteString(sep, r.X+1, r.Y+1, mofu.Hex("444444"), mofu.ColorBlack, 0)

	colWidth := (r.Width - 4) / len(k.columns)
	y := r.Y + 3

	for ci, col := range k.columns {
		x := r.X + 2 + ci*colWidth

		// Column header
		headerStyle := mofu.DefaultStyle().Fg(mofu.Hex("89b4fa")).WithAttrs(mofu.AttrBold)
		ctx.Renderer.WriteString(fmt.Sprintf(" %s (%d) ", col.Name, len(col.Items)), x, y, headerStyle.Foreground, headerStyle.Background, headerStyle.Attrs)

		// Items
		for i, item := range col.Items {
			if y+2+i >= r.Y+r.Height-2 {
				break
			}
			style := mofu.DefaultStyle().Fg(mofu.Hex("e0e0e0"))
			if ci == k.colIndex && i == k.selected {
				style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
				ctx.Renderer.WriteString(strings.Repeat(" ", colWidth-2), x, y+2+i, mofu.ColorWhite, mofu.Hex("2a2a3e"), 0)
			}
			if len(item) > colWidth-4 {
				item = item[:colWidth-7] + "..."
			}
			ctx.Renderer.WriteString(" "+item+" ", x, y+2+i, style.Foreground, style.Background, style.Attrs)
		}
	}

	// Status
	ctx.Renderer.WriteString(" ←/→: Column  ↑/↓: Item  a: Add  d: Delete  m: Move →  q: Quit", r.X, r.Y+r.Height-1, mofu.Hex("666666"), mofu.ColorBlack, 0)
}

func (k *Kanban) HandleEvent(event mofu.Event) mofu.Cmd {
	if event.Type != mofu.EventKeyPress {
		return nil
	}
	ke := event.Data.(mofu.KeyEvent)

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && (ke.Runes[0] == 'q' || ke.Runes[0] == 'Q')):
		return mofu.QuitCmd()

	case ke.Key == mofu.KeyLeft || (len(ke.Runes) > 0 && ke.Runes[0] == 'h'):
		if k.colIndex > 0 {
			k.colIndex--
			k.selected = 0
		}

	case ke.Key == mofu.KeyRight || (len(ke.Runes) > 0 && ke.Runes[0] == 'l'):
		if k.colIndex < len(k.columns)-1 {
			k.colIndex++
			k.selected = 0
		}

	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if k.selected > 0 {
			k.selected--
		}

	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		if k.selected < len(k.columns[k.colIndex].Items)-1 {
			k.selected++
		}

	case len(ke.Runes) > 0 && ke.Runes[0] == 'a':
		col := &k.columns[k.colIndex]
		col.Items = append(col.Items, fmt.Sprintf("Item %d", len(col.Items)+1))

	case len(ke.Runes) > 0 && ke.Runes[0] == 'd':
		col := &k.columns[k.colIndex]
		if k.selected >= 0 && k.selected < len(col.Items) {
			col.Items = append(col.Items[:k.selected], col.Items[k.selected+1:]...)
			if k.selected >= len(col.Items) {
				k.selected = len(col.Items) - 1
			}
		}

	case len(ke.Runes) > 0 && ke.Runes[0] == 'm':
		if k.colIndex < len(k.columns)-1 {
			col := &k.columns[k.colIndex]
			if k.selected >= 0 && k.selected < len(col.Items) {
				item := col.Items[k.selected]
				col.Items = append(col.Items[:k.selected], col.Items[k.selected+1:]...)
				k.columns[k.colIndex+1].Items = append(k.columns[k.colIndex+1].Items, item)
				if k.selected >= len(col.Items) {
					k.selected = len(col.Items) - 1
				}
			}
		}
	}

	return nil
}

func main() {
	app := NewKanban()
	if err := mofu.Run(app); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
