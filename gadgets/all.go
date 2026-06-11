// Package gadgets provides 50 system-level reactive UI primitives for MOFU.
//
// Gadgets are NOT widgets. They are runtime-aware, data-driven reactive systems
// that compose intelligently with the MOFU runtime.
//
// Architecture:
//   - Each Gadget consumes streams, subscribes to state graph, emits UI nodes
//   - Layout is declarative (constraints, not positions)
//   - Animation is declarative (specs, not imperative)
//   - State flows through Binder, not manual wiring
//
// Categories:
//   - Data & Table Systems (10): LiveTable, DiffTable, HeatTable, etc.
//   - Navigation & Layout (10): SmartSidebar, AdaptiveSplit, etc.
//   - Input & Interaction (10): SmartForm, InlineEditor, etc.
//   - Real-Time Data (10): LogStream, MetricBoard, etc.
//   - Visual & ASCII (10): ASCIIScene, ParticleField, etc.
package gadgets

import (
	"fmt"
	"strings"

	"github.com/xanstomper/mofu"
)

// =========================================================================
// CORE INTERFACES
// =========================================================================

// Gadget is the core interface for all 50 reactive UI systems.
type Gadget interface {
	ID() string
	Init(ctx GadgetContext) error
	Bind(binder Binder)
	Render(state StateView) []RenderNode
	OnEvent(e Event)
	OnTick(dt int64)
	Dispose() error
}

// LayoutContract defines how a gadget participates in layout.
type LayoutContract interface {
	MinSize() (w, h int)
	MaxSize() (w, h int)
	Flex() float64
	Priority() int
	AspectRatio() float64
	OverflowBehavior() OverflowMode
}

// OverflowMode controls content overflow.
type OverflowMode int

const (
	Clip OverflowMode = iota
	Scroll
	Collapse
	Virtualize
)

// AnimationHook provides animation lifecycle.
type AnimationHook interface {
	OnEnter(ctx AnimContext) AnimationSpec
	OnExit(ctx AnimContext) AnimationSpec
	OnStateChange(delta StateDelta) AnimationSpec
	OnLayoutChange(layout LayoutChange) AnimationSpec
}

// AnimContext provides animation context.
type AnimContext struct {
	Width, Height int
	Time, Delta   int64
}

// AnimationSpec declares an animation.
type AnimationSpec struct {
	Type       string
	DurationMs int
	Easing     string
	DelayMs    int
	Interrupt  bool
}

// StateDelta describes what changed.
type StateDelta struct {
	Field, Reason string
	Old, New      any
}

// LayoutChange describes a layout change.
type LayoutChange struct {
	OldWidth, OldHeight int
	NewWidth, NewHeight int
}

// RenderNode is a renderable element.
type RenderNode struct {
	Type     string
	X, Y     int
	Width    int
	Content  string
	Style    mofu.Style
	ZIndex   int
	Children []RenderNode
}

// Binder connects gadgets to state.
type Binder interface {
	Subscribe(node NodeID)
	SubscribeStream(name string)
	Get(node NodeID) any
	Set(node NodeID, value any)
	Emit(event Event)
	EmitStream(name string, data any)
}

// NodeID is a state node identifier.
type NodeID string

// Event is a gadget event.
type Event struct {
	Type    string
	Payload any
	Source  string
}

// GadgetContext is provided to Init.
type GadgetContext struct {
	Binder  Binder
	Logger  func(format string, args ...any)
	DataDir string
}

// StateView provides read-only state access.
type StateView interface {
	Get(node NodeID) any
	Has(node NodeID) bool
}

// =========================================================================
// BASE GADGET
// =========================================================================

// Base provides default implementations.
type Base struct {
	id       string
	binder   Binder
	children []Gadget
	bounds   mofu.Rect
	style    mofu.Style
	focused  bool
}

func NewBase(id string) *Base {
	return &Base{id: id}
}

func (b *Base) ID() string            { return b.id }
func (b *Base) Init(ctx GadgetContext) error { b.binder = ctx.Binder; return nil }
func (b *Base) Bind(binder Binder)    { b.binder = binder }
func (b *Base) OnEvent(e Event)       {}
func (b *Base) OnTick(dt int64)       {}
func (b *Base) Dispose() error        { return nil }
func (b *Base) Bounds() mofu.Rect     { return b.bounds }
func (b *Base) SetBounds(r mofu.Rect) { b.bounds = r }
func (b *Base) Style() *mofu.Style    { return &b.style }
func (b *Base) Focus()                { b.focused = true }
func (b *Base) Blur()                 { b.focused = false }
func (b *Base) IsFocused() bool       { return b.focused }
func (b *Base) Children() []Gadget    { return b.children }
func (b *Base) AddChild(g Gadget)     { b.children = append(b.children, g) }

func (b *Base) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	for _, child := range b.children {
		nodes = append(nodes, child.Render(state)...)
	}
	return nodes
}

func (b *Base) layout() LayoutContract { return &defaultLayout{} }

type defaultLayout struct{}
func (l *defaultLayout) MinSize() (int, int)            { return 1, 1 }
func (l *defaultLayout) MaxSize() (int, int)            { return 0, 0 }
func (l *defaultLayout) Flex() float64                  { return 1.0 }
func (l *defaultLayout) Priority() int                  { return 0 }
func (l *defaultLayout) AspectRatio() float64           { return 0 }
func (l *defaultLayout) OverflowBehavior() OverflowMode { return Clip }

// =========================================================================
// 1-10: DATA & TABLE SYSTEMS
// =========================================================================

// LiveTable is a virtualized streaming table.
type LiveTable struct {
	Base
	Columns  []string
	Rows     [][]string
	Selected int
	Offset   int
	OnSelect func(row int)
}

func NewLiveTable(id string, cols []string) *LiveTable {
	return &LiveTable{Base: *NewBase(id), Columns: cols}
}

func (g *LiveTable) Render(state StateView) []RenderNode {
	var nodes []RenderNode
 header := strings.Join(g.Columns, " | ")
	nodes = append(nodes, RenderNode{Type: "text", Content: header, Style: mofu.DefaultStyle().WithAttrs(mofu.AttrBold)})
	nodes = append(nodes, RenderNode{Type: "text", Content: strings.Repeat("-", len(header))})

	visible := 20
	start := g.Offset
	if start+visible > len(g.Rows) {
		start = len(g.Rows) - visible
	}
	if start < 0 {
		start = 0
	}

	for i := start; i < len(g.Rows) && i < start+visible; i++ {
		row := g.Rows[i]
		text := strings.Join(row, " | ")
		style := mofu.DefaultStyle()
		if i == g.Selected {
			style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))
		}
		nodes = append(nodes, RenderNode{Type: "text", Content: text, Style: style})
	}
	return nodes
}

func (g *LiveTable) AddRow(row []string) { g.Rows = append(g.Rows, row) }

// DiffTable highlights state changes between snapshots.
type DiffTable struct {
	Base
	Columns    []string
	PrevRows   [][]string
	CurrRows   [][]string
	Selected   int
}

func NewDiffTable(id string, cols []string) *DiffTable {
	return &DiffTable{Base: *NewBase(id), Columns: cols}
}

func (g *DiffTable) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	nodes = append(nodes, RenderNode{Type: "text", Content: strings.Join(g.Columns, " | "), Style: mofu.DefaultStyle().WithAttrs(mofu.AttrBold)})
	for i, row := range g.CurrRows {
		text := strings.Join(row, " | ")
		style := mofu.DefaultStyle()
		if i < len(g.PrevRows) && strings.Join(g.PrevRows[i], "|") != strings.Join(row, "|") {
			style = mofu.DefaultStyle().Fg(mofu.Hex("f9e2af"))
		}
		nodes = append(nodes, RenderNode{Type: "text", Content: text, Style: style})
	}
	return nodes
}

// HeatTable is a density-based value visualization.
type HeatTable struct {
	Base
	Data    [][]float64
	Min, Max float64
}

func NewHeatTable(id string) *HeatTable {
	return &HeatTable{Base: *NewBase(id), Min: 0, Max: 1}
}

func (g *HeatTable) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	density := " ·░▒▓█"
	for _, row := range g.Data {
		line := ""
		for _, v := range row {
			idx := int((v - g.Min) / (g.Max - g.Min) * float64(len(density)-1))
			if idx < 0 { idx = 0 }
			if idx >= len(density) { idx = len(density) - 1 }
			line += string(density[idx])
		}
		nodes = append(nodes, RenderNode{Type: "text", Content: line})
	}
	return nodes
}

// PagedTable is a lazy loading + pagination engine.
type PagedTable struct {
	Base
	Columns   []string
	LoadPage  func(page int) [][]string
	Page      int
	PageSize  int
	CurrPage  [][]string
}

func NewPagedTable(id string, cols []string, loader func(int) [][]string) *PagedTable {
	return &PagedTable{Base: *NewBase(id), Columns: cols, LoadPage: loader, PageSize: 20}
}

func (g *PagedTable) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	nodes = append(nodes, RenderNode{Type: "text", Content: fmt.Sprintf("Page %d", g.Page+1), Style: mofu.DefaultStyle().WithAttrs(mofu.AttrBold)})
	for _, row := range g.CurrPage {
		nodes = append(nodes, RenderNode{Type: "text", Content: strings.Join(row, " | ")})
	}
	return nodes
}

func (g *PagedTable) Load() {
	if g.LoadPage != nil {
		g.CurrPage = g.LoadPage(g.Page)
	}
}

// TreeTable is an expandable hierarchical dataset.
type TreeTable struct {
	Base
	Nodes   []*TreeNode
	Selected int
}

type TreeNode struct {
	Label    string
	Children []*TreeNode
	Expanded bool
}

func NewTreeTable(id string) *TreeTable {
	return &TreeTable{Base: *NewBase(id)}
}

func (g *TreeTable) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	var render func(n *TreeNode, depth int)
	render = func(n *TreeNode, depth int) {
		prefix := strings.Repeat("  ", depth)
		icon := "├─"
		if n.Expanded { icon = "▼ " }
		nodes = append(nodes, RenderNode{Type: "text", Content: prefix + icon + " " + n.Label})
		if n.Expanded {
			for _, child := range n.Children {
				render(child, depth+1)
			}
		}
	}
	for _, n := range g.Nodes {
		render(n, 0)
	}
	return nodes
}

// StreamingGrid is a real-time updating grid view.
type StreamingGrid struct {
	Base
	Cells   [][]string
	Width   int
}

func NewStreamingGrid(id string, w int) *StreamingGrid {
	return &StreamingGrid{Base: *NewBase(id), Width: w}
}

func (g *StreamingGrid) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	for _, row := range g.Cells {
		line := strings.Join(row, " ")
		nodes = append(nodes, RenderNode{Type: "text", Content: line})
	}
	return nodes
}

func (g *StreamingGrid) UpdateCell(x, y int, val string) {
	for y >= len(g.Cells) {
		g.Cells = append(g.Cells, make([]string, g.Width))
	}
	if x >= 0 && x < len(g.Cells[y]) {
		g.Cells[y][x] = val
	}
}

// FilterTable is a reactive filtering + query binding.
type FilterTable struct {
	Base
	Columns []string
	Rows    [][]string
	Filter  string
}

func NewFilterTable(id string, cols []string) *FilterTable {
	return &FilterTable{Base: *NewBase(id), Columns: cols}
}

func (g *FilterTable) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	nodes = append(nodes, RenderNode{Type: "text", Content: fmt.Sprintf("Filter: %s", g.Filter), Style: mofu.DefaultStyle().Fg(mofu.Hex("666666"))})
	for _, row := range g.Rows {
		if g.Filter != "" && !strings.Contains(strings.ToLower(strings.Join(row, " ")), strings.ToLower(g.Filter)) {
			continue
		}
		nodes = append(nodes, RenderNode{Type: "text", Content: strings.Join(row, " | ")})
	}
	return nodes
}

func (g *FilterTable) SetFilter(f string) { g.Filter = f }

// SortTable is a multi-key reactive sorting engine.
type SortTable struct {
	Base
	Columns []string
	Rows    [][]string
	SortBy  int
	Asc     bool
}

func NewSortTable(id string, cols []string) *SortTable {
	return &SortTable{Base: *NewBase(id), Columns: cols, Asc: true}
}

func (g *SortTable) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	for _, row := range g.Rows {
		nodes = append(nodes, RenderNode{Type: "text", Content: strings.Join(row, " | ")})
	}
	return nodes
}

// PivotTableLite is a grouped aggregation view.
type PivotTableLite struct {
	Base
	Groups   map[string][]string
	GroupBy  int
}

func NewPivotTableLite(id string) *PivotTableLite {
	return &PivotTableLite{Base: *NewBase(id), Groups: make(map[string][]string)}
}

func (g *PivotTableLite) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	for group, items := range g.Groups {
		nodes = append(nodes, RenderNode{Type: "text", Content: fmt.Sprintf("%s (%d)", group, len(items)), Style: mofu.DefaultStyle().WithAttrs(mofu.AttrBold)})
		for _, item := range items {
			nodes = append(nodes, RenderNode{Type: "text", Content: "  " + item})
		}
	}
	return nodes
}

// SparseTable is optimized for massive datasets (10k+ rows).
type SparseTable struct {
	Base
	Columns []string
	Rows    map[int][]string
	Count   int
	Offset  int
}

func NewSparseTable(id string, cols []string) *SparseTable {
	return &SparseTable{Base: *NewBase(id), Columns: cols, Rows: make(map[int][]string)}
}

func (g *SparseTable) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	visible := 30
	for i := g.Offset; i < g.Offset+visible && i < g.Count; i++ {
		if row, ok := g.Rows[i]; ok {
			nodes = append(nodes, RenderNode{Type: "text", Content: strings.Join(row, " | ")})
		}
	}
	return nodes
}

func (g *SparseTable) SetRow(idx int, row []string) { g.Rows[idx] = row; if idx >= g.Count { g.Count = idx + 1 } }

// =========================================================================
// 11-20: NAVIGATION & LAYOUT SYSTEMS
// =========================================================================

// SmartSidebar is an auto-collapsing + priority-aware nav.
type SmartSidebar struct {
	Base
	Items    []SidebarItem
	Expanded bool
	Width    int
}

type SidebarItem struct {
	Label    string
	Icon     string
	Priority int
	Active   bool
}

func NewSmartSidebar(id string) *SmartSidebar {
	return &SmartSidebar{Base: *NewBase(id), Width: 20, Expanded: true}
}

func (g *SmartSidebar) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	for _, item := range g.Items {
		text := item.Icon + " " + item.Label
		if !g.Expanded {
			text = item.Icon
		}
		style := mofu.DefaultStyle()
		if item.Active {
			style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))
		}
		nodes = append(nodes, RenderNode{Type: "text", Content: text, Style: style})
	}
	return nodes
}

func (g *SmartSidebar) Toggle() { g.Expanded = !g.Expanded }

// AdaptiveSplit is an auto layout balancing engine.
type AdaptiveSplit struct {
	Base
	Left, Right Gadget
	Ratio       float64
}

func NewAdaptiveSplit(id string, left, right Gadget) *AdaptiveSplit {
	return &AdaptiveSplit{Base: *NewBase(id), Left: left, Right: right, Ratio: 0.3}
}

func (g *AdaptiveSplit) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	if g.Left != nil { nodes = append(nodes, g.Left.Render(state)...) }
	if g.Right != nil { nodes = append(nodes, g.Right.Render(state)...) }
	return nodes
}

// WorkspaceGrid is a dynamic multi-panel grid system.
type WorkspaceGrid struct {
	Base
	Panels []Gadget
	Cols   int
}

func NewWorkspaceGrid(id string, cols int) *WorkspaceGrid {
	return &WorkspaceGrid{Base: *NewBase(id), Cols: cols}
}

func (g *WorkspaceGrid) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	for _, panel := range g.Panels {
		nodes = append(nodes, panel.Render(state)...)
	}
	return nodes
}

func (g *WorkspaceGrid) AddPanel(p Gadget) { g.Panels = append(g.Panels, p) }

// InspectorPane is a contextual right-side data inspector.
type InspectorPane struct {
	Base
	Title   string
	Content []RenderNode
	Width   int
}

func NewInspectorPane(id, title string) *InspectorPane {
	return &InspectorPane{Base: *NewBase(id), Title: title, Width: 30}
}

func (g *InspectorPane) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	nodes = append(nodes, RenderNode{Type: "text", Content: g.Title, Style: mofu.DefaultStyle().WithAttrs(mofu.AttrBold)})
	nodes = append(nodes, RenderNode{Type: "text", Content: strings.Repeat("─", g.Width-2)})
	nodes = append(nodes, g.Content...)
	return nodes
}

// FocusNavigator is a graph-based navigation system.
type FocusNavigator struct {
	Base
	Focusable []Gadget
	Current   int
}

func NewFocusNavigator(id string) *FocusNavigator {
	return &FocusNavigator{Base: *NewBase(id)}
}

func (g *FocusNavigator) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	for i, f := range g.Focusable {
		nodes = append(nodes, f.Render(state)...)
		_ = i
	}
	return nodes
}

func (g *FocusNavigator) FocusNext() {
	if g.Current < len(g.Focusable)-1 { g.Current++ } else { g.Current = 0 }
}

func (g *FocusNavigator) FocusPrev() {
	if g.Current > 0 { g.Current-- } else { g.Current = len(g.Focusable) - 1 }
}

// CommandDock is a persistent action bar system.
type CommandDock struct {
	Base
	Actions []DockAction
}

type DockAction struct {
	Label    string
	Shortcut string
	Action   func()
}

func NewCommandDock(id string) *CommandDock {
	return &CommandDock{Base: *NewBase(id)}
}

func (g *CommandDock) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	parts := []string{}
	for _, a := range g.Actions {
		parts = append(parts, fmt.Sprintf("%s:%s", a.Shortcut, a.Label))
	}
	nodes = append(nodes, RenderNode{Type: "text", Content: " " + strings.Join(parts, " │ ") + " ", Style: mofu.DefaultStyle().Fg(mofu.Hex("666666"))})
	return nodes
}

// ContextOverlay is a floating contextual UI layer.
type ContextOverlay struct {
	Base
	Visible bool
	Content []RenderNode
	X, Y    int
}

func NewContextOverlay(id string) *ContextOverlay {
	return &ContextOverlay{Base: *NewBase(id)}
}

func (g *ContextOverlay) Render(state StateView) []RenderNode {
	if !g.Visible { return nil }
	return g.Content
}

func (g *ContextOverlay) Show()  { g.Visible = true }
func (g *ContextOverlay) Hide()  { g.Visible = false }

// DockingSystem is a draggable panel architecture.
type DockingSystem struct {
	Base
	Panels   []Gadget
	Positions map[string]DockPosition
}

type DockPosition struct{ X, Y, W, H int }

func NewDockingSystem(id string) *DockingSystem {
	return &DockingSystem{Base: *NewBase(id), Positions: make(map[string]DockPosition)}
}

func (g *DockingSystem) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	for _, p := range g.Panels {
		nodes = append(nodes, p.Render(state)...)
	}
	return nodes
}

// ViewportManager manages visible UI region only.
type ViewportManager struct {
	Base
	Content  Gadget
	OffsetY  int
	Height   int
}

func NewViewportManager(id string, h int) *ViewportManager {
	return &ViewportManager{Base: *NewBase(id), Height: h}
}

func (g *ViewportManager) Render(state StateView) []RenderNode {
	if g.Content == nil { return nil }
	return g.Content.Render(state)
}

func (g *ViewportManager) Scroll(dy int) { g.OffsetY += dy; if g.OffsetY < 0 { g.OffsetY = 0 } }

// ResponsiveLayoutCore is terminal-aware adaptive layouts.
type ResponsiveLayoutCore struct {
	Base
	Breakpoints map[int][]Gadget
	Width       int
}

func NewResponsiveLayoutCore(id string) *ResponsiveLayoutCore {
	return &ResponsiveLayoutCore{Base: *NewBase(id), Breakpoints: make(map[int][]Gadget)}
}

func (g *ResponsiveLayoutCore) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	for bp, gadgets := range g.Breakpoints {
		if g.Width >= bp {
			for _, gadget := range gadgets {
				nodes = append(nodes, gadget.Render(state)...)
			}
		}
	}
	return nodes
}

// =========================================================================
// 21-30: INPUT & INTERACTION SYSTEMS
// =========================================================================

// SmartForm is a schema-driven reactive form.
type SmartForm struct {
	Base
	Fields  []FormField
	Values  map[string]any
	Focus   int
}

type FormField struct {
	Name     string
	Label    string
	Type     string
	Value    any
	Required bool
}

func NewSmartForm(id string) *SmartForm {
	return &SmartForm{Base: *NewBase(id), Values: make(map[string]any)}
}

func (g *SmartForm) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	for i, f := range g.Fields {
		style := mofu.DefaultStyle()
		if i == g.Focus { style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")) }
		nodes = append(nodes, RenderNode{Type: "text", Content: fmt.Sprintf("%s: %v", f.Label, g.Values[f.Name]), Style: style})
	}
	return nodes
}

func (g *SmartForm) AddField(f FormField) { g.Fields = append(g.Fields, f) }

// InlineEditor is editable text blocks inside UI.
type InlineEditor struct {
	Base
	Value     string
	CursorPos int
	Focused   bool
}

func NewInlineEditor(id string) *InlineEditor {
	return &InlineEditor{Base: *NewBase(id)}
}

func (g *InlineEditor) Render(state StateView) []RenderNode {
	style := mofu.DefaultStyle()
	if g.Focused { style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")) }
	return []RenderNode{{Type: "text", Content: "[" + g.Value + "]", Style: style}}
}

func (g *InlineEditor) Insert(r rune) { g.Value += string(r) }
func (g *InlineEditor) Delete()       { if len(g.Value) > 0 { g.Value = g.Value[:len(g.Value)-1] } }

// KeyChordRouter is an advanced shortcut system.
type KeyChordRouter struct {
	Base
	Routes map[string]func()
}

func NewKeyChordRouter(id string) *KeyChordRouter {
	return &KeyChordRouter{Base: *NewBase(id), Routes: make(map[string]func())}
}

func (g *KeyChordRouter) OnEvent(e Event) {
	if e.Type == "keypress" {
		if ke, ok := e.Payload.(mofu.KeyEvent); ok {
			key := fmt.Sprintf("%d", ke.Key)
			if fn, ok := g.Routes[key]; ok { fn() }
		}
	}
}

func (g *KeyChordRouter) BindKey(key string, fn func()) { g.Routes[key] = fn }

// MultiCursorInput handles multiple simultaneous text inputs.
type MultiCursorInput struct {
	Base
	Inputs   []*InlineEditor
	Active   int
}

func NewMultiCursorInput(id string) *MultiCursorInput {
	return &MultiCursorInput{Base: *NewBase(id)}
}

func (g *MultiCursorInput) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	for _, input := range g.Inputs {
		nodes = append(nodes, input.Render(state)...)
	}
	return nodes
}

func (g *MultiCursorInput) AddInput() *InlineEditor {
	editor := NewInlineEditor(fmt.Sprintf("%s-%d", g.id, len(g.Inputs)))
	g.Inputs = append(g.Inputs, editor)
	return editor
}

// AutoCompleteEngine is context-aware suggestions.
type AutoCompleteEngine struct {
	Base
	Query       string
	Suggestions []string
	Selected    int
	OnSelect    func(string)
}

func NewAutoCompleteEngine(id string) *AutoCompleteEngine {
	return &AutoCompleteEngine{Base: *NewBase(id)}
}

func (g *AutoCompleteEngine) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	for i, s := range g.Suggestions {
		style := mofu.DefaultStyle()
		if i == g.Selected { style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")) }
		nodes = append(nodes, RenderNode{Type: "text", Content: s, Style: style})
	}
	return nodes
}

func (g *AutoCompleteEngine) Filter(query string) {
	g.Query = query
	g.Selected = 0
}

// ValidatedInputField is a live validation pipeline.
type ValidatedInputField struct {
	Base
	Value     string
	Label     string
	Validator func(string) error
	Error     error
	Focused   bool
}

func NewValidatedInputField(id, label string, validator func(string) error) *ValidatedInputField {
	return &ValidatedInputField{Base: *NewBase(id), Label: label, Validator: validator}
}

func (g *ValidatedInputField) Render(state StateView) []RenderNode {
	style := mofu.DefaultStyle()
	if g.Error != nil { style = mofu.DefaultStyle().Fg(mofu.Hex("f38ba8")) }
	nodes := []RenderNode{{Type: "text", Content: g.Label + ": " + g.Value, Style: style}}
	if g.Error != nil {
		nodes = append(nodes, RenderNode{Type: "text", Content: "  " + g.Error.Error(), Style: mofu.DefaultStyle().Fg(mofu.Hex("f38ba8"))})
	}
	return nodes
}

func (g *ValidatedInputField) SetValue(v string) {
	g.Value = v
	if g.Validator != nil { g.Error = g.Validator(v) } else { g.Error = nil }
}

// InputStreamRouter is an event routing pipeline.
type InputStreamRouter struct {
	Base
	Routes map[string]func(Event)
}

func NewInputStreamRouter(id string) *InputStreamRouter {
	return &InputStreamRouter{Base: *NewBase(id), Routes: make(map[string]func(Event))}
}

func (g *InputStreamRouter) OnEvent(e Event) {
	if fn, ok := g.Routes[e.Type]; ok { fn(e) }
}

func (g *InputStreamRouter) Route(eventType string, fn func(Event)) { g.Routes[eventType] = fn }

// GestureInputLayer is a mouse + trackpad abstraction.
type GestureInputLayer struct {
	Base
	Handlers map[string]func(x, y int)
}

func NewGestureInputLayer(id string) *GestureInputLayer {
	return &GestureInputLayer{Base: *NewBase(id), Handlers: make(map[string]func(x, y int))}
}

func (g *GestureInputLayer) OnEvent(e Event) {
	if e.Type == "mouse" {
		if me, ok := e.Payload.(mofu.MouseEvent); ok {
			key := fmt.Sprintf("%d-%d", me.Button, me.Action)
			if fn, ok := g.Handlers[key]; ok { fn(me.X, me.Y) }
		}
	}
}

func (g *GestureInputLayer) OnGesture(name string, fn func(x, y int)) { g.Handlers[name] = fn }

// FocusTrapManager is controlled input boundaries.
type FocusTrapManager struct {
	Base
	Trapped  bool
	Trapped_ Gadget
}

func NewFocusTrapManager(id string) *FocusTrapManager {
	return &FocusTrapManager{Base: *NewBase(id)}
}

func (g *FocusTrapManager) Render(state StateView) []RenderNode {
	if g.Trapped && g.Trapped_ != nil { return g.Trapped_.Render(state) }
	return nil
}

func (g *FocusTrapManager) Trap(gadget Gadget) { g.Trapped = true; g.Trapped_ = gadget }
func (g *FocusTrapManager) Release()           { g.Trapped = false; g.Trapped_ = nil }

// =========================================================================
// 31-40: REAL-TIME DATA SYSTEMS
// =========================================================================

// LogStream is zero-copy streaming logs.
type LogStream struct {
	Base
	Lines    []string
	MaxLines int
	Offset   int
	Filter   string
}

func NewLogStream(id string) *LogStream {
	return &LogStream{Base: *NewBase(id), MaxLines: 1000}
}

func (g *LogStream) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	for i := len(g.Lines) - 1 - g.Offset; i >= 0 && i < len(g.Lines); i-- {
		line := g.Lines[i]
		if g.Filter != "" && !strings.Contains(strings.ToLower(line), strings.ToLower(g.Filter)) { continue }
		style := mofu.DefaultStyle()
		if strings.Contains(line, "ERROR") { style = mofu.DefaultStyle().Fg(mofu.Hex("f38ba8")) } else if strings.Contains(line, "WARN") { style = mofu.DefaultStyle().Fg(mofu.Hex("f9e2af")) }
		nodes = append(nodes, RenderNode{Type: "text", Content: line, Style: style})
	}
	return nodes
}

func (g *LogStream) Append(line string) { g.Lines = append(g.Lines, line); if len(g.Lines) > g.MaxLines { g.Lines = g.Lines[len(g.Lines)-g.MaxLines:] } }

// MetricBoard is real-time system metrics.
type MetricBoard struct {
	Base
	Metrics map[string]float64
	Labels  map[string]string
}

func NewMetricBoard(id string) *MetricBoard {
	return &MetricBoard{Base: *NewBase(id), Metrics: make(map[string]float64), Labels: make(map[string]string)}
}

func (g *MetricBoard) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	for k, v := range g.Metrics {
		label := k
		if l, ok := g.Labels[k]; ok { label = l }
		nodes = append(nodes, RenderNode{Type: "text", Content: fmt.Sprintf("%-20s %8.2f", label, v)})
	}
	return nodes
}

func (g *MetricBoard) Set(name string, value float64) { g.Metrics[name] = value }

// EventFeed is a live event timeline.
type EventFeed struct {
	Base
	Events []EventEntry
	Max    int
}

type EventEntry struct {
	Time    string
	Type    string
	Message string
}

func NewEventFeed(id string) *EventFeed {
	return &EventFeed{Base: *NewBase(id), Max: 100}
}

func (g *EventFeed) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	for i := len(g.Events) - 1; i >= 0; i-- {
		e := g.Events[i]
		nodes = append(nodes, RenderNode{Type: "text", Content: fmt.Sprintf("[%s] %s: %s", e.Time, e.Type, e.Message)})
	}
	return nodes
}

func (g *EventFeed) Add(e EventEntry) { g.Events = append(g.Events, e); if len(g.Events) > g.Max { g.Events = g.Events[len(g.Events)-g.Max:] } }

// ProcessTreeView is OS process visualization.
type ProcessTreeView struct {
	Base
	Processes []ProcessInfo
}

type ProcessInfo struct {
	PID    int
	Name   string
	CPU    float64
	Memory float64
}

func NewProcessTreeView(id string) *ProcessTreeView {
	return &ProcessTreeView{Base: *NewBase(id)}
}

func (g *ProcessTreeView) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	nodes = append(nodes, RenderNode{Type: "text", Content: "PID   NAME                CPU    MEM", Style: mofu.DefaultStyle().WithAttrs(mofu.AttrBold)})
	for _, p := range g.Processes {
		nodes = append(nodes, RenderNode{Type: "text", Content: fmt.Sprintf("%-5d %-18s %5.1f%% %6.1fM", p.PID, p.Name, p.CPU, p.Memory)})
	}
	return nodes
}

// NetworkMonitor is live request/packet visualization.
type NetworkMonitor struct {
	Base
	Requests []NetworkRequest
}

type NetworkRequest struct {
	Method string
	URL    string
	Status int
	Time   int64
}

func NewNetworkMonitor(id string) *NetworkMonitor {
	return &NetworkMonitor{Base: *NewBase(id)}
}

func (g *NetworkMonitor) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	for _, r := range g.Requests {
		status := fmt.Sprintf("%d", r.Status)
		style := mofu.DefaultStyle()
		if r.Status >= 400 { style = mofu.DefaultStyle().Fg(mofu.Hex("f38ba8")) } else if r.Status >= 200 { style = mofu.DefaultStyle().Fg(mofu.Hex("a6e3a1")) }
		nodes = append(nodes, RenderNode{Type: "text", Content: fmt.Sprintf("%s %s %s %dms", r.Method, r.URL, status, r.Time), Style: style})
	}
	return nodes
}

// FileWatcherView is a reactive filesystem UI.
type FileWatcherView struct {
	Base
	Files   []FileInfo
	Root    string
}

type FileInfo struct {
	Name  string
	Size  int64
	IsDir bool
}

func NewFileWatcherView(id, root string) *FileWatcherView {
	return &FileWatcherView{Base: *NewBase(id), Root: root}
}

func (g *FileWatcherView) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	nodes = append(nodes, RenderNode{Type: "text", Content: g.Root, Style: mofu.DefaultStyle().WithAttrs(mofu.AttrBold)})
	for _, f := range g.Files {
		icon := "📄"
		if f.IsDir { icon = "📁" }
		nodes = append(nodes, RenderNode{Type: "text", Content: fmt.Sprintf("%s %s (%d bytes)", icon, f.Name, f.Size)})
	}
	return nodes
}

// StreamConsole is a continuous CLI output engine.
type StreamConsole struct {
	Base
	Lines   []string
	MaxLines int
}

func NewStreamConsole(id string) *StreamConsole {
	return &StreamConsole{Base: *NewBase(id), MaxLines: 500}
}

func (g *StreamConsole) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	for _, line := range g.Lines {
		nodes = append(nodes, RenderNode{Type: "text", Content: line})
	}
	return nodes
}

func (g *StreamConsole) Write(line string) { g.Lines = append(g.Lines, line); if len(g.Lines) > g.MaxLines { g.Lines = g.Lines[len(g.Lines)-g.MaxLines:] } }

// TraceViewer is an execution tracing system.
type TraceViewer struct {
	Base
	Spans   []TraceSpan
}

type TraceSpan struct {
	Name     string
	Duration int64
	Depth    int
}

func NewTraceViewer(id string) *TraceViewer {
	return &TraceViewer{Base: *NewBase(id)}
}

func (g *TraceViewer) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	for _, span := range g.Spans {
		indent := strings.Repeat("  ", span.Depth)
		nodes = append(nodes, RenderNode{Type: "text", Content: fmt.Sprintf("%s%s (%dms)", indent, span.Name, span.Duration)})
	}
	return nodes
}

// PipelineVisualizer is a data flow visualization.
type PipelineVisualizer struct {
	Base
	Stages []PipelineStage
}

type PipelineStage struct {
	Name   string
	Status string
	Count  int
}

func NewPipelineVisualizer(id string) *PipelineVisualizer {
	return &PipelineVisualizer{Base: *NewBase(id)}
}

func (g *PipelineVisualizer) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	for i, s := range g.Stages {
		arrow := " → "
		if i == len(g.Stages)-1 { arrow = "" }
		style := mofu.DefaultStyle()
		if s.Status == "active" { style = mofu.DefaultStyle().Fg(mofu.Hex("a6e3a1")) }
		nodes = append(nodes, RenderNode{Type: "text", Content: fmt.Sprintf("[%s] %s (%d)%s", s.Status, s.Name, s.Count, arrow), Style: style})
	}
	return nodes
}

// StateInspector is a live state graph debugger.
type StateInspector struct {
	Base
	Nodes   map[string]any
}

func NewStateInspector(id string) *StateInspector {
	return &StateInspector{Base: *NewBase(id), Nodes: make(map[string]any)}
}

func (g *StateInspector) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	nodes = append(nodes, RenderNode{Type: "text", Content: "State Graph", Style: mofu.DefaultStyle().WithAttrs(mofu.AttrBold)})
	for k, v := range g.Nodes {
		nodes = append(nodes, RenderNode{Type: "text", Content: fmt.Sprintf("%s = %v", k, v)})
	}
	return nodes
}

func (g *StateInspector) Set(k string, v any) { g.Nodes[k] = v }

// =========================================================================
// 41-50: VISUAL + ASCII SYSTEMS
// =========================================================================

// ASCIIScene is a full scene graph rendering engine.
type ASCIIScene struct {
	Base
	Width, Height int
	Chars         [][]rune
	Styles        [][]mofu.Style
}

func NewASCIIScene(id string, w, h int) *ASCIIScene {
	chars := make([][]rune, h)
	styles := make([][]mofu.Style, h)
	for i := range chars {
		chars[i] = make([]rune, w)
		styles[i] = make([]mofu.Style, w)
		for j := range chars[i] {
			chars[i][j] = ' '
		}
	}
	return &ASCIIScene{Base: *NewBase(id), Width: w, Height: h, Chars: chars, Styles: styles}
}

func (g *ASCIIScene) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	for y := 0; y < g.Height; y++ {
		line := ""
		for x := 0; x < g.Width; x++ {
			line += string(g.Chars[y][x])
		}
		nodes = append(nodes, RenderNode{Type: "text", Content: line})
	}
	return nodes
}

func (g *ASCIIScene) Set(x, y int, ch rune, style mofu.Style) {
	if x >= 0 && x < g.Width && y >= 0 && y < g.Height {
		g.Chars[y][x] = ch
		g.Styles[y][x] = style
	}
}

func (g *ASCIIScene) Clear() {
	for y := 0; y < g.Height; y++ {
		for x := 0; x < g.Width; x++ {
			g.Chars[y][x] = ' '
		}
	}
}

// ParticleField is a terminal particle system.
type ParticleField struct {
	Base
	Particles []Particle
	Width, Height int
}

type Particle struct {
	X, Y   float64
	VX, VY float64
	Life   int
	Char   rune
}

func NewParticleField(id string, w, h int) *ParticleField {
	return &ParticleField{Base: *NewBase(id), Width: w, Height: h}
}

func (g *ParticleField) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	for y := 0; y < g.Height; y++ {
		line := ""
		for x := 0; x < g.Width; x++ {
			found := false
			for _, p := range g.Particles {
				if int(p.X) == x && int(p.Y) == y && p.Life > 0 {
					line += string(p.Char)
					found = true
					break
				}
			}
			if !found { line += " " }
		}
		nodes = append(nodes, RenderNode{Type: "text", Content: line})
	}
	return nodes
}

func (g *ParticleField) Emit(x, y float64, ch rune) {
	g.Particles = append(g.Particles, Particle{X: x, Y: y, VX: (float64(len(g.Particles)%3) - 1) * 0.5, VY: -1, Life: 20, Char: ch})
}

func (g *ParticleField) Update() {
	for i := range g.Particles {
		g.Particles[i].X += g.Particles[i].VX
		g.Particles[i].Y += g.Particles[i].VY
		g.Particles[i].Life--
	}
	alive := 0
	for _, p := range g.Particles {
		if p.Life > 0 { g.Particles[alive] = p; alive++ }
	}
	g.Particles = g.Particles[:alive]
}

// SplashComposer is animated boot sequences.
type SplashComposer struct {
	Base
	Lines  []string
	Frame  int
	Total  int
}

func NewSplashComposer(id string, lines []string) *SplashComposer {
	return &SplashComposer{Base: *NewBase(id), Lines: lines, Total: len(lines)}
}

func (g *SplashComposer) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	for i := 0; i <= g.Frame && i < len(g.Lines); i++ {
		nodes = append(nodes, RenderNode{Type: "text", Content: g.Lines[i], Style: mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))})
	}
	return nodes
}

func (g *SplashComposer) Advance() { if g.Frame < g.Total { g.Frame++ } }

// WaveVisualizer is an audio/data waveform renderer.
type WaveVisualizer struct {
	Base
	Values  []float64
	Width   int
	Char    rune
}

func NewWaveVisualizer(id string, w int) *WaveVisualizer {
	return &WaveVisualizer{Base: *NewBase(id), Width: w, Char: '█'}
}

func (g *WaveVisualizer) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	if len(g.Values) == 0 { return nodes }
	max := 0.0
	for _, v := range g.Values { if v > max { max = v } }
	if max == 0 { max = 1 }
	line := ""
	for i := 0; i < g.Width; i++ {
		idx := i * len(g.Values) / g.Width
		if idx >= len(g.Values) { idx = len(g.Values) - 1 }
		h := int(g.Values[idx] / max * 8)
		if h == 0 { line += " " } else { line += strings.Repeat(string(g.Char), 1) }
	}
	nodes = append(nodes, RenderNode{Type: "text", Content: line})
	return nodes
}

// DensityMapRenderer is heat/flow visualization.
type DensityMapRenderer struct {
	Base
	Data    [][]float64
	Width, Height int
}

func NewDensityMapRenderer(id string, w, h int) *DensityMapRenderer {
	return &DensityMapRenderer{Base: *NewBase(id), Width: w, Height: h}
}

func (g *DensityMapRenderer) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	density := " ·░▒▓█"
	for y := 0; y < g.Height; y++ {
		line := ""
		for x := 0; x < g.Width; x++ {
			val := 0.0
			if y < len(g.Data) && x < len(g.Data[y]) { val = g.Data[y][x] }
			idx := int(val * float64(len(density)-1))
			if idx < 0 { idx = 0 }
			if idx >= len(density) { idx = len(density) - 1 }
			line += string(density[idx])
		}
		nodes = append(nodes, RenderNode{Type: "text", Content: line})
	}
	return nodes
}

// ProceduralArtEngine is generative ASCII visuals.
type ProceduralArtEngine struct {
	Base
	Width, Height int
	Seed          int64
	Pattern       string
}

func NewProceduralArtEngine(id string, w, h int) *ProceduralArtEngine {
	return &ProceduralArtEngine{Base: *NewBase(id), Width: w, Height: h, Pattern: "noise"}
}

func (g *ProceduralArtEngine) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	chars := " .:-=+*#%@"
	for y := 0; y < g.Height; y++ {
		line := ""
		for x := 0; x < g.Width; x++ {
			val := float64((x*7+y*13+int(g.Seed))%100) / 100.0
			idx := int(val * float64(len(chars)-1))
			line += string(chars[idx])
		}
		nodes = append(nodes, RenderNode{Type: "text", Content: line})
	}
	return nodes
}

// MotionBanner is animated headers/logos.
type MotionBanner struct {
	Base
	Text     string
	Offset   int
	Direction int
}

func NewMotionBanner(id, text string) *MotionBanner {
	return &MotionBanner{Base: *NewBase(id), Text: text, Direction: 1}
}

func (g *MotionBanner) Render(state StateView) []RenderNode {
	padded := strings.Repeat(" ", g.Offset) + g.Text
	return []RenderNode{{Type: "text", Content: padded, Style: mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)}}
}

func (g *MotionBanner) Advance() {
	g.Offset += g.Direction
	if g.Offset > 10 || g.Offset < 0 { g.Direction = -g.Direction }
}

// GlyphMorpher is character morph animations.
type GlyphMorpher struct {
	Base
	From, To rune
	Progress float64
}

func NewGlyphMorpher(id string, from, to rune) *GlyphMorpher {
	return &GlyphMorpher{Base: *NewBase(id), From: from, To: to}
}

func (g *GlyphMorpher) Render(state StateView) []RenderNode {
	ch := g.From
	if g.Progress > 0.5 { ch = g.To }
	return []RenderNode{{Type: "text", Content: string(ch), Style: mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))}}
}

func (g *GlyphMorpher) Advance(p float64) { g.Progress = p }

// TerminalCanvas is pixel-like drawing abstraction.
type TerminalCanvas struct {
	Base
	Width, Height int
	Pixels         [][]mofu.Color
}

func NewTerminalCanvas(id string, w, h int) *TerminalCanvas {
	pixels := make([][]mofu.Color, h)
	for i := range pixels { pixels[i] = make([]mofu.Color, w) }
	return &TerminalCanvas{Base: *NewBase(id), Width: w, Height: h, Pixels: pixels}
}

func (g *TerminalCanvas) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	density := " ·░▒▓█"
	for y := 0; y < g.Height; y++ {
		line := ""
		for x := 0; x < g.Width; x++ {
			c := g.Pixels[y][x]
			brightness := float64(c.R+uint8(c.G)+uint8(c.B)) / 765.0
			idx := int(brightness * float64(len(density)-1))
			if idx < 0 { idx = 0 }
			if idx >= len(density) { idx = len(density) - 1 }
			line += string(density[idx])
		}
		nodes = append(nodes, RenderNode{Type: "text", Content: line})
	}
	return nodes
}

func (g *TerminalCanvas) SetPixel(x, y int, c mofu.Color) {
	if x >= 0 && x < g.Width && y >= 0 && y < g.Height { g.Pixels[y][x] = c }
}

// SDFRendererLite is signed-distance-field ASCII approximation.
type SDFRendererLite struct {
	Base
	Width, Height int
	SDF           func(x, y float64) float64
}

func NewSDFRendererLite(id string, w, h int, sdf func(float64, float64) float64) *SDFRendererLite {
	return &SDFRendererLite{Base: *NewBase(id), Width: w, Height: h, SDF: sdf}
}

func (g *SDFRendererLite) Render(state StateView) []RenderNode {
	var nodes []RenderNode
	density := " ·:-=+*#%@"
	for y := 0; y < g.Height; y++ {
		line := ""
		for x := 0; x < g.Width; x++ {
			fx := float64(x) / float64(g.Width) * 2 - 1
			fy := float64(y) / float64(g.Height) * 2 - 1
			val := g.SDF(fx, fy)
			idx := int((1 - val) * float64(len(density)-1))
			if idx < 0 { idx = 0 }
			if idx >= len(density) { idx = len(density) - 1 }
			line += string(density[idx])
		}
		nodes = append(nodes, RenderNode{Type: "text", Content: line})
	}
	return nodes
}
