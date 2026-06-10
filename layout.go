package mofu

// LayoutType defines how children are arranged.
type LayoutType int

const (
	LayoutRow     LayoutType = iota // Children arranged horizontally
	LayoutColumn                    // Children arranged vertically
	LayoutStack                     // Children stacked on top of each other
	LayoutOverlay                   // Children overlaid (absolute positioning)
)

// Rect defines a rectangular region.
type Rect struct {
	X, Y, Width, Height int
}

// LayoutNode is a node in the layout tree.
type LayoutNode struct {
	Component Component
	Style     Style
	Layout    LayoutType
	Visible   bool
	Rect      Rect
	Children  []*LayoutNode
}

// ComputeLayout calculates the position and size for each node.
func ComputeLayout(node *LayoutNode, bounds Rect) {
	if node == nil {
		return
	}
	node.Rect = bounds

	switch node.Layout {
	case LayoutRow:
		layoutRow(node.Children, bounds)
	case LayoutColumn:
		layoutColumn(node.Children, bounds)
	case LayoutStack:
		layoutStack(node.Children, bounds)
	case LayoutOverlay:
		for _, child := range node.Children {
			ComputeLayout(child, bounds)
		}
	}
}

func layoutRow(children []*LayoutNode, bounds Rect) {
	if len(children) == 0 {
		return
	}
	// Calculate fixed widths
	fixedW := 0
	flexCount := 0
	for _, child := range children {
		if child.Style.Width > 0 {
			fixedW += child.Style.Width
		} else {
			flexCount++
		}
	}
	gapW := bounds.Width - fixedW
	if gapW < 0 {
		gapW = 0
	}
	flexW := 0
	if flexCount > 0 {
		flexW = gapW / flexCount
	}

	x := bounds.X
	for _, child := range children {
		w := child.Style.Width
		if w <= 0 {
			w = flexW
		}
		ComputeLayout(child, Rect{x, bounds.Y, w, bounds.Height})
		x += w
	}
}

func layoutColumn(children []*LayoutNode, bounds Rect) {
	if len(children) == 0 {
		return
	}
	fixedH := 0
	flexCount := 0
	for _, child := range children {
		if child.Style.Height > 0 {
			fixedH += child.Style.Height
		} else {
			flexCount++
		}
	}
	gapH := bounds.Height - fixedH
	if gapH < 0 {
		gapH = 0
	}
	flexH := 0
	if flexCount > 0 {
		flexH = gapH / flexCount
	}

	y := bounds.Y
	for _, child := range children {
		h := child.Style.Height
		if h <= 0 {
			h = flexH
		}
		ComputeLayout(child, Rect{bounds.X, y, bounds.Width, h})
		y += h
	}
}

func layoutStack(children []*LayoutNode, bounds Rect) {
	for _, child := range children {
		ComputeLayout(child, Rect{
			X:      bounds.X + child.Style.Margin.Left,
			Y:      bounds.Y + child.Style.Margin.Top,
			Width:  bounds.Width - child.Style.Margin.Left - child.Style.Margin.Right,
			Height: bounds.Height - child.Style.Margin.Top - child.Style.Margin.Bottom,
		})
	}
}
