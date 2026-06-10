package mofu

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"time"

	"golang.org/x/term"
)

type Program struct {
	root Node
	renderer *Renderer
	theme *Theme
	scheduler *Scheduler
	animator *Animator
	eventBus *EventBus
	dataStore *DataStore
	width, height int
	running bool
	mu sync.Mutex
	ctx context.Context
	cancel context.CancelFunc

	eventCh chan Event
	renderCh chan struct{}

	oldState *term.State

	channels []OutputChannel
	sm *StateMachine
	rt *Runtime
	stateGraph *StateGraph
}

type Option func(*Program)

func WithTheme(t *Theme) Option {
	return func(p *Program) { p.theme = t }
}

func WithSize(w, h int) Option {
	return func(p *Program) { p.width = w; p.height = h }
}

func WithOutput(ch OutputChannel) Option {
	return func(p *Program) { p.channels = append(p.channels, ch) }
}

func WithStateGraph(g *StateGraph) Option {
	return func(p *Program) { p.stateGraph = g }
}

func New(root Node, opts ...Option) *Program {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Program{
		root: root,
		theme: DefaultTheme(),
		animator: NewAnimator(),
		scheduler: NewScheduler(60),
		eventBus: NewEventBus(),
		dataStore: NewDataStore(),
		eventCh: make(chan Event, 64),
		renderCh: make(chan struct{}, 1),
		ctx: ctx,
		cancel: cancel,
		sm: newStateMachine(StateInit),
	}
	p.rt = NewRuntime("local", "program", RuntimeConfig{})
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func (p *Program) Run() error {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width = 80
		height = 24
	}
	p.width = width
	p.height = height

	p.oldState, err = term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	defer term.Restore(int(os.Stdin.Fd()), p.oldState)

	p.running = true
	p.sm.TransitionTo(StateReady)
	p.renderer = NewRenderer(p.width, p.height, p.theme)

	os.Stdout.WriteString("\x1b[2J\x1b[?25l")

	cmds := p.root.Mount()
	if cmds != nil {
		go cmds()
	}

	p.scheduler.Start()
	go p.eventLoop()
	go p.renderLoop()
	p.stateLoop()

	return nil
}

func (p *Program) eventLoop() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	buf := make([]byte, 128)
	for p.running {
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			continue
		}
		data := make([]byte, n)
		copy(data, buf[:n])
		p.handleRawInput(data)
	}
}

func (p *Program) handleRawInput(data []byte) {
	if len(data) == 0 {
		return
	}
	if data[0] == 0x1b && len(data) > 1 {
		p.handleEscapeSeq(data)
		return
	}
	ev := Event{
		Type: EventKeyPress,
		Data: KeyEvent{Runes: data},
		Time: time.Now(),
	}
	p.eventBus.Publish(ev)
	select {
	case p.eventCh <- ev:
	default:
	}
}

func (p *Program) handleEscapeSeq(data []byte) {
	if len(data) >= 3 && data[1] == '[' {
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
		default:
			if data[2] >= '1' && data[2] <= '6' && len(data) >= 4 {
				switch data[2] {
				case '1':
					if len(data) >= 4 && data[3] == '~' {
						key = KeyHome
					}
				case '2':
					key = KeyF1
				case '3':
					key = KeyF2
				case '4':
					key = KeyEnd
				case '5':
					key = KeyPgUp
				case '6':
					key = KeyPgDn
				}
			}
		}
		ev := Event{
			Type: EventKeyPress,
			Data: KeyEvent{Runes: data, Key: key},
			Time: time.Now(),
		}
		p.eventBus.Publish(ev)
		select {
		case p.eventCh <- ev:
		default:
		}
	}
}

func (p *Program) stateLoop() {
	for p.running {
		select {
		case <-p.ctx.Done():
			return
		case ev := <-p.eventCh:
			p.mu.Lock()
			cmd := p.root.HandleEvent(ev)
			p.mu.Unlock()
			if cmd != nil {
				go func() {
					msg := cmd()
					if msg != nil {
						select {
						case p.eventCh <- Event{Type: EventSystem, Data: msg, Time: time.Now()}:
						default:
						}
					}
				}()
			}
			p.requestRender()
		case sig := <-signalCh():
			switch sig {
			case os.Interrupt:
				p.Quit()
				return
			}
		}
	}
}

func signalCh() chan os.Signal {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	return ch
}

func (p *Program) requestRender() {
	select {
	case p.renderCh <- struct{}{}:
	default:
	}
}

func (p *Program) renderLoop() {
	for p.running {
		select {
		case <-p.ctx.Done():
			return
		case tick := <-p.scheduler.FrameCh():
			p.mu.Lock()
			p.animator.Update(tick.Delta)
			p.renderer.Clear()

			bounds := Rect{0, 0, p.width, p.height}
			ComputeLayout(p.root, bounds)

			ctx := &RenderContext{
				Renderer: p.renderer,
				Theme:    p.theme,
				Frame:    tick.Frame,
				Delta:    tick.Delta,
				Bounds:   bounds,
			}
			p.root.Render(ctx)

			output := p.renderer.Flush()
			if output != "" {
				os.Stdout.WriteString(output)
			}
			p.mu.Unlock()
		}
	}
}

func (p *Program) Quit() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.running = false
	p.cancel()
	if p.sm != nil {
		p.sm.TransitionTo(StateStopping)
		p.sm.TransitionTo(StateDone)
	}
	p.scheduler.Stop()
	p.root.Unmount()
	os.Stdout.WriteString("\x1b[?25h\x1b[2J\x1b[1;1H")
}

func (p *Program) Send(msg Msg) {
	p.mu.Lock()
	defer p.mu.Unlock()
	ev := Event{Type: EventSystem, Data: msg, Time: time.Now()}
	cmd := p.root.HandleEvent(ev)
	if cmd != nil {
		go cmd()
	}
	p.requestRender()
}

func (p *Program) Resize(w, h int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.width = w
	p.height = h
	if p.renderer != nil {
		p.renderer.Resize(w, h)
	}
	p.requestRender()
}

func (p *Program) Theme() *Theme         { return p.theme }
func (p *Program) Renderer() *Renderer   { return p.renderer }
func (p *Program) Scheduler() *Scheduler { return p.scheduler }
func (p *Program) EventBus() *EventBus   { return p.eventBus }
func (p *Program) DataStore() *DataStore { return p.dataStore }
