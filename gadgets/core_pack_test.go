package gadgets

import (
	"testing"
	"time"

	"github.com/xanstomper/mofu"
)

// ---------------------------------------------------------------------------
// AITrace tests
// ---------------------------------------------------------------------------

func TestAITraceAddToken(t *testing.T) {
	at := NewAITrace("trace")
	at.AddToken(TokenEvent{Model: "gpt-4", Token: "hello", Type: TokenText, Cost: 0.001})
	at.AddToken(TokenEvent{Model: "gpt-4", Token: "world", Type: TokenText, Cost: 0.001})

	stats := at.Stats()
	if stats.TotalTokens != 2 {
		t.Fatalf("total tokens = %d, want 2", stats.TotalTokens)
	}
	if stats.TotalCost < 0.001 {
		t.Fatalf("total cost = %f, want > 0.001", stats.TotalCost)
	}
	if stats.Models["gpt-4"] != 2 {
		t.Fatalf("gpt-4 tokens = %d, want 2", stats.Models["gpt-4"])
	}
}

func TestAITraceRender(t *testing.T) {
	at := NewAITrace("trace")
	at.AddToken(TokenEvent{Model: "gpt-4", Token: "hello", Type: TokenText, Cost: 0.001})
	at.AddToken(TokenEvent{Model: "gpt-4", Token: "tool", Type: TokenToolCall, Cost: 0.002})

	nodes := at.Render(nil)
	if len(nodes) < 3 {
		t.Fatalf("rendered %d nodes, want >= 3", len(nodes))
	}
}

func TestTokenTypeString(t *testing.T) {
	cases := []struct {
		tt   TokenType
		want string
	}{
		{TokenText, "text"},
		{TokenToolCall, "tool_call"},
		{TokenToolResult, "tool_result"},
		{TokenThinking, "thinking"},
		{TokenError, "error"},
	}
	for _, c := range cases {
		if got := c.tt.String(); got != c.want {
			t.Errorf("TokenType(%d).String() = %q, want %q", c.tt, got, c.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Timeline tests
// ---------------------------------------------------------------------------

func TestTimelineAddEvent(t *testing.T) {
	tl := NewTimeline("timeline")
	tl.AddEvent(TimelineEvent{Label: "deploy", Category: "ops"})
	tl.AddEvent(TimelineEvent{Label: "error", Category: "alert", Color: mofu.Hex("ff0000")})

	nodes := tl.Render(nil)
	if len(nodes) < 3 {
		t.Fatalf("rendered %d nodes, want >= 3", len(nodes))
	}
}

func TestTimelineSetWindow(t *testing.T) {
	tl := NewTimeline("timeline")
	tl.SetWindow(10 * time.Minute)

	tl.AddEvent(TimelineEvent{
		Timestamp: time.Now().Add(-15 * time.Minute),
		Label:     "old",
	})

	tl.AddEvent(TimelineEvent{
		Timestamp: time.Now(),
		Label:     "new",
	})

	nodes := tl.Render(nil)
	found := false
	for _, n := range nodes {
		if n.Content != "" {
			found = true
		}
	}
	if !found {
		t.Fatal("should have rendered content")
	}
}

func TestTimelineMaxEvents(t *testing.T) {
	tl := NewTimeline("timeline")
	tl.maxEvents = 5

	for i := 0; i < 10; i++ {
		tl.AddEvent(TimelineEvent{Label: "event"})
	}

	tl.mu.Lock()
	count := len(tl.events)
	tl.mu.Unlock()

	if count != 5 {
		t.Fatalf("events = %d, want 5 (max)", count)
	}
}
