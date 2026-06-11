package agent

import (
	"sync"
	"testing"
	"time"

	"github.com/xanstomper/mofu"
)

func TestNewLiveDataFeed(t *testing.T) {
	f := NewLiveDataFeed[int](100)
	f.Send(1)
	f.Send(2)
	f.Send(3)

	v, ok := f.Recv()
	if !ok || v != 1 {
		t.Errorf("expected 1, got %d (ok=%v)", v, ok)
	}
}

func TestLiveDataFeedBackpressure(t *testing.T) {
	f := NewLiveDataFeed[int](2)
	f.Send(1)
	f.Send(2)
	f.Send(3) // should drop 1

	if f.Dropped() != 1 {
		t.Errorf("expected 1 dropped, got %d", f.Dropped())
	}

	v, ok := f.Recv()
	if !ok || v != 2 {
		t.Errorf("expected 2, got %d", v)
	}
}

func TestLiveDataFeedClose(t *testing.T) {
	f := NewLiveDataFeed[int](10)
	f.Close()

	ok := f.Send(1)
	if ok {
		t.Error("expected false after close")
	}

	_, ok = f.Recv()
	if ok {
		t.Error("expected false from closed feed")
	}
}

func TestLiveDataFeedTimeout(t *testing.T) {
	f := NewLiveDataFeed[int](10)
	_, ok := f.RecvTimeout(10 * time.Millisecond)
	if ok {
		t.Error("expected timeout")
	}
}

func TestParseStreamEvent(t *testing.T) {
	// Normal event
	chunk := parseStreamEvent(`{"choices":[{"delta":{"content":"hello"}}]}`)
	if chunk == nil || chunk.Content != "hello" {
		t.Errorf("expected 'hello', got %v", chunk)
	}

	// Done
	chunk = parseStreamEvent("[DONE]")
	if chunk == nil || !chunk.Done {
		t.Error("expected done")
	}

	// Empty
	chunk = parseStreamEvent("")
	if chunk == nil || !chunk.Done {
		t.Error("expected done for empty")
	}

	// With usage only
	chunk = parseStreamEvent(`{"usage":{"prompt_tokens":10,"completion_tokens":20,"total_tokens":30}}`)
	if chunk == nil {
		t.Fatal("expected non-nil chunk")
	}
	t.Logf("chunk.Usage=%v", chunk.Usage)
	if chunk.Usage == nil {
		t.Fatal("expected non-nil usage")
	}
	if chunk.Usage.TotalTokens != 30 {
		t.Errorf("expected 30 tokens, got TotalTokens=%d", chunk.Usage.TotalTokens)
	}

	// With usage and finish_reason
	chunk = parseStreamEvent(`{"choices":[{"delta":{"content":""},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":10,"total_tokens":15}}`)
	if chunk == nil {
		t.Fatal("expected non-nil chunk")
	}
	if !chunk.Done {
		t.Error("expected done")
	}
	if chunk.Usage == nil || chunk.Usage.TotalTokens != 15 {
		t.Errorf("expected 15 tokens, got %v", chunk.Usage)
	}
}

func TestRenderPipeline(t *testing.T) {
	rp := NewRenderPipeline(60)
	rp.Submit(RenderCommand{Type: "text", Text: "hello"})
	rp.Submit(RenderCommand{Type: "text", Text: "world"})

	count, drops := rp.Stats()
	if count != 0 || drops != 0 {
		t.Errorf("expected 0/0, got %d/%d", count, drops)
	}
}

func TestRenderPipelineBackpressure(t *testing.T) {
	rp := NewRenderPipeline(60)
	for i := 0; i < 2000; i++ {
		rp.Submit(RenderCommand{Type: "text", Text: "x"})
	}

	_, drops := rp.Stats()
	if drops == 0 {
		t.Error("expected some drops under backpressure")
	}
}

func TestRenderPipelineBatch(t *testing.T) {
	rp := NewRenderPipeline(60)
	cmds := []RenderCommand{
		{Type: "text", Text: "a"},
		{Type: "text", Text: "b"},
		{Type: "text", Text: "c"},
	}
	rp.SubmitBatch(cmds)
}

func TestStreamDisplayBuffer(t *testing.T) {
	b := NewStreamDisplayBuffer(100)
	b.Append("line 1")
	b.Append("line 2")
	b.Append("line 3")

	lines := b.Lines()
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
	if lines[0] != "line 1" {
		t.Errorf("expected 'line 1', got %s", lines[0])
	}
}

func TestStreamDisplayBufferMaxLines(t *testing.T) {
	b := NewStreamDisplayBuffer(5)
	for i := 0; i < 10; i++ {
		b.Append("line")
	}
	if len(b.Lines()) != 5 {
		t.Errorf("expected 5, got %d", len(b.Lines()))
	}
}

func TestStreamDisplayBufferScroll(t *testing.T) {
	b := NewStreamDisplayBuffer(100)
	for i := 0; i < 50; i++ {
		b.Append("line")
	}
	b.ScrollDown(10)
	b.ScrollUp(5)
	b.mu.RLock()
	if b.scrollY != 5 {
		t.Errorf("expected scrollY=5, got %d", b.scrollY)
	}
	b.mu.RUnlock()
}

func TestStreamDisplayBufferClear(t *testing.T) {
	b := NewStreamDisplayBuffer(100)
	b.Append("data")
	b.Clear()
	if len(b.Lines()) != 0 {
		t.Error("expected empty after clear")
	}
}

func TestAPIStream(t *testing.T) {
	api := NewAPIStream("http://localhost:11434/api/chat", "", "llama3")
	if api.URL != "http://localhost:11434/api/chat" {
		t.Error("URL mismatch")
	}
	if api.Model != "llama3" {
		t.Error("model mismatch")
	}
	if api.MaxTokens != 4096 {
		t.Errorf("expected 4096, got %d", api.MaxTokens)
	}
}

func TestStreamRequest(t *testing.T) {
	req := StreamRequest{
		Model: "gpt-4",
		Messages: []Message{
			{Role: "system", Content: "You are helpful."},
			{Role: "user", Content: "Hello"},
		},
		Stream: true,
	}

	if req.Model != "gpt-4" {
		t.Error("model mismatch")
	}
	if len(req.Messages) != 2 {
		t.Error("expected 2 messages")
	}
}

func TestInstantAgent(t *testing.T) {
	agent := NewInstantAgent("test", "http://localhost:11434/v1/chat/completions", "", "llama3")
	agent.SetSystemPrompt("You are a test assistant.")

	if agent.GetState() != StateIdle {
		t.Errorf("expected idle, got %v", agent.GetState())
	}
	if agent.GetOutput() != "" {
		t.Error("expected empty output")
	}
}

func TestInstantAgentReset(t *testing.T) {
	agent := NewInstantAgent("test", "http://localhost:11434/v1/chat/completions", "", "llama3")
	agent.mu.Lock()
	agent.history = []Message{{Role: "user", Content: "hi"}}
	agent.output = "hello"
	agent.totalTokens = 100
	agent.mu.Unlock()

	agent.Reset()

	if agent.GetOutput() != "" {
		t.Error("expected empty after reset")
	}
	if len(agent.GetHistory()) != 0 {
		t.Error("expected empty history")
	}
}

func TestInstantAgentToolRegistration(t *testing.T) {
	agent := NewInstantAgent("test", "http://localhost:11434/v1/chat/completions", "", "llama3")
	agent.RegisterTool("bash", func(input string) (string, error) {
		return "output", nil
	})

	agent.mu.RLock()
	_, ok := agent.toolHandlers["bash"]
	agent.mu.RUnlock()
	if !ok {
		t.Error("tool not registered")
	}
}

func TestPanel(t *testing.T) {
	p := NewPanel("Test")
	if p.Title != "Test" {
		t.Error("title mismatch")
	}
	if p.Focused {
		t.Error("should not be focused")
	}

	p.SetFocused(true)
	if !p.Focused {
		t.Error("should be focused now")
	}
	if p.BorderFg != mofu.Hex("89b4fa") {
		t.Error("border color should be blue when focused")
	}

	p.SetFocused(false)
	if p.Focused {
		t.Error("should not be focused")
	}
}

func TestStatusBar(t *testing.T) {
	sb := NewStatusBar()
	sb.Set("model", "gpt-4", "🤖", mofu.Hex("89b4fa"))
	sb.Set("status", "ready", "●", mofu.Hex("a6e3a1"))

	if len(sb.Sections) != 2 {
		t.Errorf("expected 2 sections, got %d", len(sb.Sections))
	}

	// Update existing
	sb.Set("status", "busy", "●", mofu.Hex("f9e2af"))
	if len(sb.Sections) != 2 {
		t.Errorf("expected 2 sections after update, got %d", len(sb.Sections))
	}
	if sb.Sections[1].Value != "busy" {
		t.Errorf("expected 'busy', got %s", sb.Sections[1].Value)
	}
}

func TestNotificationBar(t *testing.T) {
	nb := NewNotificationBar(3)
	nb.Push("hello", "info")
	nb.Push("warning!", "warning")
	nb.Push("error!", "error")
	nb.Push("fixed", "success")

	nb.mu.RLock()
	if len(nb.notifications) != 4 {
		t.Errorf("expected 4 notifications, got %d", len(nb.notifications))
	}
	nb.mu.RUnlock()
}

func TestAppLayout(t *testing.T) {
	al := NewAppLayout()
	al.AddPanel("Panel 1")
	al.AddPanel("Panel 2")
	al.AddPanel("Panel 3")

	al.SetFocus(1)
	if al.FocusedIdx != 1 {
		t.Errorf("expected focus 1, got %d", al.FocusedIdx)
	}
	if !al.Panels[1].Focused {
		t.Error("panel 1 should be focused")
	}
	if al.Panels[0].Focused {
		t.Error("panel 0 should not be focused")
	}
}

func TestAppLayoutCycle(t *testing.T) {
	al := NewAppLayout()
	al.AddPanel("A")
	al.AddPanel("B")
	al.AddPanel("C")

	al.SetFocus(0)
	al.NextFocus()
	if al.FocusedIdx != 1 {
		t.Errorf("expected 1, got %d", al.FocusedIdx)
	}

	al.NextFocus()
	if al.FocusedIdx != 2 {
		t.Errorf("expected 2, got %d", al.FocusedIdx)
	}

	al.NextFocus()
	if al.FocusedIdx != 0 {
		t.Errorf("expected 0 (wrap), got %d", al.FocusedIdx)
	}

	al.PrevFocus()
	if al.FocusedIdx != 2 {
		t.Errorf("expected 2 (prev wrap), got %d", al.FocusedIdx)
	}
}

func TestStreamDisplay(t *testing.T) {
	agent := NewInstantAgent("test", "http://localhost:11434/v1/chat/completions", "", "llama3")
	sd := NewStreamDisplay(agent)

	if sd.buffer == nil {
		t.Error("expected buffer")
	}
	if sd.panels == nil {
		t.Error("expected panels")
	}
}

func TestStreamDisplayBufferThreadSafety(t *testing.T) {
	b := NewStreamDisplayBuffer(100)
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				b.Append("line")
				b.Lines()
				b.ScrollDown(1)
				b.ScrollUp(1)
			}
		}()
	}
	wg.Wait()
}

func TestLiveDataFeedThreadSafety(t *testing.T) {
	f := NewLiveDataFeed[int](100)
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				f.Send(j)
				f.Recv()
				f.Len()
			}
		}()
	}
	wg.Wait()
}

func TestAppLayoutThreadSafety(t *testing.T) {
	al := NewAppLayout()
	al.AddPanel("A")
	al.AddPanel("B")
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			al.NextFocus()
			al.PrevFocus()
		}()
	}
	wg.Wait()
}

func TestNotificationBarThreadSafety(t *testing.T) {
	nb := NewNotificationBar(3)
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			nb.Push("msg", "info")
		}()
	}
	wg.Wait()
}

func TestStatusBarThreadSafety(t *testing.T) {
	sb := NewStatusBar()
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sb.Set("key", "val", "●", mofu.Hex("cdd6f4"))
		}()
	}
	wg.Wait()
}
