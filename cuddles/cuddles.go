// Package cuddles provides a semantic styling engine for MOFU.
//
// Cuddles replaces visual styling with semantic styling.
// Instead of specifying colors directly, you specify meaning:
//
//	cuddles.Primary    → theme decides the color
//	cuddles.Error      → theme decides the color
//	cuddles.Success    → theme decides the color
//
// The theme maps semantics to visuals, supporting:
//   - Runtime theme switching
//   - Density scaling
//   - Accessibility modes
//   - Animation-aware styling
package cuddles

import (
	"github.com/xanstomper/mofu"
)

// ---------------------------------------------------------------------------
// Semantic Tokens
// ---------------------------------------------------------------------------

// Semantic represents a semantic color intent.
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

// Token is a semantic color token.
type Token struct {
	Name     string
	Semantic Semantic
	Fallback mofu.Color
}

var (
	Primary   = Token{Name: "primary", Semantic: SemanticPrimary, Fallback: mofu.Hex("89b4fa")}
	Secondary = Token{Name: "secondary", Semantic: SemanticSecondary, Fallback: mofu.Hex("94e2d5")}
	Accent    = Token{Name: "accent", Semantic: SemanticAccent, Fallback: mofu.Hex("f5c2e7")}
	Success   = Token{Name: "success", Semantic: SemanticSuccess, Fallback: mofu.Hex("a6e3a1")}
	Warning   = Token{Name: "warning", Semantic: SemanticWarning, Fallback: mofu.Hex("f9e2af")}
	Error     = Token{Name: "error", Semantic: SemanticError, Fallback: mofu.Hex("f38ba8")}
	Info      = Token{Name: "info", Semantic: SemanticInfo, Fallback: mofu.Hex("7dcfff")}
	Muted     = Token{Name: "muted", Semantic: SemanticMuted, Fallback: mofu.Hex("6c7086")}
	Text      = Token{Name: "text", Semantic: SemanticText, Fallback: mofu.Hex("cdd6f4")}
	TextDim   = Token{Name: "textDim", Semantic: SemanticTextDim, Fallback: mofu.Hex("6c7086")}
	Bg        = Token{Name: "background", Semantic: SemanticBackground, Fallback: mofu.Hex("1e1e2e")}
	Surface   = Token{Name: "surface", Semantic: SemanticSurface, Fallback: mofu.Hex("313244")}
	Border    = Token{Name: "border", Semantic: SemanticBorder, Fallback: mofu.Hex("45475a")}
)

// ---------------------------------------------------------------------------
// Theme Engine
// ---------------------------------------------------------------------------

// Theme defines a complete semantic styling system.
type Theme struct {
	Name    string
	Colors  map[Semantic]mofu.Color
	Density Density
	Motion  MotionProfile
}

// Density controls spacing and sizing.
type Density int

const (
	DensityCompact Density = iota
	DensityNormal
	DensityComfortable
)

// SpacingUnit returns the base spacing unit for the density.
func (d Density) SpacingUnit() int {
	return [...]int{1, 2, 4}[d]
}

// MotionProfile defines animation characteristics.
type MotionProfile struct {
	Speed     float64 // 0.5 = slow, 1.0 = normal, 2.0 = fast
	Elasticity float64 // 0.0 = rigid, 1.0 = normal, 2.0 = bouncy
	Duration  float64 // base duration in ms
}

// DefaultMotion returns the default motion profile.
func DefaultMotion() MotionProfile {
	return MotionProfile{Speed: 1.0, Elasticity: 1.0, Duration: 300}
}

// Resolve returns the color for a semantic token.
func (t *Theme) Resolve(token Token) mofu.Color {
	if t.Colors != nil {
		if c, ok := t.Colors[token.Semantic]; ok {
			return c
		}
	}
	return token.Fallback
}

// Style returns a mofu.Style with the semantic color applied.
func (t *Theme) Style(token Token) mofu.Style {
	return mofu.DefaultStyle().Fg(t.Resolve(token))
}

// StyleBg returns a style with background color.
func (t *Theme) StyleBg(token Token) mofu.Style {
	return mofu.DefaultStyle().Bg(t.Resolve(token))
}

// ---------------------------------------------------------------------------
// Built-in Themes
// ---------------------------------------------------------------------------

// Mochi returns the Mochi theme.
func Mochi() *Theme {
	return &Theme{
		Name: "mochi",
		Colors: map[Semantic]mofu.Color{
			SemanticPrimary:   mofu.Hex("ff69b4"),
			SemanticSecondary: mofu.Hex("9b59b6"),
			SemanticAccent:    mofu.Hex("ff1493"),
			SemanticSuccess:   mofu.Hex("00ff88"),
			SemanticWarning:   mofu.Hex("ffaa00"),
			SemanticError:     mofu.Hex("ff3355"),
			SemanticInfo:      mofu.Hex("3399ff"),
			SemanticMuted:     mofu.Hex("333333"),
			SemanticText:      mofu.Hex("e0e0e0"),
			SemanticTextDim:   mofu.Hex("666666"),
			SemanticBackground: mofu.Hex("0a0a0a"),
			SemanticSurface:   mofu.Hex("1a1a2e"),
			SemanticBorder:    mofu.Hex("2a2a2a"),
		},
		Density: DensityNormal,
		Motion:  DefaultMotion(),
	}
}

// Catppuccin returns the Catppuccin Mocha theme.
func Catppuccin() *Theme {
	return &Theme{
		Name: "catppuccin",
		Colors: map[Semantic]mofu.Color{
			SemanticPrimary:   mofu.Hex("89b4fa"),
			SemanticSecondary: mofu.Hex("94e2d5"),
			SemanticAccent:    mofu.Hex("f5c2e7"),
			SemanticSuccess:   mofu.Hex("a6e3a1"),
			SemanticWarning:   mofu.Hex("f9e2af"),
			SemanticError:     mofu.Hex("f38ba8"),
			SemanticInfo:      mofu.Hex("7dcfff"),
			SemanticMuted:     mofu.Hex("6c7086"),
			SemanticText:      mofu.Hex("cdd6f4"),
			SemanticTextDim:   mofu.Hex("6c7086"),
			SemanticBackground: mofu.Hex("1e1e2e"),
			SemanticSurface:   mofu.Hex("313244"),
			SemanticBorder:    mofu.Hex("45475a"),
		},
		Density: DensityNormal,
		Motion:  DefaultMotion(),
	}
}

// TokyoNight returns the Tokyo Night theme.
func TokyoNight() *Theme {
	return &Theme{
		Name: "tokyonight",
		Colors: map[Semantic]mofu.Color{
			SemanticPrimary:   mofu.Hex("7aa2f7"),
			SemanticSecondary: mofu.Hex("bb9af7"),
			SemanticAccent:    mofu.Hex("ff9e64"),
			SemanticSuccess:   mofu.Hex("9ece6a"),
			SemanticWarning:   mofu.Hex("e0af68"),
			SemanticError:     mofu.Hex("f7768e"),
			SemanticInfo:      mofu.Hex("7dcfff"),
			SemanticMuted:     mofu.Hex("565f89"),
			SemanticText:      mofu.Hex("c0caf5"),
			SemanticTextDim:   mofu.Hex("565f89"),
			SemanticBackground: mofu.Hex("1a1b26"),
			SemanticSurface:   mofu.Hex("24283b"),
			SemanticBorder:    mofu.Hex("3b4261"),
		},
		Density: DensityNormal,
		Motion:  DefaultMotion(),
	}
}

// ---------------------------------------------------------------------------
// Theme Manager
// ---------------------------------------------------------------------------

// Manager handles theme switching and notifications.
type Manager struct {
	current  *Theme
	themes   map[string]*Theme
	onChange []func(old, new *Theme)
}

// NewManager creates a theme manager with an initial theme.
func NewManager(initial *Theme) *Manager {
	return &Manager{
		current: initial,
		themes:  map[string]*Theme{initial.Name: initial},
	}
}

// Current returns the active theme.
func (m *Manager) Current() *Theme {
	return m.current
}

// Register adds a theme.
func (m *Manager) Register(theme *Theme) {
	m.themes[theme.Name] = theme
}

// Apply switches to the named theme.
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

// OnChange registers a theme change listener.
func (m *Manager) OnChange(fn func(old, new *Theme)) {
	m.onChange = append(m.onChange, fn)
}

// Names returns all registered theme names.
func (m *Manager) Names() []string {
	names := make([]string, 0, len(m.themes))
	for n := range m.themes {
		names = append(names, n)
	}
	return names
}
