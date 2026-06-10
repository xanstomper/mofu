package effect

import (
	"context"
	"sync"
	"time"
)

type System struct {
	mu       sync.RWMutex
	handlers map[Type]Handler
	queue    chan Effect
	results  chan Result
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

func NewSystem(buffer int) *System {
	if buffer <= 0 {
		buffer = 32
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &System{
		handlers: make(map[Type]Handler),
		queue:    make(chan Effect, buffer),
		results:  make(chan Result, buffer),
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (s *System) Register(t Type, handler Handler) {
	s.mu.Lock()
	s.handlers[t] = handler
	s.mu.Unlock()
}

func (s *System) Dispatch(eff Effect) {
	select {
	case s.queue <- eff:
	default:
	}
}

func (s *System) Start() {
	s.wg.Add(1)
	go s.processLoop()
}

func (s *System) Stop() {
	s.cancel()
	s.wg.Wait()
}

func (s *System) Results() <-chan Result {
	return s.results
}

func (s *System) processLoop() {
	defer s.wg.Done()

	idleTimer := time.NewTimer(10 * time.Millisecond)
	defer idleTimer.Stop()

	for {
		if !idleTimer.Stop() {
			select {
			case <-idleTimer.C:
			default:
			}
		}
		idleTimer.Reset(10 * time.Millisecond)

		select {
		case <-s.ctx.Done():
			return
		case eff := <-s.queue:
			s.mu.RLock()
			handler, ok := s.handlers[eff.Type]
			s.mu.RUnlock()

			if !ok {
				continue
			}

			result := handler(eff)
			select {
			case s.results <- result:
			default:
			}

		case <-idleTimer.C:
			// idle tick — no work to process
		}
	}
}

func (s *System) RegisterDefaults() {
	s.Register(TypeTimer, func(eff Effect) Result {
		timerEff, ok := eff.Payload.(TimerEffect)
		if !ok {
			return Result{Success: false, Error: nil}
		}
		time.Sleep(timerEff.Duration)
		if timerEff.Callback != nil {
			newEff := timerEff.Callback()
			s.Dispatch(newEff)
		}
		return Result{Success: true}
	})

	s.Register(TypeNoop, func(eff Effect) Result {
		return Result{Success: true}
	})
}

func After(d time.Duration, fn func() Effect) Effect {
	return Effect{
		Type: TypeTimer,
		Payload: TimerEffect{
			Duration: d,
			Callback: fn,
		},
	}
}
