# Tutorial: Building a Chat Application

Build a real-time chat interface with MOFU.

## Step 1: Setup

```bash
mkdir chatapp && cd chatapp
go mod init chatapp
go get github.com/xanstomper/mofu
```

## Step 2: Create the App

```go
package main

import (
    "fmt"
    "os"
    "strings"
    "time"
    "github.com/xanstomper/mofu"
    "github.com/xanstomper/mofu/widgets"
)

type ChatApp struct {
    mofu.Minimal
    messages  []Message
    input     *widgets.InputNode
    width     int
    height    int
}

type Message struct {
    User    string
    Content string
    Time    time.Time
    IsMe    bool
}

func NewChatApp() *ChatApp {
    app := &ChatApp{
        messages: []Message{
            {User: "System", Content: "Welcome to MOFU Chat!", Time: time.Now()},
        },
        input: widgets.NewInput(),
    }
    app.input.Placeholder = "Type a message..."
    app.input.OnSubmit = func(value string) mofu.Cmd {
        if value == "" {
            return nil
        }
        app.messages = append(app.messages, Message{
            User:    "You",
            Content: value,
            Time:    time.Now(),
            IsMe:    true,
        })
        app.input.SetValue("")
        return nil
    }
    return app
}
```

## Step 3: Implement Render

```go
func (c *ChatApp) Render(ctx *mofu.RenderContext) {
    r := ctx.Bounds
    c.width = r.Width
    c.height = r.Height

    // Title
    titleStyle := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
    ctx.Renderer.WriteString(" MOFU Chat", r.X, r.Y, titleStyle.Foreground, titleStyle.Background, titleStyle.Attrs)

    // Separator
    sep := strings.Repeat("─", r.Width-2)
    ctx.Renderer.WriteString(sep, r.X+1, r.Y+1, mofu.Hex("444444"), mofu.ColorBlack, 0)

    // Messages
    msgY := r.Y + 2
    msgH := r.Height - 5
    start := 0
    if len(c.messages) > msgH {
        start = len(c.messages) - msgH
    }
    for i := start; i < len(c.messages); i++ {
        if msgY >= r.Y+r.Height-3 {
            break
        }
        msg := c.messages[i]
        text := fmt.Sprintf("[%s] %s: %s", msg.Time.Format("15:04"), msg.User, msg.Content)
        if len(text) > r.Width-2 {
            text = text[:r.Width-5] + "..."
        }
        style := mofu.DefaultStyle().Fg(mofu.Hex("e0e0e0"))
        if msg.IsMe {
            style = mofu.DefaultStyle().Fg(mofu.Hex("89b4fa"))
        }
        ctx.Renderer.WriteString(text, r.X+1, msgY, style.Foreground, style.Background, style.Attrs)
        msgY++
    }

    // Input
    c.input.SetBounds(mofu.Rect{X: r.X, Y: r.Y + r.Height - 2, Width: r.Width, Height: 1})
    c.input.Render(ctx)

    // Status
    ctx.Renderer.WriteString(" Enter: Send │ Esc: Quit", r.X, r.Y+r.Height-1, mofu.Hex("666666"), mofu.ColorBlack, 0)
}
```

## Step 4: Handle Events

```go
func (c *ChatApp) HandleEvent(event mofu.Event) mofu.Cmd {
    if event.Type == mofu.EventKeyPress {
        ke := event.Data.(mofu.KeyEvent)
        if ke.Key == mofu.KeyEsc {
            return mofu.QuitCmd()
        }
    }
    return c.input.HandleEvent(event)
}
```

## Step 5: Run

```go
func main() {
    app := NewChatApp()
    if err := mofu.Run(app); err != nil {
        fmt.Fprintf(os.Stderr, "error: %v\n", err)
        os.Exit(1)
    }
}
```

## What You Learned

- Using widgets (Input) inside your app
- Handling real-time message updates
- Scrollable message history
- Input submission handling
- Time-based message formatting

## Next Steps

- Add user names/colors
- Add message history persistence
- Add file sharing
- Add emoji support
- Deploy as a network service
