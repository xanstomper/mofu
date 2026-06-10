package widgets

import (
	"fmt"
	"strings"

	"github.com/anomalyco/mofu"
)

// Text is a simple text display component.
type Text struct {
	content string
}

// NewText creates a new text component.
func NewText(content string) *Text {
	return &Text{content: content}
}

func (t *Text) Render() string                    { return t.content }
func (t *Text) HandleEvent(msg mofu.Msg) mofu.Cmd { return nil }
func (t *Text) Mount() mofu.Cmd                   { return nil }
func (t *Text) Unmount()                          {}

// Box is a container with optional border.
type Box struct {
	style mofu.Style
	child mofu.Component
}

// NewBox creates a new box container.
func NewBox(child mofu.Component) *Box {
	return &Box{
		style: mofu.DefaultStyle().Fg(mofu.Hex("cdd6f4")).Bg(mofu.Hex("1e1e2e")),
		child: child,
	}
}

func (b *Box) Render() string {
	childText := b.child.Render()
	if b.style.Border != (mofu.BorderStyle{}) {
		bs := b.style.Border
		lines := strings.Split(childText, "\n")
		maxW := 0
		for _, l := range lines {
			if len(l) > maxW {
				maxW = len(l)
			}
		}
		var result strings.Builder
		result.WriteRune(bs.TopLeft)
		result.WriteString(strings.Repeat(string(bs.Top), maxW))
		result.WriteRune(bs.TopRight)
		result.WriteByte('\n')
		for _, l := range lines {
			result.WriteRune(bs.Left)
			result.WriteString(l)
			result.WriteString(strings.Repeat(" ", maxW-len(l)))
			result.WriteRune(bs.Right)
			result.WriteByte('\n')
		}
		result.WriteRune(bs.BottomLeft)
		result.WriteString(strings.Repeat(string(bs.Bottom), maxW))
		result.WriteRune(bs.BottomRight)
		return result.String()
	}
	return childText
}

func (b *Box) HandleEvent(msg mofu.Msg) mofu.Cmd { return b.child.HandleEvent(msg) }
func (b *Box) Mount() mofu.Cmd                   { return b.child.Mount() }
func (b *Box) Unmount()                          { b.child.Unmount() }

// Style returns the box's style for modification.
func (b *Box) Style() *mofu.Style { return &b.style }

// Stack arranges children vertically (column) or horizontally (row).
type Stack struct {
	orientation string
	children    []mofu.Component
}

// NewColumn creates a vertical stack.
func NewColumn(children ...mofu.Component) *Stack {
	return &Stack{orientation: "column", children: children}
}

// NewRow creates a horizontal stack.
func NewRow(children ...mofu.Component) *Stack {
	return &Stack{orientation: "row", children: children}
}

func (s *Stack) Render() string {
	var parts []string
	for _, child := range s.children {
		parts = append(parts, child.Render())
	}
	if s.orientation == "row" {
		return joinRows(parts)
	}
	return strings.Join(parts, "\n")
}

func joinRows(parts []string) string {
	lines := make([][]string, 0)
	maxH := 0
	for _, p := range parts {
		ls := strings.Split(p, "\n")
		lines = append(lines, ls)
		if len(ls) > maxH {
			maxH = len(ls)
		}
	}
	var result strings.Builder
	for row := 0; row < maxH; row++ {
		for i, ls := range lines {
			if row < len(ls) {
				result.WriteString(ls[row])
			}
			if i < len(lines)-1 {
				result.WriteByte(' ')
			}
		}
		result.WriteByte('\n')
	}
	return strings.TrimRight(result.String(), "\n")
}

func (s *Stack) HandleEvent(msg mofu.Msg) mofu.Cmd {
	for _, child := range s.children {
		if cmd := child.HandleEvent(msg); cmd != nil {
			return cmd
		}
	}
	return nil
}

func (s *Stack) Mount() mofu.Cmd {
	var cmds []mofu.Cmd
	for _, child := range s.children {
		if cmd := child.Mount(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return mofu.Batch(cmds...)
}

func (s *Stack) Unmount() {
	for _, child := range s.children {
		child.Unmount()
	}
}

// Spacer fills available space.
type Spacer struct{}

func (s *Spacer) Render() string                    { return "" }
func (s *Spacer) HandleEvent(msg mofu.Msg) mofu.Cmd { return nil }
func (s *Spacer) Mount() mofu.Cmd                   { return nil }
func (s *Spacer) Unmount()                          {}

// Divider draws a horizontal line.
type Divider struct {
	char rune
}

func NewDivider(char rune) *Divider {
	return &Divider{char: char}
}

func (d *Divider) Render() string {
	return fmt.Sprintf("\n%s\n", strings.Repeat(string(d.char), 40))
}
func (d *Divider) HandleEvent(msg mofu.Msg) mofu.Cmd { return nil }
func (d *Divider) Mount() mofu.Cmd                   { return nil }
func (d *Divider) Unmount()                          {}
