package mofu

import (
	"testing"
)

func TestStyleSGR(t *testing.T) {
	tests := []struct {
		name   string
		style  Style
		expect string
	}{
		{
			name:   "zero style",
			style:  Style{},
			expect: "",
		},
		{
			name:   "foreground only",
			style:  Style{}.Fg(RGB(255, 0, 0)),
			expect: "\x1b[38;2;255;0;0m",
		},
		{
			name:   "background only",
			style:  Style{}.Bg(RGB(0, 255, 0)),
			expect: "\x1b[48;2;0;255;0m",
		},
		{
			name:   "both colors",
			style:  Style{}.Fg(RGB(255, 0, 0)).Bg(RGB(0, 0, 255)),
			expect: "\x1b[38;2;255;0;0m\x1b[48;2;0;0;255m",
		},
		{
			name:   "bold attribute",
			style:  Style{}.WithAttrs(AttrBold),
			expect: "\x1b[1m",
		},
		{
			name:   "italic attribute",
			style:  Style{}.WithAttrs(AttrItalic),
			expect: "\x1b[3m",
		},
		{
			name:   "underline attribute",
			style:  Style{}.WithAttrs(AttrUnderline),
			expect: "\x1b[4m",
		},
		{
			name:   "ANSI foreground",
			style:  Style{}.Fg(ANSI(1)),
			expect: "\x1b[38;5;1m",
		},
		{
			name:   "ANSI background",
			style:  Style{}.Bg(ANSI(4)),
			expect: "\x1b[48;5;4m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.style.SGR()
			if got != tt.expect {
				t.Errorf("SGR() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestResetAttrs(t *testing.T) {
	tests := []struct {
		name   string
		attrs  AttrMask
		expect string
	}{
		{"bold", AttrBold, "\x1b[22m"},
		{"italic", AttrItalic, "\x1b[23m"},
		{"underline", AttrUnderline, "\x1b[24m"},
		{"strikethrough", AttrStrikethrough, "\x1b[29m"},
		{"none", 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResetAttrs(tt.attrs)
			if got != tt.expect {
				t.Errorf("ResetAttrs(%v) = %q, want %q", tt.attrs, got, tt.expect)
			}
		})
	}
}

func TestSpacingTokens(t *testing.T) {
	tests := []struct {
		token  SpacingToken
		expect int
	}{
		{SpacingNone, 0},
		{SpacingXXS, 1},
		{SpacingXS, 1},
		{SpacingS, 2},
		{SpacingM, 4},
		{SpacingL, 8},
		{SpacingXL, 12},
		{SpacingXXL, 16},
	}

	for _, tt := range tests {
		if got := tt.token.Value(); got != tt.expect {
			t.Errorf("SpacingToken(%d).Value() = %d, want %d", tt.token, got, tt.expect)
		}
	}
}

func TestColorHex(t *testing.T) {
	tests := []struct {
		hex    string
		r, g, b uint8
	}{
		{"#ff0000", 255, 0, 0},
		{"00ff00", 0, 255, 0},
		{"#0000ff", 0, 0, 255},
		{"#ffffff", 255, 255, 255},
		{"000000", 0, 0, 0},
	}

	for _, tt := range tests {
		c := Hex(tt.hex)
		if c.R != tt.r || c.G != tt.g || c.B != tt.b {
			t.Errorf("Hex(%q) = RGB(%d,%d,%d), want RGB(%d,%d,%d)", tt.hex, c.R, c.G, c.B, tt.r, tt.g, tt.b)
		}
	}
}

func TestBorderStyle(t *testing.T) {
	if BorderNormal.TopLeft != '┌' {
		t.Errorf("BorderNormal.TopLeft = %c, want ┌", BorderNormal.TopLeft)
	}
	if BorderRounded.TopLeft != '╭' {
		t.Errorf("BorderRounded.TopLeft = %c, want ╭", BorderRounded.TopLeft)
	}
	if BorderThick.TopLeft != '┏' {
		t.Errorf("BorderThick.TopLeft = %c, want ┏", BorderThick.TopLeft)
	}
	if BorderDouble.TopLeft != '╔' {
		t.Errorf("BorderDouble.TopLeft = %c, want ╔", BorderDouble.TopLeft)
	}
}

func TestAttrMaskHas(t *testing.T) {
	var attrs AttrMask
	attrs |= AttrBold | AttrItalic

	if !attrs.Has(AttrBold) {
		t.Error("expected Has(AttrBold)")
	}
	if !attrs.Has(AttrItalic) {
		t.Error("expected Has(AttrItalic)")
	}
	if attrs.Has(AttrUnderline) {
		t.Error("expected !Has(AttrUnderline)")
	}
}

func TestRectContains(t *testing.T) {
	r := Rect{X: 10, Y: 10, Width: 20, Height: 10}

	tests := []struct {
		x, y   int
		expect bool
	}{
		{15, 15, true},
		{10, 10, true},
		{29, 19, true},
		{30, 19, false},
		{29, 20, false},
		{5, 5, false},
	}

	for _, tt := range tests {
		if got := r.Contains(tt.x, tt.y); got != tt.expect {
			t.Errorf("Rect{%d,%d,%d,%d}.Contains(%d,%d) = %v, want %v",
				r.X, r.Y, r.Width, r.Height, tt.x, tt.y, got, tt.expect)
		}
	}
}

func TestDefaultTheme(t *testing.T) {
	theme := DefaultTheme()
	if theme == nil {
		t.Fatal("DefaultTheme() returned nil")
	}
	if theme.Name != "default" {
		t.Errorf("theme.Name = %q, want %q", theme.Name, "default")
	}
	if theme.Colors.Primary == (Color{}) {
		t.Error("theme.Colors.Primary is zero value")
	}
}

func TestMochiTheme(t *testing.T) {
	theme := MochiTheme()
	if theme == nil {
		t.Fatal("MochiTheme() returned nil")
	}
	if theme.Name != "mochi" {
		t.Errorf("theme.Name = %q, want %q", theme.Name, "mochi")
	}
}

func TestThemeManager(t *testing.T) {
	dm := NewThemeManager(DefaultTheme())
	if dm.Current() == nil {
		t.Fatal("Current() returned nil")
	}

	mochi := MochiTheme()
	dm.Register("mochi", mochi)

	if !dm.Apply("mochi") {
		t.Fatal("Apply(mochi) failed")
	}
	if dm.Current().Name != "mochi" {
		t.Errorf("Current().Name = %q, want %q", dm.Current().Name, "mochi")
	}

	names := dm.Names()
	if len(names) < 2 {
		t.Errorf("Names() returned %d themes, want >= 2", len(names))
	}
}

func BenchmarkStyleSGR(b *testing.B) {
	style := DefaultStyle().Fg(RGB(255, 128, 0)).Bg(RGB(0, 0, 128)).WithAttrs(AttrBold | AttrItalic)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = style.SGR()
	}
}

func BenchmarkColorHex(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Hex("#ff69b4")
	}
}
