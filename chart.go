package mofu

import "fmt"

type ChartType int

const (
	ChartLine ChartType = iota
	ChartBar
	ChartScatter
	ChartHeatmap
	ChartGauge
	ChartPie
	ChartArea
)

type DataPoint struct {
	X, Y float64
}

type DataSeries struct {
	Label  string
	Points []DataPoint
	Color  Color
}

type Chart struct {
	Type       ChartType
	Series     []DataSeries
	Title      string
	Width      int
	Height     int
	XMin, XMax float64
	YMin, YMax float64
	ShowGrid   bool
	ShowLegend bool
	ShowPoints bool
	Stacked    bool
}

func RenderChart(r *Renderer, chart *Chart, x, y int) {
	if len(chart.Series) == 0 || chart.Width <= 0 || chart.Height <= 0 {
		return
	}

	plotW := chart.Width - 2
	plotH := chart.Height - 2
	if plotW < 2 || plotH < 2 {
		return
	}

	xMin, xMax := chart.XMin, chart.XMax
	yMin, yMax := chart.YMin, chart.YMax
	if xMin == 0 && xMax == 0 {
		for _, s := range chart.Series {
			for _, p := range s.Points {
				if p.X < xMin {
					xMin = p.X
				}
				if p.X > xMax {
					xMax = p.X
				}
			}
		}
	}
	if yMin == 0 && yMax == 0 {
		for _, s := range chart.Series {
			for _, p := range s.Points {
				if p.Y < yMin {
					yMin = p.Y
				}
				if p.Y > yMax {
					yMax = p.Y
				}
			}
		}
	}
	if xMax == xMin {
		xMax = xMin + 1
	}
	if yMax == yMin {
		yMax = yMin + 1
	}

	xRange := xMax - xMin
	yRange := yMax - yMin


	for _, series := range chart.Series {
		switch chart.Type {
		case ChartLine:
			renderLineSeries(r, series, x+1, y+1, plotW, plotH, xMin, xRange, yMin, yRange, chart.ShowPoints)
		case ChartBar:
			renderBarSeries(r, series, x+1, y+1, plotW, plotH, yMin, yRange, chart.Stacked)
		case ChartScatter:
			renderScatterSeries(r, series, x+1, y+1, plotW, plotH, xMin, xRange, yMin, yRange)
		}
	}

	if chart.Title != "" {
		titleX := x + (chart.Width-len(chart.Title))/2
		if titleX < x {
			titleX = x + 1
		}
		r.WriteString(chart.Title, titleX, y, ColorBrightWhite, Color{}, AttrBold)
	}
}

func renderLineSeries(r *Renderer, s DataSeries, ox, oy, w, h int, xMin, xRange, yMin, yRange float64, showPoints bool) {
	pts := s.Points
	if len(pts) < 2 {
		return
	}
	fg := s.Color

	for i := 1; i < len(pts); i++ {
		x0 := ox + int((pts[i-1].X-xMin)/xRange*float64(w-1))
		y0 := oy + h - 1 - int((pts[i-1].Y-yMin)/yRange*float64(h-1))
		x1 := ox + int((pts[i].X-xMin)/xRange*float64(w-1))
		y1 := oy + h - 1 - int((pts[i].Y-yMin)/yRange*float64(h-1))

		drawLine(r, x0, y0, x1, y1, fg)
	}

	if showPoints {
		for _, p := range pts {
			px := ox + int((p.X-xMin)/xRange*float64(w-1))
			py := oy + h - 1 - int((p.Y-yMin)/yRange*float64(h-1))
			if px >= 0 && px < r.width && py >= 0 && py < r.height {
				r.front.Set(px, py, '●', fg, Color{}, 0)
			}
		}
	}
}

func renderBarSeries(r *Renderer, s DataSeries, ox, oy, w, h int, yMin, yRange float64, stacked bool) {
	pts := s.Points
	if len(pts) == 0 {
		return
	}
	fg := s.Color

	for i, p := range pts {
		barW := w / len(pts)
		if barW < 1 {
			barW = 1
		}
		bx := ox + i*barW

		valH := int((p.Y - yMin) / yRange * float64(h-1))
		if valH < 0 {
			valH = 0
		}
		if valH > h-1 {
			valH = h - 1
		}

		for dy := 0; dy < valH; dy++ {
			by := oy + h - 1 - dy
			for dx := 0; dx < barW-1; dx++ {
				ch := ' '
				frac := float64(dy) / float64(valH)
				if frac > 0.75 {
					ch = '█'
				} else if frac > 0.5 {
					ch = '▓'
				} else if frac > 0.25 {
					ch = '▒'
				} else {
					ch = '░'
				}
				r.front.Set(bx+dx, by, ch, fg, Color{}, 0)
			}
		}
	}
}

func renderScatterSeries(r *Renderer, s DataSeries, ox, oy, w, h int, xMin, xRange, yMin, yRange float64) {
	pts := s.Points
	if len(pts) == 0 {
		return
	}
	fg := s.Color

	for _, p := range pts {
		px := ox + int((p.X-xMin)/xRange*float64(w-1))
		py := oy + h - 1 - int((p.Y-yMin)/yRange*float64(h-1))
		if px >= 0 && px < r.width && py >= 0 && py < r.height {
			r.front.Set(px, py, '◆', fg, Color{}, 0)
		}
	}
}

func drawLine(r *Renderer, x0, y0, x1, y1 int, color Color) {
	dx := abs(x1 - x0)
	dy := -abs(y1 - y0)
	sx := 1
	if x0 > x1 {
		sx = -1
	}
	sy := 1
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy

	for {
		if x0 >= 0 && x0 < r.width && y0 >= 0 && y0 < r.height {
			r.front.Set(x0, y0, '─', color, Color{}, 0)
		}
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func RenderSparkline(r *Renderer, values []float64, x, y, width int, color Color) {
	if len(values) == 0 || width < 2 {
		return
	}

	min, max := values[0], values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	if max == min {
		max = min + 1
	}
	rng := max - min

	chars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	step := float64(len(values)) / float64(width)
	for i := 0; i < width && i < len(values); i++ {
		idx := int(float64(i) * step)
		if idx >= len(values) {
			idx = len(values) - 1
		}
		level := int((values[idx] - min) / rng * float64(len(chars)-1))
		if level < 0 {
			level = 0
		}
		if level >= len(chars) {
			level = len(chars) - 1
		}
		r.front.Set(x+i, y, chars[level], color, Color{}, 0)
	}
}

type ColorScale struct {
	Stops []ColorStop
}

type ColorStop struct {
	Pos   float64
	Color Color
}

func (cs *ColorScale) At(t float64) Color {
	if t <= 0 || len(cs.Stops) == 0 {
		if len(cs.Stops) > 0 {
			return cs.Stops[0].Color
		}
		return Color{}
	}
	if t >= 1 {
		return cs.Stops[len(cs.Stops)-1].Color
	}
	for i := 0; i < len(cs.Stops)-1; i++ {
		if t >= cs.Stops[i].Pos && t <= cs.Stops[i+1].Pos {
			localT := (t - cs.Stops[i].Pos) / (cs.Stops[i+1].Pos - cs.Stops[i].Pos)
			return blendColors(cs.Stops[i].Color, cs.Stops[i+1].Color, localT)
		}
	}
	return cs.Stops[len(cs.Stops)-1].Color
}

func blendColors(a, b Color, t float64) Color {
	return Color{
		R: uint8(float64(a.R)*(1-t) + float64(b.R)*t),
		G: uint8(float64(a.G)*(1-t) + float64(b.G)*t),
		B: uint8(float64(a.B)*(1-t) + float64(b.B)*t),
	}
}

var BrailleDots = [8]uint16{0x01, 0x08, 0x02, 0x10, 0x04, 0x20, 0x40, 0x80}

func BrailleChar(dots [8]bool) rune {
	var code uint16 = 0x2800
	for i, on := range dots {
		if on {
			code |= BrailleDots[i]
		}
	}
	return rune(code)
}

func RenderBrailleChart(r *Renderer, data [][]float64, x, y, w, h int, scale ColorScale) {
	if len(data) == 0 || len(data[0]) == 0 {
		return
	}
	bw := w * 2
	bh := h * 4

	min, max := data[0][0], data[0][0]
	for _, row := range data {
		for _, v := range row {
			if v < min {
				min = v
			}
			if v > max {
				max = v
			}
		}
	}
	rng := max - min
	if rng == 0 {
		rng = 1
	}

	for by := 0; by < bh; by++ {
		for bx := 0; bx < bw; bx++ {
			di := by % 4
			dj := bx % 2
			dotIdx := di*2 + dj

			ci := bx / 2
			cj := by / 4

			if ci >= len(data[0]) || cj >= len(data) {
				continue
			}

			v := data[cj][ci]
			normalized := (v - min) / rng
			on := normalized > float64(by%4)/4.0

			cx := x + ci
			cy := y + cj
			if cx >= 0 && cx < r.width && cy >= 0 && cy < r.height {
				cell := &r.front.Cells[cy][cx]
				currentDots := brailleDecode(cell.Char)
				if on {
					currentDots[dotIdx] = true
				}
				cell.Char = BrailleChar(currentDots)
				cell.Fg = scale.At(normalized)
				cell.Dirty = true
			}
		}
	}
}

func brailleDecode(ch rune) [8]bool {
	var dots [8]bool
	code := uint16(ch)
	if code < 0x2800 || code > 0x28FF {
		return dots
	}
	offset := code - 0x2800
	for i := 0; i < 8; i++ {
		dots[i] = offset&BrailleDots[i] != 0
	}
	return dots
}

type Gauge struct {
	Percent float64
	Width   int
	Fg, Bg  Color
}

func RenderGauge(r *Renderer, g *Gauge, x, y int) {
	if g.Width < 2 {
		return
	}
	filled := int(g.Percent * float64(g.Width-2))
	if filled < 0 {
		filled = 0
	}
	if filled > g.Width-2 {
		filled = g.Width - 2
	}

	r.front.Set(x, y, '[', g.Fg, g.Bg, 0)
	for i := 0; i < g.Width-2; i++ {
		if i < filled {
			r.front.Set(x+1+i, y, '█', g.Fg, g.Bg, 0)
		} else {
			r.front.Set(x+1+i, y, '░', ColorGray, g.Bg, 0)
		}
	}
	r.front.Set(x+g.Width-1, y, ']', g.Fg, g.Bg, 0)

	pctStr := fmt.Sprintf("%3.0f%%", g.Percent*100)
	pctX := x + (g.Width-len(pctStr))/2
	if pctX > x && pctX+len(pctStr) < x+g.Width {
		for i, ch := range pctStr {
			r.front.Set(pctX+i, y, ch, ColorBrightWhite, g.Bg, AttrBold)
		}
	}
}
