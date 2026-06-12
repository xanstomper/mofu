package mofu

import (
	"time"
)

type TimerMsg struct {
	Time    time.Time
	Payload any
}

type TickMsg struct {
	Time time.Time
}

type KeyStringMsg struct {
	Str string
}

func Timer(d time.Duration, fn func() Msg) Cmd {
	t := time.NewTimer(d)
	return func() Msg {
		<-t.C
		t.Stop()
		for len(t.C) > 0 {
			<-t.C
		}
		if fn != nil {
			return fn()
		}
		return TimerMsg{Time: time.Now()}
	}
}

func EveryTick(d time.Duration, fn func(time.Time) Cmd) Cmd {
	return func() Msg {
		t := time.NewTicker(d)
		defer t.Stop()
		ts := <-t.C
		if fn != nil {
			cmd := fn(ts)
			if cmd != nil {
				return cmd()
			}
		}
		return TickMsg{Time: ts}
	}
}

func After(d time.Duration, msg Msg) Cmd {
	return func() Msg {
		time.Sleep(d)
		return msg
	}
}

func WithDelay(d time.Duration, cmd Cmd) Cmd {
	return func() Msg {
		time.Sleep(d)
		if cmd != nil {
			return cmd()
		}
		return nil
	}
}
