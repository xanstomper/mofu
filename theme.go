package mofu

// ThemeColors defines a consistent color palette for a theme.
type ThemeColors struct {
	Background Color
	Surface    Color
	Text       Color
	TextDim    Color
	Primary    Color
	Secondary  Color
	Muted      Color
	Accent     Color
	Success    Color
	Warning    Color
	Error      Color
	Border     Color
}

// Theme defines the visual theme for a Mofu application.
type Theme struct {
	Name   string
	Colors ThemeColors
}

// DefaultTheme returns a Catppuccin-inspired dark theme.
func DefaultTheme() *Theme {
	return &Theme{
		Name: "default",
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
			Border:     Hex("45475a"),
		},
	}
}

// MochiTheme is inspired by Mochi's pink/black aesthetic.
func MochiTheme() *Theme {
	return &Theme{
		Name: "mochi",
		Colors: ThemeColors{
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
			Border:     Hex("2a2a2a"),
		},
	}
}

// CatppuccinMocha returns the full Catppuccin Mocha palette.
func CatppuccinMocha() *Theme {
	return &Theme{
		Name: "catppuccin-mocha",
		Colors: ThemeColors{
			Background: Hex("1e1e2e"),
			Surface:    Hex("313244"),
			Text:       Hex("cdd6f4"),
			TextDim:    Hex("6c7086"),
			Primary:    Hex("89b4fa"),
			Secondary:  Hex("a6e3a1"),
			Muted:      Hex("585b70"),
			Accent:     Hex("f5c2e7"),
			Success:    Hex("a6e3a1"),
			Warning:    Hex("f9e2af"),
			Error:      Hex("f38ba8"),
			Border:     Hex("45475a"),
		},
	}
}
