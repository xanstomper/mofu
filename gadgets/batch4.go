package gadgets

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/xanstomper/mofu"
)

// =========================================================================
// BATCH 4: Interactive & Utility Gadgets (10 gadgets)
// =========================================================================

type RealSpinner struct {
	Base
	Frames    []string
	Speed     time.Duration
	Message   string
	Running   bool
	current   int
	lastFrame time.Time
	mu        sync.RWMutex
	OnComplete func()
}

func NewRealSpinner(id string) *RealSpinner {
	return &RealSpinner{
		Base:   *NewBase(id),
		Frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		Speed:  100 * time.Millisecond,
	}
}

func (g *RealSpinner) Start(msg string) {
	g.mu.Lock()
	g.Message = msg
	g.Running = true
	g.current = 0
	g.lastFrame = time.Now()
	g.mu.Unlock()
}

func (g *RealSpinner) Stop() {
	g.mu.Lock()
	g.Running = false
	g.mu.Unlock()
}

func (g *RealSpinner) SetMessage(msg string) {
	g.mu.Lock()
	g.Message = msg
	g.mu.Unlock()
}

func (g *RealSpinner) Tick() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.Running && time.Since(g.lastFrame) >= g.Speed {
		g.current = (g.current + 1) % len(g.Frames)
		g.lastFrame = time.Now()
	}
}

func (g *RealSpinner) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if !g.Running {
		ctx.Renderer.WriteString(" ✓ Done", ctx.Bounds.X, ctx.Bounds.Y, mofu.Hex("a6e3a1"), mofu.ColorBlack, mofu.AttrBold)
		return
	}

	frame := g.Frames[g.current]
	line := fmt.Sprintf(" %s %s", frame, g.Message)
	if len(line) > ctx.Bounds.Width {
		line = line[:ctx.Bounds.Width]
	}
	ctx.Renderer.WriteString(line, ctx.Bounds.X, ctx.Bounds.Y, mofu.Hex("89b4fa"), mofu.ColorBlack, 0)
}

func (g *RealSpinner) HandleEvent(e mofu.Event) mofu.Cmd { return nil }

// RealWorldClock shows multiple timezone clocks.
type RealWorldClock struct {
	Base
	Zones    []ClockZone
	mu       sync.RWMutex
}

type ClockZone struct {
	Name   string
	Offset time.Duration
}

func NewRealWorldClock(id string) *RealWorldClock {
	return &RealWorldClock{
		Base: *NewBase(id),
		Zones: []ClockZone{
			{Name: "UTC", Offset: 0},
			{Name: "EST", Offset: -5 * time.Hour},
			{Name: "PST", Offset: -8 * time.Hour},
			{Name: "CET", Offset: 1 * time.Hour},
			{Name: "JST", Offset: 9 * time.Hour},
		},
	}
}

func (g *RealWorldClock) AddZone(name string, offset time.Duration) {
	g.mu.Lock()
	g.Zones = append(g.Zones, ClockZone{Name: name, Offset: offset})
	g.mu.Unlock()
}

func (g *RealWorldClock) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	now := time.Now()
	for _, zone := range g.Zones {
		if y >= r.Y+r.Height {
			break
		}
		localTime := now.Add(zone.Offset)
		timeStr := localTime.Format("15:04:05")
		dateStr := localTime.Format("Mon Jan 2")

		line := fmt.Sprintf(" %-6s %s  %s", zone.Name, timeStr, dateStr)
		if len(line) > r.Width {
			line = line[:r.Width]
		}

		color := mofu.Hex("cdd6f4")
		if localTime.Hour() >= 9 && localTime.Hour() < 17 {
			color = mofu.Hex("a6e3a1")
		} else if localTime.Hour() >= 17 || localTime.Hour() < 6 {
			color = mofu.Hex("fab387")
		}

		ctx.Renderer.WriteString(line, r.X, y, color, mofu.ColorBlack, 0)
		y++
	}
}

func (g *RealWorldClock) HandleEvent(e mofu.Event) mofu.Cmd { return nil }

type RealStopwatch struct {
	Base
	Running  bool
	Elapsed  time.Duration
	Laps     []time.Duration
	started  time.Time
	lastLap  time.Duration
	mu       sync.RWMutex
}

func NewRealStopwatch(id string) *RealStopwatch {
	return &RealStopwatch{Base: *NewBase(id)}
}

func (g *RealStopwatch) Start() {
	g.mu.Lock()
	if !g.Running {
		g.started = time.Now()
		g.Running = true
	}
	g.mu.Unlock()
}

func (g *RealStopwatch) Stop() {
	g.mu.Lock()
	if g.Running {
		g.Elapsed += time.Since(g.started)
		g.Running = false
	}
	g.mu.Unlock()
}

func (g *RealStopwatch) Reset() {
	g.mu.Lock()
	g.Running = false
	g.Elapsed = 0
	g.Laps = nil
	g.lastLap = 0
	g.mu.Unlock()
}

func (g *RealStopwatch) Lap() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.Running {
		current := g.Elapsed + time.Since(g.started)
		lap := current - g.lastLap
		g.Laps = append(g.Laps, lap)
		g.lastLap = current
	}
}

func (g *RealStopwatch) CurrentTime() time.Duration {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.Running {
		return g.Elapsed + time.Since(g.started)
	}
	return g.Elapsed
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Millisecond)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	ms := int(d.Milliseconds()) % 1000
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d.%03d", h, m, s, ms)
	}
	return fmt.Sprintf("%d:%02d.%03d", m, s, ms)
}

func (g *RealStopwatch) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	current := g.CurrentTime()
	timeStyle := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
	ctx.Renderer.WriteString(" Stopwatch", r.X, y, timeStyle.Foreground, timeStyle.Background, timeStyle.Attrs)
	y++

	ctx.Renderer.WriteString(" "+formatDuration(current), r.X, y, mofu.Hex("f9e2af"), mofu.ColorBlack, mofu.AttrBold)
	y++

	status := " STOPPED"
	if g.Running {
		status = " RUNNING"
	}
	ctx.Renderer.WriteString(status, r.X+25, y-1, mofu.Hex("a6e3a1"), mofu.ColorBlack, 0)

	if len(g.Laps) > 0 {
		ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
		y++

		start := len(g.Laps) - (r.Height - 3)
		if start < 0 {
			start = 0
		}

		for i := start; i < len(g.Laps); i++ {
			if y >= r.Y+r.Height {
				break
			}
			ctx.Renderer.WriteString(fmt.Sprintf("  Lap %d: %s", i+1, formatDuration(g.Laps[i])), r.X, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
			y++
		}
	}

	ctx.Renderer.WriteString(" space:start/stop l:lap r:reset q:quit", r.X, r.Y+r.Height-1, mofu.Hex("585b70"), mofu.Hex("1e1e2e"), 0)
}

func (g *RealStopwatch) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()
	case ke.Key == mofu.KeySpace:
		if g.Running {
			g.Stop()
		} else {
			g.Start()
		}
	case len(ke.Runes) > 0 && ke.Runes[0] == 'l':
		g.Lap()
	case len(ke.Runes) > 0 && ke.Runes[0] == 'r':
		g.Reset()
	}
	return nil
}

type RealPasswordGenerator struct {
	Base
	Length    int
	Uppercase bool
	Lowercase bool
	Digits    bool
	Symbols   bool
	Current   string
	Strength  int
	mu        sync.RWMutex
	OnGenerate func(pw string)
}

func NewRealPasswordGenerator(id string) *RealPasswordGenerator {
	return &RealPasswordGenerator{
		Base:     *NewBase(id),
		Length:   16,
		Lowercase: true,
		Uppercase: true,
		Digits:    true,
		Symbols:   true,
	}
}

func (g *RealPasswordGenerator) Generate() string {
	g.mu.Lock()
	defer g.mu.Unlock()

	var charset string
	if g.Lowercase {
		charset += "abcdefghijklmnopqrstuvwxyz"
	}
	if g.Uppercase {
		charset += "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	}
	if g.Digits {
		charset += "0123456789"
	}
	if g.Symbols {
		charset += "!@#$%^&*()-_=+[]{}|;:,.<>?"
	}
	if charset == "" {
		charset = "abcdefghijklmnopqrstuvwxyz"
	}

	pw := make([]byte, g.Length)
	for i := range pw {
		pw[i] = charset[rand.Intn(len(charset))]
	}
	g.Current = string(pw)
	g.Strength = g.calcStrength()

	if g.OnGenerate != nil {
		g.OnGenerate(g.Current)
	}
	return g.Current
}

func (g *RealPasswordGenerator) calcStrength() int {
	score := 0
	if g.Length >= 12 {
		score++
	}
	if g.Length >= 16 {
		score++
	}
	if g.Uppercase {
		score++
	}
	if g.Lowercase {
		score++
	}
	if g.Digits {
		score++
	}
	if g.Symbols {
		score++
	}
	return score * 100 / 6
}

func (g *RealPasswordGenerator) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(" Password Generator", r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++

	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	if g.Current != "" {
		pw := g.Current
		if len(pw) > r.Width-4 {
			pw = pw[:r.Width-7] + "..."
		}
		ctx.Renderer.WriteString(" "+pw, r.X, y, mofu.Hex("f9e2af"), mofu.Hex("1e1e2e"), 0)
		y++

		barW := r.Width - 6
		filled := g.Strength * barW / 100
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barW-filled)
		strengthColor := mofu.Hex("f38ba8")
		if g.Strength > 60 {
			strengthColor = mofu.Hex("fab387")
		}
		if g.Strength > 80 {
			strengthColor = mofu.Hex("a6e3a1")
		}
		ctx.Renderer.WriteString(" ["+bar+"]", r.X, y, strengthColor, mofu.ColorBlack, 0)
		y++

		ctx.Renderer.WriteString(fmt.Sprintf(" Strength: %d%%", g.Strength), r.X, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
		y++
	}

	y++
	ctx.Renderer.WriteString(fmt.Sprintf(" Length: %d", g.Length), r.X, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
	y++
	opts := ""
	if g.Lowercase {
		opts += "a-z "
	}
	if g.Uppercase {
		opts += "A-Z "
	}
	if g.Digits {
		opts += "0-9 "
	}
	if g.Symbols {
		opts += "!@# "
	}
	ctx.Renderer.WriteString(" Chars: "+opts, r.X, y, mofu.Hex("89b4fa"), mofu.ColorBlack, 0)
	y++
	ctx.Renderer.WriteString(" g:generate +/-:length u/l/d/s:toggle", r.X, r.Y+r.Height-1, mofu.Hex("585b70"), mofu.Hex("1e1e2e"), 0)
}

func (g *RealPasswordGenerator) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	g.mu.Lock()
	defer g.mu.Unlock()

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()
	case len(ke.Runes) > 0 && ke.Runes[0] == 'g':
		g.Generate()
	case len(ke.Runes) > 0 && ke.Runes[0] == '+':
		if g.Length < 64 {
			g.Length++
		}
	case len(ke.Runes) > 0 && ke.Runes[0] == '-':
		if g.Length > 4 {
			g.Length--
		}
	case len(ke.Runes) > 0 && ke.Runes[0] == 'u':
		g.Uppercase = !g.Uppercase
	case len(ke.Runes) > 0 && ke.Runes[0] == 'l':
		g.Lowercase = !g.Lowercase
	case len(ke.Runes) > 0 && ke.Runes[0] == 'd':
		g.Digits = !g.Digits
	case len(ke.Runes) > 0 && ke.Runes[0] == 's':
		g.Symbols = !g.Symbols
	}
	return nil
}

type RealBinaryClock struct {
	Base
	Time     time.Time
	ShowBCD  bool
	mu       sync.RWMutex
}

func NewRealBinaryClock(id string) *RealBinaryClock {
	return &RealBinaryClock{Base: *NewBase(id), ShowBCD: true}
}

func (g *RealBinaryClock) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y
	now := time.Now()

	ctx.Renderer.WriteString(" Binary Clock", r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(now.Format("15:04:05"), r.X+2, y, mofu.Hex("f9e2af"), mofu.ColorBlack, 0)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
y++

	h, m, s := now.Hour(), now.Minute(), now.Second()
	digits := []struct {
		label string
		value int
		bits  int
	}{
		{"H", h / 10, 4}, {"H", h % 10, 4},
		{"M", m / 10, 4}, {"M", m % 10, 4},
		{"S", s / 10, 4}, {"S", s % 10, 4},
	}

	bits := "●○"
	for row := 3; row >= 0; row-- {
		if y >= r.Y+r.Height-2 {
			break
		}
		line := fmt.Sprintf(" 2%d│", 1<<uint(row))
		for _, d := range digits {
			bit := (d.value >> uint(row)) & 1
			line += " " + string(bits[bit:bit+1]) + " "
		}
		ctx.Renderer.WriteString(line, r.X, y, mofu.Hex("89b4fa"), mofu.ColorBlack, 0)
		y++
	}

	ctx.Renderer.WriteString("    H  H  M  M  S  S", r.X, y, mofu.Hex("6c7086"), mofu.ColorBlack, 0)
}

func (g *RealBinaryClock) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)
	if ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q') {
		return mofu.QuitCmd()
	}
	return nil
}

type RealNotePad struct {
	Base
	Tabs       []NoteTab
	ActiveTab  int
	Modified   []bool
	mu         sync.RWMutex
	OnSave     func(name, content string)
}

type NoteTab struct {
	Name    string
	Content string
	Cursor  int
	ScrollY int
}

func NewRealNotePad(id string) *RealNotePad {
	g := &RealNotePad{Base: *NewBase(id)}
	g.Tabs = []NoteTab{
		{Name: "untitled.txt", Content: ""},
	}
	g.Modified = []bool{false}
	return g
}

func (g *RealNotePad) AddTab(name string) {
	g.mu.Lock()
	g.Tabs = append(g.Tabs, NoteTab{Name: name, Content: ""})
	g.Modified = append(g.Modified, false)
	g.ActiveTab = len(g.Tabs) - 1
	g.mu.Unlock()
}

func (g *RealNotePad) CloseTab(idx int) {
	g.mu.Lock()
	if len(g.Tabs) > 1 && idx >= 0 && idx < len(g.Tabs) {
		g.Tabs = append(g.Tabs[:idx], g.Tabs[idx+1:]...)
		g.Modified = append(g.Modified[:idx], g.Modified[idx+1:]...)
		if g.ActiveTab >= len(g.Tabs) {
			g.ActiveTab = len(g.Tabs) - 1
		}
	}
	g.mu.Unlock()
}

func (g *RealNotePad) GetContent() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.ActiveTab >= 0 && g.ActiveTab < len(g.Tabs) {
		return g.Tabs[g.ActiveTab].Content
	}
	return ""
}

func (g *RealNotePad) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	// Tab bar
	tabLine := ""
	for i, tab := range g.Tabs {
		label := tab.Name
		if g.Modified[i] {
			label += "●"
		}
		if i == g.ActiveTab {
			label = "[" + label + "]"
		} else {
			label = " " + label + " "
		}
		tabLine += label
	}
	if len(tabLine) > r.Width {
		tabLine = tabLine[:r.Width]
	}
	ctx.Renderer.WriteString(tabLine, r.X, y, mofu.Hex("89b4fa"), mofu.Hex("1e1e2e"), 0)
	y++

	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	if g.ActiveTab < len(g.Tabs) {
		tab := g.Tabs[g.ActiveTab]
		lines := strings.Split(tab.Content, "\n")

		start := tab.ScrollY
		for i := start; i < len(lines) && y < r.Y+r.Height-2; i++ {
			line := lines[i]
			num := fmt.Sprintf("%3d│", i+1)
			if len(line) > r.Width-5 {
				line = line[:r.Width-8] + "..."
			}
			ctx.Renderer.WriteString(num+line, r.X, y, mofu.Hex("6c7086"), mofu.ColorBlack, 0)
			y++
		}

		if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
			ctx.Renderer.WriteString(" [empty file]", r.X+1, y, mofu.Hex("585b70"), mofu.ColorBlack, 0)
		}
	}

	status := fmt.Sprintf(" Tab %d/%d | %s", g.ActiveTab+1, len(g.Tabs), g.Tabs[g.ActiveTab].Name)
	ctx.Renderer.WriteString(status, r.X, r.Y+r.Height-1, mofu.Hex("585b70"), mofu.Hex("1e1e2e"), 0)
}

func (g *RealNotePad) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	g.mu.Lock()
	defer g.mu.Unlock()

	if g.ActiveTab < 0 || g.ActiveTab >= len(g.Tabs) {
		return nil
	}

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()

	case ke.Key == mofu.KeyTab:
		if ke.Shift {
			if g.ActiveTab > 0 {
				g.ActiveTab--
			}
		} else {
			if g.ActiveTab < len(g.Tabs)-1 {
				g.ActiveTab++
			}
		}

	case ke.Key == mofu.KeyBack && len(g.Tabs[g.ActiveTab].Content) > 0:
		g.Tabs[g.ActiveTab].Content = g.Tabs[g.ActiveTab].Content[:len(g.Tabs[g.ActiveTab].Content)-1]
		g.Modified[g.ActiveTab] = true

	default:
		if len(ke.Runes) > 0 {
			g.Tabs[g.ActiveTab].Content += string(ke.Runes)
			g.Modified[g.ActiveTab] = true
		}
	}
	return nil
}

type RealCurrencyConverter struct {
	Base
	FromCurrency string
	ToCurrency   string
	Amount       float64
	Rates        map[string]float64
	mu           sync.RWMutex
	OnConvert    func(from, to string, amount, result float64)
}

func NewRealCurrencyConverter(id string) *RealCurrencyConverter {
	return &RealCurrencyConverter{
		Base:     *NewBase(id),
		FromCurrency: "USD",
		ToCurrency:   "EUR",
		Rates: map[string]float64{
			"USD": 1.0,
			"EUR": 0.92,
			"GBP": 0.79,
			"JPY": 149.50,
			"CAD": 1.36,
			"AUD": 1.53,
			"CHF": 0.88,
			"CNY": 7.24,
			"INR": 83.12,
		},
	}
}

func (g *RealCurrencyConverter) Convert(amount float64, from, to string) float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()

	fromRate, ok1 := g.Rates[from]
	toRate, ok2 := g.Rates[to]
	if !ok1 || !ok2 || fromRate == 0 {
		return 0
	}
	return amount / fromRate * toRate
}

func (g *RealCurrencyConverter) SetRate(currency string, rate float64) {
	g.mu.Lock()
	g.Rates[currency] = rate
	g.mu.Unlock()
}

func (g *RealCurrencyConverter) GetRate(currency string) (float64, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	r, ok := g.Rates[currency]
	return r, ok
}

func (g *RealCurrencyConverter) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(" Currency Converter", r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	result := 0.0
	if fromRate, ok1 := g.Rates[g.FromCurrency]; ok1 {
		if toRate, ok2 := g.Rates[g.ToCurrency]; ok2 && fromRate > 0 {
			result = g.Amount / fromRate * toRate
		}
	}

	ctx.Renderer.WriteString(fmt.Sprintf("  Amount:  %.2f %s", g.Amount, g.FromCurrency), r.X, y, mofu.Hex("f9e2af"), mofu.ColorBlack, 0)
	y++
	ctx.Renderer.WriteString(fmt.Sprintf("  Result:  %.2f %s", result, g.ToCurrency), r.X, y, mofu.Hex("a6e3a1"), mofu.ColorBlack, 0)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	currencies := make([]string, 0, len(g.Rates))
	for c := range g.Rates {
		currencies = append(currencies, c)
	}
	sort.Strings(currencies)

	for _, c := range currencies {
		if y >= r.Y+r.Height-1 {
			break
		}
		rate := g.Rates[c]
		line := fmt.Sprintf("  %-6s = %.4f", c, rate)
		if c == g.FromCurrency {
			ctx.Renderer.WriteString(line+"  ← FROM", r.X, y, mofu.Hex("89b4fa"), mofu.ColorBlack, mofu.AttrBold)
		} else if c == g.ToCurrency {
			ctx.Renderer.WriteString(line+"  ← TO", r.X, y, mofu.Hex("a6e3a1"), mofu.ColorBlack, mofu.AttrBold)
		} else {
			ctx.Renderer.WriteString(line, r.X, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
		}
		y++
	}
}

func (g *RealCurrencyConverter) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	g.mu.Lock()
	defer g.mu.Unlock()

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()
	case ke.Key == mofu.KeyUp:
		g.Amount += 1
	case ke.Key == mofu.KeyDown:
		if g.Amount > 0 {
			g.Amount -= 1
		}
	case len(ke.Runes) > 0 && ke.Runes[0] >= '0' && ke.Runes[0] <= '9':
		digit := float64(ke.Runes[0] - '0')
		g.Amount = g.Amount*10 + digit
	}
	return nil
}

type RealFileSize struct {
	Base
	Files []FileEntry
	mu    sync.RWMutex
}

type FileEntry struct {
	Name string
	Size int64
	Mode string
	Date time.Time
}

func NewRealFileSize(id string) *RealFileSize {
	return &RealFileSize{Base: *NewBase(id)}
}

func (g *RealFileSize) AddFile(name string, size int64, mode string, date time.Time) {
	g.mu.Lock()
	g.Files = append(g.Files, FileEntry{Name: name, Size: size, Mode: mode, Date: date})
	g.mu.Unlock()
}

func (g *RealFileSize) Clear() {
	g.mu.Lock()
	g.Files = nil
	g.mu.Unlock()
}

func formatBytesHuman(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func (g *RealFileSize) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(" File Sizes", r.X, y, mofu.Hex("89b4fa"), mofu.ColorBlack, mofu.AttrBold)
	y++

	header := fmt.Sprintf(" %-20s %12s  %-6s", "Name", "Size", "Mode")
	ctx.Renderer.WriteString(header, r.X, y, mofu.Hex("6c7086"), mofu.ColorBlack, 0)
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y+1, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y += 2

	for i, f := range g.Files {
		if y >= r.Y+r.Height-1 {
			break
		}

		name := f.Name
		if len(name) > 20 {
			name = name[:17] + "..."
		}

		line := fmt.Sprintf(" %-20s %12s  %-6s", name, formatBytesHuman(f.Size), f.Mode)

		color := mofu.Hex("cdd6f4")
		if f.Size > 1024*1024 {
			color = mofu.Hex("fab387")
		}
		if f.Size > 100*1024*1024 {
			color = mofu.Hex("f38ba8")
		}

		if i%2 == 0 {
			ctx.Renderer.WriteString(strings.Repeat(" ", r.Width-2), r.X+1, y, mofu.ColorWhite, mofu.Hex("1e1e2e"), 0)
		}
		ctx.Renderer.WriteString(line, r.X, y, color, mofu.ColorBlack, 0)
		y++
	}
}

func (g *RealFileSize) HandleEvent(e mofu.Event) mofu.Cmd { return nil }

type RealColorMixer struct {
	Base
	Color1  mofu.Color
	Color2  mofu.Color
	Ratio   float64
	mu      sync.RWMutex
	OnMix   func(result mofu.Color)
}

func NewRealColorMixer(id string) *RealColorMixer {
	return &RealColorMixer{
		Base:  *NewBase(id),
		Color1: mofu.Hex("ff0000"),
		Color2: mofu.Hex("0000ff"),
		Ratio:  0.5,
	}
}

func (g *RealColorMixer) Mix() mofu.Color {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return mofu.RGB(
		uint8(float64(g.Color1.R)*(1-g.Ratio)+float64(g.Color2.R)*g.Ratio),
		uint8(float64(g.Color1.G)*(1-g.Ratio)+float64(g.Color2.G)*g.Ratio),
		uint8(float64(g.Color1.B)*(1-g.Ratio)+float64(g.Color2.B)*g.Ratio),
	)
}

func (g *RealColorMixer) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(" Color Mixer", r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++

	ctx.Renderer.WriteString(fmt.Sprintf("  Color 1: R:%d G:%d B:%d", g.Color1.R, g.Color1.G, g.Color1.B), r.X, y, g.Color1, mofu.ColorBlack, 0)
	y++

	ctx.Renderer.WriteString(fmt.Sprintf("  Color 2: R:%d G:%d B:%d", g.Color2.R, g.Color2.G, g.Color2.B), r.X, y, g.Color2, mofu.ColorBlack, 0)
	y++

	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	result := g.Mix()
	barW := r.Width - 6
	left := int(g.Ratio * float64(barW))
	right := barW - left
	ctx.Renderer.WriteString("  "+strings.Repeat("█", left)+strings.Repeat("█", right), r.X, y, result, mofu.ColorBlack, 0)
	y++

	ctx.Renderer.WriteString(fmt.Sprintf("  Ratio: %.0f%%", g.Ratio*100), r.X, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
	y++
	ctx.Renderer.WriteString(fmt.Sprintf("  Mixed: R:%d G:%d B:%d", result.R, result.G, result.B), r.X, y, result, mofu.ColorBlack, 0)
	y++
	ctx.Renderer.WriteString(fmt.Sprintf("  ████████"), r.X, y-1, result, mofu.ColorBlack, 0)
}

func (g *RealColorMixer) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	g.mu.Lock()
	defer g.mu.Unlock()

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()
	case ke.Key == mofu.KeyLeft:
		if g.Ratio > 0.05 {
			g.Ratio -= 0.05
		}
	case ke.Key == mofu.KeyRight:
		if g.Ratio < 0.95 {
			g.Ratio += 0.05
		}
	}
	return nil
}
