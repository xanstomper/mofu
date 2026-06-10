package mofu

import (
	"bytes"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

type Attrs struct {
	Bold, Italic, Underline bool
}

type SceneCell struct {
	Char   rune
	Fg, Bg Color
	Attrs  AttrMask
	Dirty  bool
	Width  int
}

type SceneBuffer struct {
	Cells         [][]SceneCell
	Width, Height int
	dirtyRects    []Rect
	hasDirty      bool
	pool          sync.Pool
}

func NewSceneBuffer(w, h int) *SceneBuffer {
	sb := &SceneBuffer{Width: w, Height: h}
	sb.Cells = make([][]SceneCell, h)
	for y := 0; y < h; y++ {
		sb.Cells[y] = make([]SceneCell, w)
	}
	sb.pool.New = func() any {
		return &bytes.Buffer{}
	}
	return sb
}

func (sb *SceneBuffer) CellWidth(ch rune) int {
	if ch == 0 {
		return 0
	}
	w := runewidth.RuneWidth(ch)
	if w < 1 {
		return 1
	}
	return w
}

func (sb *SceneBuffer) Clear() {
	sb.dirtyRects = nil
	sb.hasDirty = false
	for y := 0; y < sb.Height; y++ {
		for x := 0; x < sb.Width; x++ {
			sb.Cells[y][x] = SceneCell{Char: ' ', Dirty: true}
		}
	}
	sb.markRect(0, 0, sb.Width, sb.Height)
}

func (sb *SceneBuffer) Set(x, y int, char rune, fg, bg Color, attrs AttrMask) {
	if x < 0 || x >= sb.Width || y < 0 || y >= sb.Height {
		return
	}
	cw := sb.CellWidth(char)
	cell := &sb.Cells[y][x]
	if cell.Char != char || cell.Fg != fg || cell.Bg != bg || cell.Attrs != attrs {
		cell.Char = char
		cell.Fg = fg
		cell.Bg = bg
		cell.Attrs = attrs
		cell.Width = cw
		cell.Dirty = true
		sb.markRect(x, y, cw, 1)
		if cw == 2 && x+1 < sb.Width {
			sb.Cells[y][x+1] = SceneCell{Char: 0, Width: 0, Dirty: true}
		}
	}
}

func (sb *SceneBuffer) markRect(x, y, w, h int) {
	sb.hasDirty = true
	sb.dirtyRects = append(sb.dirtyRects, Rect{X: x, Y: y, Width: w, Height: h})
}

func (sb *SceneBuffer) consolidateRects() []Rect {
	if len(sb.dirtyRects) <= 1 {
		return sb.dirtyRects
	}
	rects := sb.dirtyRects
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
			*last = mergeRects(*last, r)
		} else {
			merged = append(merged, r)
		}
	}
	return merged
}

var ansiCache sync.Map

func cachedSGR(fg, bg Color, attrs AttrMask) string {
	key := styleKey{F: fg, B: bg, A: attrs}
	if cached, ok := ansiCache.Load(key); ok {
		return cached.(string)
	}
	s := Style{Foreground: fg, Background: bg, Attrs: attrs}
	sgr := s.SGR()
	ansiCache.Store(key, sgr)
	return sgr
}

type styleKey struct {
	F Color
	B Color
	A AttrMask
}

type Renderer struct {
	front, back   *SceneBuffer
	width, height int
	theme         *Theme
	lastRowStyle  []AttrMask
	buf           bytes.Buffer
}

func NewRenderer(w, h int, theme *Theme) *Renderer {
	return &Renderer{
		front:        NewSceneBuffer(w, h),
		back:         NewSceneBuffer(w, h),
		width:        w,
		height:       h,
		theme:        theme,
		lastRowStyle: make([]AttrMask, h),
	}
}

func (r *Renderer) Resize(w, h int) {
	r.width = w
	r.height = h
	r.front = NewSceneBuffer(w, h)
	r.back = NewSceneBuffer(w, h)
	r.lastRowStyle = make([]AttrMask, h)
}

func (r *Renderer) Clear() {
	r.front.Clear()
}

func (r *Renderer) WriteString(text string, x, y int, fg, bg Color, attrs AttrMask) {
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
		r.front.Set(px, y, ch, fg, bg, attrs)
		px += r.front.CellWidth(ch)
	}
}

func (r *Renderer) WriteStyledString(text string, x, y int, style Style) {
	r.WriteString(text, x, y, style.Foreground, style.Background, style.Attrs)
}

func (r *Renderer) Flush() string {
	if !r.front.hasDirty {
		return ""
	}
	r.buf.Reset()
	rects := r.front.consolidateRects()

	for _, rect := range rects {
		for y := rect.Y; y < rect.Y+rect.Height && y < r.height; y++ {
			rowStart := -1
			var rowAttrs AttrMask
			var rowFg, rowBg Color

			for x := rect.X; x < rect.X+rect.Width && x < r.width; x++ {
				fc := r.front.Cells[y][x]
				bc := r.back.Cells[y][x]
				if !fc.Dirty && fc.Char == bc.Char && fc.Fg == bc.Fg && fc.Bg == bc.Bg && fc.Attrs == bc.Attrs {
					if rowStart >= 0 {
						r.flushRowSegment(&fc, x, y, rowStart, x-1, rowFg, rowBg, rowAttrs)
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
					r.flushRowSegment(&fc, x, y, rowStart, x-1, rowFg, rowBg, rowAttrs)
					rowStart = x
					rowFg = fc.Fg
					rowBg = fc.Bg
					rowAttrs = fc.Attrs
				}
				r.back.Cells[y][x] = fc
			}
			if rowStart >= 0 {
				r.flushRowSegment(nil, rect.X+rect.Width, y, rowStart, rect.X+rect.Width-1, rowFg, rowBg, rowAttrs)
			}
		}
	}
	r.front.hasDirty = false
	r.front.dirtyRects = nil

	if r.buf.Len() > 0 {
		return r.buf.String()
	}
	return ""
}

func (r *Renderer) flushRowSegment(fc *SceneCell, endX, y, startX, segEnd int, fg Color, bg Color, attrs AttrMask) {
	if startX > segEnd {
		return
	}
	r.buf.WriteString(cursorPos(y, startX))
	sgr := cachedSGR(fg, bg, attrs)
	if sgr != "" {
		r.buf.WriteString(sgr)
	}
	for x := startX; x <= segEnd && x < r.width; x++ {
		c := r.front.Cells[y][x]
		if c.Char == 0 {
			r.buf.WriteByte(' ')
		} else {
			r.buf.WriteRune(c.Char)
		}
		r.back.Cells[y][x].Dirty = false
	}
	if attrs != 0 {
		r.buf.WriteString(ResetAttrs(attrs))
	}
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
