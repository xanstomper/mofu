package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/xanstomper/mofu"
)

// GitUI — a minimal git interface example.

type GitUI struct {
	mofu.Minimal
	view    int // 0=status, 1=log, 2=diff
	status  []string
	log     []string
	diff    string
	width   int
	height  int
}

func NewGitUI() *GitUI {
	g := &GitUI{}
	g.loadStatus()
	return g
}

func (g *GitUI) loadStatus() {
	out, err := exec.Command("git", "status", "--short").Output()
	if err != nil {
		g.status = []string{"Not a git repository"}
		return
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		g.status = []string{"Nothing to commit"}
	} else {
		g.status = lines
	}
}

func (g *GitUI) loadLog() {
	out, err := exec.Command("git", "log", "--oneline", "-20").Output()
	if err != nil {
		g.log = []string{"No commits yet"}
		return
	}
	g.log = strings.Split(strings.TrimSpace(string(out)), "\n")
}

func (g *GitUI) loadDiff() {
	out, err := exec.Command("git", "diff").Output()
	if err != nil {
		g.diff = "No changes"
		return
	}
	g.diff = string(out)
	if g.diff == "" {
		g.diff = "No changes"
	}
}

func (g *GitUI) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	g.width = r.Width
	g.height = r.Height

	titleStyle := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
	ctx.Renderer.WriteString(" Git UI", r.X, r.Y, titleStyle.Foreground, titleStyle.Background, titleStyle.Attrs)

	sep := strings.Repeat("─", r.Width-2)
	ctx.Renderer.WriteString(sep, r.X+1, r.Y+1, mofu.Hex("444444"), mofu.ColorBlack, 0)

	// Tab bar
	tabs := []string{" Status ", " Log ", " Diff "}
	for i, tab := range tabs {
		style := mofu.DefaultStyle().Fg(mofu.Hex("666666"))
		if i == g.view {
			style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
		}
		ctx.Renderer.WriteString(tab, r.X+2+i*10, r.Y+1, style.Foreground, style.Background, style.Attrs)
	}

	y := r.Y + 3
	switch g.view {
	case 0: // Status
		if len(g.status) == 0 {
			ctx.Renderer.WriteString("  Clean working tree", r.X+2, y, mofu.Hex("a6e3a1"), mofu.ColorBlack, 0)
		} else {
			for i, line := range g.status {
				if y+i >= r.Y+r.Height-2 {
					break
				}
				style := mofu.DefaultStyle()
				if strings.HasPrefix(line, "M") || strings.HasPrefix(line, "A") {
					style = mofu.DefaultStyle().Fg(mofu.Hex("a6e3a1"))
				} else if strings.HasPrefix(line, "D") {
					style = mofu.DefaultStyle().Fg(mofu.Hex("f38ba8"))
				}
				ctx.Renderer.WriteString("  "+line, r.X+2, y+i, style.Foreground, style.Background, style.Attrs)
			}
		}
	case 1: // Log
		for i, line := range g.log {
			if y+i >= r.Y+r.Height-2 {
				break
			}
			ctx.Renderer.WriteString("  "+line, r.X+2, y+i, mofu.Hex("e0e0e0"), mofu.ColorBlack, 0)
		}
	case 2: // Diff
		lines := strings.Split(g.diff, "\n")
		for i, line := range lines {
			if y+i >= r.Y+r.Height-2 {
				break
			}
			style := mofu.DefaultStyle()
			if strings.HasPrefix(line, "+") {
				style = mofu.DefaultStyle().Fg(mofu.Hex("a6e3a1"))
			} else if strings.HasPrefix(line, "-") {
				style = mofu.DefaultStyle().Fg(mofu.Hex("f38ba8"))
			}
			ctx.Renderer.WriteString("  "+line, r.X+2, y+i, style.Foreground, style.Background, style.Attrs)
		}
	}

	// Status bar
	ctx.Renderer.WriteString(" 1/2/3: Switch view │ r: Refresh │ q: Quit", r.X, r.Y+r.Height-1, mofu.Hex("666666"), mofu.ColorBlack, 0)
}

func (g *GitUI) HandleEvent(event mofu.Event) mofu.Cmd {
	if event.Type != mofu.EventKeyPress {
		return nil
	}
	ke := event.Data.(mofu.KeyEvent)

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && (ke.Runes[0] == 'q' || ke.Runes[0] == 'Q')):
		return mofu.QuitCmd()
	case len(ke.Runes) > 0 && ke.Runes[0] == '1':
		g.view = 0
		g.loadStatus()
	case len(ke.Runes) > 0 && ke.Runes[0] == '2':
		g.view = 1
		g.loadLog()
	case len(ke.Runes) > 0 && ke.Runes[0] == '3':
		g.view = 2
		g.loadDiff()
	case len(ke.Runes) > 0 && (ke.Runes[0] == 'r' || ke.Runes[0] == 'R'):
		g.loadStatus()
		g.loadLog()
		g.loadDiff()
	}
	return nil
}

func main() {
	app := NewGitUI()
	if err := mofu.Run(app); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
