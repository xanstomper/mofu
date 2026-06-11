package agent

import (
	"sync"
	"testing"
	"time"

	"github.com/xanstomper/mofu"
)

func TestOrchestratorAddAgent(t *testing.T) {
	o := NewOrchestrator("tabs")
	a := o.AddAgent("agent-1")
	if a.Name != "agent-1" {
		t.Errorf("expected agent-1, got %s", a.Name)
	}
	if len(o.Agents) != 1 {
		t.Errorf("expected 1 agent, got %d", len(o.Agents))
	}
}

func TestOrchestratorRemoveAgent(t *testing.T) {
	o := NewOrchestrator("tabs")
	o.AddAgent("a")
	o.AddAgent("b")
	o.AddAgent("c")

	o.RemoveAgent("b")
	if len(o.Agents) != 2 {
		t.Errorf("expected 2 agents, got %d", len(o.Agents))
	}
	if o.Agents[0].Name != "a" || o.Agents[1].Name != "c" {
		t.Error("wrong agents remaining")
	}
}

func TestOrchestratorGetAgent(t *testing.T) {
	o := NewOrchestrator("tabs")
	o.AddAgent("find-me")
	a := o.GetAgent("find-me")
	if a == nil || a.Name != "find-me" {
		t.Error("expected to find agent")
	}
	if o.GetAgent("nope") != nil {
		t.Error("expected nil for missing agent")
	}
}

func TestOrchestratorActiveCount(t *testing.T) {
	o := NewOrchestrator("tabs")
	a1 := o.AddAgent("idle")
	a2 := o.AddAgent("busy")

	a1.State = StateIdle
	a2.State = StateStreaming

	if o.ActiveCount() != 1 {
		t.Errorf("expected 1 active, got %d", o.ActiveCount())
	}

	a1.State = StateThinking
	if o.ActiveCount() != 2 {
		t.Errorf("expected 2 active, got %d", o.ActiveCount())
	}
}

func TestEventTimelineAdd(t *testing.T) {
	et := NewEventTimeline(50)
	et.Add("agent-1", "tool_start", "read_file", mofu.Hex("89b4fa"))
	et.Add("agent-2", "thinking", "analyzing...", mofu.Hex("f9e2af"))

	if len(et.Events) != 2 {
		t.Errorf("expected 2 events, got %d", len(et.Events))
	}
	if et.Events[0].Agent != "agent-1" {
		t.Errorf("expected agent-1, got %s", et.Events[0].Agent)
	}
}

func TestEventTimelineMaxEvents(t *testing.T) {
	et := NewEventTimeline(5)
	for i := 0; i < 10; i++ {
		et.Add("agent", "info", "event", mofu.Hex("cdd6f4"))
	}
	if len(et.Events) != 5 {
		t.Errorf("expected 5 events, got %d", len(et.Events))
	}
}

func TestEventTimelineClear(t *testing.T) {
	et := NewEventTimeline(50)
	et.Add("agent", "info", "data", mofu.Hex("cdd6f4"))
	et.Clear()
	if len(et.Events) != 0 {
		t.Errorf("expected 0 events after clear")
	}
}

func TestAgentDashboard(t *testing.T) {
	d := NewAgentDashboard()
	a := d.AddAgent("worker")
	if a.Name != "worker" {
		t.Errorf("expected worker")
	}
	if len(d.Orchestrator.Agents) != 1 {
		t.Errorf("expected 1 agent in orchestrator")
	}
}

func TestAgentDashboardLog(t *testing.T) {
	d := NewAgentDashboard()
	d.LogEvent("agent-1", "tool_start", "bash: ls")
	d.LogEvent("agent-1", "tool_end", "3 files found")
	d.LogEvent("agent-2", "thinking", "planning...")

	if len(d.Timeline.Events) != 3 {
		t.Errorf("expected 3 events, got %d", len(d.Timeline.Events))
	}
}

func TestOrchestratorThreadSafety(t *testing.T) {
	o := NewOrchestrator("tabs")
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			a := o.AddAgent("agent")
			a.State = StateStreaming
			o.ActiveCount()
			o.RemoveAgent("agent")
		}(i)
	}
	wg.Wait()
}

func TestEventTimelineThreadSafety(t *testing.T) {
	et := NewEventTimeline(100)
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			et.Add("agent", "info", "event", mofu.Hex("cdd6f4"))
			et.Clear()
		}()
	}
	wg.Wait()
}

func TestTimelineEventTimestamp(t *testing.T) {
	before := time.Now()
	et := NewEventTimeline(10)
	et.Add("agent", "info", "test", mofu.Hex("cdd6f4"))
	after := time.Now()

	if et.Events[0].Timestamp.Before(before) || et.Events[0].Timestamp.After(after) {
		t.Error("timestamp out of range")
	}
}
