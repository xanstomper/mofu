package mofu

import (
	"context"
	"os"
	"os/signal"
	"sync"

	"golang.org/x/term"
)

// Program is a Mofu application instance.
type Program struct {
	root     Component
	tree     *Tree
	renderer *Renderer
	theme    *Theme
	animator *Animator
	eventBus *EventBus
	width    int
	height   int
	running  bool
	mu       sync.Mutex
	ctx      context.Context
	cancel   context.CancelFunc
	oldState *term.State
	output   *os.File
}

// New creates a new Mofu program with the given root component.
func New(root Component, opts ...Option) *Program {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Program{
		root:     root,
		tree:     NewTree(root),
		theme:    DefaultTheme(),
		animator: NewAnimator(),
		eventBus: NewEventBus(),
		ctx:      ctx,
		cancel:   cancel,
		output:   os.Stdout,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Option configures a Program.
type Option func(*Program)

// WithTheme sets the program theme.
func WithTheme(t *Theme) Option {
	return func(p *Program) { p.theme = t }
}

// WithSize sets the initial terminal size.
func WithSize(w, h int) Option {
	return func(p *Program) { p.width = w; p.height = h }
}

// Run starts the program event loop.
func (p *Program) Run() error {
	// Get terminal size
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width = 80
		height = 24
	}
	p.width = width
	p.height = height

	// Enable raw mode
	p.oldState, err = term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	defer term.Restore(int(os.Stdin.Fd()), p.oldState)

	p.running = true
	p.renderer = NewRenderer(p.width, p.height, p.theme)

	// Mount all components
	cmds := p.tree.Root.MountAll()

	// Clear screen and hide cursor
	os.Stdout.WriteString("\x1b[2J\x1b[?25l")

	// Initial render
	p.render()

	// Execute initial commands
	for _, cmd := range cmds {
		if cmd != nil {
			go func(c Cmd) { c() }(cmd)
		}
	}

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, os.Kill)

	// Read input in a goroutine
	inputCh := make(chan []byte, 64)
	go p.readInput(inputCh)

	for p.running {
		select {
		case <-p.ctx.Done():
			p.running = false
		case sig := <-sigCh:
			switch sig {
			case os.Interrupt:
				p.Quit()
			}
		case buf := <-inputCh:
			p.handleInput(buf)
		}
	}

	// Show cursor and clear
	os.Stdout.WriteString("\x1b[?25h\x1b[2J\x1b[1;1H")
	return nil
}

func (p *Program) readInput(ch chan<- []byte) {
	buf := make([]byte, 128)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			return
		}
		tmp := make([]byte, n)
		copy(tmp, buf[:n])
		ch <- tmp
	}
}

func (p *Program) handleInput(buf []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(buf) == 0 {
		return
	}

	// Check for escape sequences
	if buf[0] == 0x1b && len(buf) > 1 {
		p.handleEscape(buf)
		return
	}

	// Regular key press
	msg := KeyPressMsg{Runes: buf}
	p.eventBus.Publish(EventKeyPress, msg)
	cmd := p.root.HandleEvent(msg)
	if cmd != nil {
		go cmd()
	}
	p.render()
}

func (p *Program) handleEscape(buf []byte) {
	if len(buf) >= 3 && buf[1] == '[' {
		switch buf[2] {
		case 'A': // Up
			msg := KeyPressMsg{Runes: buf, Key: KeyUp}
			p.dispatch(msg)
		case 'B': // Down
			msg := KeyPressMsg{Runes: buf, Key: KeyDown}
			p.dispatch(msg)
		case 'C': // Right
			msg := KeyPressMsg{Runes: buf, Key: KeyRight}
			p.dispatch(msg)
		case 'D': // Left
			msg := KeyPressMsg{Runes: buf, Key: KeyLeft}
			p.dispatch(msg)
		default:
			msg := KeyPressMsg{Runes: buf}
			p.dispatch(msg)
		}
		return
	}
	msg := KeyPressMsg{Runes: buf}
	p.dispatch(msg)
}

func (p *Program) dispatch(msg Msg) {
	p.eventBus.Publish(EventKeyPress, msg)
	cmd := p.root.HandleEvent(msg)
	if cmd != nil {
		go cmd()
	}
	p.render()
}

func (p *Program) handleResize() {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return
	}
	p.mu.Lock()
	p.width = width
	p.height = height
	if p.renderer != nil {
		p.renderer.Resize(width, height)
	}
	p.mu.Unlock()
	p.eventBus.Publish(EventResize, ResizeMsg{Width: width, Height: height})
	p.render()
}

func (p *Program) render() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.renderer == nil {
		return
	}
	p.renderer.Clear()

	// Compute layout
	bounds := Rect{0, 0, p.width, p.height}
	rootNode := &LayoutNode{
		Component: p.root,
		Layout:    LayoutColumn,
		Visible:   true,
		Rect:      bounds,
	}
	ComputeLayout(rootNode, bounds)

	// Render component tree
	p.renderNode(rootNode)

	// Flush to terminal
	output := p.renderer.Flush()
	if output != "" {
		os.Stdout.WriteString(output)
	}
}

func (p *Program) renderNode(node *LayoutNode) {
	if !node.Visible {
		return
	}
	text := node.Component.Render()
	if text != "" {
		r := node.Rect
		style := DefaultStyle().Fg(p.theme.Colors.Text).Bg(p.theme.Colors.Background)
		p.renderer.WriteStyledString(text, r.X, r.Y, style)
	}
	for _, child := range node.Children {
		p.renderNode(child)
	}
}

// Quit stops the program.
func (p *Program) Quit() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.running = false
	p.cancel()
	p.tree.Root.UnmountAll()
}

// Send sends a message to the root component.
func (p *Program) Send(msg Msg) {
	p.mu.Lock()
	defer p.mu.Unlock()
	cmd := p.root.HandleEvent(msg)
	if cmd != nil {
		go cmd()
	}
	p.render()
}

// Resize updates the program dimensions.
func (p *Program) Resize(w, h int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.width = w
	p.height = h
	if p.renderer != nil {
		p.renderer.Resize(w, h)
	}
}

// Theme returns the program's theme.
func (p *Program) Theme() *Theme { return p.theme }

// Animator returns the program's animator.
func (p *Program) Animator() *Animator { return p.animator }

// EventBus returns the program's event bus.
func (p *Program) EventBus() *EventBus { return p.eventBus }

// KeyPressMsg is sent when a key is pressed.
type KeyPressMsg struct {
	Runes []byte
	Key   Key
}

// Key represents a special key.
type Key int

const (
	KeyNone  Key = 0
	KeyUp    Key = 1
	KeyDown  Key = 2
	KeyRight Key = 3
	KeyLeft  Key = 4
	KeyEnter Key = 5
	KeyEsc   Key = 6
	KeyTab   Key = 7
	KeySpace Key = 8
	KeyBack  Key = 9
)

// ResizeMsg is sent when the terminal is resized.
type ResizeMsg struct {
	Width  int
	Height int
}
