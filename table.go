package mofu

import (
	"fmt"
	"strings"
	"sync"
)

type TableColumn struct {
	Title string
	Width int
	Align Align
}

type Table struct {
	mu         sync.Mutex
	columns    []TableColumn
	rows       [][]string
	cursor     int
	selected   int
	width      int
	height     int
	focused    bool
	keyMap     *KeyMap
	styles     TableStyles
	sortCol    int
	sortAsc    bool
	onSelect   func(int, []string)
	prefix     string
}

type TableStyles struct {
	Header      Style
	Cell        Style
	CellFocused Style
	CellSelect  Style
	Border      Style
	RowNum      Style
}

func DefaultTableStyles() TableStyles {
	return TableStyles{
		Header:      DefaultStyle().Fg(Hex("89b4fa")).WithAttrs(AttrBold),
		Cell:        DefaultStyle().Fg(Hex("cdd6f4")),
		CellFocused: DefaultStyle().Fg(Hex("cdd6f4")).Bg(Hex("313244")),
		CellSelect:  DefaultStyle().Fg(Hex("1e1e2e")).Bg(Hex("89b4fa")),
		Border:      DefaultStyle().Fg(Hex("45475a")),
		RowNum:      DefaultStyle().Fg(Hex("585b70")),
	}
}

func NewTable(columns []TableColumn, rows [][]string) *Table {
	t := &Table{
		columns: columns,
		rows:    rows,
		keyMap:  NewKeyMap(),
		styles:  DefaultTableStyles(),
	}
	t.keyMap.Set("up", NewBinding(KeyUp, HelpText{Key: "↑", Desc: "up"}))
	t.keyMap.Set("down", NewBinding(KeyDown, HelpText{Key: "↓", Desc: "down"}))
	t.keyMap.Set("pgup", NewBinding(KeyPgUp, HelpText{Key: "pgup", Desc: "page up"}))
	t.keyMap.Set("pgdown", NewBinding(KeyPgDn, HelpText{Key: "pgdn", Desc: "page down"}))
	t.keyMap.Set("enter", NewBinding(KeyEnter, HelpText{Key: "enter", Desc: "select"}))
	t.keyMap.Set("home", NewBinding(KeyHome, HelpText{Key: "home", Desc: "first"}))
	t.keyMap.Set("end", NewBinding(KeyEnd, HelpText{Key: "end", Desc: "last"}))
	return t
}

func (t *Table) SetSize(w, h int) { t.mu.Lock(); t.width = w; t.height = h; t.mu.Unlock() }
func (t *Table) Focus()           { t.mu.Lock(); t.focused = true; t.mu.Unlock() }
func (t *Table) Blur()            { t.mu.Lock(); t.focused = false; t.mu.Unlock() }
func (t *Table) OnSelect(fn func(int, []string)) { t.mu.Lock(); t.onSelect = fn; t.mu.Unlock() }

func (t *Table) SetRows(rows [][]string) {
	t.mu.Lock()
	t.rows = rows
	t.selected = 0
	t.cursor = 0
	t.mu.Unlock()
}

func (t *Table) SelectedRow() ([]string, int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.selected >= 0 && t.selected < len(t.rows) {
		return t.rows[t.selected], t.selected
	}
	return nil, -1
}

func (t *Table) HandleEvent(e Event) {
	if e.Type != EventKeyPress {
		return
	}
	ke, ok := e.Data.(KeyEvent)
	if !ok {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	switch ke.Key {
	case KeyUp:
		if t.selected > 0 {
			t.selected--
			if t.cursor > 0 {
				t.cursor--
			}
		}
	case KeyDown:
		if t.selected < len(t.rows)-1 {
			t.selected++
			if t.cursor < t.height-3 {
				t.cursor++
			}
		}
	case KeyPgUp:
		t.selected -= t.height
		if t.selected < 0 {
			t.selected = 0
		}
		t.cursor = 0
	case KeyPgDn:
		t.selected += t.height
		if t.selected >= len(t.rows) {
			t.selected = len(t.rows) - 1
		}
	case KeyHome:
		t.selected = 0
		t.cursor = 0
	case KeyEnd:
		t.selected = len(t.rows) - 1
	case KeyEnter:
		if t.onSelect != nil && t.selected >= 0 && t.selected < len(t.rows) {
			t.onSelect(t.selected, t.rows[t.selected])
		}
	}
}

func (t *Table) SortBy(col int, ascending bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.sortCol = col
	t.sortAsc = ascending
	if col < 0 || col >= len(t.columns) {
		return
	}
	for i := 0; i < len(t.rows); i++ {
		for j := i + 1; j < len(t.rows); j++ {
			a := ""
			if col < len(t.rows[i]) {
				a = t.rows[i][col]
			}
			b := ""
			if col < len(t.rows[j]) {
				b = t.rows[j][col]
			}
			swap := false
			if ascending {
				swap = a > b
			} else {
				swap = a < b
			}
			if swap {
				t.rows[i], t.rows[j] = t.rows[j], t.rows[i]
			}
		}
	}
}

func (t *Table) Render() string {
	t.mu.Lock()
	defer t.mu.Unlock()

	var out strings.Builder

	for i, col := range t.columns {
		title := col.Title
		if i == t.sortCol {
			if t.sortAsc {
				title += " ▲"
			} else {
				title += " ▼"
			}
		}
		out.WriteString(t.styles.Header.Apply(fmt.Sprintf(" %-*s", col.Width, title)))
	}
	out.WriteString("\n")

	separator := strings.Repeat("─", t.totalWidth())
	out.WriteString(t.styles.Border.Apply(" "+separator) + "\n")

	start := t.selected - t.cursor
	if start < 0 {
		start = 0
	}
	end := start + t.height - 2
	if end > len(t.rows) {
		end = len(t.rows)
	}

	for i := start; i < end; i++ {
		row := t.rows[i]
		selected := i == t.selected

		numStyle := t.styles.RowNum
		cellStyle := t.styles.Cell
		if selected && t.focused {
			cellStyle = t.styles.CellSelect
		} else if selected {
			cellStyle = t.styles.CellFocused
		}

		out.WriteString(numStyle.Apply(fmt.Sprintf(" %3d", i+1)))

		for j, col := range t.columns {
			cell := ""
			if j < len(row) {
				cell = row[j]
			}
			if len(cell) > col.Width {
				cell = cell[:col.Width-1] + "…"
			}
			switch col.Align {
			case AlignRight:
				out.WriteString(cellStyle.Apply(fmt.Sprintf(" %*s ", col.Width, cell)))
			case AlignCenter:
				pad := (col.Width - len(cell)) / 2
				out.WriteString(cellStyle.Apply(fmt.Sprintf(" %s%s ", strings.Repeat(" ", pad), cell)))
			default:
				out.WriteString(cellStyle.Apply(fmt.Sprintf(" %-*s ", col.Width, cell)))
			}
		}
		out.WriteString("\n")
	}

	total := len(t.rows)
	paginate := fmt.Sprintf(" %d-%d of %d", start+1, end, total)
	out.WriteString(t.styles.Border.Apply(paginate))

	return out.String()
}

func (t *Table) totalWidth() int {
	w := 4
	for _, col := range t.columns {
		w += col.Width + 2
	}
	return w
}
