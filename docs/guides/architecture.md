# MOFU Architecture Guide

## Core Philosophy

MOFU is NOT:
- A widget library
- A terminal toolkit
- A styling package
- A message loop framework

MOFU IS:
- A streaming-first reactive runtime
- A terminal-native application runtime
- A reactive visual layer for live data

## Execution Model

```
Input Streams → Router → State Graph → Compute → Render Diff → Terminal
```

### Key Principles

1. **No global Update() loop** — Everything is event-stream driven
2. **Reactive state graph** — O(1) dirty tracking, not O(N) re-renders
3. **Incremental rendering** — Only changed cells are written to terminal
4. **Structured concurrency** — No raw goroutines, scheduler lanes
5. **Declarative animation** — Gadgets declare motion, runtime executes

## State Graph

```
Atom (primitive state)
  ↓
Computed (derived values)
  ↓
Effect (side effects)
  ↓
UI (render projection)
```

When an Atom changes:
1. All dependent Computed nodes recompute
2. Only affected UI nodes are marked dirty
3. Only dirty nodes are re-rendered
4. Only changed cells are written to terminal

Result: **O(changed nodes), not O(app tree)**

## Scheduler Lanes

| Lane | Purpose | Priority |
|------|---------|----------|
| Realtime | Input, focus, UI interaction | Highest |
| Stream | AI tokens, logs, events | High |
| Compute | Derived state updates | Medium |
| Render | Diff + terminal output | Medium |
| Background | Caching, indexing, cleanup | Lowest |

**Rule:** No lane may block another.

## Rendering Pipeline

```
Previous Frame
    ↓
Current Frame (from state graph)
    ↓
Diff Engine (cell-level comparison)
    ↓
Dirty Cell Map
    ↓
ANSI Patch Builder (batched, cached)
    ↓
Terminal Flush (single write)
```

**Optimizations:**
- Cell-level diffing
- Row merging
- ANSI batching
- Dirty region tracking
- SGR cache (zero-allocation style lookups)
- Synchronized Output (CSI 2026)

## Gadget System

Gadgets are NOT widgets. They are:

1. **Runtime-aware** — Connected to scheduler lanes
2. **Data-driven** — Subscribe to state graph
3. **Stream-compatible** — Consume live data
4. **Layout-constrained** — Declare constraints, not positions
5. **Animation-capable** — Declare motion, not imperative

### Gadget Lifecycle

```
Init → Bind → Render → OnEvent → OnTick → Dispose
```

### Layout Contracts

Gadgets declare constraints:
- `MinSize()` — Minimum dimensions
- `MaxSize()` — Maximum dimensions
- `Flex()` — Layout weight
- `Priority()` — Layout importance
- `OverflowBehavior()` — How to handle overflow

### Animation Hooks

Gadgets declare motion intentions:
- `OnEnter()` — When entering viewport
- `OnExit()` — When leaving viewport
- `OnStateChange()` — When state changes
- `OnLayoutChange()` — When layout changes

Runtime executes animations safely.

## Data Flow

```
User Input
    ↓
Event Router
    ↓
State Graph Update
    ↓
Dirty Propagation
    ↓
UI Recomputation
    ↓
Diff Render
    ↓
Terminal Output
```

## Why This Beats Traditional TUI Frameworks

| Aspect | Traditional | MOFU |
|--------|-------------|------|
| State updates | Full model copy | Partial graph update |
| Rendering | Full redraw | Incremental diff |
| Concurrency | Manual goroutines | Structured lanes |
| Layout | Manual coordinates | Declarative constraints |
| Animation | Bolt-on | Built-in hooks |
| Streaming | Not native | First-class |
| Debugging | Hard | Built-in inspectors |

## Memory Model

- **Arena allocation** per frame
- **Structural sharing** for state graphs
- **Incremental GC** windows
- **Zero-copy** streaming buffers
- **Preallocated** frame buffers

## Performance Targets

| Metric | Target | How |
|--------|--------|-----|
| Input latency | <1ms | Fast path bypass |
| Frame rate | 60fps | Scheduler lanes |
| Flicker | Zero | CSI 2026 |
| Rendering | O(Δ) | Cell-level diff |
| GC pressure | Near-zero | Preallocated buffers |
| Layout | Cached | Hash fingerprinting |
