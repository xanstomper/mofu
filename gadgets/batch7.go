package gadgets

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/xanstomper/mofu"
)

// =========================================================================
// BATCH 7: Advanced Data Visualization (10 gadgets)
// =========================================================================

type RealRadarChart struct {
	Base
	Title   string
	Labels  []string
	Values  []float64
	MaxVal  float64
	mu      sync.RWMutex
}

func NewRealRadarChart(id string, labels []string) *RealRadarChart {
	return &RealRadarChart{Base: *NewBase(id), Labels: labels, MaxVal: 100}
}

func (g *RealRadarChart) SetValues(vals []float64) {
	g.mu.Lock()
	g.Values = vals
	g.mu.Unlock()
}

func (g *RealRadarChart) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	if g.Title != "" {
		ctx.Renderer.WriteString(g.Title, r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
		y++
	}

	rad := r.Height / 2 - 1
	if rad > r.Width/4 {
		rad = r.Width / 4
	}
	cx := r.X + r.Width/2
	cy := r.Y + rad + 2

	n := len(g.Labels)
	if n == 0 || g.MaxVal == 0 {
		return
	}

	layers := []float64{0.25, 0.5, 0.75, 1.0}
	for _, layer := range layers {
		ring := make([]string, rad*2+1)
		for i := range ring {
			ring[i] = strings.Repeat(" ", rad*4)
		}
		for a := 0; a < 360; a += 2 {
			angle := float64(a) * math.Pi / 180
			px := int(float64(rad*2) * layer * math.Cos(angle))
			py := int(float64(rad) * layer * math.Sin(angle))
			if py+rad >= 0 && py+rad <= rad*2 {
				idx := py + rad
				posX := px + rad*2
				if posX >= 0 && posX < len(ring[idx]) {
					runes := []rune(ring[idx])
					runes[posX] = '·'
					ring[idx] = string(runes)
				}
			}
		}
		for i, line := range ring {
			if cy-rad+i >= y && cy-rad+i < r.Y+r.Height-2 {
				ctx.Renderer.WriteString(line, cx-rad*2, cy-rad+i, mofu.Hex("444444"), mofu.ColorBlack, 0)
			}
		}
	}

	if len(g.Values) == n {
		points := make([]struct{ x, y int }, n)
		for i := 0; i < n; i++ {
			angle := float64(i) * 2 * math.Pi / float64(n) - math.Pi/2
			val := g.Values[i] / g.MaxVal
			points[i] = struct{ x, y int }{
				x: cx + int(float64(rad*2)*val*math.Cos(angle)),
				y: cy + int(float64(rad)*val*math.Sin(angle)),
			}
		}
		for i := 0; i < n; i++ {
			next := (i + 1) % n
			x0, y0 := points[i].x, points[i].y
			x1, y1 := points[next].x, points[next].y
			steps := int(math.Max(math.Abs(float64(x1-x0)), math.Abs(float64(y1-y0))))
			if steps == 0 {
				steps = 1
			}
			for s := 0; s <= steps; s++ {
				t := float64(s) / float64(steps)
				px := int(float64(x0)*(1-t) + float64(x1)*t)
				py := int(float64(y0)*(1-t) + float64(y1)*t)
				if py >= r.Y && py < r.Y+r.Height-2 && px >= r.X && px < r.X+r.Width {
					ctx.Renderer.WriteString("█", px, py, mofu.Hex("89b4fa"), mofu.ColorBlack, 0)
				}
			}
		}
	}

	for i, label := range g.Labels {
		angle := float64(i) * 2 * math.Pi / float64(n) - math.Pi/2
		lx := cx + int(float64(rad+2)*2*math.Cos(angle)) - len(label)/2
		ly := cy + int(float64(rad+2)*math.Sin(angle))
		if ly >= r.Y && ly < r.Y+r.Height-1 && lx >= r.X && lx < r.X+r.Width {
			ctx.Renderer.WriteString(label, lx, ly, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
		}
	}
}

func (g *RealRadarChart) HandleEvent(e mofu.Event) mofu.Cmd { return nil }

type RealWaterfallChart struct {
	Base
	Title   string
	Items   []WaterfallItem
	mu      sync.RWMutex
}

type WaterfallItem struct {
	Label  string
	Start  float64
	End    float64
	Color  mofu.Color
}

func NewRealWaterfallChart(id string) *RealWaterfallChart {
	return &RealWaterfallChart{Base: *NewBase(id)}
}

func (g *RealWaterfallChart) SetItems(items []WaterfallItem) {
	g.mu.Lock()
	g.Items = items
	g.mu.Unlock()
}

func (g *RealWaterfallChart) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	if g.Title != "" {
		ctx.Renderer.WriteString(g.Title, r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
		y++
	}

	if len(g.Items) == 0 {
		return
	}

	minVal := g.Items[0].Start
	maxVal := g.Items[0].End
	for _, item := range g.Items {
		if item.Start < minVal {
			minVal = item.Start
		}
		if item.End > maxVal {
			maxVal = item.End
		}
	}

	range_ := maxVal - minVal
	if range_ == 0 {
		range_ = 1
	}

	barW := r.Width - 20

	for _, item := range g.Items {
		if y >= r.Y+r.Height-1 {
			break
		}

		label := item.Label
		if len(label) > 8 {
			label = label[:6] + ".."
		}

		startX := int((item.Start - minVal) / range_ * float64(barW))
		endX := int((item.End - minVal) / range_ * float64(barW))
		width := endX - startX
		if width < 1 {
			width = 1
		}

		bar := strings.Repeat("█", width)
		ctx.Renderer.WriteString(fmt.Sprintf(" %-8s %s%s", label, strings.Repeat(" ", startX), bar), r.X, y, item.Color, mofu.ColorBlack, 0)
		y++
	}
}

func (g *RealWaterfallChart) HandleEvent(e mofu.Event) mofu.Cmd { return nil }

type RealFunnelChart struct {
	Base
	Title  string
	Stages []FunnelStage
	mu     sync.RWMutex
}

type FunnelStage struct {
	Label string
	Value float64
}

func NewRealFunnelChart(id string) *RealFunnelChart {
	return &RealFunnelChart{Base: *NewBase(id)}
}

func (g *RealFunnelChart) SetStages(stages []FunnelStage) {
	g.mu.Lock()
	g.Stages = stages
	g.mu.Unlock()
}

func (g *RealFunnelChart) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	if g.Title != "" {
		ctx.Renderer.WriteString(g.Title, r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
		y++
	}

	if len(g.Stages) == 0 {
		return
	}

	maxVal := g.Stages[0].Value
	if maxVal == 0 {
		return
	}

	barMaxW := r.Width - 20
	halfMax := barMaxW / 2
	colors := []mofu.Color{
		mofu.Hex("f38ba8"), mofu.Hex("fab387"), mofu.Hex("f9e2af"),
		mofu.Hex("a6e3a1"), mofu.Hex("94e2d5"), mofu.Hex("89b4fa"),
	}

	for i, stage := range g.Stages {
		if y >= r.Y+r.Height-2 {
			break
		}

		width := int(stage.Value / maxVal * float64(halfMax))
		if width < 1 {
			width = 1
		}

		bar := strings.Repeat("█", width*2)
		pct := stage.Value / maxVal * 100

		color := colors[i%len(colors)]
		pad := halfMax - width
		ctx.Renderer.WriteString(fmt.Sprintf(" %s%s %s %.1f%%", strings.Repeat(" ", pad), bar, stage.Label, pct), r.X, y, color, mofu.ColorBlack, 0)
		y++

		if i < len(g.Stages)-1 {
			nextW := int(g.Stages[i+1].Value / maxVal * float64(halfMax))
			if nextW < width {
				connector := strings.Repeat(" ", pad+width-nextW) + strings.Repeat("/", nextW) + strings.Repeat("\\", nextW)
				if len(connector) > r.Width-1 {
					connector = connector[:r.Width-1]
				}
				ctx.Renderer.WriteString(connector, r.X, y, mofu.Hex("585b70"), mofu.ColorBlack, 0)
				y++
			}
		}
	}
}

func (g *RealFunnelChart) HandleEvent(e mofu.Event) mofu.Cmd { return nil }

type RealSankeyDiagram struct {
	Base
	Title string
	Nodes []SankeyNode
	Links []SankeyLink
	mu    sync.RWMutex
}

type SankeyNode struct {
	Name  string
	Value float64
	Color mofu.Color
}

type SankeyLink struct {
	From  int
	To    int
	Value float64
}

func NewRealSankeyDiagram(id string) *RealSankeyDiagram {
	return &RealSankeyDiagram{Base: *NewBase(id)}
}

func (g *RealSankeyDiagram) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	if g.Title != "" {
		ctx.Renderer.WriteString(g.Title, r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
		y++
	}

	if len(g.Nodes) == 0 {
		return
	}

	leftX := r.X + 2
	rightX := r.X + r.Width/2
	nodeH := 3

	for i, node := range g.Nodes {
		if y+nodeH >= r.Y+r.Height {
			break
		}
		nY := y + i*(nodeH+1)

		ctx.Renderer.WriteString(strings.Repeat("█", 12), leftX, nY, node.Color, mofu.ColorBlack, 0)
		ctx.Renderer.WriteString(fmt.Sprintf(" %s (%.0f)", node.Name, node.Value), leftX+13, nY, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)

		if i < r.Height/4-2 {
			for dy := 0; dy < nodeH; dy++ {
				ctx.Renderer.WriteString("░░░░░░░░░░░░", rightX, nY+dy, mofu.Hex("444444"), mofu.ColorBlack, 0)
			}
		}
	}
}

func (g *RealSankeyDiagram) HandleEvent(e mofu.Event) mofu.Cmd { return nil }

type RealTreemapChart struct {
	Base
	Title   string
	Items   []TreemapItem
	mu      sync.RWMutex
}

type TreemapItem struct {
	Name  string
	Value float64
	Color mofu.Color
}

func NewRealTreemapChart(id string) *RealTreemapChart {
	return &RealTreemapChart{Base: *NewBase(id)}
}

func (g *RealTreemapChart) SetItems(items []TreemapItem) {
	g.mu.Lock()
	g.Items = items
	g.mu.Unlock()
}

func (g *RealTreemapChart) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	if g.Title != "" {
		ctx.Renderer.WriteString(g.Title, r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
		y++
	}

	if len(g.Items) == 0 {
		return
	}

	total := 0.0
	for _, item := range g.Items {
		total += item.Value
	}
	if total == 0 {
		return
	}

	sorted := make([]TreemapItem, len(g.Items))
	copy(sorted, g.Items)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Value > sorted[j].Value })

	x := r.X
	curY := y
	maxW := r.Width - 2
	remaining := total

	for _, item := range sorted {
		if curY >= r.Y+r.Height-1 {
			break
		}

		frac := item.Value / remaining
		w := int(frac * float64(maxW))
		if w < 3 {
			w = 3
		}
		if x+w > r.X+maxW {
			w = r.X + maxW - x
		}

		h := r.Height - 2 - (curY - r.Y)
		if h > 4 {
			h = 4
		}

		block := strings.Repeat("█", w)
		for dy := 0; dy < h; dy++ {
			if curY+dy >= r.Y+r.Height-1 {
				break
			}
			ctx.Renderer.WriteString(block, x, curY+dy, item.Color, mofu.ColorBlack, 0)
		}

		if w > len(item.Name)+2 {
			name := item.Name
			if len(name) > w-2 {
				name = name[:w-4] + ".."
			}
			ctx.Renderer.WriteString(" "+name, x, curY, mofu.Hex("1e1e2e"), item.Color, mofu.AttrBold)
		}

		x += w
		if x >= r.X+maxW {
			x = r.X
			curY += h + 1
		}
		remaining -= item.Value
	}
}

func (g *RealTreemapChart) HandleEvent(e mofu.Event) mofu.Cmd { return nil }

type RealHeatCalendar struct {
	Base
	Title    string
	Data     map[string]int
	MaxVal   int
	mu       sync.RWMutex
}

func NewRealHeatCalendar(id string) *RealHeatCalendar {
	return &RealHeatCalendar{Base: *NewBase(id), Data: make(map[string]int)}
}

func (g *RealHeatCalendar) SetDay(date string, value int) {
	g.mu.Lock()
	g.Data[date] = value
	if value > g.MaxVal {
		g.MaxVal = value
	}
	g.mu.Unlock()
}

func (g *RealHeatCalendar) SetRange(start, end time.Time, values []int) {
	g.mu.Lock()
	d := start
	for i := 0; i < len(values) && !d.After(end); i++ {
		key := d.Format("2006-01-02")
		g.Data[key] = values[i]
		if values[i] > g.MaxVal {
			g.MaxVal = values[i]
		}
		d = d.AddDate(0, 0, 1)
	}
	g.mu.Unlock()
}

func (g *RealHeatCalendar) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	if g.Title != "" {
		ctx.Renderer.WriteString(g.Title, r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
		y++
	}

	now := time.Now()
	startOfYear := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.Local)
	startWeekday := int(startOfYear.Weekday())

	x := r.X
	for dy := 0; dy < 7; dy++ {
		if y+dy >= r.Y+r.Height-1 {
			break
		}
		dayLabel := time.Weekday(dy).String()[:1]
		ctx.Renderer.WriteString(dayLabel, r.X, y+dy, mofu.Hex("585b70"), mofu.ColorBlack, 0)
	}

	x = r.X + 3
	for week := 0; week < 53; week++ {
		if x >= r.X+r.Width-1 {
			break
		}
		for day := 0; day < 7; day++ {
			dayOfYear := week*7 + day - startWeekday
			if dayOfYear < 0 || dayOfYear > 365 {
				continue
			}
			date := startOfYear.AddDate(0, 0, dayOfYear)
			key := date.Format("2006-01-02")
			val := g.Data[key]

			color := mofu.Hex("1e1e2e")
			if g.MaxVal > 0 && val > 0 {
				intensity := float64(val) / float64(g.MaxVal)
				switch {
				case intensity > 0.75:
					color = mofu.Hex("a6e3a1")
				case intensity > 0.50:
					color = mofu.Hex("94e2d5")
				case intensity > 0.25:
					color = mofu.Hex("89b4fa")
				default:
					color = mofu.Hex("585b70")
				}
			}
			ctx.Renderer.WriteString("██", x, y+day, color, mofu.ColorBlack, 0)
		}
		x += 2
	}
}

func (g *RealHeatCalendar) HandleEvent(e mofu.Event) mofu.Cmd { return nil }

type RealBoxPlot struct {
	Base
	Title   string
	Data    []BoxPlotData
	mu      sync.RWMutex
}

type BoxPlotData struct {
	Label string
	Min   float64
	Q1    float64
	Med   float64
	Q3    float64
	Max   float64
	Color mofu.Color
}

func NewRealBoxPlot(id string) *RealBoxPlot {
	return &RealBoxPlot{Base: *NewBase(id)}
}

func (g *RealBoxPlot) SetData(data []BoxPlotData) {
	g.mu.Lock()
	g.Data = data
	g.mu.Unlock()
}

func (g *RealBoxPlot) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	if g.Title != "" {
		ctx.Renderer.WriteString(g.Title, r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
		y++
	}

	if len(g.Data) == 0 {
		return
	}

	globalMin := g.Data[0].Min
	globalMax := g.Data[0].Max
	for _, d := range g.Data {
		if d.Min < globalMin {
			globalMin = d.Min
		}
		if d.Max > globalMax {
			globalMax = d.Max
		}
	}

	range_ := globalMax - globalMin
	if range_ == 0 {
		range_ = 1
	}

	barW := r.Width - 16

	for _, d := range g.Data {
		if y >= r.Y+r.Height-1 {
			break
		}

		minX := int((d.Min - globalMin) / range_ * float64(barW))
		q1X := int((d.Q1 - globalMin) / range_ * float64(barW))
		medX := int((d.Med - globalMin) / range_ * float64(barW))
		q3X := int((d.Q3 - globalMin) / range_ * float64(barW))
		maxX := int((d.Max - globalMin) / range_ * float64(barW))

		line := strings.Repeat(" ", barW)
		runes := []rune(line)
		for i := minX; i <= maxX && i < len(runes); i++ {
			runes[i] = '─'
		}
		if q1X < len(runes) && q3X < len(runes) {
			for i := q1X; i <= q3X && i < len(runes); i++ {
				runes[i] = '█'
			}
		}
		if medX < len(runes) {
			runes[medX] = '│'
		}
		if minX < len(runes) {
			runes[minX] = '├'
		}
		if maxX < len(runes) {
			runes[maxX] = '┤'
		}

		ctx.Renderer.WriteString(fmt.Sprintf(" %-8s %s", d.Label, string(runes)), r.X, y, d.Color, mofu.ColorBlack, 0)
		y++
	}
}

func (g *RealBoxPlot) HandleEvent(e mofu.Event) mofu.Cmd { return nil }

type RealPolarAreaChart struct {
	Base
	Title   string
	Items   []PolarItem
	mu      sync.RWMutex
}

type PolarItem struct {
	Label string
	Value float64
	Color mofu.Color
}

func NewRealPolarAreaChart(id string) *RealPolarAreaChart {
	return &RealPolarAreaChart{Base: *NewBase(id)}
}

func (g *RealPolarAreaChart) SetItems(items []PolarItem) {
	g.mu.Lock()
	g.Items = items
	g.mu.Unlock()
}

func (g *RealPolarAreaChart) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	if g.Title != "" {
		ctx.Renderer.WriteString(g.Title, r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
		y++
	}

	if len(g.Items) == 0 {
		return
	}

	maxVal := 0.0
	for _, item := range g.Items {
		if item.Value > maxVal {
			maxVal = item.Value
		}
	}
	if maxVal == 0 {
		return
	}

	rad := r.Height / 2 - 1
	if rad > r.Width/4 {
		rad = r.Width / 4
	}
	cx := r.X + r.Width/2
	cy := r.Y + rad + 2
	n := len(g.Items)
	sweepAngle := 360.0 / float64(n)

	for ring := 1; ring <= 4; ring++ {
		layerRadius := float64(rad) * float64(ring) / 4.0
		for angle := 0; angle < 360; angle += 3 {
			radAngle := float64(angle) * math.Pi / 180.0
			px := cx + int(layerRadius*2*math.Cos(radAngle))
			py := cy + int(layerRadius*math.Sin(radAngle))
			if py >= r.Y && py < r.Y+r.Height && px >= r.X && px < r.X+r.Width {
				ctx.Renderer.WriteString("·", px, py, mofu.Hex("444444"), mofu.ColorBlack, 0)
			}
		}
	}

	for i, item := range g.Items {
		startAngle := float64(i) * sweepAngle
		endAngle := startAngle + sweepAngle
		radius := item.Value / maxVal * float64(rad)

		for angle := int(startAngle); angle < int(endAngle); angle += 2 {
			radAngle := float64(angle) * math.Pi / 180.0
			px := cx + int(radius*2*math.Cos(radAngle))
			py := cy + int(radius*math.Sin(radAngle))
			if py >= r.Y && py < r.Y+r.Height-2 && px >= r.X && px < r.X+r.Width {
				ctx.Renderer.WriteString("█", px, py, item.Color, mofu.ColorBlack, 0)
			}
		}

		midAngle := (startAngle + endAngle) / 2 * math.Pi / 180.0
		lx := cx + int(float64(rad+3)*2*math.Cos(midAngle))
		ly := cy + int(float64(rad+3)*math.Sin(midAngle))
		if ly >= r.Y && ly < r.Y+r.Height-1 && lx >= r.X && lx < r.X+r.Width-10 {
			pct := item.Value / maxVal * 100
			ctx.Renderer.WriteString(fmt.Sprintf(" %s %.0f%%", item.Label, pct), lx, ly, item.Color, mofu.ColorBlack, 0)
		}
	}
}

func (g *RealPolarAreaChart) HandleEvent(e mofu.Event) mofu.Cmd { return nil }

type RealStockCandle struct {
	Base
	Title    string
	Candles  []CandleData
	Width    int
	mu       sync.RWMutex
}

type CandleData struct {
	Date   string
	Open   float64
	High   float64
	Low    float64
	Close  float64
}

func NewRealStockCandle(id string) *RealStockCandle {
	return &RealStockCandle{Base: *NewBase(id), Width: 30}
}

func (g *RealStockCandle) SetCandles(candles []CandleData) {
	g.mu.Lock()
	g.Candles = candles
	g.mu.Unlock()
}

func (g *RealStockCandle) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	if g.Title != "" {
		ctx.Renderer.WriteString(g.Title, r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
		y++
	}

	if len(g.Candles) == 0 {
		return
	}

	minP := g.Candles[0].Low
	maxP := g.Candles[0].High
	for _, c := range g.Candles {
		if c.Low < minP {
			minP = c.Low
		}
		if c.High > maxP {
			maxP = c.High
		}
	}

	range_ := maxP - minP
	if range_ == 0 {
		range_ = 1
	}

	chartH := r.Height - 3
	if chartH < 1 {
		chartH = 1
	}

	candleW := (r.Width - 4) / len(g.Candles)
	if candleW < 1 {
		candleW = 1
	}

	for _, c := range g.Candles {
		if y >= r.Y+r.Height-2 {
			break
		}

		highY := int((maxP - c.High) / range_ * float64(chartH))
		lowY := int((maxP - c.Low) / range_ * float64(chartH))
		bodyTop := int((maxP - c.Open) / range_ * float64(chartH))
		bodyBot := int((maxP - c.Close) / range_ * float64(chartH))

		if bodyTop > bodyBot {
			bodyTop, bodyBot = bodyBot, bodyTop
		}

		bullish := c.Close >= c.Open

		for dy := 0; dy < chartH; dy++ {
			if y+dy >= r.Y+r.Height-2 {
				break
			}
			char := " "
			color := mofu.ColorBlack
			if dy >= highY && dy <= lowY && dy >= bodyTop && dy <= bodyBot {
				if bullish {
					char = "█"
					color = mofu.Hex("a6e3a1")
				} else {
					char = "█"
					color = mofu.Hex("f38ba8")
				}
			} else if dy >= highY && dy <= lowY {
				char = "│"
				if bullish {
					color = mofu.Hex("a6e3a1")
				} else {
					color = mofu.Hex("f38ba8")
				}
			}
			ctx.Renderer.WriteString(strings.Repeat(char, candleW), r.X+2, y+dy, color, mofu.ColorBlack, 0)
		}
		y += chartH + 1
	}

	ctx.Renderer.WriteString(fmt.Sprintf(" High: %.2f  Low: %.2f", maxP, minP), r.X, r.Y+r.Height-1, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
}

func (g *RealStockCandle) HandleEvent(e mofu.Event) mofu.Cmd { return nil }

type RealDotPlot struct {
	Base
	Title    string
	Series   []DotSeries
	MaxX     int
	mu       sync.RWMutex
}

type DotSeries struct {
	Label string
	Points []DotPoint
	Color mofu.Color
}

type DotPoint struct {
	X float64
	Y float64
}

func NewRealDotPlot(id string) *RealDotPlot {
	return &RealDotPlot{Base: *NewBase(id), MaxX: 100}
}

func (g *RealDotPlot) SetSeries(series []DotSeries) {
	g.mu.Lock()
	g.Series = series
	g.mu.Unlock()
}

func (g *RealDotPlot) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	if g.Title != "" {
		ctx.Renderer.WriteString(g.Title, r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
		y++
	}

	chartW := r.Width - 4
	chartH := r.Height - 4
	if chartW < 5 || chartH < 3 {
		return
	}

	grid := make([][]rune, chartH)
	for i := range grid {
		grid[i] = make([]rune, chartW)
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}

	for ax := 0; ax < chartW; ax++ {
		grid[chartH-1][ax] = '─'
	}
	for ay := 0; ay < chartH; ay++ {
		grid[ay][0] = '│'
	}
	grid[chartH-1][0] = '└'

	dotChars := []rune("●○◆◇★☆")
	for si, series := range g.Series {
		for _, pt := range series.Points {
			px := int(pt.X / float64(g.MaxX) * float64(chartW-1))
			py := int(pt.Y / float64(g.MaxX) * float64(chartH-2))
			py = chartH - 2 - py
			if px > 0 && px < chartW && py >= 0 && py < chartH-1 {
				grid[py][px] = dotChars[si%len(dotChars)]
			}
		}
	}

	for i, row := range grid {
		ctx.Renderer.WriteString(string(row), r.X+2, y+i, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
	}

	y2 := y + chartH + 1
	for si, series := range g.Series {
		if y2 >= r.Y+r.Height {
			break
		}
		ctx.Renderer.WriteString(fmt.Sprintf(" %c %s", dotChars[si%len(dotChars)], series.Label), r.X, y2, series.Color, mofu.ColorBlack, 0)
		y2++
	}
}

func (g *RealDotPlot) HandleEvent(e mofu.Event) mofu.Cmd { return nil }
