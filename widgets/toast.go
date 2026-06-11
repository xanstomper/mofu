package widgets

import (
	"time"

	"github.com/xanstomper/mofu"
)

// Toast is a temporary notification message.
type Toast struct {
	mofu.BaseNode
	Message  string
	Level    ToastLevel
	Visible  bool
	Expires  time.Time
	Duration time.Duration
}

type ToastLevel int

const (
	ToastInfo ToastLevel = iota
	ToastSuccess
	ToastWarning
	ToastError
)

// NewToast creates a toast notification.
func NewToast(message string, level ToastLevel) *Toast {
	return NewToastDuration(message, level, 3*time.Second)
}

// NewToastDuration creates a toast with custom duration.
func NewToastDuration(message string, level ToastLevel, duration time.Duration) *Toast {
	return &Toast{
		Message:  message,
		Level:    level,
		Visible:  true,
		Duration: duration,
		Expires:  time.Now().Add(duration),
	}
}

func (t *Toast) Show() {
	t.Visible = true
	t.Expires = time.Now().Add(t.Duration)
	t.SetDirty()
}

func (t *Toast) Dismiss() {
	t.Visible = false
	t.SetDirty()
}

func (t *Toast) Render(ctx *mofu.RenderContext) {
	if !t.Visible || time.Now().After(t.Expires) {
		t.Visible = false
		return
	}

	r := ctx.Bounds
	if r.Width <= 0 || r.Height <= 0 {
		return
	}

	// Toast color by level
	var fg, bg mofu.Color
	switch t.Level {
	case ToastSuccess:
		fg, bg = mofu.Hex("a6e3a1"), mofu.Hex("1a1a2e")
	case ToastWarning:
		fg, bg = mofu.Hex("f9e2af"), mofu.Hex("1a1a2e")
	case ToastError:
		fg, bg = mofu.Hex("f38ba8"), mofu.Hex("1a1a2e")
	default:
		fg, bg = mofu.Hex("7dcfff"), mofu.Hex("1a1a2e")
	}

	// Icons
	icon := "ℹ"
	switch t.Level {
	case ToastSuccess:
		icon = "✓"
	case ToastWarning:
		icon = "⚠"
	case ToastError:
		icon = "✗"
	}

	text := icon + " " + t.Message
	if len(text) > r.Width-2 {
		text = text[:r.Width-5] + "..."
	}

	// Position at bottom
	y := r.Y + r.Height - 1
	ctx.Renderer.WriteString(" "+text+" ", r.X+1, y, fg, bg, 0)
}

func (t *Toast) HandleEvent(event mofu.Event) mofu.Cmd {
	if event.Type == mofu.EventKeyPress {
		ke, ok := event.Data.(mofu.KeyEvent)
		if ok && (ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q')) {
			t.Dismiss()
		}
	}
	return nil
}

func (t *Toast) Mount() mofu.Cmd { return nil }
func (t *Toast) Unmount()        {}
