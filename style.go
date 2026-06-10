package mofu

import "fmt"

// BorderStyle defines the characters used for drawing borders.
type BorderStyle struct {
	Top         rune
	Bottom      rune
	Left        rune
	Right       rune
	TopLeft     rune
	TopRight    rune
	BottomLeft  rune
	BottomRight rune
}

// Predefined border styles
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

// Spacing represents margin or padding.
type Spacing struct {
	Top, Right, Bottom, Left int
}

// Style defines the visual appearance of a component.
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
	Align      Align
}

// Align represents text alignment.
type Align int

const (
	AlignLeft   Align = 0
	AlignCenter Align = 1
	AlignRight  Align = 2
)

// DefaultStyle returns a style with sensible defaults.
func DefaultStyle() Style {
	return Style{
		Foreground: ColorWhite,
		Background: ColorTransparent,
	}
}

// SGR returns the ANSI SGR escape sequence for this style.
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

// Reset returns the ANSI reset sequence.
func (s Style) Reset() string {
	return "\x1b[0m"
}

// Apply wraps the given text in style/reset sequences.
func (s Style) Apply(text string) string {
	if text == "" {
		return ""
	}
	return s.SGR() + text + s.Reset()
}

// Fg sets the foreground color.
func (s Style) Fg(c Color) Style {
	s.Foreground = c
	return s
}

// Bg sets the background color.
func (s Style) Bg(c Color) Style {
	s.Background = c
	return s
}

// Width returns a style string with a fixed width.
func Width(w int) string {
	return fmt.Sprintf("\x1b[%dG", w)
}
