package mofu

type Rect struct {
	X, Y, Width, Height int
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
		yOff := 0
		switch cs.Align {
		case AlignCenter:
			yOff = (bounds.Height - cs.Height) / 2
		case AlignRight:
			yOff = bounds.Height - cs.Height
		case AlignStretch:
			cs.Height = bounds.Height
		}
		if yOff < 0 {
			yOff = 0
		}
		child.SetBounds(Rect{x, bounds.Y + yOff, w, bounds.Height - yOff})
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
		xOff := 0
		switch cs.Align {
		case AlignCenter:
			xOff = (bounds.Width - cs.Width) / 2
		case AlignRight:
			xOff = bounds.Width - cs.Width
		case AlignStretch:
			cs.Width = bounds.Width
		}
		if xOff < 0 {
			xOff = 0
		}
		child.SetBounds(Rect{bounds.X + xOff, y, bounds.Width - xOff, h})
		child.SetDirty()
		ComputeLayout(child, child.Bounds())
		y += h + s.Gap
	}
}
