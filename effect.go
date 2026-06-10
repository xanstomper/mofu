package mofu

import (
	"context"
	"sync"
	"time"
)

type Effect interface {
	Execute() Msg
	Cancel()
	Done() <-chan struct{}
	ID() string
}

type effect struct {
	id      string
	execute func() Msg
	ctx     context.Context
	cancel  context.CancelFunc
	done    chan struct{}
	once    sync.Once
}

var effectID int64

func NewEffect(name string, fn func() Msg) Effect {
	ctx, cancel := context.WithCancel(context.Background())
	return &effect{
		id:      name,
		execute: fn,
		ctx:     ctx,
		cancel:  cancel,
		done:    make(chan struct{}),
	}
}

func NewTimerEffect(name string, delay time.Duration, fn func() Msg) Effect {
	ctx, cancel := context.WithCancel(context.Background())
	return &effect{
		id: name,
		execute: func() Msg {
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil
			case <-timer.C:
				if fn != nil {
					return fn()
				}
				return nil
			}
		},
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),
	}
}

func NewRetryEffect(name string, maxRetries int, fn func() (Msg, error)) Effect {
	ctx, cancel := context.WithCancel(context.Background())
	return &effect{
		id: name,
		execute: func() Msg {
			var lastMsg Msg
			for i := 0; i < maxRetries; i++ {
				select {
				case <-ctx.Done():
					return nil
				default:
				}
				msg, err := fn()
				if err == nil {
					return msg
				}
				lastMsg = msg
				time.Sleep(time.Duration(100*(i+1)) * time.Millisecond)
			}
			return lastMsg
		},
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),
	}
}

func (e *effect) Execute() Msg {
	defer e.once.Do(func() { close(e.done) })
	select {
	case <-e.ctx.Done():
		return nil
	default:
	}
	if e.execute != nil {
		return e.execute()
	}
	return nil
}

func (e *effect) Cancel() {
	e.cancel()
}

func (e *effect) Done() <-chan struct{} {
	return e.done
}

func (e *effect) ID() string {
	return e.id
}
