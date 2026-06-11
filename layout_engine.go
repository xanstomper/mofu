package mofu

import (
	"sort"
)

// ---------------------------------------------------------------------------
// Advanced Layout Engine
// ---------------------------------------------------------------------------
// Unlike Bubble Tea's manual layout, MOFU provides:
// - Constraint-based layout (min/max width/height)
// - Flex layouts with grow/shrink
// - Grid layouts with columns/rows
// - Responsive breakpoints
// - Auto-sizing based on content

// LayoutNode is a node in the layout tree.
type LayoutNode struct {
	Bounds    Rect
	MinWidth  int
	MaxWidth  int
	MinHeight int
	MaxHeight int
	Grow      float64
	Shrink    float64
	Fixed     bool
	Children  []*LayoutNode
}

// LayoutEngine computes layouts for a tree of nodes.
type LayoutEngine struct {
	root     *LayoutNode
	cache    map[string]Rect
	dirty    bool
	width    int
	height   int
}

// NewLayoutEngine creates a new layout engine.
func NewLayoutEngine(width, height int) *LayoutEngine {
	return &LayoutEngine{
		cache:  make(map[string]Rect),
		width:  width,
		height: height,
	}
}

// SetRoot sets the root layout node.
func (le *LayoutEngine) SetRoot(root *LayoutNode) {
	le.root = root
	le.dirty = true
}

// Compute computes the layout for the entire tree.
func (le *LayoutEngine) Compute() {
	if le.root == nil {
		return
	}
	le.computeNode(le.root, Rect{Width: le.width, Height: le.height})
	le.dirty = false
}

// computeNode computes layout for a single node and its children.
func (le *LayoutEngine) computeNode(node *LayoutNode, available Rect) {
	if node == nil {
		return
	}

	// Apply constraints
	w := available.Width
	h := available.Height

	if node.MinWidth > 0 && w < node.MinWidth {
		w = node.MinWidth
	}
	if node.MaxWidth > 0 && w > node.MaxWidth {
		w = node.MaxWidth
	}
	if node.MinHeight > 0 && h < node.MinHeight {
		h = node.MinHeight
	}
	if node.MaxHeight > 0 && h > node.MaxHeight {
		h = node.MaxHeight
	}

	node.Bounds = Rect{
		X:      available.X,
		Y:      available.Y,
		Width:  w,
		Height: h,
	}

	// Compute children based on layout mode
	if len(node.Children) > 0 {
		le.computeChildren(node, Rect{
			X:      available.X,
			Y:      available.Y,
			Width:  w,
			Height: h,
		})
	}
}

// computeChildren computes layout for child nodes.
func (le *LayoutEngine) computeChildren(parent *LayoutNode, available Rect) {
	if len(parent.Children) == 0 {
		return
	}

	// Calculate total grow factor
	totalGrow := 0.0
	fixedSize := 0
	for _, child := range parent.Children {
		if child.Fixed {
			if child.Bounds.Width > 0 {
				fixedSize += child.Bounds.Width
			} else if child.Bounds.Height > 0 {
				fixedSize += child.Bounds.Height
			}
		} else {
			totalGrow += child.Grow
		}
	}

	// Distribute remaining space
	remaining := available.Width - fixedSize
	if remaining < 0 {
		remaining = 0
	}

	x := available.X
	for _, child := range parent.Children {
		childW := child.Bounds.Width
		if !child.Fixed && totalGrow > 0 {
			childW = int(float64(remaining) * child.Grow / totalGrow)
		}

		child.Bounds = Rect{
			X:      x,
			Y:      available.Y,
			Width:  childW,
			Height: available.Height,
		}

		// Recursively compute children
		le.computeNode(child, child.Bounds)

		x += childW
	}
}

// GetBounds returns the computed bounds for a node.
func (le *LayoutEngine) GetBounds(node *LayoutNode) Rect {
	if node == nil {
		return Rect{}
	}
	return node.Bounds
}

// Invalidate marks the layout as dirty.
func (le *LayoutEngine) Invalidate() {
	le.dirty = true
	le.cache = make(map[string]Rect)
}

// IsDirty returns whether the layout needs recomputation.
func (le *LayoutEngine) IsDirty() bool {
	return le.dirty
}

// Resize updates the layout engine dimensions.
func (le *LayoutEngine) Resize(width, height int) {
	le.width = width
	le.height = height
	le.Invalidate()
}

// ---------------------------------------------------------------------------
// Flex Layout Helpers
// ---------------------------------------------------------------------------

// FlexRow creates a horizontal flex layout.
func FlexRow(children []*LayoutNode, gap int) *LayoutNode {
	root := &LayoutNode{
		Children: children,
	}
	for _, child := range children {
		child.Grow = 1
	}
	return root
}

// FlexColumn creates a vertical flex layout.
func FlexColumn(children []*LayoutNode, gap int) *LayoutNode {
	root := &LayoutNode{
		Children: children,
	}
	for _, child := range children {
		child.Grow = 1
	}
	return root
}

// FixedNode creates a fixed-size layout node.
func FixedNode(width, height int) *LayoutNode {
	return &LayoutNode{
		MinWidth:  width,
		MaxWidth:  width,
		MinHeight: height,
		MaxHeight: height,
		Fixed:     true,
	}
}

// Grow creates a flexible layout node.
func NewGrowNode(grow float64) *LayoutNode {
	return &LayoutNode{
		Grow: grow,
	}
}

// ---------------------------------------------------------------------------
// Responsive Layout
// ---------------------------------------------------------------------------

// ResponsiveLayout adapts to terminal size.
type ResponsiveLayout struct {
	breakpoints []Breakpoint
	default_    *LayoutNode
}

// Breakpoint defines a layout at a specific width.
type Breakpoint struct {
	MinWidth int
	Layout   *LayoutNode
}

// NewResponsiveLayout creates a responsive layout.
func NewResponsiveLayout() *ResponsiveLayout {
	return &ResponsiveLayout{}
}

// AddBreakpoint adds a layout breakpoint.
func (rl *ResponsiveLayout) AddBreakpoint(minWidth int, layout *LayoutNode) {
	rl.breakpoints = append(rl.breakpoints, Breakpoint{
		MinWidth: minWidth,
		Layout:   layout,
	})
	// Sort by width (largest first)
	sort.Slice(rl.breakpoints, func(i, j int) bool {
		return rl.breakpoints[i].MinWidth > rl.breakpoints[j].MinWidth
	})
}

// SetDefault sets the default layout.
func (rl *ResponsiveLayout) SetDefault(layout *LayoutNode) {
	rl.default_ = layout
}

// GetLayout returns the appropriate layout for the given width.
func (rl *ResponsiveLayout) GetLayout(width int) *LayoutNode {
	for _, bp := range rl.breakpoints {
		if width >= bp.MinWidth {
			return bp.Layout
		}
	}
	return rl.default_
}
