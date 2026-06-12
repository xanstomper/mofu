package mofu

type CursorShape int

const (
	CursorBlock CursorShape = iota
	CursorUnderline
	CursorBar
)

type Cursor struct {
	X, Y   int
	Shape  CursorShape
	Blink  bool
	Hidden bool
}

func NewCursor(x, y int) *Cursor {
	return &Cursor{
		X:    x,
		Y:    y,
		Shape: CursorBlock,
		Blink: true,
	}
}

func (c *Cursor) Hide() string {
	return "\x1b[?25l"
}

func (c *Cursor) Show() string {
	return "\x1b[?25h"
}

func (c *Cursor) SetPosition(x, y int) string {
	return "\x1b[" + intToStr(y+1) + ";" + intToStr(x+1) + "H"
}

func (c *Cursor) SetShape(shape CursorShape) string {
	switch shape {
	case CursorBlock:
		return "\x1b[2 q"
	case CursorUnderline:
		return "\x1b[4 q"
	case CursorBar:
		return "\x1b[6 q"
	}
	return ""
}

func (c *Cursor) StartBlink() string {
	return "\x1b[?12h"
}

func (c *Cursor) StopBlink() string {
	return "\x1b[?12l"
}

func (c *Cursor) EnableSuspend() string {
	return "\x1b[?34h"
}

func (c *Cursor) DisableSuspend() string {
	return "\x1b[?34l"
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + intToStr(-n)
	}
	buf := make([]byte, 0, 10)
	for n > 0 {
		buf = append(buf, byte('0'+n%10))
		n /= 10
	}
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}
