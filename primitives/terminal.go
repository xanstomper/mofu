package primitives

import (
	"fmt"
	"image"
	"image/color"
	"sort"
	"strings"
	"unicode/utf8"
)

// ---------------------------------------------------------------------------
// Terminal primitives from Anthology primitive corpus
// ---------------------------------------------------------------------------

// Cell is a terminal cell with glyph and style.
type Cell struct {
	Glyph rune
	Width int
	Fg    uint32
	Bg    uint32
	Attrs Attrs
}

// Attrs stores terminal text attributes.
type Attrs uint32

const (
	AttrBold Attrs = 1 << iota
	AttrItalic
	AttrUnderline
	AttrBlink
	AttrReverse
	AttrDim
)

// Buffer is an off-screen cell framebuffer.
type Buffer struct {
	Width  int
	Height int
	Cells  []Cell
	Dirty  []bool
}

// NewBuffer returns a Buffer.
func NewBuffer(width, height int) *Buffer {
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	return &Buffer{Width: width, Height: height, Cells: make([]Cell, width*height), Dirty: make([]bool, width*height)}
}

// Set writes a cell and marks dirty if changed.
func (b *Buffer) Set(x, y int, c Cell) {
	if x < 0 || y < 0 || x >= b.Width || y >= b.Height {
		return
	}
	idx := y*b.Width + x
	if b.Cells[idx] != c {
		b.Cells[idx] = c
		b.Dirty[idx] = true
	}
}

// Get returns a cell.
func (b *Buffer) Get(x, y int) (Cell, bool) {
	if x < 0 || y < 0 || x >= b.Width || y >= b.Height {
		return Cell{}, false
	}
	return b.Cells[y*b.Width+x], true
}

// WriteString writes text at x,y.
func (b *Buffer) WriteString(x, y int, text string, fg, bg uint32, attrs Attrs) {
	for _, r := range text {
		if r == '\n' {
			y++
			continue
		}
		w := utf8.RuneLen(r)
		if w < 0 {
			w = 1
		}
		b.Set(x, y, Cell{Glyph: r, Width: w, Fg: fg, Bg: bg, Attrs: attrs})
		x += w
	}
}

// DirtyRects returns consolidated dirty rectangles.
func (b *Buffer) DirtyRects() []image.Rectangle {
	var rects []image.Rectangle
	for y := 0; y < b.Height; y++ {
		start := -1
		for x := 0; x < b.Width; x++ {
			idx := y*b.Width + x
			if b.Dirty[idx] && start < 0 {
				start = x
			}
			if (!b.Dirty[idx] || x == b.Width-1) && start >= 0 {
				end := x
				if !b.Dirty[idx] {
					end--
				}
				rects = append(rects, image.Rect(start, y, end+1, y+1))
				start = -1
			}
		}
	}
	return rects
}

// ClearDirty clears dirty flags.
func (b *Buffer) ClearDirty() {
	for i := range b.Dirty {
		b.Dirty[i] = false
	}
}

// ANSI escape helpers.

// CursorUp returns CSI A.
func CursorUp(n int) string { return fmt.Sprintf("\x1b[%dA", n) }

// CursorDown returns CSI B.
func CursorDown(n int) string { return fmt.Sprintf("\x1b[%dB", n) }

// CursorRight returns CSI C.
func CursorRight(n int) string { return fmt.Sprintf("\x1b[%dC", n) }

// CursorLeft returns CSI D.
func CursorLeft(n int) string { return fmt.Sprintf("\x1b[%dD", n) }

// CursorMove returns CSI row;col H.
func CursorMove(row, col int) string { return fmt.Sprintf("\x1b[%d;%dH", row, col) }

// ClearScreen returns CSI 2 J.
func ClearScreen() string { return "\x1b[2J" }

// ClearLine returns CSI K.
func ClearLine() string { return "\x1b[K" }

// HideCursor returns CSI ?25 l.
func HideCursor() string { return "\x1b[?25l" }

// ShowCursor returns CSI ?25 h.
func ShowCursor() string { return "\x1b[?25h" }

// SGR returns SGR sequence.
func SGR(codes ...int) string {
	if len(codes) == 0 {
		return "\x1b[0m"
	}
	parts := make([]string, len(codes))
	for i, c := range codes {
		parts[i] = fmt.Sprint(c)
	}
	return "\x1b[" + strings.Join(parts, ";") + "m"
}

// TrueColorFG returns 38;2 RGB SGR.
func TrueColorFG(r, g, b uint8) string { return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b) }

// TrueColorBG returns 48;2 RGB SGR.
func TrueColorBG(r, g, b uint8) string { return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b) }

// KittyKeyboardEnable enables kitty keyboard progressive enhancement.
func KittyKeyboardEnable() string { return "\x1b[>1u" }

// KittyKeyboardDisable disables kitty keyboard protocol.
func KittyKeyboardDisable() string { return "\x1b[<1u" }

// SynchronizedOutputBegin starts synchronized output.
func SynchronizedOutputBegin() string { return "\x1b[?2026h" }

// SynchronizedOutputEnd ends synchronized output.
func SynchronizedOutputEnd() string { return "\x1b[?2026l" }

// BracketedPasteBegin enables bracketed paste.
func BracketedPasteBegin() string { return "\x1b[?2004h" }

// BracketedPasteEnd disables bracketed paste.
func BracketedPasteEnd() string { return "\x1b[?2004l" }

// Image primitives (Chafa/Sixel placeholders).

// SixelFromRGBA converts an image to a simple sixel string.
func SixelFromRGBA(img image.Image) string {
	bounds := img.Bounds()
	var sb strings.Builder
	sb.WriteString("\x1bPq")
	for y := bounds.Min.Y; y < bounds.Max.Y; y += 6 {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			for bit := 0; bit < 6; bit++ {
				yy := y + bit
				if yy >= bounds.Max.Y {
					continue
				}
				c := img.At(x, yy)
				r, g, b, _ := c.RGBA()
				if r > 0 || g > 0 || b > 0 {
					sb.WriteByte(byte('?' + bit))
				}
			}
			sb.WriteByte('#')
			c := img.At(x, y)
			sb.WriteString(fmt.Sprintf("%d;%d;%d", colorComponent(c), colorComponent(c), colorComponent(c)))
			sb.WriteByte(';')
		}
		sb.WriteByte('$')
		sb.WriteByte('\n')
	}
	sb.WriteString("\x1b\\")
	return sb.String()
}

func colorComponent(c color.Color) int {
	r, _, _, _ := c.RGBA()
	return int(r >> 8)
}

// Unicode plots.

// ScaleToGrid maps continuous values to terminal grid cells.
func ScaleToGrid(values []float64, width, height int) [][]bool {
	grid := make([][]bool, height)
	for i := range grid {
		grid[i] = make([]bool, width)
	}
	if len(values) == 0 || width <= 0 || height <= 0 {
		return grid
	}
	minVal, maxVal := values[0], values[0]
	for _, v := range values {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}
	if maxVal == minVal {
		maxVal = minVal + 1
	}
	step := float64(len(values)-1) / float64(width-1)
	for x := 0; x < width; x++ {
		idx := int(float64(x) * step)
		if idx >= len(values) {
			idx = len(values) - 1
		}
		y := height - 1 - int((values[idx]-minVal)/(maxVal-minVal)*float64(height-1))
		if y >= 0 && y < height {
			grid[y][x] = true
		}
	}
	return grid
}

// UnscaleFromGrid maps a grid cell back to a value index.
func UnscaleFromGrid(x, y, width, height int, minVal, maxVal float64) float64 {
	if width <= 1 {
		return minVal
	}
	if height <= 1 {
		return minVal
	}
	nx := float64(x) / float64(width-1)
	ny := 1 - float64(y)/float64(height-1)
	_ = nx
	return minVal + (maxVal-minVal)*ny
}

// Text effects.

// EffectChain applies effects sequentially.
type EffectChain []func(string) string

// Apply applies all effects in order.
func (c EffectChain) Apply(s string) string {
	for _, fn := range c {
		if fn != nil {
			s = fn(s)
		}
	}
	return s
}

// EffectGroup applies effects in parallel and joins results.
type EffectGroup []func(string) string

// Apply applies all effects and joins with newline.
func (g EffectGroup) Apply(s string) string {
	parts := make([]string, 0, len(g))
	for _, fn := range g {
		if fn != nil {
			parts = append(parts, fn(s))
		}
	}
	return strings.Join(parts, "\n")
}

// RichStyle is a simple rich-terminal style.
type RichStyle struct {
	Fg        string
	Bg        string
	Bold      bool
	Italic    bool
	Underline bool
}

// RenderRich returns an ANSI-styled string.
func RenderRich(text string, style RichStyle) string {
	var codes []int
	if style.Bold {
		codes = append(codes, 1)
	}
	if style.Italic {
		codes = append(codes, 3)
	}
	if style.Underline {
		codes = append(codes, 4)
	}
	if style.Fg != "" {
		codes = append(codes, 38, 2, 0, 0, 0)
	}
	if style.Bg != "" {
		codes = append(codes, 48, 2, 0, 0, 0)
	}
	if len(codes) > 0 {
		return SGR(codes...) + text + SGR(0)
	}
	return text
}

// Fuzzy search primitive.

// FuzzyMatch returns true if needle is a subsequence of haystack.
func FuzzyMatch(haystack, needle string) bool {
	h := []rune(strings.ToLower(haystack))
	n := []rune(strings.ToLower(needle))
	if len(n) == 0 {
		return true
	}
	j := 0
	for _, r := range h {
		if r == n[j] {
			j++
			if j == len(n) {
				return true
			}
		}
	}
	return false
}

// TopN returns top n scored matches sorted descending.
func TopN(items []FuzzyItem, n int) []FuzzyItem {
	cp := append([]FuzzyItem(nil), items...)
	sort.Slice(cp, func(i, j int) bool { return cp[i].Score > cp[j].Score })
	if n < len(cp) {
		cp = cp[:n]
	}
	return cp
}

// FuzzyItem pairs a label with a score.
type FuzzyItem struct {
	Label string
	Score int
}
