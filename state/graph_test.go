package state

import (
	"sync"
	"testing"
)

func TestAtomSetGet(t *testing.T) {
	a := NewAtom(42)
	if a.Value() != 42 {
		t.Fatalf("initial value = %v, want 42", a.Value())
	}
	a.SetValue(7)
	if a.Value() != 7 {
		t.Fatalf("after SetValue = %v, want 7", a.Value())
	}
	if !a.IsDirty() {
		t.Fatal("SetValue did not mark dirty")
	}
	a.MarkClean()
	if a.IsDirty() {
		t.Fatal("MarkClean did not clear dirty")
	}
}

func TestComputedRecomputes(t *testing.T) {
	g := NewGraph()
	count := NewAtom(5)
	g.Add(count)
	doubled := NewComputed([]StateNode{count}, func(deps []any) any {
		return deps[0].(int) * 2
	})
	g.Add(doubled)

	if doubled.Value() != 10 {
		t.Fatalf("initial computed = %v, want 10", doubled.Value())
	}

	count.SetValue(21)
	g.Propagate(count.ID())
	if doubled.Value() != 42 {
		t.Fatalf("after propagate = %v, want 42", doubled.Value())
	}
}

func TestComputedChain(t *testing.T) {
	g := NewGraph()
	a := NewAtom(1)
	g.Add(a)
	b := NewComputed([]StateNode{a}, func(d []any) any { return d[0].(int) + 1 })
	g.Add(b)
	c := NewComputed([]StateNode{b}, func(d []any) any { return d[0].(int) * 10 })
	g.Add(c)

	if c.Value() != 20 {
		t.Fatalf("initial chain = %v, want 20", c.Value())
	}
	a.SetValue(4)
	g.Propagate(a.ID())
	if c.Value() != 50 {
		t.Fatalf("chain after propagate = %v, want 50", c.Value())
	}
}

func TestCollectDirty(t *testing.T) {
	g := NewGraph()
	a := NewAtom(1)
	b := NewAtom(2)
	g.Add(a)
	g.Add(b)
	a.MarkClean()
	b.MarkClean()

	a.SetValue(9)
	dirty := g.CollectDirty()
	if len(dirty) != 1 || dirty[0].ID() != a.ID() {
		t.Fatalf("CollectDirty = %d nodes, want only atom a", len(dirty))
	}
}

func TestStreamPush(t *testing.T) {
	s := NewStream("stdin")
	s.Push("line1")
	if s.Value() != "line1" {
		t.Fatalf("stream value = %v, want line1", s.Value())
	}
}

func TestOnChangeFires(t *testing.T) {
	a := NewAtom(0)
	var got any
	a.OnChange(func(ev ChangeEvent) { got = ev.Value })
	a.SetValue(99)
	if got != 99 {
		t.Fatalf("OnChange got %v, want 99", got)
	}
}

func TestSnapshot(t *testing.T) {
	g := NewGraph()
	a := NewAtom("x")
	g.Add(a)
	snap := g.Snapshot()
	if snap[a.ID()] != "x" {
		t.Fatalf("snapshot missing atom value: %v", snap)
	}
}

func TestConcurrentSetValue(t *testing.T) {
	a := NewAtom(0)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			a.SetValue(n)
		}(i)
	}
	wg.Wait()
	if _, ok := a.Value().(int); !ok {
		t.Fatal("concurrent SetValue corrupted value")
	}
}

func TestPropagateCycleSafe(t *testing.T) {
	g := NewGraph()
	a := NewAtom(1)
	g.Add(a)
	a.AddDependent(a.ID())
	g.Propagate(a.ID())
}

// ---------------------------------------------------------------------------
// Transaction tests
// ---------------------------------------------------------------------------

func TestTransactionBatchPropagate(t *testing.T) {
	g := NewGraph()
	a := NewAtom(0)
	b := NewAtom(0)
	g.Add(a)
	g.Add(b)

	// Track how many times the graph propagates
	propagateCount := 0
	c := NewComputed([]StateNode{a, b}, func(d []any) any {
		propagateCount++
		return d[0].(int) + d[1].(int)
	})
	g.Add(c)

	// Without transaction: each SetValue triggers propagation separately
	a.SetValue(10)
	g.Propagate(a.ID())
	b.SetValue(20)
	g.Propagate(b.ID())
	if c.Value() != 30 {
		t.Fatalf("without txn: computed = %v, want 30", c.Value())
	}

	// With transaction: batch both changes, propagate once
	propagateCount = 0
	txn := g.BeginTransaction()
	a.SetValue(100)
	b.SetValue(200)
	txn.Commit()

	if c.Value() != 300 {
		t.Fatalf("with txn: computed = %v, want 300", c.Value())
	}
}

func TestTransactionNested(t *testing.T) {
	g := NewGraph()
	a := NewAtom(0)
	g.Add(a)

	outer := g.BeginTransaction()
	a.SetValue(10)

	inner := g.BeginTransaction()
	a.SetValue(20)
	inner.Commit() // Should NOT propagate (nested)

	// Value should be 20 but no propagation yet
	outer.Commit() // Outer commit triggers propagation
}

func TestTransactionRollback(t *testing.T) {
	g := NewGraph()
	a := NewAtom(0)
	g.Add(a)

	txn := g.BeginTransaction()
	a.SetValue(42)
	txn.Rollback()

	// Value is already mutated (rollback just prevents propagation)
	// Use SaveSnapshot/RestoreSnapshot for true rollback
	if a.Value() != 42 {
		t.Fatalf("value should still be 42 after rollback (mutation already happened)")
	}
}

// ---------------------------------------------------------------------------
// Snapshot/Restore tests
// ---------------------------------------------------------------------------

func TestSaveRestoreSnapshot(t *testing.T) {
	g := NewGraph()
	a := NewAtom(10)
	b := NewAtom(20)
	g.Add(a)
	g.Add(b)

	g.SaveSnapshot()
	a.SetValue(99)
	b.SetValue(88)

	if a.Value() != 99 {
		t.Fatalf("before restore: a = %v, want 99", a.Value())
	}

	ok := g.RestoreSnapshot()
	if !ok {
		t.Fatal("RestoreSnapshot returned false")
	}

	if a.Value() != 10 {
		t.Fatalf("after restore: a = %v, want 10", a.Value())
	}
	if b.Value() != 20 {
		t.Fatalf("after restore: b = %v, want 20", b.Value())
	}
}

func TestSnapshotStack(t *testing.T) {
	g := NewGraph()
	a := NewAtom(1)
	g.Add(a)

	g.SaveSnapshot() // depth 1
	a.SetValue(2)
	g.SaveSnapshot() // depth 2
	a.SetValue(3)

	if g.SnapshotDepth() != 2 {
		t.Fatalf("depth = %d, want 2", g.SnapshotDepth())
	}

	g.RestoreSnapshot() // restore to 2
	if a.Value() != 2 {
		t.Fatalf("first restore: a = %v, want 2", a.Value())
	}

	g.RestoreSnapshot() // restore to 1
	if a.Value() != 1 {
		t.Fatalf("second restore: a = %v, want 1", a.Value())
	}

	if g.RestoreSnapshot() {
		t.Fatal("should fail with empty snapshot stack")
	}
}

// ---------------------------------------------------------------------------
// PropagateAll tests
// ---------------------------------------------------------------------------

func TestPropagateAll(t *testing.T) {
	g := NewGraph()
	a := NewAtom(1)
	b := NewAtom(2)
	g.Add(a)
	g.Add(b)

	sum := NewComputed([]StateNode{a, b}, func(d []any) any {
		return d[0].(int) + d[1].(int)
	})
	g.Add(sum)

	a.MarkClean()
	b.MarkClean()

	a.SetValue(10)
	b.SetValue(20)

	g.PropagateAll()

	if sum.Value() != 30 {
		t.Fatalf("PropagateAll: sum = %v, want 30", sum.Value())
	}
}

// ---------------------------------------------------------------------------
// MemoizedSelector tests
// ---------------------------------------------------------------------------

func TestMemoizedSelector(t *testing.T) {
	g := NewGraph()
	a := NewAtom(5)
	g.Add(a)

	ms := NewMemoizedSelector([]StateNode{a}, func(d []any) any {
		return d[0].(int) * 2
	})
	g.Add(ms)

	if ms.Value() != 10 {
		t.Fatalf("initial selector = %v, want 10", ms.Value())
	}

	// Change to same effective value (shallow equal)
	a.SetValue(5)
	g.Propagate(a.ID())
	if ms.Changed() {
		t.Fatal("selector should not report changed for same value")
	}

	// Change to different value
	a.SetValue(7)
	g.Propagate(a.ID())
	if !ms.Changed() {
		t.Fatal("selector should report changed for new value")
	}
	if ms.Value() != 14 {
		t.Fatalf("selector = %v, want 14", ms.Value())
	}
}

// ---------------------------------------------------------------------------
// TopologicalSort tests
// ---------------------------------------------------------------------------

func TestTopologicalSort(t *testing.T) {
	g := NewGraph()
	a := NewAtom(1)
	g.Add(a)
	b := NewComputed([]StateNode{a}, func(d []any) any { return d[0].(int) + 1 })
	g.Add(b)
	c := NewComputed([]StateNode{b}, func(d []any) any { return d[0].(int) * 10 })
	g.Add(c)

	order := g.TopologicalSort()
	if len(order) != 3 {
		t.Fatalf("topo sort = %d nodes, want 3", len(order))
	}

	// a must come before b, b must come before c
	aIdx, bIdx, cIdx := -1, -1, -1
	for i, id := range order {
		switch id {
		case a.ID():
			aIdx = i
		case b.ID():
			bIdx = i
		case c.ID():
			cIdx = i
		}
	}
	if aIdx >= bIdx || bIdx >= cIdx {
		t.Fatalf("topological order wrong: a@%d b@%d c@%d", aIdx, bIdx, cIdx)
	}
}

// ---------------------------------------------------------------------------
// Dependency introspection tests
// ---------------------------------------------------------------------------

func TestDependentsOf(t *testing.T) {
	g := NewGraph()
	a := NewAtom(1)
	g.Add(a)
	b := NewComputed([]StateNode{a}, func(d []any) any { return d[0].(int) })
	g.Add(b)

	deps := g.DependentsOf(a.ID())
	if len(deps) != 1 || deps[0] != b.ID() {
		t.Fatalf("DependentsOf(a) = %v, want [%d]", deps, b.ID())
	}
}

func TestDependenciesOf(t *testing.T) {
	g := NewGraph()
	a := NewAtom(1)
	g.Add(a)
	b := NewComputed([]StateNode{a}, func(d []any) any { return d[0].(int) })
	g.Add(b)

	deps := g.DependenciesOf(b.ID())
	if len(deps) != 1 || deps[0] != a.ID() {
		t.Fatalf("DependenciesOf(b) = %v, want [%d]", deps, a.ID())
	}
}

func TestGraphString(t *testing.T) {
	g := NewGraph()
	a := NewAtom(1)
	g.Add(a)
	b := NewComputed([]StateNode{a}, func(d []any) any { return d[0].(int) })
	g.Add(b)

	s := g.String()
	if s == "" {
		t.Fatal("String() returned empty")
	}
}

// ---------------------------------------------------------------------------
// Benchmarks (C10)
// ---------------------------------------------------------------------------

func BenchmarkAtomSetValue(b *testing.B) {
	a := NewAtom(0)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.SetValue(i)
	}
}

func BenchmarkPropagateChain10(b *testing.B) {
	g := NewGraph()
	root := NewAtom(0)
	g.Add(root)
	var prev StateNode = root
	for i := 0; i < 10; i++ {
		c := NewComputed([]StateNode{prev}, func(d []any) any {
			if v, ok := d[0].(int); ok {
				return v + 1
			}
			return 0
		})
		g.Add(c)
		prev = c
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		root.SetValue(i)
		g.Propagate(root.ID())
	}
}

func BenchmarkCollectDirty100(b *testing.B) {
	g := NewGraph()
	atoms := make([]*Atom, 100)
	for i := range atoms {
		atoms[i] = NewAtom(i)
		g.Add(atoms[i])
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		atoms[i%100].SetValue(i)
		_ = g.CollectDirty()
	}
}

func BenchmarkCollectDirty1000(b *testing.B) {
	g := NewGraph()
	atoms := make([]*Atom, 1000)
	for i := range atoms {
		atoms[i] = NewAtom(i)
		g.Add(atoms[i])
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		atoms[i%1000].SetValue(i)
		_ = g.CollectDirty()
	}
}

func BenchmarkCollectDirtyNoDirty(b *testing.B) {
	g := NewGraph()
	for i := 0; i < 1000; i++ {
		g.Add(NewAtom(i))
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = g.CollectDirty()
	}
}

func BenchmarkTransactionBatch(b *testing.B) {
	g := NewGraph()
	atoms := make([]*Atom, 100)
	for i := range atoms {
		atoms[i] = NewAtom(i)
		g.Add(atoms[i])
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		txn := g.BeginTransaction()
		for j := range atoms {
			atoms[j].SetValue(i*100 + j)
		}
		txn.Commit()
	}
}

func BenchmarkPropagateAll100(b *testing.B) {
	g := NewGraph()
	atoms := make([]*Atom, 100)
	for i := range atoms {
		atoms[i] = NewAtom(i)
		g.Add(atoms[i])
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := range atoms {
			atoms[j].SetValue(i*100 + j)
		}
		g.PropagateAll()
	}
}
