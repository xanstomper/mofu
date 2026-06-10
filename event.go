package mofu

import (
	"sync"
	"time"
)

type EventType int

const (
	EventKeyPress EventType = iota
	EventMouse
	EventResize
	EventData
	EventAnimation
	EventSystem
	EventCustom
)

type Event struct {
	Type   EventType
	Data   Msg
	Time   time.Time
	Source string
}

type KeyEvent struct {
	Runes            []byte
	Key              Key
	Alt, Ctrl, Shift bool
}

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
