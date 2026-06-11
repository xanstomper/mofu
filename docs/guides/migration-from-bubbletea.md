# Migrating from Bubble Tea to MOFU

This guide helps you migrate existing Bubble Tea applications to MOFU.

## Key Differences

| Concept | Bubble Tea | MOFU |
|---------|-----------|------|
| Architecture | Elm loop | Reactive graph |
| State | Full model copy | Partial graph update |
| Update | `Update(msg) (Model, Cmd)` | `HandleEvent(event) Cmd` |
| View | `View() string` | `Render(ctx *RenderContext)` |
| Commands | `tea.Cmd` | `mofu.Cmd` |
| Messages | `tea.Msg` | `mofu.Msg` |

## Migration Steps

### 1. Replace Model with Minimal

```go
// Bubble Tea
type Model struct {
    count int
}

// MOFU
type App struct {
    mofu.Minimal
    count int
}
```

### 2. Replace Update with HandleEvent

```go
// Bubble Tea
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg.(type) {
    case tea.KeyMsg:
        // handle key
    }
    return m, nil
}

// MOFU
func (a *App) HandleEvent(event mofu.Event) mofu.Cmd {
    if event.Type == mofu.EventKeyPress {
        ke := event.Data.(mofu.KeyEvent)
        // handle key
    }
    return nil
}
```

### 3. Replace View with Render

```go
// Bubble Tea
func (m Model) View() string {
    return fmt.Sprintf("Count: %d", m.count)
}

// MOFU
func (a *App) Render(ctx *mofu.RenderContext) {
    r := ctx.Bounds
    text := fmt.Sprintf("Count: %d", a.count)
    ctx.Renderer.WriteString(text, r.X, r.Y, mofu.ColorWhite, mofu.ColorBlack, 0)
}
```

### 4. Replace tea.Cmd with mofu.Cmd

```go
// Bubble Tea
func increment() tea.Msg {
    return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
}

// MOFU
func increment() mofu.Msg {
    return mofu.KeyEvent{Key: mofu.KeyDown}
}
```

### 5. Replace tea.Quit with mofu.QuitCmd

```go
// Bubble Tea
return m, tea.Quit

// MOFU
return mofu.QuitCmd()
```

### 6. Replace tea.Batch with mofu.Batch

```go
// Bubble Tea
return m, tea.Batch(cmd1, cmd2)

// MOFU
return mofu.Batch(cmd1, cmd2)
```

### 7. Replace tea.Program with mofu.Run

```go
// Bubble Tea
p := tea.NewProgram(model)
if _, err := p.Run(); err != nil {
    log.Fatal(err)
}

// MOFU
if err := mofu.Run(&App{}); err != nil {
    log.Fatal(err)
}
```

## Complete Migration Example

### Before (Bubble Tea)

```go
package main

import (
    "fmt"
    "github.com/charmbracelet/bubbletea"
)

type Model struct {
    count int
}

func (m Model) Init() tea.Cmd {
    return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg.(type) {
    case tea.KeyMsg:
        switch msg.(tea.KeyMsg).Type {
        case tea.KeyCtrlC, tea.KeyEsc:
            return m, tea.Quit
        case tea.KeyDown:
            m.count++
        case tea.KeyUp:
            m.count--
        }
    }
    return m, nil
}

func (m Model) View() string {
    return fmt.Sprintf("Count: %d\n\nPress j/k to change, q to quit", m.count)
}

func main() {
    p := tea.NewProgram(Model{})
    if _, err := p.Run(); err != nil {
        fmt.Fprintf(os.Stderr, "error: %v\n", err)
        os.Exit(1)
    }
}
```

### After (MOFU)

```go
package main

import (
    "fmt"
    "os"
    "github.com/xanstomper/mofu"
)

type App struct {
    mofu.Minimal
    count int
}

func (a *App) Render(ctx *mofu.RenderContext) {
    r := ctx.Bounds
    text := fmt.Sprintf("Count: %d\n\nPress j/k to change, q to quit", a.count)
    ctx.Renderer.WriteString(text, r.X, r.Y, mofu.ColorWhite, mofu.ColorBlack, 0)
}

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

## Benefits of Migration

1. **Better performance** — O(1) dirty tracking vs O(N) re-render
2. **Reactive state** — Automatic dependency tracking
3. **Built-in widgets** — 50 gadgets vs ~15 bubbles
4. **Semantic styling** — Theme-aware, not manual hex codes
5. **Schema forms** — Declarative, not manual construction
6. **Animation** — Spring physics built-in
7. **Streaming** — First-class support for live data
8. **Accessibility** — ARIA support built-in

## Common Pitfalls

1. **Don't copy state** — MOFU uses references, not copies
2. **Use semantic styling** — Don't hardcode colors
3. **Use gadgets** — Don't reinvent widgets
4. **Use meow for forms** — Don't build forms manually
5. **Use cuddles for styling** — Don't use raw hex codes
