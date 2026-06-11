package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/xanstomper/mofu"
)

type Note struct {
	Title   string
	Content string
	Modified bool
}

type Notepad struct {
	mofu.Minimal
	notes      []Note
	selected   int
	mode       string
	cursor     int
	scrollY    int
	width      int
	height     int
	statusMsg  string
}

func NewNotepad() *Notepad {
	return &Notepad{
		notes: []Note{
			{Title: "Welcome", Content: "# Welcome to MOFU Notepad\n\nThis is a simple note-taking app.\n\n## Controls\n- j/k: navigate notes\n- Enter: edit note\n- n: new note\n- d: delete note\n- /: search notes\n- q: quit", Modified: false},
			{Title: "TODO", Content: "- [ ] Build more gadgets\n- [ ] Write documentation\n- [x] Fix bugs\n- [ ] Add tests\n- [ ] Release v1.0", Modified: false},
			{Title: "Ideas", Content: "## Project Ideas\n\n1. Terminal dashboard for monitoring\n2. CLI tool for code review\n3. Real-time chat application\n4. Kanban board for task management", Modified: false},
		},
	}
}

func (n *Notepad) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	n.width = r.Width
	n.height = r.Height

	leftW := 20
	rightW := r.Width - leftW

	// Sidebar
	sidebarStyle := mofu.DefaultStyle().Fg(mofu.Hex("89b4fa")).WithAttrs(mofu.AttrBold)
	ctx.Renderer.WriteString(" Notes", r.X, r.Y, sidebarStyle.Foreground, sidebarStyle.Background, sidebarStyle.Attrs)

	for i, note := range n.notes {
		y := r.Y + 1 + i
		if y >= r.Y+r.Height-1 {
			break
		}

		style := mofu.DefaultStyle()
		prefix := "  "
		if i == n.selected {
			prefix = "▸ "
			style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
		}

		title := note.Title
		if len(title) > 14 {
			title = title[:11] + "..."
		}

		marker := ""
		if note.Modified {
			marker = "*"
		}

		ctx.Renderer.WriteString(fmt.Sprintf("%s%s%s", prefix, title, marker), r.X, y, style.Foreground, style.Background, style.Attrs)
	}

	ctx.Renderer.WriteString("│", r.X+leftW-1, r.Y, mofu.Hex("444444"), mofu.ColorBlack, 0)

	// Content pane
	if n.selected >= 0 && n.selected < len(n.notes) {
		note := n.notes[n.selected]
		titleStyle := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
		ctx.Renderer.WriteString(fmt.Sprintf(" %s", note.Title), r.X+leftW, r.Y, titleStyle.Foreground, titleStyle.Background, titleStyle.Attrs)

		lines := strings.Split(note.Content, "\n")
		start := n.scrollY
		if start > len(lines) {
			start = len(lines)
		}

		for i := start; i < len(lines) && i < start+r.Height-2; i++ {
			y := r.Y + 2 + (i - start)
			line := lines[i]
			if len(line) > rightW-2 {
				line = line[:rightW-5] + "..."
			}

			lineStyle := mofu.DefaultStyle()
			if strings.HasPrefix(strings.TrimSpace(line), "#") {
				lineStyle = mofu.DefaultStyle().Fg(mofu.Hex("89b4fa")).WithAttrs(mofu.AttrBold)
			} else if strings.HasPrefix(strings.TrimSpace(line), "-") {
				lineStyle = mofu.DefaultStyle().Fg(mofu.Hex("a6e3a1"))
			}

			ctx.Renderer.WriteString(line, r.X+leftW, y, lineStyle.Foreground, lineStyle.Background, lineStyle.Attrs)
		}
	}

	// Status
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, r.Y+r.Height-2, mofu.Hex("444444"), mofu.ColorBlack, 0)
	status := " j/k:navigate n:new d:delete /:search q:quit"
	if n.statusMsg != "" {
		status = " " + n.statusMsg
	}
	if len(status) > r.Width {
		status = status[:r.Width]
	}
	ctx.Renderer.WriteString(status, r.X, r.Y+r.Height-1, mofu.Hex("585b70"), mofu.Hex("1e1e2e"), 0)
}

func (n *Notepad) HandleEvent(event mofu.Event) mofu.Cmd {
	if event.Type != mofu.EventKeyPress {
		return nil
	}
	ke := event.Data.(mofu.KeyEvent)

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()

	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		if n.selected < len(n.notes)-1 {
			n.selected++
			n.scrollY = 0
		}

	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if n.selected > 0 {
			n.selected--
			n.scrollY = 0
		}

	case len(ke.Runes) > 0 && ke.Runes[0] == 'n':
		n.notes = append(n.notes[:n.selected+1], append([]Note{{Title: "Untitled", Content: "", Modified: true}}, n.notes[n.selected+1:]...)...)
		n.selected++
		n.statusMsg = "Created new note"

	case len(ke.Runes) > 0 && ke.Runes[0] == 'd' && len(n.notes) > 1:
		title := n.notes[n.selected].Title
		n.notes = append(n.notes[:n.selected], n.notes[n.selected+1:]...)
		if n.selected >= len(n.notes) {
			n.selected = len(n.notes) - 1
		}
		n.statusMsg = fmt.Sprintf("Deleted '%s'", title)

	case ke.Key == mofu.KeyPgDn:
		n.scrollY += n.height - 4
		if n.selected < len(n.notes) {
			lines := strings.Split(n.notes[n.selected].Content, "\n")
			if n.scrollY > len(lines) {
				n.scrollY = len(lines)
			}
		}

	case ke.Key == mofu.KeyPgUp:
		n.scrollY -= n.height - 4
		if n.scrollY < 0 {
			n.scrollY = 0
		}
	}

	return nil
}

func main() {
	app := NewNotepad()
	if err := mofu.Run(app); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
