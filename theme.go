package mofu

import (
	"encoding/json"
	"os"
	"sync"
)

// ThemeManager manages theme registration, switching, and change notifications.
type ThemeManager struct {
	mu       sync.RWMutex
	current  *Theme
	themes   map[string]*Theme
	onChange []func(old, new *Theme)
}

// NewThemeManager creates a theme manager with an initial theme.
func NewThemeManager(initial *Theme) *ThemeManager {
	return &ThemeManager{
		current: initial,
		themes:  map[string]*Theme{initial.Name: initial},
	}
}

// Current returns the currently active theme.
func (tm *ThemeManager) Current() *Theme {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.current
}

// Register adds a theme with the given name.
func (tm *ThemeManager) Register(name string, theme *Theme) {
	theme.Name = name
	tm.mu.Lock()
	tm.themes[name] = theme
	tm.mu.Unlock()
}

// Apply switches to the named theme. Returns false if not found.
func (tm *ThemeManager) Apply(name string) bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	theme, ok := tm.themes[name]
	if !ok {
		return false
	}
	old := tm.current
	tm.current = theme
	for _, fn := range tm.onChange {
		fn(old, theme)
	}
	return true
}

func (tm *ThemeManager) OnChange(fn func(old, new *Theme)) {
	tm.mu.Lock()
	tm.onChange = append(tm.onChange, fn)
	tm.mu.Unlock()
}

func (tm *ThemeManager) Names() []string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	names := make([]string, 0, len(tm.themes))
	for n := range tm.themes {
		names = append(names, n)
	}
	return names
}

func (tm *ThemeManager) LoadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var theme Theme
	if err := json.Unmarshal(data, &theme); err != nil {
		return err
	}
	tm.Register(theme.Name, &theme)
	return nil
}

type Typography struct {
	Title, Subtitle, Body, Label, Mono Style
}

type SpacingScale struct {
	X0, X1, X2, X4, X8, X12, X16 int
	Scale                        []int
}

type WidgetTheme struct {
	Normal, Focused, Hover, Pressed, Disabled, Error Style
}

type WidgetThemes struct {
	Button    WidgetTheme
	Input     WidgetTheme
	List      WidgetTheme
	Scrollbar WidgetTheme
	Checkbox  WidgetTheme
	Radio     WidgetTheme
	Progress  WidgetTheme
}

type SemanticColors struct {
	TextPrimary   Color
	TextSecondary Color
	TextDisabled  Color
	BorderDefault Color
	BorderFocused Color
	BorderError   Color
	Shadow        Color
}

type ColorProfile struct {
	ANSI4Bit, ANSI256, TrueColor Color
}

type Theme struct {
	Name       string         `json:"name"`
	Version    string         `json:"version,omitempty"`
	Colors     ThemeColors    `json:"colors"`
	Semantic   SemanticColors `json:"semantic"`
	Typography Typography     `json:"typography"`
	Spacing    SpacingScale   `json:"spacing"`
	Border     BorderStyle    `json:"-"`
	Radius     int            `json:"radius"`
	Widgets    WidgetThemes   `json:"widgets"`
}

type ThemeColors struct {
	Background Color   `json:"background"`
	Surface    Color   `json:"surface"`
	Text       Color   `json:"text"`
	TextDim    Color   `json:"textDim"`
	Primary    Color   `json:"primary"`
	Secondary  Color   `json:"secondary"`
	Muted      Color   `json:"muted"`
	Accent     Color   `json:"accent"`
	Success    Color   `json:"success"`
	Warning    Color   `json:"warning"`
	Error      Color   `json:"error"`
	Info       Color   `json:"info"`
	Border     Color   `json:"border"`
	Neutral    []Color `json:"neutral,omitempty"`
}

func DefaultTheme() *Theme {
	return &Theme{
		Name:    "default",
		Version: "1.0.0",
		Colors: ThemeColors{
			Background: Hex("1e1e2e"),
			Surface:    Hex("313244"),
			Text:       Hex("cdd6f4"),
			TextDim:    Hex("6c7086"),
			Primary:    Hex("89b4fa"),
			Secondary:  Hex("94e2d5"),
			Muted:      Hex("585b70"),
			Accent:     Hex("f5c2e7"),
			Success:    Hex("a6e3a1"),
			Warning:    Hex("f9e2af"),
			Error:      Hex("f38ba8"),
			Info:       Hex("7dcfff"),
			Border:     Hex("45475a"),
			Neutral: []Color{
				Hex("45475a"), Hex("585b70"), Hex("6c7086"),
				Hex("7f849c"), Hex("9399b2"), Hex("a6adc8"),
				Hex("bac2de"), Hex("cdd6f4"),
			},
		},
		Semantic: SemanticColors{
			TextPrimary:   Hex("cdd6f4"),
			TextSecondary: Hex("6c7086"),
			TextDisabled:  Hex("45475a"),
			BorderDefault: Hex("45475a"),
			BorderFocused: Hex("89b4fa"),
			BorderError:   Hex("f38ba8"),
			Shadow:        Hex("11111b"),
		},
		Spacing: SpacingScale{
			X0: 0, X1: 1, X2: 2, X4: 4, X8: 8, X12: 12, X16: 16,
			Scale: []int{0, 1, 2, 3, 4, 6, 8, 12, 16, 24, 32},
		},
		Border: BorderRounded,
		Radius: 2,
		Typography: Typography{
			Title:    DefaultStyle().Fg(Hex("cdd6f4")).WithAttrs(AttrBold),
			Subtitle: DefaultStyle().Fg(Hex("6c7086")),
			Body:     DefaultStyle().Fg(Hex("cdd6f4")),
			Label:    DefaultStyle().Fg(Hex("89b4fa")),
			Mono:     DefaultStyle().Fg(Hex("cdd6f4")),
		},
		Widgets: WidgetThemes{
			Button: WidgetTheme{
				Normal:   DefaultStyle().Fg(Hex("cdd6f4")).Bg(Hex("89b4fa")),
				Hover:    DefaultStyle().Fg(Hex("1e1e2e")).Bg(Hex("a6e3a1")),
				Pressed:  DefaultStyle().Fg(Hex("1e1e2e")).Bg(Hex("5a87f7")),
				Disabled: DefaultStyle().Fg(Hex("585b70")).Bg(Hex("313244")),
			},
			Input: WidgetTheme{
				Normal:  DefaultStyle().Fg(Hex("cdd6f4")).Bg(Hex("313244")),
				Focused: DefaultStyle().Fg(Hex("cdd6f4")).Bg(Hex("1e1e2e")),
				Error:   DefaultStyle().Fg(Hex("f38ba8")).Bg(Hex("1e1e2e")),
			},
			Scrollbar: WidgetTheme{
				Normal: DefaultStyle().Bg(Hex("313244")),
				Hover:  DefaultStyle().Bg(Hex("585b70")),
			},
		},
	}
}

func MochiTheme() *Theme {
	t := DefaultTheme()
	t.Name = "mochi"
	t.Colors = ThemeColors{
		Background: Hex("0a0a0a"),
		Surface:    Hex("1a1a2e"),
		Text:       Hex("e0e0e0"),
		TextDim:    Hex("666666"),
		Primary:    Hex("ff69b4"),
		Secondary:  Hex("9b59b6"),
		Muted:      Hex("333333"),
		Accent:     Hex("ff1493"),
		Success:    Hex("00ff88"),
		Warning:    Hex("ffaa00"),
		Error:      Hex("ff3355"),
		Info:       Hex("3399ff"),
		Border:     Hex("2a2a2a"),
		Neutral:    []Color{Hex("1a1a2e"), Hex("2a2a3e"), Hex("3a3a4e")},
	}
	t.Semantic = SemanticColors{
		TextPrimary:   Hex("e0e0e0"),
		TextSecondary: Hex("666666"),
		TextDisabled:  Hex("333333"),
		BorderDefault: Hex("2a2a2a"),
		BorderFocused: Hex("ff69b4"),
		BorderError:   Hex("ff3355"),
		Shadow:        Hex("000000"),
	}
	t.Border = BorderThick
	t.Radius = 0
	t.Typography = Typography{
		Title:    DefaultStyle().Fg(Hex("ff69b4")).WithAttrs(AttrBold),
		Subtitle: DefaultStyle().Fg(Hex("9b59b6")),
		Body:     DefaultStyle().Fg(Hex("e0e0e0")),
		Label:    DefaultStyle().Fg(Hex("ff69b4")),
		Mono:     DefaultStyle().Fg(Hex("e0e0e0")),
	}
	t.Widgets = WidgetThemes{
		Button: WidgetTheme{
			Normal:   DefaultStyle().Fg(Hex("e0e0e0")).Bg(Hex("ff69b4")),
			Hover:    DefaultStyle().Fg(Hex("0a0a0a")).Bg(Hex("00ff88")),
			Pressed:  DefaultStyle().Fg(Hex("0a0a0a")).Bg(Hex("cc5599")),
			Disabled: DefaultStyle().Fg(Hex("666666")).Bg(Hex("1a1a2e")),
		},
		Input: WidgetTheme{
			Normal:  DefaultStyle().Fg(Hex("e0e0e0")).Bg(Hex("1a1a2e")),
			Focused: DefaultStyle().Fg(Hex("e0e0e0")).Bg(Hex("0a0a0a")),
			Error:   DefaultStyle().Fg(Hex("ff3355")).Bg(Hex("0a0a0a")),
		},
	}
	return t
}

func CatppuccinMocha() *Theme {
	return DefaultTheme()
}

func (s Style) WithAttrs(flags AttrMask) Style {
	s.Attrs |= flags
	s.cachedSGR = ""
	return s
}

func (s Style) WithBorder(bs BorderStyle) Style {
	s.Border = bs
	return s
}

func (s Style) PaddingAll(v int) Style {
	s.Padding = Spacing{Top: v, Right: v, Bottom: v, Left: v}
	return s
}

func (s Style) MarginAll(v int) Style {
	s.Margin = Spacing{Top: v, Right: v, Bottom: v, Left: v}
	return s
}

func (s Style) Bold() Style              { return s.WithAttrs(AttrBold) }
func (s Style) Italic() Style            { return s.WithAttrs(AttrItalic) }
func (s Style) Underline() Style         { return s.WithAttrs(AttrUnderline) }
func (s Style) Dim() Style               { return s.WithAttrs(AttrDim) }
func (s Style) Strikethrough() Style     { return s.WithAttrs(AttrStrikethrough) }
func (s Style) Reverse() Style           { return s.WithAttrs(AttrReverse) }
func (s Style) SetWidth(w int) Style     { s.Width = w; return s }
func (s Style) SetHeight(h int) Style    { s.Height = h; return s }
func (s Style) AlignLeft() Style         { s.Align = AlignLeft; return s }
func (s Style) AlignCenter() Style       { s.Align = AlignCenter; return s }
func (s Style) AlignRight() Style        { s.Align = AlignRight; return s }
func (s Style) PaddingHorizontal(v int) Style { s.Padding.Left = v; s.Padding.Right = v; return s }
func (s Style) PaddingVertical(v int) Style   { s.Padding.Top = v; s.Padding.Bottom = v; return s }
func (s Style) MarginHorizontal(v int) Style  { s.Margin.Left = v; s.Margin.Right = v; return s }
func (s Style) MarginVertical(v int) Style    { s.Margin.Top = v; s.Margin.Bottom = v; return s }
func (s Style) WithRoundedBorder() Style  { s.Border = BorderRounded; return s }
func (s Style) WithNormalBorder() Style   { s.Border = BorderNormal; return s }
func (s Style) WithThickBorder() Style    { s.Border = BorderThick; return s }
func (s Style) WithDoubleBorder() Style   { s.Border = BorderDouble; return s }
func (s Style) WithNoBorder() Style       { s.Border = BorderNone; return s }

func SakuraTheme() *Theme {
	t := DefaultTheme()
	t.Name = "sakura"
	t.Colors = ThemeColors{
		Background: Hex("1a1020"),
		Surface:    Hex("2a1a30"),
		Text:       Hex("f0d0e0"),
		TextDim:    Hex("7a6080"),
		Primary:    Hex("ffb7d5"),
		Secondary:  Hex("c4a0ff"),
		Muted:      Hex("3a2a40"),
		Accent:     Hex("ff69b4"),
		Success:    Hex("a0f0c0"),
		Warning:    Hex("ffd080"),
		Error:      Hex("ff6080"),
		Info:       Hex("a0d0ff"),
		Border:     Hex("3a2a40"),
		Neutral:    []Color{Hex("2a1a30"), Hex("3a2a40"), Hex("4a3a50"), Hex("6a5a70")},
	}
	t.Semantic = SemanticColors{
		TextPrimary:   Hex("f0d0e0"),
		TextSecondary: Hex("7a6080"),
		TextDisabled:  Hex("3a2a40"),
		BorderDefault: Hex("3a2a40"),
		BorderFocused: Hex("ffb7d5"),
		BorderError:   Hex("ff6080"),
		Shadow:        Hex("0a0510"),
	}
	t.Border = BorderRounded
	t.Radius = 2
	t.Typography = Typography{
		Title:    DefaultStyle().Fg(Hex("ffb7d5")).WithAttrs(AttrBold),
		Subtitle: DefaultStyle().Fg(Hex("c4a0ff")),
		Body:     DefaultStyle().Fg(Hex("f0d0e0")),
		Label:    DefaultStyle().Fg(Hex("ffb7d5")),
		Mono:     DefaultStyle().Fg(Hex("f0d0e0")),
	}
	t.Widgets = WidgetThemes{
		Button: WidgetTheme{
			Normal:   DefaultStyle().Fg(Hex("1a1020")).Bg(Hex("ffb7d5")),
			Hover:    DefaultStyle().Fg(Hex("1a1020")).Bg(Hex("a0f0c0")),
			Pressed:  DefaultStyle().Fg(Hex("1a1020")).Bg(Hex("c4a0ff")),
			Disabled: DefaultStyle().Fg(Hex("7a6080")).Bg(Hex("2a1a30")),
		},
		Input: WidgetTheme{
			Normal:  DefaultStyle().Fg(Hex("f0d0e0")).Bg(Hex("2a1a30")),
			Focused: DefaultStyle().Fg(Hex("f0d0e0")).Bg(Hex("1a1020")),
			Error:   DefaultStyle().Fg(Hex("ff6080")).Bg(Hex("1a1020")),
		},
	}
	return t
}
