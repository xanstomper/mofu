package mofu

// ---------------------------------------------------------------------------
// Grid Layout — CSS Grid-like terminal layout
// ---------------------------------------------------------------------------

// GridDef defines a CSS Grid-like layout with named columns and rows.
type GridDef struct {
	Columns []TrackSize // column track sizes
	Rows    []TrackSize // row track sizes
	Gap     int         // gap between cells
}

// TrackSize specifies the size of a grid track (column or row).
type TrackSize struct {
	Fixed  int     // fixed size in cells (0 = not fixed)
	Fr     float64 // fractional unit (1 = equal share of remaining space)
	Min    int     // minimum size
	Max    int     // maximum size (0 = unlimited)
	Auto   bool    // auto-size to content
}

// Fixed creates a fixed-size track.
func Fixed(size int) TrackSize {
	return TrackSize{Fixed: size}
}

// Fr creates a fractional track.
func Fr(units float64) TrackSize {
	return TrackSize{Fr: units}
}

// MinMax creates a track with min/max constraints.
func MinMax(min, max int) TrackSize {
	return TrackSize{Min: min, Max: max}
}

// AutoSize creates an auto-sized track.
func AutoSize() TrackSize {
	return TrackSize{Auto: true}
}

// GridPlacement specifies where a child is placed in the grid.
type GridPlacement struct {
	ColStart int // 1-indexed column start
	ColEnd   int // 1-indexed column end (0 = ColStart + 1)
	RowStart int // 1-indexed row start
	RowEnd   int // 1-indexed row end (0 = RowStart + 1)
}

// CSSGridNode is a container that lays out children in a CSS Grid.
type CSSGridNode struct {
	BaseNode
	grid       GridDef
	placements map[Node]GridPlacement
}

// NewCSSGrid creates a CSS Grid layout node.
func NewCSSGrid(grid GridDef, children ...Node) *CSSGridNode {
	g := &CSSGridNode{
		grid:       grid,
		placements: make(map[Node]GridPlacement),
	}
	g.children = children
	return g
}

// SetPlacement sets the grid placement for a child node.
func (g *CSSGridNode) SetPlacement(child Node, p GridPlacement) {
	g.placements[child] = p
}

func (g *CSSGridNode) Render(ctx *RenderContext) {
	bounds := ctx.Bounds
	s := g.BaseNode.style
	inner := Rect{
		X:      bounds.X + s.Padding.Left + s.Margin.Left,
		Y:      bounds.Y + s.Padding.Top + s.Margin.Top,
		Width:  bounds.Width - s.Padding.Left - s.Padding.Right - s.Margin.Left - s.Margin.Right,
		Height: bounds.Height - s.Padding.Top - s.Padding.Bottom - s.Margin.Top - s.Margin.Bottom,
	}
	if inner.Width < 0 {
		inner.Width = 0
	}
	if inner.Height < 0 {
		inner.Height = 0
	}

	colSizes := resolveTrackSizes(g.grid.Columns, inner.Width, g.grid.Gap)
	rowSizes := resolveTrackSizes(g.grid.Rows, inner.Height, g.grid.Gap)

	for _, child := range g.children {
		p, ok := g.placements[child]
		if !ok {
			p = GridPlacement{ColStart: 1, ColEnd: 2, RowStart: 1, RowEnd: 2}
		}

		colStart := max(0, p.ColStart-1)
		colEnd := min(len(colSizes), p.ColEnd)
		if colEnd <= colStart {
			colEnd = colStart + 1
		}
		rowStart := max(0, p.RowStart-1)
		rowEnd := min(len(rowSizes), p.RowEnd)
		if rowEnd <= rowStart {
			rowEnd = rowStart + 1
		}

		x := inner.X
		for i := 0; i < colStart; i++ {
			x += colSizes[i] + g.grid.Gap
		}
		w := 0
		for i := colStart; i < colEnd && i < len(colSizes); i++ {
			w += colSizes[i]
			if i > colStart {
				w += g.grid.Gap
			}
		}

		y := inner.Y
		for i := 0; i < rowStart; i++ {
			y += rowSizes[i] + g.grid.Gap
		}
		h := 0
		for i := rowStart; i < rowEnd && i < len(rowSizes); i++ {
			h += rowSizes[i]
			if i > rowStart {
				h += g.grid.Gap
			}
		}

		cellBounds := Rect{X: x, Y: y, Width: w, Height: h}
		child.SetBounds(cellBounds)
		childCtx := *ctx
		childCtx.Bounds = cellBounds
		child.Render(&childCtx)
	}
}

func (g *CSSGridNode) HandleEvent(event Event) Cmd {
	for _, child := range g.children {
		if cmd := child.HandleEvent(event); cmd != nil {
			return cmd
		}
	}
	return nil
}

func (g *CSSGridNode) Mount() Cmd {
	var cmds []Cmd
	for _, child := range g.children {
		if cmd := child.Mount(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return Batch(cmds...)
}

func (g *CSSGridNode) Unmount() {
	for _, child := range g.children {
		child.Unmount()
	}
}

// resolveTrackSizes calculates actual pixel sizes for grid tracks.
func resolveTrackSizes(tracks []TrackSize, available int, gap int) []int {
	if len(tracks) == 0 {
		return nil
	}

	sizes := make([]int, len(tracks))
	totalFixed := 0
	totalFr := 0.0
	gapTotal := gap * (len(tracks) - 1)

	for i, t := range tracks {
		if t.Fixed > 0 {
			sizes[i] = t.Fixed
			totalFixed += t.Fixed
		} else if t.Auto {
			sizes[i] = 0
		} else {
			totalFr += t.Fr
		}
	}

	remaining := available - totalFixed - gapTotal
	if remaining < 0 {
		remaining = 0
	}

	if totalFr > 0 {
		for i, t := range tracks {
			if t.Fr > 0 {
				sizes[i] = int(float64(remaining) * t.Fr / totalFr)
			}
		}
	}

	for i, t := range tracks {
		if t.Min > 0 && sizes[i] < t.Min {
			sizes[i] = t.Min
		}
		if t.Max > 0 && sizes[i] > t.Max {
			sizes[i] = t.Max
		}
	}

	autoCount := 0
	used := 0
	for i, t := range tracks {
		if t.Auto && sizes[i] == 0 {
			autoCount++
		}
		used += sizes[i]
	}
	if autoCount > 0 {
		remaining = available - used - gapTotal
		if remaining < 0 {
			remaining = 0
		}
		perAuto := remaining / autoCount
		for i, t := range tracks {
			if t.Auto && sizes[i] == 0 {
				sizes[i] = perAuto
			}
		}
	}

	return sizes
}

// ---------------------------------------------------------------------------
// Constraint Solver — min/max/aspect-ratio constraints
// ---------------------------------------------------------------------------

// Constraint represents a layout constraint.
type Constraint struct {
	MinWidth  int     // minimum width (0 = no constraint)
	MaxWidth  int     // maximum width (0 = unlimited)
	MinHeight int     // minimum height (0 = no constraint)
	MaxHeight int     // maximum height (0 = unlimited)
	Aspect    float64 // width/height ratio (0 = no constraint)
}

// ApplyConstraints applies min/max/aspect constraints to a Rect.
func ApplyConstraints(r Rect, c Constraint) Rect {
	w := r.Width
	h := r.Height

	if c.MinWidth > 0 && w < c.MinWidth {
		w = c.MinWidth
	}
	if c.MaxWidth > 0 && w > c.MaxWidth {
		w = c.MaxWidth
	}

	if c.MinHeight > 0 && h < c.MinHeight {
		h = c.MinHeight
	}
	if c.MaxHeight > 0 && h > c.MaxHeight {
		h = c.MaxHeight
	}

	if c.Aspect > 0 {
		idealH := int(float64(w) / c.Aspect)
		if idealH >= c.MinHeight && (c.MaxHeight == 0 || idealH <= c.MaxHeight) {
			h = idealH
		} else {
			idealW := int(float64(h) * c.Aspect)
			if idealW >= c.MinWidth && (c.MaxWidth == 0 || idealW <= c.MaxWidth) {
				w = idealW
			}
		}
	}

	return Rect{X: r.X, Y: r.Y, Width: w, Height: h}
}

// ---------------------------------------------------------------------------
// Responsive breakpoints
// ---------------------------------------------------------------------------

// SizeClass represents terminal size categories.
type SizeClass int

const (
	SizeCompact  SizeClass = iota // < 80 cols
	SizeMedium                    // 80-120 cols
	SizeExpanded                  // > 120 cols
)

// ClassifySize returns the size class for the given terminal width.
func ClassifySize(width int) SizeClass {
	switch {
	case width < 80:
		return SizeCompact
	case width <= 120:
		return SizeMedium
	default:
		return SizeExpanded
	}
}
