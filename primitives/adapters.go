package primitives

import (
	"fmt"
	"strings"
)

// Message adapter primitives.

// BTMsg is a generic message adapter.
type BTMsg any

// BTCmd is a command adapter.
type BTCmd func() BTMsg

// BTModel is a model adapter.
type BTModel interface {
	Init() BTCmd
	Update(BTMsg) (BTModel, BTCmd)
	View() string
}

// BTBatch runs commands concurrently and joins results.
func BTBatch(cmds ...BTCmd) BTCmd {
	return func() BTMsg {
		return BTBatchMsg(cmds)
	}
}

// BTBatchMsg represents a batch of commands.
type BTBatchMsg []BTCmd

// BTEmpty returns an empty message.
func BTEmpty() BTMsg { return nil }

// BTRender renders a model view into ANSI text.
func BTRender(model BTModel) string {
	if model == nil {
		return ""
	}
	return model.View()
}

// Style primitives.

// LGStyle stores a subset of styling.
type LGStyle struct {
	Bold      bool
	Italic    bool
	Underline bool
	Reverse   bool
	Width     int
	Align     string
	Fg        uint32
	Bg        uint32
}

// LGText renders styled text.
func LGText(text string, style LGStyle) string {
	var sb strings.Builder
	if style.Bold {
		sb.WriteString("\x1b[1m")
	}
	if style.Italic {
		sb.WriteString("\x1b[3m")
	}
	if style.Underline {
		sb.WriteString("\x1b[4m")
	}
	if style.Reverse {
		sb.WriteString("\x1b[7m")
	}
	if style.Width > 0 {
		switch strings.ToLower(style.Align) {
		case "center":
			text = Center(text, style.Width)
		case "right":
			text = Right(text, style.Width)
		default:
			text = Left(text, style.Width)
		}
	}
	sb.WriteString(text)
	sb.WriteString("\x1b[0m")
	return sb.String()
}

// Left pads right.
func Left(s string, width int) string { return PadRight(s, width) }

// Right pads left.
func Right(s string, width int) string { return PadLeft(s, width) }

// Center pads both sides.
func Center(s string, width int) string { return PadCenter(s, width) }

// PadRight pads right.
func PadRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// PadLeft pads left.
func PadLeft(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return strings.Repeat(" ", width-len(s)) + s
}

// PadCenter centers text.
func PadCenter(s string, width int) string {
	if len(s) >= width {
		return s
	}
	l := (width - len(s)) / 2
	r := width - len(s) - l
	return strings.Repeat(" ", l) + s + strings.Repeat(" ", r)
}

// Log primitives.

// LogLine is a log entry.
type LogLine struct {
	Level string
	Text  string
}

// Log renders a compact log line.
func Log(level, text string) string { return "[" + level + "] " + text }

// Table renders a simple markdown-ish table.
func Table(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("| ")
	sb.WriteString(strings.Join(headers, " | "))
	sb.WriteString(" |\n| ")
	sb.WriteString(strings.Repeat("--- | ", len(headers)))
	sb.WriteString("\n")
	for _, row := range rows {
		sb.WriteString("| ")
		sb.WriteString(strings.Join(row, " | "))
		sb.WriteString(" |\n")
	}
	return sb.String()
}

// Spinner renders spinner frames by index.
func Spinner(frame int) string {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	return frames[frame%len(frames)]
}

// Notcurses-style cell primitives.

// NCPlane is a rectangular cell plane.
type NCPlane struct {
	Width  int
	Height int
	Buffer *Buffer
}

// NewNCPlane returns a plane.
func NewNCPlane(width, height int) *NCPlane {
	return &NCPlane{Width: width, Height: height, Buffer: NewBuffer(width, height)}
}

// PutChar writes a character.
func (p *NCPlane) PutChar(x, y int, ch rune, fg, bg uint32) {
	p.Buffer.Set(x, y, Cell{Glyph: ch, Width: 1, Fg: fg, Bg: bg})
}

// Printf writes formatted text.
func (p *NCPlane) Printf(x, y int, fg, bg uint32, format string, args ...any) {
	p.Buffer.WriteString(x, y, fmt.Sprintf(format, args...), fg, bg, 0)
}
