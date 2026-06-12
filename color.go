package mofu

import "fmt"

// Color represents a terminal color.
type Color struct {
	R, G, B  uint8
	IsANSI   bool
	ANSICode uint8
}

// RGB creates a true-color Color.
func RGB(r, g, b uint8) Color {
	return Color{R: r, G: g, B: b}
}

// Hex creates a Color from a hex string (e.g. "#ff00ff" or "ff00ff").
func Hex(hex string) Color {
	if len(hex) > 0 && hex[0] == '#' {
		hex = hex[1:]
	}
	if len(hex) != 6 {
		return Color{}
	}
	r := hexToByte(hex[0:2])
	g := hexToByte(hex[2:4])
	b := hexToByte(hex[4:6])
	return RGB(r, g, b)
}

func hexToByte(s string) uint8 {
	var v uint8
	for _, c := range s {
		v *= 16
		switch {
		case c >= '0' && c <= '9':
			v += uint8(c - '0')
		case c >= 'a' && c <= 'f':
			v += uint8(c - 'a' + 10)
		case c >= 'A' && c <= 'F':
			v += uint8(c - 'A' + 10)
		}
	}
	return v
}

// ANSI creates an ANSI indexed color.
func ANSI(code uint8) Color {
	return Color{IsANSI: true, ANSICode: code}
}

// ANSI escape codes for SGR parameters
func (c Color) foreground() string {
	if c.IsANSI {
		return fmt.Sprintf("\x1b[38;5;%dm", c.ANSICode)
	}
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", c.R, c.G, c.B)
}

func (c Color) background() string {
	if c.IsANSI {
		return fmt.Sprintf("\x1b[48;5;%dm", c.ANSICode)
	}
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", c.R, c.G, c.B)
}

// Common ANSI color codes
var (
	ColorBlack        = ANSI(0)
	ColorRed          = ANSI(1)
	ColorGreen        = ANSI(2)
	ColorYellow       = ANSI(3)
	ColorBlue         = ANSI(4)
	ColorMagenta      = ANSI(5)
	ColorCyan         = ANSI(6)
	ColorWhite        = ANSI(7)
	ColorBrightBlack  = ANSI(8)
	ColorBrightRed    = ANSI(9)
	ColorBrightGreen  = ANSI(10)
	ColorBrightYellow = ANSI(11)
	ColorBrightBlue   = ANSI(12)
	ColorBrightCyan   = ANSI(14)
	ColorBrightWhite  = ANSI(15)
)

// Common true colors
var (
	ColorTransparent = Color{}
	ColorBlackTrue   = RGB(0, 0, 0)
	ColorWhiteTrue   = RGB(255, 255, 255)
	ColorGray        = RGB(128, 128, 128)
)

func Blend(a, b Color, ratio float64) Color {
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	return Color{
		R: uint8(float64(a.R) + (float64(b.R)-float64(a.R))*ratio),
		G: uint8(float64(a.G) + (float64(b.G)-float64(a.G))*ratio),
		B: uint8(float64(a.B) + (float64(b.B)-float64(a.B))*ratio),
	}
}

func Lerp(a, b Color, t float64) Color { return Blend(a, b, t) }

func (c Color) Lighten(amount float64) Color { return Blend(c, ColorWhiteTrue, amount) }
func (c Color) Darken(amount float64) Color  { return Blend(c, ColorBlackTrue, amount) }

func (c Color) Saturate(amount float64) Color {
	gray := uint8(float64(c.R)*0.299 + float64(c.G)*0.587 + float64(c.B)*0.114)
	return Blend(Color{R: gray, G: gray, B: gray}, c, 1+amount)
}
