package mofu

import "time"

type Msg any

type Cmd func() Msg

var NoCmd Cmd = nil

func Batch(cmds ...Cmd) Cmd {
	return func() Msg {
		var last Msg
		for _, cmd := range cmds {
			if cmd != nil {
				last = cmd()
			}
		}
		return last
	}
}

func Sequence(cmds ...Cmd) Cmd {
	return func() Msg {
		var msg Msg
		for _, cmd := range cmds {
			if cmd != nil {
				msg = cmd()
			}
		}
		return msg
	}
}

func Tick(delay time.Duration, fn func() Msg) Cmd {
	return func() Msg {
		time.Sleep(delay)
		if fn != nil {
			return fn()
		}
		return nil
	}
}
