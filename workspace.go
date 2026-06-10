package mofu

import (
	"fmt"
	"sort"
	"sync"
)

type PaneId int

var paneIdCounter PaneId

func newPaneId() PaneId {
	paneIdCounter++
	return paneIdCounter
}

type Pane struct {
	ID        PaneId
	Rect      Rect
	Widget    Node
	Title     string
	Border    BorderStyle
	IsFocused bool
	IsVisible bool
	ZIndex    int
	MinWidth  int
	MinHeight int
	MaxWidth  int
	MaxHeight int
}

type SplitDir int

const (
	SplitHorizontal SplitDir = iota
	SplitVertical
)

type SplitOp struct {
	Dir    SplitDir
	PaneId PaneId
	Ratio  float64
	NewId  PaneId
}

type PaneManager struct {
	mu         sync.Mutex
	Panes      map[PaneId]*Pane
	FocusOrder []PaneId
	History    []SplitOp
	MaxHistory int
	Root       Node
	nextZ      int
}

func NewPaneManager(root Node) *PaneManager {
	return &PaneManager{
		Panes:      make(map[PaneId]*Pane),
		FocusOrder: make([]PaneId, 0),
		History:    make([]SplitOp, 0, 50),
		MaxHistory: 50,
		Root:       root,
	}
}

func (pm *PaneManager) Add(widget Node, rect Rect) PaneId {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	id := newPaneId()
	pane := &Pane{
		ID:        id,
		Rect:      rect,
		Widget:    widget,
		Border:    BorderRounded,
		IsVisible: true,
		MinWidth:  10,
		MinHeight: 3,
		ZIndex:    pm.nextZ,
	}
	pm.nextZ++
	pm.Panes[id] = pane
	pm.FocusOrder = append(pm.FocusOrder, id)
	pm.Root.AddChild(widget)
	return id
}

func (pm *PaneManager) AddWithTitle(widget Node, title string, rect Rect) PaneId {
	id := pm.Add(widget, rect)
	pm.mu.Lock()
	pm.Panes[id].Title = title
	pm.mu.Unlock()
	return id
}

func (pm *PaneManager) Split(id PaneId, dir SplitDir, ratio float64) (PaneId, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pane, ok := pm.Panes[id]
	if !ok {
		return 0, fmt.Errorf("pane %d not found", id)
	}

	var newRect Rect
	switch dir {
	case SplitHorizontal:
		splitW := int(float64(pane.Rect.Width) * ratio)
		if splitW < pane.MinWidth || pane.Rect.Width-splitW < pane.MinWidth {
			return 0, fmt.Errorf("split would create pane below min width")
		}
		newRect = Rect{
			X: pane.Rect.X + splitW, Y: pane.Rect.Y,
			Width: pane.Rect.Width - splitW, Height: pane.Rect.Height,
		}
		pane.Rect.Width = splitW
	case SplitVertical:
		splitH := int(float64(pane.Rect.Height) * ratio)
		if splitH < pane.MinHeight || pane.Rect.Height-splitH < pane.MinHeight {
			return 0, fmt.Errorf("split would create pane below min height")
		}
		newRect = Rect{
			X: pane.Rect.X, Y: pane.Rect.Y + splitH,
			Width: pane.Rect.Width, Height: pane.Rect.Height - splitH,
		}
		pane.Rect.Height = splitH
	}

	newId := newPaneId()
	newPane := &Pane{
		ID: newId, Rect: newRect, Widget: NewBox(NewText("")),
		Border: pane.Border, IsVisible: true,
		MinWidth: 10, MinHeight: 3, ZIndex: pm.nextZ,
	}
	pm.nextZ++
	pm.Panes[newId] = newPane

	idx := indexOf(pm.FocusOrder, id)
	if idx >= 0 {
		insertAfter := make([]PaneId, 0, len(pm.FocusOrder)+1)
		insertAfter = append(insertAfter, pm.FocusOrder[:idx+1]...)
		insertAfter = append(insertAfter, newId)
		insertAfter = append(insertAfter, pm.FocusOrder[idx+1:]...)
		pm.FocusOrder = insertAfter
	}

	pm.History = append(pm.History, SplitOp{Dir: dir, PaneId: id, Ratio: ratio, NewId: newId})
	if len(pm.History) > pm.MaxHistory {
		pm.History = pm.History[1:]
	}

	return newId, nil
}

func (pm *PaneManager) Close(id PaneId) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if len(pm.Panes) <= 1 {
		return fmt.Errorf("cannot close last pane")
	}

	pane, ok := pm.Panes[id]
	if !ok {
		return fmt.Errorf("pane %d not found", id)
	}

	for _, other := range pm.Panes {
		if other.ID != id && other.ID != 0 {
			adj := pm.findAdjacent(pane.Rect, other.Rect)
			if adj != 0 {
				other.Rect.Width += adj
			}
			break
		}
	}

	pane.Widget.Unmount()
	pm.Root.RemoveChild(pane.Widget)
	delete(pm.Panes, id)

	pm.FocusOrder = filter(pm.FocusOrder, func(pid PaneId) bool { return pid != id })
	if len(pm.FocusOrder) > 0 {
		pm.Panes[pm.FocusOrder[len(pm.FocusOrder)-1]].IsFocused = true
	}

	return nil
}

func (pm *PaneManager) findAdjacent(a, b Rect) int {
	if a.Y+a.Height >= b.Y && a.Y <= b.Y+b.Height {
		if b.X+b.Width == a.X {
			return a.Width
		}
		if a.X+a.Width == b.X {
			return b.Width
		}
	}
	if a.X+a.Width >= b.X && a.X <= b.X+b.Width {
		if b.Y+b.Height == a.Y {
			return a.Height
		}
		if a.Y+a.Height == b.Y {
			return b.Height
		}
	}
	return 0
}

func (pm *PaneManager) Focus(id PaneId) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for _, p := range pm.Panes {
		p.IsFocused = p.ID == id
	}
	idx := indexOf(pm.FocusOrder, id)
	if idx >= 0 {
		pm.FocusOrder = append(append(pm.FocusOrder[:idx], pm.FocusOrder[idx+1:]...), id)
	}
}

func (pm *PaneManager) FocusNext() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if len(pm.FocusOrder) == 0 {
		return
	}
	current := pm.FocusOrder[len(pm.FocusOrder)-1]
	var next PaneId
	found := false
	for _, id := range pm.FocusOrder {
		if found {
			next = id
			break
		}
		if id == current {
			found = true
		}
	}
	if !found || next == 0 {
		next = pm.FocusOrder[0]
	}
	for _, p := range pm.Panes {
		p.IsFocused = p.ID == next
	}
	pm.FocusOrder = append(pm.FocusOrder[:len(pm.FocusOrder)-1], next)
}

func (pm *PaneManager) Render(ctx *RenderContext) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	sorted := make([]*Pane, 0, len(pm.Panes))
	for _, p := range pm.Panes {
		if p.IsVisible {
			sorted = append(sorted, p)
		}
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ZIndex < sorted[j].ZIndex
	})

	r := ctx.Renderer
	for _, pane := range sorted {
		borderStyle := pane.Border
		if pane.IsFocused {
			r.WriteStyledString("● "+pane.Title+" ", pane.Rect.X+2, pane.Rect.Y, ctx.Theme.Typography.Label)
		} else if pane.Title != "" {
			r.WriteStyledString("○ "+pane.Title+" ", pane.Rect.X+2, pane.Rect.Y, ctx.Theme.Typography.Body)
		}

		drawBorder(r, pane.Rect, borderStyle)

		inner := Rect{
			X: pane.Rect.X + 1, Y: pane.Rect.Y + 1,
			Width: pane.Rect.Width - 2, Height: pane.Rect.Height - 2,
		}
		if inner.Width > 0 && inner.Height > 0 {
			pane.Widget.SetBounds(inner)
			wCtx := *ctx
			wCtx.Bounds = inner
			pane.Widget.Render(&wCtx)
		}
	}
}

func drawBorder(r *Renderer, rect Rect, bs BorderStyle) {
	if bs == BorderNone || bs == BorderHidden {
		return
	}
	if rect.Width < 2 || rect.Height < 2 {
		return
	}
	x1, y1 := rect.X, rect.Y
	x2, y2 := rect.X+rect.Width-1, rect.Y+rect.Height-1

	r.front.Set(x1, y1, bs.TopLeft, Color{}, Color{}, 0)
	r.front.Set(x2, y1, bs.TopRight, Color{}, Color{}, 0)
	r.front.Set(x1, y2, bs.BottomLeft, Color{}, Color{}, 0)
	r.front.Set(x2, y2, bs.BottomRight, Color{}, Color{}, 0)

	for x := x1 + 1; x < x2; x++ {
		r.front.Set(x, y1, bs.Top, Color{}, Color{}, 0)
		r.front.Set(x, y2, bs.Bottom, Color{}, Color{}, 0)
	}
	for y := y1 + 1; y < y2; y++ {
		r.front.Set(x1, y, bs.Left, Color{}, Color{}, 0)
		r.front.Set(x2, y, bs.Right, Color{}, Color{}, 0)
	}
}

type Tab struct {
	ID       PaneId
	Title    string
	Icon     rune
	IsDirty  bool
	CanClose bool
}

type TabBar struct {
	Tabs      []Tab
	ActiveIdx int
	X, Y      int
}

func (tb *TabBar) Render(r *Renderer, theme *Theme) {
	x := tb.X
	for i, tab := range tb.Tabs {
		if i == tb.ActiveIdx {
			r.WriteStyledString(" ", x, tb.Y, theme.Typography.Label)
			x++
		}
		label := tab.Title
		if tab.Icon != 0 {
			label = string(tab.Icon) + " " + label
		}
		if tab.IsDirty {
			label = "* " + label
		}
		sty := theme.Typography.Body
		if i == tb.ActiveIdx {
			sty = theme.Typography.Label
			sty.Attrs |= AttrUnderline
		}
		r.WriteStyledString(" "+label+" ", x, tb.Y, sty)
		x += len(label) + 2

		if tab.CanClose {
			r.WriteString("×", x, tb.Y, theme.Colors.TextDim, Color{}, 0)
			x += 2
		} else {
			x++
		}
	}
}

type LayoutStrategy int

const (
	LayoutFlex LayoutStrategy = iota
	LayoutGrid
	LayoutStacked
	LayoutTabbed
)

func (pm *PaneManager) ApplyLayout(strategy LayoutStrategy, bounds Rect) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	visible := make([]*Pane, 0, len(pm.Panes))
	for _, p := range pm.Panes {
		if p.IsVisible {
			visible = append(visible, p)
		}
	}
	if len(visible) == 0 {
		return
	}

	switch strategy {
	case LayoutGrid:
		cols := 2
		if len(visible) <= 2 {
			cols = len(visible)
		}
		rows := (len(visible) + cols - 1) / cols
		pW := bounds.Width / cols
		pH := bounds.Height / rows
		for i, p := range visible {
			col := i % cols
			row := i / cols
			p.Rect = Rect{
				X: bounds.X + col*pW, Y: bounds.Y + row*pH,
				Width: pW, Height: pH,
			}
		}
	case LayoutStacked:
		for i, p := range visible {
			p.Rect = bounds
			if i == len(visible)-1 {
				p.IsVisible = true
			} else {
				p.IsVisible = i == len(visible)-1
			}
		}
	default:
		n := len(visible)
		if n == 1 {
			visible[0].Rect = bounds
			return
		}
		ratio := 1.0 / float64(n)
		for i, p := range visible {
			p.Rect = Rect{
				X:      bounds.X + int(float64(i)*ratio*float64(bounds.Width)),
				Y:      bounds.Y,
				Width:  int(ratio * float64(bounds.Width)),
				Height: bounds.Height,
			}
		}
	}
}

func indexOf(ids []PaneId, id PaneId) int {
	for i, v := range ids {
		if v == id {
			return i
		}
	}
	return -1
}

func filter[T any](s []T, fn func(T) bool) []T {
	out := make([]T, 0, len(s))
	for _, v := range s {
		if fn(v) {
			out = append(out, v)
		}
	}
	return out
}
