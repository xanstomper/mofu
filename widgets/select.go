package widgets

import (
	"github.com/xanstomper/mofu"
)

// Select is a dropdown select widget.
type Select struct {
	mofu.BaseNode
	Options  []string
	Selected int
	Open     bool
	Focused  bool
	OnChange func(index int) mofu.Cmd
	Style    mofu.Style
	FocusStyle mofu.Style
}

// NewSelect creates a select widget with options.
func NewSelect(options []string) *Select {
	return &Select{
		Options: options,
		Style: mofu.DefaultStyle().Fg(mofu.Hex("e0e0e0")),
		FocusStyle: mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")),
	}
}

func (s *Select) Focus()   { s.Focused = true; s.SetDirty() }
func (s *Select) Blur()    { s.Focused = false; s.Open = false; s.SetDirty() }
func (s *Select) IsFocused() bool { return s.Focused }

func (s *Select) SetSelected(i int) {
	if i >= 0 && i < len(s.Options) {
		s.Selected = i
		s.SetDirty()
	}
}

func (s *Select) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	if r.Width <= 0 || r.Height <= 0 {
		return
	}

	style := s.Style
	if s.Focused {
		style = s.FocusStyle
	}

	// Current value
	current := ""
	if s.Selected >= 0 && s.Selected < len(s.Options) {
		current = s.Options[s.Selected]
	}
	if len(current) > r.Width-4 {
		current = current[:r.Width-7] + "..."
	}

	// Draw select box
	text := current + " ▾"
	if len(text) > r.Width-2 {
		text = text[:r.Width-5] + "..."
	}
	ctx.Renderer.WriteString(text, r.X, r.Y, style.Foreground, style.Background, style.Attrs)

	// Draw dropdown if open
	if s.Open && s.Focused {
		maxVisible := r.Height - 1
		if maxVisible > len(s.Options) {
			maxVisible = len(s.Options)
		}
		for i := 0; i < maxVisible; i++ {
			y := r.Y + 1 + i
			if y >= r.Y+r.Height {
				break
			}
			opt := s.Options[i]
			if len(opt) > r.Width-4 {
				opt = opt[:r.Width-7] + "..."
			}
			optStyle := s.Style
			if i == s.Selected {
				optStyle = s.FocusStyle
				ctx.Renderer.WriteString(" ", r.X, y, optStyle.Foreground, optStyle.Background, optStyle.Attrs)
			}
			ctx.Renderer.WriteString(opt, r.X+1, y, optStyle.Foreground, optStyle.Background, optStyle.Attrs)
		}
	}
}

func (s *Select) HandleEvent(event mofu.Event) mofu.Cmd {
	if !s.Focused || event.Type != mofu.EventKeyPress {
		return nil
	}
	ke, ok := event.Data.(mofu.KeyEvent)
	if !ok {
		return nil
	}

	if !s.Open {
		if ke.Key == mofu.KeyEnter || (len(ke.Runes) > 0 && ke.Runes[0] == ' ') {
			s.Open = true
			s.SetDirty()
			return nil
		}
		return nil
	}

	switch ke.Key {
	case mofu.KeyEsc:
		s.Open = false
		s.SetDirty()
	case mofu.KeyDown:
		if s.Selected < len(s.Options)-1 {
			s.Selected++
			s.SetDirty()
		}
	case mofu.KeyUp:
		if s.Selected > 0 {
			s.Selected--
			s.SetDirty()
		}
	case mofu.KeyEnter:
		s.Open = false
		s.SetDirty()
		if s.OnChange != nil {
			return s.OnChange(s.Selected)
		}
	}
	return nil
}

func (s *Select) Mount() mofu.Cmd { return nil }
func (s *Select) Unmount()        {}
