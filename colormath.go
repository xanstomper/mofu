package mofu

import "math"

type HSV struct {
	H, S, V float64
}

func RGBToHSV(c Color) HSV {
	r := float64(c.R) / 255.0
	g := float64(c.G) / 255.0
	b := float64(c.B) / 255.0

	max := math.Max(r, math.Max(g, b))
	min := math.Min(r, math.Min(g, b))
	delta := max - min

	var h float64
	switch max {
	case r:
		h = 60.0 * math.Mod((g-b)/delta, 6.0)
	case g:
		h = 60.0 * ((b-r)/delta + 2.0)
	case b:
		h = 60.0 * ((r-g)/delta + 4.0)
	}
	if h < 0 {
		h += 360.0
	}

	s := 0.0
	if max != 0 {
		s = delta / max
	}

	return HSV{H: h, S: s, V: max}
}

func HSVToRGB(hsv HSV) Color {
	h := hsv.H
	s := hsv.S
	v := hsv.V

	c := v * s
	x := c * (1.0 - math.Abs(math.Mod(h/60.0, 2.0)-1.0))
	m := v - c

	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}

	return Color{
		R: uint8((r + m) * 255.0),
		G: uint8((g + m) * 255.0),
		B: uint8((b + m) * 255.0),
	}
}

func BlendColors(a, b Color, t float64) Color {
	t = clamp01(t)
	return Color{
		R: uint8(float64(a.R)*(1-t) + float64(b.R)*t),
		G: uint8(float64(a.G)*(1-t) + float64(b.G)*t),
		B: uint8(float64(a.B)*(1-t) + float64(b.B)*t),
	}
}

func Darken(c Color, amount float64) Color {
	amount = clamp01(amount)
	factor := 1.0 - amount
	return Color{
		R: uint8(float64(c.R) * factor),
		G: uint8(float64(c.G) * factor),
		B: uint8(float64(c.B) * factor),
	}
}

func Lighten(c Color, amount float64) Color {
	return BlendColors(c, ColorWhite, amount)
}

func Complementary(c Color) Color {
	hsv := RGBToHSV(c)
	hsv.H = math.Mod(hsv.H+180.0, 360.0)
	return HSVToRGB(hsv)
}

func Analogous(c Color, angle float64) (Color, Color) {
	hsv := RGBToHSV(c)
	h1 := math.Mod(hsv.H+angle, 360.0)
	h2 := math.Mod(hsv.H-angle+360.0, 360.0)
	return HSVToRGB(HSV{H: h1, S: hsv.S, V: hsv.V}), HSVToRGB(HSV{H: h2, S: hsv.S, V: hsv.V})
}

func RelativeLuminance(c Color) float64 {
	linearize := func(v uint8) float64 {
		srgb := float64(v) / 255.0
		if srgb <= 0.04045 {
			return srgb / 12.92
		}
		return math.Pow((srgb+0.055)/1.055, 2.4)
	}
	return 0.2126*linearize(c.R) + 0.7152*linearize(c.G) + 0.0722*linearize(c.B)
}

func ContrastRatio(a, b Color) float64 {
	l1 := RelativeLuminance(a)
	l2 := RelativeLuminance(b)
	if l1 < l2 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05)
}

func MeetsWCAG(fg, bg Color, level string) bool {
	ratio := ContrastRatio(fg, bg)
	switch level {
	case "AAA":
		return ratio >= 7.0
	case "AA":
		return ratio >= 4.5
	case "AALarge":
		return ratio >= 3.0
	default:
		return ratio >= 4.5
	}
}

func TextColorForBackground(bg Color) Color {
	if RelativeLuminance(bg) > 0.5 {
		return Color{R: 0, G: 0, B: 0}
	}
	return Color{R: 255, G: 255, B: 255}
}

func IsColorBlindSafe(colors []Color) bool {
	if len(colors) < 2 {
		return true
	}
	for i := 0; i < len(colors); i++ {
		for j := i + 1; j < len(colors); j++ {
			d := colorDistance(colors[i], colors[j])
			if d < 30.0 {
				deutDist := colorDistance(simulateDeuteranopia(colors[i]), simulateDeuteranopia(colors[j]))
				if deutDist < 20.0 {
					return false
				}
			}
		}
	}
	return true
}

func colorDistance(a, b Color) float64 {
	dr := float64(a.R) - float64(b.R)
	dg := float64(a.G) - float64(b.G)
	db := float64(a.B) - float64(b.B)
	return math.Sqrt(dr*dr + dg*dg + db*db)
}

func simulateDeuteranopia(c Color) Color {
	r := float64(c.R)
	g := float64(c.G)
	b := float64(c.B)
	return Color{
		R: uint8(clamp(r*0.625+g*0.375+b*0.0, 0, 255)),
		G: uint8(clamp(r*0.7+g*0.3+b*0.0, 0, 255)),
		B: uint8(clamp(r*0.0+g*0.3+b*0.7, 0, 255)),
	}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
