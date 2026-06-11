package widgets

import (
	"strings"

	"github.com/xanstomper/mofu"
)

// TreeNode represents a node in a tree structure.
type TreeNode struct {
	Label    string
	Children []*TreeNode
	Expanded bool
	Data     any
}

// Tree displays hierarchical data with expand/collapse.
type Tree struct {
	mofu.BaseNode
	Root     *TreeNode
	Selected *TreeNode
	Offset   int
	Focused  bool
	OnSelect func(node *TreeNode) mofu.Cmd
	Style    mofu.Style
	FocusStyle mofu.Style
}

// NewTree creates a tree widget.
func NewTree(root *TreeNode) *Tree {
	return &Tree{
		Root: root,
		Style: mofu.DefaultStyle().Fg(mofu.Hex("e0e0e0")),
		FocusStyle: mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")),
	}
}

func (t *Tree) Focus()   { t.Focused = true; t.SetDirty() }
func (t *Tree) Blur()    { t.Focused = false; t.SetDirty() }
func (t *Tree) IsFocused() bool { return t.Focused }

func (t *Tree) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	if r.Width <= 0 || r.Height <= 0 || t.Root == nil {
		return
	}

	y := r.Y
	var renderNode func(node *TreeNode, depth int)
	renderNode = func(node *TreeNode, depth int) {
		if y >= r.Y+r.Height {
			return
		}

		indent := strings.Repeat("  ", depth)
		icon := "├─"
		if len(node.Children) == 0 {
			icon = "└─"
		}
		if node.Expanded {
			icon = "▼ "
		}

		text := indent + icon + " " + node.Label
		if len(text) > r.Width-2 {
			text = text[:r.Width-5] + "..."
		}

		style := t.Style
		if node == t.Selected && t.Focused {
			style = t.FocusStyle
			ctx.Renderer.WriteString(strings.Repeat(" ", r.Width-2), r.X+1, y, mofu.ColorWhite, mofu.Hex("2a2a3e"), 0)
		}

		ctx.Renderer.WriteString(text, r.X+1, y, style.Foreground, style.Background, style.Attrs)
		y++

		if node.Expanded {
			for _, child := range node.Children {
				renderNode(child, depth+1)
			}
		}
	}

	renderNode(t.Root, 0)
}

func (t *Tree) HandleEvent(event mofu.Event) mofu.Cmd {
	if !t.Focused || event.Type != mofu.EventKeyPress {
		return nil
	}
	ke, ok := event.Data.(mofu.KeyEvent)
	if !ok {
		return nil
	}

	switch ke.Key {
	case mofu.KeyDown:
		t.moveDown()
	case mofu.KeyUp:
		t.moveUp()
	case mofu.KeyLeft:
		t.collapse()
	case mofu.KeyRight:
		t.expand()
	case mofu.KeyEnter:
		if t.OnSelect != nil && t.Selected != nil {
			return t.OnSelect(t.Selected)
		}
	}

	// vim bindings
	for _, b := range ke.Runes {
		switch b {
		case 'j':
			t.moveDown()
		case 'k':
			t.moveUp()
		case 'h':
			t.collapse()
		case 'l':
			t.expand()
		}
	}

	t.SetDirty()
	return nil
}

func (t *Tree) moveDown() {
	if t.Selected == nil {
		t.Selected = t.Root
		return
	}
	// Simple: just select next visible node
	t.Selected = t.findNext(t.Selected)
}

func (t *Tree) moveUp() {
	if t.Selected == nil {
		t.Selected = t.Root
		return
	}
	t.Selected = t.findPrev(t.Selected)
}

func (t *Tree) expand() {
	if t.Selected != nil && len(t.Selected.Children) > 0 {
		t.Selected.Expanded = true
	}
}

func (t *Tree) collapse() {
	if t.Selected != nil && t.Selected.Expanded {
		t.Selected.Expanded = false
	}
}

func (t *Tree) findNext(node *TreeNode) *TreeNode {
	if node == nil || t.Root == nil {
		return t.Root
	}
	// If expanded and has children, go to first child
	if node.Expanded && len(node.Children) > 0 {
		return node.Children[0]
	}
	// Otherwise, go to next sibling or parent's next sibling
	return t.findNextSibling(t.Root, node)
}

func (t *Tree) findPrev(node *TreeNode) *TreeNode {
	if node == nil || t.Root == nil {
		return t.Root
	}
	return t.findPrevNode(t.Root, node, nil)
}

func (t *Tree) findNextSibling(root, target *TreeNode) *TreeNode {
	if root == nil {
		return nil
	}
	for i, child := range root.Children {
		if child == target {
			if i+1 < len(root.Children) {
				return root.Children[i+1]
			}
			return t.findNextSibling(nil, root)
		}
		if found := t.findNextSibling(child, target); found != nil {
			return found
		}
	}
	return nil
}

func (t *Tree) findPrevNode(root, target, prev *TreeNode) *TreeNode {
	if root == nil {
		return nil
	}
	if root == target {
		return prev
	}
	for _, child := range root.Children {
		if found := t.findPrevNode(child, target, root); found != nil {
			return found
		}
	}
	return nil
}

func (t *Tree) Mount() mofu.Cmd { return nil }
func (t *Tree) Unmount()        {}
