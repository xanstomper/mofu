package mofu

import (
	"bytes"
	"strings"
	"unicode/utf8"
)

type SceneCell struct {
	Char                    rune
	Fg, Bg                  Color
	Bold, Italic, Underline bool
	Dirty                   bool
}

type SceneBuffer struct {
	Cells         [][]SceneCell
	Width, Height int
	minX, minY    int
	maxX, maxY    int
	hasDirty      bool
}

func NewSceneBuffer(w, h int) *SceneBuffer {
	sb := &SceneBuffer{Width: w, Height: h}
	sb.Cells = make([][]SceneCell, h)
	for y := 0; y < h; y++ {
		sb.Cells[y] = make([]SceneCell, w)
	}
	sb.minX = w
	sb.minY = h
	return sb
}

func (sb *SceneBuffer) Clear() {
	sb.minX = sb.Width
	sb.minY = sb.Height
	sb.maxX = 0
	sb.maxY = 0
	sb.hasDirty = false
	for y := 0; y < sb.Height; y++ {
		for x := 0; x < sb.Width; x++ {
			sb.Cells[y][x] = SceneCell{Char: ' ', Dirty: true}
		}
	}
}

func (sb *SceneBuffer) Set(x, y int, char rune, fg, bg Color, bold, italic, underline bool) {
	if x < 0 || x >= sb.Width || y < 0 || y >= sb.Height {
		return
	}
	cell := &sb.Cells[y][x]
	if cell.Char != char || cell.Fg != fg || cell.Bg != bg || cell.Bold != bold {
		cell.Char = char
		cell.Fg = fg
		cell.Bg = bg
		cell.Bold = bold
		cell.Italic = italic
		cell.Underline = underline
		cell.Dirty = true
		if x < sb.minX {
			sb.minX = x
		}
		if y < sb.minY {
			sb.minY = y
		}
		if x > sb.maxX {
			sb.maxX = x
		}
		if y > sb.maxY {
			sb.maxY = y
		}
		sb.hasDirty = true
	}
}

type Renderer struct {
	front, back   *SceneBuffer
	width, height int
	theme         *Theme
	buf           bytes.Buffer
}

func NewRenderer(w, h int, theme *Theme) *Renderer {
	return &Renderer{
		front:  NewSceneBuffer(w, h),
		back:   NewSceneBuffer(w, h),
		width:  w,
		height: h,
		theme:  theme,
	}
}

func (r *Renderer) Resize(w, h int) {
	r.width = w
	r.height = h
	r.front = NewSceneBuffer(w, h)
	r.back = NewSceneBuffer(w, h)
}

func (r *Renderer) Clear() {
	r.front.Clear()
}

func (r *Renderer) WriteString(text string, x, y int, fg, bg Color, bold, italic, underline bool) {
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
		r.front.Set(px, y, ch, fg, bg, bold, italic, underline)
		px++
	}
}

func (r *Renderer) WriteStyledString(text string, x, y int, style Style) {
	r.WriteString(text, x, y, style.Foreground, style.Background, style.Bold, style.Italic, style.Underline)
}

func (r *Renderer) Flush() string {
	if !r.front.hasDirty && !r.back.hasDirty {
		return ""
	}
	r.buf.Reset()
	startY := r.front.minY
	endY := r.front.maxY + 1
	if endY <= startY {
		endY = r.height
		startY = 0
	}
	if startY < 0 {
		startY = 0
	}
	if endY > r.height {
		endY = r.height
	}
	for y := startY; y < endY; y++ {
		for x := 0; x < r.width; x++ {
			fc := r.front.Cells[y][x]
			bc := r.back.Cells[y][x]
			if !fc.Dirty && fc.Char == bc.Char && fc.Fg == bc.Fg && fc.Bg == bc.Bg && fc.Bold == bc.Bold {
				continue
			}
			r.buf.WriteString(cursorPos(y, x))
			if fc.Char != ' ' || fc.Bold || fc.Fg != (Color{}) || fc.Bg != (Color{}) {
				s := Style{Foreground: fc.Fg, Background: fc.Bg, Bold: fc.Bold}
				r.buf.WriteString(s.SGR())
				r.buf.WriteRune(fc.Char)
				r.buf.WriteString(s.Reset())
			} else {
				r.buf.WriteRune(fc.Char)
			}
			r.back.Cells[y][x] = fc
		}
	}
	r.front.hasDirty = false
	r.back.hasDirty = false
	return r.buf.String()
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

func RenderText(text string, width int) string {
	if width <= 0 {
		return text
	}
	var result strings.Builder
	for _, line := range strings.Split(text, "\n") {
		words := strings.Fields(line)
		if len(words) == 0 {
			result.WriteByte('\n')
			continue
		}
		lineLen := 0
		for _, word := range words {
			wordLen := utf8.RuneCountInString(word)
			if lineLen > 0 && lineLen+1+wordLen > width {
				result.WriteByte('\n')
				lineLen = 0
			}
			if lineLen > 0 {
				result.WriteByte(' ')
				lineLen++
			}
			result.WriteString(word)
			lineLen += wordLen
		}
		result.WriteByte('\n')
	}
	return strings.TrimRight(result.String(), "\n")
}
