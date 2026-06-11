package widgets

import (
	"testing"

	"github.com/xanstomper/mofu"
)

func TestTextInput(t *testing.T) {
	input := NewInput()
	input.Placeholder = "Type here..."

	// Test initial state
	if input.Value != "" {
		t.Errorf("expected empty value, got %q", input.Value)
	}
	if input.Focused {
		t.Error("expected not focused")
	}

	// Test focus/blur
	input.Focus()
	if !input.Focused {
		t.Error("expected focused")
	}
	input.Blur()
	if input.Focused {
		t.Error("expected not focused")
	}

	// Test insert rune
	input.Focus()
	input.InsertRune('h')
	input.InsertRune('i')
	if input.Value != "hi" {
		t.Errorf("expected %q, got %q", "hi", input.Value)
	}
	if input.CursorPos != 2 {
		t.Errorf("expected cursor at 2, got %d", input.CursorPos)
	}

	// Test delete
	input.DeleteBefore()
	if input.Value != "h" {
		t.Errorf("expected %q, got %q", "h", input.Value)
	}

	// Test cursor movement
	input.SetCursor(0)
	if input.CursorPos != 0 {
		t.Errorf("expected cursor at 0, got %d", input.CursorPos)
	}
}

func TestTextInputMaxLen(t *testing.T) {
	input := NewInput()
	input.MaxLen = 3

	input.InsertRune('a')
	input.InsertRune('b')
	input.InsertRune('c')
	input.InsertRune('d') // Should be rejected

	if input.Value != "abc" {
		t.Errorf("expected %q, got %q", "abc", input.Value)
	}
}

func TestTextInputPassword(t *testing.T) {
	input := NewInput()
	input.Password = true
	input.InsertRune('s')
	input.InsertRune('e')
	input.InsertRune('c')
	input.InsertRune('r')
	input.InsertRune('e')
	input.InsertRune('t')

	if input.Value != "secret" {
		t.Errorf("expected %q, got %q", "secret", input.Value)
	}
}

func TestTextInputValidator(t *testing.T) {
	input := NewInput()
	input.Validator = func(s string) bool {
		return len(s) >= 3
	}

	input.InsertRune('a')
	if input.Valid() {
		t.Error("expected invalid for length < 3")
	}

	input.InsertRune('b')
	input.InsertRune('c')
	if !input.Valid() {
		t.Error("expected valid for length >= 3")
	}
}

func TestButton(t *testing.T) {
	pressed := false
	btn := NewButton("Click", func() mofu.Cmd {
		pressed = true
		return nil
	})

	btn.Focus()
	if !btn.Focused {
		t.Error("expected focused")
	}

	// Simulate enter press
	event := mofu.Event{
		Type: mofu.EventKeyPress,
		Data: mofu.KeyEvent{Key: mofu.KeyEnter},
	}
	btn.HandleEvent(event)

	if !pressed {
		t.Error("expected button to be pressed")
	}
}

func TestCheckbox(t *testing.T) {
	check := NewCheckbox("Test", false)

	if check.Checked {
		t.Error("expected unchecked")
	}

	check.Toggle()
	if !check.Checked {
		t.Error("expected checked after toggle")
	}

	check.Toggle()
	if check.Checked {
		t.Error("expected unchecked after second toggle")
	}
}

func TestProgressBar(t *testing.T) {
	bar := NewProgressBar(0.5)

	if bar.Value != 0.5 {
		t.Errorf("expected 0.5, got %f", bar.Value)
	}

	bar.SetValue(1.5) // Should clamp to 1.0
	if bar.Value != 1.0 {
		t.Errorf("expected 1.0, got %f", bar.Value)
	}

	bar.SetValue(-0.5) // Should clamp to 0.0
	if bar.Value != 0.0 {
		t.Errorf("expected 0.0, got %f", bar.Value)
	}
}

func TestList(t *testing.T) {
	items := []ListItem{
		{Title: "Item 1"},
		{Title: "Item 2"},
		{Title: "Item 3"},
	}
	list := NewList(items)

	if list.Selected != 0 {
		t.Errorf("expected selected 0, got %d", list.Selected)
	}

	// Navigate down
	event := mofu.Event{
		Type: mofu.EventKeyPress,
		Data: mofu.KeyEvent{Key: mofu.KeyDown},
	}
	list.HandleEvent(event)

	if list.Selected != 1 {
		t.Errorf("expected selected 1, got %d", list.Selected)
	}

	// Navigate up
	event = mofu.Event{
		Type: mofu.EventKeyPress,
		Data: mofu.KeyEvent{Key: mofu.KeyUp},
	}
	list.HandleEvent(event)

	if list.Selected != 0 {
		t.Errorf("expected selected 0, got %d", list.Selected)
	}
}

func TestSelect(t *testing.T) {
	sel := NewSelect([]string{"A", "B", "C"})

	if sel.Selected != 0 {
		t.Errorf("expected selected 0, got %d", sel.Selected)
	}
	if sel.Open {
		t.Error("expected closed")
	}

	// Open
	sel.Focus()
	event := mofu.Event{
		Type: mofu.EventKeyPress,
		Data: mofu.KeyEvent{Key: mofu.KeyEnter},
	}
	sel.HandleEvent(event)

	if !sel.Open {
		t.Error("expected open")
	}

	// Navigate down
	event = mofu.Event{
		Type: mofu.EventKeyPress,
		Data: mofu.KeyEvent{Key: mofu.KeyDown},
	}
	sel.HandleEvent(event)

	if sel.Selected != 1 {
		t.Errorf("expected selected 1, got %d", sel.Selected)
	}

	// Select
	event = mofu.Event{
		Type: mofu.EventKeyPress,
		Data: mofu.KeyEvent{Key: mofu.KeyEnter},
	}
	sel.HandleEvent(event)

	if sel.Open {
		t.Error("expected closed after select")
	}
}

func TestTabs(t *testing.T) {
	tabs := NewTabs([]Tab{
		{Label: "Tab 1"},
		{Label: "Tab 2"},
		{Label: "Tab 3"},
	})

	if tabs.Selected != 0 {
		t.Errorf("expected selected 0, got %d", tabs.Selected)
	}

	// Navigate right
	event := mofu.Event{
		Type: mofu.EventKeyPress,
		Data: mofu.KeyEvent{Key: mofu.KeyRight},
	}
	tabs.HandleEvent(event)

	if tabs.Selected != 1 {
		t.Errorf("expected selected 1, got %d", tabs.Selected)
	}

	// Navigate left
	event = mofu.Event{
		Type: mofu.EventKeyPress,
		Data: mofu.KeyEvent{Key: mofu.KeyLeft},
	}
	tabs.HandleEvent(event)

	if tabs.Selected != 0 {
		t.Errorf("expected selected 0, got %d", tabs.Selected)
	}
}

func TestMenu(t *testing.T) {
	selected := -1
	menu := NewMenu([]MenuItem{
		{Label: "New"},
		{Label: "Open"},
		{Label: "Save"},
	})
	menu.OnSelect = func(i int) mofu.Cmd {
		selected = i
		return nil
	}

	menu.Focus()

	// Navigate down
	event := mofu.Event{
		Type: mofu.EventKeyPress,
		Data: mofu.KeyEvent{Key: mofu.KeyDown},
	}
	menu.HandleEvent(event)

	if menu.Selected != 1 {
		t.Errorf("expected selected 1, got %d", menu.Selected)
	}

	// Select
	event = mofu.Event{
		Type: mofu.EventKeyPress,
		Data: mofu.KeyEvent{Key: mofu.KeyEnter},
	}
	menu.HandleEvent(event)

	if selected != 1 {
		t.Errorf("expected selected item 1, got %d", selected)
	}
}

func TestToast(t *testing.T) {
	toast := NewToast("Test message", ToastInfo)

	if !toast.Visible {
		t.Error("expected visible")
	}

	toast.Dismiss()
	if toast.Visible {
		t.Error("expected not visible after dismiss")
	}

	toast.Show()
	if !toast.Visible {
		t.Error("expected visible after show")
	}
}

func TestTooltip(t *testing.T) {
	tooltip := NewTooltip("Help text")

	if tooltip.Visible {
		t.Error("expected not visible")
	}

	tooltip.Show()
	if !tooltip.Visible {
		t.Error("expected visible after show")
	}

	tooltip.Hide()
	if tooltip.Visible {
		t.Error("expected not visible after hide")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		width    int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "he..."},
		{"hi", 5, "hi"},
		{"", 5, ""},
	}

	for _, tt := range tests {
		got := Truncate(tt.input, tt.width, true)
		if got != tt.expected {
			t.Errorf("Truncate(%q, %d) = %q, want %q", tt.input, tt.width, got, tt.expected)
		}
	}
}

func BenchmarkInputInsertRune(b *testing.B) {
	input := NewInput()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		input.InsertRune('a')
		input.Value = ""
		input.CursorPos = 0
	}
}

func BenchmarkListNavigate(b *testing.B) {
	items := make([]ListItem, 100)
	for i := range items {
		items[i] = ListItem{Title: "Item"}
	}
	list := NewList(items)
	event := mofu.Event{
		Type: mofu.EventKeyPress,
		Data: mofu.KeyEvent{Key: mofu.KeyDown},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		list.HandleEvent(event)
		if list.Selected >= 99 {
			list.Selected = 0
		}
	}
}
