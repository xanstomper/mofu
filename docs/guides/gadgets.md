# Gadgets Guide

Gadgets are MOFU's reactive UI systems. Unlike traditional widgets, Gadgets are runtime-aware, data-driven systems that compose intelligently.

## What Makes Gadgets Different

| Feature | Traditional Widgets | MOFU Gadgets |
|---------|-------------------|--------------|
| State | Manual updates | Reactive graph |
| Layout | Manual positioning | Constraint-based |
| Rendering | Full redraw | Incremental diff |
| Animation | Bolt-on | Built-in hooks |
| Data | Polling | Streaming |

## Using Gadgets

```go
import "github.com/xanstomper/mofu/gadgets"

// Create a log stream
logStream := gadgets.NewLogStream("logs")
logStream.Append("Server started")
logStream.Append("Request processed")

// Create a metric board
metrics := gadgets.NewMetricBoard("metrics")
metrics.Set("cpu", 23.5)
metrics.Set("memory", 4.2)

// Create a command palette
palette := gadgets.NewCommandPalette("palette")
palette.AddCommand(gadgets.Command{
    Name:     "Save",
    Shortcut: "Ctrl+S",
    Action:   func() mofu.Cmd { return nil },
})
```

## Gadget Categories

### Data & Table Systems (10)

| Gadget | Use Case |
|--------|----------|
| LiveTable | Streaming data display |
| DiffTable | Change highlighting |
| HeatTable | Density visualization |
| PagedTable | Large dataset pagination |
| TreeTable | Hierarchical data |
| StreamingGrid | Real-time grid |
| FilterTable | Search/filter |
| SortTable | Multi-key sorting |
| PivotTableLite | Aggregation |
| SparseTable | 10k+ rows |

### Navigation & Layout (10)

| Gadget | Use Case |
|--------|----------|
| SmartSidebar | Auto-collapsing nav |
| AdaptiveSplit | Layout balancing |
| WorkspaceGrid | Multi-panel |
| InspectorPane | Detail view |
| FocusNavigator | Keyboard nav |
| CommandDock | Action bar |
| ContextOverlay | Floating UI |
| DockingSystem | Draggable panels |
| ViewportManager | Virtual scrolling |
| ResponsiveLayoutCore | Adaptive layouts |

### Input & Interaction (10)

| Gadget | Use Case |
|--------|----------|
| SmartForm | Form building |
| InlineEditor | Text editing |
| KeyChordRouter | Shortcuts |
| MultiCursorInput | Multiple inputs |
| AutoCompleteEngine | Suggestions |
| ValidatedInputField | Validation |
| CommandPalette | Command search |
| InputStreamRouter | Event routing |
| GestureInputLayer | Mouse handling |
| FocusTrapManager | Input boundaries |

### Real-Time Data (10)

| Gadget | Use Case |
|--------|----------|
| LogStream | Log viewing |
| MetricBoard | Metrics display |
| EventFeed | Event timeline |
| ProcessTreeView | Process monitoring |
| NetworkMonitor | Network visualization |
| FileWatcherView | File system |
| StreamConsole | CLI output |
| TraceViewer | Execution tracing |
| PipelineVisualizer | Data flow |
| StateInspector | Debugging |

### Visual & ASCII (10)

| Gadget | Use Case |
|--------|----------|
| ASCIIScene | Scene rendering |
| ParticleField | Particle effects |
| SplashComposer | Boot animations |
| WaveVisualizer | Waveforms |
| DensityMapRenderer | Heat maps |
| ProceduralArtEngine | Generative art |
| MotionBanner | Animated headers |
| GlyphMorpher | Character morphing |
| TerminalCanvas | Pixel drawing |
| SDFRendererLite | Distance fields |

## Creating Custom Gadgets

```go
type MyGadget struct {
    gadgets.Base
    value int
}

func (g *MyGadget) Render(state gadgets.StateView) []gadgets.RenderNode {
    return []gadgets.RenderNode{
        {Type: "text", Content: fmt.Sprintf("Value: %d", g.value)},
    }
}

func (g *MyGadget) OnEvent(e gadgets.Event) {
    if e.Type == "update" {
        g.value++
    }
}
```

## Layout Contracts

Gadgets declare constraints, not positions:

```go
func (g *MyGadget) MinSize() (int, int) { return 10, 5 }
func (g *MyGadget) MaxSize() (int, int) { return 100, 50 }
func (g *MyGadget) Flex() float64       { return 1.0 }
func (g *MyGadget) Priority() int       { return 10 }
```

## Animation Hooks

Gadgets declare motion, the runtime executes:

```go
func (g *MyGadget) OnStateChange(delta gadgets.StateDelta) gadgets.AnimationSpec {
    return gadgets.AnimationSpec{
        Type:       "fade",
        DurationMs: 300,
        Easing:     "ease-out",
    }
}
```
