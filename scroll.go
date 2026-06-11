package mofu

import (
	"math"
)

// ---------------------------------------------------------------------------
// ScrollState (Anthology Ch.7 §7.1)
// ---------------------------------------------------------------------------
// ScrollDirection enumerates scroll directions.
type ScrollDirection uint8

const (
	ScrollUp ScrollDirection = iota
	ScrollDown
	ScrollLeft
	ScrollRight
)

// ScrollState tracks the current, target, and max scroll offsets and
// the viewport size.  X/Y axes are both supported; callers that only need
// vertical scrolling can ignore ScrollX / MaxX.
type ScrollState struct {
	ScrollY         int
	ScrollX         int
	TargetY         int
	TargetX         int
	MaxY            int
	MaxX            int
	ViewportHeight  int
	ViewportWidth   int
	IsDragging      bool
	DragStartY      int
	DragStartScroll int
	Responsiveness  float64 // 0..1 lerp factor; 0 = never, 1 = instant
}

// NewScrollState returns a zero-offset ScrollState with the given
// viewport dimensions.
func NewScrollState(viewportHeight, viewportWidth int) *ScrollState {
	return &ScrollState{
		ViewportHeight: viewportHeight,
		ViewportWidth:  viewportWidth,
		MaxY:           math.MaxInt,
		MaxX:           math.MaxInt,
		Responsiveness: 0.15,
	}
}

// SetMaxContent clamps MaxY / MaxX.
func (s *ScrollState) SetMaxContent(contentHeight, contentWidth int) {
	s.MaxY = contentHeight - s.ViewportHeight
	if s.MaxY < 0 {
		s.MaxY = 0
	}
	s.MaxX = contentWidth - s.ViewportWidth
	if s.MaxX < 0 {
		s.MaxX = 0
	}
}

// ScrollBy adjusts TargetY / TargetX by dy / dx, clamped to valid range.
func (s *ScrollState) ScrollBy(dy, dx int) {
	s.TargetY = clampInt(s.TargetY+dy, 0, s.MaxY)
	s.TargetX = clampInt(s.TargetX+dx, 0, s.MaxX)
}

// ScrollTo sets TargetY / TargetX to absolute values, clamped.
func (s *ScrollState) ScrollTo(y, x int) {
	s.TargetY = clampInt(y, 0, s.MaxY)
	s.TargetX = clampInt(x, 0, s.MaxX)
}

// PageUp moves up by one viewport height.
func (s *ScrollState) PageUp() {
	s.TargetY = s.ScrollY - s.ViewportHeight
	if s.TargetY < 0 {
		s.TargetY = 0
	}
}

// PageDown moves down by one viewport height.
func (s *ScrollState) PageDown() {
	s.TargetY = s.ScrollY + s.ViewportHeight
	if s.TargetY > s.MaxY {
		s.TargetY = s.MaxY
	}
}

// Home jumps to the top.
func (s *ScrollState) Home() { s.TargetY = 0 }

// End jumps to the bottom.
func (s *ScrollState) End() { s.TargetY = s.MaxY }

// Update lerps current scroll offsets toward targets.
func (s *ScrollState) Update() {
	f := s.Responsiveness
	if f <= 0 {
		f = 0.15
	}
	s.ScrollY = lerpInt(s.ScrollY, s.TargetY, f)
	s.ScrollX = lerpInt(s.ScrollX, s.TargetX, f)
}

// IsAtTop reports whether ScrollY == 0.
func (s *ScrollState) IsAtTop() bool { return s.ScrollY <= 0 }

// IsAtBottom reports whether ScrollY >= MaxY.
func (s *ScrollState) IsAtBottom() bool { return s.ScrollY >= s.MaxY }

// ClampToValid ensures ScrollY / ScrollX are within Valid range.
func (s *ScrollState) ClampToValid() {
	if s.ScrollY < 0 {
		s.ScrollY = 0
	} else if s.ScrollY > s.MaxY {
		s.ScrollY = s.MaxY
	}
	if s.ScrollX < 0 {
		s.ScrollX = 0
	} else if s.ScrollX > s.MaxX {
		s.ScrollX = s.MaxX
	}
}

// OnMouseDown begins a drag operation.
func (s *ScrollState) OnMouseDown(y int) {
	s.IsDragging = true
	s.DragStartY = y
	s.DragStartScroll = s.ScrollY
}

// OnMouseDrag updates ScrollY while dragging.
func (s *ScrollState) OnMouseDrag(currentY int) {
	if !s.IsDragging {
		return
	}
	delta := s.DragStartY - currentY
	s.ScrollY = clampInt(s.DragStartScroll+delta, 0, s.MaxY)
	s.TargetY = s.ScrollY
}

// OnMouseUp ends a drag.
func (s *ScrollState) OnMouseUp() { s.IsDragging = false }

// ---------------------------------------------------------------------------
// ScrollPhysics (Anthology Ch.7 §7.2)
// ---------------------------------------------------------------------------

// ScrollSnap controls post-scroll snapping behaviour.
type ScrollSnap uint8

const (
	ScrollSnapNone    ScrollSnap = iota
	ScrollSnapInteger            // Snap to whole lines
	ScrollSnapHalf               // Snap to half-lines
	ScrollSnapPixel              // Free / sub-pixel
)

// ScrollPhysics adds velocity, friction, and momentum scrolling.
type ScrollPhysics struct {
	State          *ScrollState
	Velocity       Vec2
	Friction       float64
	Responsiveness float64
	Snap           ScrollSnap
}

// NewScrollPhysics returns a physics-enabled ScrollPhysics.
func NewScrollPhysics(state *ScrollState) *ScrollPhysics {
	return &ScrollPhysics{
		State:          state,
		Friction:       0.92,
		Responsiveness: 0.12,
		Snap:           ScrollSnapInteger,
	}
}

// Advance applies momentum scrolling for deltaMs milliseconds.
func (p *ScrollPhysics) Advance(deltaMs uint64) {
	if p.State.IsDragging {
		p.State.Update()
		return
	}
	dt := float64(deltaMs) / 1000
	p.Velocity.Y *= p.Friction
	p.State.TargetY = clampInt(
		int(float64(p.State.TargetY)+p.Velocity.Y*dt),
		0, p.State.MaxY,
	)
	p.State.Update()
}

// ---------------------------------------------------------------------------
// VirtualViewport (Anthology Ch.7 §7.4)
// ---------------------------------------------------------------------------

// VirtualViewport supports content larger than the viewport.
type VirtualViewport struct {
	ContentWidth   int
	ContentHeight  int
	ScrollY        int
	ScrollX        int
	ViewportWidth  int
	ViewportHeight int
}

// NewVirtualViewport returns a VirtualViewport.
func NewVirtualViewport(contentWidth, contentHeight, viewportWidth, viewportHeight int) *VirtualViewport {
	return &VirtualViewport{
		ContentWidth:   contentWidth,
		ContentHeight:  contentHeight,
		ViewportWidth:  viewportWidth,
		ViewportHeight: viewportHeight,
	}
}

// VisibleRegion returns the self-intersecting Rect of the visible content window.
func (v *VirtualViewport) VisibleRegion() Rect {
	return Rect{
		X:      v.ScrollX,
		Y:      v.ScrollY,
		Width:  v.ViewportWidth,
		Height: v.ViewportHeight,
	}
}

// ContentToViewport maps a content-space coordinate to viewport-space, or
// returns false when off-screen.
func (v *VirtualViewport) ContentToViewport(contentX, contentY int) (int, int, bool) {
	vpx := contentX - v.ScrollX
	vpy := contentY - v.ScrollY
	if vpx < 0 || vpy < 0 || vpx >= v.ViewportWidth || vpy >= v.ViewportHeight {
		return 0, 0, false
	}
	return vpx, vpy, true
}

// ViewportToContent maps viewport-space to content-space.
func (v *VirtualViewport) ViewportToContent(vpx, vpy int) int {
	return v.ScrollY + vpy
}

// ClampToValid clamps scroll offsets.
func (v *VirtualViewport) ClampToValid() {
	maxY := v.ContentHeight - v.ViewportHeight
	if maxY < 0 {
		maxY = 0
	}
	maxX := v.ContentWidth - v.ViewportWidth
	if maxX < 0 {
		maxX = 0
	}
	if v.ScrollY < 0 {
		v.ScrollY = 0
	} else if v.ScrollY > maxY {
		v.ScrollY = maxY
	}
	if v.ScrollX < 0 {
		v.ScrollX = 0
	} else if v.ScrollX > maxX {
		v.ScrollX = maxX
	}
}

// IsVisible reports whether (contentX, contentHeight, contentWidth) is
// entirely within the viewport.
func (v *VirtualViewport) IsVisible(contentX, contentY, cw, ch int) bool {
	return contentX+cw > v.ScrollX &&
		contentX < v.ScrollX+v.ViewportWidth &&
		contentY+ch > v.ScrollY &&
		contentY < v.ScrollY+v.ViewportHeight
}

// ---------------------------------------------------------------------------
// ScrollAnchor (Anthology Ch.7 §7.5)
// ---------------------------------------------------------------------------

// AnchorID identifies a scroll anchor.
type AnchorID uint64

// ScrollAnchor marks a widget that should stick to a relative position.
type ScrollAnchor struct {
	ID            AnchorID
	OffsetFromTop float64 // 0.0 = top, 1.0 = bottom
	IsSticky      bool
	ZIndex        int
	WidgetRect    Rect
}

// AnchorSystem manages sticky widgets.
type AnchorSystem struct {
	anchors       []ScrollAnchor
	stickyWidgets []AnchorEntry
}

// AnchorEntry is a resolved sticky widget position.
type AnchorEntry struct {
	ID     AnchorID
	Offset int
	Rect   Rect
}

// NewAnchorSystem returns an empty AnchorSystem.
func NewAnchorSystem() *AnchorSystem {
	return &AnchorSystem{}
}

// AddAnchor registers a new anchor.
func (a *AnchorSystem) AddAnchor(anchor ScrollAnchor) AnchorID {
	anchor.ID = AnchorID(len(a.anchors) + 1)
	a.anchors = append(a.anchors, anchor)
	return anchor.ID
}

// Update recalculates sticky widgets for the current scroll Y.
func (a *AnchorSystem) Update(scrollY, viewportHeight int) {
	a.stickyWidgets = a.stickyWidgets[:0]
	for _, anchor := range a.anchors {
		if !anchor.IsSticky {
			continue
		}
		idealTop := int(float64(scrollY) + anchor.OffsetFromTop*float64(viewportHeight))
		if idealTop < scrollY {
			a.stickyWidgets = append(a.stickyWidgets, AnchorEntry{
				ID:     anchor.ID,
				Offset: scrollY,
				Rect: Rect{
					X:      anchor.WidgetRect.X,
					Y:      scrollY,
					Width:  anchor.WidgetRect.Width,
					Height: anchor.WidgetRect.Height,
				},
			})
		}
	}
}

// StickyWidgets returns resolved sticky widget entries.
func (a *AnchorSystem) StickyWidgets() []AnchorEntry {
	return a.stickyWidgets
}

// ---------------------------------------------------------------------------
// ScrollBar (Anthology-enhanced)
// ---------------------------------------------------------------------------

// ScrollBarStyle describes the visual style of a scroll bar.
type ScrollBarStyle uint8

const (
	ScrollBarASCII ScrollBarStyle = iota
	ScrollBarBlock
	ScrollBarHalfBlock
	ScrollBarBraille
)

// ScrollBar renders scroll indicator for a ScrollState.
type ScrollBar struct {
	State *ScrollState
	Style ScrollBarStyle
	Width int
}

// NewScrollBar returns a ScrollBar.
func NewScrollBar(state *ScrollState) *ScrollBar {
	return &ScrollBar{
		State: state,
		Style: ScrollBarASCII,
		Width: 1,
	}
}

// Visible reports whether the scroll bar should be drawn.
func (sb *ScrollBar) Visible() bool {
	return sb.State.MaxY > sb.State.ViewportHeight
}

// Thumb returns the 0..1 ratio position and the 0..1 thumb size.
func (sb *ScrollBar) Thumb() (ratio, size float64) {
	if sb.State.MaxY <= 0 {
		return 0, 1
	}
	ratio = float64(sb.State.ScrollY) / float64(sb.State.MaxY)
	vh := float64(sb.State.ViewportHeight)
	ch := float64(sb.State.MaxY + sb.State.ViewportHeight)
	size = vh / ch
	if size > 1 {
		size = 1
	}
	return
}

// ---------------------------------------------------------------------------
// ScrollCache (Anthology §7.4 – cached rendered lines)
// ---------------------------------------------------------------------------

// ScrollCache stores rendered line strings for fast re-display.
type ScrollCache struct {
	lines       []string
	width       int
	maxLines    int
	dirty       map[int]bool
	invalidated bool
}

// NewScrollCache returns a ScrollCache holding up to maxLines.
func NewScrollCache(maxLines, width int) *ScrollCache {
	return &ScrollCache{
		lines:    make([]string, 0, maxLines),
		maxLines: maxLines,
		width:    width,
		dirty:    make(map[int]bool),
	}
}

// SetLine replaces a cached line.
func (c *ScrollCache) SetLine(idx int, line string) {
	if idx < 0 {
		return
	}
	for len(c.lines) <= idx {
		c.lines = append(c.lines, "")
	}
	c.lines[idx] = line
	c.dirty[idx] = true
	c.invalidated = true
}

// Line returns a cached line, empty string if absent.
func (c *ScrollCache) Line(idx int) string {
	if idx < 0 || idx >= len(c.lines) {
		return ""
	}
	return c.lines[idx]
}

// Invalidate marks all lines dirty.
func (c *ScrollCache) Invalidate() { c.invalidated = true }

// Clear removes all cached lines.
func (c *ScrollCache) Clear() {
	c.lines = c.lines[:0]
	c.dirty = make(map[int]bool)
	c.invalidated = false
}

// IsDirty reports whether any line has changed since last ClearDirty.
func (c *ScrollCache) IsDirty() bool { return c.invalidated }

// ClearDirty resets the dirty flag.
func (c *ScrollCache) ClearDirty() { c.invalidated = false }

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func lerpInt(a, b int, t float64) int {
	return int(float64(a) + (float64(b)-float64(a))*t)
}
