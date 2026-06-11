package mofu

import "time"

// ---------------------------------------------------------------------------
// Input Parser — comprehensive terminal input handling
// ---------------------------------------------------------------------------
//
// Handles: arrow keys, function keys, Ctrl+key, Alt+key, mouse (SGR),
// bracketed paste, and Unicode input.

// parseInput parses raw terminal input bytes into a typed Event.
func (p *Program) parseInput(data []byte) *Event {
	if len(data) == 0 {
		return nil
	}

	// ESC sequence
	if data[0] == 0x1b {
		if len(data) == 1 {
			// Bare Escape key
			return &Event{
				Type: EventKeyPress,
				Data: KeyEvent{Key: KeyEsc, Runes: data},
				Time: time.Now(),
			}
		}
		return p.parseEscape(data)
	}

	// Ctrl+key (bytes 1-31, except ESC which is 0x1b)
	if data[0] < 32 && data[0] != 0x1b {
		return p.parseCtrlKey(data)
	}

	// Backspace (127)
	if data[0] == 127 {
		return &Event{
			Type: EventKeyPress,
			Data: KeyEvent{Key: KeyBack, Runes: data},
			Time: time.Now(),
		}
	}

	// Regular key
	return &Event{
		Type: EventKeyPress,
		Data: KeyEvent{Runes: data},
		Time: time.Now(),
	}
}

// parseCtrlKey handles Ctrl+letter and special control characters.
func (p *Program) parseCtrlKey(data []byte) *Event {
	b := data[0]
	var key Key

	switch b {
	case 9:
		key = KeyTab
	case 10:
		key = KeyEnter
	case 13:
		key = KeyEnter
	case 127:
		key = KeyBack
	case 0:
		key = KeyCtrlAt
	case 1:
		key = KeyCtrlA
	case 2:
		key = KeyCtrlB
	case 3:
		key = KeyCtrlC
	case 4:
		key = KeyCtrlD
	case 5:
		key = KeyCtrlE
	case 6:
		key = KeyCtrlF
	case 7:
		key = KeyCtrlG
	case 8:
		key = KeyBack
	case 11:
		key = KeyCtrlK
	case 12:
		key = KeyCtrlL
	case 14:
		key = KeyCtrlN
	case 15:
		key = KeyCtrlO
	case 16:
		key = KeyCtrlP
	case 17:
		key = KeyCtrlQ
	case 18:
		key = KeyCtrlR
	case 19:
		key = KeyCtrlS
	case 20:
		key = KeyCtrlT
	case 21:
		key = KeyCtrlU
	case 22:
		key = KeyCtrlV
	case 23:
		key = KeyCtrlW
	case 24:
		key = KeyCtrlX
	case 25:
		key = KeyCtrlY
	case 26:
		key = KeyCtrlZ
	case 27:
		key = KeyEsc
	case 28:
		key = KeyCtrlBackslash
	case 29:
		key = KeyCtrlCloseBracket
	case 30:
		key = KeyCtrlCaret
	case 31:
		key = KeyCtrlUnderscore
	}

	return &Event{
		Type: EventKeyPress,
		Data: KeyEvent{Key: key, Ctrl: key >= KeyCtrlA && key <= KeyCtrlZ, Runes: data},
		Time: time.Now(),
	}
}

// parseEscape handles ESC-prefixed sequences.
func (p *Program) parseEscape(data []byte) *Event {
	if len(data) < 2 {
		return nil
	}

	// ESC [ — CSI sequences
	if data[1] == '[' {
		return p.parseCSI(data)
	}

	// ESC O — SS3 sequences (function keys, some terminals)
	if data[1] == 'O' {
		return p.parseSS3(data)
	}

	// ESC followed by a character — Alt+key
	if len(data) == 2 && data[1] >= 0x20 && data[1] < 0x7f {
		return &Event{
			Type: EventKeyPress,
			Data: KeyEvent{
				Runes: data[1:],
				Key:   KeyNone,
				Alt:   true,
			},
			Time: time.Now(),
		}
	}

	// ESC [ ? ... — DEC private mode (mouse, bracketed paste, etc.)
	if len(data) >= 3 && data[1] == '[' && data[2] == '?' {
		return p.parseDecPrivate(data)
	}

	return nil
}

// parseCSI handles CSI (Control Sequence Introducer) sequences.
// Format: ESC [ <params> <final>
func (p *Program) parseCSI(data []byte) *Event {
	if len(data) < 3 {
		return nil
	}

	// Find the final byte (letter or ~) starting from position 2
	finalIdx := -1
	for i := 2; i < len(data); i++ {
		if (data[i] >= 'A' && data[i] <= 'Z') || (data[i] >= 'a' && data[i] <= 'z') || data[i] == '~' {
			finalIdx = i
			break
		}
	}
	if finalIdx < 0 {
		return nil
	}

	final := data[finalIdx]
	params := ""
	if finalIdx > 2 {
		params = string(data[2:finalIdx])
	}

	var key Key

	switch final {
	case 'A':
		key = KeyUp
	case 'B':
		key = KeyDown
	case 'C':
		key = KeyRight
	case 'D':
		key = KeyLeft
	case 'H':
		key = KeyHome
	case 'F':
		key = KeyEnd
	case 'Z':
		key = KeyShiftTab
	case '~':
		key = p.parseTildeKey(params)
	case 'M':
		// SGR mouse: ESC [ < Cb ; Cx ; Cy M
		return p.parseMouseSGR(data, false)
	case 'm':
		// SGR mouse release: ESC [ < Cb ; Cx ; Cy m
		return p.parseMouseSGR(data, true)
	case 'R':
		// Cursor position report: ESC [ row ; col R
		return nil // ignore
	}

	// Check for modifier suffix: 1 ; <mod> A
	if final >= 'A' && final <= 'Z' && params != "" {
		if idx := indexOf(params, ';'); idx >= 0 {
			modStr := params[idx+1:]
			if mod := parseModifier(modStr); mod > 0 {
				return &Event{
					Type: EventKeyPress,
					Data: KeyEvent{
						Key:    key,
						Runes:  data,
						Ctrl:   mod&1 != 0,
						Shift:  mod&2 != 0,
						Alt:    mod&4 != 0,
					},
					Time: time.Now(),
				}
			}
		}
	}

	if key != 0 {
		return &Event{
			Type: EventKeyPress,
			Data: KeyEvent{Key: key, Runes: data},
			Time: time.Now(),
		}
	}

	return nil
}

// parseTildeKey maps parameter strings to keys for sequences ending in ~.
func (p *Program) parseTildeKey(params string) Key {
	switch params {
	case "1":
		return KeyHome
	case "2":
		return KeyInsert
	case "3":
		return KeyDelete
	case "4":
		return KeyEnd
	case "5":
		return KeyPgUp
	case "6":
		return KeyPgDn
	case "11":
		return KeyF1
	case "12":
		return KeyF2
	case "13":
		return KeyF3
	case "14":
		return KeyF4
	case "15":
		return KeyF5
	case "17":
		return KeyF6
	case "18":
		return KeyF7
	case "19":
		return KeyF8
	case "20":
		return KeyF9
	case "21":
		return KeyF10
	case "23":
		return KeyF11
	case "24":
		return KeyF12
	}
	return KeyNone
}

// parseSS3 handles SS3 (Single Shift Select) sequences: ESC O <letter>.
func (p *Program) parseSS3(data []byte) *Event {
	if len(data) < 3 {
		return nil
	}
	var key Key
	switch data[2] {
	case 'A':
		key = KeyUp
	case 'B':
		key = KeyDown
	case 'C':
		key = KeyRight
	case 'D':
		key = KeyLeft
	case 'H':
		key = KeyHome
	case 'F':
		key = KeyEnd
	case 'P':
		key = KeyF1
	case 'Q':
		key = KeyF2
	case 'R':
		key = KeyF3
	case 'S':
		key = KeyF4
	}
	if key != 0 {
		return &Event{
			Type: EventKeyPress,
			Data: KeyEvent{Key: key, Runes: data},
			Time: time.Now(),
		}
	}
	return nil
}

// parseDecPrivate handles DEC private mode sequences (mouse, bracketed paste).
func (p *Program) parseDecPrivate(data []byte) *Event {
	// ESC [ ? 1000 h/l — basic mouse
	// ESC [ ? 1002 h/l — button-event mouse
	// ESC [ ? 1003 h/l — any-event mouse
	// ESC [ ? 1006 h/l — SGR mouse
	// ESC [ ? 2004 h/l — bracketed paste
	// These are enable/disable sequences, not input events.
	// Mouse events come as ESC [ < ... M/m
	return nil
}

// parseMouseSGR parses SGR extended mouse reports.
// Format: ESC [ < Cb ; Cx ; Cy M (press) or m (release)
func (p *Program) parseMouseSGR(data []byte, release bool) *Event {
	if len(data) < 6 || data[0] != 0x1b || data[1] != '[' || data[2] != '<' {
		return nil
	}

	// Find M or m terminator
	termIdx := -1
	for i := len(data) - 1; i >= 3; i-- {
		if data[i] == 'M' || data[i] == 'm' {
			termIdx = i
			break
		}
	}
	if termIdx < 0 {
		return nil
	}

	// Parse Cb;Cx;Cy
	payload := string(data[3:termIdx])
	parts := splitByte(payload, ';')
	if len(parts) < 3 {
		return nil
	}

	cb := atoi(parts[0])
	cx := atoi(parts[1]) - 1 // 1-based to 0-based
	cy := atoi(parts[2]) - 1

	if cx < 0 {
		cx = 0
	}
	if cy < 0 {
		cy = 0
	}

	var btn MouseButton
	var action MouseAction

	if release {
		action = MouseRelease
	} else {
		action = MousePress
	}

	switch {
	case cb&3 == 0:
		btn = MouseLeft
	case cb&3 == 1:
		btn = MouseMiddle
	case cb&3 == 2:
		btn = MouseRight
	case cb&3 == 3:
		btn = MouseNone
	case cb&64 != 0:
		// Scroll wheel
		if cb&1 != 0 {
			btn = MouseWheelDown
		} else {
			btn = MouseWheelUp
		}
	}

	if cb&32 != 0 {
		action = MouseDrag
	}

	return &Event{
		Type: EventMouse,
		Data: MouseEvent{
			X:      cx,
			Y:      cy,
			Button: btn,
			Action: action,
		},
		Time: time.Now(),
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func indexOf(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func splitByte(s string, sep byte) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}

func parseModifier(s string) int {
	n := atoi(s)
	// Modifier bitmask: 1=shift, 2=alt, 4=ctrl, 8=super
	return n
}
