package gadgets

import (
	"fmt"
	"testing"
)

// mockGadget is a test gadget implementation.
type mockGadget struct {
	Base
	initErr    error
	disposeErr bool
	tickCount  int
	disposed   bool
}

func newMockGadget(id string) *mockGadget {
	return &mockGadget{Base: *NewBase(id)}
}

func (m *mockGadget) Init(ctx GadgetContext) error {
	if m.initErr != nil {
		return m.initErr
	}
	m.Base.Init(ctx)
	return nil
}

func (m *mockGadget) OnTick(dt int64) {
	m.tickCount++
}

func (m *mockGadget) Dispose() error {
	m.disposed = true
	if m.disposeErr {
		return fmt.Errorf("dispose error")
	}
	return nil
}

func (m *mockGadget) Render(state StateView) []RenderNode {
	return []RenderNode{{Type: "text", Content: m.id}}
}

// ---------------------------------------------------------------------------
// GadgetState tests
// ---------------------------------------------------------------------------

func TestGadgetStateString(t *testing.T) {
	cases := []struct {
		state GadgetState
		want  string
	}{
		{GadgetDiscovered, "discovered"},
		{GadgetResolved, "resolved"},
		{GadgetMounted, "mounted"},
		{GadgetActive, "active"},
		{GadgetSuspended, "suspended"},
		{GadgetUnmounted, "unmounted"},
		{GadgetFailed, "failed"},
		{GadgetState(99), "unknown"},
	}
	for _, c := range cases {
		if got := c.state.String(); got != c.want {
			t.Errorf("GadgetState(%d).String() = %q, want %q", c.state, got, c.want)
		}
	}
}

func TestValidGadgetTransition(t *testing.T) {
	cases := []struct {
		from, to GadgetState
		valid    bool
	}{
		{GadgetDiscovered, GadgetResolved, true},
		{GadgetDiscovered, GadgetActive, false},
		{GadgetResolved, GadgetMounted, true},
		{GadgetMounted, GadgetActive, true},
		{GadgetMounted, GadgetSuspended, true},
		{GadgetActive, GadgetSuspended, true},
		{GadgetActive, GadgetUnmounted, true},
		{GadgetSuspended, GadgetActive, true},
		{GadgetSuspended, GadgetUnmounted, true},
		{GadgetUnmounted, GadgetActive, false},
		{GadgetFailed, GadgetDiscovered, true},
	}
	for _, c := range cases {
		got := ValidGadgetTransition(c.from, c.to)
		if got != c.valid {
			t.Errorf("Valid(%s→%s) = %v, want %v", c.from, c.to, got, c.valid)
		}
	}
}

// ---------------------------------------------------------------------------
// GadgetInstance tests
// ---------------------------------------------------------------------------

func TestGadgetInstanceLifecycle(t *testing.T) {
	g := newMockGadget("test")
	inst := NewGadgetInstance(g)

	if inst.State() != GadgetDiscovered {
		t.Fatalf("initial state = %s, want discovered", inst.State())
	}

	inst.transitionTo(GadgetResolved)
	if inst.State() != GadgetResolved {
		t.Fatalf("after resolve = %s, want resolved", inst.State())
	}

	history := inst.History()
	if len(history) != 1 {
		t.Fatalf("history len = %d, want 1", len(history))
	}
}

func TestGadgetInstanceInvalidTransition(t *testing.T) {
	g := newMockGadget("test")
	inst := NewGadgetInstance(g)

	ok := inst.transitionTo(GadgetActive)
	if ok {
		t.Fatal("discovered→active should be invalid")
	}
	if inst.State() != GadgetDiscovered {
		t.Fatalf("state should still be discovered")
	}
}

// ---------------------------------------------------------------------------
// GadgetManager tests
// ---------------------------------------------------------------------------

func TestGadgetManagerRegister(t *testing.T) {
	ctx := GadgetContext{Binder: NewBinder(), Logger: func(f string, a ...any) {}}
	gm := NewGadgetManager(ctx)

	g := newMockGadget("test")
	if err := gm.Register(g); err != nil {
		t.Fatalf("register: %v", err)
	}

	inst := gm.Get("test")
	if inst == nil {
		t.Fatal("Get returned nil")
	}
	if inst.State() != GadgetResolved {
		t.Fatalf("state = %s, want resolved", inst.State())
	}
}

func TestGadgetManagerDuplicateRegister(t *testing.T) {
	ctx := GadgetContext{Binder: NewBinder(), Logger: func(f string, a ...any) {}}
	gm := NewGadgetManager(ctx)

	gm.Register(newMockGadget("test"))
	err := gm.Register(newMockGadget("test"))
	if err == nil {
		t.Fatal("duplicate register should error")
	}
}

func TestGadgetManagerMount(t *testing.T) {
	ctx := GadgetContext{Binder: NewBinder(), Logger: func(f string, a ...any) {}}
	gm := NewGadgetManager(ctx)

	g := newMockGadget("test")
	gm.Register(g)
	gm.Mount("test")

	inst := gm.Get("test")
	if inst.State() != GadgetActive {
		t.Fatalf("state = %s, want active", inst.State())
	}
}

func TestGadgetManagerMountInitError(t *testing.T) {
	ctx := GadgetContext{Binder: NewBinder(), Logger: func(f string, a ...any) {}}
	gm := NewGadgetManager(ctx)

	g := newMockGadget("test")
	g.initErr = fmt.Errorf("init failed")
	gm.Register(g)
	err := gm.Mount("test")

	if err == nil {
		t.Fatal("mount with init error should fail")
	}
	inst := gm.Get("test")
	if inst.State() != GadgetFailed {
		t.Fatalf("state = %s, want failed", inst.State())
	}
	if inst.Error() != "init failed" {
		t.Fatalf("error = %q, want 'init failed'", inst.Error())
	}
}

func TestGadgetManagerSuspendResume(t *testing.T) {
	ctx := GadgetContext{Binder: NewBinder(), Logger: func(f string, a ...any) {}}
	gm := NewGadgetManager(ctx)

	gm.Register(newMockGadget("test"))
	gm.Mount("test")

	gm.Suspend("test")
	inst := gm.Get("test")
	if inst.State() != GadgetSuspended {
		t.Fatalf("state = %s, want suspended", inst.State())
	}

	gm.Resume("test")
	if inst.State() != GadgetActive {
		t.Fatalf("state = %s, want active", inst.State())
	}
}

func TestGadgetManagerUnmount(t *testing.T) {
	ctx := GadgetContext{Binder: NewBinder(), Logger: func(f string, a ...any) {}}
	gm := NewGadgetManager(ctx)

	g := newMockGadget("test")
	gm.Register(g)
	gm.Mount("test")
	gm.Unmount("test")

	inst := gm.Get("test")
	if inst.State() != GadgetUnmounted {
		t.Fatalf("state = %s, want unmounted", inst.State())
	}
	if !g.disposed {
		t.Fatal("Dispose should have been called")
	}
}

func TestGadgetManagerReload(t *testing.T) {
	ctx := GadgetContext{Binder: NewBinder(), Logger: func(f string, a ...any) {}}
	gm := NewGadgetManager(ctx)

	old := newMockGadget("test")
	gm.Register(old)
	gm.Mount("test")

	newG := newMockGadget("test")
	gm.Reload("test", newG)

	inst := gm.Get("test")
	if inst.State() != GadgetActive {
		t.Fatalf("state = %s, want active", inst.State())
	}
	if !old.disposed {
		t.Fatal("old gadget should be disposed")
	}
}

func TestGadgetManagerOnChange(t *testing.T) {
	ctx := GadgetContext{Binder: NewBinder(), Logger: func(f string, a ...any) {}}
	gm := NewGadgetManager(ctx)

	var changes []string
	gm.OnChange(func(id string, state GadgetState) {
		changes = append(changes, id+":"+state.String())
	})

	gm.Register(newMockGadget("test"))
	gm.Mount("test")

	if len(changes) < 2 {
		t.Fatalf("expected at least 2 changes, got %d: %v", len(changes), changes)
	}
}

func TestGadgetManagerTick(t *testing.T) {
	ctx := GadgetContext{Binder: NewBinder(), Logger: func(f string, a ...any) {}}
	gm := NewGadgetManager(ctx)

	g := newMockGadget("test")
	gm.Register(g)
	gm.Mount("test")

	gm.Tick(16)
	gm.Tick(16)

	if g.tickCount != 2 {
		t.Fatalf("tickCount = %d, want 2", g.tickCount)
	}
}

func TestGadgetManagerActive(t *testing.T) {
	ctx := GadgetContext{Binder: NewBinder(), Logger: func(f string, a ...any) {}}
	gm := NewGadgetManager(ctx)

	gm.Register(newMockGadget("a"))
	gm.Register(newMockGadget("b"))
	gm.Mount("a")
	gm.Mount("b")
	gm.Suspend("b")

	active := gm.Active()
	if len(active) != 1 || active[0].ID() != "a" {
		t.Fatalf("active = %v, want [a]", active)
	}
}

func TestGadgetManagerMountAll(t *testing.T) {
	ctx := GadgetContext{Binder: NewBinder(), Logger: func(f string, a ...any) {}}
	gm := NewGadgetManager(ctx)

	gm.Register(newMockGadget("a"))
	gm.Register(newMockGadget("b"))

	errs := gm.MountAll()
	if len(errs) != 0 {
		t.Fatalf("MountAll errors: %v", errs)
	}

	if len(gm.Active()) != 2 {
		t.Fatalf("active = %d, want 2", len(gm.Active()))
	}
}

func TestGadgetManagerUnmountAll(t *testing.T) {
	ctx := GadgetContext{Binder: NewBinder(), Logger: func(f string, a ...any) {}}
	gm := NewGadgetManager(ctx)

	gm.Register(newMockGadget("a"))
	gm.Register(newMockGadget("b"))
	gm.MountAll()
	gm.UnmountAll()

	if len(gm.Active()) != 0 {
		t.Fatalf("active after unmount all = %d, want 0", len(gm.Active()))
	}
}

// ---------------------------------------------------------------------------
// GadgetScope tests
// ---------------------------------------------------------------------------

func TestGadgetScope(t *testing.T) {
	scope := NewGadgetScope("test")

	scope.Set("key", "value")
	v, ok := scope.Get("key")
	if !ok || v != "value" {
		t.Fatalf("Get = %v, %v, want value, true", v, ok)
	}

	scope.Delete("key")
	_, ok = scope.Get("key")
	if ok {
		t.Fatal("deleted key should not exist")
	}
}

func TestGadgetScopeSnapshot(t *testing.T) {
	scope := NewGadgetScope("test")
	scope.Set("a", 1)
	scope.Set("b", 2)

	snap := scope.Snapshot()
	if len(snap) != 2 {
		t.Fatalf("snapshot len = %d, want 2", len(snap))
	}

	scope.Set("c", 3)
	scope.Restore(snap)

	_, ok := scope.Get("c")
	if ok {
		t.Fatal("c should not exist after restore")
	}
}

func TestGadgetScopePersistent(t *testing.T) {
	scope := NewGadgetScope("test")
	scope.SetPersistent("config", "value")

	v, ok := scope.GetPersistent("config")
	if !ok || v != "value" {
		t.Fatalf("GetPersistent = %v, %v", v, ok)
	}

	snap := scope.PersistentSnapshot()
	if snap["config"] != "value" {
		t.Fatalf("persistent snapshot wrong: %v", snap)
	}
}
