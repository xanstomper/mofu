# Testing Guide

MOFU provides tools for testing terminal applications.

## Unit Testing

```go
func TestCounter(t *testing.T) {
    app := &App{count: 0}

    // Simulate key press
    event := mofu.Event{
        Type: mofu.EventKeyPress,
        Data: mofu.KeyEvent{Key: mofu.KeyDown},
    }
    app.HandleEvent(event)

    if app.count != 1 {
        t.Errorf("count = %d, want 1", app.count)
    }
}
```

## Widget Testing

```go
func TestInput(t *testing.T) {
    input := widgets.NewInput()
    input.Focus()
    input.InsertRune('h')
    input.InsertRune('i')

    if input.Value != "hi" {
        t.Errorf("value = %q, want %q", input.Value, "hi")
    }
}
```

## Form Testing

```go
func TestFormValidation(t *testing.T) {
    form := meow.NewForm(
        meow.Input("email", "Email").Validate(meow.ValidateEmail),
    )

    form.SetValue("email", "invalid")
    if form.Valid() {
        t.Error("expected invalid email")
    }

    form.SetValue("email", "test@example.com")
    if !form.Valid() {
        t.Error("expected valid email")
    }
}
```

## Snapshot Testing

```go
func TestRenderSnapshot(t *testing.T) {
    app := &App{count: 42}

    // Create renderer
    renderer := mofu.NewRenderer(80, 24, mofu.DefaultTheme())

    // Render
    ctx := &mofu.RenderContext{
        Renderer: renderer,
        Bounds:   mofu.Rect{Width: 80, Height: 24},
    }
    app.Render(ctx)

    // Get output
    output := renderer.Flush()

    // Compare with expected
    expected := "Count: 42"
    if !strings.Contains(output, expected) {
        t.Errorf("output missing %q", expected)
    }
}
```

## Benchmark Testing

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

func BenchmarkRender(b *testing.B) {
    app := &App{count: 0}
    renderer := mofu.NewRenderer(80, 24, mofu.DefaultTheme())

    b.ReportAllocs()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        ctx := &mofu.RenderContext{
            Renderer: renderer,
            Bounds:   mofu.Rect{Width: 80, Height: 24},
        }
        app.Render(ctx)
        renderer.Flush()
    }
}
```

## Integration Testing

```go
func TestAppFlow(t *testing.T) {
    app := NewApp()

    // Simulate user interaction
    events := []mofu.Event{
        {Type: mofu.EventKeyPress, Data: mofu.KeyEvent{Key: mofu.KeyDown}},
        {Type: mofu.EventKeyPress, Data: mofu.KeyEvent{Key: mofu.KeyDown}},
        {Type: mofu.EventKeyPress, Data: mofu.KeyEvent{Key: mofu.KeyUp}},
    }

    for _, event := range events {
        app.HandleEvent(event)
    }

    if app.count != 1 {
        t.Errorf("count = %d, want 1", app.count)
    }
}
```

## Test Utilities

```go
// Create test event
func keyEvent(key mofu.Key) mofu.Event {
    return mofu.Event{
        Type: mofu.EventKeyPress,
        Data: mofu.KeyEvent{Key: key},
    }
}

// Create test key press
func charEvent(r rune) mofu.Event {
    return mofu.Event{
        Type: mofu.EventKeyPress,
        Data: mofu.KeyEvent{Runes: []byte{r}},
    }
}

// Assert state
func assertCount(t *testing.T, app *App, expected int) {
    t.Helper()
    if app.count != expected {
        t.Errorf("count = %d, want %d", app.count, expected)
    }
}
```

## Running Tests

```bash
# Run all tests
go test ./...

# Run specific test
go test -v ./...

# Run benchmarks
go test -bench=. -benchmem ./...

# Run with coverage
go test -cover ./...
```
