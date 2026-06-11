package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/xanstomper/mofu"
)

// Calculator — a functional calculator example.

type Calculator struct {
	mofu.Minimal
	display   string
	operand   float64
	operator  string
	newNumber bool
	width     int
	height    int
}

func NewCalculator() *Calculator {
	return &Calculator{display: "0", newNumber: true}
}

func (c *Calculator) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	c.width = r.Width
	c.height = r.Height

	titleStyle := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
	ctx.Renderer.WriteString(" Calculator", r.X, r.Y, titleStyle.Foreground, titleStyle.Background, titleStyle.Attrs)

	sep := strings.Repeat("─", r.Width-2)
	ctx.Renderer.WriteString(sep, r.X+1, r.Y+1, mofu.Hex("444444"), mofu.ColorBlack, 0)

	// Display
	displayStyle := mofu.DefaultStyle().Fg(mofu.Hex("a6e3a1")).WithAttrs(mofu.AttrBold)
	ctx.Renderer.WriteString(fmt.Sprintf("%20s", c.display), r.X+2, r.Y+3, displayStyle.Foreground, displayStyle.Background, displayStyle.Attrs)

	// Buttons
	buttons := []string{
		"7 8 9 /",
		"4 5 6 *",
		"1 2 3 -",
		"0 . = +",
	}

	y := r.Y + 5
	for _, row := range buttons {
		cols := strings.Split(row, " ")
		x := r.X + 2
		for _, col := range cols {
			style := mofu.DefaultStyle().Fg(mofu.Hex("e0e0e0")).Bg(mofu.Hex("333333"))
			if col == "=" || col == "+" || col == "-" || col == "*" || col == "/" {
				style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).Bg(mofu.Hex("1a1a2e"))
			}
			ctx.Renderer.WriteString(fmt.Sprintf(" %s ", col), x, y, style.Foreground, style.Background, style.Attrs)
			x += 4
		}
		y++
	}

	// Status
	ctx.Renderer.WriteString(" 1-9: Number  +−×÷: Operator  =: Calculate  c: Clear  q: Quit", r.X, r.Y+r.Height-1, mofu.Hex("666666"), mofu.ColorBlack, 0)
}

func (c *Calculator) HandleEvent(event mofu.Event) mofu.Cmd {
	if event.Type != mofu.EventKeyPress {
		return nil
	}
	ke := event.Data.(mofu.KeyEvent)

	if ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && (ke.Runes[0] == 'q' || ke.Runes[0] == 'Q')) {
		return mofu.QuitCmd()
	}

	if ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && (ke.Runes[0] == 'c' || ke.Runes[0] == 'C')) {
		c.display = "0"
		c.operand = 0
		c.operator = ""
		c.newNumber = true
		return nil
	}

	for _, r := range ke.Runes {
		switch {
		case r >= '0' && r <= '9':
			if c.newNumber {
				c.display = string(r)
				c.newNumber = false
			} else {
				c.display += string(r)
			}
		case r == '.':
			if !strings.Contains(c.display, ".") {
				c.display += "."
				c.newNumber = false
			}
		case r == '+' || r == '-' || r == '*' || r == '/':
			c.calculate()
			c.operator = string(r)
			c.newNumber = true
		case r == '=':
			c.calculate()
			c.operator = ""
			c.newNumber = true
		}
	}

	return nil
}

func (c *Calculator) calculate() {
	if c.operator == "" {
		return
	}
	current, _ := strconv.ParseFloat(c.display, 64)
	var result float64

	switch c.operator {
	case "+":
		result = c.operand + current
	case "-":
		result = c.operand - current
	case "*":
		result = c.operand * current
	case "/":
		if current != 0 {
			result = c.operand / current
		} else {
			c.display = "Error"
			return
		}
	}

	c.display = fmt.Sprintf("%g", result)
	c.operand = result
}

func main() {
	app := NewCalculator()
	if err := mofu.Run(app); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
