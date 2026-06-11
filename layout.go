package mofu

// Rect represents a rectangular region in the terminal grid.
type Rect struct {
	X, Y, Width, Height int
}

// Contains reports whether the point (x, y) is inside the rectangle.
func (r Rect) Contains(x, y int) bool {
	return x >= r.X && x < r.X+r.Width && y >= r.Y && y < r.Y+r.Height
}

func mergeRects(a, b Rect) Rect {
	x1 := min(a.X, b.X)
	y1 := min(a.Y, b.Y)
	x2 := max(a.X+a.Width, b.X+b.Width)
	y2 := max(a.Y+a.Height, b.Y+b.Height)
	return Rect{X: x1, Y: y1, Width: x2 - x1, Height: y2 - y1}
}

func ComputeLayout(node Node, bounds Rect) {
	if node == nil {
		return
	}
	node.SetBounds(bounds)
	s := node.Style()
	inner := Rect{
		X:      bounds.X + s.Margin.Left + s.Padding.Left,
		Y:      bounds.Y + s.Margin.Top + s.Padding.Top,
		Width:  bounds.Width - s.Margin.Left - s.Margin.Right - s.Padding.Left - s.Padding.Right,
		Height: bounds.Height - s.Margin.Top - s.Margin.Bottom - s.Padding.Top - s.Padding.Bottom,
	}
	if inner.Width < 0 {
		inner.Width = 0
	}
	if inner.Height < 0 {
		inner.Height = 0
	}

	switch sn := node.(type) {
	case *StackNode:
		if s.Direction == DirectionRow {
			layoutRow(sn.Children(), inner, s)
		} else {
			layoutCol(sn.Children(), inner, s)
		}
	case *BoxNode:
		for _, child := range sn.Children() {
			ComputeLayout(child, inner)
		}
	case *OverlayNode:
		for _, child := range sn.Children() {
			ComputeLayout(child, inner)
		}
	default:
		for _, child := range node.Children() {
			ComputeLayout(child, inner)
		}
	}
}

func layoutRow(children []Node, bounds Rect, s *Style) {
	if len(children) == 0 {
		return
	}
	fixedW := 0
	totalGrow := 0.0
	flexCount := 0
	for _, child := range children {
		cs := child.Style()
		if cs.Width > 0 {
			fixedW += cs.Width
		} else if cs.Grow > 0 {
			totalGrow += cs.Grow
			flexCount++
		} else {
			flexCount++
		}
	}
	gapTotal := s.Gap * (len(children) - 1)
	availW := bounds.Width - fixedW - gapTotal
	if availW < 0 {
		availW = 0
	}
	flexW := 0
	if flexCount > 0 {
		flexW = availW / flexCount
	}

	x := bounds.X
	for _, child := range children {
		cs := child.Style()
		w := cs.Width
		if w <= 0 {
			if cs.Grow > 0 && totalGrow > 0 {
				w = int(float64(availW) * cs.Grow / totalGrow)
			} else {
				w = flexW
			}
		}
		h := cs.Height
		yOff := 0
		switch cs.Align {
		case AlignCenter:
			yOff = (bounds.Height - h) / 2
		case AlignRight:
			yOff = bounds.Height - h
		case AlignStretch:
			h = bounds.Height
		}
		if yOff < 0 {
			yOff = 0
		}
		child.SetBounds(Rect{x, bounds.Y + yOff, w, h - yOff})
		child.SetDirty()
		ComputeLayout(child, child.Bounds())
		x += w + s.Gap
	}
}

func layoutCol(children []Node, bounds Rect, s *Style) {
	if len(children) == 0 {
		return
	}
	fixedH := 0
	totalGrow := 0.0
	flexCount := 0
	for _, child := range children {
		cs := child.Style()
		if cs.Height > 0 {
			fixedH += cs.Height
		} else if cs.Grow > 0 {
			totalGrow += cs.Grow
			flexCount++
		} else {
			flexCount++
		}
	}
	gapTotal := s.Gap * (len(children) - 1)
	availH := bounds.Height - fixedH - gapTotal
	if availH < 0 {
		availH = 0
	}
	flexH := 0
	if flexCount > 0 {
		flexH = availH / flexCount
	}

	y := bounds.Y
	for _, child := range children {
		cs := child.Style()
		h := cs.Height
		if h <= 0 {
			if cs.Grow > 0 && totalGrow > 0 {
				h = int(float64(availH) * cs.Grow / totalGrow)
			} else {
				h = flexH
			}
		}
		w := cs.Width
		xOff := 0
		switch cs.Align {
		case AlignCenter:
			xOff = (bounds.Width - w) / 2
		case AlignRight:
			xOff = bounds.Width - w
		case AlignStretch:
			w = bounds.Width
		}
		if xOff < 0 {
			xOff = 0
		}
		child.SetBounds(Rect{bounds.X + xOff, y, w - xOff, h})
		child.SetDirty()
		ComputeLayout(child, child.Bounds())
		y += h + s.Gap
	}
}
