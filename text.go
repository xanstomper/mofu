package mofu

import (
	"github.com/mattn/go-runewidth"
	"strings"
)

// ---------------------------------------------------------------------------
// Text Alignment (Anthology Ch.8 §8.4)
// Separate from mofu.style go 'Align' to preserve Anthology API surface
// ---------------------------------------------------------------------------

// TextAlign specifies how text is aligned inside a line.
type TextAlign uint8

const (
	TextAlignLeft    TextAlign = 0
	TextAlignCenter  TextAlign = 1
	TextAlignRight   TextAlign = 2
	TextAlignJustify TextAlign = 3
)

// StyledSegment is a run of text sharing one mofu.Style.
type StyledSegment struct {
	Text  string
	Style Style
}

// StyledChar is a single character with display width and style.
type StyledChar struct {
	Ch    rune
	Width int
	Style Style
}

// TextLayout holds pre-computed layout for a block of styled text.
type TextLayout struct {
	Lines      []string
	Width      int
	Height     int
	LineWidths []int
}

// TextRendererConfig controls wrapping, ellipsis, and tab-width.
type TextRendererConfig struct {
	WrapWidth int
	Ellipsis  bool
	TabWidth  int
	WordWrap  bool
}

// DefaultTextRendererConfig returns sensible defaults.
func DefaultTextRendererConfig() TextRendererConfig {
	return TextRendererConfig{
		WrapWidth: 80,
		Ellipsis:  true,
		TabWidth:  4,
		WordWrap:  true,
	}
}

// ---------------------------------------------------------------------------
// Measure Width (Anthology Ch.8 §8.1) using runewidth
// ---------------------------------------------------------------------------

// RuneWidth returns the terminal display width of a rune via runewidth.
func RuneWidth(r rune) int { return runewidth.RuneWidth(r) }

// MeasureWidth returns the cell count of a string.
func MeasureWidth(text string) int {
	var w int
	for _, r := range text {
		w += runewidth.RuneWidth(r)
	}
	return w
}

// MeasureSegmentsWidth sums the cell width across a slice of segments.
func MeasureSegmentsWidth(segs []StyledSegment) int {
	var total int
	for _, seg := range segs {
		total += MeasureWidth(seg.Text)
	}
	return total
}

// ---------------------------------------------------------------------------
// Word Wrap (Anthology Ch.8 §8.1) no ellipsis
// ---------------------------------------------------------------------------

// WordWrap wraps text to maxWidth columns, preserving word boundaries.
func WordWrap(text string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{text}
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	var lines []string
	var cur strings.Builder
	var curW int

	for _, word := range words {
		ww := MeasureWidth(word)
		sp := 0
		if curW > 0 {
			sp = 1
		}
		if curW+sp+ww > maxWidth && curW > 0 {
			lines = append(lines, cur.String())
			cur.Reset()
			curW = 0
		}
		if curW > 0 {
			cur.WriteByte(' ')
			curW += 1
		}
		cur.WriteString(word)
		curW += ww
	}
	if cur.Len() > 0 {
		lines = append(lines, cur.String())
	}
	return lines
}

// CharWrap wraps character-by-character (CJK-safe).
func CharWrap(text string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{text}
	}
	var lines []string
	var cur strings.Builder
	var curW int
	for _, r := range text {
		cw := runewidth.RuneWidth(r)
		if curW+cw > maxWidth && curW > 0 {
			lines = append(lines, cur.String())
			cur.Reset()
			curW = 0
		}
		cur.WriteRune(r)
		curW += cw
	}
	if cur.Len() > 0 {
		lines = append(lines, cur.String())
	}
	return lines
}

// ---------------------------------------------------------------------------
// Truncate / Ellipsis (Anthology Ch.8 §8.6)
// ---------------------------------------------------------------------------

// EllipsisRune is the Unicode ellipsis character.
const EllipsisRune = '…'

// Truncate clips text to maxWidth and optionally adds an ellipsis.
func Truncate(text string, maxWidth int, withEllipsis bool) string {
	if maxWidth <= 0 {
		return ""
	}
	if MeasureWidth(text) <= maxWidth {
		return text
	}
	if withEllipsis && maxWidth > 1 {
		inner := Truncate(text, maxWidth-1, false)
		return inner + string(EllipsisRune)
	}
	var out strings.Builder
	var w int
	for _, r := range text {
		cw := runewidth.RuneWidth(r)
		if w+cw > maxWidth {
			break
		}
		out.WriteRune(r)
		w += cw
	}
	return out.String()
}

// TruncateMiddle truncates the middle: "Hello World" → "He…ld" at limit 5.
func TruncateMiddle(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if MeasureWidth(text) <= maxWidth || maxWidth < 3 {
		return Truncate(text, maxWidth, false)
	}
	runes := []rune(text)
	keep := maxWidth - 1
	left := keep / 2
	right := keep - left
	var out strings.Builder
	out.WriteString(string(runes[:left]))
	out.WriteRune(EllipsisRune)
	out.WriteString(string(runes[len(runes)-right:]))
	return out.String()
}

// ---------------------------------------------------------------------------
// Alignment (Anthology Ch.8 §8.4)
// ---------------------------------------------------------------------------

// PadRight pads with spaces on the right.
func PadRight(text string, width int) string {
	w := MeasureWidth(text)
	if w >= width {
		return text
	}
	return text + strings.Repeat(" ", width-w)
}

// PadLeft pads with spaces on the left.
func PadLeft(text string, width int) string {
	w := MeasureWidth(text)
	if w >= width {
		return text
	}
	return strings.Repeat(" ", width-w) + text
}

// PadCenter centers text.
func PadCenter(text string, width int) string {
	w := MeasureWidth(text)
	if w >= width {
		return text
	}
	l := (width - w) / 2
	r := width - w - l
	return strings.Repeat(" ", l) + text + strings.Repeat(" ", r)
}

// AlignLine returns text aligned per TextAlign rule within width.
func AlignLine(text string, width int, align TextAlign) string {
	if width <= 0 {
		return text
	}
	switch align {
	case TextAlignCenter:
		return PadCenter(text, width)
	case TextAlignRight:
		return PadLeft(text, width)
	case TextAlignJustify:
		return justifyText(text, width)
	default:
		return PadRight(text, width)
	}
}

// justifyText distributes extra spaces between words.
func justifyText(text string, width int) string {
	words := strings.Fields(text)
	if len(words) <= 1 || width <= 0 {
		return PadRight(text, width)
	}
	totalW := MeasureWidth(text)
	if totalW >= width {
		return text
	}
	extra := width - totalW
	gaps := len(words) - 1
	base := extra / gaps
	rem := extra % gaps
	var b strings.Builder
	for i, w := range words {
		if i > 0 {
			b.WriteString(strings.Repeat(" ", base+1))
			if i <= rem {
				b.WriteByte(' ')
			}
		}
		b.WriteString(w)
	}
	return b.String()
}

// ---------------------------------------------------------------------------
// Text Layout (Anthology Ch.8 §8.3)
// ---------------------------------------------------------------------------

// LayoutText computes a TextLayout with wrapping + alignment.
func LayoutText(text string, cfg TextRendererConfig, align TextAlign) TextLayout {
	maxW := cfg.WrapWidth
	if maxW <= 0 {
		maxW = 80
	}
	var raw []string
	if cfg.WordWrap {
		raw = WordWrap(text, maxW)
	} else {
		raw = CharWrap(text, maxW)
	}
	lines := make([]string, len(raw))
	widths := make([]int, len(raw))
	maxLayoutW := 0
	for i, line := range raw {
		aligned := AlignLine(line, maxW, align)
		lines[i] = aligned
		lw := MeasureWidth(aligned)
		widths[i] = lw
		if lw > maxLayoutW {
			maxLayoutW = lw
		}
	}
	return TextLayout{
		Lines:      lines,
		Width:      maxLayoutW,
		Height:     len(lines),
		LineWidths: widths,
	}
}

// LayoutSegments lays out StyledSegments into rows of at most maxWidth.
func LayoutSegments(segs []StyledSegment, maxWidth int) [][]StyledSegment {
	var rows [][]StyledSegment
	var cur []StyledSegment
	var curW int

	flush := func() {
		if len(cur) > 0 {
			rows = append(rows, append([]StyledSegment(nil), cur...))
		}
		cur = cur[:0]
		curW = 0
	}

	for _, seg := range segs {
		rs := []rune(seg.Text)
		avail := maxWidth
		if curW > 0 {
			avail -= 1
		}
		if len(rs) == 0 {
			continue
		}
		// Estimate fit by measuring
		if MeasureWidth(seg.Text)+curW <= maxWidth || curW == 0 {
			cur = append(cur, seg)
			curW += MeasureWidth(seg.Text)
			if curW >= maxWidth {
				flush()
			}
			continue
		}
		// Split seg across rows
		var before, after string
		var w int
		found := false
		for i, r := range rs {
			cw := runewidth.RuneWidth(r)
			if w+cw > avail && i > 0 {
				before = string(rs[:i])
				after = string(rs[i:])
				found = true
				break
			}
			w += cw
		}
		if !found {
			before = seg.Text
			after = ""
		}
		if before != "" {
			sp := seg
			sp.Text = before
			cur = append(cur, sp)
			curW += MeasureWidth(before)
		}
		flush()
		if after != "" {
			sp := seg
			sp.Text = after
			cur = append(cur, sp)
			curW = MeasureWidth(after)
		}
	}
	flush()
	return rows
}

// ---------------------------------------------------------------------------
// Rich Text Parser (Anthology Ch.8 §8.2)
// ---------------------------------------------------------------------------

// RichParser parses [bracketed-tag] rich text into StyledSegments.
type RichParser struct{}

// ParseRichText parses tagged text.
func ParseRichText(input string) []StyledSegment { return (&RichParser{}).Parse(input) }

// Parse walks the input and produces styled segments.
func (rp *RichParser) Parse(input string) []StyledSegment {
	var out []StyledSegment
	var cur strings.Builder
	style := DefaultStyle()

	flush := func() {
		if cur.Len() > 0 {
			out = append(out, StyledSegment{Text: cur.String(), Style: style})
			cur.Reset()
		}
	}

	rs := []rune(input)
	i := 0
	for i < len(rs) {
		if rs[i] == '[' {
			j := i + 1
			for j < len(rs) && rs[j] != ']' {
				j++
			}
			if j >= len(rs) {
				cur.WriteRune(rs[i])
				i++
				continue
			}
			flush()
			rp.applyTag(string(rs[i+1:j]), &style)
			i = j + 1
			continue
		}
		cur.WriteRune(rs[i])
		i++
	}
	flush()
	return out
}

// ApplyTag handles a tag. The `tag` label parameter distinguishes it from applyTag method.
func (rp *RichParser) ApplyTag(tag string, style *Style) { rp.applyTag(tag, style) }

// applyTag applies a mofu-style rich-text tag to the given style.
func (rp *RichParser) applyTag(tag string, style *Style) {
	switch tag {
	case "b", "bold":
		style.Attrs |= AttrBold
	case "i", "italic":
		style.Attrs |= AttrItalic
	case "u", "underline":
		style.Attrs |= AttrUnderline
	case "strike":
		style.Attrs |= AttrStrikethrough
	case "dim":
		style.Attrs |= AttrDim
	case "reverse":
		style.Attrs |= AttrReverse
	case "hidden":
		style.Attrs |= AttrHidden
	case "/", "reset":
		*style = DefaultStyle()
	default:
		if len(tag) > 0 && tag[0] == '#' && (len(tag) == 7 || len(tag) == 4) {
			style.Foreground = Hex(tag)
		} else if strings.HasPrefix(tag, "fg:") {
			style.Foreground = Hex(tag[3:])
		} else if strings.HasPrefix(tag, "bg:") {
			style.Background = Hex(tag[3:])
		}
	}
}

// ---------------------------------------------------------------------------
// Tabular helpers (Anthology §8.7)
// ---------------------------------------------------------------------------

// ColumnAlign holds width + alignment for a single column.
type ColumnAlign struct {
	Width int
	Align TextAlign
}

// FormatTable formats rows into aligned columns.
func FormatTable(headers []string, rows [][]string, cols []ColumnAlign) []string {
	if len(cols) == 0 {
		return nil
	}
	var result []string
	for _, row := range rows {
		var b strings.Builder
		for i, cell := range row {
			if i < len(cols) {
				b.WriteString(AlignLine(cell, cols[i].Width, cols[i].Align))
			} else {
				b.WriteString(cell)
			}
			if i < len(row)-1 {
				b.WriteByte(' ')
			}
		}
		result = append(result, b.String())
	}
	return result
}
