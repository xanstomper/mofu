package mofu

import "fmt"

// BorderStyle defines the characters used to draw borders.
type BorderStyle struct {
	Top, Bottom, Left, Right                   rune
	TopLeft, TopRight, BottomLeft, BottomRight rune
}

var (
	// BorderNone is no border.
	BorderNone = BorderStyle{}
	// BorderHidden is a hidden border (same as none).
	BorderHidden = BorderStyle{}
	// BorderNormal is a standard box border.
	BorderNormal = BorderStyle{
		Top: '─', Bottom: '─', Left: '│', Right: '│',
		TopLeft: '┌', TopRight: '┐', BottomLeft: '└', BottomRight: '┘',
	}
	// BorderRounded is a rounded corner border.
	BorderRounded = BorderStyle{
		Top: '─', Bottom: '─', Left: '│', Right: '│',
		TopLeft: '╭', TopRight: '╮', BottomLeft: '╰', BottomRight: '╯',
	}
	// BorderThick is a thick border.
	BorderThick = BorderStyle{
		Top: '━', Bottom: '━', Left: '┃', Right: '┃',
		TopLeft: '┏', TopRight: '┓', BottomLeft: '┗', BottomRight: '┛',
	}
	// BorderDouble is a double-line border.
	BorderDouble = BorderStyle{
		Top: '═', Bottom: '═', Left: '║', Right: '║',
		TopLeft: '╔', TopRight: '╗', BottomLeft: '╚', BottomRight: '╝',
	}
)

// Spacing defines padding or margin on all four sides.
type Spacing struct{ Top, Right, Bottom, Left int }

// Align controls cross-axis alignment.
type Align int

const (
	// AlignLeft aligns content to the left.
	AlignLeft Align = 0
	// AlignCenter centers content.
	AlignCenter Align = 1
	// AlignRight aligns content to the right.
	AlignRight Align = 2
	// AlignStretch stretches content to fill.
	AlignStretch Align = 3
)

// Justify controls main-axis alignment.
type Justify int

const (
	// JustifyStart aligns to the start.
	JustifyStart Justify = 0
	// JustifyCenter centers content.
	JustifyCenter Justify = 1
	// JustifyEnd aligns to the end.
	JustifyEnd Justify = 2
	// JustifySpaceBetween spaces items evenly.
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
	// Emit foreground if it's not the zero value (transparent/unset)
	if s.Foreground != (Color{}) {
		out += s.Foreground.foreground()
	}
	// Emit background if it's not the zero value (transparent/unset)
	if s.Background != (Color{}) {
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

// ---------------------------------------------------------------------------
// Spacing Tokens — proportional spacing scale
// ---------------------------------------------------------------------------

// SpacingToken defines a spacing value by semantic name.
type SpacingToken int

const (
	SpacingNone SpacingToken = iota
	SpacingXXS
	SpacingXS
	SpacingS
	SpacingM
	SpacingL
	SpacingXL
	SpacingXXL
)

// SpacingValue returns the cell count for a spacing token.
func (t SpacingToken) Value() int {
	return [...]int{0, 1, 1, 2, 4, 8, 12, 16}[t]
}

// Spacing returns a Spacing with all sides set to the token value.
func SpacingTokenAll(t SpacingToken) Spacing {
	v := t.Value()
	return Spacing{Top: v, Right: v, Bottom: v, Left: v}
}

// ---------------------------------------------------------------------------
// Semantic Styling — colors by meaning, not appearance
// ---------------------------------------------------------------------------

// SemanticColor represents a color with semantic meaning.
type SemanticColor int

const (
	SemanticNone SemanticColor = iota
	SemanticSuccess
	SemanticWarning
	SemanticError
	SemanticInfo
	SemanticPrimary
	SemanticSecondary
	SemanticMuted
	SemanticAccent
)

// SemanticFg returns a Style with the foreground set to the semantic color
// from the given theme. Falls back to a default if theme is nil.
func SemanticFg(sc SemanticColor, theme *Theme) Style {
	s := DefaultStyle()
	if theme == nil {
		switch sc {
		case SemanticSuccess:
			s.Foreground = Hex("a6e3a1")
		case SemanticWarning:
			s.Foreground = Hex("f9e2af")
		case SemanticError:
			s.Foreground = Hex("f38ba8")
		case SemanticInfo:
			s.Foreground = Hex("7dcfff")
		case SemanticPrimary:
			s.Foreground = Hex("89b4fa")
		case SemanticSecondary:
			s.Foreground = Hex("94e2d5")
		case SemanticMuted:
			s.Foreground = Hex("6c7086")
		case SemanticAccent:
			s.Foreground = Hex("f5c2e7")
		}
		return s
	}
	switch sc {
	case SemanticSuccess:
		s.Foreground = theme.Colors.Success
	case SemanticWarning:
		s.Foreground = theme.Colors.Warning
	case SemanticError:
		s.Foreground = theme.Colors.Error
	case SemanticInfo:
		s.Foreground = theme.Colors.Info
	case SemanticPrimary:
		s.Foreground = theme.Colors.Primary
	case SemanticSecondary:
		s.Foreground = theme.Colors.Secondary
	case SemanticMuted:
		s.Foreground = theme.Colors.Muted
	case SemanticAccent:
		s.Foreground = theme.Colors.Accent
	}
	return s
}

// SemanticBg returns a Style with the background set to the semantic color.
func SemanticBg(sc SemanticColor, theme *Theme) Style {
	s := SemanticFg(sc, theme)
	s.Background = s.Foreground
	s.Foreground = theme.Colors.Background
	return s
}
