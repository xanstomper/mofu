package mofu

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"time"

	"golang.org/x/term"

	"github.com/anomalyco/mofu/kernel"
	"github.com/anomalyco/mofu/message"
	"github.com/anomalyco/mofu/state"
)

// Program represents a MOFU terminal application.
// It wraps the v2 kernel.Kernel with backward-compatible Node interface support.
type Program struct {
	root Node

	// v2 kernel — owns event loop, state graph, effect system, render scheduling
	kern *kernel.Kernel

	// Render
	renderer      *Renderer
	theme         *Theme
	width, height int

	// Legacy subsystems (kept for backward compat)
	scheduler *Scheduler
	animator  *Animator
	eventBus  *EventBus
	dataStore *DataStore

	running bool
	mu      sync.Mutex
	ctx     context.Context
	cancel  context.CancelFunc

	oldState *term.State
	channels []OutputChannel
	sm       *StateMachine
	rt       *Runtime
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

// New creates a Program rooted at the given Node.
// The kernel is created internally with default config.
func New(root Node, opts ...Option) *Program {
	ctx, cancel := context.WithCancel(context.Background())

	// Use fast-path-only kernel by default — no plugin overhead
	k := kernel.New(kernel.Config{
		FPSCap:       60,
		EventBufSize: 64,
		MaxTasks:     100,
		FastPathOnly: false,
	})

	p := &Program{
		root:      root,
		kern:      k,
		theme:     DefaultTheme(),
		animator:  NewAnimator(),
		scheduler: NewScheduler(60),
		eventBus:  NewEventBus(),
		dataStore: NewDataStore(),
		ctx:       ctx,
		cancel:    cancel,
		sm:        newStateMachine(StateInit),
		rt:        NewRuntime("local", "program", RuntimeConfig{}),
	}

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

	p.running = true
	p.sm.TransitionTo(StateReady)
	p.renderer = NewRenderer(p.width, p.height, p.theme)

	os.Stdout.WriteString("\x1b[2J\x1b[?25l")

	// Mount root node
	cmds := p.root.Mount()
	if cmds != nil {
		go cmds()
	}

	// ---------------------------------------------------------------
	// Wire v2 kernel callbacks — this is the main integration point
	// ---------------------------------------------------------------

	// Layout callback — compute widget tree layout each frame
	p.kern.OnLayout(func() {
		bounds := Rect{0, 0, p.width, p.height}
		ComputeLayout(p.root, bounds)
	})

	// UI materialization callback — render widget tree each frame
	p.kern.OnUI(func() any {
		p.animator.Update(0)
		ctx := &RenderContext{
			Renderer: p.renderer,
			Theme:    p.theme,
			Frame:    p.kern.FrameCount(),
			Delta:    time.Second / time.Duration(p.kern.FrameCount()),
			Bounds:   Rect{0, 0, p.width, p.height},
		}
		p.root.Render(ctx)
		return nil
	})

	// Render callback — flush diff to terminal
	p.kern.OnRender(func(dt time.Duration) {
		output := p.renderer.Flush()
		if output != "" {
			os.Stdout.WriteString(output)
		}
	})

	// Wire stdin → kernel message bus (fast path)
	go p.stdinLoop()

	// Wire root.HandleEvent → kernel message subscribers
	p.kern.Bus.Subscribe(message.TypeInput, func(msg message.Message) {
		data, ok := msg.Payload.([]byte)
		if !ok || len(data) == 0 {
			return
		}
		p.mu.Lock()
		ev := p.parseInput(data)
		p.mu.Unlock()
		if ev != nil {
			p.handleEvent(*ev)
		}
	})

	p.kern.Bus.Subscribe(message.TypeCommand, func(msg message.Message) {
		p.mu.Lock()
		ev := Event{Type: EventSystem, Data: msg.Payload, Time: time.Now()}
		cmd := p.root.HandleEvent(ev)
		p.mu.Unlock()
		if cmd != nil {
			go func() {
				if m := cmd(); m != nil {
					p.kern.Bus.Publish(message.NewMessage(message.TypeCustom, m))
				}
			}()
		}
	})

	p.kern.Bus.Subscribe(message.TypeResize, func(msg message.Message) {
		if dims, ok := msg.Payload.([2]int); ok {
			p.width = dims[0]
			p.height = dims[1]
			p.renderer.Resize(p.width, p.height)
		}
	})

	// State change callback — mark widgets dirty when state changes
	p.kern.OnStateChange(func(id state.NodeID, oldVal, newVal any) {
		p.root.SetDirty()
	})

	// ---------------------------------------------------------------
	// Start the kernel — this blocks until Stop
	// ---------------------------------------------------------------
	p.kern.Init()

	// Set up signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	go func() {
		<-sigCh
		p.Quit()
	}()

	defer func() {
		term.Restore(int(os.Stdin.Fd()), p.oldState)
		os.Stdout.WriteString("\x1b[?25h\x1b[2J\x1b[1;1H")
	}()

	p.kern.Run()

	return nil
}

// stdinLoop reads raw bytes from stdin and publishes them to the kernel message bus.
func (p *Program) stdinLoop() {
	buf := make([]byte, 128)
	for p.running {
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			continue
		}
		data := make([]byte, n)
		copy(data, buf[:n])
		p.kern.Bus.Publish(message.NewInput(data))
	}
}

// parseInput converts raw stdin bytes into an Event for the Node tree.
func (p *Program) parseInput(data []byte) *Event {
	if len(data) == 0 {
		return nil
	}
	if data[0] == 0x1b && len(data) > 1 {
		return p.parseEscapeSeq(data)
	}
	ev := Event{
		Type: EventKeyPress,
		Data: KeyEvent{Runes: data},
		Time: time.Now(),
	}
	p.eventBus.Publish(ev)
	return &ev
}

func (p *Program) parseEscapeSeq(data []byte) *Event {
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
		return &ev
	}
	return nil
}

// handleEvent dispatches an Event to the root Node tree.
func (p *Program) handleEvent(ev Event) {
	cmd := p.root.HandleEvent(ev)
	if cmd != nil {
		go func() {
			if m := cmd(); m != nil {
				p.kern.Bus.Publish(message.NewMessage(message.TypeCustom, m))
			}
		}()
	}
}

func (p *Program) Quit() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.running {
		return
	}
	p.running = false
	p.cancel()
	if p.sm != nil {
		p.sm.TransitionTo(StateStopping)
		p.sm.TransitionTo(StateDone)
	}
	p.kern.Stop()
	p.root.Unmount()
}

func (p *Program) Send(msg Msg) {
	p.kern.Bus.Publish(message.NewMessage(message.TypeCommand, msg))
}

func (p *Program) Resize(w, h int) {
	p.kern.Bus.Publish(message.Message{
		Type:    message.TypeResize,
		Payload: [2]int{w, h},
	})
}

func (p *Program) Theme() *Theme          { return p.theme }
func (p *Program) Renderer() *Renderer    { return p.renderer }
func (p *Program) Scheduler() *Scheduler  { return p.scheduler }
func (p *Program) EventBus() *EventBus    { return p.eventBus }
func (p *Program) DataStore() *DataStore  { return p.dataStore }
func (p *Program) Kernel() *kernel.Kernel { return p.kern }
