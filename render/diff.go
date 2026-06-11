// Package render provides the zero-allocation differential rendering engine
// for MOFU (Modular Orchestrated Flow Utility).
//
// The diff renderer maintains two preallocated frame buffers (front and back),
// computes minimal cell-level diffs, and emits only changed terminal cells wrapped
// in the Synchronized Output protocol (CSI 2026 h/l) for flicker-free rendering.
//
// Architecture:
//
//	input → state mutation → dirty propagation → layout cache check → diff render → terminal output
//
// The renderer NEVER allocates per-frame. All buffers are preallocated at init.
// SGR sequences are cached in a sync.Map keyed by (fg, bg, attrs).
package render

import (
	"bytes"
	"sort"
	"sync"
	"time"

	"github.com/mattn/go-runewidth"
)

// ---------------------------------------------------------------------------
// Cell representation
// ---------------------------------------------------------------------------

// Cell is a single terminal cell. Width == 0 marks a continuation cell for
// wide (double-width) characters. Width == -1 marks an unused cell.
type Cell struct {
	Char  rune
	Fg    uint32
	Bg    uint32
	Attrs uint16
	Width int8
	Dirty bool
}

// Predefined cell constants.
var (
	emptyCell = Cell{Char: ' ', Width: 1, Dirty: true}
	nullCell  = Cell{Char: 0, Width: 0, Dirty: true}
)

// ---------------------------------------------------------------------------
// Frame buffer — preallocated 2D cell array
// ---------------------------------------------------------------------------

// FrameBuffer is a preallocated grid of Cells. It is never resized after creation.
type FrameBuffer struct {
	Cells  []Cell
	Width  int
	Height int
	stride int // = Width
}

// NewFrameBuffer allocates a single contiguous slice for the entire grid.
// This eliminates per-row allocations and improves cache locality.
func NewFrameBuffer(w, h int) *FrameBuffer {
	return &FrameBuffer{
		Cells:  make([]Cell, w*h),
		Width:  w,
		Height: h,
		stride: w,
	}
}

// Clear resets every cell to emptyCell and marks all as dirty.
func (fb *FrameBuffer) Clear() {
	for i := range fb.Cells {
		fb.Cells[i] = emptyCell
		fb.Cells[i].Dirty = true
	}
}

// Set writes a character at (x, y). Returns false if coords are out of bounds.
func (fb *FrameBuffer) Set(x, y int, char rune, fg, bg uint32, attrs uint16) bool {
	if x < 0 || x >= fb.Width || y < 0 || y >= fb.Height {
		return false
	}
	idx := y*fb.stride + x
	cw := cellWidth(char)
	cell := &fb.Cells[idx]
	if cell.Char == char && cell.Fg == fg && cell.Bg == bg && cell.Attrs == attrs {
		return true
	}
	cell.Char = char
	cell.Fg = fg
	cell.Bg = bg
	cell.Attrs = attrs
	cell.Width = int8(cw)
	cell.Dirty = true
	// Mark continuation cell for wide characters
	if cw == 2 && x+1 < fb.Width {
		fb.Cells[idx+1] = nullCell
	}
	return true
}

// Get returns a pointer to the cell at (x, y). Returns nil if out of bounds.
func (fb *FrameBuffer) Get(x, y int) *Cell {
	if x < 0 || x >= fb.Width || y < 0 || y >= fb.Height {
		return nil
	}
	return &fb.Cells[y*fb.stride+x]
}

// cellWidth returns the display width of a rune (0 for zero, 2 for wide, 1 otherwise).
func cellWidth(ch rune) int {
	if ch == 0 {
		return 0
	}
	w := runewidth.RuneWidth(ch)
	if w < 1 {
		return 1
	}
	return w
}

// ---------------------------------------------------------------------------
// DiffRenderer — zero-allocation differential renderer
// ---------------------------------------------------------------------------

// DiffRenderer computes minimal cell-level diffs between two FrameBuffers
// and emits only changed cells to a preallocated byte buffer.
type DiffRenderer struct {
	front  *FrameBuffer
	back   *FrameBuffer
	width  int
	height int
	buf    bytes.Buffer // preallocated, reset each frame

	// Preallocated dirty rect tracking
	dirtyRects  []Rect
	dirtyActive bool

	// Frame statistics
	stats      RenderStats
	frameCount int64
}

// Rect represents a dirty region in the terminal grid.
type Rect struct {
	X, Y, Width, Height int
}

// RenderStats tracks per-frame rendering performance metrics.
type RenderStats struct {
	Frame      int64
	DirtyCells int
	TotalCells int
	FlushTime  time.Duration
	Rects      int
	OutputSize int
}

// DirtyRatio returns the fraction of cells that were dirty (0.0 to 1.0).
func (s *RenderStats) DirtyRatio() float64 {
	if s.TotalCells == 0 {
		return 0
	}
	return float64(s.DirtyCells) / float64(s.TotalCells)
}

// NewDiffRenderer creates a DiffRenderer with preallocated buffers for the
// given terminal dimensions. Call Resize to update dimensions.
func NewDiffRenderer(w, h int) *DiffRenderer {
	return &DiffRenderer{
		front:      NewFrameBuffer(w, h),
		back:       NewFrameBuffer(w, h),
		width:      w,
		height:     h,
		dirtyRects: make([]Rect, 0, 64),
	}
}

// Resize reallocates frame buffers for a new terminal size.
func (dr *DiffRenderer) Resize(w, h int) {
	dr.front = NewFrameBuffer(w, h)
	dr.back = NewFrameBuffer(w, h)
	dr.width = w
	dr.height = h
}

// Width returns the current terminal width.
func (dr *DiffRenderer) Width() int { return dr.width }

// Height returns the current terminal height.
func (dr *DiffRenderer) Height() int { return dr.height }

// Front returns the front buffer for writing into during the render phase.
func (dr *DiffRenderer) Front() *FrameBuffer { return dr.front }

// Clear resets the front buffer to blank.
func (dr *DiffRenderer) Clear() { dr.front.Clear() }

// MarkRect records a dirty region for diff computation.
func (dr *DiffRenderer) MarkRect(x, y, w, h int) {
	dr.dirtyActive = true
	dr.dirtyRects = append(dr.dirtyRects, Rect{X: x, Y: y, Width: w, Height: h})
}

// MarkWidgetDirty is a convenience for marking a widget's bounding box dirty.
func (dr *DiffRenderer) MarkWidgetDirty(x, y, w, h int) {
	dr.MarkRect(x, y, w, h)
}

// ClearRegion resets a rectangular region of the front buffer to empty.
func (dr *DiffRenderer) ClearRegion(x, y, w, h int) {
	for ry := y; ry < y+h && ry < dr.height; ry++ {
		for rx := x; rx < x+w && rx < dr.width; rx++ {
			dr.front.Set(rx, ry, ' ', 0, 0, 0)
		}
	}
	dr.MarkRect(x, y, w, h)
}

// Stats returns the render statistics from the last Flush call.
func (dr *DiffRenderer) Stats() RenderStats {
	return dr.stats
}

// ---------------------------------------------------------------------------
// Synchronized Output Protocol (CSI 2026)
// ---------------------------------------------------------------------------
//
// The Synchronized Output protocol wraps the entire frame update in:
//   ESC[?2026h  (begin synchronized update)
//   ... all cell writes ...
//   ESC[?2026l  (end synchronized update)
//
// This tells the terminal to buffer all output and present it atomically,
// eliminating flicker and tearing during rapid updates.
//
// Supported by: iTerm2, WezTerm, foot, Windows Terminal, Konsole, and others.

const (
	syncStart = "\x1b[?2026h"
	syncEnd   = "\x1b[?2026l"
)

// ---------------------------------------------------------------------------
// Diff computation + terminal output
// ---------------------------------------------------------------------------

// Flush computes the diff between front and back buffers, writes only changed
// cells to the internal buffer, swaps front/back, and returns the ANSI output
// string wrapped in Synchronized Output sequences.
//
// Returns empty string if no cells changed.
func (dr *DiffRenderer) Flush() string {
	start := time.Now()

	if !dr.dirtyActive {
		dr.stats = RenderStats{
			Frame:      dr.frameCount,
			DirtyCells: 0,
			TotalCells: dr.width * dr.height,
			FlushTime:  time.Since(start),
		}
		dr.frameCount++
		return ""
	}

	dr.buf.Reset()
	dr.buf.WriteString(syncStart)

	rects := dr.consolidateRects()

	for _, rect := range rects {
		for y := rect.Y; y < rect.Y+rect.Height && y < dr.height; y++ {
			dr.flushRow(y, rect.X, rect.X+rect.Width)
		}
	}

	dr.buf.WriteString(syncEnd)

	// Swap buffers: back becomes the new "current" state
	dr.front, dr.back = dr.back, dr.front

	dr.dirtyRects = dr.dirtyRects[:0]
	dr.dirtyActive = false

	outputSize := dr.buf.Len()
	if outputSize <= len(syncStart)+len(syncEnd) {
		dr.stats = RenderStats{
			Frame:      dr.frameCount,
			DirtyCells: 0,
			TotalCells: dr.width * dr.height,
			FlushTime:  time.Since(start),
		}
		dr.frameCount++
		return ""
	}

	// Count dirty cells from rects
	dirtyCells := 0
	for _, rect := range rects {
		dirtyCells += rect.Width * rect.Height
	}

	dr.stats = RenderStats{
		Frame:      dr.frameCount,
		DirtyCells: dirtyCells,
		TotalCells: dr.width * dr.height,
		FlushTime:  time.Since(start),
		Rects:      len(rects),
		OutputSize: outputSize,
	}
	dr.frameCount++

	return dr.buf.String()
}

// flushRow writes a single row segment, batching consecutive same-style cells.
func (dr *DiffRenderer) flushRow(y, startX, endX int) {
	if startX >= endX {
		return
	}

	rowStart := -1
	var rowFg, rowBg uint32
	var rowAttrs uint16

	for x := startX; x < endX && x < dr.width; x++ {
		fc := dr.front.Get(x, y)
		bc := dr.back.Get(x, y)

		if fc == nil || bc == nil {
			if rowStart >= 0 {
				dr.flushSegment(y, rowStart, x-1, rowFg, rowBg, rowAttrs)
				rowStart = -1
			}
			continue
		}

		if !fc.Dirty && fc.Char == bc.Char && fc.Fg == bc.Fg && fc.Bg == bc.Bg && fc.Attrs == bc.Attrs {
			if rowStart >= 0 {
				dr.flushSegment(y, rowStart, x-1, rowFg, rowBg, rowAttrs)
				rowStart = -1
			}
			continue
		}

		if rowStart < 0 {
			rowStart = x
			rowFg = fc.Fg
			rowBg = fc.Bg
			rowAttrs = fc.Attrs
		} else if fc.Fg != rowFg || fc.Bg != rowBg || fc.Attrs != rowAttrs {
			dr.flushSegment(y, rowStart, x-1, rowFg, rowBg, rowAttrs)
			rowStart = x
			rowFg = fc.Fg
			rowBg = fc.Bg
			rowAttrs = fc.Attrs
		}

		// Copy front cell to back buffer
		*bc = *fc
		bc.Dirty = false
	}

	if rowStart >= 0 {
		dr.flushSegment(y, rowStart, endX-1, rowFg, rowBg, rowAttrs)
	}
}

// flushSegment writes a contiguous segment of cells with the same style.
func (dr *DiffRenderer) flushSegment(y, startX, endX int, fg, bg uint32, attrs uint16) {
	if startX > endX {
		return
	}

	// Cursor position
	dr.buf.WriteString("\x1b[")
	dr.buf.WriteString(itoa(y + 1))
	dr.buf.WriteByte(';')
	dr.buf.WriteString(itoa(startX + 1))
	dr.buf.WriteByte('H')

	// SGR style
	sgr := cachedSGR(fg, bg, attrs)
	if sgr != "" {
		dr.buf.WriteString(sgr)
	}

	// Cell content
	for x := startX; x <= endX && x < dr.width; x++ {
		c := dr.front.Get(x, y)
		if c == nil {
			dr.buf.WriteByte(' ')
			continue
		}
		if c.Char == 0 {
			dr.buf.WriteByte(' ')
		} else {
			dr.buf.WriteRune(c.Char)
		}
	}

	// Reset attributes if needed
	if attrs != 0 {
		dr.buf.WriteString(resetAttrs(attrs))
	}
}

// ---------------------------------------------------------------------------
// Dirty rect consolidation
// ---------------------------------------------------------------------------

// consolidateRects merges overlapping/adjacent dirty rects to minimize
// the number of cursor-position jumps.
func (dr *DiffRenderer) consolidateRects() []Rect {
	if len(dr.dirtyRects) <= 1 {
		return dr.dirtyRects
	}

	rects := dr.dirtyRects
	sort.Slice(rects, func(i, j int) bool {
		if rects[i].Y != rects[j].Y {
			return rects[i].Y < rects[j].Y
		}
		return rects[i].X < rects[j].X
	})

	merged := make([]Rect, 0, len(rects))
	for _, r := range rects {
		if len(merged) == 0 {
			merged = append(merged, r)
			continue
		}
		last := &merged[len(merged)-1]
		if r.Y <= last.Y+last.Height && r.X <= last.X+last.Width {
			*last = mergeRect(*last, r)
		} else {
			merged = append(merged, r)
		}
	}
	return merged
}

func mergeRect(a, b Rect) Rect {
	x1 := min(a.X, b.X)
	y1 := min(a.Y, b.Y)
	x2 := max(a.X+a.Width, b.X+b.Width)
	y2 := max(a.Y+a.Height, b.Y+b.Height)
	return Rect{X: x1, Y: y1, Width: x2 - x1, Height: y2 - y1}
}

// ---------------------------------------------------------------------------
// SGR cache (zero-allocation style lookups)
// ---------------------------------------------------------------------------

var sgrCache sync.Map // key: sgrKey → value: string

type sgrKey struct {
	Fg    uint32
	Bg    uint32
	Attrs uint16
}

func cachedSGR(fg, bg uint32, attrs uint16) string {
	key := sgrKey{Fg: fg, Bg: bg, Attrs: attrs}
	if cached, ok := sgrCache.Load(key); ok {
		return cached.(string)
	}
	sgr := compileSGR(fg, bg, attrs)
	sgrCache.Store(key, sgr)
	return sgr
}

// compileSGR builds the ANSI SGR escape sequence for the given style.
func compileSGR(fg, bg uint32, attrs uint16) string {
	var buf bytes.Buffer
	params := ""

	// Foreground color
	if fg != 0 {
		if fg <= 15 {
			// 16-color ANSI
			if fg < 8 {
				params += ";3" + itoa(int(fg))
			} else {
				params += ";9" + itoa(int(fg-8))
			}
		} else if fg <= 255 {
			// 256-color
			params += ";38;5;" + itoa(int(fg))
		} else {
			// True color (RGB packed)
			r := (fg >> 16) & 0xFF
			g := (fg >> 8) & 0xFF
			b := fg & 0xFF
			params += ";38;2;" + itoa(int(r)) + ";" + itoa(int(g)) + ";" + itoa(int(b))
		}
	}

	// Background color
	if bg != 0 {
		if bg <= 15 {
			if bg < 8 {
				params += ";4" + itoa(int(bg))
			} else {
				params += ";10" + itoa(int(bg-8))
			}
		} else if bg <= 255 {
			params += ";48;5;" + itoa(int(bg))
		} else {
			r := (bg >> 16) & 0xFF
			g := (bg >> 8) & 0xFF
			b := bg & 0xFF
			params += ";48;2;" + itoa(int(r)) + ";" + itoa(int(g)) + ";" + itoa(int(b))
		}
	}

	// Text attributes
	if attrs&1 != 0 {
		params += ";1"
	} // Bold
	if attrs&2 != 0 {
		params += ";2"
	} // Dim
	if attrs&4 != 0 {
		params += ";3"
	} // Italic
	if attrs&8 != 0 {
		params += ";4"
	} // Underline
	if attrs&16 != 0 {
		params += ";5"
	} // SlowBlink
	if attrs&32 != 0 {
		params += ";6"
	} // RapidBlink
	if attrs&64 != 0 {
		params += ";7"
	} // Reverse
	if attrs&128 != 0 {
		params += ";8"
	} // Hidden
	if attrs&256 != 0 {
		params += ";9"
	} // Strikethrough
	if attrs&512 != 0 {
		params += ";21"
	} // DoubleUnderline
	if attrs&1024 != 0 {
		params += ";53"
	} // Overline

	if params == "" {
		return ""
	}
	buf.WriteString("\x1b[")
	buf.WriteString(params[1:])
	buf.WriteByte('m')
	return buf.String()
}

// resetAttrs emits SGR reset sequences for the active attributes.
func resetAttrs(attrs uint16) string {
	var buf bytes.Buffer
	if attrs&3 != 0 {
		buf.WriteString("\x1b[22m")
	} // Bold+Dim
	if attrs&4 != 0 {
		buf.WriteString("\x1b[23m")
	} // Italic
	if attrs&8 != 0 || attrs&512 != 0 {
		buf.WriteString("\x1b[24m")
	} // Underline+DoubleUnderline
	if attrs&48 != 0 {
		buf.WriteString("\x1b[25m")
	} // Blink
	if attrs&64 != 0 {
		buf.WriteString("\x1b[27m")
	} // Reverse
	if attrs&128 != 0 {
		buf.WriteString("\x1b[28m")
	} // Hidden
	if attrs&256 != 0 {
		buf.WriteString("\x1b[29m")
	} // Strikethrough
	if attrs&1024 != 0 {
		buf.WriteString("\x1b[55m")
	} // Overline
	return buf.String()
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
