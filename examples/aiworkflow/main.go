package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/xanstomper/mofu"
	"github.com/xanstomper/mofu/agent"
)

type AIWorkflow struct {
	mofu.Minimal
	agent    *agent.Agent
	tools    *agent.ToolPanel
	costs    *agent.CostBar
	thinking *agent.ThinkingDisplay
	width    int
	height   int
	step     int
}

func NewAIWorkflow() *AIWorkflow {
	a := agent.NewAgent("mofu-assistant")
	a.State = agent.StateIdle

	return &AIWorkflow{
		agent:    a,
		tools:    agent.NewToolPanel(),
		costs:    agent.NewCostBar(128000),
		thinking: agent.NewThinkingDisplay(),
	}
}

func (w *AIWorkflow) simulateWorkflow() {
	w.agent.BeginThinking("Analyzing the codebase to understand architecture...")
	w.thinking.AddStep("Architecture Analysis", "Scanning directory structure\nIdentifying key packages\nMapping dependencies", 250)

	w.agent.BeginToolCall("list_files", "src/**/*.go")
	w.tools.Begin("list_files", "src/**/*.go")
	w.agent.EndToolCall("Found 42 Go files across 8 packages", nil)
	w.tools.End("list_files", "42 files found", false)
	w.costs.AddTokens(500, 0, 0.001, 0)

	w.agent.BeginToolCall("read_file", "src/renderer.go")
	w.tools.Begin("read_file", "src/renderer.go")
	w.agent.EndToolCall("Read 317 lines - scene buffer, diff renderer", nil)
	w.tools.End("read_file", "317 lines read", false)
	w.costs.AddTokens(800, 0, 0.002, 0)

	w.agent.BeginThinking("Understanding the rendering pipeline...")
	w.thinking.AddStep("Render Pipeline", "SceneBuffer → Diff → Terminal\nDouble-buffered rendering\nDirty rect tracking", 180)
	w.agent.EndThinking()

	w.agent.AppendStream("# Architecture Analysis\n\n")
	w.agent.AppendStream("The MOFU codebase uses a reactive rendering pipeline\n")
	w.agent.AppendStream("with scene buffer diffing for efficient terminal updates.\n\n")
	w.agent.AppendStream("Key components:\n")
	w.agent.AppendStream("- **SceneBuffer**: Pre-allocated cell grid\n")
	w.agent.AppendStream("- **Diff Renderer**: Only sends changed cells\n")
	w.agent.AppendStream("- **State Graph**: Dirty-bit propagation\n")
	w.agent.AppendStream("- **Layout Engine**: Constraint-based layout\n")
	w.costs.AddTokens(0, 450, 0, 0.003)

	w.agent.BeginToolCall("search", "TODO|FIXME|HACK")
	w.tools.Begin("search", "TODO|FIXME|HACK")
	w.agent.EndToolCall("Found 3 TODOs, 0 FIXMEs, 0 HACKs", nil)
	w.tools.End("search", "3 results", false)
	w.costs.AddTokens(300, 0, 0.001, 0)

	w.agent.FinishStep(2050, 0.007)
}

func (w *AIWorkflow) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	w.width = r.Width
	w.height = r.Height

	// Split layout: 2/3 agent, 1/3 sidebar
	leftW := r.Width * 2 / 3
	rightW := r.Width - leftW

	// Agent panel (left, top 80%)
	agentH := r.Height * 4 / 5
	agentCtx := &mofu.RenderContext{
		Bounds:   mofu.Rect{X: r.X, Y: r.Y, Width: leftW - 1, Height: agentH},
		Renderer: ctx.Renderer,
	}
	w.agent.Render(agentCtx)

	// Separator
	ctx.Renderer.WriteString("│", r.X+leftW-1, r.Y, mofu.Hex("444444"), mofu.ColorBlack, 0)

	// Tools panel (right, top 50%)
	toolsH := r.Height * 3 / 5
	toolsCtx := &mofu.RenderContext{
		Bounds:   mofu.Rect{X: r.X + leftW, Y: r.Y, Width: rightW, Height: toolsH},
		Renderer: ctx.Renderer,
	}
	w.tools.Render(toolsCtx)

	// Thinking panel (right, bottom 50%)
	thinkingY := r.Y + toolsH
	thinkingH := r.Height - toolsH - 2
	thinkingCtx := &mofu.RenderContext{
		Bounds:   mofu.Rect{X: r.X + leftW, Y: thinkingY, Width: rightW, Height: thinkingH},
		Renderer: ctx.Renderer,
	}
	w.thinking.Render(thinkingCtx)

	// Bottom separator
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, r.Y+r.Height-2, mofu.Hex("444444"), mofu.ColorBlack, 0)

	// Cost bar (bottom)
	costCtx := &mofu.RenderContext{
		Bounds:   mofu.Rect{X: r.X, Y: r.Y + r.Height - 1, Width: r.Width, Height: 1},
		Renderer: ctx.Renderer,
	}
	w.costs.Render(costCtx)
}

func (w *AIWorkflow) HandleEvent(event mofu.Event) mofu.Cmd {
	if event.Type != mofu.EventKeyPress {
		return nil
	}
	ke := event.Data.(mofu.KeyEvent)

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()
	case ke.Key == mofu.KeyCtrlC:
		return mofu.QuitCmd()
	case len(ke.Runes) > 0 && ke.Runes[0] == 'r':
		w.step = 0
		w.agent = agent.NewAgent("mofu-assistant")
		w.tools = agent.NewToolPanel()
		w.costs = agent.NewCostBar(128000)
		w.thinking = agent.NewThinkingDisplay()
		go w.simulateWorkflow()
	}
	return nil
}

func main() {
	app := NewAIWorkflow()
	go app.simulateWorkflow()

	if err := mofu.Run(app); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
