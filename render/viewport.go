// Package render provides viewport-aware computation for MOFU.
//
// Only visible content is computed and rendered, avoiding wasted CPU on
// off-screen data. This is critical for large tables, logs, and lists
// where computing all rows would be wasteful.
package render

// Viewport represents the visible region of a scrollable content area.
type Viewport struct {
	OffsetX  int // Horizontal scroll offset
	OffsetY  int // Vertical scroll offset
	Width    int // Visible width in cells
	Height   int // Visible height in rows
	ContentW int // Total content width
	ContentH int // Total content height
}

// NewViewport creates a Viewport with the given visible dimensions.
func NewViewport(w, h int) *Viewport {
	return &Viewport{Width: w, Height: h}
}

// VisibleRange returns the start and end row indices that are currently
// visible, clamped to valid bounds. Use this to slice data before rendering.
func (vp *Viewport) VisibleRange(totalRows int) (start, end int) {
	start = vp.OffsetY
	if start < 0 {
		start = 0
	}
	end = start + vp.Height
	if end > totalRows {
		end = totalRows
	}
	if start > end {
		start = end
	}
	return start, end
}

// VisibleColRange returns the start and end column indices visible.
func (vp *Viewport) VisibleColRange(totalCols int) (start, end int) {
	start = vp.OffsetX
	if start < 0 {
		start = 0
	}
	end = start + vp.Width
	if end > totalCols {
		end = totalCols
	}
	if start > end {
		start = end
	}
	return start, end
}

// ScrollBy adjusts the scroll offset by the given delta, clamped to content bounds.
func (vp *Viewport) ScrollBy(dx, dy int) {
	vp.OffsetX += dx
	vp.OffsetY += dy
	vp.clamp()
}

// ScrollTo sets the absolute scroll position, clamped to content bounds.
func (vp *Viewport) ScrollTo(x, y int) {
	vp.OffsetX = x
	vp.OffsetY = y
	vp.clamp()
}

// ScrollIntoView ensures the row at index is visible within the viewport.
func (vp *Viewport) ScrollIntoView(row int) {
	if row < vp.OffsetY {
		vp.OffsetY = row
	} else if row >= vp.OffsetY+vp.Height {
		vp.OffsetY = row - vp.Height + 1
	}
	vp.clamp()
}

// Clamp ensures the viewport offset is within valid content bounds.
func (vp *Viewport) clamp() {
	if vp.OffsetX < 0 {
		vp.OffsetX = 0
	}
	if vp.OffsetY < 0 {
		vp.OffsetY = 0
	}
	maxX := vp.ContentW - vp.Width
	if maxX < 0 {
		maxX = 0
	}
	if vp.OffsetX > maxX {
		vp.OffsetX = maxX
	}
	maxY := vp.ContentH - vp.Height
	if maxY < 0 {
		maxY = 0
	}
	if vp.OffsetY > maxY {
		vp.OffsetY = maxY
	}
}

// SetContentSize updates the total content dimensions and re-clamps.
func (vp *Viewport) SetContentSize(w, h int) {
	vp.ContentW = w
	vp.ContentH = h
	vp.clamp()
}

// FractionVisible returns the fraction of total content currently visible (0.0 to 1.0).
func (vp *Viewport) FractionVisible() float64 {
	if vp.ContentH <= 0 {
		return 1.0
	}
	f := float64(vp.Height) / float64(vp.ContentH)
	if f > 1.0 {
		f = 1.0
	}
	return f
}

// ScrollPercent returns the current scroll position as a percentage (0.0 to 1.0).
func (vp *Viewport) ScrollPercent() float64 {
	if vp.ContentH <= vp.Height {
		return 0.0
	}
	return float64(vp.OffsetY) / float64(vp.ContentH-vp.Height)
}

// ScrollbarThumbPos computes the thumb position and size for a scrollbar
// of the given track height. Returns (position, size).
func (vp *Viewport) ScrollbarThumbPos(trackHeight int) (pos, size int) {
	if vp.ContentH <= vp.Height {
		return 0, trackHeight
	}
	size = int(float64(trackHeight) * float64(vp.Height) / float64(vp.ContentH))
	if size < 1 {
		size = 1
	}
	pos = int(float64(trackHeight-size) * vp.ScrollPercent())
	if pos < 0 {
		pos = 0
	}
	if pos+size > trackHeight {
		pos = trackHeight - size
	}
	return pos, size
}
