package mofu

// ---------------------------------------------------------------------------
// Simplified API — MOFU as simple as possible
// ---------------------------------------------------------------------------
//
// Usage:
//
//	type myModel struct {
//	    mofu.Minimal
//	    count int
//	}
//
//	func (m *myModel) Render(ctx *mofu.RenderContext) {
//	    ctx.Renderer.WriteString(fmt.Sprintf("Count: %d", m.count), 0, 0, ...)
//	}
//
//	func (m *myModel) HandleEvent(event mofu.Event) mofu.Cmd {
//	    if event.Type == mofu.EventKeyPress {
//	        ke := event.Data.(mofu.KeyEvent)
//	        if ke.Key == mofu.KeyUp { m.count++ }
//	        if ke.Key == mofu.KeyDown { m.count-- }
//	        if ke.Key == mofu.KeyEsc { return mofu.Quit() }
//	    }
//	    return nil
//	}
//
//	func main() {
//	    mofu.Run(&myModel{})
//	}

// Minimal provides default implementations for all Node methods except
// Render and HandleEvent. Embed this in your model to get started fast.
type Minimal struct {
	style    Style
	bounds   Rect
	dirty    bool
	children []Node
}

func (m *Minimal) Mount() Cmd                  { return nil }
func (m *Minimal) Unmount()                    {}
func (m *Minimal) Children() []Node            { return m.children }
func (m *Minimal) AddChild(child Node)         { m.children = append(m.children, child); m.dirty = true }
func (m *Minimal) RemoveChild(child Node) {
	for i, c := range m.children {
		if c == child {
			m.children = append(m.children[:i], m.children[i+1:]...)
			m.dirty = true
			return
		}
	}
}
func (m *Minimal) SetDirty()           { m.dirty = true }
func (m *Minimal) Dirty() bool         { return m.dirty }
func (m *Minimal) Bounds() Rect        { return m.bounds }
func (m *Minimal) SetBounds(r Rect)    { m.bounds = r }
func (m *Minimal) Style() *Style       { return &m.style }

// Run is the simplest way to start a MOFU application.
// It creates a Program with sensible defaults and runs it.
func Run(model Node) error {
	return RunWithOpts(model)
}

// RunWithOpts starts a MOFU application with options.
func RunWithOpts(model Node, opts ...Option) error {
	opts = append([]Option{
		WithTheme(DefaultTheme()),
		WithFPS(60),
	}, opts...)
	p := New(model, opts...)
	return p.Run()
}

// QuitCmd returns a Cmd that sends a quit message. Return this from HandleEvent to exit.
func QuitCmd() Cmd {
	return func() Msg { return QuitMsg{} }
}

// SendCmd returns a Cmd that sends an arbitrary message.
func SendCmd(msg Msg) Cmd {
	return func() Msg { return msg }
}
