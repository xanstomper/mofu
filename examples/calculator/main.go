package main

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xanstomper/mofu"
)

// Calculator — sleek TUI calculator with splash, animations, history.

type Calculator struct {
	mofu.Minimal
	display    string
	operand    float64
	operator   string
	newNumber  bool
	width      int
	height     int
	pressed    string
	pressTime  time.Time
	history    []string
	showSplash bool
	splashTick int
	resultAnim float64
	animTarget float64
	animating  bool
	mu         sync.Mutex
}

func NewCalculator() *Calculator {
	c := &Calculator{
		display:    "0",
		newNumber:  true,
		showSplash: true,
	}
	go c.splashAnimation()
	return c
}

func (c *Calculator) splashAnimation() {
	for i := 0; i < 40; i++ {
		c.mu.Lock()
		c.splashTick = i
		c.mu.Unlock()
		time.Sleep(30 * time.Millisecond)
	}
	c.mu.Lock()
	c.showSplash = false
	c.mu.Unlock()
}

func (c *Calculator) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	c.mu.Lock()
	c.width = r.Width
	c.height = r.Height
	splash := c.showSplash
	tick := c.splashTick
	c.mu.Unlock()

	if splash {
		c.renderSplash(ctx, r, tick)
		return
	}
	c.renderApp(ctx, r)
}

func (c *Calculator) renderSplash(ctx *mofu.RenderContext, r mofu.Rect, tick int) {
	bg := mofu.Hex("0a0a0f")
	accent := mofu.Hex("ff69b4")

	// Center the splash
	cx := r.X + r.Width/2
	cy := r.Y + r.Height/2

	// Logo - builds up character by character
	logo := "╔═══════════════════╗"
	logoLine2 := "║   M O F U   C A L C ║"
	logoLine3 := "╚═══════════════════╝"

	if tick > 5 {
		if tick-5 < len(logo) {
			ctx.Renderer.WriteString(logo[:tick-5], cx-len(logo)/2, cy-1, accent, bg, mofu.AttrBold)
		} else {
			ctx.Renderer.WriteString(logo, cx-len(logo)/2, cy-1, accent, bg, mofu.AttrBold)
		}
	}

	if tick > 15 {
		if tick-15 < len(logoLine2) {
			ctx.Renderer.WriteString(logoLine2[:tick-15], cx-len(logoLine2)/2, cy, mofu.Hex("cdd6f4"), bg, 0)
		} else {
			ctx.Renderer.WriteString(logoLine2, cx-len(logoLine2)/2, cy, mofu.Hex("cdd6f4"), bg, 0)
		}
	}

	if tick > 25 {
		if tick-25 < len(logoLine3) {
			ctx.Renderer.WriteString(logoLine3[:tick-25], cx-len(logoLine3)/2, cy+1, accent, bg, mofu.AttrBold)
		} else {
			ctx.Renderer.WriteString(logoLine3, cx-len(logoLine3)/2, cy+1, accent, bg, mofu.AttrBold)
		}
	}

	// Loading bar
	if tick > 30 {
		barW := 30
		filled := int(float64(tick-30) / 10.0 * float64(barW))
		if filled > barW {
			filled = barW
		}
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barW-filled)
		ctx.Renderer.WriteString(bar, cx-barW/2, cy+3, mofu.Hex("585b70"), bg, 0)
	}
}

func (c *Calculator) renderApp(ctx *mofu.RenderContext, r mofu.Rect) {
	bg := mofu.Hex("0a0a0f")
	panelBg := mofu.Hex("111119")
	btnBg := mofu.Hex("1a1a2e")
	opBg := mofu.Hex("1a1a35")
	accent := mofu.Hex("ff69b4")
	dim := mofu.Hex("45475a")
	text := mofu.Hex("cdd6f4")
	green := mofu.Hex("a6e3a1")

	// Main container with rounded border
	for x := r.X; x < r.X+r.Width; x++ {
		ctx.Renderer.WriteString("─", x, r.Y, dim, bg, 0)
		ctx.Renderer.WriteString("─", x, r.Y+r.Height-1, dim, bg, 0)
	}
	for y := r.Y; y < r.Y+r.Height; y++ {
		ctx.Renderer.WriteString("│", r.X, y, dim, bg, 0)
		ctx.Renderer.WriteString("│", r.X+r.Width-1, y, dim, bg, 0)
	}
	ctx.Renderer.WriteString("╭", r.X, r.Y, accent, bg, mofu.AttrBold)
	ctx.Renderer.WriteString("╮", r.X+r.Width-1, r.Y, accent, bg, mofu.AttrBold)
	ctx.Renderer.WriteString("╰", r.X, r.Y+r.Height-1, accent, bg, mofu.AttrBold)
	ctx.Renderer.WriteString("╯", r.X+r.Width-1, r.Y+r.Height-1, accent, bg, mofu.AttrBold)

	// Title
	title := " MOFU CALC"
	ctx.Renderer.WriteString(title, r.X+2, r.Y, accent, bg, mofu.AttrBold)
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-4), r.X+1, r.Y+1, dim, bg, 0)

	// Display area
	dispY := r.Y + 2
	for dy := 0; dy < 4; dy++ {
		ctx.Renderer.WriteString(strings.Repeat(" ", r.Width-2), r.X+1, dispY+dy, panelBg, panelBg, 0)
	}

	c.mu.Lock()
	display := c.display
	op := c.operator
	history := c.history
	pressTime := c.pressTime
	c.mu.Unlock()

	// History (show last 2 operations)
	if len(history) > 0 {
		lastOp := history[len(history)-1]
		if len(lastOp) > r.Width-4 {
			lastOp = lastOp[:r.Width-7] + "..."
		}
		ctx.Renderer.WriteString(lastOp, r.X+3, dispY+1, dim, panelBg, 0)
	}

	// Main display with glow effect
	dispText := display
	if len(dispText) > r.Width-4 {
		dispText = dispText[len(dispText)-(r.Width-7):] + "..."
	}

	// Animate result
	if time.Since(pressTime) < 300*time.Millisecond {
		ctx.Renderer.WriteString(fmt.Sprintf("%*s", r.Width-4, dispText), r.X+2, dispY+2, green, panelBg, mofu.AttrBold)
	} else {
		ctx.Renderer.WriteString(fmt.Sprintf("%*s", r.Width-4, dispText), r.X+2, dispY+2, text, panelBg, mofu.AttrBold)
	}

	// Operator indicator
	if op != "" {
		opName := map[string]string{"+": "ADD", "-": "SUB", "*": "MUL", "/": "DIV"}
		ctx.Renderer.WriteString(fmt.Sprintf("  %s", opName[op]), r.X+2, dispY+3, accent, panelBg, 0)
	}

	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-4), r.X+1, r.Y+6, dim, bg, 0)

	// Button grid
	buttons := [][]string{
		{"C", "±", "%", "÷"},
		{"7", "8", "9", "×"},
		{"4", "5", "6", "−"},
		{"1", "2", "3", "+"},
		{" 0  ", ".", "="},
	}

	startY := r.Y + 7
	btnH := 3
	btnW := (r.Width - 6) / 4

	for row, btnRow := range buttons {
		by := startY + row*btnH
		bx := r.X + 2

		for _, label := range btnRow {
			isOp := label == "÷" || label == "×" || label == "−" || label == "+" || label == "="
			isSpecial := label == "C" || label == "±" || label == "%"

			// Button background
			bgColor := btnBg
			textColor := text
			if isOp {
				bgColor = opBg
				textColor = accent
			}
			if isSpecial {
				textColor = dim
			}

			// Press animation
			if label == c.pressed && time.Since(pressTime) < 100*time.Millisecond {
				bgColor = accent
				textColor = bg
			}

			// Draw button
			for dy := 0; dy < btnH; dy++ {
				line := strings.Repeat(" ", btnW)
				ctx.Renderer.WriteString(line, bx, by+dy, bgColor, bgColor, 0)
			}

			// Button label (centered)
			labelText := label
			if len(labelText) > btnW-2 {
				labelText = labelText[:btnW-2]
			}
			lx := bx + (btnW-len(labelText))/2
			ly := by + btnH/2
			ctx.Renderer.WriteString(labelText, lx, ly, textColor, bgColor, mofu.AttrBold)

			bx += btnW + 1
		}
	}

	// Bottom bar
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-4), r.X+1, r.Y+r.Height-2, dim, bg, 0)
	ctx.Renderer.WriteString(" 1-9:Num +−×÷:Op =:Calc c:Clear q:Quit", r.X+2, r.Y+r.Height-1, dim, bg, 0)
}

func (c *Calculator) HandleEvent(event mofu.Event) mofu.Cmd {
	if event.Type != mofu.EventKeyPress {
		return nil
	}
	ke := event.Data.(mofu.KeyEvent)

	c.mu.Lock()
	c.pressed = ""
	c.mu.Unlock()

	if ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && (ke.Runes[0] == 'q' || ke.Runes[0] == 'Q')) {
		return mofu.QuitCmd()
	}

	if ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && (ke.Runes[0] == 'c' || ke.Runes[0] == 'C')) {
		c.mu.Lock()
		c.display = "0"
		c.operand = 0
		c.operator = ""
		c.newNumber = true
		c.pressed = "C"
		c.pressTime = time.Now()
		c.mu.Unlock()
		return nil
	}

	for _, r := range ke.Runes {
		c.mu.Lock()
		switch {
		case r >= '0' && r <= '9':
			c.pressed = string(r)
			c.pressTime = time.Now()
			if c.newNumber {
				c.display = string(r)
				c.newNumber = false
			} else {
				c.display += string(r)
			}
		case r == '.':
			c.pressed = "."
			c.pressTime = time.Now()
			if !strings.Contains(c.display, ".") {
				c.display += "."
				c.newNumber = false
			}
		case r == '+':
			c.pressed = "+"
			c.pressTime = time.Now()
			c.calculate()
			c.operator = "+"
			c.newNumber = true
		case r == '-':
			c.pressed = "−"
			c.pressTime = time.Now()
			c.calculate()
			c.operator = "-"
			c.newNumber = true
		case r == '*':
			c.pressed = "×"
			c.pressTime = time.Now()
			c.calculate()
			c.operator = "*"
			c.newNumber = true
		case r == '/':
			c.pressed = "÷"
			c.pressTime = time.Now()
			c.calculate()
			c.operator = "/"
			c.newNumber = true
		case r == '=':
			c.pressed = "="
			c.pressTime = time.Now()
			c.calculate()
			c.operator = ""
			c.newNumber = true
		case r == '%':
			c.pressed = "%"
			c.pressTime = time.Now()
			if val, err := strconv.ParseFloat(c.display, 64); err == nil {
				c.display = fmt.Sprintf("%g", val/100)
			}
		}
		c.mu.Unlock()
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
			c.history = append(c.history, fmt.Sprintf("%g ÷ 0 = Error", c.operand))
			return
		}
	}

	c.history = append(c.history, fmt.Sprintf("%g %s %g = %g", c.operand, c.operator, current, result))
	if len(c.history) > 5 {
		c.history = c.history[len(c.history)-5:]
	}

	c.display = fmt.Sprintf("%g", result)
	c.operand = result

	// Clean up -0
	if c.display == "-0" {
		c.display = "0"
	}
	if math.IsNaN(result) || math.IsInf(result, 0) {
		c.display = "Error"
	}
}

func main() {
	app := NewCalculator()
	if err := mofu.Run(app); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
