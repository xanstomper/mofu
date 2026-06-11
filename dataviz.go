package mofu

import (
	"math"
	"sort"
)

// ---------------------------------------------------------------------------
// Data Visualization (Anthology Ch.13)
// ---------------------------------------------------------------------------

// DataLabelledPoint is a DataPoint with an optional label.
type DataLabelledPoint struct {
	DataPoint
	Label string
}

// AxisConfig controls axis rendering.
type AxisConfig struct {
	Show     bool
	Min      float64
	Max      float64
	Ticks    int
	Labels   []string
	MinLabel string
	MaxLabel string
}

// LegendConfig controls chart legend rendering.
type LegendConfig struct {
	Show     bool
	Position string
	Items    []LegendItem
}

// LegendItem is one legend row.
type LegendItem struct {
	Label string
	Color Color
}

// ChartFrame describes the target cell region.
type ChartFrame struct {
	X, Y     int
	Width    int
	Height   int
	Renderer *Renderer
}

// BarChart renders vertical or horizontal bars.
type BarChart struct {
	Title      string
	Values     []float64
	Labels     []string
	Width      int
	Height     int
	Stacked    bool
	Color      Color
	ShowLegend bool
}

// RenderBarChart renders a simple bar chart into the renderer.
func RenderBarChart(r *Renderer, chart BarChart, x, y int) {
	if r == nil || len(chart.Values) == 0 || chart.Width <= 0 || chart.Height <= 0 {
		return
	}
	maxVal := 0.0
	for _, v := range chart.Values {
		if v > maxVal {
			maxVal = v
		}
	}
	if maxVal == 0 {
		maxVal = 1
	}
	barW := chart.Width / len(chart.Values)
	if barW < 1 {
		barW = 1
	}
	for i, v := range chart.Values {
		h := int(math.Round((v / maxVal) * float64(chart.Height)))
		if h < 0 {
			h = 0
		}
		baseY := y + chart.Height
		for j := 0; j < h; j++ {
			r.WriteStyledString("█", x+i*barW, baseY-j-1, DefaultStyle().Fg(chart.Color))
		}
		if i < len(chart.Labels) {
			r.WriteStyledString(Truncate(chart.Labels[i], barW, true), x+i*barW, y+chart.Height, DefaultStyle())
		}
	}
}

// LineChart renders connected points.
type LineChart struct {
	Title      string
	Series     []DataSeries
	Width      int
	Height     int
	XMin, XMax float64
	YMin, YMax float64
	ShowGrid   bool
	ShowPoints bool
}

// RenderLineChart renders a line chart with simple ASCII/Unicode strokes.
func RenderLineChart(r *Renderer, chart LineChart, x, y int) {
	if r == nil || len(chart.Series) == 0 || chart.Width <= 0 || chart.Height <= 0 {
		return
	}
	plotW := chart.Width - 2
	plotH := chart.Height - 2
	if plotW < 2 || plotH < 2 {
		return
	}
	xMin, xMax := chart.XMin, chart.XMax
	yMin, yMax := chart.YMin, chart.YMax
	if xMin == xMax {
		xMax = xMin + 1
	}
	if yMin == yMax {
		yMax = yMin + 1
	}
	for _, s := range chart.Series {
		for i, p := range s.Points {
			cx := x + 1 + int((p.X-xMin)/(xMax-xMin)*float64(plotW))
			cy := y + 1 + int((1-(p.Y-yMin)/(yMax-yMin))*float64(plotH))
			style := DefaultStyle().Fg(s.Color)
			r.WriteStyledString("•", cx, cy, style)
			if i > 0 {
				prev := s.Points[i-1]
				px := x + 1 + int((prev.X-xMin)/(xMax-xMin)*float64(plotW))
				py := y + 1 + int((1-(prev.Y-yMin)/(yMax-yMin))*float64(plotH))
				drawLineStyled(r, px, py, cx, cy, style)
			}
		}
	}
}

// ScatterPlot renders points without connecting lines.
type ScatterPlot struct {
	Points     []DataLabelledPoint
	Width      int
	Height     int
	XMin, XMax float64
	YMin, YMax float64
}

// RenderScatterPlot renders a scatter plot.
func RenderScatterPlot(r *Renderer, chart ScatterPlot, x, y int) {
	if r == nil || len(chart.Points) == 0 || chart.Width <= 0 || chart.Height <= 0 {
		return
	}
	plotW := chart.Width - 2
	plotH := chart.Height - 2
	if plotW < 2 || plotH < 2 {
		return
	}
	for _, p := range chart.Points {
		cx := x + 1 + int((p.X-chart.XMin)/(chart.XMax-chart.XMin)*float64(plotW))
		cy := y + 1 + int((1-(p.Y-chart.YMin)/(chart.YMax-chart.YMin))*float64(plotH))
		r.WriteStyledString("×", cx, cy, DefaultStyle())
	}
}

// Histogram renders bucketed values.
type Histogram struct {
	Values  []float64
	Buckets int
	Width   int
	Height  int
	Color   Color
}

// RenderHistogram renders a histogram.
func RenderHistogram(r *Renderer, chart Histogram, x, y int) {
	if chart.Buckets <= 0 {
		chart.Buckets = 10
	}
	if len(chart.Values) == 0 {
		return
	}
	minVal, maxVal := minmax(chart.Values)
	if minVal == maxVal {
		maxVal = minVal + 1
	}
	bucket := make([]int, chart.Buckets)
	for _, v := range chart.Values {
		idx := int((v - minVal) / (maxVal - minVal) * float64(chart.Buckets))
		if idx >= chart.Buckets {
			idx = chart.Buckets - 1
		}
		bucket[idx]++
	}
	RenderBarChart(r, BarChart{Values: floatSlice(bucket), Width: chart.Width, Height: chart.Height, Color: chart.Color}, x, y)
}

// Heatmap renders a 2D matrix as shaded cells.
type Heatmap struct {
	Rows      [][]float64
	Width     int
	Height    int
	HighColor Color
	LowColor  Color
}

// RenderHeatmap renders a Heatmap into the renderer.
func RenderHeatmap(r *Renderer, chart Heatmap, x, y int) {
	if r == nil || len(chart.Rows) == 0 || len(chart.Rows[0]) == 0 {
		return
	}
	minVal, maxVal := 0.0, 0.0
	for i, row := range chart.Rows {
		for j, v := range row {
			if i == 0 && j == 0 {
				minVal, maxVal = v, v
			}
			if v < minVal {
				minVal = v
			}
			if v > maxVal {
				maxVal = v
			}
		}
	}
	if maxVal == minVal {
		maxVal = minVal + 1
	}
	for i, row := range chart.Rows {
		for j, v := range row {
			t := (v - minVal) / (maxVal - minVal)
			c := blendColors(chart.LowColor, chart.HighColor, t)
			r.WriteStyledString("▓", x+j, y+i, DefaultStyle().Fg(c))
		}
	}
}

// Sparkline renders an inline mini line chart.
func Sparkline(values []float64, width int) string {
	if len(values) == 0 || width <= 0 {
		return ""
	}
	minVal, maxVal := minmax(values)
	if maxVal == minVal {
		maxVal = minVal + 1
	}
	glyphs := []rune("▁▂▃▄▅▆▇█")
	step := float64(len(values)-1) / float64(width-1)
	var out []rune
	for i := 0; i < width; i++ {
		idx := int(math.Round(float64(i) * step))
		if idx >= len(values) {
			idx = len(values) - 1
		}
		g := int((values[idx] - minVal) / (maxVal - minVal) * float64(len(glyphs)-1))
		out = append(out, glyphs[g])
	}
	return string(out)
}

// Stats computes min, max, average, and median.
func Stats(values []float64) (min, max, avg, median float64) {
	if len(values) == 0 {
		return 0, 0, 0, 0
	}
	min, max = values[0], values[0]
	sum := 0.0
	cp := make([]float64, len(values))
	copy(cp, values)
	sort.Float64s(cp)
	for _, v := range values {
		sum += v
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	avg = sum / float64(len(values))
	mid := len(cp) / 2
	if len(cp)%2 == 0 {
		median = (cp[mid-1] + cp[mid]) / 2
	} else {
		median = cp[mid]
	}
	return
}

func minmax(values []float64) (float64, float64) {
	if len(values) == 0 {
		return 0, 0
	}
	minVal, maxVal := values[0], values[0]
	for _, v := range values[1:] {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}
	return minVal, maxVal
}

func floatSlice(src []int) []float64 {
	out := make([]float64, len(src))
	for i, v := range src {
		out[i] = float64(v)
	}
	return out
}

func drawLineStyled(r *Renderer, x0, y0, x1, y1 int, style Style) {
	dx := absInt(x1 - x0)
	dy := -absInt(y1 - y0)
	sx, sy := 1, -1
	err := dx + dy
	x, y := x0, y0
	for {
		r.WriteStyledString("·", x, y, style)
		if x == x1 && y == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x += sx
		}
		if e2 <= dx {
			err += dx
			y += sy
		}
	}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
