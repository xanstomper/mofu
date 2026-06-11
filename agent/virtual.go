package agent

import (
	"fmt"
	"strings"
	"sync"

	"github.com/xanstomper/mofu"
)

// =========================================================================
// VirtualScroll — render only visible lines for massive datasets
// Supports millions of lines with O(1) scroll.
// =========================================================================

type VirtualScroll struct {
	lines       [][]renderSegment
	totalLines  int
	scrollY     int
	viewHeight  int
	viewWidth   int
	cursorLine  int
	selection   [2]int
	hasSelection bool
	mu          sync.RWMutex
}

type renderLine struct {
	segments []renderSegment
}

type renderSegment struct {
	text  string
	fg    mofu.Color
	bg    mofu.Color
	attrs mofu.AttrMask
}

func NewVirtualScroll() *VirtualScroll {
	return &VirtualScroll{}
}

func (vs *VirtualScroll) SetLines(lines [][]renderSegment) {
	vs.mu.Lock()
	vs.lines = lines
	vs.totalLines = len(lines)
	vs.mu.Unlock()
}

func (vs *VirtualScroll) AppendLine(segments []renderSegment) {
	vs.mu.Lock()
	vs.lines = append(vs.lines, segments)
	vs.totalLines = len(vs.lines)
	vs.mu.Unlock()
}

func (vs *VirtualScroll) AppendText(text string, fg, bg mofu.Color) {
	vs.AppendLine([]renderSegment{{text: text, fg: fg, bg: bg}})
}

func (vs *VirtualScroll) Clear() {
	vs.mu.Lock()
	vs.lines = nil
	vs.totalLines = 0
	vs.scrollY = 0
	vs.mu.Unlock()
}

func (vs *VirtualScroll) Len() int {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	return vs.totalLines
}

func (vs *VirtualScroll) ScrollTo(line int) {
	vs.mu.Lock()
	if line < 0 {
		line = 0
	}
	if line >= vs.totalLines {
		line = vs.totalLines - 1
	}
	if line < 0 {
		line = 0
	}
	vs.scrollY = line
	vs.mu.Unlock()
}

func (vs *VirtualScroll) ScrollToBottom() {
	vs.mu.Lock()
	vs.scrollY = vs.totalLines - vs.viewHeight
	if vs.scrollY < 0 {
		vs.scrollY = 0
	}
	vs.mu.Unlock()
}

func (vs *VirtualScroll) ScrollUp(n int) {
	vs.mu.Lock()
	vs.scrollY -= n
	if vs.scrollY < 0 {
		vs.scrollY = 0
	}
	vs.mu.Unlock()
}

func (vs *VirtualScroll) ScrollDown(n int) {
	vs.mu.Lock()
	vs.scrollY += n
	max := vs.totalLines - vs.viewHeight
	if max < 0 {
		max = 0
	}
	if vs.scrollY > max {
		vs.scrollY = max
	}
	vs.mu.Unlock()
}

func (vs *VirtualScroll) PageUp() {
	vs.mu.Lock()
	vs.scrollY -= vs.viewHeight - 2
	if vs.scrollY < 0 {
		vs.scrollY = 0
	}
	vs.mu.Unlock()
}

func (vs *VirtualScroll) PageDown() {
	vs.mu.Lock()
	vs.scrollY += vs.viewHeight - 2
	max := vs.totalLines - vs.viewHeight
	if max < 0 {
		max = 0
	}
	if vs.scrollY > max {
		vs.scrollY = max
	}
	vs.mu.Unlock()
}

func (vs *VirtualScroll) Home() {
	vs.mu.Lock()
	vs.scrollY = 0
	vs.mu.Unlock()
}

func (vs *VirtualScroll) End() {
	vs.mu.Lock()
	vs.scrollY = vs.totalLines - vs.viewHeight
	if vs.scrollY < 0 {
		vs.scrollY = 0
	}
	vs.mu.Unlock()
}

func (vs *VirtualScroll) CursorUp() {
	vs.mu.Lock()
	if vs.cursorLine > 0 {
		vs.cursorLine--
	}
	if vs.cursorLine < vs.scrollY {
		vs.scrollY = vs.cursorLine
	}
	vs.mu.Unlock()
}

func (vs *VirtualScroll) CursorDown() {
	vs.mu.Lock()
	if vs.cursorLine < vs.totalLines-1 {
		vs.cursorLine++
	}
	if vs.cursorLine >= vs.scrollY+vs.viewHeight {
		vs.scrollY = vs.cursorLine - vs.viewHeight + 1
	}
	vs.mu.Unlock()
}

func (vs *VirtualScroll) Render(ctx *mofu.RenderContext) {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	r := ctx.Bounds
	vs.viewWidth = r.Width
	vs.viewHeight = r.Height

	start := vs.scrollY
	end := start + r.Height
	if end > vs.totalLines {
		end = vs.totalLines
	}

	y := r.Y
	for i := start; i < end; i++ {
		if y >= r.Y+r.Height {
			break
		}

		line := vs.lines[i]
		x := r.X

		for _, seg := range line {
			if x >= r.X+r.Width {
				break
			}
			text := seg.text
			if x+len(text) > r.X+r.Width {
				text = text[:r.X+r.Width-x]
			}
			ctx.Renderer.WriteString(text, x, y, seg.fg, seg.bg, seg.attrs)
			x += len(text)
		}

		// Fill remaining space
		if x < r.X+r.Width {
			ctx.Renderer.WriteString(strings.Repeat(" ", r.X+r.Width-x), x, y, mofu.ColorBlack, mofu.ColorBlack, 0)
		}

		y++
	}

	// Scroll indicator
	if vs.totalLines > r.Height {
		scrollPct := float64(vs.scrollY) / float64(vs.totalLines-r.Height)
		barH := r.Height
		thumbH := r.Height * r.Height / vs.totalLines
		if thumbH < 1 {
			thumbH = 1
		}
		thumbPos := int(scrollPct * float64(barH-thumbH))

		for i := 0; i < barH; i++ {
			char := "│"
			if i >= thumbPos && i < thumbPos+thumbH {
				char = "█"
			}
			ctx.Renderer.WriteString(char, r.X+r.Width-1, r.Y+i, mofu.Hex("585b70"), mofu.ColorBlack, 0)
		}
	}
}

func (vs *VirtualScroll) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	switch ke.Key {
	case mofu.KeyUp:
		vs.CursorUp()
	case mofu.KeyDown:
		vs.CursorDown()
	case mofu.KeyPgUp:
		vs.PageUp()
	case mofu.KeyPgDn:
		vs.PageDown()
	case mofu.KeyHome:
		vs.Home()
	case mofu.KeyEnd:
		vs.End()
	}

	if len(ke.Runes) > 0 {
		switch ke.Runes[0] {
		case 'j':
			vs.CursorDown()
		case 'k':
			vs.CursorUp()
		case 'g':
			vs.Home()
		case 'G':
			vs.End()
		}
	}

	return nil
}

// =========================================================================
// DiffRenderer — differential cell rendering
// Only writes cells that changed since last frame.
// =========================================================================

type DiffRenderer struct {
	frontBuf [][]cellState
	backBuf  [][]cellState
	width    int
	height   int
	dirty    bool
	mu       sync.Mutex
}

type cellState struct {
	char  rune
	fg    mofu.Color
	bg    mofu.Color
	attrs mofu.AttrMask
}

func NewDiffRenderer(w, h int) *DiffRenderer {
	dr := &DiffRenderer{
		width:  w,
		height: h,
	}
	dr.frontBuf = make([][]cellState, h)
	dr.backBuf = make([][]cellState, h)
	for i := range dr.frontBuf {
		dr.frontBuf[i] = make([]cellState, w)
		dr.backBuf[i] = make([]cellState, w)
	}
	return dr
}

func (dr *DiffRenderer) SetCell(x, y int, ch rune, fg, bg mofu.Color, attrs mofu.AttrMask) {
	if x < 0 || x >= dr.width || y < 0 || y >= dr.height {
		return
	}
	dr.backBuf[y][x] = cellState{char: ch, fg: fg, bg: bg, attrs: attrs}
}

func (dr *DiffRenderer) WriteString(s string, x, y int, fg, bg mofu.Color, attrs mofu.AttrMask) {
	for i, ch := range s {
		dr.SetCell(x+i, y, ch, fg, bg, attrs)
	}
}

func (dr *DiffRenderer) Flush(ctx *mofu.RenderContext) int {
	dr.mu.Lock()
	defer dr.mu.Unlock()

	writes := 0
	for y := 0; y < dr.height; y++ {
		for x := 0; x < dr.width; x++ {
			back := dr.backBuf[y][x]
			front := dr.frontBuf[y][x]

			if back.char != front.char || back.fg != front.fg || back.bg != front.bg || back.attrs != front.attrs {
				ctx.Renderer.WriteString(string(back.char), x, y, back.fg, back.bg, back.attrs)
				dr.frontBuf[y][x] = back
				writes++
			}
		}
	}

	// Clear back buffer
	for y := 0; y < dr.height; y++ {
		for x := 0; x < dr.width; x++ {
			dr.backBuf[y][x] = cellState{}
		}
	}

	return writes
}

func (dr *DiffRenderer) Resize(w, h int) {
	dr.mu.Lock()
	defer dr.mu.Unlock()

	dr.width = w
	dr.height = h
	dr.frontBuf = make([][]cellState, h)
	dr.backBuf = make([][]cellState, h)
	for i := range dr.frontBuf {
		dr.frontBuf[i] = make([]cellState, w)
		dr.backBuf[i] = make([]cellState, w)
	}
}

// =========================================================================
// MassiveLogViewer — handle 100k+ log lines efficiently
// =========================================================================

type MassiveLogViewer struct {
	scroll     *VirtualScroll
	filter     string
	level      string
	search     string
	matchCount int
	mu         sync.RWMutex
}

type LogLine struct {
	Timestamp string
	Level     string
	Source    string
	Message   string
	Raw       string
}

func NewMassiveLogViewer() *MassiveLogViewer {
	return &MassiveLogViewer{
		scroll: NewVirtualScroll(),
	}
}

func (mlv *MassiveLogViewer) AddLine(line LogLine) {
	mlv.mu.Lock()
	defer mlv.mu.Unlock()

	fg := mofu.Hex("cdd6f4")
	switch strings.ToUpper(line.Level) {
	case "ERROR", "FATAL":
		fg = mofu.Hex("f38ba8")
	case "WARN", "WARNING":
		fg = mofu.Hex("fab387")
	case "INFO":
		fg = mofu.Hex("a6e3a1")
	case "DEBUG":
		fg = mofu.Hex("585b70")
	case "TRACE":
		fg = mofu.Hex("444444")
	}

	ts := fmt.Sprintf("%-12s", line.Timestamp)
	src := fmt.Sprintf("%-10s", line.Source)
	lvl := fmt.Sprintf("%-6s", strings.ToUpper(line.Level))

	segments := []renderSegment{
		{text: ts, fg: mofu.Hex("585b70"), bg: mofu.ColorBlack},
		{text: " ", fg: mofu.ColorBlack, bg: mofu.ColorBlack},
		{text: lvl, fg: fg, bg: mofu.ColorBlack},
		{text: " ", fg: mofu.ColorBlack, bg: mofu.ColorBlack},
		{text: src, fg: mofu.Hex("89b4fa"), bg: mofu.ColorBlack},
		{text: " ", fg: mofu.ColorBlack, bg: mofu.ColorBlack},
		{text: line.Message, fg: fg, bg: mofu.ColorBlack},
	}

	mlv.scroll.AppendLine(segments)
}

func (mlv *MassiveLogViewer) AddLines(lines []LogLine) {
	mlv.mu.Lock()
	defer mlv.mu.Unlock()

	for _, line := range lines {
		fg := mofu.Hex("cdd6f4")
		switch strings.ToUpper(line.Level) {
		case "ERROR", "FATAL":
			fg = mofu.Hex("f38ba8")
		case "WARN", "WARNING":
			fg = mofu.Hex("fab387")
		case "INFO":
			fg = mofu.Hex("a6e3a1")
		case "DEBUG":
			fg = mofu.Hex("585b70")
		}

		segments := []renderSegment{
			{text: fmt.Sprintf("%-12s", line.Timestamp), fg: mofu.Hex("585b70"), bg: mofu.ColorBlack},
			{text: " ", fg: mofu.ColorBlack, bg: mofu.ColorBlack},
			{text: fmt.Sprintf("%-6s", strings.ToUpper(line.Level)), fg: fg, bg: mofu.ColorBlack},
			{text: " ", fg: mofu.ColorBlack, bg: mofu.ColorBlack},
			{text: line.Message, fg: fg, bg: mofu.ColorBlack},
		}

		mlv.scroll.lines = append(mlv.scroll.lines, segments)
		mlv.scroll.totalLines = len(mlv.scroll.lines)
	}
}

func (mlv *MassiveLogViewer) Clear() {
	mlv.mu.Lock()
	mlv.scroll.Clear()
	mlv.mu.Unlock()
}

func (mlv *MassiveLogViewer) Len() int {
	return mlv.scroll.Len()
}

func (mlv *MassiveLogViewer) ScrollToBottom() {
	mlv.scroll.ScrollToBottom()
}

func (mlv *MassiveLogViewer) SetFilter(filter string) {
	mlv.mu.Lock()
	mlv.filter = filter
	mlv.mu.Unlock()
}

func (mlv *MassiveLogViewer) Render(ctx *mofu.RenderContext) {
	mlv.scroll.Render(ctx)
}

func (mlv *MassiveLogViewer) HandleEvent(e mofu.Event) mofu.Cmd {
	return mlv.scroll.HandleEvent(e)
}

// =========================================================================
// AgentStreamDisplay — real-time streaming agent output display
// Optimized for 1000+ tokens/sec rendering.
// =========================================================================

type AgentStreamDisplay struct {
	buffer       *StreamingBuffer
	scroll       *VirtualScroll
	cursorBlink  bool
	cursorVisible bool
	autoScroll   bool
	lastToken    string
	tokenCount   int
	bytesPerSec  float64
	lastUpdate   int64
	mu           sync.RWMutex
}

func NewAgentStreamDisplay(maxLines int) *AgentStreamDisplay {
	return &AgentStreamDisplay{
		buffer:      NewStreamingBuffer(maxLines, 50),
		scroll:      NewVirtualScroll(),
		autoScroll:  true,
	}
}

func (asd *AgentStreamDisplay) WriteToken(token string) {
asd.mu.Lock()
asd.tokenCount++
asd.lastToken = token
asd.mu.Unlock()

	if token == "\n" {
	asd.buffer.Newline()
	} else {
	asd.buffer.AppendToken(token)
	}

	lines := asd.buffer.Lines()
	scrollLines := make([][]renderSegment, len(lines))
	for i, line := range lines {
		scrollLines[i] = []renderSegment{
			{text: line, fg: mofu.Hex("cdd6f4"), bg: mofu.ColorBlack},
		}
	}
	asd.scroll.SetLines(scrollLines)

	if asd.autoScroll {
		asd.scroll.ScrollToBottom()
	}
}

func (asd *AgentStreamDisplay) WriteChunk(chunk string) {
	for _, ch := range chunk {
		asd.WriteToken(string(ch))
	}
}

func (asd *AgentStreamDisplay) Reset() {
asd.buffer.Reset()
asd.scroll.Clear()
asd.mu.Lock()
asd.tokenCount = 0
asd.mu.Unlock()
}

func (asd *AgentStreamDisplay) TokenCount() int {
	asd.mu.RLock()
	defer asd.mu.RUnlock()
	return asd.tokenCount
}

func (asd *AgentStreamDisplay) Render(ctx *mofu.RenderContext) {
asd.scroll.Render(ctx)
}

func (asd *AgentStreamDisplay) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	if len(ke.Runes) > 0 && ke.Runes[0] == 'G' {
		asd.mu.Lock()
		asd.autoScroll = true
		asd.mu.Unlock()
		asd.scroll.End()
	}

	return asd.scroll.HandleEvent(e)
}
