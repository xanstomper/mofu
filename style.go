package mofu

import "fmt"

type BorderStyle struct {
	Top, Bottom, Left, Right                   rune
	TopLeft, TopRight, BottomLeft, BottomRight rune
}

var (
	BorderNone   = BorderStyle{}
	BorderHidden = BorderStyle{}
	BorderNormal = BorderStyle{
		Top: '─', Bottom: '─', Left: '│', Right: '│',
		TopLeft: '┌', TopRight: '┐', BottomLeft: '└', BottomRight: '┘',
	}
	BorderRounded = BorderStyle{
		Top: '─', Bottom: '─', Left: '│', Right: '│',
		TopLeft: '╭', TopRight: '╮', BottomLeft: '╰', BottomRight: '╯',
	}
	BorderThick = BorderStyle{
		Top: '━', Bottom: '━', Left: '┃', Right: '┃',
		TopLeft: '┏', TopRight: '┓', BottomLeft: '┗', BottomRight: '┛',
	}
	BorderDouble = BorderStyle{
		Top: '═', Bottom: '═', Left: '║', Right: '║',
		TopLeft: '╔', TopRight: '╗', BottomLeft: '╚', BottomRight: '╝',
	}
)

type Spacing struct{ Top, Right, Bottom, Left int }

type Align int

const (
	AlignLeft    Align = 0
	AlignCenter  Align = 1
	AlignRight   Align = 2
	AlignStretch Align = 3
)

type Justify int

const (
	JustifyStart        Justify = 0
	JustifyCenter       Justify = 1
	JustifyEnd          Justify = 2
	JustifySpaceBetween Justify = 3
)

type Direction int

const (
	DirectionRow    Direction = 0
	DirectionColumn Direction = 1
)

type AttrMask uint16

const (
	AttrBold            AttrMask = 1 << 0
	AttrDim             AttrMask = 1 << 1
	AttrItalic          AttrMask = 1 << 2
	AttrUnderline       AttrMask = 1 << 3
	AttrSlowBlink       AttrMask = 1 << 4
	AttrRapidBlink      AttrMask = 1 << 5
	AttrReverse         AttrMask = 1 << 6
	AttrHidden          AttrMask = 1 << 7
	AttrStrikethrough   AttrMask = 1 << 8
	AttrDoubleUnderline AttrMask = 1 << 9
	AttrOverline        AttrMask = 1 << 10
)

func (a AttrMask) Has(flag AttrMask) bool { return a&flag != 0 }

type Style struct {
	Foreground Color
	Background Color
	Attrs      AttrMask
	Border     BorderStyle
	Margin     Spacing
	Padding    Spacing
	Width      int
	Height     int
	MinWidth   int
	MinHeight  int
	MaxWidth   int
	MaxHeight  int
	Align      Align
	Gap        int
	Grow       float64
	Shrink     float64
	Direction  Direction
	Opacity    float64
	OffsetX    int
	OffsetY    int
	Gutter     int
	cachedSGR  string
}

func DefaultStyle() Style {
	return Style{
		Foreground: ColorWhite,
		Background: ColorTransparent,
		Opacity:    1.0,
	}
}

func (s Style) Fg(c Color) Style {
	s.Foreground = c
	s.cachedSGR = ""
	return s
}

func (s Style) Bg(c Color) Style {
	s.Background = c
	s.cachedSGR = ""
	return s
}

func (s Style) SGR() string {
	if s.cachedSGR != "" {
		return s.cachedSGR
	}
	s.cachedSGR = s.compileSGR()
	return s.cachedSGR
}

func (s Style) Reset() string {
	return "\x1b[0m"
}

func (s Style) Apply(text string) string {
	if text == "" {
		return ""
	}
	return s.SGR() + text + s.Reset()
}

func (s Style) compileSGR() string {
	var out string
	if !s.Foreground.IsANSI || s.Foreground != (Color{}) {
		out += s.Foreground.foreground()
	}
	if !s.Background.IsANSI || s.Background != (Color{}) {
		out += s.Background.background()
	}
	params := ""
	if s.Attrs.Has(AttrBold) {
		params += ";1"
	}
	if s.Attrs.Has(AttrDim) {
		params += ";2"
	}
	if s.Attrs.Has(AttrItalic) {
		params += ";3"
	}
	if s.Attrs.Has(AttrUnderline) {
		params += ";4"
	}
	if s.Attrs.Has(AttrSlowBlink) {
		params += ";5"
	}
	if s.Attrs.Has(AttrRapidBlink) {
		params += ";6"
	}
	if s.Attrs.Has(AttrReverse) {
		params += ";7"
	}
	if s.Attrs.Has(AttrHidden) {
		params += ";8"
	}
	if s.Attrs.Has(AttrStrikethrough) {
		params += ";9"
	}
	if s.Attrs.Has(AttrDoubleUnderline) {
		params += ";21"
	}
	if s.Attrs.Has(AttrOverline) {
		params += ";53"
	}
	if params != "" {
		out += "\x1b[" + params[1:] + "m"
	}
	return out
}

func ResetAttrs(in AttrMask) string {
	var out string
	if in.Has(AttrBold) || in.Has(AttrDim) {
		out += "\x1b[22m"
	}
	if in.Has(AttrItalic) {
		out += "\x1b[23m"
	}
	if in.Has(AttrUnderline) || in.Has(AttrDoubleUnderline) {
		out += "\x1b[24m"
	}
	if in.Has(AttrSlowBlink) || in.Has(AttrRapidBlink) {
		out += "\x1b[25m"
	}
	if in.Has(AttrReverse) {
		out += "\x1b[27m"
	}
	if in.Has(AttrHidden) {
		out += "\x1b[28m"
	}
	if in.Has(AttrStrikethrough) {
		out += "\x1b[29m"
	}
	if in.Has(AttrOverline) {
		out += "\x1b[55m"
	}
	return out
}

func Width(w int) string {
	return fmt.Sprintf("\x1b[%dG", w)
}
