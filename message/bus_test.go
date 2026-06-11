package message

import (
	"testing"
)

func TestBusPublishChannel(t *testing.T) {
	b := NewBus(8)
	b.Publish(NewInput([]byte("a")))
	select {
	case msg := <-b.Channel():
		if msg.Type != TypeInput || msg.Source != "stdin" {
			t.Fatalf("wrong message: %+v", msg)
		}
	default:
		t.Fatal("message not buffered")
	}
}

func TestBusPublishFullDrops(t *testing.T) {
	b := NewBus(1)
	b.Publish(NewMessage(TypeCustom, 1))
	b.Publish(NewMessage(TypeCustom, 2))
	msg := <-b.Channel()
	if msg.Payload != 1 {
		t.Fatalf("got %v, want first message", msg.Payload)
	}
	select {
	case extra := <-b.Channel():
		t.Fatalf("overflow message not dropped: %+v", extra)
	default:
	}
}

func TestBusDispatch(t *testing.T) {
	b := NewBus(8)
	var got any
	b.Subscribe(TypeCommand, func(msg Message) { got = msg.Payload })
	b.Dispatch(NewCommand("save", "args"))
	if got != "args" {
		t.Fatalf("handler got %v, want args", got)
	}
}

func TestBusDispatchNoHandler(t *testing.T) {
	b := NewBus(8)
	b.Dispatch(NewMessage(TypeTimer, nil))
}

func TestSubscribeAny(t *testing.T) {
	b := NewBus(8)
	count := 0
	b.SubscribeAny(func(msg Message) { count++ })
	for _, ty := range AllTypes() {
		b.Dispatch(NewMessage(ty, nil))
	}
	if count != len(AllTypes()) {
		t.Fatalf("SubscribeAny handled %d, want %d", count, len(AllTypes()))
	}
}

func TestNewMessageDefaults(t *testing.T) {
	m := NewMessage(TypeSystem, "p")
	if m.Priority != PriorityNormal || m.Timestamp.IsZero() || m.Payload != "p" {
		t.Fatalf("NewMessage defaults wrong: %+v", m)
	}
}

// ---------------------------------------------------------------------------
// Benchmarks (C10)
// ---------------------------------------------------------------------------

func BenchmarkBusDispatch(b *testing.B) {
	bus := NewBus(64)
	bus.Subscribe(TypeInput, func(msg Message) {})
	msg := NewInput([]byte("x"))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Dispatch(msg)
	}
}

func BenchmarkBusPublish(b *testing.B) {
	bus := NewBus(1)
	msg := NewMessage(TypeCustom, nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Publish(msg)
	}
}
