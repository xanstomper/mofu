package mofu

import (
	"strings"
	"unicode/utf8"
)

// Cell represents a single terminal cell with character and style.
type Cell struct {
	Char  rune
	Fg    Color
	Bg    Color
	Bold  bool
	Dirty bool
}

// CellBuffer holds a 2D grid of cells for the renderer.
type CellBuffer struct {
	Cells  [][]Cell
	Width  int
	Height int
}

// NewCellBuffer creates a new cell buffer with the given dimensions.
func NewCellBuffer(w, h int) *CellBuffer {
	cb := &CellBuffer{Width: w, Height: h}
	cb.Cells = make([][]Cell, h)
	for y := 0; y < h; y++ {
		cb.Cells[y] = make([]Cell, w)
	}
	return cb
}

// Clear marks all cells as dirty and resets them.
func (cb *CellBuffer) Clear() {
	for y := 0; y < cb.Height; y++ {
		for x := 0; x < cb.Width; x++ {
			cb.Cells[y][x] = Cell{Char: ' ', Dirty: true}
		}
	}
}

// Set places a character at the given position with style.
func (cb *CellBuffer) Set(x, y int, char rune, fg, bg Color, bold bool) {
	if x < 0 || x >= cb.Width || y < 0 || y >= cb.Height {
		return
	}
	cell := &cb.Cells[y][x]
	if cell.Char != char || cell.Fg != fg || cell.Bg != bg || cell.Bold != bold {
		cell.Char = char
		cell.Fg = fg
		cell.Bg = bg
		cell.Bold = bold
		cell.Dirty = true
	}
}

// Renderer handles drawing to the terminal.
type Renderer struct {
	front  *CellBuffer
	back   *CellBuffer
	width  int
	height int
	theme  *Theme
	styles map[string]Style
}

// NewRenderer creates a new renderer.
func NewRenderer(w, h int, theme *Theme) *Renderer {
	return &Renderer{
		front:  NewCellBuffer(w, h),
		back:   NewCellBuffer(w, h),
		width:  w,
		height: h,
		theme:  theme,
		styles: make(map[string]Style),
	}
}

// Resize resizes the renderer's buffers.
func (r *Renderer) Resize(w, h int) {
	r.width = w
	r.height = h
	r.front = NewCellBuffer(w, h)
	r.back = NewCellBuffer(w, h)
}

// Clear resets the front buffer.
func (r *Renderer) Clear() {
	r.front.Clear()
}

// SetStyle registers a named style.
func (r *Renderer) SetStyle(name string, s Style) {
	r.styles[name] = s
}

// GetStyle retrieves a named style.
func (r *Renderer) GetStyle(name string) Style {
	return r.styles[name]
}

// WriteString writes a string at the given position.
func (r *Renderer) WriteString(text string, x, y int, fg, bg Color, bold bool) {
	px := x
	for _, ch := range text {
		if ch == '\n' {
			y++
			px = x
			continue
		}
		if ch == '\t' {
			px += 4
			continue
		}
		if px >= r.width {
			px = x
			y++
			if y >= r.height {
				break
			}
		}
		r.front.Set(px, y, ch, fg, bg, bold)
		px++
	}
}

// WriteStyledString writes a string with inline style support.
func (r *Renderer) WriteStyledString(text string, x, y int, style Style) {
	r.WriteString(text, x, y, style.Foreground, style.Background, style.Bold)
}

// Flush outputs the differences between front and back buffers.
// Returns the escape sequences needed to update the screen.
func (r *Renderer) Flush() string {
	var sb strings.Builder
	for y := 0; y < r.height; y++ {
		for x := 0; x < r.width; x++ {
			frontCell := r.front.Cells[y][x]
			backCell := r.back.Cells[y][x]

			if frontCell.Dirty || frontCell != backCell {
				// Move cursor and apply style
				if frontCell.Char != backCell.Char ||
					frontCell.Fg != backCell.Fg ||
					frontCell.Bg != backCell.Bg ||
					frontCell.Bold != backCell.Bold {
					sb.WriteString(cursorPos(y, x))
					s := Style{Foreground: frontCell.Fg, Background: frontCell.Bg, Bold: frontCell.Bold}
					sb.WriteString(s.SGR())
					sb.WriteRune(frontCell.Char)
					sb.WriteString(s.Reset())
				}
				r.back.Cells[y][x] = frontCell
			}
		}
	}
	return sb.String()
}

func cursorPos(row, col int) string {
	return "\x1b[" + itoa(row+1) + ";" + itoa(col+1) + "H"
}

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

// UpdateRegion marks a rectangular region as dirty.
func (r *Renderer) UpdateRegion(x, y, w, h int) {
	for dy := y; dy < y+h && dy < r.height; dy++ {
		for dx := x; dx < x+w && dx < r.width; dx++ {
			if dy >= 0 && dx >= 0 {
				r.front.Cells[dy][dx].Dirty = true
			}
		}
	}
}

// RenderText renders text with word wrapping into a string.
func RenderText(text string, width int, style Style) string {
	if width <= 0 {
		return text
	}
	var result strings.Builder
	for _, line := range strings.Split(text, "\n") {
		words := strings.Fields(line)
		if len(words) == 0 {
			result.WriteString("\n")
			continue
		}
		lineLen := 0
		for _, word := range words {
			wordLen := utf8.RuneCountInString(word)
			if lineLen > 0 && lineLen+1+wordLen > width {
				result.WriteString("\n")
				lineLen = 0
			}
			if lineLen > 0 {
				result.WriteString(" ")
				lineLen++
			}
			result.WriteString(word)
			lineLen += wordLen
		}
		result.WriteString("\n")
	}
	return strings.TrimRight(result.String(), "\n")
}
