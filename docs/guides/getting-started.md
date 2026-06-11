# Getting Started with MOFU

## Installation

```bash
go get github.com/xanstomper/mofu
```

## Your First App

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
    style := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))
    text := fmt.Sprintf("Count: %d\n\nPress j/k to change, q to quit", a.count)
    ctx.Renderer.WriteStyledString(text, r.X, r.Y, style)
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

## Key Concepts

### 1. Minimal Embed

```go
type MyApp struct {
    mofu.Minimal  // Provides default Node implementations
    // your state here
}
```

`Minimal` gives you default implementations for all Node methods except `Render` and `HandleEvent`.

### 2. Render Method

```go
func (a *MyApp) Render(ctx *mofu.RenderContext) {
    r := ctx.Bounds  // Available space
    // Draw to ctx.Renderer
}
```

### 3. HandleEvent Method

```go
func (a *MyApp) HandleEvent(event mofu.Event) mofu.Cmd {
    // Handle keyboard, mouse, system events
    // Return mofu.QuitCmd() to exit
    return nil
}
```

### 4. Running

```go
mofu.Run(&MyApp{})  // Simple
mofu.RunWithOpts(&MyApp{}, mofu.WithTheme(mofu.MochiTheme()))  // With options
```

## Next Steps

- [Gadgets Guide](./gadgets.md) - Learn about the 50 reactive UI systems
- [Styling Guide](./styling.md) - Semantic styling with Cuddles
- [Forms Guide](./forms.md) - Schema-driven forms with Meow
- [Examples](../../examples/) - See MOFU in action
