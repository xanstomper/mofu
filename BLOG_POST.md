# Introducing MOFU: A Reactive Terminal UI Runtime for Go

Today I'm open-sourcing MOFU — a new approach to building terminal UIs in Go.

## The Problem

Building terminal UIs in Go is harder than it should be. Existing frameworks like Bubble Tea use a full-model-copy approach where every state change triggers a complete re-render. This works for simple apps but breaks down at scale:

- 1000 widgets → full model copy on every keystroke
- No incremental rendering → unnecessary redraws
- Manual dirty tracking → error-prone
- No built-in animation → bolt-on afterthoughts

## The Solution

MOFU takes a different approach. Instead of copying the entire model on every change, MOFU uses a **reactive state graph** with O(1) dirty tracking:

```
Input → Event Router → State Graph → Dirty Propagation → Layout → Diff Render → Terminal
```

Only changed components re-render. Only changed cells are written to the terminal. The result is **zero unnecessary work**.

## Architecture

MOFU is built on three pillars:

### 1. Reactive State Graph

```go
type Atom struct { ... }      // Primitive state
type Computed struct { ... }  // Derived values
type Stream struct { ... }    // External input
```

When an `Atom` changes, all dependent `Computed` nodes automatically recompute. Only the affected subtree is marked dirty.

### 2. Incremental Diff Renderer

MOFU maintains a double-buffered framebuffer. Each frame, it computes the minimal set of changed cells and emits only those ANSI sequences. No full-screen redraws.

### 3. Spring Physics Animations

```go
spring := mofu.NewSpring(0)
spring.SetTarget(100)
// Smooth interpolation with configurable stiffness/damping
```

No CSS-like transitions. Real physics-based motion.

## API

MOFU's API is as simple as Bubble Tea:

```go
type myModel struct {
    mofu.Minimal
    count int
}

func (m *myModel) Render(ctx *mofu.RenderContext) {
    ctx.Renderer.WriteString(fmt.Sprintf("Count: %d", m.count), 0, 0, ...)
}

func (m *myModel) HandleEvent(event mofu.Event) mofu.Cmd {
    // handle events
    return nil
}

func main() {
    mofu.Run(&myModel{})
}
```

Two methods. One embed. That's it.

## Widgets

MOFU includes 15 widgets:

- Text, Input, List, Button, ProgressBar, Tabs
- Checkbox, Select, Modal, Table, Toast, Tooltip
- Tree, Menu, Container

## Performance

| Metric | MOFU | Bubble Tea |
|--------|------|------------|
| State update | 185ns | ~1μs |
| Dirty tracking (1000 nodes) | 106μs | O(N) full scan |
| Input parse | <100ns | <100ns |

## Examples

9 examples included:

- Counter (hello world)
- Dashboard (multi-panel)
- Chat (real-time messaging)
- File Manager (directory navigation)
- Form (inputs, checkbox, button)
- Settings (configuration panel)
- Log Viewer (filtering, scrolling)
- Wizard (multi-step setup)
- Monitor (real-time system stats)

## What's Next

- More widgets (Tree, DatePicker, ColorPicker)
- More examples (IDE, chat client, database browser)
- Plugin system
- GoDoc on pkg.go.dev
- Community

## Try It

```bash
go get github.com/xanstomper/mofu
cd examples/counter && go run main.go
```

## License

MIT

---

*MOFU is a reactive terminal UI runtime for Go. It's not a widget library, not a styling package, not a Bubble Tea clone. It's a fundamentally better architecture for building terminal applications.*
