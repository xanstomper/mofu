# MOFU: The Complete Guide

A comprehensive tutorial from zero to production-ready terminal applications.

## Table of Contents

1. [Installation](#installation)
2. [First App](#first-app)
3. [Understanding the Architecture](#understanding-the-architecture)
4. [Working with Gadgets](#working-with-gadgets)
5. [Styling with Cuddles](#styling-with-cuddles)
6. [Building Forms with Meow](#building-forms-with-meow)
7. [Handling Events](#handling-events)
8. [Layout System](#layout-system)
9. [Animation](#animation)
10. [Real-Time Data](#real-time-data)
11. [Advanced Patterns](#advanced-patterns)
12. [Production Deployment](#production-deployment)

---

## Installation

```bash
# Install MOFU
go get github.com/xanstomper/mofu

# Verify installation
go list -m github.com/xanstomper/mofu
```

---

## First App

Let's build a simple counter to understand MOFU fundamentals.

### Step 1: Create Project

```bash
mkdir myapp && cd myapp
go mod init myapp
go get github.com/xanstomper/mofu
```

### Step 2: Write the App

Create `main.go`:

```go
package main

import (
    "fmt"
    "os"
    "github.com/xanstomper/mofu"
)

// App is our application state
type App struct {
    mofu.Minimal  // Embed for default implementations
    count int     // Our state
}

// Render draws the UI
func (a *App) Render(ctx *mofu.RenderContext) {
    r := ctx.Bounds  // Available space

    // Title
    titleStyle := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
    ctx.Renderer.WriteString(" Counter App", r.X, r.Y, titleStyle.Foreground, titleStyle.Background, titleStyle.Attrs)

    // Separator
    sep := strings.Repeat("─", r.Width-2)
    ctx.Renderer.WriteString(sep, r.X+1, r.Y+1, mofu.Hex("444444"), mofu.ColorBlack, 0)

    // Counter display
    counterStyle := mofu.DefaultStyle().Fg(mofu.Hex("a6e3a1")).WithAttrs(mofu.AttrBold)
    text := fmt.Sprintf("Count: %d", a.count)
    ctx.Renderer.WriteString(text, r.X+2, r.Y+3, counterStyle.Foreground, counterStyle.Background, counterStyle.Attrs)

    // Instructions
    helpStyle := mofu.DefaultStyle().Fg(mofu.Hex("666666"))
    ctx.Renderer.WriteString("j/k: Change count  q: Quit", r.X+2, r.Y+5, helpStyle.Foreground, helpStyle.Background, helpStyle.Attrs)
}

// HandleEvent processes keyboard input
func (a *App) HandleEvent(event mofu.Event) mofu.Cmd {
    if event.Type != mofu.EventKeyPress {
        return nil
    }

    ke := event.Data.(mofu.KeyEvent)

    switch {
    case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
        return mofu.QuitCmd()
    case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
        a.count++
    case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
        a.count--
    }

    return nil
}

func main() {
    if err := mofu.Run(&App{}); err != nil {
        fmt.Fprintf(os.Stderr, "error: %v\n", err)
        os.Exit(1)
    }
}
```

### Step 3: Run

```bash
go run main.go
```

### What Just Happened?

1. **`mofu.Minimal`** — Provides default implementations for all Node methods
2. **`Render()`** — Called every frame to draw the UI
3. **`HandleEvent()`** — Called for every keyboard/mouse event
4. **`mofu.Run()`** — Starts the application with sensible defaults

---

## Understanding the Architecture

MOFU uses a reactive state graph with O(1) dirty tracking:

```
Input → Event Router → State Graph → Dirty Propagation → Layout → Diff Render → Terminal
```

### Key Concepts

| Concept | Description |
|---------|-------------|
| **Node** | Any UI component (your app is a Node) |
| **Render** | Draw the UI within bounds |
| **HandleEvent** | Process input events |
| **Cmd** | Side effect that returns a Msg |
| **Msg** | Message that updates state |

### The Event Flow

1. User presses a key
2. MOFU parses the input
3. `HandleEvent()` is called with the event
4. Return a `Cmd` to trigger side effects
5. `Render()` is called to update the display

---

## Working with Gadgets

Gadgets are MOFU's reactive UI systems. They're not just widgets — they're runtime-aware, data-driven systems.

### Using Gadgets

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

### Available Gadgets

**Data & Table Systems (10)**
- LiveTable, DiffTable, HeatTable, PagedTable, TreeTable
- StreamingGrid, FilterTable, SortTable, PivotTableLite, SparseTable

**Navigation & Layout (10)**
- SmartSidebar, AdaptiveSplit, WorkspaceGrid, InspectorPane, FocusNavigator
- CommandDock, ContextOverlay, DockingSystem, ViewportManager, ResponsiveLayoutCore

**Input & Interaction (10)**
- SmartForm, InlineEditor, KeyChordRouter, MultiCursorInput, AutoCompleteEngine
- ValidatedInputField, CommandPalette, InputStreamRouter, GestureInputLayer, FocusTrapManager

**Real-Time Data (10)**
- LogStream, MetricBoard, EventFeed, ProcessTreeView, NetworkMonitor
- FileWatcherView, StreamConsole, TraceViewer, PipelineVisualizer, StateInspector

**Visual & ASCII (10)**
- ASCIIScene, ParticleField, SplashComposer, WaveVisualizer, DensityMapRenderer
- ProceduralArtEngine, MotionBanner, GlyphMorpher, TerminalCanvas, SDFRendererLite

---

## Styling with Cuddles

Cuddles is MOFU's semantic styling engine. Instead of specifying colors directly, you specify meaning.

### Basic Usage

```go
import "github.com/xanstomper/mofu/cuddles"

// Get current theme
theme := cuddles.Mochi()

// Use semantic tokens
primaryStyle := theme.Style(cuddles.Primary)
errorStyle := theme.Style(cuddles.Error)
successStyle := theme.Style(cuddles.Success)
```

### Why Semantic Styling?

```go
// ❌ Bad: Visual styling
style := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))

// ✅ Good: Semantic styling
style := theme.Style(cuddles.Primary)
```

When you change themes, all "primary" elements update automatically.

### Theme Switching

```go
manager := cuddles.NewManager(cuddles.Mochi())
manager.Register(cuddles.Catppuccin())
manager.Apply("catppuccin")
```

### Semantic Tokens

| Token | Use Case |
|-------|----------|
| Primary | Main brand color |
| Secondary | Supporting color |
| Accent | Highlight color |
| Success | Positive feedback |
| Warning | Caution |
| Error | Negative feedback |
| Info | Informational |
| Muted | De-emphasized |
| Text | Primary text |
| TextDim | Secondary text |
| Background | App background |
| Surface | Card/panel background |
| Border | Borders and dividers |

---

## Building Forms with Meow

Meow is MOFU's schema-driven form system. Define forms declaratively, get validation for free.

### Basic Form

```go
import "github.com/xanstomper/mofu/meow"

form := meow.NewForm(
    meow.Input("name", "Name").SetRequired(),
    meow.Input("email", "Email").Validate(meow.ValidateEmail),
    meow.Select("role", "Role", []string{"Admin", "User"}),
    meow.Checkbox("agree", "I agree to terms"),
)

form.OnSubmit(func(values map[string]any) mofu.Cmd {
    fmt.Printf("Name: %s\n", values["name"])
    fmt.Printf("Email: %s\n", values["email"])
    return nil
})
```

### Validators

```go
meow.ValidateEmail           // Email format
meow.ValidateMinLength(3)    // Minimum length
meow.ValidateMaxLength(100)  // Maximum length
meow.ValidateMinValue(0)     // Number minimum
meow.ValidateMaxValue(100)   // Number maximum
meow.ValidateRange(0, 100)   // Number range
meow.ValidateURL             // URL format
meow.ValidatePhone           // Phone format
meow.ValidateOneOf("a","b")  // Enum validation
```

### Conditional Fields

```go
meow.Input("company", "Company").When(func(values map[string]any) bool {
    role, _ := values["role"].(string)
    return role == "business"
})
```

---

## Handling Events

### Keyboard Events

```go
func (a *App) HandleEvent(event mofu.Event) mofu.Cmd {
    if event.Type != mofu.EventKeyPress {
        return nil
    }

    ke := event.Data.(mofu.KeyEvent)

    // Arrow keys
    switch ke.Key {
    case mofu.KeyUp:    // Arrow up
    case mofu.KeyDown:  // Arrow down
    case mofu.KeyLeft:  // Arrow left
    case mofu.KeyRight: // Arrow right
    }

    // Function keys
    switch ke.Key {
    case mofu.KeyF1:   // F1
    case mofu.KeyF12:  // F12
    }

    // Ctrl+key
    if ke.Ctrl {
        switch ke.Key {
        case mofu.KeyCtrlC: // Ctrl+C
        case mofu.KeyCtrlZ: // Ctrl+Z
        }
    }

    // Regular characters
    for _, r := range ke.Runes {
        switch r {
        case 'q': // q key
        case 'j': // j key
        }
    }

    // Modifier keys
    if ke.Alt { /* Alt held */ }
    if ke.Shift { /* Shift held */ }

    return nil
}
```

### Mouse Events

```go
func (a *App) HandleEvent(event mofu.Event) mofu.Cmd {
    if event.Type == mofu.EventMouse {
        me := event.Data.(mofu.MouseEvent)

        switch me.Button {
        case mofu.MouseLeft:
            switch me.Action {
            case mofu.MousePress:
                fmt.Printf("Click at (%d, %d)\n", me.X, me.Y)
            case mofu.MouseRelease:
                fmt.Printf("Release at (%d, %d)\n", me.X, me.Y)
            }
        case mofu.MouseWheelUp:
            // Scroll up
        case mofu.MouseWheelDown:
            // Scroll down
        }
    }
    return nil
}
```

---

## Layout System

MOFU uses constraint-based layout:

```go
// Set bounds for a child
child.SetBounds(mofu.Rect{
    X:      10,
    Y:      5,
    Width:  40,
    Height: 20,
})

// Check bounds
r := child.Bounds()
fmt.Printf("Position: (%d, %d), Size: %dx%d\n", r.X, r.Y, r.Width, r.Height)
```

### Layout Patterns

```go
// Flex layout
row := mofu.NewRow(child1, child2, child3)
row.Style().Gap = 2
row.Style().Direction = mofu.DirectionRow

// Grid layout
grid := mofu.NewSimpleGrid(3, child1, child2, child3)

// Stack layout
stack := mofu.NewColumn(child1, child2, child3)
```

---

## Animation

MOFU has built-in spring physics:

```go
// Create a spring
spring := mofu.NewSpring(0)
spring.SetTarget(100)
spring.Stiffness = 170
spring.Damping = 26

// Update in your render loop
spring.Advance(deltaMs)
value := spring.Current

// Use presets
spring = mofu.NewSpringFromPreset(0, mofu.SpringWobbly)
```

### Easing Functions

```go
mofu.EaseLinear      // Linear
mofu.EaseInQuad      // Quadratic ease-in
mofu.EaseOutQuad     // Quadratic ease-out
mofu.EaseInOutCubic  // Cubic ease-in-out
mofu.EaseOutElastic  // Elastic ease-out
mofu.EaseOutBounce   // Bounce ease-out
```

---

## Real-Time Data

MOFU has first-class streaming support:

```go
import "github.com/xanstomper/mofu/gadgets"

// Create log stream
logStream := gadgets.NewLogStream("logs")

// Append data (from any goroutine)
go func() {
    for line := range dataChannel {
        logStream.Append(line)
    }
}()

// Create metric board
metrics := gadgets.NewMetricBoard("metrics")

// Update metrics
go func() {
    for metric := range metricChannel {
        metrics.Set(metric.Name, metric.Value)
    }
}()
```

---

## Advanced Patterns

### State Management

```go
type App struct {
    mofu.Minimal
    items    []string
    selected int
    filter   string
}

func (a *App) filteredItems() []string {
    if a.filter == "" {
        return a.items
    }
    var filtered []string
    for _, item := range a.items {
        if strings.Contains(strings.ToLower(item), strings.ToLower(a.filter)) {
            filtered = append(filtered, item)
        }
    }
    return filtered
}
```

### Component Composition

```go
type Dashboard struct {
    mofu.Minimal
    sidebar *Sidebar
    content *Content
    inspector *Inspector
}

func (d *Dashboard) Render(ctx *mofu.RenderContext) {
    r := ctx.Bounds

    // Sidebar (fixed width)
    d.sidebar.SetBounds(mofu.Rect{X: r.X, Y: r.Y, Width: 20, Height: r.Height})
    d.sidebar.Render(ctx)

    // Content (flexible)
    d.content.SetBounds(mofu.Rect{X: r.X + 21, Y: r.Y, Width: r.Width - 51, Height: r.Height})
    d.content.Render(ctx)

    // Inspector (fixed width)
    d.inspector.SetBounds(mofu.Rect{X: r.X + r.Width - 30, Y: r.Y, Width: 30, Height: r.Height})
    d.inspector.Render(ctx)
}
```

---

## Production Deployment

### Building

```bash
# Build for current platform
go build -o myapp .

# Cross-compile
GOOS=linux GOARCH=amd64 go build -o myapp-linux .
GOOS=darwin GOARCH=arm64 go build -o myapp-mac .
GOOS=windows GOARCH=amd64 go build -o myapp.exe .
```

### Distribution

```bash
# Create release archive
tar -czf myapp-1.0.0-linux-amd64.tar.gz myapp-linux
tar -czf myapp-1.0.0-darwin-arm64.tar.gz myapp-mac
zip myapp-1.0.0-windows-amd64.zip myapp.exe
```

### Configuration

```go
// Use environment variables
func getConfig() Config {
    return Config{
        Debug: os.Getenv("DEBUG") == "true",
        Theme: os.Getenv("THEME"),
    }
}
```

---

## Next Steps

- [Gadgets Guide](../guides/gadgets.md) - Learn about the 50 reactive UI systems
- [Styling Guide](../guides/styling.md) - Semantic styling with Cuddles
- [Forms Guide](../guides/forms.md) - Schema-driven forms with Meow
- [API Reference](../api/README.md) - Complete API documentation
- [Examples](../../examples/) - See MOFU in action

---

## Conclusion

MOFU provides:

- **Reactive state graph** — O(1) dirty tracking
- **50 Gadgets** — Runtime-aware UI systems
- **Semantic styling** — Theme-aware, accessible
- **Schema forms** — Declarative, validated
- **Spring physics** — Smooth animations
- **First-class streaming** — Real-time data support
- **Production ready** — Build, test, deploy

Build beautiful terminal applications with minimal code.

```go
mofu.Run(&MyApp{})
```

That's it. That's MOFU.
