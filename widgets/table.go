package widgets

import (
	"sort"
	"strings"

	"github.com/anomalyco/mofu"
)

type Column struct {
	Title    string
	Width    int
	MinWidth int
	Align    mofu.Align
	Sortable bool
}

type Table struct {
	mofu.BaseNode
	Columns  []Column
	Rows     [][]string
	Selected int
	Offset   int
	SortCol  int
	SortAsc  bool
	OnSelect func(row int)
}

func NewTable(columns []Column) *Table {
	return &Table{
		Columns:  columns,
		Rows:     make([][]string, 0),
		Selected: -1,
		SortCol:  -1,
		SortAsc:  true,
	}
}

func (t *Table) AddRow(values ...string) {
	row := make([]string, len(t.Columns))
	for i, v := range values {
		if i < len(row) {
			row[i] = v
		}
	}
	t.Rows = append(t.Rows, row)
	t.SetDirty()
}

func (t *Table) Clear() {
	t.Rows = nil
	t.Selected = -1
	t.Offset = 0
	t.SetDirty()
}

func (t *Table) SortBy(col int) {
	if col < 0 || col >= len(t.Columns) || !t.Columns[col].Sortable {
		return
	}
	if t.SortCol == col {
		t.SortAsc = !t.SortAsc
	} else {
		t.SortCol = col
		t.SortAsc = true
	}
	sort.SliceStable(t.Rows, func(i, j int) bool {
		a := ""
		b := ""
		if col < len(t.Rows[i]) {
			a = t.Rows[i][col]
		}
		if col < len(t.Rows[j]) {
			b = t.Rows[j][col]
		}
		if t.SortAsc {
			return a < b
		}
		return a > b
	})
	t.SetDirty()
}

func (t *Table) Render(ctx *mofu.RenderContext) {
	b := t.BaseNode.Bounds()
	r := ctx.Renderer

	totalW := 0
	for i, col := range t.Columns {
		w := col.Width
		if w <= 0 {
			w = 10
		}
		if i == len(t.Columns)-1 {
			remaining := b.Width - totalW
			if remaining > w {
				w = remaining
			}
		}
		totalW += w + 1
	}
	if totalW > b.Width {
		totalW = b.Width
	}

	x := b.X
	theme := ctx.Theme
	for i, col := range t.Columns {
		w := col.Width
		if w <= 0 {
			w = 10
		}
		if i == len(t.Columns)-1 {
			remaining := b.Width - (x - b.X)
			if remaining > w {
				w = remaining
			}
		}
		if x+w > b.X+b.Width {
			w = b.X + b.Width - x
		}

		headerStyle := theme.Typography.Label
		title := col.Title
		if t.SortCol == i {
			if t.SortAsc {
				title = "▲ " + title
			} else {
				title = "▼ " + title
			}
		}
		r.WriteStyledString(truncate(title, w), x, b.Y, headerStyle)
		x += w + 1
	}

	for rowIdx := 0; rowIdx < len(t.Rows) && rowIdx-t.Offset < b.Height-1; rowIdx++ {
		rowY := b.Y + 1 + rowIdx - t.Offset
		if rowY < b.Y+1 || rowY >= b.Y+b.Height {
			continue
		}
		row := t.Rows[rowIdx]
		x = b.X
		rowStyle := theme.Typography.Body
		if rowIdx == t.Selected {
			rowStyle = rowStyle.Bg(theme.Colors.Primary)
		} else if rowIdx%2 == 1 {
			rowStyle = rowStyle.Bg(theme.Colors.Surface)
		}
		for i, col := range t.Columns {
			w := col.Width
			if w <= 0 {
				w = 10
			}
			if i == len(t.Columns)-1 {
				remaining := b.Width - (x - b.X)
				if remaining > w {
					w = remaining
				}
			}
			if x+w > b.X+b.Width {
				w = b.X + b.Width - x
			}
			val := ""
			if i < len(row) {
				val = row[i]
			}
			val = truncate(val, w)
			switch col.Align {
			case mofu.AlignCenter:
				pad := (w - len(val)) / 2
				if pad > 0 {
					val = strings.Repeat(" ", pad) + val
				}
			case mofu.AlignRight:
				pad := w - len(val)
				if pad > 0 {
					val = strings.Repeat(" ", pad) + val
				}
			}
			r.WriteStyledString(val, x, rowY, rowStyle)
			x += w + 1
		}
	}
}

func (t *Table) HandleEvent(event mofu.Event) mofu.Cmd {
	if event.Type == mofu.EventKeyPress {
		ke, ok := event.Data.(mofu.KeyEvent)
		if ok {
			switch ke.Key {
			case mofu.KeyUp:
				if t.Selected > 0 {
					t.Selected--
					if t.Selected < t.Offset {
						t.Offset = t.Selected
					}
					t.SetDirty()
				}
			case mofu.KeyDown:
				if t.Selected < len(t.Rows)-1 {
					t.Selected++
					if t.Selected-t.Offset >= t.BaseNode.Bounds().Height-1 {
						t.Offset++
					}
					t.SetDirty()
				}
			case mofu.KeyEnter:
				if t.OnSelect != nil && t.Selected >= 0 {
					t.OnSelect(t.Selected)
				}
			}
		}
	}
	return nil
}

func (t *Table) Children() []mofu.Node { return nil }
func (t *Table) Mount() mofu.Cmd       { return nil }
func (t *Table) Unmount()              {}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen < 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
