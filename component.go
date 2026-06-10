package mofu

// Cmd is a function that returns a Msg when executed.
type Cmd func() Msg

// Msg is an arbitrary message passed through the update loop.
type Msg any

// Component defines the core interface for all Mofu widgets.
type Component interface {
	// Render returns the visual representation of the component.
	Render() string
	// HandleEvent processes a message and returns an optional command.
	HandleEvent(msg Msg) Cmd
	// Mount is called when the component is added to the tree.
	Mount() Cmd
	// Unmount is called when the component is removed from the tree.
	Unmount()
}

// Batch combines multiple commands into one that runs them all.
func Batch(cmds ...Cmd) Cmd {
	return func() Msg {
		for _, cmd := range cmds {
			if cmd != nil {
				cmd()
			}
		}
		return nil
	}
}

// Sequence runs commands one after another, feeding each result to the next.
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

// Tick creates a command that sends a message after a delay.
func Tick(delay int, fn func() Msg) Cmd {
	return func() Msg {
		// In a real implementation, this would use a timer.
		// For now, call immediately.
		if fn != nil {
			return fn()
		}
		return nil
	}
}

// NoCmd is a nil command (no operation).
var NoCmd Cmd = nil
