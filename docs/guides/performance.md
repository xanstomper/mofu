# Performance Guide

MOFU is designed for high-performance terminal applications.

## Performance Targets

| Metric | Target | How |
|--------|--------|-----|
| Input latency | <1ms | Fast path bypass |
| Frame rate | 60fps | Scheduler lanes |
| Flicker | Zero | CSI 2026 |
| Rendering | O(Δ) | Cell-level diff |
| GC pressure | Near-zero | Preallocated buffers |
| Layout | Cached | Hash fingerprinting |

## Key Optimizations

### 1. O(1) Dirty Tracking

```go
// When state changes, only affected nodes are marked dirty
atom.SetValue(42)  // O(1) - only marks this node dirty

// CollectDirty returns only dirty nodes
dirty := graph.CollectDirty()  // O(dirty nodes), not O(all nodes)
```

### 2. Incremental Diff Rendering

```go
// Only changed cells are written to terminal
renderer.Flush()  // Computes diff, emits minimal ANSI sequences
```

### 3. SGR Cache

```go
// Style→ANSI compilation is cached
style.SGR()  // First call: compile. Subsequent: cache hit.
```

### 4. Preallocated Buffers

```go
// Frame buffers are preallocated
renderer := NewRenderer(w, h, theme)  // No per-frame allocations
```

### 5. Scheduler Lanes

```go
// Critical operations are never blocked
scheduler.SubmitRealtime(fn)  // Highest priority
scheduler.SubmitBackground(fn)  // Lowest priority
```

## Profiling

MOFU uses zero-allocation hot paths. Benchmark your app:

```go
go test -bench=. -benchmem ./...
```

Key metrics to watch:
- Ring buffer write: ~90ns, 0 allocs
- State graph get/set: ~50ns, 0 allocs
- Scene buffer set: ~20ns, 0 allocs

## Memory Optimization

### Use Preallocated Buffers

```go
// ❌ Bad: Allocates every frame
func render() {
    buf := make([]byte, 1024)
    // ...
}

// ✅ Good: Reuse buffer
var buf [1024]byte
func render() {
    // Use buf
}
```

### Use Object Pooling

```go
// ❌ Bad: Creates new objects
func process() {
    node := new(Node)
    // ...
}

// ✅ Good: Pool objects
var nodePool = sync.Pool{
    New: func() any { return new(Node) },
}

func process() {
    node := nodePool.Get().(*Node)
    defer nodePool.Put(node)
    // ...
}
```

## Benchmarking

```go
func BenchmarkStateUpdate(b *testing.B) {
    graph := state.NewGraph()
    atom := state.NewAtom(0)
    graph.Add(atom)

    b.ReportAllocs()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        atom.SetValue(i)
    }
}
```

## Common Performance Mistakes

### 1. Don't Re-render Everything

```go
// ❌ Bad: Full re-render
func (a *App) Render(ctx *mofu.RenderContext) {
    for _, child := range a.children {
        child.Render(ctx)  // Renders all children
    }
}

// ✅ Good: Only render dirty
func (a *App) Render(ctx *mofu.RenderContext) {
    for _, child := range a.children {
        if child.Dirty() {
            child.Render(ctx)
        }
    }
}
```

### 2. Don't Allocate in Hot Paths

```go
// ❌ Bad: Allocates every call
func process(data []byte) string {
    return string(data)
}

// ✅ Good: Reuse buffer
var buf strings.Builder
func process(data []byte) string {
    buf.Reset()
    buf.Write(data)
    return buf.String()
}
```

### 3. Don't Block the Render Lane

```go
// ❌ Bad: Blocks rendering
func (a *App) HandleEvent(event mofu.Event) mofu.Cmd {
    time.Sleep(time.Second)  // Blocks!
    return nil
}

// ✅ Good: Use background lane
func (a *App) HandleEvent(event mofu.Event) mofu.Cmd {
    return func() mofu.Msg {
        time.Sleep(time.Second)  // Runs in background
        return nil
    }
}
```
