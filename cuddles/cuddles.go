package cuddles

import "github.com/xanstomper/mofu"

type Semantic int

const (
	SemanticNone Semantic = iota
	SemanticPrimary
	SemanticSecondary
	SemanticAccent
	SemanticSuccess
	SemanticWarning
	SemanticError
	SemanticInfo
	SemanticMuted
	SemanticText
	SemanticTextDim
	SemanticBackground
	SemanticSurface
	SemanticBorder
)

type Token struct {
	Name     string
	Semantic Semantic
	Fallback mofu.Color
}

var (
	Primary   = Token{"primary", SemanticPrimary, mofu.Hex("89b4fa")}
	Secondary = Token{"secondary", SemanticSecondary, mofu.Hex("94e2d5")}
	Accent    = Token{"accent", SemanticAccent, mofu.Hex("f5c2e7")}
	Success   = Token{"success", SemanticSuccess, mofu.Hex("a6e3a1")}
	Warning   = Token{"warning", SemanticWarning, mofu.Hex("f9e2af")}
	Error     = Token{"error", SemanticError, mofu.Hex("f38ba8")}
	Info      = Token{"info", SemanticInfo, mofu.Hex("7dcfff")}
	Muted     = Token{"muted", SemanticMuted, mofu.Hex("6c7086")}
	Text      = Token{"text", SemanticText, mofu.Hex("cdd6f4")}
	TextDim   = Token{"textDim", SemanticTextDim, mofu.Hex("6c7086")}
	Bg        = Token{"background", SemanticBackground, mofu.Hex("1e1e2e")}
	Surface   = Token{"surface", SemanticSurface, mofu.Hex("313244")}
	Border    = Token{"border", SemanticBorder, mofu.Hex("45475a")}
)

type Theme struct {
	Name    string
	Colors  map[Semantic]mofu.Color
	Density Density
	Motion  MotionProfile
}

type Density int

const (
	DensityCompact Density = iota
	DensityNormal
	DensityComfortable
)

func (d Density) SpacingUnit() int {
	return [...]int{1, 2, 4}[d]
}

type MotionProfile struct {
	Speed      float64
	Elasticity float64
	Duration   float64
}

func DefaultMotion() MotionProfile {
	return MotionProfile{Speed: 1.0, Elasticity: 1.0, Duration: 300}
}

func (t *Theme) Resolve(token Token) mofu.Color {
	if t.Colors != nil {
		if c, ok := t.Colors[token.Semantic]; ok {
			return c
		}
	}
	return token.Fallback
}

func (t *Theme) Style(token Token) mofu.Style {
	return mofu.DefaultStyle().Fg(t.Resolve(token))
}

func (t *Theme) StyleBg(token Token) mofu.Style {
	return mofu.DefaultStyle().Bg(t.Resolve(token))
}

func Mochi() *Theme {
	return &Theme{
		Name: "mochi",
		Colors: map[Semantic]mofu.Color{
			SemanticPrimary:    mofu.Hex("ff69b4"),
			SemanticSecondary:  mofu.Hex("9b59b6"),
			SemanticAccent:     mofu.Hex("ff1493"),
			SemanticSuccess:    mofu.Hex("00ff88"),
			SemanticWarning:    mofu.Hex("ffaa00"),
			SemanticError:      mofu.Hex("ff3355"),
			SemanticInfo:       mofu.Hex("3399ff"),
			SemanticMuted:      mofu.Hex("333333"),
			SemanticText:       mofu.Hex("e0e0e0"),
			SemanticTextDim:    mofu.Hex("666666"),
			SemanticBackground: mofu.Hex("0a0a0a"),
			SemanticSurface:    mofu.Hex("1a1a2e"),
			SemanticBorder:     mofu.Hex("2a2a2a"),
		},
		Density: DensityNormal,
		Motion:  DefaultMotion(),
	}
}

func Catppuccin() *Theme {
	return &Theme{
		Name: "catppuccin",
		Colors: map[Semantic]mofu.Color{
			SemanticPrimary:    mofu.Hex("89b4fa"),
			SemanticSecondary:  mofu.Hex("94e2d5"),
			SemanticAccent:     mofu.Hex("f5c2e7"),
			SemanticSuccess:    mofu.Hex("a6e3a1"),
			SemanticWarning:    mofu.Hex("f9e2af"),
			SemanticError:      mofu.Hex("f38ba8"),
			SemanticInfo:       mofu.Hex("7dcfff"),
			SemanticMuted:      mofu.Hex("6c7086"),
			SemanticText:       mofu.Hex("cdd6f4"),
			SemanticTextDim:    mofu.Hex("6c7086"),
			SemanticBackground: mofu.Hex("1e1e2e"),
			SemanticSurface:    mofu.Hex("313244"),
			SemanticBorder:     mofu.Hex("45475a"),
		},
		Density: DensityNormal,
		Motion:  DefaultMotion(),
	}
}

func TokyoNight() *Theme {
	return &Theme{
		Name: "tokyonight",
		Colors: map[Semantic]mofu.Color{
			SemanticPrimary:    mofu.Hex("7aa2f7"),
			SemanticSecondary:  mofu.Hex("bb9af7"),
			SemanticAccent:     mofu.Hex("ff9e64"),
			SemanticSuccess:    mofu.Hex("9ece6a"),
			SemanticWarning:    mofu.Hex("e0af68"),
			SemanticError:      mofu.Hex("f7768e"),
			SemanticInfo:       mofu.Hex("7dcfff"),
			SemanticMuted:      mofu.Hex("565f89"),
			SemanticText:       mofu.Hex("c0caf5"),
			SemanticTextDim:    mofu.Hex("565f89"),
			SemanticBackground: mofu.Hex("1a1b26"),
			SemanticSurface:    mofu.Hex("24283b"),
			SemanticBorder:     mofu.Hex("3b4261"),
		},
		Density: DensityNormal,
		Motion:  DefaultMotion(),
	}
}

type Manager struct {
	current  *Theme
	themes   map[string]*Theme
	onChange []func(old, new *Theme)
}

func NewManager(initial *Theme) *Manager {
	return &Manager{
		current: initial,
		themes:  map[string]*Theme{initial.Name: initial},
	}
}

func (m *Manager) Current() *Theme  { return m.current }
func (m *Manager) Register(theme *Theme) { m.themes[theme.Name] = theme }

func (m *Manager) Apply(name string) bool {
	theme, ok := m.themes[name]
	if !ok {
		return false
	}
	old := m.current
	m.current = theme
	for _, fn := range m.onChange {
		fn(old, theme)
	}
	return true
}

func (m *Manager) OnChange(fn func(old, new *Theme)) {
	m.onChange = append(m.onChange, fn)
}

func (m *Manager) Names() []string {
	names := make([]string, 0, len(m.themes))
	for n := range m.themes {
		names = append(names, n)
	}
	return names
}

type StyleBuilder struct {
	style mofu.Style
}

func NewStyle() *StyleBuilder {
	return &StyleBuilder{style: mofu.DefaultStyle()}
}

func (b *StyleBuilder) Fg(c mofu.Color) *StyleBuilder    { b.style.Foreground = c; return b }
func (b *StyleBuilder) Bg(c mofu.Color) *StyleBuilder    { b.style.Background = c; return b }
func (b *StyleBuilder) Bold() *StyleBuilder               { b.style.Attrs |= mofu.AttrBold; return b }
func (b *StyleBuilder) Italic() *StyleBuilder             { b.style.Attrs |= mofu.AttrItalic; return b }
func (b *StyleBuilder) Underline() *StyleBuilder          { b.style.Attrs |= mofu.AttrUnderline; return b }
func (b *StyleBuilder) Dim() *StyleBuilder                { b.style.Attrs |= mofu.AttrDim; return b }
func (b *StyleBuilder) Reverse() *StyleBuilder            { b.style.Attrs |= mofu.AttrReverse; return b }
func (b *StyleBuilder) Strikethrough() *StyleBuilder      { b.style.Attrs |= mofu.AttrStrikethrough; return b }
func (b *StyleBuilder) Width(w int) *StyleBuilder         { b.style.Width = w; return b }
func (b *StyleBuilder) Height(h int) *StyleBuilder        { b.style.Height = h; return b }
func (b *StyleBuilder) Align(a mofu.Align) *StyleBuilder  { b.style.Align = a; return b }
func (b *StyleBuilder) Border(bs mofu.BorderStyle) *StyleBuilder { b.style.Border = bs; return b }
func (b *StyleBuilder) Padding(v int) *StyleBuilder {
	b.style.Padding = mofu.Spacing{Top: v, Right: v, Bottom: v, Left: v}
	return b
}
func (b *StyleBuilder) Margin(v int) *StyleBuilder {
	b.style.Margin = mofu.Spacing{Top: v, Right: v, Bottom: v, Left: v}
	return b
}
func (b *StyleBuilder) Gap(g int) *StyleBuilder           { b.style.Gap = g; return b }
func (b *StyleBuilder) Grow(g float64) *StyleBuilder      { b.style.Grow = g; return b }
func (b *StyleBuilder) Build() mofu.Style                 { return b.style }

func PrimaryStyle(t *Theme) mofu.Style   { return t.Style(Primary) }
func SecondaryStyle(t *Theme) mofu.Style { return t.Style(Secondary) }
func SuccessStyle(t *Theme) mofu.Style   { return t.Style(Success) }
func WarningStyle(t *Theme) mofu.Style   { return t.Style(Warning) }
func ErrorStyle(t *Theme) mofu.Style     { return t.Style(Error) }
func InfoStyle(t *Theme) mofu.Style      { return t.Style(Info) }
func MutedStyle(t *Theme) mofu.Style     { return t.Style(Muted) }
func TextStyle(t *Theme) mofu.Style      { return t.Style(Text) }
func TextDimStyle(t *Theme) mofu.Style   { return t.Style(TextDim) }
func BgStyle(t *Theme) mofu.Style        { return t.StyleBg(Bg) }
func SurfaceStyle(t *Theme) mofu.Style   { return t.StyleBg(Surface) }
func BorderStyleColor(t *Theme) mofu.Style { return t.Style(Border) }
