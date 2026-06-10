package mofu

import "sync"

// EventHandler is a function that handles an event.
type EventHandler func(msg Msg)

// EventBus allows components to subscribe to and publish messages.
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[string][]EventHandler
}

// NewEventBus creates a new event bus.
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]EventHandler),
	}
}

// Subscribe registers a handler for the given event type.
func (eb *EventBus) Subscribe(event string, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.subscribers[event] = append(eb.subscribers[event], handler)
}

// Unsubscribe removes all handlers for the given event type.
func (eb *EventBus) Unsubscribe(event string) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	delete(eb.subscribers, event)
}

// Publish sends a message to all subscribers of the given event type.
func (eb *EventBus) Publish(event string, msg Msg) {
	eb.mu.RLock()
	handlers := eb.subscribers[event]
	eb.mu.RUnlock()
	for _, handler := range handlers {
		if handler != nil {
			handler(msg)
		}
	}
}

// Common event types
const (
	EventResize   = "resize"
	EventKeyPress = "keypress"
	EventMouse    = "mouse"
	EventFocus    = "focus"
	EventBlur     = "blur"
	EventQuit     = "quit"
	EventCustom   = "custom"
)
