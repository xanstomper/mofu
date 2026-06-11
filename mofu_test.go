package mofu_test

import (
	"testing"

	"github.com/xanstomper/mofu"
)

func TestCounterApp(t *testing.T) {
	type Counter struct {
		mofu.Minimal
		count int
	}

	handleEvent := func(c *Counter, e mofu.Event) {
		if e.Type != mofu.EventKeyPress {
			return
		}
		ke := e.Data.(mofu.KeyEvent)
		switch ke.Key {
		case mofu.KeyUp:
			c.count++
		case mofu.KeyDown:
			c.count--
		default:
			if len(ke.Runes) > 0 {
				switch ke.Runes[0] {
				case 'j':
					c.count++
				case 'k':
					c.count--
				}
			}
		}
	}

	app := &Counter{}

	tests := []struct {
		name   string
		key    mofu.Key
		runes  []byte
		expect int
	}{
		{"increment with j", 0, []byte{'j'}, 1},
		{"decrement with k", 0, []byte{'k'}, 0},
		{"increment with up", mofu.KeyUp, nil, 1},
		{"decrement with down", mofu.KeyDown, nil, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handleEvent(app, mofu.Event{
				Type: mofu.EventKeyPress,
				Data: mofu.KeyEvent{Key: tt.key, Runes: tt.runes},
			})
			if app.count != tt.expect {
				t.Errorf("count = %d, want %d", app.count, tt.expect)
			}
		})
	}
}

func TestMinimalBounds(t *testing.T) {
	m := &mofu.Minimal{}
	r := mofu.Rect{X: 10, Y: 20, Width: 80, Height: 24}
	m.SetBounds(r)

	if m.Bounds() != r {
		t.Errorf("bounds = %v, want %v", m.Bounds(), r)
	}
}

func TestMinimalDirty(t *testing.T) {
	m := &mofu.Minimal{}
	if m.Dirty() {
		t.Error("should start clean")
	}

	m.SetDirty()
	if !m.Dirty() {
		t.Error("should be dirty after SetDirty")
	}
}

func TestStyleCreation(t *testing.T) {
	s := mofu.DefaultStyle()
	c := mofu.Hex("ff69b4")

	styled := s.Fg(c)
	if styled.Foreground != c {
		t.Errorf("foreground = %v, want %v", styled.Foreground, c)
	}
}

func TestStyleChain(t *testing.T) {
	s := mofu.DefaultStyle()
	s = s.Fg(mofu.Hex("ff69b4")).Bg(mofu.Hex("1e1e2e")).WithAttrs(mofu.AttrBold)

	if s.Foreground != mofu.Hex("ff69b4") {
		t.Error("foreground mismatch")
	}
	if s.Background != mofu.Hex("1e1e2e") {
		t.Error("background mismatch")
	}
	if !s.Attrs.Has(mofu.AttrBold) {
		t.Error("bold not set")
	}
}

func TestColorHex(t *testing.T) {
	c := mofu.Hex("ff69b4")
	if c.R != 255 || c.G != 105 || c.B != 180 {
		t.Errorf("RGB = (%d,%d,%d), want (255,105,180)", c.R, c.G, c.B)
	}
}

func TestColorHexWithHash(t *testing.T) {
	c := mofu.Hex("#ff69b4")
	if c.R != 255 || c.G != 105 || c.B != 180 {
		t.Errorf("RGB = (%d,%d,%d), want (255,105,180)", c.R, c.G, c.B)
	}
}

func TestColorRGB(t *testing.T) {
	c := mofu.RGB(10, 20, 30)
	if c.R != 10 || c.G != 20 || c.B != 30 {
		t.Errorf("RGB = (%d,%d,%d), want (10,20,30)", c.R, c.G, c.B)
	}
}

func TestStateGraph(t *testing.T) {
	g := mofu.NewStateGraph()
	g.Set("key1", 42)
	g.Set("key2", "hello")

	v, ok := g.Get("key1")
	if !ok || v != 42 {
		t.Errorf("Get = %v, %v, want 42, true", v, ok)
	}

	v, ok = g.Get("key2")
	if !ok || v != "hello" {
		t.Errorf("Get = %v, %v, want hello, true", v, ok)
	}

	_, ok = g.Get("nonexistent")
	if ok {
		t.Error("Get nonexistent should return false")
	}
}

func TestQuitCmd(t *testing.T) {
	cmd := mofu.QuitCmd()
	if cmd == nil {
		t.Fatal("QuitCmd should not be nil")
	}
	msg := cmd()
	if _, ok := msg.(mofu.QuitMsg); !ok {
		t.Error("QuitCmd should return QuitMsg")
	}
}

func TestBatchCmd(t *testing.T) {
	called := 0
	cmd1 := func() mofu.Msg { called++; return nil }
	cmd2 := func() mofu.Msg { called++; return nil }

	batch := mofu.Batch(cmd1, cmd2)
	if batch == nil {
		t.Fatal("Batch should not be nil")
	}

	msg := batch()
	batchMsg, ok := msg.(mofu.BatchMsg)
	if !ok {
		t.Fatal("Batch should return BatchMsg")
	}
	if len(batchMsg) != 2 {
		t.Errorf("batch has %d commands, want 2", len(batchMsg))
	}
}

func TestSequenceCmd(t *testing.T) {
	seq := mofu.Sequence(
		func() mofu.Msg { return nil },
		func() mofu.Msg { return nil },
	)

	msg := seq()
	_, ok := msg.(mofu.SequenceMsg)
	if !ok {
		t.Fatal("Sequence should return SequenceMsg")
	}
}

func TestKeyEvent(t *testing.T) {
	e := mofu.Event{
		Type: mofu.EventKeyPress,
		Data: mofu.KeyEvent{
			Key:   mofu.KeyUp,
			Runes: []byte{'k'},
		},
	}

	if e.Type != mofu.EventKeyPress {
		t.Error("type mismatch")
	}

	ke, ok := e.Data.(mofu.KeyEvent)
	if !ok {
		t.Fatal("data should be KeyEvent")
	}
	if ke.Key != mofu.KeyUp {
		t.Errorf("key = %v, want KeyUp", ke.Key)
	}
	if ke.Runes[0] != 'k' {
		t.Errorf("rune = %c, want k", ke.Runes[0])
	}
}

func TestSceneBuffer(t *testing.T) {
	r := mofu.NewSceneBuffer(80, 24)
	if r == nil {
		t.Fatal("SceneBuffer should not be nil")
	}

	r.Set(5, 3, 'X', mofu.Hex("ff69b4"), mofu.ColorBlack, 0)
	r.Clear()
}

func TestEventTypes(t *testing.T) {
	keyEvent := mofu.Event{Type: mofu.EventKeyPress}
	mouseEvent := mofu.Event{Type: mofu.EventMouse}

	if keyEvent.Type != mofu.EventKeyPress {
		t.Error("key event type mismatch")
	}
	if mouseEvent.Type != mofu.EventMouse {
		t.Error("mouse event type mismatch")
	}
}

func TestRect(t *testing.T) {
	r := mofu.Rect{X: 10, Y: 20, Width: 80, Height: 24}
	if r.X != 10 || r.Y != 20 || r.Width != 80 || r.Height != 24 {
		t.Errorf("rect = %v", r)
	}
}

func TestAttrsAccumulation(t *testing.T) {
	s := mofu.DefaultStyle().WithAttrs(mofu.AttrBold)
	if !s.Attrs.Has(mofu.AttrBold) {
		t.Error("bold not set")
	}
	s2 := s.WithAttrs(mofu.AttrItalic)
	if !s2.Attrs.Has(mofu.AttrBold) || !s2.Attrs.Has(mofu.AttrItalic) {
		t.Error("attrs not accumulated")
	}
}
