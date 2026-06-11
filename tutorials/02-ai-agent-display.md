# Tutorial 2: Build an AI Agent Display with Streaming API

This tutorial builds a real-time AI agent display that streams from an OpenAI-compatible API. You'll learn the `agent/` package.

## What We're Building

An agent that connects to an API, streams responses token-by-token, shows tool calls, and tracks costs.

## Step 1: Connect to an API

```go
package main

import (
	"fmt"
	"os"

	"github.com/xanstomper/mofu"
	"github.com/xanstomper/mofu/agent"
)

func main() {
	a := agent.NewInstantAgent(
		"assistant",
		"https://api.openai.com/v1/chat/completions",
		os.Getenv("OPENAI_API_KEY"),
		"gpt-4",
	)

	a.SetSystemPrompt("You are a helpful coding assistant.")
	a.OnToken(func(token string) {
		fmt.Print(token) // In real app, render to TUI
	})

	a.Send("Explain MOFU in 3 sentences.")
}
```

`NewInstantAgent` creates an agent connected to any OpenAI-compatible API (OpenAI, Anthropic, Ollama, vLLM, etc).

## Step 2: Add Tool Calls

Register tools the agent can invoke:

```go
a.RegisterTool("bash", func(input string) (string, error) {
	// Execute shell command
	return "output", nil
})

a.RegisterTool("read_file", func(input string) (string, error) {
	// Read file contents
	return "file contents", nil
})
```

## Step 3: Display with TUI

Use the `agent.StreamDisplay` for a polished terminal UI:

```go
type App struct {
	mofu.Minimal
	display *agent.StreamDisplay
	agent   *agent.InstantAgent
	input   string
}

func NewApp(apiURL, apiKey, model string) *App {
	a := agent.NewInstantAgent("assistant", apiURL, apiKey, model)
	a.SetSystemPrompt("You are a helpful assistant.")

	return &App{
		agent:   a,
		display: agent.NewStreamDisplay(a),
	}
}

func (app *App) Render(ctx *mofu.RenderContext) {
	app.display.Render(ctx)

	// Input line at bottom
	r := ctx.Bounds
	input := "> " + app.input
	if len(input) > r.Width-1 {
		input = input[:r.Width-1]
	}
	ctx.Renderer.WriteString(input, r.X, r.Y+r.Height-1, mofu.Hex("cdd6f4"), mofu.Hex("1e1e2e"), 0)
}

func (app *App) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()
	case ke.Key == mofu.KeyEnter && len(app.input) > 0:
		app.agent.Send(app.input)
		app.input = ""
	case ke.Key == mofu.KeyBack && len(app.input) > 0:
		app.input = app.input[:len(app.input)-1]
	default:
		if len(ke.Runes) > 0 {
			app.input += string(ke.Runes)
		}
	}
	return nil
}

func main() {
	app := NewApp(
		"https://api.openai.com/v1/chat/completions",
		os.Getenv("OPENAI_API_KEY"),
		"gpt-4",
	)
	mofu.Run(app)
}
```

## Step 4: Multi-Agent Display

For multiple agents working in parallel:

```go
orch := agent.NewOrchestrator("tabs")
agent1 := orch.AddAgent("researcher")
agent2 := orch.AddAgent("coder")

// Feed tokens from each agent
agent1.AppendStream("Analyzing code...")
agent2.AppendStream("Writing tests...")

// Use in render:
func (app *App) Render(ctx *mofu.RenderContext) {
	orch.Render(ctx)
}
```

## Key Concepts

| Component | Purpose |
|-----------|---------|
| `InstantAgent` | Production agent with API streaming, tool calls, history |
| `APIStream` | HTTP client for OpenAI-compatible SSE endpoints |
| `StreamDisplay` | Polished terminal UI with panels and status bar |
| `Orchestrator` | Multi-agent tab display |
| `CostBar` | Token usage and cost tracking |
| `ToolPanel` | Side panel showing active tool calls |
| `VirtualScroll` | O(1) scroll through millions of log lines |
| `MarkdownRenderer` | Terminal-native markdown rendering |

## Environment Variables

```bash
export OPENAI_API_KEY="sk-..."
# Or for Ollama:
# No key needed, just set URL to http://localhost:11434/v1/chat/completions
```

## What You Learned

1. **InstantAgent** — wraps API streaming with callbacks
2. **APIStream** — connects to any OpenAI-compatible endpoint
3. **StreamDisplay** — instant terminal rendering of streamed tokens
4. **Orchestrator** — multi-agent tab display
5. **Zero-alloc streaming** — ring buffers, SSE parser, no GC pressure
