package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/xanstomper/mofu"
)

// MarkdownViewer — a markdown rendering example

type MarkdownViewer struct {
	mofu.Minimal
	content    string
	scrollY    int
	width      int
	height     int
	title      string
}

func NewMarkdownViewer(content string) *MarkdownViewer {
	return &MarkdownViewer{content: content}
}

func (m *MarkdownViewer) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	m.width = r.Width
	m.height = r.Height

	// Title
	titleStyle := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
	ctx.Renderer.WriteString(" Markdown Viewer", r.X, r.Y, titleStyle.Foreground, titleStyle.Background, titleStyle.Attrs)

	sep := strings.Repeat("─", r.Width-2)
	ctx.Renderer.WriteString(sep, r.X+1, r.Y+1, mofu.Hex("444444"), mofu.ColorBlack, 0)

	// Render markdown
	lines := strings.Split(m.content, "\n")
	y := r.Y + 2

	for i := m.scrollY; i < len(lines) && y < r.Y+r.Height-2; i++ {
		line := lines[i]
		style := mofu.DefaultStyle()

		// Headers
		if strings.HasPrefix(line, "# ") {
			style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
			line = line[2:]
		} else if strings.HasPrefix(line, "## ") {
			style = mofu.DefaultStyle().Fg(mofu.Hex("89b4fa")).WithAttrs(mofu.AttrBold)
			line = line[3:]
		} else if strings.HasPrefix(line, "### ") {
			style = mofu.DefaultStyle().Fg(mofu.Hex("94e2d5")).WithAttrs(mofu.AttrBold)
			line = line[4:]
		}

		// Bold
		if strings.HasPrefix(line, "**") && strings.HasSuffix(line, "**") {
			style = mofu.DefaultStyle().WithAttrs(mofu.AttrBold)
			line = line[2 : len(line)-2]
		}

		// Lists
		if strings.HasPrefix(line, "- ") {
			line = "  • " + line[2:]
		} else if strings.HasPrefix(line, "* ") {
			line = "  • " + line[2:]
		}

		// Code blocks
		if strings.HasPrefix(line, "```") {
			style = mofu.DefaultStyle().Fg(mofu.Hex("a6e3a1"))
		}

		// Blockquotes
		if strings.HasPrefix(line, "> ") {
			style = mofu.DefaultStyle().Fg(mofu.Hex("6c7086"))
			line = "  │ " + line[2:]
		}

		// Truncate if too long
		if len(line) > r.Width-2 {
			line = line[:r.Width-5] + "..."
		}

		ctx.Renderer.WriteString(line, r.X+1, y, style.Foreground, style.Background, style.Attrs)
		y++
	}

	// Scroll indicator
	if len(lines) > r.Height-2 {
		scrollPct := float64(m.scrollY) / float64(len(lines)-r.Height+2) * 100
		ctx.Renderer.WriteString(fmt.Sprintf(" Line %d/%d (%.0f%%)", m.scrollY+1, len(lines), scrollPct), r.X, r.Y+r.Height-1, mofu.Hex("666666"), mofu.ColorBlack, 0)
	}
}

func (m *MarkdownViewer) HandleEvent(event mofu.Event) mofu.Cmd {
	if event.Type != mofu.EventKeyPress {
		return nil
	}
	ke := event.Data.(mofu.KeyEvent)

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()

	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		lines := strings.Split(m.content, "\n")
		if m.scrollY < len(lines)-m.height+2 {
			m.scrollY++
		}

	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if m.scrollY > 0 {
			m.scrollY--
		}

	case ke.Key == mofu.KeyPgDn:
		lines := strings.Split(m.content, "\n")
		m.scrollY += m.height - 4
		if m.scrollY > len(lines)-m.height+2 {
			m.scrollY = len(lines) - m.height + 2
		}

	case ke.Key == mofu.KeyPgUp:
		m.scrollY -= m.height - 4
		if m.scrollY < 0 {
			m.scrollY = 0
		}

	case ke.Key == mofu.KeyHome:
		m.scrollY = 0

	case ke.Key == mofu.KeyEnd:
		lines := strings.Split(m.content, "\n")
		m.scrollY = len(lines) - m.height + 2
		if m.scrollY < 0 {
			m.scrollY = 0
		}
	}

	return nil
}

func main() {
	content := "# MOFU Documentation\n\n## Getting Started\n\nMOFU is a reactive terminal UI runtime for Go.\n\n### Installation\n\n```bash\ngo get github.com/xanstomper/mofu\n```\n\n### Basic Usage\n\n```go\npackage main\n\nimport \"github.com/xanstomper/mofu\"\n\ntype App struct {\n    mofu.Minimal\n    count int\n}\n\nfunc (a *App) Render(ctx *mofu.RenderContext) {\n    // Draw your UI\n}\n\nfunc main() {\n    mofu.Run(&App{})\n}\n```\n\n## Features\n\n- Reactive state graph\n- Incremental rendering\n- Spring physics animations\n- 95 gadgets\n- Semantic styling\n\n## Architecture\n\nMOFU uses a streaming-first architecture:\n\n1. Input streams are parsed\n2. State graph is updated\n3. Dirty nodes are propagated\n4. Layout is computed\n5. Tree is diffed\n6. Changed cells are rendered\n\n## Performance\n\nMOFU achieves:\n\n- 124ns state updates\n- 52ns dirty tracking\n- 0 allocations per frame\n- 60fps rendering\n\n## Conclusion\n\nMOFU is the future of terminal UI development.\n\n> The best framework is the one that gets out of your way and lets you build."

	app := NewMarkdownViewer(content)
	if err := mofu.Run(app); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
