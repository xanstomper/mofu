package widgets

import (
	"strings"

	"github.com/anomalyco/mofu"
)

// Input is a text input field.
type Input struct {
	value       string
	placeholder string
	cursorPos   int
	focused     bool
	width       int
	onSubmit    func(string) mofu.Cmd
}

// NewInput creates a new input component.
func NewInput() *Input {
	return &Input{
		width: 40,
	}
}

// SetPlaceholder sets the placeholder text.
func (i *Input) SetPlaceholder(p string) *Input {
	i.placeholder = p
	return i
}

// SetWidth sets the input width.
func (i *Input) SetWidth(w int) *Input {
	i.width = w
	return i
}

// OnSubmit sets the submit callback.
func (i *Input) OnSubmit(fn func(string) mofu.Cmd) *Input {
	i.onSubmit = fn
	return i
}

// Value returns the current input value.
func (i *Input) Value() string { return i.value }

// Focus focuses the input.
func (i *Input) Focus() { i.focused = true }

// Blur blurs the input.
func (i *Input) Blur() { i.focused = false }

func (i *Input) Render() string {
	display := i.value
	if display == "" && !i.focused {
		display = i.placeholder
	}
	if len(display) > i.width-2 {
		display = display[:i.width-5] + "..."
	}
	if i.focused {
		display += strings.Repeat(" ", i.width-2-len(display)) + "█"
	}
	return "[ " + display + " ]"
}

func (i *Input) HandleEvent(msg mofu.Msg) mofu.Cmd {
	if !i.focused {
		return nil
	}
	switch msg := msg.(type) {
	case mofu.KeyPressMsg:
		for _, b := range msg.Runes {
			switch b {
			case '\r', '\n':
				if i.onSubmit != nil {
					return i.onSubmit(i.value)
				}
			case '\x7f', '\b':
				if len(i.value) > 0 {
					i.value = i.value[:len(i.value)-1]
				}
			default:
				if b >= 32 && b <= 126 {
					i.value += string(b)
				}
			}
		}
	}
	return nil
}

func (i *Input) Mount() mofu.Cmd { return nil }
func (i *Input) Unmount()        {}
