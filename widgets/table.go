package widgets

import (
	"sort"

	"github.com/xanstomper/mofu"
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
	t.clamp()
	t.SetDirty()
}

func (t *Table) SetRows(rows [][]string) {
	t.Rows = rows
	t.clamp()
	t.SetDirty()
}

func (t *Table) SetColumns(columns []Column) {
	t.Columns = columns
	t.clamp()
	t.SetDirty()
}

func (t *Table) Clear() {
	t.Rows = nil
	t.Selected = -1
	t.Offset = 0
	t.SetDirty()
}

func (t *Table) clamp() {
	if len(t.Rows) == 0 {
		t.Selected = -1
		t.Offset = 0
		return
	}
	if t.Selected < 0 {
		t.Selected = 0
	}
	if t.Selected >= len(t.Rows) {
		t.Selected = len(t.Rows) - 1
	}
	if t.Offset < 0 {
		t.Offset = 0
	}
	if t.Offset > t.Selected {
		t.Offset = t.Selected
	}
	if h := t.visibleHeight(); h > 0 && t.Selected-t.Offset >= h {
		t.Offset = t.Selected - h + 1
	}
	if t.Offset < 0 {
		t.Offset = 0
	}
}

func (t *Table) visibleHeight() int {
	h := t.BaseNode.Bounds().Height - 1
	if h <= 0 {
		return 1
	}
	return h
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
	if b.Width <= 0 || b.Height <= 0 || len(t.Columns) == 0 {
		return
	}
	t.clamp()

	widths := t.columnWidths(b.Width)
	if len(widths) == 0 {
		return
	}
	r := ctx.Renderer
	headerStyle := ctx.Theme.Typography.Label
	bodyStyle := ctx.Theme.Typography.Body
	selectedStyle := bodyStyle.Bg(ctx.Theme.Colors.Primary)
	surfaceStyle := bodyStyle.Bg(ctx.Theme.Colors.Surface)

	x := b.X
	for i, col := range t.Columns {
		w := widths[i]
		if w <= 0 {
			continue
		}
		title := col.Title
		if t.SortCol == i {
			if t.SortAsc {
				title = "▲ " + title
			} else {
				title = "▼ " + title
			}
		}
		r.WriteString(mofu.Truncate(title, w, true), x, b.Y, headerStyle.Foreground, headerStyle.Background, headerStyle.Attrs)
		x += w + 1
	}

	h := t.visibleHeight()
	start := t.Offset
	if start < 0 {
		start = 0
	}
	if start > len(t.Rows)-h {
		start = len(t.Rows) - h
	}
	if start < 0 {
		start = 0
	}
	end := start + h
	if end > len(t.Rows) {
		end = len(t.Rows)
	}
	for rowIdx := start; rowIdx < end; rowIdx++ {
		rowY := b.Y + 1 + rowIdx - t.Offset
		if rowY < b.Y+1 || rowY >= b.Y+b.Height {
			continue
		}
		row := t.Rows[rowIdx]
		x = b.X
		rowStyle := bodyStyle
		if rowIdx == t.Selected {
			rowStyle = selectedStyle
		} else if rowIdx%2 == 1 {
			rowStyle = surfaceStyle
		}
		for i, col := range t.Columns {
			w := widths[i]
			if w <= 0 {
				x += w + 1
				continue
			}
			val := ""
			if i < len(row) {
				val = row[i]
			}
			val = mofu.Truncate(val, w, true)
			switch col.Align {
			case mofu.AlignCenter:
				val = mofu.PadCenter(val, w)
			case mofu.AlignRight:
				val = mofu.PadLeft(val, w)
			default:
				val = mofu.PadRight(val, w)
			}
			r.WriteString(val, x, rowY, rowStyle.Foreground, rowStyle.Background, rowStyle.Attrs)
			x += w + 1
		}
	}
}

func (t *Table) columnWidths(total int) []int {
	if len(t.Columns) == 0 || total <= 0 {
		return nil
	}
	widths := make([]int, len(t.Columns))
	used := 0
	for i, col := range t.Columns {
		w := col.Width
		if w <= 0 {
			w = 10
		}
		if w < col.MinWidth {
			w = col.MinWidth
		}
		widths[i] = w
		used += w
	}
	if used < total {
		for i := range widths {
			widths[i] += (total - used) / len(widths)
		}
		rem := (total - used) % len(widths)
		for i := 0; i < rem; i++ {
			widths[i]++
		}
	}
	if used > total {
		for i := range widths {
			widths[i] = max(1, widths[i]*total/used)
		}
	}
	return widths
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
