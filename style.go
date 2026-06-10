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

type Spacing struct {
	Top, Right, Bottom, Left int
}

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

type Style struct {
	Foreground Color
	Background Color
	Bold       bool
	Italic     bool
	Underline  bool
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
}

func DefaultStyle() Style {
	return Style{
		Foreground: ColorWhite,
		Background: ColorTransparent,
		Opacity:    1.0,
	}
}

func (s Style) SGR() string {
	var out string
	if !s.Foreground.IsANSI || s.Foreground != (Color{}) {
		out += s.Foreground.foreground()
	}
	if !s.Background.IsANSI || s.Background != (Color{}) {
		out += s.Background.background()
	}
	if s.Bold {
		out += "\x1b[1m"
	}
	if s.Italic {
		out += "\x1b[3m"
	}
	if s.Underline {
		out += "\x1b[4m"
	}
	return out
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

func (s Style) Fg(c Color) Style {
	s.Foreground = c
	return s
}

func (s Style) Bg(c Color) Style {
	s.Background = c
	return s
}

func Width(w int) string {
	return fmt.Sprintf("\x1b[%dG", w)
}
