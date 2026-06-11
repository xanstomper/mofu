package mofu

import (
	"strings"
	"sync"
)

// ---------------------------------------------------------------------------
// Render Mode — inline vs fullscreen
// ---------------------------------------------------------------------------

// RenderMode identifies the rendering mode.
type RenderMode int

const (
	// RenderFullscreen uses the alternate screen buffer (full terminal).
	RenderFullscreen RenderMode = iota

	// RenderInline renders below the cursor, scrolling with terminal output.
	RenderInline
)

// ---------------------------------------------------------------------------
// Inline Renderer — renders below cursor, scrolls with terminal
// ---------------------------------------------------------------------------

// InlineRenderer renders UI below the current cursor position.
// Unlike fullscreen mode, it doesn't use the alternate screen buffer
// and output scrolls naturally with the terminal.
type InlineRenderer struct {
	mu       sync.Mutex
	width    int
	height   int
	lines    []string
	maxLines int
}

// NewInlineRenderer creates an inline renderer.
func NewInlineRenderer(width, maxLines int) *InlineRenderer {
	return &InlineRenderer{
		width:    width,
		maxLines: maxLines,
	}
}

// SetWidth updates the render width.
func (ir *InlineRenderer) SetWidth(width int) {
	ir.mu.Lock()
	ir.width = width
	ir.mu.Unlock()
}

// Render renders content as inline output (appended below cursor).
func (ir *InlineRenderer) Render(content string) string {
	ir.mu.Lock()
	defer ir.mu.Unlock()

	lines := strings.Split(content, "\n")
	if len(lines) > ir.maxLines {
		lines = lines[len(lines)-ir.maxLines:]
	}
	ir.lines = lines

	var buf strings.Builder

	// Move cursor to start of our region
	buf.WriteString("\r")

	// Clear previous output
	for range ir.lines {
		buf.WriteString("\x1b[2K") // clear line
		buf.WriteString("\x1b[1B") // move down
	}
	// Move back up
	for range ir.lines {
		buf.WriteString("\x1b[1A")
	}

	// Write new content
	for i, line := range ir.lines {
		if len(line) > ir.width {
			line = line[:ir.width]
		}
		buf.WriteString(line)
		if i < len(ir.lines)-1 {
			buf.WriteString("\r\n")
		}
	}

	return buf.String()
}

// Lines returns the current rendered lines.
func (ir *InlineRenderer) Lines() []string {
	ir.mu.Lock()
	defer ir.mu.Unlock()
	out := make([]string, len(ir.lines))
	copy(out, ir.lines)
	return out
}

// ---------------------------------------------------------------------------
// Fullscreen Renderer — alternate screen buffer
// ---------------------------------------------------------------------------

// FullscreenRenderer uses the alternate screen buffer.
type FullscreenRenderer struct {
	mu       sync.Mutex
	width    int
	height   int
	enabled  bool
}

// NewFullscreenRenderer creates a fullscreen renderer.
func NewFullscreenRenderer(width, height int) *FullscreenRenderer {
	return &FullscreenRenderer{
		width:  width,
		height: height,
	}
}

// Enable activates the alternate screen buffer.
func (fr *FullscreenRenderer) Enable() string {
	fr.mu.Lock()
	fr.enabled = true
	fr.mu.Unlock()
	return "\x1b[?1049h" // enable alternate screen
}

// Disable returns to normal screen buffer.
func (fr *FullscreenRenderer) Disable() string {
	fr.mu.Lock()
	fr.enabled = false
	fr.mu.Unlock()
	return "\x1b[?1049l" // disable alternate screen
}

// Resize updates dimensions.
func (fr *FullscreenRenderer) Resize(width, height int) {
	fr.mu.Lock()
	fr.width = width
	fr.height = height
	fr.mu.Unlock()
}

// Clear clears the screen.
func (fr *FullscreenRenderer) Clear() string {
	return "\x1b[2J\x1b[H" // clear screen + cursor home
}

// HideCursor hides the terminal cursor.
func (fr *FullscreenRenderer) HideCursor() string {
	return "\x1b[?25l"
}

// ShowCursor shows the terminal cursor.
func (fr *FullscreenRenderer) ShowCursor() string {
	return "\x1b[?25h"
}

// Enabled reports whether fullscreen mode is active.
func (fr *FullscreenRenderer) Enabled() bool {
	fr.mu.Lock()
	defer fr.mu.Unlock()
	return fr.enabled
}

// ---------------------------------------------------------------------------
// Dual-mode Renderer — switches between inline and fullscreen
// ---------------------------------------------------------------------------

// DualRenderer supports both inline and fullscreen rendering modes.
type DualRenderer struct {
	mu         sync.Mutex
	mode       RenderMode
	inline     *InlineRenderer
	fullscreen *FullscreenRenderer
	onSwitch   func(old, new RenderMode)
}

// NewDualRenderer creates a dual-mode renderer.
func NewDualRenderer(width, height int) *DualRenderer {
	return &DualRenderer{
		mode:       RenderFullscreen,
		inline:     NewInlineRenderer(width, height),
		fullscreen: NewFullscreenRenderer(width, height),
	}
}

// Mode returns the current render mode.
func (dr *DualRenderer) Mode() RenderMode {
	dr.mu.Lock()
	defer dr.mu.Unlock()
	return dr.mode
}

// SwitchMode switches between inline and fullscreen mode.
// Returns the ANSI escape sequence needed for the transition.
func (dr *DualRenderer) SwitchMode(mode RenderMode) string {
	dr.mu.Lock()
	old := dr.mode
	dr.mode = mode
	dr.mu.Unlock()

	if old == mode {
		return ""
	}

	var escape string
	switch {
	case old == RenderFullscreen && mode == RenderInline:
		escape = dr.fullscreen.Disable()
	case old == RenderInline && mode == RenderFullscreen:
		escape = dr.fullscreen.Enable()
	}

	if dr.onSwitch != nil {
		dr.onSwitch(old, mode)
	}

	return escape
}

// OnSwitch registers a callback for mode switches.
func (dr *DualRenderer) OnSwitch(fn func(old, new RenderMode)) {
	dr.mu.Lock()
	dr.onSwitch = fn
	dr.mu.Unlock()
}

// Resize updates both renderers.
func (dr *DualRenderer) Resize(width, height int) {
	dr.mu.Lock()
	dr.inline.SetWidth(width)
	dr.fullscreen.Resize(width, height)
	dr.mu.Unlock()
}

// Render renders content in the current mode.
func (dr *DualRenderer) Render(content string) string {
	dr.mu.Lock()
	mode := dr.mode
	dr.mu.Unlock()

	switch mode {
	case RenderInline:
		return dr.inline.Render(content)
	case RenderFullscreen:
		// In fullscreen, content is rendered directly to the screen buffer
		return content
	}
	return content
}

// Clear clears the current mode's output.
func (dr *DualRenderer) Clear() string {
	dr.mu.Lock()
	mode := dr.mode
	dr.mu.Unlock()

	switch mode {
	case RenderInline:
		return "\r\x1b[2K"
	case RenderFullscreen:
		return dr.fullscreen.Clear()
	}
	return ""
}

// ---------------------------------------------------------------------------
// Inline Block — a self-contained inline UI region
// ---------------------------------------------------------------------------

// InlineBlock is a self-contained UI block that renders inline.
type InlineBlock struct {
	mu       sync.Mutex
	id       string
	content  string
	height   int
	width    int
}

// NewInlineBlock creates a new inline block.
func NewInlineBlock(id string, width int) *InlineBlock {
	return &InlineBlock{
		id:     id,
		width:  width,
	}
}

// Update updates the block content.
func (ib *InlineBlock) Update(content string) {
	ib.mu.Lock()
	ib.content = content
	ib.height = strings.Count(content, "\n") + 1
	ib.mu.Unlock()
}

// Render returns the ANSI output for this block.
func (ib *InlineBlock) Render() string {
	ib.mu.Lock()
	defer ib.mu.Unlock()

	var buf strings.Builder

	// Clear previous content
	for i := 0; i < ib.height; i++ {
		buf.WriteString("\x1b[2K")
		if i < ib.height-1 {
			buf.WriteString("\x1b[1B")
		}
	}
	// Move back up
	for i := 1; i < ib.height; i++ {
		buf.WriteString("\x1b[1A")
	}

	// Write new content
	buf.WriteString(ib.content)

	return buf.String()
}

// Height returns the block height in lines.
func (ib *InlineBlock) Height() int {
	ib.mu.Lock()
	defer ib.mu.Unlock()
	return ib.height
}
