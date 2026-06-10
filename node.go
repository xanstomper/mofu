package mofu

import "time"

type RenderContext struct {
	Renderer *Renderer
	Theme    *Theme
	Frame    int64
	Delta    time.Duration
	Bounds   Rect
}

type Node interface {
	Render(ctx *RenderContext)
	HandleEvent(event Event) Cmd
	Mount() Cmd
	Unmount()
	Children() []Node
	AddChild(child Node)
	RemoveChild(child Node)
	SetDirty()
	Dirty() bool
	Bounds() Rect
	SetBounds(Rect)
	Style() *Style
}

type BaseNode struct {
	style    Style
	bounds   Rect
	dirty    bool
	children []Node
	parent   Node
}

func (n *BaseNode) SetBounds(r Rect) { n.bounds = r }
func (n *BaseNode) Bounds() Rect     { return n.bounds }
func (n *BaseNode) SetDirty()        { n.dirty = true }
func (n *BaseNode) Dirty() bool      { return n.dirty }
func (n *BaseNode) Style() *Style    { return &n.style }
func (n *BaseNode) Children() []Node {
	if n.children == nil {
		return nil
	}
	return n.children
}
func (n *BaseNode) AddChild(child Node) {
	n.children = append(n.children, child)
	n.SetDirty()
}
func (n *BaseNode) RemoveChild(child Node) {
	for i, c := range n.children {
		if c == child {
			n.children = append(n.children[:i], n.children[i+1:]...)
			n.SetDirty()
			return
		}
	}
}
func (n *BaseNode) Render(ctx *RenderContext)   {}
func (n *BaseNode) HandleEvent(event Event) Cmd { return nil }
func (n *BaseNode) Mount() Cmd                  { return nil }
func (n *BaseNode) Unmount()                    {}

type BoxNode struct {
	BaseNode
	children []Node
}

func NewBox(children ...Node) *BoxNode {
	return &BoxNode{children: children}
}
func (n *BoxNode) Render(ctx *RenderContext) {
	if len(n.children) == 0 {
		return
	}
	child := n.children[0]
	r := ctx.Bounds
	s := n.BaseNode.style
	inner := Rect{
		X:      r.X + s.Padding.Left + s.Margin.Left,
		Y:      r.Y + s.Padding.Top + s.Margin.Top,
		Width:  r.Width - s.Padding.Left - s.Padding.Right - s.Margin.Left - s.Margin.Right,
		Height: r.Height - s.Padding.Top - s.Padding.Bottom - s.Margin.Top - s.Margin.Bottom,
	}
	child.SetBounds(inner)
	childCtx := *ctx
	childCtx.Bounds = inner
	child.Render(&childCtx)
}
func (n *BoxNode) Children() []Node { return n.children }
func (n *BoxNode) HandleEvent(event Event) Cmd {
	for _, child := range n.children {
		if cmd := child.HandleEvent(event); cmd != nil {
			return cmd
		}
	}
	return nil
}
func (n *BoxNode) Mount() Cmd {
	var cmds []Cmd
	for _, child := range n.children {
		if cmd := child.Mount(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return Batch(cmds...)
}
func (n *BoxNode) Unmount() {
	for _, child := range n.children {
		child.Unmount()
	}
}

type TextNode struct {
	BaseNode
	Content string
}

func NewText(content string) *TextNode {
	return &TextNode{Content: content}
}
func (n *TextNode) Render(ctx *RenderContext) {
	if n.Content == "" {
		return
	}
	r := ctx.Bounds
	ctx.Renderer.WriteStyledString(n.Content, r.X, r.Y, n.BaseNode.style)
}

type StackNode struct {
	BaseNode
	children []Node
}

func NewRow(children ...Node) *StackNode {
	return &StackNode{children: children}
}
func NewColumn(children ...Node) *StackNode {
	return &StackNode{children: children}
}
func (n *StackNode) Children() []Node { return n.children }
func (n *StackNode) Render(ctx *RenderContext) {
	for _, child := range n.children {
		childCtx := *ctx
		childCtx.Bounds = child.Bounds()
		child.Render(&childCtx)
	}
}
func (n *StackNode) HandleEvent(event Event) Cmd {
	for _, child := range n.children {
		if cmd := child.HandleEvent(event); cmd != nil {
			return cmd
		}
	}
	return nil
}
func (n *StackNode) Mount() Cmd {
	var cmds []Cmd
	for _, child := range n.children {
		if cmd := child.Mount(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return Batch(cmds...)
}
func (n *StackNode) Unmount() {
	for _, child := range n.children {
		child.Unmount()
	}
}

type OverlayNode struct {
	BaseNode
	children []Node
}

func NewOverlay(children ...Node) *OverlayNode {
	return &OverlayNode{children: children}
}
func (n *OverlayNode) Children() []Node { return n.children }
func (n *OverlayNode) Render(ctx *RenderContext) {
	for _, child := range n.children {
		childCtx := *ctx
		childCtx.Bounds = ctx.Bounds
		child.Render(&childCtx)
	}
}
func (n *OverlayNode) HandleEvent(event Event) Cmd {
	for _, child := range n.children {
		if cmd := child.HandleEvent(event); cmd != nil {
			return cmd
		}
	}
	return nil
}
func (n *OverlayNode) Mount() Cmd {
	var cmds []Cmd
	for _, child := range n.children {
		if cmd := child.Mount(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return Batch(cmds...)
}
func (n *OverlayNode) Unmount() {
	for _, child := range n.children {
		child.Unmount()
	}
}

type ScrollNode struct {
	BaseNode
	child              Node
	offsetX, offsetY   int
	contentW, contentH int
}

func NewScroll(child Node) *ScrollNode {
	return &ScrollNode{child: child}
}
func (n *ScrollNode) Children() []Node { return []Node{n.child} }
func (n *ScrollNode) ScrollTo(x, y int) {
	n.offsetX = x
	n.offsetY = y
}
func (n *ScrollNode) ScrollBy(dx, dy int) {
	n.offsetX += dx
	n.offsetY += dy
}
func (n *ScrollNode) Render(ctx *RenderContext) {
	if n.child == nil {
		return
	}
	childCtx := *ctx
	b := ctx.Bounds
	n.contentW = b.Width
	n.contentH = b.Height * 2
	childBounds := Rect{X: b.X - n.offsetX, Y: b.Y - n.offsetY, Width: n.contentW, Height: n.contentH}
	n.child.SetBounds(childBounds)
	childCtx.Bounds = childBounds
	n.child.Render(&childCtx)
}
func (n *ScrollNode) HandleEvent(event Event) Cmd {
	if n.child != nil {
		return n.child.HandleEvent(event)
	}
	return nil
}
func (n *ScrollNode) Mount() Cmd {
	if n.child != nil {
		return n.child.Mount()
	}
	return nil
}
func (n *ScrollNode) Unmount() {
	if n.child != nil {
		n.child.Unmount()
	}
}
