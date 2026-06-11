package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xanstomper/mofu"
)

// FileManager — a simple file browser example.

type FileManager struct {
	mofu.Minimal
	files      []string
	selected   int
	offset     int
	dir        string
	width      int
	height     int
}

func NewFileManager(dir string) *FileManager {
	fm := &FileManager{dir: dir}
	fm.loadDir()
	return fm
}

func (fm *FileManager) loadDir() {
	entries, err := os.ReadDir(fm.dir)
	if err != nil {
		fm.files = []string{"Error reading directory"}
		return
	}

	fm.files = []string{".."}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			name += "/"
		}
		fm.files = append(fm.files, name)
	}
	fm.selected = 0
	fm.offset = 0
}

func (fm *FileManager) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	fm.width = r.Width
	fm.height = r.Height

	// Header
	titleStyle := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
	ctx.Renderer.WriteString(" File Manager", r.X, r.Y, titleStyle.Foreground, titleStyle.Background, titleStyle.Attrs)

	// Current directory
	dirStyle := mofu.DefaultStyle().Fg(mofu.Hex("89b4fa"))
	ctx.Renderer.WriteString(" "+fm.dir, r.X, r.Y+1, dirStyle.Foreground, dirStyle.Background, dirStyle.Attrs)

	// Separator
	sep := strings.Repeat("─", r.Width-2)
	ctx.Renderer.WriteString(sep, r.X+1, r.Y+2, mofu.Hex("444444"), mofu.ColorBlack, 0)

	// File list
	listY := r.Y + 3
	listH := r.Height - 5
	start := fm.offset
	if start < 0 {
		start = 0
	}
	end := start + listH
	if end > len(fm.files) {
		end = len(fm.files)
	}

	for i := start; i < end; i++ {
		y := listY + (i - start)
		if y >= r.Y+r.Height-2 {
			break
		}

		file := fm.files[i]
		if len(file) > r.Width-4 {
			file = file[:r.Width-7] + "..."
		}

		style := mofu.DefaultStyle().Fg(mofu.Hex("e0e0e0"))
		prefix := "  "
		if i == fm.selected {
			prefix = "▸ "
			style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
			ctx.Renderer.WriteString(strings.Repeat(" ", r.Width-2), r.X+1, y, mofu.ColorWhite, mofu.Hex("2a2a3e"), 0)
		}

		if strings.HasSuffix(file, "/") {
			style = mofu.DefaultStyle().Fg(mofu.Hex("89b4fa"))
		}

		ctx.Renderer.WriteString(prefix+file, r.X+1, y, style.Foreground, style.Background, style.Attrs)
	}

	// Status bar
	status := fmt.Sprintf(" %d items │ j/k: Navigate │ Enter: Open │ q: Quit", len(fm.files))
	ctx.Renderer.WriteString(status, r.X, r.Y+r.Height-1, mofu.Hex("666666"), mofu.ColorBlack, 0)
}

func (fm *FileManager) HandleEvent(event mofu.Event) mofu.Cmd {
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
		if fm.selected < len(fm.files)-1 {
			fm.selected++
			fm.clamp()
		}

	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if fm.selected > 0 {
			fm.selected--
			fm.clamp()
		}

	case ke.Key == mofu.KeyEnter || (len(ke.Runes) > 0 && ke.Runes[0] == 'l'):
		fm.openSelected()
	}
	return nil
}

func (fm *FileManager) openSelected() {
	if fm.selected < 0 || fm.selected >= len(fm.files) {
		return
	}
	file := fm.files[fm.selected]
	path := filepath.Join(fm.dir, file)

	if file == ".." {
		fm.dir = filepath.Dir(fm.dir)
		fm.loadDir()
		return
	}

	info, err := os.Stat(path)
	if err != nil {
		return
	}
	if info.IsDir() {
		fm.dir = path
		fm.loadDir()
	}
}

func (fm *FileManager) clamp() {
	listH := fm.height - 5
	if listH < 1 {
		listH = 1
	}
	if fm.selected < fm.offset {
		fm.offset = fm.selected
	}
	if fm.selected >= fm.offset+listH {
		fm.offset = fm.selected - listH + 1
	}
	if fm.offset < 0 {
		fm.offset = 0
	}
}

func (fm *FileManager) Mount() mofu.Cmd { return nil }
func (fm *FileManager) Unmount()        {}

func main() {
	dir := "."
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	app := NewFileManager(dir)
	if err := mofu.Run(app); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
