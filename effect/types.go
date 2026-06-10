package effect

import "time"

type Type string

const (
	TypeNoop       Type = "noop"
	TypeTimer      Type = "timer"
	TypeIO         Type = "io"
	TypePlugin     Type = "plugin"
	TypeTask       Type = "task"
	TypeHTTP       Type = "http"
	TypeSubprocess Type = "subprocess"
	TypeFileWatch  Type = "filewatch"
	TypeLog        Type = "log"
)

type Effect struct {
	Type    Type
	Payload any
	Meta    map[string]any
}

type TimerEffect struct {
	Duration time.Duration
	Callback func() Effect
}

type IOEffect struct {
	Data   []byte
	Target string
}

type PluginEffect struct {
	PluginID string
	Action   string
	Payload  any
}

type TaskEffect struct {
	Name     string
	Priority int
	Work     func() (any, error)
}

type Result struct {
	Success bool
	Value   any
	Error   error
	Effects []Effect
}

type Handler func(effect Effect) Result
