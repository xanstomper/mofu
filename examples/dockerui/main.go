package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/xanstomper/mofu"
)

// DockerUI — a minimal docker interface example.

type DockerUI struct {
	mofu.Minimal
	view       int // 0=containers, 1=images
	containers []Container
	images     []Image
	selected   int
	width      int
	height     int
}

type Container struct {
	ID    string
	Name  string
	State string
	Ports string
}

type Image struct {
	ID   string
	Name string
	Size string
}

func NewDockerUI() *DockerUI {
	d := &DockerUI{}
	d.loadContainers()
	return d
}

func (d *DockerUI) loadContainers() {
	out, err := exec.Command("docker", "ps", "-a", "--format", "{{.ID}}\t{{.Names}}\t{{.State}}\t{{.Ports}}").Output()
	if err != nil {
		d.containers = []Container{{Name: "Docker not available"}}
		return
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	d.containers = nil
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) >= 3 {
			d.containers = append(d.containers, Container{
				ID:    parts[0],
				Name:  parts[1],
				State: parts[2],
				Ports: strings.Join(parts[3:], " "),
			})
		}
	}
	if len(d.containers) == 0 {
		d.containers = []Container{{Name: "No containers"}}
	}
}

func (d *DockerUI) loadImages() {
	out, err := exec.Command("docker", "images", "--format", "{{.ID}}\t{{.Repository}}\t{{.Size}}").Output()
	if err != nil {
		d.images = []Image{{Name: "Docker not available"}}
		return
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	d.images = nil
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) >= 2 {
			d.images = append(d.images, Image{
				ID:   parts[0],
				Name: parts[1],
				Size: strings.Join(parts[2:], " "),
			})
		}
	}
	if len(d.images) == 0 {
		d.images = []Image{{Name: "No images"}}
	}
}

func (d *DockerUI) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	d.width = r.Width
	d.height = r.Height

	titleStyle := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
	ctx.Renderer.WriteString(" Docker UI", r.X, r.Y, titleStyle.Foreground, titleStyle.Background, titleStyle.Attrs)

	sep := strings.Repeat("─", r.Width-2)
	ctx.Renderer.WriteString(sep, r.X+1, r.Y+1, mofu.Hex("444444"), mofu.ColorBlack, 0)

	// Tab bar
	tabs := []string{" Containers ", " Images "}
	for i, tab := range tabs {
		style := mofu.DefaultStyle().Fg(mofu.Hex("666666"))
		if i == d.view {
			style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
		}
		ctx.Renderer.WriteString(tab, r.X+2+i*15, r.Y+1, style.Foreground, style.Background, style.Attrs)
	}

	y := r.Y + 3
	switch d.view {
	case 0: // Containers
		// Header
		headerStyle := mofu.DefaultStyle().Fg(mofu.Hex("89b4fa")).WithAttrs(mofu.AttrBold)
		ctx.Renderer.WriteString("  ID        NAME          STATE     PORTS", r.X+1, y, headerStyle.Foreground, headerStyle.Background, headerStyle.Attrs)
		y++

		for i, c := range d.containers {
			if y+i >= r.Y+r.Height-2 {
				break
			}
			style := mofu.DefaultStyle()
			if i == d.selected {
				style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))
				ctx.Renderer.WriteString(strings.Repeat(" ", r.Width-2), r.X+1, y+i, mofu.ColorWhite, mofu.Hex("2a2a3e"), 0)
			}

			stateColor := mofu.Hex("a6e3a1")
			if c.State != "running" {
				stateColor = mofu.Hex("f38ba8")
			}

			id := c.ID
			if len(id) > 12 {
				id = id[:12]
			}
			name := c.Name
			if len(name) > 13 {
				name = name[:10] + "..."
			}

			ctx.Renderer.WriteString(fmt.Sprintf("  %-10s %-13s", id, name), r.X+1, y+i, style.Foreground, style.Background, style.Attrs)
			ctx.Renderer.WriteString(c.State, r.X+25, y+i, stateColor, mofu.ColorBlack, 0)
		}

	case 1: // Images
		headerStyle := mofu.DefaultStyle().Fg(mofu.Hex("89b4fa")).WithAttrs(mofu.AttrBold)
		ctx.Renderer.WriteString("  ID          NAME                    SIZE", r.X+1, y, headerStyle.Foreground, headerStyle.Background, headerStyle.Attrs)
		y++

		for i, img := range d.images {
			if y+i >= r.Y+r.Height-2 {
				break
			}
			style := mofu.DefaultStyle()
			if i == d.selected {
				style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))
				ctx.Renderer.WriteString(strings.Repeat(" ", r.Width-2), r.X+1, y+i, mofu.ColorWhite, mofu.Hex("2a2a3e"), 0)
			}

			id := img.ID
			if len(id) > 12 {
				id = id[:12]
			}
			name := img.Name
			if len(name) > 23 {
				name = name[:20] + "..."
			}

			ctx.Renderer.WriteString(fmt.Sprintf("  %-12s %-23s %s", id, name, img.Size), r.X+1, y+i, style.Foreground, style.Background, style.Attrs)
		}
	}

	// Status bar
	ctx.Renderer.WriteString(" 1/2: Switch view │ j/k: Navigate │ r: Refresh │ q: Quit", r.X, r.Y+r.Height-1, mofu.Hex("666666"), mofu.ColorBlack, 0)
}

func (d *DockerUI) HandleEvent(event mofu.Event) mofu.Cmd {
	if event.Type != mofu.EventKeyPress {
		return nil
	}
	ke := event.Data.(mofu.KeyEvent)

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && (ke.Runes[0] == 'q' || ke.Runes[0] == 'Q')):
		return mofu.QuitCmd()
	case len(ke.Runes) > 0 && ke.Runes[0] == '1':
		d.view = 0
		d.selected = 0
		d.loadContainers()
	case len(ke.Runes) > 0 && ke.Runes[0] == '2':
		d.view = 1
		d.selected = 0
		d.loadImages()
	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		d.selected++
		d.clamp()
	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		d.selected--
		d.clamp()
	case len(ke.Runes) > 0 && (ke.Runes[0] == 'r' || ke.Runes[0] == 'R'):
		d.loadContainers()
		d.loadImages()
	}
	return nil
}

func (d *DockerUI) clamp() {
	max := 0
	if d.view == 0 {
		max = len(d.containers) - 1
	} else {
		max = len(d.images) - 1
	}
	if d.selected < 0 {
		d.selected = 0
	}
	if d.selected > max {
		d.selected = max
	}
}

func main() {
	app := NewDockerUI()
	if err := mofu.Run(app); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
