package mofu

import (
	"sync"
	"time"
)

// EventType identifies the kind of event.
type EventType int

const (
	// EventKeyPress is a keyboard event.
	EventKeyPress EventType = iota
	// EventMouse is a mouse event.
	EventMouse
	// EventResize is a terminal resize event.
	EventResize
	// EventData is a data event.
	EventData
	// EventAnimation is an animation tick event.
	EventAnimation
	// EventSystem is a system event.
	EventSystem
	// EventCustom is a custom event.
	EventCustom
)

// Event is a typed event with data and timestamp.
type Event struct {
	Type   EventType
	Data   Msg
	Time   time.Time
	Source string
}

// KeyEvent carries keyboard event data.
type KeyEvent struct {
	Runes            []byte
	Key              Key
	Alt, Ctrl, Shift bool
}

// Key is a keyboard key identifier.
type Key int

const (
	KeyNone Key = iota
	KeyUp
	KeyDown
	KeyRight
	KeyLeft
	KeyEnter
	KeyEsc
	KeyTab
	KeySpace
	KeyBack
	KeyHome
	KeyEnd
	KeyPgUp
	KeyPgDn
	KeyInsert
	KeyDelete
	KeyF1
	KeyF2
	KeyF3
	KeyF4
	KeyF5
	KeyF6
	KeyF7
	KeyF8
	KeyF9
	KeyF10
	KeyF11
	KeyF12
	KeyShiftTab

	// Ctrl+key combinations
	KeyCtrlAt            // Ctrl+Space / Ctrl+@
	KeyCtrlA             // Ctrl+A
	KeyCtrlB             // Ctrl+B
	KeyCtrlC             // Ctrl+C
	KeyCtrlD             // Ctrl+D
	KeyCtrlE             // Ctrl+E
	KeyCtrlF             // Ctrl+F
	KeyCtrlG             // Ctrl+G
	KeyCtrlJ             // Ctrl+J
	KeyCtrlK             // Ctrl+K
	KeyCtrlL             // Ctrl+L
	KeyCtrlN             // Ctrl+N
	KeyCtrlO             // Ctrl+O
	KeyCtrlP             // Ctrl+P
	KeyCtrlQ             // Ctrl+Q
	KeyCtrlR             // Ctrl+R
	KeyCtrlS             // Ctrl+S
	KeyCtrlT             // Ctrl+T
	KeyCtrlU             // Ctrl+U
	KeyCtrlV             // Ctrl+V
	KeyCtrlW             // Ctrl+W
	KeyCtrlX             // Ctrl+X
	KeyCtrlY             // Ctrl+Y
	KeyCtrlZ             // Ctrl+Z
	KeyCtrlBackslash     // Ctrl+\
	KeyCtrlCloseBracket  // Ctrl+]
	KeyCtrlCaret         // Ctrl+^
	KeyCtrlUnderscore    // Ctrl+_
)

type MouseEvent struct {
	X, Y   int
	Button MouseButton
	Action MouseAction
}

type MouseButton int

const (
	MouseLeft MouseButton = iota
	MouseRight
	MouseMiddle
	MouseWheelUp
	MouseWheelDown
	MouseNone
)

type MouseAction int

const (
	MousePress MouseAction = iota
	MouseRelease
	MouseDrag
	MouseMove
)

type EventHandler func(Event)

type EventBus struct {
	mu          sync.RWMutex
	subscribers map[string][]EventHandler
}

func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]EventHandler),
	}
}

func (eb *EventBus) Subscribe(event string, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.subscribers[event] = append(eb.subscribers[event], handler)
}

func (eb *EventBus) Unsubscribe(event string) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	delete(eb.subscribers, event)
}

func (eb *EventBus) Publish(event Event) {
	eb.mu.RLock()
	handlers := eb.subscribers[eventTypeName(event.Type)]
	eb.mu.RUnlock()
	for _, handler := range handlers {
		if handler != nil {
			handler(event)
		}
	}
}

func eventTypeName(t EventType) string {
	switch t {
	case EventKeyPress:
		return "keypress"
	case EventMouse:
		return "mouse"
	case EventResize:
		return "resize"
	case EventData:
		return "data"
	case EventAnimation:
		return "animation"
	case EventSystem:
		return "system"
	default:
		return "custom"
	}
}
