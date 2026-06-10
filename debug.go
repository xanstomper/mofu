package mofu

import (
	"fmt"
	"strings"
	"sync/atomic"
)

type DebugOverlay struct {
	enabled    bool
	showTree   bool
	showDirty  bool
	showTiming bool
	stats      FrameStats
}

func NewDebugOverlay() *DebugOverlay {
	return &DebugOverlay{}
}

func (d *DebugOverlay) Toggle() {
	d.enabled = !d.enabled
}

func (d *DebugOverlay) SetStats(stats FrameStats) {
	d.stats = stats
}

func (d *DebugOverlay) Render(ctx *RenderContext) {
	if !d.enabled {
		return
	}
	r := ctx.Renderer
	info := fmt.Sprintf("FPS: %.1f | Frame: %d | Render: %v | Cells: %d/%d",
		d.stats.FPS,
		atomic.LoadInt64(&d.stats.FrameCount),
		d.stats.RenderTime,
		d.stats.DirtyCells,
		d.stats.TotalCells,
	)
	sty := DefaultStyle().Fg(Hex("a6e3a1")).Bg(Hex("1e1e2e"))
	sty.Bold = true
	r.WriteStyledString(info, 0, 0, sty)
}

type DebugInfo struct {
	stats atomic.Value
}

func NewDebugInfo() *DebugInfo {
	return &DebugInfo{}
}

func (d *DebugInfo) String() string {
	s, ok := d.stats.Load().(FrameStats)
	if !ok {
		return ""
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("FPS: %.1f\n", s.FPS))
	b.WriteString(fmt.Sprintf("Frames: %d\n", atomic.LoadInt64(&s.FrameCount)))
	b.WriteString(fmt.Sprintf("Render: %v\n", s.RenderTime))
	b.WriteString(fmt.Sprintf("Dirty: %d / %d\n", s.DirtyCells, s.TotalCells))
	return b.String()
}

func ClearScreen() string {
	return "\x1b[2J"
}

func HideCursor() string {
	return "\x1b[?25l"
}

func ShowCursor() string {
	return "\x1b[?25h"
}

func EnableAltScreen() string {
	return "\x1b[?1049h"
}

func DisableAltScreen() string {
	return "\x1b[?1049l"
}
