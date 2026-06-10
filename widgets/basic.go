package widgets

import (
	"strings"

	"github.com/anomalyco/mofu"
)

type TextNode struct {
	mofu.BaseNode
	Content string
}

func NewText(content string) *TextNode {
	return &TextNode{Content: content}
}

func (t *TextNode) Render(ctx *mofu.RenderContext) {
	if t.Content == "" {
		return
	}
	r := ctx.Bounds
	ctx.Renderer.WriteStyledString(t.Content, r.X, r.Y, *t.Style())
}

type BoxNode struct {
	mofu.BaseNode
	Child mofu.Node
}

func NewBox(child mofu.Node) *BoxNode {
	return &BoxNode{Child: child}
}

func (b *BoxNode) Children() []mofu.Node {
	if b.Child == nil {
		return nil
	}
	return []mofu.Node{b.Child}
}

func (b *BoxNode) Render(ctx *mofu.RenderContext) {
	if b.Child == nil {
		return
	}
	s := b.Style()
	r := ctx.Bounds
	inner := mofu.Rect{
		X:      r.X + s.Padding.Left + s.Margin.Left,
		Y:      r.Y + s.Padding.Top + s.Margin.Top,
		Width:  r.Width - s.Padding.Left - s.Padding.Right - s.Margin.Left - s.Margin.Right,
		Height: r.Height - s.Padding.Top - s.Padding.Bottom - s.Margin.Top - s.Margin.Bottom,
	}
	b.Child.SetBounds(inner)
	childCtx := *ctx
	childCtx.Bounds = inner
	b.Child.Render(&childCtx)
}

func (b *BoxNode) HandleEvent(event mofu.Event) mofu.Cmd {
	if b.Child != nil {
		return b.Child.HandleEvent(event)
	}
	return nil
}

func (b *BoxNode) Mount() mofu.Cmd {
	if b.Child != nil {
		return b.Child.Mount()
	}
	return nil
}

func (b *BoxNode) Unmount() {
	if b.Child != nil {
		b.Child.Unmount()
	}
}

type StackNode struct {
	mofu.BaseNode
	ChildrenList []mofu.Node
}

func NewColumn(children ...mofu.Node) *StackNode {
	return &StackNode{ChildrenList: children}
}

func NewRow(children ...mofu.Node) *StackNode {
	s := &StackNode{ChildrenList: children}
	s.Style().Direction = mofu.DirectionRow
	return s
}

func (s *StackNode) Children() []mofu.Node { return s.ChildrenList }

func (s *StackNode) Render(ctx *mofu.RenderContext) {
	for _, child := range s.ChildrenList {
		childCtx := *ctx
		childCtx.Bounds = child.Bounds()
		child.Render(&childCtx)
	}
}

func (s *StackNode) HandleEvent(event mofu.Event) mofu.Cmd {
	for _, child := range s.ChildrenList {
		if cmd := child.HandleEvent(event); cmd != nil {
			return cmd
		}
	}
	return nil
}

func (s *StackNode) Mount() mofu.Cmd {
	var cmds []mofu.Cmd
	for _, child := range s.ChildrenList {
		if cmd := child.Mount(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return mofu.Batch(cmds...)
}

func (s *StackNode) Unmount() {
	for _, child := range s.ChildrenList {
		child.Unmount()
	}
}

type SpacerNode struct{ mofu.BaseNode }

func NewSpacer() *SpacerNode { return &SpacerNode{} }

type DividerNode struct {
	mofu.BaseNode
	Char rune
}

func NewDivider(char rune) *DividerNode {
	return &DividerNode{Char: char}
}

func (d *DividerNode) Render(ctx *mofu.RenderContext) {
	line := strings.Repeat(string(d.Char), 40)
	ctx.Renderer.WriteStyledString("\n"+line+"\n", ctx.Bounds.X, ctx.Bounds.Y, *d.Style())
}
