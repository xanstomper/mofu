package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/xanstomper/mofu"
)

type Transaction struct {
	Category string
	Amount   float64
	Type     string
	Date     string
}

type Budget struct {
	mofu.Minimal
	income       float64
	expenses     []Transaction
	categories   map[string]float64
	budgets      map[string]float64
	selected     int
	showCategory string
	width        int
	height       int
}

func NewBudget() *Budget {
	b := &Budget{
		income:   5000.00,
		categories: make(map[string]float64),
		budgets: map[string]float64{
			"Rent":       1500,
			"Food":       600,
			"Transport":  300,
			"Utilities":  200,
			"Entertainment": 200,
			"Savings":    1000,
			"Other":      200,
		},
	}

	b.expenses = []Transaction{
		{Category: "Rent", Amount: 1500, Type: "fixed", Date: "2026-06-01"},
		{Category: "Food", Amount: 85.50, Type: "variable", Date: "2026-06-02"},
		{Category: "Food", Amount: 42.30, Type: "variable", Date: "2026-06-03"},
		{Category: "Transport", Amount: 45.00, Type: "variable", Date: "2026-06-04"},
		{Category: "Utilities", Amount: 120.00, Type: "fixed", Date: "2026-06-05"},
		{Category: "Entertainment", Amount: 35.00, Type: "variable", Date: "2026-06-06"},
		{Category: "Food", Amount: 67.80, Type: "variable", Date: "2026-06-07"},
		{Category: "Food", Amount: 55.20, Type: "variable", Date: "2026-06-08"},
		{Category: "Transport", Amount: 30.00, Type: "variable", Date: "2026-06-09"},
		{Category: "Savings", Amount: 500.00, Type: "savings", Date: "2026-06-10"},
	}

	for _, e := range b.expenses {
		b.categories[e.Category] += e.Amount
	}

	return b
}

func (b *Budget) totalExpenses() float64 {
	total := 0.0
	for _, e := range b.expenses {
		total += e.Amount
	}
	return total
}

func (b *Budget) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	b.width = r.Width
	b.height = r.Height

	y := r.Y

	titleStyle := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
	ctx.Renderer.WriteString(" Budget Tracker", r.X, y, titleStyle.Foreground, titleStyle.Background, titleStyle.Attrs)
	y++

	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	totalExp := b.totalExpenses()
	remaining := b.income - totalExp

	ctx.Renderer.WriteString(fmt.Sprintf("  Income:      $%.2f", b.income), r.X, y, mofu.Hex("a6e3a1"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(fmt.Sprintf("  Expenses:    $%.2f", totalExp), r.X, y, mofu.Hex("f38ba8"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(fmt.Sprintf("  Remaining:   $%.2f", remaining), r.X, y, mofu.Hex("f9e2af"), mofu.ColorBlack, mofu.AttrBold)
	y++

	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	// Budget bars
	categories := []string{"Rent", "Food", "Transport", "Utilities", "Entertainment", "Savings", "Other"}

	for i, cat := range categories {
		if y >= r.Y+r.Height-5 {
			break
		}

		spent := b.categories[cat]
		budget := b.budgets[cat]
		pct := spent / budget * 100

		barW := r.Width/2 - 12
		if barW < 5 {
			barW = 5
		}
		filled := int(pct / 100 * float64(barW))
		if filled > barW {
			filled = barW
		}

		barColor := mofu.Hex("a6e3a1")
		if pct > 80 {
			barColor = mofu.Hex("fab387")
		}
		if pct > 100 {
			barColor = mofu.Hex("f38ba8")
		}

		bar := strings.Repeat("█", filled) + strings.Repeat("░", barW-filled)
		label := fmt.Sprintf("  %-14s", cat)
		if i == b.selected {
			ctx.Renderer.WriteString(strings.Repeat(" ", r.Width-2), r.X+1, y, mofu.ColorWhite, mofu.Hex("2a2a3e"), 0)
		}
		ctx.Renderer.WriteString(label+bar+fmt.Sprintf(" $%.0f/$%.0f", spent, budget), r.X, y, barColor, mofu.ColorBlack, 0)
		y++
	}

	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	// Recent transactions
	ctx.Renderer.WriteString(" Recent Transactions", r.X, y, mofu.Hex("89b4fa"), mofu.ColorBlack, mofu.AttrBold)
	y++

	start := len(b.expenses) - 5
	if start < 0 {
		start = 0
	}

	for i := start; i < len(b.expenses); i++ {
		if y >= r.Y+r.Height-2 {
			break
		}
		tx := b.expenses[i]
		line := fmt.Sprintf("  %s  %-14s  $%8.2f  %s", tx.Date, tx.Category, tx.Amount, tx.Type)
		if len(line) > r.Width-2 {
			line = line[:r.Width-2]
		}
		ctx.Renderer.WriteString(line, r.X, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
		y++
	}

	// Status
	status := " j/k:select d:add e:edit q:quit"
	ctx.Renderer.WriteString(status, r.X, r.Y+r.Height-1, mofu.Hex("585b70"), mofu.Hex("1e1e2e"), 0)
}

func (b *Budget) HandleEvent(event mofu.Event) mofu.Cmd {
	if event.Type != mofu.EventKeyPress {
		return nil
	}
	ke := event.Data.(mofu.KeyEvent)

	categories := []string{"Rent", "Food", "Transport", "Utilities", "Entertainment", "Savings", "Other"}

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()

	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		if b.selected < len(categories)-1 {
			b.selected++
		}

	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if b.selected > 0 {
			b.selected--
		}

	case ke.Key == mofu.KeyEnter:
		if b.selected < len(categories) {
			b.showCategory = categories[b.selected]
		}
	}

	return nil
}

func main() {
	app := NewBudget()
	if err := mofu.Run(app); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
