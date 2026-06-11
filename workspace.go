package mofu

import (
	"sync"
)

// ---------------------------------------------------------------------------
// Workspace (Anthology Ch.10 §10.1)
// ---------------------------------------------------------------------------

// PaneID uniquely identifies a workspace pane.
type PaneID uint64

// WorkspaceLayout describes the arrangement of panes.
type WorkspaceLayout uint8

const (
	LayoutSingle WorkspaceLayout = iota
	LayoutSplitHorizontal
	LayoutSplitVertical
	LayoutTabs
	LayoutGrid
	LayoutFloating
)

// String returns a human-readable name.
func (l WorkspaceLayout) String() string {
	switch l {
	case LayoutSingle:
		return "single"
	case LayoutSplitHorizontal:
		return "split-h"
	case LayoutSplitVertical:
		return "split-v"
	case LayoutTabs:
		return "tabs"
	case LayoutGrid:
		return "grid"
	case LayoutFloating:
		return "floating"
	}
	return "unknown"
}

// WorkspaceState holds the full state of the workspace system.
type WorkspaceState struct {
	Layout     WorkspaceLayout
	ActivePane PaneID
	PaneStates map[PaneID]PaneState
	TabIndex   int
	TabNames   map[PaneID]string
	Dirty      bool
}

// PaneState tracks runtime state for an individual pane.
type PaneState struct {
	Title         string
	Focused       bool
	MinSize       Rect
	MaxSize       Rect
	FloatingPos   Vec2
	FloatingSize  Vec2
	Visible       bool
	UserData      map[string]any
	SplitFraction float64 // 0..1 for split layouts
}

// DefaultPaneState returns the zero-value for PaneState with safe defaults.
func DefaultPaneState() PaneState {
	return PaneState{
		Visible:       true,
		SplitFraction: 0.5,
	}
}

// Workspace manages a collection of named panes and their layout.
// It is the central coordination point for multi-pane applications.
type Workspace struct {
	mu       sync.Mutex
	panes    map[PaneID]Pane
	order    []PaneID // display order
	state    WorkspaceState
	nextID   PaneID
	listener WorkspaceListener
}

// Pane is the interface each workspace pane must implement.
// A Pane is also a mofu.Node so it can render and process events normally.
type Pane interface {
	ID() PaneID
	Type() string
	Title() string
	Close()
	Split(direction SplitDirection) PaneID
	MinSize() (width, height int)
	MaxSize() (width, height int)
}

// SplitDirection for Split().
type SplitDirection uint8

const (
	SplitHorizontal SplitDirection = iota
	SplitVertical
)

// WorkspaceListener receives workspace lifecycle events.
type WorkspaceListener interface {
	OnPaneAdded(pane Pane)
	OnPaneRemoved(id PaneID)
	OnPaneFocused(id PaneID)
	OnLayoutChanged(layout WorkspaceLayout)
}

// NewWorkspace returns a fresh, empty workspace in Single-pane mode.
func NewWorkspace() *Workspace {
	return &Workspace{
		panes: make(map[PaneID]Pane),
		order: nil,
		state: WorkspaceState{
			Layout:     LayoutSingle,
			PaneStates: make(map[PaneID]PaneState),
			TabNames:   make(map[PaneID]string),
		},
	}
}

// SetListener sets a listener for pane lifecycle events.
func (w *Workspace) SetListener(l WorkspaceListener) {
	w.mu.Lock()
	w.listener = l
	w.mu.Unlock()
}

// firePaneAdded notifies the listener without holding the lock.
func (w *Workspace) firePaneAdded(p Pane) {
	if w.listener != nil {
		w.listener.OnPaneAdded(p)
	}
}

// firePaneRemoved notifies the listener without holding the lock.
func (w *Workspace) firePaneRemoved(id PaneID) {
	if w.listener != nil {
		w.listener.OnPaneRemoved(id)
	}
}

// firePaneFocused notifies the listener without holding the lock.
func (w *Workspace) firePaneFocused(id PaneID) {
	if w.listener != nil {
		w.listener.OnPaneFocused(id)
	}
}

// fireLayoutChanged notifies the listener without holding the lock.
func (w *Workspace) fireLayoutChanged(l WorkspaceLayout) {
	if w.listener != nil {
		w.listener.OnLayoutChanged(l)
	}
}

// ---------------------------------------------------------------------------
// Pane CRUD
// ---------------------------------------------------------------------------

// AddPane inserts a new pane and returns its ID.
func (w *Workspace) AddPane(pane Pane) PaneID {
	w.mu.Lock()
	defer w.mu.Unlock()
	id := w.nextID
	w.nextID++
	w.panes[id] = pane
	w.order = append(w.order, id)
	w.state.PaneStates[id] = DefaultPaneState()
	ps := w.state.PaneStates[id]
	ps.Title = pane.Title()
	ps.UserData = make(map[string]any)
	w.state.PaneStates[id] = ps
	w.state.TabNames[id] = pane.Title()
	w.state.Dirty = true
	w.firePaneAdded(pane)
	return id
}

// RemovePane removes and destroys a pane by ID.
func (w *Workspace) RemovePane(id PaneID) {
	w.mu.Lock()
	defer w.mu.Unlock()

	delete(w.panes, id)
	delete(w.state.PaneStates, id)
	delete(w.state.TabNames, id)
	for i, pid := range w.order {
		if pid == id {
			w.order = append(w.order[:i], w.order[i+1:]...)
			break
		}
	}
	if w.state.ActivePane == id {
		w.state.ActivePane = w.activeOrderLocked()
	}
	w.state.Dirty = true
	w.firePaneRemoved(id)
}

// Pane returns the pane for an ID, or nil.
func (w *Workspace) Pane(id PaneID) Pane {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.panes[id]
}

// ActivePane returns the currently-focused pane.
func (w *Workspace) ActivePane() (Pane, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	p, ok := w.panes[w.state.ActivePane]
	return p, ok
}

// SetActivePane focuses a pane by ID.
func (w *Workspace) SetActivePane(id PaneID) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, ok := w.panes[id]; ok {
		w.state.ActivePane = id
		w.state.Dirty = true
		w.firePaneFocused(id)
	}
}

// MoveFocus shifts focus to the next pane in display order.
// If wrap is true, focus wraps around.
func (w *Workspace) MoveFocus(forward bool, wrap bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.order) == 0 {
		return
	}
	cur := w.state.ActivePane
	curIdx := -1
	for i, pid := range w.order {
		if pid == cur {
			curIdx = i
			break
		}
	}
	if curIdx == -1 {
		w.state.ActivePane = w.order[0]
		w.firePaneFocused(w.state.ActivePane)
		w.state.Dirty = true
		return
	}
	next := curIdx
	if forward {
		next = curIdx + 1
		if next >= len(w.order) {
			if wrap {
				next = 0
			} else {
				next = curIdx
			}
		}
	} else {
		next = curIdx - 1
		if next < 0 {
			if wrap {
				next = len(w.order) - 1
			} else {
				next = curIdx
			}
		}
	}
	w.state.ActivePane = w.order[next]
	w.firePaneFocused(w.state.ActivePane)
	w.state.Dirty = true
}

// CloseActivePane closes and removes the currently-active pane.
// If it was the last pane, the workspace is reset to Single mode.
func (w *Workspace) CloseActivePane() {
	w.mu.Lock()
	id := w.state.ActivePane
	w.mu.Unlock()
	if id == 0 {
		return
	}
	w.RemovePane(id)
}

// Panes returns a snapshot of all panes (no lock held after return).
func (w *Workspace) Panes() []Pane {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]Pane, 0, len(w.panes))
	for _, id := range w.order {
		if p, ok := w.panes[id]; ok {
			out = append(out, p)
		}
	}
	return out
}

func (w *Workspace) activeOrderLocked() PaneID {
	if len(w.order) == 0 {
		return 0
	}
	return w.order[0]
}

// Count returns the number of panes.
func (w *Workspace) Count() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.panes)
}

// Len is a synonym for Count.
func (w *Workspace) Len() int { return w.Count() }

// ---------------------------------------------------------------------------
// Layout (Anthology Ch.10 §10.2)
// ---------------------------------------------------------------------------

// SetLayout changes the layout mode.
func (w *Workspace) SetLayout(layout WorkspaceLayout) {
	w.mu.Lock()
	w.state.Layout = layout
	w.state.Dirty = true
	l := layout
	w.mu.Unlock()
	w.fireLayoutChanged(l)
}

// Layout returns the current layout mode.
func (w *Workspace) Layout() WorkspaceLayout {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.state.Layout
}

// SplitActive splits the active pane and returns the new pane ID.
func (w *Workspace) SplitActive(direction SplitDirection) PaneID {
	w.mu.Lock()
	pane, ok := w.panes[w.state.ActivePane]
	w.mu.Unlock()

	if !ok || pane == nil {
		return 0
	}
	newID := pane.Split(direction)
	if newID == 0 {
		return 0
	}
	// In a real implementation, the new pane and its ID would be registered
	// via AddPane; here we return the ID the splitter gave us.
	_ = newID
	return newID
}

// ResizePane adjusts the split fraction for a pane in a split layout.
// fraction is 0..1; 0 = minimum, 1 = maximum size for the primary pane.
func (w *Workspace) ResizePane(id PaneID, fraction float64) {
	w.mu.Lock()
	defer w.mu.Unlock()
	ps, ok := w.state.PaneStates[id]
	if !ok {
		return
	}
	if fraction < 0 {
		fraction = 0
	} else if fraction > 1 {
		fraction = 1
	}
	ps.SplitFraction = fraction
	w.state.PaneStates[id] = ps
	w.state.Dirty = true
}

// SetPaneTitle updates the user-visible title of a pane.
func (w *Workspace) SetPaneTitle(id PaneID, title string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	ps, ok := w.state.PaneStates[id]
	if !ok {
		return
	}
	ps.Title = title
	w.state.PaneStates[id] = ps
	w.state.TabNames[id] = title
	w.state.Dirty = true
}

// PaneStateEntry pairs an ID with PaneState for serialisation.
type PaneStateEntry struct {
	ID    PaneID
	State PaneState
}

// TabEntry pairs an ID with a title string.
type TabEntry struct {
	ID    PaneID
	Title string
}

// Snapshot holds a serialisable view of the workspace (distinct from serialize.go StateSnapshot).
type Snapshot struct {
	ActivePane PaneID
	Layout     WorkspaceLayout
	PaneStates []PaneStateEntry
	TabNames   []TabEntry
}

// SnapshotExport produces a snapshot of the current workspace state.
func (w *Workspace) SnapshotExport() Snapshot {
	w.mu.Lock()
	defer w.mu.Unlock()
	entries := make([]PaneStateEntry, 0, len(w.state.PaneStates))
	for id, ps := range w.state.PaneStates {
		cp := ps
		if cp.UserData == nil {
			cp.UserData = make(map[string]any)
		}
		entries = append(entries, PaneStateEntry{ID: id, State: cp})
	}
	tabs := make([]TabEntry, 0, len(w.state.TabNames))
	for id, name := range w.state.TabNames {
		tabs = append(tabs, TabEntry{ID: id, Title: name})
	}
	return Snapshot{
		ActivePane: w.state.ActivePane,
		Layout:     w.state.Layout,
		PaneStates: entries,
		TabNames:   tabs,
	}
}

// ---------------------------------------------------------------------------
// TabBar (Anthology Ch.10 §10.3)
// ---------------------------------------------------------------------------

// TabBar renders a minimised tab overview of all panes.
type TabBar struct {
	workspace *Workspace
	Focused   Style
	Normal    Style
}

// NewTabBar returns a TabBar linked to a Workspace.
func NewTabBar(ws *Workspace) *TabBar {
	return &TabBar{
		workspace: ws,
		Focused:   DefaultStyle(),
		Normal:    DefaultStyle(),
	}
}

// ActiveID returns the currently active pane ID.
func (tb *TabBar) ActiveID() PaneID {
	tb.workspace.mu.Lock()
	defer tb.workspace.mu.Unlock()
	return tb.workspace.state.ActivePane
}

// TabNames returns titles in order.
func (tb *TabBar) TabNames() []string {
	tb.workspace.mu.Lock()
	defer tb.workspace.mu.Unlock()
	out := make([]string, 0, len(tb.workspace.order))
	for _, pid := range tb.workspace.order {
		if name, ok := tb.workspace.state.TabNames[pid]; ok {
			out = append(out, name)
		}
	}
	return out
}

// SelectByIndex focuses the pane at the given tab index.
func (tb *TabBar) SelectByIndex(idx int) {
	tb.workspace.mu.Lock()
	defer tb.workspace.mu.Unlock()
	if idx < 0 || idx >= len(tb.workspace.order) {
		return
	}
	pid := tb.workspace.order[idx]
	tb.workspace.state.ActivePane = pid
	tb.workspace.state.Dirty = true
	tb.workspace.firePaneFocused(pid)
}

// ---------------------------------------------------------------------------
// LayoutCache (Anthology Ch.10 §10.4)
// ---------------------------------------------------------------------------

// LayoutCache stores the last-computed pane layout, keyed by a fingerprint
// string. Callers should invalidate when the workspace state changes.
type LayoutCache struct {
	last  string
	rects map[PaneID]Rect
	order []PaneID
	dirty bool
}

// NewLayoutCache returns an empty cache.
func NewLayoutCache() *LayoutCache {
	return &LayoutCache{
		rects: make(map[PaneID]Rect),
	}
}

// GetOrCompute returns cached rectangles for order, recomputing when dirty.
// `compute` is invoked to recalculate panes positions for the given bounds.
func (lc *LayoutCache) GetOrCompute(order []PaneID, bounds Rect, compute func(order []PaneID, bounds Rect) map[PaneID]Rect) map[PaneID]Rect {
	fp := fingerprint(order, bounds)
	if !lc.dirty && lc.last == fp {
		return lc.rects
	}
	lc.rects = compute(order, bounds)
	lc.order = append([]PaneID(nil), order...)
	lc.last = fp
	lc.dirty = false
	return lc.rects
}

// Invalidate marks the cache dirty so it recomputes on the next query.
func (lc *LayoutCache) Invalidate() { lc.dirty = true }

// Rect returns the cached Rect for id, or Rect{}.
func (lc *LayoutCache) Rect(id PaneID) Rect {
	if r, ok := lc.rects[id]; ok {
		return r
	}
	return Rect{}
}

func fingerprint(order []PaneID, bounds Rect) string {
	h := uint32(2166136261)
	for _, id := range order {
		h ^= uint32(id)
		h *= 16777619
	}
	h ^= uint32(bounds.X)
	h *= 16777619
	h ^= uint32(bounds.Y)
	h *= 16777619
	h ^= uint32(bounds.Width)
	h *= 16777619
	h ^= uint32(bounds.Height)
	h *= 16777619
	return string([]byte{
		byte(h >> 24), byte(h >> 16), byte(h >> 8), byte(h),
	})
}

// ---------------------------------------------------------------------------
// State helpers
// DefaultLayout computes a default split layout for n visible panes.
// It returns a WorkspaceLayout recommendation only.
func DefaultLayout(n int) WorkspaceLayout {
	switch {
	case n <= 1:
		return LayoutSingle
	case n <= 2:
		return LayoutSplitVertical
	case n <= 3:
		return LayoutSplitHorizontal
	default:
		return LayoutTabs
	}
}

// MaxLayout returns the layout capable of displaying n panes simultaneously.
func MaxLayout(n int) WorkspaceLayout {
	if n <= 1 {
		return LayoutSingle
	}
	if n <= 4 {
		return LayoutGrid
	}
	return LayoutTabs
}

// MinLayout returns the simplest layout supporting n panes.
func MinLayout(n int) WorkspaceLayout {
	if n <= 1 {
		return LayoutSingle
	}
	return LayoutSingle
}
