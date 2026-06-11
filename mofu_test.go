package mofu_test

import (
	"testing"

	"github.com/xanstomper/mofu"
	"github.com/xanstomper/mofu/widgets"
)

// TestCounterApp tests a simple counter application flow.
func TestCounterApp(t *testing.T) {
	type Counter struct {
		mofu.Minimal
		count int
	}

	app := &Counter{}

	// Simulate key press events
	tests := []struct {
		name   string
		key    mofu.Key
		runes  []byte
		expect int
	}{
		{"increment with j", 0, []byte{'j'}, 1},
		{"increment with down", mofu.KeyDown, nil, 2},
		{"decrement with k", 0, []byte{'k'}, 1},
		{"decrement with up", mofu.KeyUp, nil, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Handle event
			switch {
			case tt.key == mofu.KeyDown || (len(tt.runes) > 0 && tt.runes[0] == 'j'):
				app.count++
			case tt.key == mofu.KeyUp || (len(tt.runes) > 0 && tt.runes[0] == 'k'):
				app.count--
			}

			if app.count != tt.expect {
				t.Errorf("count = %d, want %d", app.count, tt.expect)
			}
		})
	}
}

// TestInputWidget tests the Input widget in isolation.
func TestInputWidget(t *testing.T) {
	input := widgets.NewInput()
	input.Placeholder = "Enter text"

	// Test typing
	input.Focus()
	for _, r := range "hello" {
		input.InsertRune(r)
	}

	if input.Value != "hello" {
		t.Errorf("value = %q, want %q", input.Value, "hello")
	}

	// Test cursor movement
	input.SetCursor(2)
	if input.CursorPos != 2 {
		t.Errorf("cursor = %d, want 2", input.CursorPos)
	}

	// Test delete
	input.DeleteBefore()
	if input.Value != "hllo" {
		t.Errorf("after delete: value = %q, want %q", input.Value, "hllo")
	}
}

// TestListWidget tests the List widget navigation.
func TestListWidget(t *testing.T) {
	items := []widgets.ListItem{
		{Title: "First"},
		{Title: "Second"},
		{Title: "Third"},
	}
	list := widgets.NewList(items)

	// Test navigation
	downEvent := mofu.Event{
		Type: mofu.EventKeyPress,
		Data: mofu.KeyEvent{Key: mofu.KeyDown},
	}
	upEvent := mofu.Event{
		Type: mofu.EventKeyPress,
		Data: mofu.KeyEvent{Key: mofu.KeyUp},
	}

	list.HandleEvent(downEvent)
	if list.Selected != 1 {
		t.Errorf("after down: selected = %d, want 1", list.Selected)
	}

	list.HandleEvent(downEvent)
	if list.Selected != 2 {
		t.Errorf("after down: selected = %d, want 2", list.Selected)
	}

	list.HandleEvent(upEvent)
	if list.Selected != 1 {
		t.Errorf("after up: selected = %d, want 1", list.Selected)
	}
}

// TestCheckboxWidget tests the Checkbox toggle.
func TestCheckboxWidget(t *testing.T) {
	check := widgets.NewCheckbox("Accept terms", false)

	spaceEvent := mofu.Event{
		Type: mofu.EventKeyPress,
		Data: mofu.KeyEvent{Key: mofu.KeySpace},
	}

	check.Focus()
	check.HandleEvent(spaceEvent)

	if !check.Checked {
		t.Error("expected checked after space")
	}

	check.HandleEvent(spaceEvent)
	if check.Checked {
		t.Error("expected unchecked after second space")
	}
}

// TestButtonWidget tests the Button press.
func TestButtonWidget(t *testing.T) {
	pressed := false
	btn := widgets.NewButton("Submit", func() mofu.Cmd {
		pressed = true
		return nil
	})

	btn.Focus()
	enterEvent := mofu.Event{
		Type: mofu.EventKeyPress,
		Data: mofu.KeyEvent{Key: mofu.KeyEnter},
	}
	btn.HandleEvent(enterEvent)

	if !pressed {
		t.Error("expected button to be pressed")
	}
}

// TestTabsWidget tests the Tabs navigation.
func TestTabsWidget(t *testing.T) {
	tabs := widgets.NewTabs([]widgets.Tab{
		{Label: "Tab A"},
		{Label: "Tab B"},
		{Label: "Tab C"},
	})

	rightEvent := mofu.Event{
		Type: mofu.EventKeyPress,
		Data: mofu.KeyEvent{Key: mofu.KeyRight},
	}
	leftEvent := mofu.Event{
		Type: mofu.EventKeyPress,
		Data: mofu.KeyEvent{Key: mofu.KeyLeft},
	}

	tabs.HandleEvent(rightEvent)
	if tabs.Selected != 1 {
		t.Errorf("selected = %d, want 1", tabs.Selected)
	}

	tabs.HandleEvent(rightEvent)
	if tabs.Selected != 2 {
		t.Errorf("selected = %d, want 2", tabs.Selected)
	}

	tabs.HandleEvent(leftEvent)
	if tabs.Selected != 1 {
		t.Errorf("selected = %d, want 1", tabs.Selected)
	}
}

// TestProgressBarValue tests the ProgressBar clamping.
func TestProgressBarValue(t *testing.T) {
	bar := widgets.NewProgressBar(0.5)

	bar.SetValue(1.5)
	if bar.Value != 1.0 {
		t.Errorf("value = %f, want 1.0", bar.Value)
	}

	bar.SetValue(-0.5)
	if bar.Value != 0.0 {
		t.Errorf("value = %f, want 0.0", bar.Value)
	}

	bar.SetValue(0.75)
	if bar.Value != 0.75 {
		t.Errorf("value = %f, want 0.75", bar.Value)
	}
}

// TestQuitCmd tests the QuitCmd function.
func TestQuitCmd(t *testing.T) {
	cmd := mofu.QuitCmd()
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}

	msg := cmd()
	if _, ok := msg.(mofu.QuitMsg); !ok {
		t.Errorf("expected QuitMsg, got %T", msg)
	}
}

// TestBatchCmd tests the Batch function.
func TestBatchCmd(t *testing.T) {
	count := 0
	cmd1 := func() mofu.Msg { count++; return nil }
	cmd2 := func() mofu.Msg { count++; return nil }

	batch := mofu.Batch(cmd1, cmd2)
	if batch == nil {
		t.Fatal("expected non-nil batch")
	}

	msg := batch()
	if _, ok := msg.(mofu.BatchMsg); !ok {
		t.Errorf("expected BatchMsg, got %T", msg)
	}
}

// TestStyleFgBg tests the Style Fg/Bg methods.
func TestStyleFgBg(t *testing.T) {
	style := mofu.DefaultStyle()
	red := mofu.RGB(255, 0, 0)
	blue := mofu.RGB(0, 0, 255)

	style = style.Fg(red)
	if style.Foreground != red {
		t.Error("foreground not set")
	}

	style = style.Bg(blue)
	if style.Background != blue {
		t.Error("background not set")
	}
}

// TestHexColor tests the Hex color parser.
func TestHexColor(t *testing.T) {
	tests := []struct {
		hex    string
		r, g, b uint8
	}{
		{"#ff0000", 255, 0, 0},
		{"00ff00", 0, 255, 0},
		{"#0000ff", 0, 0, 255},
		{"#ffffff", 255, 255, 255},
	}

	for _, tt := range tests {
		c := mofu.Hex(tt.hex)
		if c.R != tt.r || c.G != tt.g || c.B != tt.b {
			t.Errorf("Hex(%q) = RGB(%d,%d,%d), want RGB(%d,%d,%d)",
				tt.hex, c.R, c.G, c.B, tt.r, tt.g, tt.b)
		}
	}
}

// TestThemeManager tests theme switching.
func TestThemeManager(t *testing.T) {
	dm := mofu.NewThemeManager(mofu.DefaultTheme())

	mochi := mofu.MochiTheme()
	dm.Register("mochi", mochi)

	if !dm.Apply("mochi") {
		t.Fatal("Apply failed")
	}
	if dm.Current().Name != "mochi" {
		t.Errorf("current = %q, want %q", dm.Current().Name, "mofu")
	}

	if dm.Apply("nonexistent") {
		t.Error("expected false for nonexistent theme")
	}
}

// TestSpacingTokens tests the spacing token system.
func TestSpacingTokens(t *testing.T) {
	tests := []struct {
		token  mofu.SpacingToken
		expect int
	}{
		{mofu.SpacingNone, 0},
		{mofu.SpacingS, 2},
		{mofu.SpacingM, 4},
		{mofu.SpacingL, 8},
		{mofu.SpacingXL, 12},
	}

	for _, tt := range tests {
		if got := tt.token.Value(); got != tt.expect {
			t.Errorf("SpacingToken(%d).Value() = %d, want %d", tt.token, got, tt.expect)
		}
	}
}

// TestRectContains tests the Rect.Contains method.
func TestRectContains(t *testing.T) {
	r := mofu.Rect{X: 10, Y: 10, Width: 20, Height: 10}

	tests := []struct {
		x, y   int
		expect bool
	}{
		{15, 15, true},
		{10, 10, true},
		{29, 19, true},
		{30, 19, false},
		{5, 5, false},
	}

	for _, tt := range tests {
		if got := r.Contains(tt.x, tt.y); got != tt.expect {
			t.Errorf("Rect.Contains(%d,%d) = %v, want %v", tt.x, tt.y, got, tt.expect)
		}
	}
}
