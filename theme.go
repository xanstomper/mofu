package mofu

type Typography struct {
	Title    Style
	Subtitle Style
	Body     Style
	Label    Style
	Mono     Style
}

type SpacingScale struct {
	X0, X1, X2, X4, X8, X12, X16 int
}

type Theme struct {
	Name       string
	Colors     ThemeColors
	Typography Typography
	Spacing    SpacingScale
	Border     BorderStyle
	Radius     int
}

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
		Spacing: SpacingScale{X0: 0, X1: 1, X2: 2, X4: 4, X8: 8, X12: 12, X16: 16},
		Border:  BorderRounded,
		Radius:  2,
		Typography: Typography{
			Title:    DefaultStyle().Fg(Hex("cdd6f4")),
			Subtitle: DefaultStyle().Fg(Hex("6c7086")),
			Body:     DefaultStyle().Fg(Hex("cdd6f4")),
			Label:    DefaultStyle().Fg(Hex("89b4fa")),
			Mono:     DefaultStyle().Fg(Hex("cdd6f4")),
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
		Border:     Hex("2a2a2a"),
	}
	t.Border = BorderThick
	t.Radius = 0
	t.Typography = Typography{
		Title:    DefaultStyle().Fg(Hex("ff69b4")),
		Subtitle: DefaultStyle().Fg(Hex("9b59b6")),
		Body:     DefaultStyle().Fg(Hex("e0e0e0")),
		Label:    DefaultStyle().Fg(Hex("ff69b4")),
		Mono:     DefaultStyle().Fg(Hex("e0e0e0")),
	}
	return t
}

func CatppuccinMocha() *Theme {
	return DefaultTheme()
}
