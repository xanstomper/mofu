package mofu

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type FilePicker struct {
	mu           sync.Mutex
	path         string
	dir          string
	files        []os.FileInfo
	selected     int
	offset       int
	height       int
	width        int
	allowedTypes []string
	selectedFile string
	styles       FilePickerStyles
	keyMap       *KeyMap
}

type FilePickerStyles struct {
	Selected Style
	Dir      Style
	File     Style
	Cursor   Style
}

func DefaultFilePickerStyles() FilePickerStyles {
	return FilePickerStyles{
		Selected: DefaultStyle().Fg(Hex("1e1e2e")).Bg(Hex("89b4fa")),
		Dir:      DefaultStyle().Fg(Hex("89b4fa")).WithAttrs(AttrBold),
		File:     DefaultStyle().Fg(Hex("cdd6f4")),
		Cursor:   DefaultStyle().Fg(Hex("f5c2e7")),
	}
}

func NewFilePicker() *FilePicker {
	fp := &FilePicker{
		dir:    ".",
		height: 10,
		width:  40,
		styles: DefaultFilePickerStyles(),
		keyMap: NewKeyMap(),
	}
	fp.keyMap.Set("up", NewBinding(KeyUp, HelpText{Key: "↑", Desc: "up"}))
	fp.keyMap.Set("down", NewBinding(KeyDown, HelpText{Key: "↓", Desc: "down"}))
	fp.keyMap.Set("enter", NewBinding(KeyEnter, HelpText{Key: "enter", Desc: "select"}))
	fp.keyMap.Set("back", NewBinding(KeyBack, HelpText{Key: "⌫", Desc: "parent dir"}))
	fp.loadDir()
	return fp
}

func (fp *FilePicker) SetSize(w, h int) {
	fp.mu.Lock()
	fp.width = w
	fp.height = h
	fp.mu.Unlock()
}

func (fp *FilePicker) SetAllowedTypes(types []string) {
	fp.mu.Lock()
	fp.allowedTypes = types
	fp.mu.Unlock()
}

func (fp *FilePicker) SelectedFile() string {
	fp.mu.Lock()
	defer fp.mu.Unlock()
	return fp.selectedFile
}

func (fp *FilePicker) Chosen() bool {
	fp.mu.Lock()
	defer fp.mu.Unlock()
	return fp.selectedFile != ""
}

func (fp *FilePicker) loadDir() {
	entries, err := os.ReadDir(fp.dir)
	if err != nil {
		fp.files = nil
		return
	}
	fp.files = nil
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		fp.files = append(fp.files, info)
	}
	sort.Slice(fp.files, func(i, j int) bool {
		if fp.files[i].IsDir() != fp.files[j].IsDir() {
			return fp.files[i].IsDir()
		}
		return strings.ToLower(fp.files[i].Name()) < strings.ToLower(fp.files[j].Name())
	})
	fp.selected = 0
	fp.offset = 0
}

func (fp *FilePicker) HandleEvent(e Event) {
	if e.Type != EventKeyPress {
		return
	}
	ke, ok := e.Data.(KeyEvent)
	if !ok {
		return
	}

	fp.mu.Lock()
	defer fp.mu.Unlock()

	switch ke.Key {
	case KeyUp:
		if fp.selected > 0 {
			fp.selected--
			if fp.selected < fp.offset {
				fp.offset = fp.selected
			}
		}
	case KeyDown:
		if fp.selected < len(fp.files)-1 {
			fp.selected++
			if fp.selected >= fp.offset+fp.height {
				fp.offset = fp.selected - fp.height + 1
			}
		}
	case KeyEnter:
		if fp.selected < len(fp.files) {
			f := fp.files[fp.selected]
			full := filepath.Join(fp.dir, f.Name())
			if f.IsDir() {
				fp.dir = full
				fp.loadDir()
			} else {
				if fp.isAllowed(f.Name()) {
					fp.selectedFile = full
				}
			}
		}
	case KeyBack:
		parent := filepath.Dir(fp.dir)
		if parent != fp.dir {
			fp.dir = parent
			fp.loadDir()
		}
	}
}

func (fp *FilePicker) isAllowed(name string) bool {
	if len(fp.allowedTypes) == 0 {
		return true
	}
	ext := strings.ToLower(filepath.Ext(name))
	for _, t := range fp.allowedTypes {
		if ext == t {
			return true
		}
	}
	return false
}

func (fp *FilePicker) Render() string {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	var out strings.Builder

	header := fp.styles.Dir.Apply(" " + fp.dir)
	out.WriteString(header + "\n")

	end := fp.offset + fp.height
	if end > len(fp.files) {
		end = len(fp.files)
	}

	for i := fp.offset; i < end; i++ {
		f := fp.files[i]
		selected := i == fp.selected

		icon := "  "
		if f.IsDir() {
			icon = fp.styles.Dir.Apply("📁")
		} else {
			icon = fp.styles.File.Apply("  ")
		}

		name := f.Name()
		if len(name) > fp.width-6 {
			name = name[:fp.width-9] + "..."
		}

		var line string
		if selected {
			line = fp.styles.Selected.Apply("▸ "+name)
		} else if f.IsDir() {
			line = fp.styles.Dir.Apply("  "+name+"/")
		} else {
			line = fp.styles.File.Apply("  "+name)
		}
		out.WriteString(icon + line + "\n")
	}

	remaining := fp.height - (end - fp.offset)
	for i := 0; i < remaining; i++ {
		out.WriteString("\n")
	}

	return out.String()
}
