package agent

import (
	"fmt"
	"strings"
	"sync"

	"github.com/xanstomper/mofu"
)

// =========================================================================
// MarkdownRenderer — terminal-native markdown rendering
// Supports: headers, bold, italic, code blocks, lists, links, blockquotes
// =========================================================================

type MarkdownRenderer struct {
	mofu.Minimal
	Content    string
	ScrollY    int
	Width      int
	Height     int
	mu         sync.RWMutex
}

func NewMarkdownRenderer() *MarkdownRenderer {
	return &MarkdownRenderer{}
}

func (md *MarkdownRenderer) SetContent(content string) {
	md.mu.Lock()
	md.Content = content
	md.mu.Unlock()
}

func (md *MarkdownRenderer) Render(ctx *mofu.RenderContext) {
	md.mu.RLock()
	defer md.mu.RUnlock()

	r := ctx.Bounds
	md.Width = r.Width
	md.Height = r.Height
	lines := strings.Split(md.Content, "\n")

	y := r.Y
	inCodeBlock := false
	codeLang := ""

	for i := md.ScrollY; i < len(lines) && y < r.Y+r.Height; i++ {
		line := lines[i]

		// Code blocks
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			if inCodeBlock {
				inCodeBlock = false
				ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
				y++
			} else {
				inCodeBlock = true
				codeLang = strings.TrimSpace(strings.TrimPrefix(line, "```"))
				if codeLang != "" {
					ctx.Renderer.WriteString(" "+codeLang, r.X, y, mofu.Hex("6c7086"), mofu.ColorBlack, 0)
					y++
				}
				ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
				y++
			}
			continue
		}

		if inCodeBlock {
			display := "  " + line
			if len(display) > r.Width-1 {
				display = display[:r.Width-4] + "..."
			}
			ctx.Renderer.WriteString(display, r.X, y, mofu.Hex("a6e3a1"), mofu.Hex("1e1e2e"), 0)
			y++
			continue
		}

		trimmed := strings.TrimSpace(line)

		// Headers
		if strings.HasPrefix(trimmed, "### ") {
			text := trimmed[4:]
			text = md.stripInline(text)
			if len(text) > r.Width-2 {
				text = text[:r.Width-5] + "..."
			}
			ctx.Renderer.WriteString(" "+text, r.X, y, mofu.Hex("94e2d5"), mofu.ColorBlack, mofu.AttrBold)
			y++
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			text := trimmed[3:]
			text = md.stripInline(text)
			if len(text) > r.Width-2 {
				text = text[:r.Width-5] + "..."
			}
			ctx.Renderer.WriteString(" "+text, r.X, y, mofu.Hex("89b4fa"), mofu.ColorBlack, mofu.AttrBold)
			y++
			continue
		}
		if strings.HasPrefix(trimmed, "# ") {
			text := trimmed[2:]
			text = md.stripInline(text)
			if len(text) > r.Width-2 {
				text = text[:r.Width-5] + "..."
			}
			ctx.Renderer.WriteString(text, r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
			y++
			continue
		}

		// Horizontal rule
		if trimmed == "---" || trimmed == "***" || trimmed == "___" {
			ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
			y++
			continue
		}

		// Blockquote
		if strings.HasPrefix(trimmed, "> ") {
			text := trimmed[2:]
			text = md.renderInline(text, r.Width-4)
			ctx.Renderer.WriteString(" │ "+text, r.X, y, mofu.Hex("6c7086"), mofu.ColorBlack, 0)
			y++
			continue
		}

		// Unordered list
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			text := trimmed[2:]
			text = md.renderInline(text, r.Width-6)
			ctx.Renderer.WriteString("   • "+text, r.X, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
			y++
			continue
		}

		// Ordered list
		if len(trimmed) > 2 && trimmed[0] >= '0' && trimmed[0] <= '9' {
			dotIdx := strings.Index(trimmed, ". ")
			if dotIdx > 0 && dotIdx < 4 {
				num := trimmed[:dotIdx]
				text := trimmed[dotIdx+2:]
				text = md.renderInline(text, r.Width-6)
				ctx.Renderer.WriteString("   "+num+". "+text, r.X, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
				y++
				continue
			}
		}

		// Regular paragraph
		if trimmed != "" {
			text := md.renderInline(trimmed, r.Width-2)
			ctx.Renderer.WriteString(" "+text, r.X, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
			y++
		} else {
			y++
		}
	}
}

func (md *MarkdownRenderer) stripInline(text string) string {
	text = strings.ReplaceAll(text, "**", "")
	text = strings.ReplaceAll(text, "*", "")
	text = strings.ReplaceAll(text, "`", "")
	return text
}

func (md *MarkdownRenderer) renderInline(text string, maxW int) string {
	if len(text) > maxW {
		text = text[:maxW-3] + "..."
	}
	return text
}

func (md *MarkdownRenderer) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	md.mu.Lock()
	defer md.mu.Unlock()

	switch {
	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		lines := strings.Split(md.Content, "\n")
		if md.ScrollY < len(lines)-md.Height {
			md.ScrollY++
		}
	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if md.ScrollY > 0 {
			md.ScrollY--
		}
	case ke.Key == mofu.KeyPgDn:
		md.ScrollY += md.Height - 2
	case ke.Key == mofu.KeyPgUp:
		md.ScrollY -= md.Height - 2
		if md.ScrollY < 0 {
			md.ScrollY = 0
		}
	case ke.Key == mofu.KeyHome:
		md.ScrollY = 0
	case ke.Key == mofu.KeyEnd:
		lines := strings.Split(md.Content, "\n")
		md.ScrollY = len(lines) - md.Height
		if md.ScrollY < 0 {
			md.ScrollY = 0
		}
	}
	return nil
}

// =========================================================================
// ThinkingDisplay — collapsible thinking/reasoning display
// =========================================================================

type ThinkingDisplay struct {
	mofu.Minimal
	Steps     []ThinkingStep
	Expanded  map[int]bool
	Selected  int
	mu        sync.RWMutex
}

type ThinkingStep struct {
	Label   string
	Content string
	Duration int64
}

func NewThinkingDisplay() *ThinkingDisplay {
	return &ThinkingDisplay{Expanded: make(map[int]bool)}
}

func (td *ThinkingDisplay) AddStep(label, content string, durationMs int64) {
	td.mu.Lock()
	td.Steps = append(td.Steps, ThinkingStep{Label: label, Content: content, Duration: durationMs})
	td.mu.Unlock()
}

func (td *ThinkingDisplay) Render(ctx *mofu.RenderContext) {
	td.mu.RLock()
	defer td.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(" Reasoning", r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	for i, step := range td.Steps {
		if y >= r.Y+r.Height-1 {
			break
		}

		icon := "▸"
		if td.Expanded[i] {
			icon = "▾"
		}

		elapsed := ""
		if step.Duration > 0 {
			elapsed = fmt.Sprintf(" (%dms)", step.Duration)
		}

		style := mofu.DefaultStyle().Fg(mofu.Hex("89b4fa"))
		if i == td.Selected {
			style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
		}

		ctx.Renderer.WriteString(fmt.Sprintf(" %s %s%s", icon, step.Label, elapsed), r.X, y, style.Foreground, style.Background, style.Attrs)
		y++

		if td.Expanded[i] {
			lines := strings.Split(step.Content, "\n")
			for _, line := range lines {
				if y >= r.Y+r.Height-1 {
					break
				}
				display := "   " + line
				if len(display) > r.Width-2 {
					display = display[:r.Width-5] + "..."
				}
				ctx.Renderer.WriteString(display, r.X, y, mofu.Hex("6c7086"), mofu.ColorBlack, 0)
				y++
			}
		}
	}
}

func (td *ThinkingDisplay) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	td.mu.Lock()
	defer td.mu.Unlock()

	switch {
	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		if td.Selected < len(td.Steps)-1 {
			td.Selected++
		}
	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if td.Selected > 0 {
			td.Selected--
		}
	case ke.Key == mofu.KeyEnter || ke.Key == mofu.KeySpace:
		td.Expanded[td.Selected] = !td.Expanded[td.Selected]
	}
	return nil
}

// =========================================================================
// WorkflowView — complete agent workflow display
// =========================================================================

type WorkflowView struct {
	mofu.Minimal
	Agent      *Agent
	Tools      *ToolPanel
	Thinking   *ThinkingDisplay
	Costs      *CostBar
	Markdown   *MarkdownRenderer
	ShowMarkdown bool
	mu         sync.RWMutex
}

func NewWorkflowView(agentName string) *WorkflowView {
	return &WorkflowView{
		Agent:    NewAgent(agentName),
		Tools:    NewToolPanel(),
		Thinking: NewThinkingDisplay(),
		Costs:    NewCostBar(128000),
		Markdown: NewMarkdownRenderer(),
	}
}

func (wv *WorkflowView) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	wv.mu.RLock()
	defer wv.mu.RUnlock()

	topH := r.Height * 2 / 3
	botH := r.Height - topH

	leftW := r.Width * 2 / 3
	rightW := r.Width - leftW

	if topH > 2 {
		agentCtx := &mofu.RenderContext{
			Bounds:   mofu.Rect{X: r.X, Y: r.Y, Width: leftW - 1, Height: topH},
			Renderer: ctx.Renderer,
		}
		wv.Agent.Render(agentCtx)
	}

	if rightW > 15 && topH > 2 {
		toolsCtx := &mofu.RenderContext{
			Bounds:   mofu.Rect{X: r.X + leftW, Y: r.Y, Width: rightW, Height: topH},
			Renderer: ctx.Renderer,
		}
		wv.Tools.Render(toolsCtx)
	}

	if topH > 0 && topH < r.Height {
		ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, r.Y+topH, mofu.Hex("444444"), mofu.ColorBlack, 0)
	}

	if botH > 1 {
		botCtx := &mofu.RenderContext{
			Bounds:   mofu.Rect{X: r.X, Y: r.Y + topH + 1, Width: leftW - 1, Height: botH - 1},
			Renderer: ctx.Renderer,
		}
		if wv.ShowMarkdown {
			wv.Markdown.Render(botCtx)
		} else {
			wv.Thinking.Render(botCtx)
		}
	}

	if rightW > 15 && botH > 1 {
		costCtx := &mofu.RenderContext{
			Bounds:   mofu.Rect{X: r.X + leftW, Y: r.Y + topH + 1, Width: rightW, Height: botH - 1},
			Renderer: ctx.Renderer,
		}
		wv.Costs.Render(costCtx)
	}

	ctx.Renderer.WriteString("│", r.X+leftW-1, r.Y, mofu.Hex("444444"), mofu.ColorBlack, 0)
}

func (wv *WorkflowView) HandleEvent(e mofu.Event) mofu.Cmd {
	return wv.Agent.HandleEvent(e)
}
