package mofu

import (
	"testing"
)

func TestParseInput(t *testing.T) {
	p := &Program{}

	tests := []struct {
		name   string
		input  []byte
		key    Key
		ctrl   bool
		alt    bool
		shift  bool
		mouse  bool
		x, y   int
	}{
		// Arrow keys
		{"up", []byte("\x1b[A"), KeyUp, false, false, false, false, 0, 0},
		{"down", []byte("\x1b[B"), KeyDown, false, false, false, false, 0, 0},
		{"right", []byte("\x1b[C"), KeyRight, false, false, false, false, 0, 0},
		{"left", []byte("\x1b[D"), KeyLeft, false, false, false, false, 0, 0},

		// Home/End
		{"home", []byte("\x1b[H"), KeyHome, false, false, false, false, 0, 0},
		{"end", []byte("\x1b[F"), KeyEnd, false, false, false, false, 0, 0},

		// Function keys via ~ sequences
		{"f1", []byte("\x1b[11~"), KeyF1, false, false, false, false, 0, 0},
		{"f2", []byte("\x1b[12~"), KeyF2, false, false, false, false, 0, 0},
		{"f3", []byte("\x1b[13~"), KeyF3, false, false, false, false, 0, 0},
		{"f4", []byte("\x1b[14~"), KeyF4, false, false, false, false, 0, 0},
		{"f5", []byte("\x1b[15~"), KeyF5, false, false, false, false, 0, 0},
		{"f6", []byte("\x1b[17~"), KeyF6, false, false, false, false, 0, 0},
		{"f7", []byte("\x1b[18~"), KeyF7, false, false, false, false, 0, 0},
		{"f8", []byte("\x1b[19~"), KeyF8, false, false, false, false, 0, 0},
		{"f9", []byte("\x1b[20~"), KeyF9, false, false, false, false, 0, 0},
		{"f10", []byte("\x1b[21~"), KeyF10, false, false, false, false, 0, 0},
		{"f11", []byte("\x1b[23~"), KeyF11, false, false, false, false, 0, 0},
		{"f12", []byte("\x1b[24~"), KeyF12, false, false, false, false, 0, 0},

		// Navigation
		{"insert", []byte("\x1b[2~"), KeyInsert, false, false, false, false, 0, 0},
		{"delete", []byte("\x1b[3~"), KeyDelete, false, false, false, false, 0, 0},
		{"pgup", []byte("\x1b[5~"), KeyPgUp, false, false, false, false, 0, 0},
		{"pgdn", []byte("\x1b[6~"), KeyPgDn, false, false, false, false, 0, 0},

		// Special keys
		{"enter", []byte{13}, KeyEnter, false, false, false, false, 0, 0},
		{"tab", []byte{9}, KeyTab, false, false, false, false, 0, 0},
		{"backspace", []byte{127}, KeyBack, false, false, false, false, 0, 0},
		{"escape", []byte{27}, KeyEsc, false, false, false, false, 0, 0},
		{"space", []byte{' '}, KeyNone, false, false, false, false, 0, 0},

		// Ctrl combinations (bytes 1-26 are Ctrl+A through Ctrl+Z)
		{"ctrl+c", []byte{3}, KeyCtrlC, true, false, false, false, 0, 0},
		{"ctrl+d", []byte{4}, KeyCtrlD, true, false, false, false, 0, 0},
		{"ctrl+z", []byte{26}, KeyCtrlZ, true, false, false, false, 0, 0},
		{"ctrl+a", []byte{1}, KeyCtrlA, true, false, false, false, 0, 0},

		// Alt+key
		{"alt+a", []byte{0x1b, 'a'}, KeyNone, false, true, false, false, 0, 0},
		{"alt+b", []byte{0x1b, 'b'}, KeyNone, false, true, false, false, 0, 0},

		// Regular characters
		{"char a", []byte{'a'}, KeyNone, false, false, false, false, 0, 0},
		{"char Z", []byte{'Z'}, KeyNone, false, false, false, false, 0, 0},
		{"char 0", []byte{'0'}, KeyNone, false, false, false, false, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev := p.parseInput(tt.input)
			if ev == nil {
				t.Fatalf("parseInput(%v) returned nil", tt.input)
			}

			ke, ok := ev.Data.(KeyEvent)
			if !ok {
				t.Fatalf("expected KeyEvent, got %T", ev.Data)
			}

			if ke.Key != tt.key {
				t.Errorf("key = %v, want %v", ke.Key, tt.key)
			}
			if ke.Ctrl != tt.ctrl {
				t.Errorf("ctrl = %v, want %v", ke.Ctrl, tt.ctrl)
			}
			if ke.Alt != tt.alt {
				t.Errorf("alt = %v, want %v", ke.Alt, tt.alt)
			}
			if ke.Shift != tt.shift {
				t.Errorf("shift = %v, want %v", ke.Shift, tt.shift)
			}
		})
	}
}

func TestParseInputEmpty(t *testing.T) {
	p := &Program{}
	ev := p.parseInput(nil)
	if ev != nil {
		t.Fatal("expected nil for empty input")
	}
	ev = p.parseInput([]byte{})
	if ev != nil {
		t.Fatal("expected nil for empty input")
	}
}

func TestParseMouseSGR(t *testing.T) {
	p := &Program{}

	// Mouse click at (10, 5) with left button
	input := []byte("\x1b[<0;10;5M")
	ev := p.parseInput(input)
	if ev == nil {
		t.Fatal("expected mouse event")
	}
	if ev.Type != EventMouse {
		t.Fatalf("expected EventMouse, got %v", ev.Type)
	}
	mouse, ok := ev.Data.(MouseEvent)
	if !ok {
		t.Fatalf("expected MouseEvent, got %T", ev.Data)
	}
	if mouse.X != 9 { // 1-based to 0-based
		t.Errorf("X = %d, want 9", mouse.X)
	}
	if mouse.Y != 4 { // 1-based to 0-based
		t.Errorf("Y = %d, want 4", mouse.Y)
	}
	if mouse.Button != MouseLeft {
		t.Errorf("Button = %v, want MouseLeft", mouse.Button)
	}
}

func TestParseShiftTab(t *testing.T) {
	p := &Program{}
	// Shift+Tab: ESC [ Z
	input := []byte{0x1b, '[', 'Z'}
	ev := p.parseInput(input)
	if ev == nil {
		t.Fatal("expected event")
	}
	ke, ok := ev.Data.(KeyEvent)
	if !ok {
		t.Fatalf("expected KeyEvent, got %T", ev.Data)
	}
	if ke.Key != KeyShiftTab {
		t.Errorf("key = %v, want KeyShiftTab", ke.Key)
	}
}

func TestParseAltKey(t *testing.T) {
	p := &Program{}
	input := []byte{0x1b, 'x'}
	ev := p.parseInput(input)
	if ev == nil {
		t.Fatal("expected event")
	}
	ke, ok := ev.Data.(KeyEvent)
	if !ok {
		t.Fatalf("expected KeyEvent, got %T", ev.Data)
	}
	if !ke.Alt {
		t.Error("expected Alt=true")
	}
	if ke.Key != KeyNone {
		t.Errorf("key = %v, want KeyNone", ke.Key)
	}
}

func BenchmarkParseInput(b *testing.B) {
	p := &Program{}
	input := []byte("\x1b[A")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.parseInput(input)
	}
}

func BenchmarkParseInputRegular(b *testing.B) {
	p := &Program{}
	input := []byte{'a'}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.parseInput(input)
	}
}

func BenchmarkParseInputCtrl(b *testing.B) {
	p := &Program{}
	input := []byte{3} // Ctrl+C
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.parseInput(input)
	}
}
