package agent

import (
	"sync"
	"testing"
	"time"
)

func TestAgentState(t *testing.T) {
	a := NewAgent("test-agent")
	if a.State != StateIdle {
		t.Errorf("expected idle, got %v", a.State)
	}
	if a.Name != "test-agent" {
		t.Errorf("expected test-agent, got %s", a.Name)
	}
}

func TestAgentThinking(t *testing.T) {
	a := NewAgent("test")
	a.BeginThinking("analyzing code...")
	if a.State != StateThinking {
		t.Errorf("expected thinking, got %v", a.State)
	}
	if a.Thinking != "analyzing code..." {
		t.Errorf("expected thinking content, got %s", a.Thinking)
	}
	a.EndThinking()
	if a.Thinking != "" {
		t.Errorf("expected empty thinking after end")
	}
}

func TestAgentToolCall(t *testing.T) {
	a := NewAgent("test")
	a.BeginToolCall("read_file", "path/to/file.go")
	if a.State != StateToolCall {
		t.Errorf("expected tool_call, got %v", a.State)
	}
	if len(a.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(a.Steps))
	}
	if a.Steps[0].ToolName != "read_file" {
		t.Errorf("expected read_file, got %s", a.Steps[0].ToolName)
	}

	a.EndToolCall("file contents here", nil)
	if a.State != StateStreaming {
		t.Errorf("expected streaming after end tool call, got %v", a.State)
	}
	if a.Steps[0].ToolOutput != "file contents here" {
		t.Errorf("expected output, got %s", a.Steps[0].ToolOutput)
	}
}

func TestAgentStream(t *testing.T) {
	a := NewAgent("test")
	a.AppendStream("Hello ")
	a.AppendStream("world!")
	if a.Current != "Hello world!" {
		t.Errorf("expected 'Hello world!', got %s", a.Current)
	}
}

func TestAgentFinishStep(t *testing.T) {
	a := NewAgent("test")
	a.Current = "response text"
	a.FinishStep(100, 0.001)
	if a.TotalTokens != 100 {
		t.Errorf("expected 100 tokens, got %d", a.TotalTokens)
	}
	if a.TotalCost != 0.001 {
		t.Errorf("expected 0.001 cost, got %f", a.TotalCost)
	}
	if a.Current != "" {
		t.Errorf("expected empty current after finish")
	}
	if len(a.Steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(a.Steps))
	}
}

func TestAgentMultipleToolCalls(t *testing.T) {
	a := NewAgent("test")
	a.BeginToolCall("read_file", "a.go")
	a.EndToolCall("content a", nil)
	a.BeginToolCall("write_file", "b.go")
	a.EndToolCall("written", nil)
	a.BeginToolCall("grep", "pattern")
	a.EndToolCall("found 5 matches", nil)

	if len(a.Steps) != 3 {
		t.Errorf("expected 3 steps, got %d", len(a.Steps))
	}
}

func TestToolCall(t *testing.T) {
	tc := NewToolCall("bash")
	if tc.Status != "running" {
		t.Errorf("expected running, got %s", tc.Status)
	}
	tc.SetOutput("done!")
	if tc.Status != "done" {
		t.Errorf("expected done, got %s", tc.Status)
	}
}

func TestToolCallError(t *testing.T) {
	tc := NewToolCall("api_call")
	tc.SetError("connection refused")
	if tc.Status != "error" {
		t.Errorf("expected error, got %s", tc.Status)
	}
	if tc.Error != "connection refused" {
		t.Errorf("expected error message")
	}
}

func TestTokenStream(t *testing.T) {
	ts := NewTokenStream(100)
	ts.Write("Hello ")
	ts.Write("world\n")
	ts.Write("new line")

	if ts.Tokens != 3 {
		t.Errorf("expected 3 tokens, got %d", ts.Tokens)
	}
	if len(ts.Buffer) != 2 {
		t.Errorf("expected 2 lines, got %d", len(ts.Buffer))
	}
	if ts.Buffer[0] != "Hello world" {
		t.Errorf("expected 'Hello world', got %s", ts.Buffer[0])
	}
	if ts.Buffer[1] != "new line" {
		t.Errorf("expected 'new line', got %s", ts.Buffer[1])
	}
}

func TestTokenStreamMaxLines(t *testing.T) {
	ts := NewTokenStream(3)
	for i := 0; i < 10; i++ {
		ts.Write("line\n")
	}
	if len(ts.Buffer) > 3 {
		t.Errorf("expected max 3 lines, got %d", len(ts.Buffer))
	}
}

func TestTokenStreamClear(t *testing.T) {
	ts := NewTokenStream(100)
	ts.Write("data")
	ts.Clear()
	if len(ts.Buffer) != 0 {
		t.Errorf("expected empty after clear")
	}
	if ts.Tokens != 0 {
		t.Errorf("expected 0 tokens after clear")
	}
}

func TestToolPanel(t *testing.T) {
	tp := NewToolPanel()
	tp.Begin("read_file", "path.go")
	if len(tp.Calls) != 1 {
		t.Errorf("expected 1 call, got %d", len(tp.Calls))
	}
	if tp.Calls[0].Status != "running" {
		t.Errorf("expected running")
	}

	tp.End("read_file", "contents", false)
	if tp.Calls[0].Status != "done" {
		t.Errorf("expected done")
	}
}

func TestToolPanelError(t *testing.T) {
	tp := NewToolPanel()
	tp.Begin("api_call", "endpoint")
	tp.End("api_call", "error msg", true)
	if tp.Calls[0].Status != "error" {
		t.Errorf("expected error")
	}
}

func TestCostBar(t *testing.T) {
	cb := NewCostBar(1000)
	cb.AddTokens(500, 200, 0.01, 0.02)
	if cb.TokensIn != 500 {
		t.Errorf("expected 500 tokens in")
	}
	if cb.TokensOut != 200 {
		t.Errorf("expected 200 tokens out")
	}
	if cb.CostIn != 0.01 {
		t.Errorf("expected 0.01 cost in")
	}
	cb.Reset()
	if cb.TokensIn != 0 {
		t.Errorf("expected 0 after reset")
	}
}

func TestThinkingDisplay(t *testing.T) {
	td := NewThinkingDisplay()
	td.AddStep("Analyze", "looking at code structure", 150)
	td.AddStep("Plan", "decomposition strategy", 200)

	if len(td.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(td.Steps))
	}
	if td.Steps[0].Duration != 150 {
		t.Errorf("expected 150ms")
	}
}

func TestWorkflowView(t *testing.T) {
	wv := NewWorkflowView("my-agent")
	if wv.Agent.Name != "my-agent" {
		t.Errorf("expected my-agent")
	}
	if wv.Tools == nil {
		t.Error("expected tools panel")
	}
	if wv.Costs == nil {
		t.Error("expected costs bar")
	}
}

func TestStepDuration(t *testing.T) {
	s := Step{StartedAt: time.Now().Add(-time.Second)}
	if s.Duration() < time.Second {
		t.Errorf("expected at least 1s duration")
	}

	s2 := Step{StartedAt: time.Now(), EndedAt: time.Now().Add(50 * time.Millisecond)}
	if s2.Duration() != 50*time.Millisecond {
		t.Errorf("expected 50ms, got %v", s2.Duration())
	}
}

func TestMarkdownRenderer(t *testing.T) {
	md := NewMarkdownRenderer()
	md.SetContent("# Hello\n\nWorld\n\n- item 1\n- item 2")
	if md.Content != "# Hello\n\nWorld\n\n- item 1\n- item 2" {
		t.Errorf("content mismatch")
	}
}

func TestAgentThreadSafety(t *testing.T) {
	a := NewAgent("test")
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.BeginThinking("thinking")
			a.AppendStream("token")
			a.FinishStep(10, 0.001)
			a.BeginToolCall("tool", "input")
			a.EndToolCall("output", nil)
		}()
	}
	wg.Wait()
}

func TestTokenStreamThreadSafety(t *testing.T) {
	ts := NewTokenStream(100)
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ts.Write("data ")
			ts.Clear()
		}()
	}
	wg.Wait()
}

func TestToolPanelThreadSafety(t *testing.T) {
	tp := NewToolPanel()
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tp.Begin("tool", "input")
			tp.End("tool", "output", false)
		}()
	}
	wg.Wait()
}

func TestCostBarThreadSafety(t *testing.T) {
	cb := NewCostBar(1000)
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cb.AddTokens(10, 5, 0.001, 0.002)
			cb.Reset()
		}()
	}
	wg.Wait()
}
