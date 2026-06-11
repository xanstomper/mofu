package message

import (
	"sync"
	"time"
)

type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

type Type string

const (
	TypeInput   Type = "input"
	TypeCommand Type = "command"
	TypeSystem  Type = "system"
	TypePlugin  Type = "plugin"
	TypeStream  Type = "stream"
	TypeTimer   Type = "timer"
	TypeResize  Type = "resize"
	TypeCustom  Type = "custom"
)

type Message struct {
	ID        uint64
	Type      Type
	Priority  Priority
	Payload   any
	Source    string
	Timestamp time.Time
}

type Handler func(msg Message)

type Bus struct {
	input      chan Message
	handlers   map[Type][]Handler
	handlersMu sync.RWMutex
	done       chan struct{}
}

func NewBus(buffer int) *Bus {
	if buffer <= 0 {
		buffer = 64
	}
	return &Bus{
		input:    make(chan Message, buffer),
		handlers: make(map[Type][]Handler),
		done:     make(chan struct{}),
	}
}

func (b *Bus) Publish(msg Message) {
	select {
	case b.input <- msg:
	default:
	}
}

func (b *Bus) Subscribe(t Type, handler Handler) {
	b.handlersMu.Lock()
	b.handlers[t] = append(b.handlers[t], handler)
	b.handlersMu.Unlock()
}

func (b *Bus) SubscribeAny(handler Handler) {
	b.handlersMu.Lock()
	for _, t := range AllTypes() {
		b.handlers[t] = append(b.handlers[t], handler)
	}
	b.handlersMu.Unlock()
}

func (b *Bus) Dispatch(msg Message) {
	b.handlersMu.RLock()
	handlers := b.handlers[msg.Type]
	b.handlersMu.RUnlock()
	for _, h := range handlers {
		if h != nil {
			h(msg)
		}
	}
}

func (b *Bus) Channel() <-chan Message {
	return b.input
}

func (b *Bus) Stop() {
	close(b.done)
}

func AllTypes() []Type {
	return []Type{
		TypeInput, TypeCommand, TypeSystem, TypePlugin,
		TypeStream, TypeTimer, TypeResize, TypeCustom,
	}
}

func NewMessage(t Type, payload any) Message {
	return Message{
		Type:      t,
		Priority:  PriorityNormal,
		Payload:   payload,
		Timestamp: time.Now(),
	}
}

func NewInput(data []byte) Message {
	return Message{
		Type:      TypeInput,
		Priority:  PriorityNormal,
		Payload:   data,
		Timestamp: time.Now(),
		Source:    "stdin",
	}
}

func NewCommand(name string, args any) Message {
	return Message{
		Type:      TypeCommand,
		Priority:  PriorityNormal,
		Payload:   args,
		Timestamp: time.Now(),
		Source:    name,
	}
}
