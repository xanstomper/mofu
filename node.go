package mofu

// Node is a tree node that wraps a Component.
type Node struct {
	Component Component
	Children  []*Node
	Parent    *Node
}

// NewNode creates a new tree node wrapping the given component.
func NewNode(component Component) *Node {
	return &Node{Component: component}
}

// AddChild adds a child node.
func (n *Node) AddChild(child *Node) {
	child.Parent = n
	n.Children = append(n.Children, child)
}

// RemoveChild removes a child node.
func (n *Node) RemoveChild(child *Node) {
	for i, c := range n.Children {
		if c == child {
			n.Children = append(n.Children[:i], n.Children[i+1:]...)
			child.Parent = nil
			return
		}
	}
}

// MountAll calls Mount on every node in the tree.
func (n *Node) MountAll() []Cmd {
	var cmds []Cmd
	if cmd := n.Component.Mount(); cmd != nil {
		cmds = append(cmds, cmd)
	}
	for _, child := range n.Children {
		childCmds := child.MountAll()
		cmds = append(cmds, childCmds...)
	}
	return cmds
}

// UnmountAll calls Unmount on every node in the tree.
func (n *Node) UnmountAll() {
	n.Component.Unmount()
	for _, child := range n.Children {
		child.UnmountAll()
	}
}

// Tree is the root-level component tree.
type Tree struct {
	Root *Node
}

// NewTree creates a new tree with the given root component.
func NewTree(component Component) *Tree {
	return &Tree{Root: NewNode(component)}
}
