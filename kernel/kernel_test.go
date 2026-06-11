package kernel

import (
	"testing"
	"time"

	"github.com/xanstomper/mofu/state"
)

func TestLayoutCacheCheckUpdate(t *testing.T) {
	lc := NewLayoutCache()
	if !lc.Check(80, 24, 1) {
		t.Fatal("fresh cache should require layout")
	}
	lc.Update(80, 24, 1)
	if lc.Check(80, 24, 1) {
		t.Fatal("unchanged inputs should skip layout")
	}
	if !lc.Check(100, 24, 1) {
		t.Fatal("width change should require layout")
	}
	if !lc.Check(80, 24, 2) {
		t.Fatal("state hash change should require layout")
	}
	lc.Invalidate()
	if !lc.Check(80, 24, 1) {
		t.Fatal("Invalidate should force layout")
	}
}

func TestHashStateDiffers(t *testing.T) {
	g := state.NewGraph()
	a := state.NewAtom("hello")
	g.Add(a)
	h1 := HashState(g)
	a.SetValue("world")
	h2 := HashState(g)
	if h1 == h2 {
		t.Fatal("hash unchanged after string value change")
	}
}

func TestKernelLifecycle(t *testing.T) {
	k := New(DefaultConfig())
	k.Init()

	rendered := make(chan struct{}, 1)
	k.OnRender(func(dt time.Duration) {
		select {
		case rendered <- struct{}{}:
		default:
		}
	})

	a := state.NewAtom(0)
	k.State.Add(a)

	done := make(chan struct{})
	go func() {
		k.Run()
		close(done)
	}()

	a.SetValue(1)
	k.requestRender()

	select {
	case <-rendered:
	case <-time.After(2 * time.Second):
		t.Fatal("render callback never fired")
	}

	k.Stop()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("kernel did not stop")
	}

	if k.Running() {
		t.Fatal("Running() true after Stop")
	}
	if k.FrameCount() == 0 {
		t.Fatal("no frames counted")
	}
}

func TestKernelStateChangeCallback(t *testing.T) {
	k := New(DefaultConfig())
	k.Init()

	changed := make(chan any, 4)
	k.OnStateChange(func(id state.NodeID, oldVal, newVal any) {
		select {
		case changed <- newVal:
		default:
		}
	})

	a := state.NewAtom(0)
	k.State.Add(a)

	go k.Run()
	defer k.Stop()

	a.SetValue(42)
	k.requestRender()

	select {
	case v := <-changed:
		if v != 42 {
			t.Fatalf("state change got %v, want 42", v)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("state change callback never fired")
	}
}

// ---------------------------------------------------------------------------
// Benchmarks (C10)
// ---------------------------------------------------------------------------

func BenchmarkLayoutCacheCheck(b *testing.B) {
	lc := NewLayoutCache()
	lc.Update(120, 40, 12345)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = lc.Check(120, 40, 12345)
	}
}

func BenchmarkHashState100(b *testing.B) {
	g := state.NewGraph()
	for i := 0; i < 100; i++ {
		g.Add(state.NewAtom("value"))
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = HashState(g)
	}
}

func BenchmarkKernelTick(b *testing.B) {
	k := New(DefaultConfig())
	a := state.NewAtom(0)
	k.State.Add(a)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.SetValue(i)
		k.tick()
	}
}
