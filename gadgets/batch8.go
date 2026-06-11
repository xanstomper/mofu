package gadgets

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/xanstomper/mofu"
)

// =========================================================================
// BATCH 8: Text Tools & Interactive Gadgets (10 gadgets)
// =========================================================================

type RealWordCounter struct {
	Base
	Text     string
	Words    int
	Lines    int
	Chars    int
	Spaces   int
	Digits   int
	mu       sync.RWMutex
}

func NewRealWordCounter(id string) *RealWordCounter {
	return &RealWordCounter{Base: *NewBase(id)}
}

func (g *RealWordCounter) SetText(text string) {
	g.mu.Lock()
	g.Text = text
	g.Words = len(strings.Fields(text))
	g.Lines = len(strings.Split(text, "\n"))
	g.Chars = len(text)
	g.Spaces = strings.Count(text, " ")
	g.Digits = 0
	for _, r := range text {
		if unicode.IsDigit(r) {
			g.Digits++
		}
	}
	g.mu.Unlock()
}

func (g *RealWordCounter) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(" Word Counter", r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	stats := []struct {
		label string
		value int
	}{
		{"Words", g.Words},
		{"Lines", g.Lines},
		{"Characters", g.Chars},
		{"Spaces", g.Spaces},
		{"Digits", g.Digits},
	}

	for _, s := range stats {
		if y >= r.Y+r.Height-2 {
			break
		}
		barW := r.Width - 25
		filled := 0
		if g.Chars > 0 {
			filled = s.value * barW / g.Chars
		}
		if filled > barW {
			filled = barW
		}
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barW-filled)
		ctx.Renderer.WriteString(fmt.Sprintf(" %-12s %s %d", s.label, bar, s.value), r.X, y, mofu.Hex("89b4fa"), mofu.ColorBlack, 0)
		y++
	}
}

func (g *RealWordCounter) HandleEvent(e mofu.Event) mofu.Cmd { return nil }

type RealTextTransform struct {
	Base
	Text       string
	Mode       int
_modes      []string
	mu         sync.RWMutex
	OnTransform func(result string)
}

func NewRealTextTransform(id string) *RealTextTransform {
	return &RealTextTransform{
		Base:  *NewBase(id),
		_modes: []string{"UPPER", "lower", "Title", "Reversed", "CamelCase", "snake_case", "kebab-case", "ROT13"},
	}
}

func (g *RealTextTransform) Transform() string {
	g.mu.Lock()
	defer g.mu.Unlock()

	switch g.Mode {
	case 0:
		return strings.ToUpper(g.Text)
	case 1:
		return strings.ToLower(g.Text)
	case 2:
		return strings.Title(strings.ToLower(g.Text))
	case 3:
		runes := []rune(g.Text)
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		return string(runes)
	case 4:
		words := strings.Fields(g.Text)
		result := ""
		for i, w := range words {
			if i == 0 {
				result += strings.ToLower(w)
			} else {
				result += strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
			}
		}
		return result
	case 5:
		return strings.ReplaceAll(strings.ToLower(g.Text), " ", "_")
	case 6:
		return strings.ReplaceAll(strings.ToLower(g.Text), " ", "-")
	case 7:
		return strings.Map(func(r rune) rune {
			if r >= 'a' && r <= 'z' {
				return (r-'a'+13)%26 + 'a'
			}
			if r >= 'A' && r <= 'Z' {
				return (r-'A'+13)%26 + 'A'
			}
			return r
		}, g.Text)
	}
	return g.Text
}

func (g *RealTextTransform) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(" Text Transform", r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++

	modeName := "UNKNOWN"
	if g.Mode >= 0 && g.Mode < len(g._modes) {
		modeName = g._modes[g.Mode]
	}
	ctx.Renderer.WriteString(fmt.Sprintf(" Mode: [%s]", modeName), r.X, y, mofu.Hex("89b4fa"), mofu.ColorBlack, 0)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	if g.Text != "" {
		ctx.Renderer.WriteString(" Input:", r.X, y, mofu.Hex("6c7086"), mofu.ColorBlack, 0)
		y++
		input := g.Text
		if len(input) > r.Width-2 {
			input = input[:r.Width-5] + "..."
		}
		ctx.Renderer.WriteString(" "+input, r.X, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
		y++
		y++

		result := g.Transform()
		ctx.Renderer.WriteString(" Output:", r.X, y, mofu.Hex("a6e3a1"), mofu.ColorBlack, 0)
		y++
		if len(result) > r.Width-2 {
			result = result[:r.Width-5] + "..."
		}
		ctx.Renderer.WriteString(" "+result, r.X, y, mofu.Hex("a6e3a1"), mofu.ColorBlack, mofu.AttrBold)
		y++
	} else {
		ctx.Renderer.WriteString(" Type text to transform...", r.X, y, mofu.Hex("585b70"), mofu.ColorBlack, 0)
		y++
	}

	y++
	ctx.Renderer.WriteString(" Tab:cycle mode Enter:copy q:quit", r.X, r.Y+r.Height-1, mofu.Hex("585b70"), mofu.Hex("1e1e2e"), 0)
}

func (g *RealTextTransform) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	g.mu.Lock()
	defer g.mu.Unlock()

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()
	case ke.Key == mofu.KeyTab:
		g.Mode = (g.Mode + 1) % len(g._modes)
	case ke.Key == mofu.KeyBack && len(g.Text) > 0:
		g.Text = g.Text[:len(g.Text)-1]
	case ke.Key == mofu.KeyEnter:
		if g.OnTransform != nil {
			g.OnTransform(g.Transform())
		}
	default:
		if len(ke.Runes) > 0 {
			g.Text += string(ke.Runes)
		}
	}
	return nil
}

type RealColorPalette struct {
	Base
	Current  mofu.Color
	History  []mofu.Color
	mu       sync.RWMutex
	OnPick   func(c mofu.Color)
}

func NewRealColorPalette(id string) *RealColorPalette {
	return &RealColorPalette{Base: *NewBase(id)}
}

func (g *RealColorPalette) Pick(c mofu.Color) {
	g.mu.Lock()
	g.Current = c
	g.History = append([]mofu.Color{c}, g.History...)
	if len(g.History) > 20 {
		g.History = g.History[:20]
	}
	g.mu.Unlock()
}

func (g *RealColorPalette) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(" Color Palette", r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++

	ctx.Renderer.WriteString(fmt.Sprintf(" Current: R:%d G:%d B:%d", g.Current.R, g.Current.G, g.Current.B), r.X, y, g.Current, mofu.ColorBlack, 0)
	y++

	preview := strings.Repeat("████████", r.Width/8)
	ctx.Renderer.WriteString(preview, r.X, y, g.Current, mofu.ColorBlack, 0)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	if len(g.History) > 0 {
		ctx.Renderer.WriteString(" History:", r.X, y, mofu.Hex("6c7086"), mofu.ColorBlack, 0)
		y++
		x := r.X
		for i, c := range g.History {
			if x+2 >= r.X+r.Width {
				x = r.X
				y++
				if y >= r.Y+r.Height-2 {
					break
				}
			}
			ctx.Renderer.WriteString("██", x, y, c, mofu.ColorBlack, 0)
			x += 2
			_ = i
		}
	}
}

func (g *RealColorPalette) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)
	if ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q') {
		return mofu.QuitCmd()
	}
	return nil
}

type RealProgressBarAnimated struct {
	Base
	Value     float64
	Max       float64
	Label     string
	Animating bool
	frame     int
	mu        sync.RWMutex
}

func NewRealProgressBarAnimated(id, label string, max float64) *RealProgressBarAnimated {
	return &RealProgressBarAnimated{Base: *NewBase(id), Label: label, Max: max}
}

func (g *RealProgressBarAnimated) SetValue(v float64) {
	g.mu.Lock()
	g.Value = v
	g.mu.Unlock()
}

func (g *RealProgressBarAnimated) Tick() {
	g.mu.Lock()
	g.frame++
	g.mu.Unlock()
}

func (g *RealProgressBarAnimated) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	pct := 0.0
	if g.Max > 0 {
		pct = g.Value / g.Max * 100
	}

	barW := r.Width - 20
	filled := int(pct / 100 * float64(barW))

	spinners := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	frame := spinners[g.frame%len(spinners)]

	if g.Animating {
		ctx.Renderer.WriteString(fmt.Sprintf(" %s %s", frame, g.Label), r.X, r.Y, mofu.Hex("89b4fa"), mofu.ColorBlack, 0)
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barW-filled)
	color := mofu.Hex("89b4fa")
	if pct >= 100 {
		color = mofu.Hex("a6e3a1")
	} else if pct > 60 {
		color = mofu.Hex("f9e2af")
	} else if pct > 30 {
		color = mofu.Hex("fab387")
	}

	ctx.Renderer.WriteString(fmt.Sprintf(" %s %s %5.1f%%", g.Label, bar, pct), r.X, r.Y, color, mofu.ColorBlack, 0)
}

func (g *RealProgressBarAnimated) HandleEvent(e mofu.Event) mofu.Cmd { return nil }

type RealGrepViewer struct {
	Base
	Pattern  string
	Results  []GrepResult
	Source   []string
	Selected int
	mu       sync.RWMutex
	OnSelect func(idx int, r GrepResult)
}

type GrepResult struct {
	LineNum int
	Line    string
	Match   string
}

func NewRealGrepViewer(id string) *RealGrepViewer {
	return &RealGrepViewer{Base: *NewBase(id)}
}

func (g *RealGrepViewer) Search(pattern string, lines []string) {
	g.mu.Lock()
	g.Pattern = pattern
	g.Source = lines
	g.Results = nil
	g.Selected = 0

	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), strings.ToLower(pattern)) {
			g.Results = append(g.Results, GrepResult{LineNum: i + 1, Line: line, Match: pattern})
		}
	}
	g.mu.Unlock()
}

func (g *RealGrepViewer) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(fmt.Sprintf(" Grep: %s (%d matches)", g.Pattern, len(g.Results)), r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	for i, result := range g.Results {
		if y >= r.Y+r.Height-1 {
			break
		}

		style := mofu.DefaultStyle()
		if i == g.Selected {
			ctx.Renderer.WriteString(strings.Repeat(" ", r.Width-2), r.X+1, y, mofu.ColorWhite, mofu.Hex("2a2a3e"), 0)
			style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))
		}

		lineNum := fmt.Sprintf("%4d", result.LineNum)
		line := result.Line
		if len(line) > r.Width-8 {
			line = line[:r.Width-11] + "..."
		}

		ctx.Renderer.WriteString(fmt.Sprintf(" %s: %s", lineNum, line), r.X, y, style.Foreground, style.Background, style.Attrs)
		y++
	}

	if len(g.Results) == 0 && g.Pattern != "" {
		ctx.Renderer.WriteString(" No matches found", r.X, y, mofu.Hex("585b70"), mofu.ColorBlack, 0)
	}
}

func (g *RealGrepViewer) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	g.mu.Lock()
	defer g.mu.Unlock()

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()
	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		if g.Selected < len(g.Results)-1 {
			g.Selected++
		}
	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if g.Selected > 0 {
			g.Selected--
		}
	case ke.Key == mofu.KeyEnter && g.OnSelect != nil:
		if g.Selected < len(g.Results) {
			g.OnSelect(g.Selected, g.Results[g.Selected])
		}
	}
	return nil
}

type RealCRUDTable struct {
	Base
	Headers []string
	Rows    [][]string
	Selected int
	Editing  bool
	EditRow  int
	EditCol  int
	mu       sync.RWMutex
	OnUpdate func(row, col int, val string)
	OnDelete func(row int)
}

func NewRealCRUDTable(id string, headers []string) *RealCRUDTable {
	return &RealCRUDTable{Base: *NewBase(id), Headers: headers}
}

func (g *RealCRUDTable) AddRow(row []string) {
	g.mu.Lock()
	g.Rows = append(g.Rows, row)
	g.mu.Unlock()
}

func (g *RealCRUDTable) UpdateCell(row, col int, val string) {
	g.mu.Lock()
	if row >= 0 && row < len(g.Rows) && col >= 0 && col < len(g.Rows[row]) {
		g.Rows[row][col] = val
	}
	g.mu.Unlock()
}

func (g *RealCRUDTable) DeleteRow(idx int) {
	g.mu.Lock()
	if idx >= 0 && idx < len(g.Rows) {
		g.Rows = append(g.Rows[:idx], g.Rows[idx+1:]...)
		if g.Selected >= len(g.Rows) && g.Selected > 0 {
			g.Selected--
		}
	}
	g.mu.Unlock()
}

func (g *RealCRUDTable) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(fmt.Sprintf(" Data (%d rows)", len(g.Rows)), r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++

	colW := (r.Width - 2) / len(g.Headers)
	if colW < 5 {
		colW = 5
	}

	header := ""
	for _, h := range g.Headers {
		header += fmt.Sprintf(" %-*s", colW, h)
	}
	ctx.Renderer.WriteString(header, r.X, y, mofu.Hex("89b4fa"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	for i, row := range g.Rows {
		if y >= r.Y+r.Height-1 {
			break
		}

		if i == g.Selected {
			ctx.Renderer.WriteString(strings.Repeat(" ", r.Width-2), r.X+1, y, mofu.ColorWhite, mofu.Hex("2a2a3e"), 0)
		}

		line := ""
		for j, cell := range row {
			if j >= len(g.Headers) {
				break
			}
			if len(cell) > colW-1 {
				cell = cell[:colW-3] + ".."
			}
			line += fmt.Sprintf(" %-*s", colW, cell)
		}

		color := mofu.Hex("cdd6f4")
		if i == g.Selected {
			color = mofu.Hex("ff69b4")
		}
		ctx.Renderer.WriteString(line, r.X, y, color, mofu.ColorBlack, 0)
		y++
	}
}

func (g *RealCRUDTable) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	g.mu.Lock()
	defer g.mu.Unlock()

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()
	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		if g.Selected < len(g.Rows)-1 {
			g.Selected++
		}
	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if g.Selected > 0 {
			g.Selected--
		}
	case (len(ke.Runes) > 0 && ke.Runes[0] == 'd') && g.OnDelete != nil:
		g.OnDelete(g.Selected)
	}
	return nil
}

type RealProgressBarSteps struct {
	Base
	Steps      []string
	Current    int
	Completed  []bool
	mu         sync.RWMutex
}

func NewRealProgressBarSteps(id string, steps []string) *RealProgressBarSteps {
	completed := make([]bool, len(steps))
	return &RealProgressBarSteps{Base: *NewBase(id), Steps: steps, Completed: completed}
}

func (g *RealProgressBarSteps) Next() {
	g.mu.Lock()
	if g.Current < len(g.Steps)-1 {
		g.Completed[g.Current] = true
		g.Current++
	} else if g.Current == len(g.Steps)-1 {
		g.Completed[g.Current] = true
	}
	g.mu.Unlock()
}

func (g *RealProgressBarSteps) Reset() {
	g.mu.Lock()
	g.Current = 0
	g.Completed = make([]bool, len(g.Steps))
	g.mu.Unlock()
}

func (g *RealProgressBarSteps) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(" Steps", r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++

	doneCount := 0
	for _, c := range g.Completed {
		if c {
			doneCount++
		}
	}

	barW := r.Width - 8
	filled := doneCount * barW / len(g.Steps)
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barW-filled)
	ctx.Renderer.WriteString(fmt.Sprintf(" %s %d/%d", bar, doneCount, len(g.Steps)), r.X, y, mofu.Hex("89b4fa"), mofu.ColorBlack, 0)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	for i, step := range g.Steps {
		if y >= r.Y+r.Height-1 {
			break
		}

		icon := "○"
		color := mofu.Hex("585b70")
		if g.Completed[i] {
			icon = "●"
			color = mofu.Hex("a6e3a1")
		} else if i == g.Current {
			icon = "▶"
			color = mofu.Hex("f9e2af")
		}

		ctx.Renderer.WriteString(fmt.Sprintf(" %s %s", icon, step), r.X, y, color, mofu.ColorBlack, 0)
		y++
	}
}

func (g *RealProgressBarSteps) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()
	case ke.Key == mofu.KeyEnter || (len(ke.Runes) > 0 && ke.Runes[0] == 'n'):
		g.Next()
	case len(ke.Runes) > 0 && ke.Runes[0] == 'r':
		g.Reset()
	}
	return nil
}

type RealTypingTest struct {
	Base
	Text        string
	Typed       string
	StartTime   time.Time
	Correct     int
	Wrong       int
	Finished    bool
	mu          sync.RWMutex
	OnComplete  func(wpm, accuracy float64)
}

var typingTexts = []string{
	"the quick brown fox jumps over the lazy dog",
	"pack my box with five dozen liquor jugs",
	"how vexingly quick daft zebras jump",
	"the five boxing wizards jump quickly",
	"jackdaws love my big sphinx of quartz",
}

func NewRealTypingTest(id string) *RealTypingTest {
	g := &RealTypingTest{Base: *NewBase(id)}
	g.Text = typingTexts[rand.Intn(len(typingTexts))]
	return g
}

func (g *RealTypingTest) WPM() float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.StartTime.IsZero() || len(g.Typed) == 0 {
		return 0
	}
	elapsed := time.Since(g.StartTime).Minutes()
	if elapsed == 0 {
		return 0
	}
	return float64(g.Correct) / 5.0 / elapsed
}

func (g *RealTypingTest) Accuracy() float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	total := g.Correct + g.Wrong
	if total == 0 {
		return 0
	}
	return float64(g.Correct) / float64(total) * 100
}

func (g *RealTypingTest) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(" Typing Test", r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	textLine := ""
	for i, ch := range g.Text {
		if len(textLine) >= r.Width-2 {
			break
		}
		if i < len(g.Typed) {
			if g.Typed[i] == byte(ch) {
				textLine += string(ch)
			} else {
				textLine += "█"
			}
		} else if i == len(g.Typed) {
			textLine += "█"
		} else {
			textLine += string(ch)
		}
	}
	ctx.Renderer.WriteString(" "+textLine, r.X, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
	y += 2

	if !g.Finished {
		ctx.Renderer.WriteString(fmt.Sprintf(" WPM: %.0f  Accuracy: %.1f%%", g.WPM(), g.Accuracy()), r.X, y, mofu.Hex("89b4fa"), mofu.ColorBlack, 0)
	} else {
		ctx.Renderer.WriteString(fmt.Sprintf(" DONE! WPM: %.0f  Accuracy: %.1f%%", g.WPM(), g.Accuracy()), r.X, y, mofu.Hex("a6e3a1"), mofu.ColorBlack, mofu.AttrBold)
	}
}

func (g *RealTypingTest) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	g.mu.Lock()
	defer g.mu.Unlock()

	if g.Finished {
		if len(ke.Runes) > 0 && ke.Runes[0] == 'r' {
			g.Text = typingTexts[rand.Intn(len(typingTexts))]
			g.Typed = ""
			g.Correct = 0
			g.Wrong = 0
			g.Finished = false
			g.StartTime = time.Time{}
		}
		return nil
	}

	if ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q') {
		return mofu.QuitCmd()
	}

	if ke.Key == mofu.KeyBack && len(g.Typed) > 0 {
		g.Typed = g.Typed[:len(g.Typed)-1]
		return nil
	}

	if len(ke.Runes) > 0 && !ke.Ctrl && !ke.Alt {
		if g.StartTime.IsZero() {
			g.StartTime = time.Now()
		}

		idx := len(g.Typed)
		if idx < len(g.Text) {
			g.Typed += string(ke.Runes)
			if byte(ke.Runes[0]) == g.Text[idx] {
				g.Correct++
			} else {
				g.Wrong++
			}

			if len(g.Typed) >= len(g.Text) {
				g.Finished = true
				if g.OnComplete != nil {
					g.OnComplete(g.WPM(), g.Accuracy())
				}
			}
		}
	}
	return nil
}
