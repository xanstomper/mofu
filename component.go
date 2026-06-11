package mofu

import "time"

// Msg is any message sent to a Model's HandleEvent.
type Msg any

// Cmd is an IO operation returning a Msg when complete.
type Cmd func() Msg

// NoCmd is a no-op.
var NoCmd Cmd = nil

// Model is the interface that MOFU programs implement.
// It is compatible with the legacy Node interface via type alias.
type Model = Node

// BatchMsg runs commands concurrently.
type BatchMsg []Cmd

// SequenceMsg runs commands in order.
type SequenceMsg []Cmd

// QuitMsg signals exit.
type QuitMsg struct{}

// Quit sends a quit message.
func Quit() Msg { return QuitMsg{} }

// InterruptMsg signals SIGINT.
type InterruptMsg struct{}

// Interrupt sends an interrupt message.
func Interrupt() Msg { return InterruptMsg{} }

// SuspendMsg signals suspend (ctrl+z).
type SuspendMsg struct{}

// ResumeMsg signals resume.
type ResumeMsg struct{}

// WindowSizeMsg carries terminal dimensions.
type WindowSizeMsg struct{ Width, Height int }

// RawMsg contains a raw ANSI escape sequence.
type RawMsg struct{ Content any }

// Raw writes a raw ANSI sequence.
func Raw(content any) Msg { return RawMsg{content} }

// ClearScreenMsg requests a full screen redraw.
type ClearScreenMsg struct{}

// ClearScreen sends a clear message.
func ClearScreen() Msg { return ClearScreenMsg{} }

// ColorProfileMsg carries the detected terminal color profile.
type ColorProfileMsg struct{ Profile string }

// EnvMsg carries environment variables.
type EnvMsg map[string]string

// Batch runs commands concurrently.
func Batch(cmds ...Cmd) Cmd {
	return func() Msg { return BatchMsg(cmds) }
}

// Sequence runs commands in order.
func Sequence(cmds ...Cmd) Cmd {
	return func() Msg { return SequenceMsg(cmds) }
}

// Tick produces a message after a fixed duration.
func Tick(delay time.Duration, fn func() Msg) Cmd {
	t := time.NewTimer(delay)
	return func() Msg {
		<-t.C
		t.Stop()
		for len(t.C) > 0 {
			<-t.C
		}
		if fn != nil {
			return fn()
		}
		return nil
	}
}

// Every produces a message synchronized with the system clock.
func Every(duration time.Duration, fn func(time.Time) Msg) Cmd {
	n := time.Now()
	d := n.Truncate(duration).Add(duration).Sub(n)
	t := time.NewTimer(d)
	return func() Msg {
		ts := <-t.C
		t.Stop()
		for len(t.C) > 0 {
			<-t.C
		}
		if fn != nil {
			return fn(ts)
		}
		return nil
	}
}

// KickstartMsg is an internal tick used by the renderer.
type KickstartMsg struct{}
