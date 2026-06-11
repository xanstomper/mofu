package mofu

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"golang.org/x/term"

	"github.com/xanstomper/mofu/kernel"
	"github.com/xanstomper/mofu/message"
	"github.com/xanstomper/mofu/state"
)

var (
	ErrProgramPanic  = fmt.Errorf("mofu: program experienced a panic")
	ErrProgramKilled = fmt.Errorf("mofu: program was killed")
	ErrInterrupted   = fmt.Errorf("mofu: program was interrupted")
)

type channelHandlers struct {
	mu       sync.RWMutex
	handlers []chan struct{}
}

func (h *channelHandlers) add(ch chan struct{}) {
	h.mu.Lock()
	h.handlers = append(h.handlers, ch)
	h.mu.Unlock()
}

func (h *channelHandlers) shutdown() {
	var wg sync.WaitGroup
	h.mu.RLock()
	for _, ch := range h.handlers {
		wg.Add(1)
		go func(ch chan struct{}) {
			<-ch
			wg.Done()
		}(ch)
	}
	h.mu.RUnlock()
	wg.Wait()
}

type Option func(*Program)

func WithTheme(t *Theme) Option {
	return func(p *Program) { p.theme = t }
}

func WithSize(w, h int) Option {
	return func(p *Program) { p.width.Store(int32(w)); p.height.Store(int32(h)) }
}

func WithFPS(fps int) Option {
	return func(p *Program) {
		if fps > 0 && fps <= 120 {
			p.fps = fps
		}
	}
}

func WithInput(r io.Reader) Option {
	return func(p *Program) { p.input = r }
}

func WithOutputWriter(w io.Writer) Option {
	return func(p *Program) { p.output = w }
}

func WithoutRenderer() Option {
	return func(p *Program) { p.disableRenderer = true }
}

func WithoutSignalHandler() Option {
	return func(p *Program) { p.disableSignalHandler = true }
}

func WithoutCatchPanics() Option {
	return func(p *Program) { p.disableCatchPanics = true }
}

func WithHardTabs() Option {
	return func(p *Program) { p.useHardTabs = true }
}

func WithBackspace() Option {
	return func(p *Program) { p.useBackspace = true }
}

func WithInputEnabled(enabled bool) Option {
	return func(p *Program) { p.disableInput = !enabled }
}

type Program struct {
	root Node
	kern *kernel.Kernel

	renderer      *Renderer
	theme         *Theme
	width, height atomic.Int32
	animator      *Animator
	eventBus      *EventBus
	dataStore     *DataStore

	running     atomic.Bool
	mu          sync.Mutex
	ctx         context.Context
	cancel      context.CancelFunc
	externalCtx context.Context

	oldState     *term.State
	finished     chan struct{}
	shutdownOnce sync.Once
	handlers     channelHandlers

	sm *StateMachine
	rt *Runtime

	fps                  int
	disableRenderer      bool
	disableInput         bool
	disableSignalHandler bool
	disableCatchPanics   bool
	useHardTabs          bool
	useBackspace         bool

	readLoopDone chan struct{}
	rendererDone chan struct{}
	once         sync.Once

	cancelReader cancelReader
	input        io.Reader
	output       io.Writer
	logger       *log.Logger
	environ      []string

	msgs chan Msg
	errs chan error
}

type cancelReader interface {
	Cancel() bool
	Close() error
}

type ioReader interface {
	Read([]byte) (int, error)
}

type ioWriter interface {
	Write([]byte) (int, error)
}

// New creates a new MOFU Program.
func New(root Node, opts ...Option) *Program {
	if root == nil {
		panic("mofu: root model cannot be nil")
	}

	ctx, cancel := context.WithCancel(context.Background())

	k := kernel.New(kernel.Config{
		FPSCap:       60,
		EventBufSize: 64,
		MaxTasks:     100,
		FastPathOnly: false,
	})

	p := &Program{
		root:        root,
		kern:        k,
		theme:       DefaultTheme(),
		animator:    NewAnimator(),
		ctx:         ctx,
		cancel:      cancel,
		finished:    make(chan struct{}),
		errs:        make(chan error, 1),
		msgs:        make(chan Msg, 64),
		input:       os.Stdin,
		output:      os.Stdout,
		logger:      nil,
		environ:     os.Environ(),
		fps:         60,
		sm:          newStateMachine(StateInit),
		rt:          NewRuntime("local", "program", RuntimeConfig{}),
	}

	for _, opt := range opts {
		opt(p)
	}

	if path, ok := os.LookupEnv("MOFU_TRACE"); ok && path != "" {
		if f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o600); err == nil {
			p.logger = log.New(f, "mofu: ", log.LstdFlags|log.Lshortfile)
		}
	}

	return p
}

// Run initializes the terminal and starts the MOFU event loop.
func (p *Program) Run() error {
	if p.root == nil {
		return fmt.Errorf("mofu: root model cannot be nil")
	}

	if !p.disableCatchPanics {
		defer func() {
			if r := recover(); r != nil {
				p.recoverFromPanic(r)
				p.kern.Stop()
				p.shutdown(true)
			}
		}()
	}

	width, height, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		width, height = 80, 24
	}
	p.width.Store(int32(width))
	p.height.Store(int32(height))

	p.oldState, err = term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}

	p.running.Store(true)
	p.sm.TransitionTo(StateReady)

	p.kern.SetTerminalSize(int(p.width.Load()), int(p.height.Load()))

	if !p.disableRenderer {
		p.renderer = NewRenderer(int(p.width.Load()), int(p.height.Load()), p.theme)
	}

	os.Stdout.WriteString("\x1b[2J\x1b[?25l")
	p.handlers = channelHandlers{}
	cmds := make(chan Cmd)
	p.finished = make(chan struct{})
	defer close(p.finished)
	defer p.cancel()

	if !p.disableSignalHandler {
		p.handlers.add(p.handleSignals())
	}

	if mountCmd := p.root.Mount(); mountCmd != nil {
		ch := make(chan struct{})
		p.handlers.add(ch)
		go func() {
			defer close(ch)
			select {
			case cmds <- mountCmd:
			case <-p.ctx.Done():
			}
		}()
	}

	p.kern.OnLayout(func() {
		bounds := Rect{0, 0, int(p.width.Load()), int(p.height.Load())}
		ComputeLayout(p.root, bounds)
	})

	p.kern.OnUI(func() any {
		p.animator.Update(uint64(p.kern.LastDelta().Milliseconds()))
		if p.renderer != nil {
			ctx := &RenderContext{
				Renderer: p.renderer,
				Theme:    p.theme,
				Frame:    p.kern.FrameCount(),
				Bounds:   Rect{0, 0, int(p.width.Load()), int(p.height.Load())},
			}
			p.root.Render(ctx)
		}
		return nil
	})

	p.kern.OnRender(func(dt time.Duration) {
		p.flushFrame()
	})

	p.kern.OnStateChange(func(id state.NodeID, oldVal, newVal any) {
		p.root.SetDirty()
	})

	p.kern.Bus.Subscribe(message.TypeInput, func(msg message.Message) {
		data, ok := msg.Payload.([]byte)
		if !ok || len(data) == 0 {
			return
		}
		p.mu.Lock()
		ev := p.parseInput(data)
		p.mu.Unlock()
		if ev != nil {
			p.dispatchEvent(*ev)
		}
	})

	p.kern.Bus.Subscribe(message.TypeCommand, func(msg message.Message) {
		p.mu.Lock()
		cmd := p.root.HandleEvent(Event{
			Type: EventSystem,
			Data: msg.Payload,
			Time: time.Now(),
		})
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
			p.width.Store(int32(dims[0]))
			p.height.Store(int32(dims[1]))
			p.renderer.Resize(dims[0], dims[1])
			p.kern.SetTerminalSize(dims[0], dims[1])
		}
	})

	go p.stdinLoop()
	p.handlers.add(p.handleCommands(cmds))
	p.kern.Init()
	p.renderFrame()
	p.kern.Run()

	p.restoreTerminal()
	return nil
}

func (p *Program) handleCommands(cmds chan Cmd) chan struct{} {
	ch := make(chan struct{})
	go func() {
		defer close(ch)
		for {
			select {
			case <-p.ctx.Done():
				return
			case cmd, ok := <-cmds:
				if !ok || cmd == nil {
					continue
				}
				go func() {
					if !p.disableCatchPanics {
						defer func() {
							if r := recover(); r != nil {
								p.recoverFromGoPanic(r)
							}
						}()
					}
					result := cmd()
					switch msg := result.(type) {
					case BatchMsg:
						p.execBatch(msg)
					case SequenceMsg:
						p.execSequence(msg)
					default:
						if result != nil {
							p.kern.Bus.Publish(message.NewMessage(message.TypeCustom, result))
						}
					}
				}()
			}
		}
	}()
	return ch
}

func (p *Program) execBatch(cmds BatchMsg) {
	var wg sync.WaitGroup
	for _, cmd := range cmds {
		if cmd == nil {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if !p.disableCatchPanics {
				defer func() {
					if r := recover(); r != nil {
						p.recoverFromGoPanic(r)
					}
				}()
			}
			result := cmd()
			if result != nil {
				p.kern.Bus.Publish(message.NewMessage(message.TypeCustom, result))
			}
		}()
	}
	wg.Wait()
}

func (p *Program) execSequence(cmds SequenceMsg) {
	for _, cmd := range cmds {
		if cmd == nil {
			continue
		}
		result := cmd()
		switch msg := result.(type) {
		case BatchMsg:
			p.execBatch(msg)
		case SequenceMsg:
			p.execSequence(msg)
		default:
			if result != nil {
				p.kern.Bus.Publish(message.NewMessage(message.TypeCustom, result))
			}
		}
	}
}

func (p *Program) recoverFromPanic(r interface{}) {
	p.kern.Stop()
	p.shutdown(true)
	rec := strings.ReplaceAll(fmt.Sprintf("%s", r), "\n", "\r\n")
	fmt.Fprintf(os.Stderr, "\r\nmofu: panic\r\n\r\n%s\r\n\r\n", rec)
	stack := strings.ReplaceAll(fmt.Sprintf("%s\n", debug.Stack()), "\n", "\r\n")
	fmt.Fprint(os.Stderr, stack)
	if v, err := strconv.ParseBool(os.Getenv("MOFU_TRACE")); err == nil && v {
		f, err := os.Create(fmt.Sprintf("mofu-panic-%d.log", time.Now().Unix()))
		if err == nil {
			defer f.Close()
			fmt.Fprintln(f, rec)
			fmt.Fprintln(f, stack)
		}
	}
}

func (p *Program) recoverFromGoPanic(r interface{}) {
	p.cancel()
	rec := strings.ReplaceAll(fmt.Sprintf("%s", r), "\n", "\r\n")
	fmt.Fprintf(os.Stderr, "\r\nmofu: goroutine panic\r\n\r\n%s\r\n\r\n", rec)
}

func (p *Program) handleSignals() chan struct{} {
	ch := make(chan struct{})
	go func() {
		defer close(ch)
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(sig)
		for {
			select {
			case <-p.ctx.Done():
				return
			case s := <-sig:
				if atomic.LoadUint32(&ignoreSignals) == 0 {
					switch s {
					case syscall.SIGINT:
						p.Send(InterruptMsg{})
					default:
						p.Send(QuitMsg{})
					}
				}
			}
		}
	}()
	return ch
}

// suspend is a no-op on Windows.
func (p *Program) suspend() {}

func (p *Program) renderFrame() {
	if p.renderer == nil || p.root == nil {
		return
	}
	ctx := &RenderContext{
		Renderer: p.renderer,
		Theme:    p.theme,
		Frame:    p.kern.FrameCount(),
		Bounds:   Rect{0, 0, int(p.width.Load()), int(p.height.Load())},
	}
	p.root.Render(ctx)
	output := p.renderer.Flush()
	if output != "" {
		p.output.Write([]byte(output))
	}
}

// flushFrame only flushes the renderer diff to the terminal.
// The scene tree is already rendered by the kernel's onUI callback.
func (p *Program) flushFrame() {
	if p.renderer == nil {
		return
	}
	output := p.renderer.Flush()
	if output != "" {
		p.output.Write([]byte(output))
	}
}

func (p *Program) restoreTerminal() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.oldState != nil {
		_ = term.Restore(int(os.Stdin.Fd()), p.oldState)
		p.oldState = nil
	}
	p.output.Write([]byte("\x1b[?25h\x1b[2J\x1b[1;1H"))
	runtime.GC()
}

func (p *Program) shutdown(kill bool) {
	p.shutdownOnce.Do(func() {
		p.cancel()
		p.handlers.shutdown()
		if p.cancelReader != nil {
			p.cancelReader.Cancel()
			p.cancelReader.Close()
		}
		p.kern.Stop()
		p.restoreTerminal()
	})
}

func (p *Program) stdinLoop() {
	buf := make([]byte, 128)
	batchBuf := make([]byte, 0, 512)
	batchTimer := time.NewTimer(0)
	if !batchTimer.Stop() {
		<-batchTimer.C
	}
	batching := false

	for atomic.LoadUint32(&ignoreSignals) == 0 {
		n, err := p.input.Read(buf)
		if err != nil || n == 0 {
			continue
		}

		if !batching {
			batchBuf = append(batchBuf[:0], buf[:n]...)
			batching = true
			batchTimer.Reset(time.Millisecond)
			continue
		}

		batchBuf = append(batchBuf, buf[:n]...)
		select {
		case <-batchTimer.C:
			if len(batchBuf) > 0 {
				data := make([]byte, len(batchBuf))
				copy(data, batchBuf)
				p.kern.Bus.Publish(message.NewInput(data))
				batchBuf = batchBuf[:0]
			}
			batching = false
		default:
		}
	}

	if len(batchBuf) > 0 {
		data := make([]byte, len(batchBuf))
		copy(data, batchBuf)
		p.kern.Bus.Publish(message.NewInput(data))
	}
	batchTimer.Stop()

	if p.readLoopDone != nil {
		close(p.readLoopDone)
	}
}

func (p *Program) dispatchEvent(ev Event) {
	p.eventBus.Publish(ev)
	cmd := p.root.HandleEvent(ev)
	if cmd != nil {
		go func() {
			if m := cmd(); m != nil {
				p.kern.Bus.Publish(message.NewMessage(message.TypeCustom, m))
			}
		}()
	}
}

func (p *Program) Send(msg Msg) {
	select {
	case <-p.ctx.Done():
	case p.msgs <- msg:
	}
}

func (p *Program) Kill() {
	p.shutdown(true)
}

func (p *Program) Wait() {
	<-p.finished
}

func (p *Program) SetDirty()   { p.root.SetDirty() }
func (p *Program) Dirty() bool { return p.root.Dirty() }
func (p *Program) Width() int  { return int(p.width.Load()) }
func (p *Program) Height() int { return int(p.height.Load()) }

func (p *Program) Theme() *Theme          { return p.theme }
func (p *Program) Renderer() *Renderer    { return p.renderer }
func (p *Program) EventBus() *EventBus    { return p.eventBus }
func (p *Program) DataStore() *DataStore  { return p.dataStore }
func (p *Program) Kernel() *kernel.Kernel { return p.kern }
func (p *Program) Model() Model           { return p.root }

var ignoreSignals uint32
