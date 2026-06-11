package widgets

import (
	"strings"

	"github.com/xanstomper/mofu"
)

// ---------------------------------------------------------------------------
// Widget Lifecycle (Anthology Ch.9 §9.1)
// ---------------------------------------------------------------------------

// WidgetID uniquely identifies a widget in the tree.
type WidgetID uint64

// WidgetState tracks a widget's lifecycle phase.
type WidgetState uint8

const (
	WidgetUnmounted WidgetState = iota
	WidgetMounted
	WidgetFocused
	WidgetDisabled
)

// Focusable can be implemented by widgets that accept focus.
type Focusable interface {
	Focus()
	Blur()
	IsFocused() bool
}

// Selectable can be implemented by widgets that support selection.
type Selectable interface {
	Select()
	Deselect()
	IsSelected() bool
}

// ---------------------------------------------------------------------------
// Container (Anthology Ch.9 §9.2)
// ---------------------------------------------------------------------------

// Container wraps a single child with border, padding, margin, and background.
type Container struct {
	mofu.BaseNode
	Child       mofu.Node
	BorderStyle mofu.BorderStyle
	Background  *mofu.Color
	Title       string
}

// NewContainer returns a Container wrapping child.
func NewContainer(child mofu.Node) *Container {
	return &Container{Child: child, BorderStyle: mofu.BorderNormal}
}

func (c *Container) Children() []mofu.Node {
	if c.Child == nil {
		return nil
	}
	return []mofu.Node{c.Child}
}

func (c *Container) Render(ctx *mofu.RenderContext) {
	s := c.Style()
	r := ctx.Bounds

	// Draw border if set
	if c.BorderStyle != (mofu.BorderStyle{}) {
		drawBoxBorder(ctx, r, c.BorderStyle, s.Foreground)
		// Title in border
		if c.Title != "" {
			titleX := r.X + 2
			if titleX < r.X+r.Width {
				ctx.Renderer.WriteStyledString(
					" "+c.Title+" ",
					titleX, r.Y,
					s.Fg(s.Foreground),
				)
			}
		}
	}

	// Inner rect after border + padding + margin
	bw := 0
	if c.BorderStyle != (mofu.BorderStyle{}) {
		bw = 1
	}
	inner := mofu.Rect{
		X:      r.X + s.Margin.Left + s.Padding.Left + bw,
		Y:      r.Y + s.Margin.Top + s.Padding.Top + bw,
		Width:  r.Width - s.Margin.Left - s.Margin.Right - s.Padding.Left - s.Padding.Right - bw*2,
		Height: r.Height - s.Margin.Top - s.Margin.Bottom - s.Padding.Top - s.Padding.Bottom - bw*2,
	}
	if inner.Width < 0 {
		inner.Width = 0
	}
	if inner.Height < 0 {
		inner.Height = 0
	}
	if c.Child != nil {
		c.Child.SetBounds(inner)
		childCtx := *ctx
		childCtx.Bounds = inner
		c.Child.Render(&childCtx)
	}
}

func (c *Container) HandleEvent(event mofu.Event) mofu.Cmd {
	if c.Child != nil {
		return c.Child.HandleEvent(event)
	}
	return nil
}

func (c *Container) Mount() mofu.Cmd {
	if c.Child != nil {
		return c.Child.Mount()
	}
	return nil
}

func (c *Container) Unmount() {
	if c.Child != nil {
		c.Child.Unmount()
	}
}

// ---------------------------------------------------------------------------
// Overlay (Anthology Ch.9 §9.2)
// ---------------------------------------------------------------------------

// Overlay renders children stacked (z-ordered).
type Overlay struct {
	mofu.BaseNode
	ChildrenList []mofu.Node
}

// NewOverlay returns an Overlay.
func NewOverlay(children ...mofu.Node) *Overlay {
	return &Overlay{ChildrenList: children}
}

func (o *Overlay) Children() []mofu.Node { return o.ChildrenList }

func (o *Overlay) Render(ctx *mofu.RenderContext) {
	for _, child := range o.ChildrenList {
		childCtx := *ctx
		childCtx.Bounds = child.Bounds()
		child.Render(&childCtx)
	}
}

func (o *Overlay) HandleEvent(event mofu.Event) mofu.Cmd {
	// Events go to the topmost (last) child first (capture)
	for i := len(o.ChildrenList) - 1; i >= 0; i-- {
		if cmd := o.ChildrenList[i].HandleEvent(event); cmd != nil {
			return cmd
		}
	}
	return nil
}

func (o *Overlay) Mount() mofu.Cmd {
	var cmds []mofu.Cmd
	for _, child := range o.ChildrenList {
		if cmd := child.Mount(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return mofu.Batch(cmds...)
}

func (o *Overlay) Unmount() {
	for _, child := range o.ChildrenList {
		child.Unmount()
	}
}

// ---------------------------------------------------------------------------
// Flex (Anthology Ch.9 §9.2)
// ---------------------------------------------------------------------------

// FlexDirection controls flex layout direction.
type FlexDirection uint8

const (
	FlexRow FlexDirection = iota
	FlexColumn
)

// FlexAlign controls cross-axis alignment.
type FlexAlign uint8

const (
	FlexAlignStart FlexAlign = iota
	FlexAlignCenter
	FlexAlignEnd
	FlexAlignStretch
)

// FlexChild is a single item in a Flex container.
type FlexChild struct {
	Node   mofu.Node
	Grow   float64
	Shrink float64
	Basis  int
}

// Flex lays out children with grow/shrink/basis semantics.
type Flex struct {
	mofu.BaseNode
	Direction FlexDirection
	Items     []FlexChild
	Gap       int
	Align     FlexAlign
}

// NewFlexRow returns a horizontal Flex.
func NewFlexRow(items ...FlexChild) *Flex {
	return &Flex{Direction: FlexRow, Items: items}
}

// NewFlexColumn returns a vertical Flex.
func NewFlexColumn(items ...FlexChild) *Flex {
	return &Flex{Direction: FlexColumn, Items: items}
}

func (f *Flex) Children() []mofu.Node {
	out := make([]mofu.Node, len(f.Items))
	for i, item := range f.Items {
		out[i] = item.Node
	}
	return out
}

func (f *Flex) Render(ctx *mofu.RenderContext) {
	for _, item := range f.Items {
		childCtx := *ctx
		childCtx.Bounds = item.Node.Bounds()
		item.Node.Render(&childCtx)
	}
}

func (f *Flex) HandleEvent(event mofu.Event) mofu.Cmd {
	for _, item := range f.Items {
		if cmd := item.Node.HandleEvent(event); cmd != nil {
			return cmd
		}
	}
	return nil
}

func (f *Flex) Mount() mofu.Cmd {
	var cmds []mofu.Cmd
	for _, item := range f.Items {
		if cmd := item.Node.Mount(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return mofu.Batch(cmds...)
}

func (f *Flex) Unmount() {
	for _, item := range f.Items {
		item.Node.Unmount()
	}
}

// ---------------------------------------------------------------------------
// RenderCache (Anthology Ch.9 §9.4)
// ---------------------------------------------------------------------------

// RenderCache stores pre-rendered content keyed by a content hash.
// Callers should Invalidate() when state changes.
type RenderCache struct {
	lines []string
	hash  uint64
	dirty bool
	width int
}

// NewRenderCache returns an empty cache.
func NewRenderCache(width int) *RenderCache {
	return &RenderCache{width: width, dirty: true}
}

// GetOrCompute returns cached lines or calls compute to regenerate.
func (rc *RenderCache) GetOrCompute(hash uint64, compute func() []string) []string {
	if !rc.dirty && rc.hash == hash {
		return rc.lines
	}
	rc.lines = compute()
	rc.hash = hash
	rc.dirty = false
	return rc.lines
}

// Invalidate forces recomputation on the next GetOrCompute call.
func (rc *RenderCache) Invalidate() { rc.dirty = true }

// Lines returns the cached lines (may be nil).
func (rc *RenderCache) Lines() []string { return rc.lines }

// ---------------------------------------------------------------------------
// Helper: draw box border
// ---------------------------------------------------------------------------

func drawBoxBorder(ctx *mofu.RenderContext, r mofu.Rect, bs mofu.BorderStyle, fg mofu.Color) {
	if r.Width < 2 || r.Height < 2 {
		return
	}
	style := mofu.DefaultStyle().Fg(fg)
	// Top
	ctx.Renderer.WriteStyledString(
		string(bs.TopLeft)+strings.Repeat(string(bs.Top), r.Width-2)+string(bs.TopRight),
		r.X, r.Y, style,
	)
	// Bottom
	ctx.Renderer.WriteStyledString(
		string(bs.BottomLeft)+strings.Repeat(string(bs.Bottom), r.Width-2)+string(bs.BottomRight),
		r.X, r.Y+r.Height-1, style,
	)
	// Sides
	for y := r.Y + 1; y < r.Y+r.Height-1; y++ {
		ctx.Renderer.WriteStyledString(string(bs.Left), r.X, y, style)
		ctx.Renderer.WriteStyledString(string(bs.Right), r.X+r.Width-1, y, style)
	}
}
