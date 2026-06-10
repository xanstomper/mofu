package mofu

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type MouseMode int

const (
	MouseOff MouseMode = iota
	MouseBasic
	MouseSGRMode
)

func EnableMouse(mode MouseMode) string {
	switch mode {
	case MouseBasic:
		return "\x1b[?1000h"
	case MouseSGRMode:
		return "\x1b[?1000h\x1b[?1002h\x1b[?1006h"
	default:
		return ""
	}
}

func DisableMouse() string {
	return "\x1b[?1006l\x1b[?1002l\x1b[?1000l"
}

type MouseState struct {
	X, Y    int
	Pressed bool
	Button  MouseButton
}

func ParseSGRMouse(data []byte) (bool, int, int, MouseButton, MouseAction) {
	s := string(data)
	if !strings.HasPrefix(s, "\x1b[<") {
		return false, 0, 0, MouseNone, MouseRelease
	}
	end := strings.IndexByte(s, 'M')
	if end < 0 {
		end = strings.IndexByte(s, 'm')
		if end < 0 {
			return false, 0, 0, MouseNone, MouseRelease
		}
	}
	inner := s[3:end]
	parts := strings.Split(inner, ";")
	if len(parts) != 3 {
		return false, 0, 0, MouseNone, MouseRelease
	}
	cb, _ := strconv.Atoi(parts[0])
	cx, _ := strconv.Atoi(parts[1])
	cy, _ := strconv.Atoi(parts[2])
	if cx < 1 {
		cx = 1
	}
	if cy < 1 {
		cy = 1
	}

	action := MousePress
	if s[end] == 'm' {
		action = MouseRelease
	}

	btn := MouseLeft
	code := cb & 0x03
	switch code {
	case 0:
		btn = MouseLeft
	case 1:
		btn = MouseMiddle
	case 2:
		btn = MouseRight
	}

	if cb&0x40 != 0 {
		btn = MouseWheelUp
	} else if cb&0x41 != 0 {
		btn = MouseWheelDown
	}

	if cb&0x20 != 0 {
		action = MouseDrag
	}

	return true, cx - 1, cy - 1, btn, action
}

func ParseBasicMouse(data []byte) (bool, int, int, MouseButton, MouseAction) {
	if len(data) < 6 || data[0] != 0x1b || data[1] != '[' || data[2] != 'M' {
		return false, 0, 0, MouseNone, MouseRelease
	}
	cb := int(data[3]) - 32
	cx := int(data[4]) - 32
	cy := int(data[5]) - 32

	action := MousePress
	if cb&0x20 != 0 {
		action = MouseDrag
	} else if cb&0x40 != 0 {
		action = MouseRelease
	}

	btn := MouseLeft
	switch cb & 0x03 {
	case 0:
		btn = MouseLeft
	case 1:
		btn = MouseMiddle
	case 2:
		btn = MouseRight
	}
	if cb&0x40 != 0 {
		if cb&0x01 != 0 {
			btn = MouseWheelDown
		} else {
			btn = MouseWheelUp
		}
	}

	return true, cx - 1, cy - 1, btn, action
}

func (p *Program) enableMouse() {
	os.Stdout.WriteString(EnableMouse(MouseSGRMode))
}

func (p *Program) disableMouse() {
	os.Stdout.WriteString(DisableMouse())
}

func (p *Program) handleMouseEvent(data []byte) {
	ok, x, y, btn, action := ParseSGRMouse(data)
	if !ok {
		ok, x, y, btn, action = ParseBasicMouse(data)
	}
	if !ok {
		return
	}

	ev := Event{
		Type: EventMouse,
		Data: MouseEvent{X: x, Y: y, Button: btn, Action: action},
		Time: time.Now(),
	}
	p.eventBus.Publish(ev)
	select {
	case p.eventCh <- ev:
	default:
	}
}

func (p *Program) initMouseHandler() {
	p.eventBus.Subscribe("mouse", func(ev Event) {})
}
