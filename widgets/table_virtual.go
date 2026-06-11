package widgets

import (
	"sort"
	"strings"

	"github.com/xanstomper/mofu"
)

// ---------------------------------------------------------------------------
// Virtual Table Engine (Anthology Data/Table chapters)
// ---------------------------------------------------------------------------

// TableDataSource allows rendering millions of rows without retaining all rows in memory.
type TableDataSource interface {
	Len() int
	Row(index int, dst []string) []string
}

// SliceTableSource adapts [][]string to TableDataSource.
type SliceTableSource [][]string

func (s SliceTableSource) Len() int { return len(s) }
func (s SliceTableSource) Row(index int, dst []string) []string {
	if index < 0 || index >= len(s) {
		return dst[:0]
	}
	if cap(dst) < len(s[index]) {
		dst = make([]string, len(s[index]))
	}
	copy(dst, s[index])
	return dst[:len(s[index])]
}

// TableFilter returns true when a row should be visible.
type TableFilter func(row []string) bool

// TableSort describes incremental sorting.
type TableSort struct {
	Column int
	Asc    bool
	Less   func(a, b []string) bool
}

// VirtualTable renders large datasets with vertical/horizontal virtualization.
type VirtualTable struct {
	mofu.BaseNode
	Columns          []Column
	Source           TableDataSource
	Filter           TableFilter
	Sort             TableSort
	Selected         int
	Offset           int
	HorizontalOffset int
	FrozenColumns    int
	RowHeight        int
	Cache            bool
	rowCache         map[int][]string
	OnSelect         func(row int)
}

// NewVirtualTable returns a VirtualTable.
func NewVirtualTable(columns []Column, source TableDataSource) *VirtualTable {
	return &VirtualTable{Columns: columns, Source: source, RowHeight: 1, rowCache: make(map[int][]string)}
}

// SetSource swaps the data source and clears row cache.
func (t *VirtualTable) SetSource(source TableDataSource) {
	t.Source = source
	t.rowCache = make(map[int][]string)
	t.Selected = 0
	t.Offset = 0
	t.SetDirty()
}

// SetFilter applies a filter and resets the viewport.
func (t *VirtualTable) SetFilter(filter TableFilter) {
	t.Filter = filter
	t.rowCache = make(map[int][]string)
	t.Selected = 0
	t.Offset = 0
	t.SetDirty()
}

// SetSort applies sorting and clears cache.
func (t *VirtualTable) SetSort(sort TableSort) {
	t.Sort = sort
	t.rowCache = make(map[int][]string)
	t.SetDirty()
}

// ClearCache invalidates row cache.
func (t *VirtualTable) ClearCache() { t.rowCache = make(map[int][]string) }

func (t *VirtualTable) Children() []mofu.Node { return nil }
func (t *VirtualTable) Mount() mofu.Cmd       { return nil }
func (t *VirtualTable) Unmount()              {}

func (t *VirtualTable) Render(ctx *mofu.RenderContext) {
	b := t.Bounds()
	if b.Width <= 0 || b.Height <= 0 {
		return
	}
	r := ctx.Renderer
	theme := ctx.Theme
	widths := computeTableWidths(t.Columns, b.Width)
	visibleRows := b.Height - 1
	if visibleRows < 1 {
		visibleRows = 1
	}

	t.renderHeader(r, b, theme, widths)
	for i := 0; i < visibleRows; i++ {
		rowIndex := t.Offset + i
		if rowIndex >= t.Len() {
			break
		}
		rowY := b.Y + 1 + i
		row := t.Row(rowIndex)
		if !t.matchesFilter(row) {
			continue
		}
		style := theme.Typography.Body
		if rowIndex == t.Selected {
			style = style.Bg(theme.Colors.Primary)
		} else if rowIndex%2 == 1 {
			style = style.Bg(theme.Colors.Surface)
		}
		x := b.X
		for col := 0; col < len(t.Columns); col++ {
			w := widths[col]
			if x+w > b.X+b.Width {
				w = b.X + b.Width - x
			}
			if w <= 0 {
				continue
			}
			val := ""
			if col < len(row) {
				val = row[col]
			}
			val = mofu.Truncate(val, w, true)
			val = alignCell(val, w, t.Columns[col].Align)
			r.WriteStyledString(val, x, rowY, style)
			x += w + 1
		}
	}
}

func (t *VirtualTable) renderHeader(r *mofu.Renderer, b mofu.Rect, theme *mofu.Theme, widths []int) {
	x := b.X
	for i, col := range t.Columns {
		w := widths[i]
		if x+w > b.X+b.Width {
			w = b.X + b.Width - x
		}
		if w <= 0 {
			continue
		}
		title := col.Title
		if t.Sort.Column == i {
			if t.Sort.Asc {
				title = "▲ " + title
			} else {
				title = "▼ " + title
			}
		}
		r.WriteStyledString(mofu.Truncate(title, w, true), x, b.Y, theme.Typography.Label)
		x += w + 1
	}
}

func (t *VirtualTable) Len() int {
	if t.Source == nil {
		return 0
	}
	return t.Source.Len()
}

func (t *VirtualTable) Row(index int) []string {
	if t.Cache {
		if row, ok := t.rowCache[index]; ok {
			return row
		}
	}
	dst := make([]string, len(t.Columns))
	row := t.Source.Row(index, dst)
	if len(row) < len(t.Columns) {
		grown := make([]string, len(t.Columns))
		copy(grown, row)
		row = grown
	}
	if t.Cache {
		t.rowCache[index] = row
	}
	return row
}

func (t *VirtualTable) matchesFilter(row []string) bool {
	return t.Filter == nil || t.Filter(row)
}

func (t *VirtualTable) SortBy(col int) {
	if col < 0 || col >= len(t.Columns) {
		return
	}
	if t.Sort.Column == col {
		t.Sort.Asc = !t.Sort.Asc
	} else {
		t.Sort.Column = col
		t.Sort.Asc = true
	}
	t.ClearCache()
	t.SetDirty()
}

func (t *VirtualTable) HandleEvent(event mofu.Event) mofu.Cmd {
	if event.Type != mofu.EventKeyPress {
		return nil
	}
	ke, ok := event.Data.(mofu.KeyEvent)
	if !ok {
		return nil
	}
	bh := t.Bounds().Height - 1
	if bh < 1 {
		bh = 1
	}
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
		if t.Selected < t.Len()-1 {
			t.Selected++
			if t.Selected-t.Offset >= bh {
				t.Offset++
			}
			t.SetDirty()
		}
	case mofu.KeyPgUp:
		t.Selected -= bh
		if t.Selected < 0 {
			t.Selected = 0
		}
		t.Offset -= bh
		if t.Offset < 0 {
			t.Offset = 0
		}
		t.SetDirty()
	case mofu.KeyPgDn:
		t.Selected += bh
		if t.Selected >= t.Len() {
			t.Selected = t.Len() - 1
		}
		t.Offset += bh
		t.SetDirty()
	case mofu.KeyHome:
		t.Selected = 0
		t.Offset = 0
		t.SetDirty()
	case mofu.KeyEnd:
		t.Selected = t.Len() - 1
		t.Offset = t.Selected
		t.SetDirty()
	case mofu.KeyEnter:
		if t.OnSelect != nil && t.Selected >= 0 {
			t.OnSelect(t.Selected)
		}
	}
	return nil
}

func computeTableWidths(cols []Column, total int) []int {
	if len(cols) == 0 {
		return nil
	}
	widths := make([]int, len(cols))
	available := total
	for i, col := range cols {
		w := col.Width
		if w <= 0 {
			w = 10
		}
		if w < col.MinWidth {
			w = col.MinWidth
		}
		widths[i] = w
		available -= w + 1
	}
	if available > 0 && len(cols) > 0 {
		last := len(cols) - 1
		widths[last] += available
	}
	return widths
}

func alignCell(s string, width int, align mofu.Align) string {
	switch align {
	case mofu.AlignCenter:
		return mofu.PadCenter(s, width)
	case mofu.AlignRight:
		return mofu.PadLeft(s, width)
	default:
		return mofu.PadRight(s, width)
	}
}

// IncrementalTable wraps a VirtualTable and keeps sorted indices incrementally.
type IncrementalTable struct {
	*VirtualTable
	indices []int
}

// NewIncrementalTable returns an IncrementalTable.
func NewIncrementalTable(columns []Column, source TableDataSource) *IncrementalTable {
	vt := NewVirtualTable(columns, source)
	return &IncrementalTable{VirtualTable: vt, indices: make([]int, 0, source.Len())}
}

func (t *IncrementalTable) Len() int { return len(t.indices) }

func (t *IncrementalTable) Row(index int) []string {
	if index < 0 || index >= len(t.indices) {
		return nil
	}
	return t.VirtualTable.Row(t.indices[index])
}

func (t *IncrementalTable) RebuildIndices() {
	n := t.VirtualTable.Len()
	t.indices = make([]int, n)
	for i := 0; i < n; i++ {
		t.indices[i] = i
	}
	if t.Sort.Less != nil {
		sort.SliceStable(t.indices, func(i, j int) bool {
			a := t.VirtualTable.Row(t.indices[i])
			b := t.VirtualTable.Row(t.indices[j])
			if t.Sort.Asc {
				return t.Sort.Less(a, b)
			}
			return t.Sort.Less(b, a)
		})
	}
	t.SetDirty()
}

// FilteredRows returns visible row indices after filter.
func (t *IncrementalTable) FilteredRows() []int {
	out := make([]int, 0, len(t.indices))
	for _, idx := range t.indices {
		row := t.VirtualTable.Row(idx)
		if t.Filter == nil || t.Filter(row) {
			out = append(out, idx)
		}
	}
	return out
}

// LiveUpdate applies a row replacement and invalidates only that row.
func (t *VirtualTable) LiveUpdate(index int, values []string) {
	if t.Cache {
		delete(t.rowCache, index)
	}
	if st, ok := t.Source.(interface{ SetRow(int, []string) }); ok {
		st.SetRow(index, values)
	}
	t.SetDirty()
}

// AppendRow appends a row when the source supports it.
func (t *VirtualTable) AppendRow(values []string) bool {
	updater, ok := t.Source.(interface{ Append([]string) int })
	if !ok {
		return false
	}
	idx := updater.Append(values)
	if t.Cache {
		delete(t.rowCache, idx)
	}
	t.SetDirty()
	return true
}

// MutableSliceSource is a slice source with update hooks.
type MutableSliceSource struct {
	rows [][]string
}

func NewMutableSliceSource(rows [][]string) *MutableSliceSource {
	return &MutableSliceSource{rows: rows}
}
func (s *MutableSliceSource) Len() int { return len(s.rows) }
func (s *MutableSliceSource) Row(index int, dst []string) []string {
	if index < 0 || index >= len(s.rows) {
		return dst[:0]
	}
	if cap(dst) < len(s.rows[index]) {
		dst = make([]string, len(s.rows[index]))
	}
	copy(dst, s.rows[index])
	return dst[:len(s.rows[index])]
}
func (s *MutableSliceSource) SetRow(index int, values []string) {
	if index >= 0 && index < len(s.rows) {
		s.rows[index] = values
	}
}
func (s *MutableSliceSource) Append(values []string) int {
	s.rows = append(s.rows, values)
	return len(s.rows) - 1
}
func (s *MutableSliceSource) Rows() [][]string { return s.rows }
func (s *MutableSliceSource) FilterText(term string) {
	if term == "" {
		return
	}
	term = strings.ToLower(term)
	kept := s.rows[:0]
	for _, row := range s.rows {
		match := false
		for _, cell := range row {
			if strings.Contains(strings.ToLower(cell), term) {
				match = true
				break
			}
		}
		if match {
			kept = append(kept, row)
		}
	}
}
