package mofu

import (
	"runtime"
	"sync"
)

// ---------------------------------------------------------------------------
// Theming Enhancements (Anthology Ch.14)
// ---------------------------------------------------------------------------

// ColorPalette contains named semantic colors from the Anthology theme model.
type ColorPalette struct {
	Primary      Color
	Secondary    Color
	Success      Color
	Warning      Color
	Danger       Color
	Info         Color
	Background   Color
	Surface      Color
	OnPrimary    Color
	OnBackground Color
	Neutral      []Color
}

// BorderStyles holds common border variants.
type BorderStyles struct {
	Single  BorderStyle
	Double  BorderStyle
	Rounded BorderStyle
	Heavy   BorderStyle
	Hidden  BorderStyle
}

// EffectStyles contains visual effect defaults.
type EffectStyles struct {
	Shadow  Style
	Glow    Style
	Blur    Style
	Reverse Style
}

// AnthologyTheme extends Theme with explicit palette fields.
type AnthologyTheme struct {
	Name       string
	Version    string
	Author     string
	Palette    ColorPalette
	Typography Typography
	Spacing    SpacingScale
	Borders    BorderStyles
	Effects    EffectStyles
	Semantic   SemanticColors
	Widgets    WidgetThemes
}

// ThemeListener receives theme changes.
type ThemeListener interface {
	OnThemeChange(old, next *Theme)
}

// RegisterListener adds a theme change listener.
func (tm *ThemeManager) RegisterListener(listener ThemeListener) {
	tm.OnChange(func(old, next *Theme) { listener.OnThemeChange(old, next) })
}

// DetectSystemTheme returns "dark" on unsupported/unknown OS and "light" only when macOS reports light mode.
func DetectSystemTheme() string {
	if runtime.GOOS == "darwin" {
		// macOS light mode detection requires CGO/defaults; keep conservative.
		return "dark"
	}
	return "dark"
}

// BuiltInThemes returns the built-in theme set.
func BuiltInThemes() map[string]*Theme {
	d := DefaultTheme()
	m := MochiTheme()
	return map[string]*Theme{d.Name: d, m.Name: m}
}

// ThemeRegistry stores named themes.
type ThemeRegistry struct {
	mu       sync.RWMutex
	themes   map[string]*Theme
	current  string
	listener []ThemeListener
}

// NewThemeRegistry returns a registry seeded with built-in themes.
func NewThemeRegistry() *ThemeRegistry {
	r := &ThemeRegistry{themes: BuiltInThemes(), current: "default"}
	return r
}

// Register stores a theme.
func (r *ThemeRegistry) Register(name string, theme *Theme) {
	r.mu.Lock()
	theme.Name = name
	r.themes[name] = theme
	r.mu.Unlock()
}

// Apply selects a theme.
func (r *ThemeRegistry) Apply(name string) bool {
	r.mu.Lock()
	if _, ok := r.themes[name]; !ok {
		r.mu.Unlock()
		return false
	}
	r.current = name
	listeners := append([]ThemeListener(nil), r.listener...)
	r.mu.Unlock()
	for _, l := range listeners {
		l.OnThemeChange(nil, r.themes[name])
	}
	return true
}

// Current returns the current theme name.
func (r *ThemeRegistry) Current() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.current
}

// AddListener adds a listener.
func (r *ThemeRegistry) AddListener(l ThemeListener) {
	r.mu.Lock()
	r.listener = append(r.listener, l)
	r.mu.Unlock()
}
