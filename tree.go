package mofu

import (
	"fmt"
	"strings"
	"sync"
)

// ---------------------------------------------------------------------------
// Tree-Based Rendering Engine
// ---------------------------------------------------------------------------
// MOFU uses a scene-graph approach with reactive dirty-bit propagation,
// MOFU maintains an actual tree of nodes with:
// - Efficient diffing (only changed subtrees re-render)
// - Precise event targeting (events go to specific nodes)
// - Incremental updates (no full tree rebuild)
// - Memory-efficient (no string concatenation)

// TreeNode is a node in the rendering tree.
type TreeNode struct {
	ID        string
	Type      string
	Props     map[string]any
	Children  []*TreeNode
	Parent    *TreeNode
	Style     *Style
	Bounds    Rect
	Visible   bool
	Focused   bool
	Dirty     bool
	zIndex    int
	listeners map[string][]func(Event)
}

// NewTreeNode creates a new tree node.
func NewTreeNode(id, nodeType string) *TreeNode {
	return &TreeNode{
		ID:        id,
		Type:      nodeType,
		Props:     make(map[string]any),
		Children:  make([]*TreeNode, 0),
		Visible:   true,
		listeners: make(map[string][]func(Event)),
	}
}

// SetProp sets a property on the node.
func (n *TreeNode) SetProp(key string, value any) {
	n.Props[key] = value
	n.Dirty = true
}

// GetProp gets a property from the node.
func (n *TreeNode) GetProp(key string) any {
	return n.Props[key]
}

// AddChild adds a child node.
func (n *TreeNode) AddChild(child *TreeNode) {
	child.Parent = n
	n.Children = append(n.Children, child)
	n.Dirty = true
}

// RemoveChild removes a child node.
func (n *TreeNode) RemoveChild(id string) {
	for i, child := range n.Children {
		if child.ID == id {
			n.Children = append(n.Children[:i], n.Children[i+1:]...)
			n.Dirty = true
			return
		}
	}
}

// Find finds a node by ID in the subtree.
func (n *TreeNode) Find(id string) *TreeNode {
	if n.ID == id {
		return n
	}
	for _, child := range n.Children {
		if found := child.Find(id); found != nil {
			return found
		}
	}
	return nil
}

// FindByType finds all nodes of a given type.
func (n *TreeNode) FindByType(nodeType string) []*TreeNode {
	var result []*TreeNode
	if n.Type == nodeType {
		result = append(result, n)
	}
	for _, child := range n.Children {
		result = append(result, child.FindByType(nodeType)...)
	}
	return result
}

// On adds an event listener.
func (n *TreeNode) On(event string, handler func(Event)) {
	n.listeners[event] = append(n.listeners[event], handler)
}

// Emit emits an event to listeners.
func (n *TreeNode) Emit(event Event) {
	eventType := fmt.Sprintf("%d", event.Type)
	for _, handler := range n.listeners[eventType] {
		handler(event)
	}
	// Bubble up to parent
	if n.Parent != nil {
		n.Parent.Emit(event)
	}
}

// ---------------------------------------------------------------------------
// Tree Diffing Engine
// ---------------------------------------------------------------------------

// DiffResult describes a change between two tree states.
type DiffResult struct {
	Type    string // "add", "remove", "update", "move"
	Node    *TreeNode
	Old     *TreeNode
	Changes map[string]any
}

// DiffTrees computes the minimal set of changes between two trees.
func DiffTrees(old, new *TreeNode) []DiffResult {
	var results []DiffResult

	if old == nil && new != nil {
		results = append(results, DiffResult{Type: "add", Node: new})
		return results
	}
	if old != nil && new == nil {
		results = append(results, DiffResult{Type: "remove", Node: old})
		return results
	}
	if old == nil || new == nil {
		return results
	}

	// Compare properties
	changes := DiffProps(old.Props, new.Props)
	if len(changes) > 0 {
		results = append(results, DiffResult{
			Type:    "update",
			Node:    new,
			Old:     old,
			Changes: changes,
		})
	}

	// Compare children
	oldMap := make(map[string]*TreeNode)
	for _, child := range old.Children {
		oldMap[child.ID] = child
	}

	newMap := make(map[string]*TreeNode)
	for _, child := range new.Children {
		newMap[child.ID] = child
	}

	// Find added and updated children
	for id, newChild := range newMap {
		if oldChild, exists := oldMap[id]; exists {
			// Child exists - recurse
			results = append(results, DiffTrees(oldChild, newChild)...)
		} else {
			// New child
			results = append(results, DiffResult{Type: "add", Node: newChild})
		}
	}

	// Find removed children
	for id, oldChild := range oldMap {
		if _, exists := newMap[id]; !exists {
			results = append(results, DiffResult{Type: "remove", Node: oldChild})
		}
	}

	return results
}

// DiffProps compares two property maps and returns changes.
func DiffProps(old, new map[string]any) map[string]any {
	changes := make(map[string]any)

	// Check for changed or added props
	for key, newVal := range new {
		oldVal, exists := old[key]
		if !exists || fmt.Sprintf("%v", oldVal) != fmt.Sprintf("%v", newVal) {
			changes[key] = newVal
		}
	}

	// Check for removed props
	for key := range old {
		if _, exists := new[key]; !exists {
			changes[key] = nil // nil means removed
		}
	}

	return changes
}

// ---------------------------------------------------------------------------
// Tree Renderer
// ---------------------------------------------------------------------------

// TreeRenderer renders a tree of nodes to the terminal.
type TreeRenderer struct {
	root      *TreeNode
	renderer  *Renderer
	theme     *Theme
	frame     int64
	delta     int64
	dirtyMap  map[string]bool
	mu        sync.RWMutex
}

// NewTreeRenderer creates a new tree renderer.
func NewTreeRenderer(root *TreeNode, renderer *Renderer, theme *Theme) *TreeRenderer {
	return &TreeRenderer{
		root:     root,
		renderer: renderer,
		theme:    theme,
		dirtyMap: make(map[string]bool),
	}
}

// Render renders the entire tree.
func (tr *TreeRenderer) Render(bounds Rect) {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	tr.frame++
	tr.renderNode(tr.root, bounds)
}

// renderNode renders a single node and its children.
func (tr *TreeRenderer) renderNode(node *TreeNode, bounds Rect) {
	if node == nil || !node.Visible {
		return
	}

	// Update bounds
	node.Bounds = bounds

	// Render based on type
	switch node.Type {
	case "text":
		tr.renderText(node, bounds)
	case "box":
		tr.renderBox(node, bounds)
	case "row":
		tr.renderRow(node, bounds)
	case "column":
		tr.renderColumn(node, bounds)
	case "input":
		tr.renderInput(node, bounds)
	case "button":
		tr.renderButton(node, bounds)
	case "list":
		tr.renderList(node, bounds)
	case "table":
		tr.renderTable(node, bounds)
	case "progress":
		tr.renderProgress(node, bounds)
	default:
		// Custom rendering via props
		if renderFn, ok := node.Props["render"].(func(*TreeNode, *RenderContext)); ok {
			ctx := &RenderContext{
				Renderer: tr.renderer,
				Theme:    tr.theme,
				Frame:    tr.frame,
				Bounds:   bounds,
			}
			renderFn(node, ctx)
		}
	}

	// Render children
	for _, child := range node.Children {
		childBounds := tr.computeChildBounds(node, child, bounds)
		tr.renderNode(child, childBounds)
	}
}

// computeChildBounds computes bounds for a child node.
func (tr *TreeRenderer) computeChildBounds(parent, child *TreeNode, parentBounds Rect) Rect {
	// Use child's explicit bounds if set
	if child.Bounds.Width > 0 && child.Bounds.Height > 0 {
		return child.Bounds
	}
	// Default: fill parent
	return parentBounds
}

// renderText renders a text node.
func (tr *TreeRenderer) renderText(node *TreeNode, bounds Rect) {
	text, _ := node.Props["text"].(string)
	if text == "" {
		return
	}

	style := DefaultStyle()
	if s, ok := node.Props["style"].(Style); ok {
		style = s
	}

	tr.renderer.WriteString(text, bounds.X, bounds.Y, style.Foreground, style.Background, style.Attrs)
}

// renderBox renders a box container.
func (tr *TreeRenderer) renderBox(node *TreeNode, bounds Rect) {
	if border, ok := node.Props["border"].(BorderStyle); ok && border != (BorderStyle{}) {
		// Draw border
		style := DefaultStyle().Fg(Hex("444444"))
		if s, ok := node.Props["borderStyle"].(Style); ok {
			style = s
		}
		tr.renderer.WriteStyledString(
			string(border.TopLeft)+strings.Repeat(string(border.Top), bounds.Width-2)+string(border.TopRight),
			bounds.X, bounds.Y, style,
		)
		tr.renderer.WriteStyledString(
			string(border.BottomLeft)+strings.Repeat(string(border.Bottom), bounds.Width-2)+string(border.BottomRight),
			bounds.X, bounds.Y+bounds.Height-1, style,
		)
		for y := bounds.Y + 1; y < bounds.Y+bounds.Height-1; y++ {
			tr.renderer.WriteStyledString(string(border.Left), bounds.X, y, style)
			tr.renderer.WriteStyledString(string(border.Right), bounds.X+bounds.Width-1, y, style)
		}
	}
}

// renderRow renders children in a horizontal row.
func (tr *TreeRenderer) renderRow(node *TreeNode, bounds Rect) {
	gap, _ := node.Props["gap"].(int)
	if gap == 0 {
		gap = 1
	}

	x := bounds.X
	for _, child := range node.Children {
		childW := 10
		if w, ok := child.Props["width"].(int); ok {
			childW = w
		}
		childBounds := Rect{X: x, Y: bounds.Y, Width: childW, Height: bounds.Height}
		tr.renderNode(child, childBounds)
		x += childW + gap
	}
}

// renderColumn renders children in a vertical column.
func (tr *TreeRenderer) renderColumn(node *TreeNode, bounds Rect) {
	gap, _ := node.Props["gap"].(int)
	if gap == 0 {
		gap = 1
	}

	y := bounds.Y
	for _, child := range node.Children {
		childH := 1
		if h, ok := child.Props["height"].(int); ok {
			childH = h
		}
		childBounds := Rect{X: bounds.X, Y: y, Width: bounds.Width, Height: childH}
		tr.renderNode(child, childBounds)
		y += childH + gap
	}
}

// renderInput renders an input field.
func (tr *TreeRenderer) renderInput(node *TreeNode, bounds Rect) {
	value, _ := node.Props["value"].(string)
	placeholder, _ := node.Props["placeholder"].(string)
	focused, _ := node.Props["focused"].(bool)

	text := value
	if text == "" {
		text = placeholder
	}

	style := DefaultStyle()
	if focused {
		style = DefaultStyle().Fg(Hex("ff69b4"))
	}

	tr.renderer.WriteString("["+text+"]", bounds.X, bounds.Y, style.Foreground, style.Background, style.Attrs)
}

// renderButton renders a button.
func (tr *TreeRenderer) renderButton(node *TreeNode, bounds Rect) {
	label, _ := node.Props["label"].(string)
	focused, _ := node.Props["focused"].(bool)

	style := DefaultStyle().Fg(Hex("e0e0e0")).Bg(Hex("333333"))
	if focused {
		style = DefaultStyle().Fg(Hex("ffffff")).Bg(Hex("ff69b4"))
	}

	text := "[" + label + "]"
	tr.renderer.WriteString(text, bounds.X, bounds.Y, style.Foreground, style.Background, style.Attrs)
}

// renderList renders a list.
func (tr *TreeRenderer) renderList(node *TreeNode, bounds Rect) {
	items, _ := node.Props["items"].([]string)
	selected, _ := node.Props["selected"].(int)

	for i, item := range items {
		if i >= bounds.Height {
			break
		}
		style := DefaultStyle()
		if i == selected {
			style = DefaultStyle().Fg(Hex("ff69b4"))
			tr.renderer.WriteString("▸ ", bounds.X, bounds.Y+i, style.Foreground, style.Background, style.Attrs)
		}
		tr.renderer.WriteString(item, bounds.X+2, bounds.Y+i, style.Foreground, style.Background, style.Attrs)
	}
}

// renderTable renders a table.
func (tr *TreeRenderer) renderTable(node *TreeNode, bounds Rect) {
	columns, _ := node.Props["columns"].([]string)
	rows, _ := node.Props["rows"].([][]string)

	if len(columns) > 0 {
		header := strings.Join(columns, " | ")
		tr.renderer.WriteString(header, bounds.X, bounds.Y, DefaultStyle().WithAttrs(AttrBold).Foreground, DefaultStyle().WithAttrs(AttrBold).Background, DefaultStyle().WithAttrs(AttrBold).Attrs)
	}

	for i, row := range rows {
		if i+1 >= bounds.Height {
			break
		}
		text := strings.Join(row, " | ")
		tr.renderer.WriteString(text, bounds.X, bounds.Y+i+1, DefaultStyle().Foreground, DefaultStyle().Background, DefaultStyle().Attrs)
	}
}

// renderProgress renders a progress bar.
func (tr *TreeRenderer) renderProgress(node *TreeNode, bounds Rect) {
	value, _ := node.Props["value"].(float64)
	filled := int(value * float64(bounds.Width))
	empty := bounds.Width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	tr.renderer.WriteString(bar, bounds.X, bounds.Y, Hex("a6e3a1"), ColorBlack, 0)
}

// UpdateTree updates the tree and renders changes.
func (tr *TreeRenderer) UpdateTree(newRoot *TreeNode) {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	if tr.root == nil {
		tr.root = newRoot
		tr.Render(Rect{Width: 80, Height: 24})
		return
	}

	// Diff and apply changes
	results := DiffTrees(tr.root, newRoot)
	for _, result := range results {
		switch result.Type {
		case "add":
			if result.Node != nil {
				tr.renderNode(result.Node, result.Node.Bounds)
			}
		case "remove":
			if result.Node != nil {
				// Clear the node's area
				tr.clearNode(result.Node)
			}
		case "update":
			if result.Node != nil {
				tr.renderNode(result.Node, result.Node.Bounds)
			}
		}
	}

	tr.root = newRoot
}

// clearNode clears a node's rendered area.
func (tr *TreeRenderer) clearNode(node *TreeNode) {
	if node == nil {
		return
	}
	for y := node.Bounds.Y; y < node.Bounds.Y+node.Bounds.Height; y++ {
		for x := node.Bounds.X; x < node.Bounds.X+node.Bounds.Width; x++ {
			tr.renderer.WriteString(" ", x, y, ColorBlack, ColorBlack, 0)
		}
	}
}

// Root returns the root node.
func (tr *TreeRenderer) Root() *TreeNode {
	return tr.root
}

// FindNode finds a node by ID.
func (tr *TreeRenderer) FindNode(id string) *TreeNode {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	if tr.root == nil {
		return nil
	}
	return tr.root.Find(id)
}
