package main

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/xanstomper/mofu"
)

type Stock struct {
	Symbol string
	Name   string
	Price  float64
	Open   float64
	High   float64
	Low    float64
	Volume int64
	Change float64
	History []float64
}

type StockTracker struct {
	mofu.Minimal
	stocks    []Stock
	selected  int
	showDetail bool
	width     int
	height    int
	tick      int
}

func NewStockTracker() *StockTracker {
	stocks := []Stock{
		{Symbol: "AAPL", Name: "Apple Inc.", Price: 198.50, Open: 195.20, Volume: 52_000_000},
		{Symbol: "GOOGL", Name: "Alphabet Inc.", Price: 175.80, Open: 174.10, Volume: 18_000_000},
		{Symbol: "MSFT", Name: "Microsoft Corp.", Price: 445.30, Open: 442.50, Volume: 22_000_000},
		{Symbol: "AMZN", Name: "Amazon.com Inc.", Price: 185.90, Open: 183.40, Volume: 35_000_000},
		{Symbol: "NVDA", Name: "NVIDIA Corp.", Price: 125.60, Open: 120.30, Volume: 85_000_000},
		{Symbol: "TSLA", Name: "Tesla Inc.", Price: 245.70, Open: 250.10, Volume: 45_000_000},
		{Symbol: "META", Name: "Meta Platforms", Price: 505.20, Open: 501.80, Volume: 15_000_000},
		{Symbol: "BRK.B", Name: "Berkshire Hathaway", Price: 435.80, Open: 433.20, Volume: 3_000_000},
	}

	for i := range stocks {
		stocks[i].High = stocks[i].Price + rand.Float64()*3
		stocks[i].Low = stocks[i].Price - rand.Float64()*3
		stocks[i].Change = stocks[i].Price - stocks[i].Open

		history := make([]float64, 30)
		p := stocks[i].Open
		for j := range history {
			p += (rand.Float64() - 0.48) * 2
			history[j] = p
		}
		history[29] = stocks[i].Price
		stocks[i].History = history
	}

	return &StockTracker{stocks: stocks}
}

func (s *StockTracker) simulateTick() {
	for i := range s.stocks {
		delta := (rand.Float64() - 0.48) * 0.5
		s.stocks[i].Price += delta
		s.stocks[i].Price = math.Max(s.stocks[i].Price, 1)
		s.stocks[i].Change = s.stocks[i].Price - s.stocks[i].Open
		s.stocks[i].High = math.Max(s.stocks[i].High, s.stocks[i].Price)
		s.stocks[i].Low = math.Min(s.stocks[i].Low, s.stocks[i].Price)
		s.stocks[i].Volume += int64(rand.Intn(100000))

		s.stocks[i].History = append(s.stocks[i].History[1:], s.stocks[i].Price)
	}
	s.tick++
}

func (s *StockTracker) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	s.width = r.Width
	s.height = r.Height

	// Header
	title := " Stock Tracker"
	ctx.Renderer.WriteString(title, r.X, r.Y, mofu.Hex("f9e2af"), mofu.ColorBlack, mofu.AttrBold)

	timestamp := time.Now().Format("15:04:05")
	ctx.Renderer.WriteString(fmt.Sprintf(" Tick: %d  %s ", s.tick, timestamp), r.X+r.Width-len(timestamp)-20, r.Y, mofu.Hex("6c7086"), mofu.ColorBlack, 0)

	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, r.Y+1, mofu.Hex("444444"), mofu.ColorBlack, 0)

	// Table header
	y := r.Y + 2
	header := fmt.Sprintf(" %-7s %-20s %10s %10s %8s", "Symbol", "Name", "Price", "Change", "Vol(M)")
	ctx.Renderer.WriteString(header, r.X, y, mofu.Hex("89b4fa"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	// Stocks
	for i, stock := range s.stocks {
		if y >= r.Y+r.Height-4 {
			break
		}

		changePct := stock.Change / stock.Open * 100
		changeStr := fmt.Sprintf("%+.2f (%.2f%%)", stock.Change, changePct)
		volM := float64(stock.Volume) / 1_000_000

		line := fmt.Sprintf(" %-7s %-20s %10.2f %10s %8.1f",
			stock.Symbol, stock.Name, stock.Price, changeStr, volM)

		color := mofu.Hex("cdd6f4")
		if stock.Change > 0 {
			color = mofu.Hex("a6e3a1")
		} else if stock.Change < 0 {
			color = mofu.Hex("f38ba8")
		}

		if i == s.selected {
			ctx.Renderer.WriteString(strings.Repeat(" ", r.Width-2), r.X+1, y, mofu.ColorWhite, mofu.Hex("2a2a3e"), 0)
		}
		ctx.Renderer.WriteString(line, r.X, y, color, mofu.ColorBlack, 0)
		y++
	}

	// Detail panel
	if s.showDetail && s.selected < len(s.stocks) {
		y++
		stock := s.stocks[s.selected]
		ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
		y++

		ctx.Renderer.WriteString(fmt.Sprintf(" %s — %s", stock.Symbol, stock.Name), r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
		y++

		ctx.Renderer.WriteString(fmt.Sprintf("  Price: $%.2f  Open: $%.2f  High: $%.2f  Low: $%.2f", stock.Price, stock.Open, stock.High, stock.Low), r.X, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
		y++

		// Sparkline
		if len(stock.History) > 0 && y < r.Y+r.Height-1 {
			minP, maxP := stock.History[0], stock.History[0]
			for _, p := range stock.History {
				if p < minP {
					minP = p
				}
				if p > maxP {
					maxP = p
				}
			}
			range_ := maxP - minP
			if range_ == 0 {
				range_ = 1
			}

			sparkW := r.Width - 4
			if sparkW > len(stock.History) {
				sparkW = len(stock.History)
			}
			blocks := []rune(" ▁▂▃▄▅▆▇█")
			spark := ""
			step := len(stock.History) / sparkW
			if step < 1 {
				step = 1
			}
			for j := 0; j < sparkW; j++ {
				idx := j * step
				if idx >= len(stock.History) {
					break
				}
				pct := (stock.History[idx] - minP) / range_
				blockIdx := int(pct * float64(len(blocks)-1))
				if blockIdx >= len(blocks) {
					blockIdx = len(blocks) - 1
				}
				spark += string(blocks[blockIdx])
			}

			sparkColor := mofu.Hex("a6e3a1")
			if stock.Change < 0 {
				sparkColor = mofu.Hex("f38ba8")
			}
			ctx.Renderer.WriteString("  "+spark, r.X, y, sparkColor, mofu.ColorBlack, 0)
		}
	}

	// Status
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, r.Y+r.Height-2, mofu.Hex("444444"), mofu.ColorBlack, 0)
	status := " q:quit ↑↓:select Enter:detail"
	ctx.Renderer.WriteString(status, r.X, r.Y+r.Height-1, mofu.Hex("585b70"), mofu.Hex("1e1e2e"), 0)
}

func (s *StockTracker) HandleEvent(event mofu.Event) mofu.Cmd {
	if event.Type != mofu.EventKeyPress {
		return nil
	}
	ke := event.Data.(mofu.KeyEvent)

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()
	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		if s.selected < len(s.stocks)-1 {
			s.selected++
		}
	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if s.selected > 0 {
			s.selected--
		}
	case ke.Key == mofu.KeyEnter:
		s.showDetail = !s.showDetail
	}
	return nil
}

func main() {
	app := NewStockTracker()
	if err := mofu.Run(app); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
