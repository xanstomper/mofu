package mofu

import (
	"strings"
	"sync"
)

type Textarea struct {
	mu         sync.Mutex
	value      []rune
	lines      [][]rune
	cursorRow  int
	cursorCol  int
	maxWidth   int
	maxHeight  int
	focused    bool
	maxLines   int
	placeholder string
	showCursor bool
	style      Style
	focusStyle Style
	callback   func(string)
}

func NewTextarea() *Textarea {
	return &Textarea{
		lines:      [][]rune{{}},
		maxWidth:   40,
		maxHeight:  10,
		showCursor: true,
		style:      DefaultStyle().Fg(Hex("cdd6f4")).Bg(Hex("313244")),
		focusStyle: DefaultStyle().Fg(Hex("cdd6f4")).Bg(Hex("1e1e2e")),
	}
}

func (t *Textarea) SetMaxWidth(w int)  { t.mu.Lock(); t.maxWidth = w; t.mu.Unlock() }
func (t *Textarea) SetMaxHeight(h int) { t.mu.Lock(); t.maxHeight = h; t.mu.Unlock() }
func (t *Textarea) SetPlaceholder(s string) { t.mu.Lock(); t.placeholder = s; t.mu.Unlock() }
func (t *Textarea) Focus()            { t.mu.Lock(); t.focused = true; t.mu.Unlock() }
func (t *Textarea) Blur()             { t.mu.Lock(); t.focused = false; t.mu.Unlock() }
func (t *Textarea) Focused() bool     { t.mu.Lock(); defer t.mu.Unlock(); return t.focused }

func (t *Textarea) Value() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	var sb strings.Builder
	for i, line := range t.lines {
		sb.WriteString(string(line))
		if i < len(t.lines)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func (t *Textarea) SetValue(s string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.lines = make([][]rune, 0)
	for _, line := range strings.Split(s, "\n") {
		t.lines = append(t.lines, []rune(line))
	}
	if len(t.lines) == 0 {
		t.lines = [][]rune{{}}
	}
	t.cursorRow = 0
	t.cursorCol = 0
	t.value = []rune(s)
}

func (t *Textarea) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.lines = [][]rune{{}}
	t.cursorRow = 0
	t.cursorCol = 0
	t.value = nil
}

func (t *Textarea) OnChange(fn func(string)) {
	t.mu.Lock()
	t.callback = fn
	t.mu.Unlock()
}

func (t *Textarea) HandleEvent(e Event) {
	if !t.focused || e.Type != EventKeyPress {
		return
	}
	ke, ok := e.Data.(KeyEvent)
	if !ok {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	switch ke.Key {
	case KeyEnter:
		if t.maxLines > 0 && len(t.lines) >= t.maxLines {
			return
		}
		left := make([]rune, t.cursorCol)
		copy(left, t.lines[t.cursorRow][:t.cursorCol])
		right := make([]rune, len(t.lines[t.cursorRow])-t.cursorCol)
		copy(right, t.lines[t.cursorRow][t.cursorCol:])
		t.lines[t.cursorRow] = left
		newRow := make([]rune, 0)
		newRow = append(newRow, right...)
		t.lines = append(t.lines[:t.cursorRow+1], append([][]rune{newRow}, t.lines[t.cursorRow+1:]...)...)
		t.cursorRow++
		t.cursorCol = 0
	case KeyBack:
		if t.cursorCol > 0 {
			t.lines[t.cursorRow] = append(t.lines[t.cursorRow][:t.cursorCol-1], t.lines[t.cursorRow][t.cursorCol:]...)
			t.cursorCol--
		} else if t.cursorRow > 0 {
			prevLen := len(t.lines[t.cursorRow-1])
			t.lines[t.cursorRow-1] = append(t.lines[t.cursorRow-1], t.lines[t.cursorRow]...)
			t.lines = append(t.lines[:t.cursorRow], t.lines[t.cursorRow+1:]...)
			t.cursorRow--
			t.cursorCol = prevLen
		}
	case KeyDelete:
		if t.cursorCol < len(t.lines[t.cursorRow]) {
			t.lines[t.cursorRow] = append(t.lines[t.cursorRow][:t.cursorCol], t.lines[t.cursorRow][t.cursorCol+1:]...)
		} else if t.cursorRow < len(t.lines)-1 {
			t.lines[t.cursorRow] = append(t.lines[t.cursorRow], t.lines[t.cursorRow+1]...)
			t.lines = append(t.lines[:t.cursorRow+1], t.lines[t.cursorRow+2:]...)
		}
	case KeyLeft:
		if t.cursorCol > 0 {
			t.cursorCol--
		} else if t.cursorRow > 0 {
			t.cursorRow--
			t.cursorCol = len(t.lines[t.cursorRow])
		}
	case KeyRight:
		if t.cursorCol < len(t.lines[t.cursorRow]) {
			t.cursorCol++
		} else if t.cursorRow < len(t.lines)-1 {
			t.cursorRow++
			t.cursorCol = 0
		}
	case KeyUp:
		if t.cursorRow > 0 {
			t.cursorRow--
			if t.cursorCol > len(t.lines[t.cursorRow]) {
				t.cursorCol = len(t.lines[t.cursorRow])
			}
		}
	case KeyDown:
		if t.cursorRow < len(t.lines)-1 {
			t.cursorRow++
			if t.cursorCol > len(t.lines[t.cursorRow]) {
				t.cursorCol = len(t.lines[t.cursorRow])
			}
		}
	case KeyHome:
		t.cursorCol = 0
	case KeyEnd:
		t.cursorCol = len(t.lines[t.cursorRow])
	default:
		if len(ke.Runes) > 0 && !ke.Ctrl && !ke.Alt {
			insert := []rune(string(ke.Runes))
			newLine := make([]rune, 0, len(t.lines[t.cursorRow])+len(insert))
			newLine = append(newLine, t.lines[t.cursorRow][:t.cursorCol]...)
			newLine = append(newLine, insert...)
			newLine = append(newLine, t.lines[t.cursorRow][t.cursorCol:]...)
			t.lines[t.cursorRow] = newLine
			t.cursorCol += len(insert)
		}
	}

	if t.callback != nil {
		t.callback(t.Value())
	}
}

func (t *Textarea) Render() string {
	t.mu.Lock()
	defer t.mu.Unlock()

	st := t.style
	if t.focused {
		st = t.focusStyle
	}

	if len(t.lines) == 0 && t.placeholder != "" && !t.focused {
		return st.Apply(t.placeholder)
	}

	var out strings.Builder
	visibleRows := t.maxHeight
	if visibleRows <= 0 {
		visibleRows = len(t.lines)
	}

	for row := 0; row < visibleRows; row++ {
		if row >= len(t.lines) {
			out.WriteString("\n")
			continue
		}
		line := string(t.lines[row])
		if len(line) > t.maxWidth {
			line = line[:t.maxWidth]
		}

		if row == t.cursorRow && t.focused && t.showCursor {
			before := line
			if t.cursorCol <= len(before) {
				before = line[:t.cursorCol]
				cursor := ""
				if t.cursorCol < len(line) {
					cursor = st.Apply(string(line[t.cursorCol]))
				} else {
					cursor = st.Apply(" ")
				}
				after := ""
				if t.cursorCol+1 <= len(line) {
					after = line[t.cursorCol+1:]
				}
				line = st.Apply(before) + "\x1b[7m" + cursor + "\x1b[27m" + after
			}
		} else {
			line = st.Apply(line)
		}

		out.WriteString(line)
		if row < visibleRows-1 {
			out.WriteString("\n")
		}
	}

	return out.String()
}
