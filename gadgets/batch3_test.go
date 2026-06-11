package gadgets

import (
	"sync"
	"testing"

	"github.com/xanstomper/mofu"
)

func hex(s string) mofu.Color {
	return mofu.Hex(s)
}

func TestRealPieChart(t *testing.T) {
	g := NewRealPieChart("pie")
	g.SetSegments([]PieSegment{
		{Label: "Go", Value: 45, Color: hex("89b4fa")},
		{Label: "Rust", Value: 30, Color: hex("f38ba8")},
		{Label: "Python", Value: 25, Color: hex("a6e3a1")},
	})

	total := g.GetTotal()
	if total != 100 {
		t.Errorf("expected total 100, got %f", total)
	}

	g.mu.RLock()
	n := len(g.Segments)
	g.mu.RUnlock()
	if n != 3 {
		t.Errorf("expected 3 segments, got %d", n)
	}
}

func TestRealMiniMap(t *testing.T) {
	g := NewRealMiniMap("minimap")
	lines := make([]string, 100)
	for i := range lines {
		lines[i] = "line content here"
	}
	g.SetContent(lines, 10, 20)

	g.mu.RLock()
	if g.TotalHeight != 100 {
		t.Errorf("expected total height 100, got %d", g.TotalHeight)
	}
	if g.ViewportY != 10 {
		t.Errorf("expected viewport Y 10, got %d", g.ViewportY)
	}
	g.mu.RUnlock()
}

func TestRealTagCloud(t *testing.T) {
	g := NewRealTagCloud("tags")
	g.SetTags([]TagEntry{
		{Name: "Go", Weight: 50, Color: hex("89b4fa")},
		{Name: "Rust", Weight: 30, Color: hex("f38ba8")},
		{Name: "Python", Weight: 20, Color: hex("a6e3a1")},
	})

	g.mu.RLock()
	if g.MaxWeight != 50 {
		t.Errorf("expected max weight 50, got %d", g.MaxWeight)
	}
	g.mu.RUnlock()

	g.AddTag("Go", 10, hex("89b4fa"))
	g.mu.RLock()
	if g.MaxWeight != 60 {
		t.Errorf("expected max weight 60, got %d", g.MaxWeight)
	}
	found := false
	for _, tag := range g.Tags {
		if tag.Name == "Go" && tag.Weight == 60 {
			found = true
		}
	}
	g.mu.RUnlock()
	if !found {
		t.Error("Go tag not updated")
	}
}

func TestRealScoreBoard(t *testing.T) {
	g := NewRealScoreBoard("scores", 5)
	g.AddScore("Alice", 100)
	g.AddScore("Bob", 200)
	g.AddScore("Charlie", 150)

	g.mu.RLock()
	if len(g.Entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(g.Entries))
	}
	if g.Entries[0].Name != "Bob" {
		t.Errorf("expected Bob first, got %s", g.Entries[0].Name)
	}
	if g.Entries[0].Score != 200 {
		t.Errorf("expected Bob score 200, got %d", g.Entries[0].Score)
	}
	g.mu.RUnlock()

	g.AddScore("Bob", 50)
	g.mu.RLock()
	if g.Entries[0].Score != 250 {
		t.Errorf("expected Bob score 250, got %d", g.Entries[0].Score)
	}
	g.mu.RUnlock()

	g.Clear()
	g.mu.RLock()
	if len(g.Entries) != 0 {
		t.Errorf("expected 0 entries after clear, got %d", len(g.Entries))
	}
	g.mu.RUnlock()
}

func TestRealChatInput(t *testing.T) {
	g := NewRealChatInput("chat")

	g.AddMessage("Alice", "Hello!", hex("89b4fa"))
	g.AddMessage("Bob", "Hi there!", hex("a6e3a1"))

	msgs := g.GetMessages()
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Author != "Alice" {
		t.Errorf("expected Alice, got %s", msgs[0].Author)
	}
	if msgs[0].Content != "Hello!" {
		t.Errorf("expected Hello!, got %s", msgs[0].Content)
	}

	g.ClearHistory()
	msgs = g.GetMessages()
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages after clear, got %d", len(msgs))
	}
}

func TestRealNotificationPanel(t *testing.T) {
	g := NewRealNotificationPanel("notifs")

	g.Add("Alert", "Disk full", "warning", hex("fab387"))
	g.Add("Info", "Deploy done", "info", hex("a6e3a1"))
	g.Add("Error", "Crash!", "error", hex("f38ba8"))

	if g.GetUnreadCount() != 3 {
		t.Errorf("expected 3 unread, got %d", g.GetUnreadCount())
	}

	g.MarkAllRead()
	if g.GetUnreadCount() != 0 {
		t.Errorf("expected 0 unread, got %d", g.GetUnreadCount())
	}

	g.Dismiss(0)
	g.mu.RLock()
	n := len(g.Notifications)
	g.mu.RUnlock()
	if n != 2 {
		t.Errorf("expected 2 notifications after dismiss, got %d", n)
	}
}

func TestRealResourceMonitor(t *testing.T) {
	g := NewRealResourceMonitor("resources")
	g.Set("CPU", 65, 100, "%", hex("89b4fa"))
	g.Set("Memory", 8.5, 16, "GB", hex("a6e3a1"))

	val, ok := g.Get("CPU")
	if !ok || val != 65 {
		t.Errorf("expected CPU=65, got %f (found=%v)", val, ok)
	}

	val, ok = g.Get("Memory")
	if !ok || val != 8.5 {
		t.Errorf("expected Memory=8.5, got %f (found=%v)", val, ok)
	}

	_, ok = g.Get("Disk")
	if ok {
		t.Error("expected Disk not found")
	}
}

func TestRealQueryBuilder(t *testing.T) {
	g := NewRealQueryBuilder("query", []string{"name", "age", "email", "status"})

	query := g.BuildQuery()
	if query != "SELECT * FROM table" {
		t.Errorf("expected empty query, got %s", query)
	}

	g.AddCondition("age", ">", "18", "AND")
	g.AddCondition("status", "=", "active", "AND")

	query = g.BuildQuery()
	expected := "SELECT * FROM table WHERE age > '18' AND status = 'active'"
	if query != expected {
		t.Errorf("expected %s, got %s", expected, query)
	}

	g.RemoveCondition(0)
	query = g.BuildQuery()
	if query != "SELECT * FROM table WHERE status = 'active'" {
		t.Errorf("expected status-only query, got %s", query)
	}
}

func TestRealSyntaxHighlighter(t *testing.T) {
	g := NewRealSyntaxHighlighter("syntax")
	g.SetCode([]string{
		"func main() {",
		"	fmt.Println(\"hello\")",
		"}",
	}, "go")

	g.mu.RLock()
	if len(g.Lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(g.Lines))
	}
	if g.Language != "go" {
		t.Errorf("expected go, got %s", g.Language)
	}
	g.mu.RUnlock()
}

func TestRealColorPicker(t *testing.T) {
	g := NewRealColorPicker("color")
	g.mu.RLock()
	if len(g.Palette) != 12 {
		t.Errorf("expected 12 palette colors, got %d", len(g.Palette))
	}
	if g.Selected != 0 {
		t.Errorf("expected selected 0, got %d", g.Selected)
	}
	g.mu.RUnlock()

	g.SetCurrent(hex("ff0000"))
	g.mu.RLock()
	if g.Current.R != 255 || g.Current.G != 0 || g.Current.B != 0 {
		t.Errorf("expected red, got R:%d G:%d B:%d", g.Current.R, g.Current.G, g.Current.B)
	}
	g.mu.RUnlock()
}

func TestRealCalendarView(t *testing.T) {
	g := NewRealCalendarView("cal")
	g.AddEvent(15, "Meeting")
	g.AddEvent(15, "Lunch")
	g.AddEvent(25, "Deadline")

	g.mu.RLock()
	if len(g.Events[15]) != 2 {
		t.Errorf("expected 2 events on day 15, got %d", len(g.Events[15]))
	}
	if len(g.Events[25]) != 1 {
		t.Errorf("expected 1 event on day 25, got %d", len(g.Events[25]))
	}
	g.mu.RUnlock()

	g.ClearEvents(15)
	g.mu.RLock()
	if len(g.Events[15]) != 0 {
		t.Errorf("expected 0 events on day 15 after clear, got %d", len(g.Events[15]))
	}
	g.mu.RUnlock()
}

func TestRealPieChartThreadSafety(t *testing.T) {
	g := NewRealPieChart("pie")
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(v float64) {
			defer wg.Done()
			g.SetSegments([]PieSegment{{Label: "A", Value: v}})
			g.GetTotal()
		}(float64(i * 10))
	}
	wg.Wait()
}

func TestRealScoreBoardThreadSafety(t *testing.T) {
	g := NewRealScoreBoard("scores", 10)
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			g.AddScore("Player1", n*10)
			g.Clear()
		}(i)
	}
	wg.Wait()
}

func TestRealNotificationPanelThreadSafety(t *testing.T) {
	g := NewRealNotificationPanel("notifs")
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			g.Add("Title", "Msg", "info", hex("89b4fa"))
			g.GetUnreadCount()
			g.MarkAllRead()
			g.Dismiss(0)
		}(i)
	}
	wg.Wait()
}

func TestRealChatInputThreadSafety(t *testing.T) {
	g := NewRealChatInput("chat")
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			g.AddMessage("User", "msg", hex("89b4fa"))
			g.GetMessages()
			g.ClearHistory()
		}(i)
	}
	wg.Wait()
}

func TestRealQueryBuilderThreadSafety(t *testing.T) {
	g := NewRealQueryBuilder("q", []string{"a", "b"})
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			g.AddCondition("a", "=", "1", "AND")
			g.BuildQuery()
			g.RemoveCondition(0)
		}()
	}
	wg.Wait()
}

func TestRealCalendarViewThreadSafety(t *testing.T) {
	g := NewRealCalendarView("cal")
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			g.AddEvent(n+1, "Event")
			g.ClearEvents(n + 1)
		}(i)
	}
	wg.Wait()
}

func TestRealResourceMonitorThreadSafety(t *testing.T) {
	g := NewRealResourceMonitor("res")
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			g.Set("cpu", float64(n)*10, 100, "%", hex("89b4fa"))
			g.Get("cpu")
		}(i)
	}
	wg.Wait()
}

func TestRealMiniMapThreadSafety(t *testing.T) {
	g := NewRealMiniMap("map")
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			lines := make([]string, n*10)
			g.SetContent(lines, 0, 20)
		}(i)
	}
	wg.Wait()
}

func TestRealTagCloudThreadSafety(t *testing.T) {
	g := NewRealTagCloud("tags")
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			g.AddTag("tag", n*10, hex("89b4fa"))
		}(i)
	}
	wg.Wait()
}

func TestRealSyntaxHighlighterThreadSafety(t *testing.T) {
	g := NewRealSyntaxHighlighter("syn")
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			g.SetCode([]string{"line1", "line2"}, "go")
		}()
	}
	wg.Wait()
}

func TestRealColorPickerThreadSafety(t *testing.T) {
	g := NewRealColorPicker("cp")
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			g.SetCurrent(hex("ff0000"))
		}(i)
	}
	wg.Wait()
}

func TestScoreBoardCallbacks(t *testing.T) {
	g := NewRealScoreBoard("scores", 5)
	var callbackRank int
	var callbackEntry ScoreEntry
	g.OnUpdate = func(rank int, entry ScoreEntry) {
		callbackRank = rank
		callbackEntry = entry
	}

	g.AddScore("Alice", 100)
	g.AddScore("Bob", 200)
	g.AddScore("Alice", 50)

	if callbackEntry.Name != "Alice" {
		t.Errorf("expected callback for Alice, got %s", callbackEntry.Name)
	}
	if callbackRank != 1 {
		t.Errorf("expected rank 1, got %d", callbackRank)
	}
	if callbackEntry.Score != 150 {
		t.Errorf("expected score 150, got %d", callbackEntry.Score)
	}
}

func TestTagCloudCallbacks(t *testing.T) {
	g := NewRealTagCloud("tags")
	g.SetTags([]TagEntry{
		{Name: "Go", Weight: 10, Color: hex("89b4fa")},
		{Name: "Rust", Weight: 20, Color: hex("f38ba8")},
	})

	var selected string
	g.OnSelect = func(tag string) {
		selected = tag
	}

	g.mu.Lock()
	g.Selected = 1
	g.OnSelect("Rust")
	g.mu.Unlock()

	if selected != "Rust" {
		t.Errorf("expected Rust, got %s", selected)
	}
}

func TestNotificationCallbacks(t *testing.T) {
	g := NewRealNotificationPanel("notifs")
	g.Add("Alert", "Disk full", "warning", hex("fab387"))

	var readIdx int
	g.OnRead = func(idx int) {
		readIdx = idx
	}

	g.mu.Lock()
	g.OnRead(0)
	g.mu.Unlock()

	if readIdx != 0 {
		t.Errorf("expected read index 0, got %d", readIdx)
	}
}

func TestQueryBuilderCallbacks(t *testing.T) {
	g := NewRealQueryBuilder("q", []string{"a", "b"})
	g.AddCondition("a", "=", "1", "AND")

	var called bool
	g.OnQuery = func(conds []QueryCondition) {
		called = true
	}

	g.mu.Lock()
	g.OnQuery(g.Conditions)
	g.mu.Unlock()

	if !called {
		t.Error("OnQuery callback not called")
	}
}

func TestCalendarCallbacks(t *testing.T) {
	g := NewRealCalendarView("cal")

	var selectedDay int
	g.OnDaySelect = func(day int) {
		selectedDay = day
	}

	g.mu.Lock()
	g.OnDaySelect(15)
	g.mu.Unlock()

	if selectedDay != 15 {
		t.Errorf("expected day 15, got %d", selectedDay)
	}
}

func TestColorPickerCallbacks(t *testing.T) {
	g := NewRealColorPicker("cp")

	var picked mofu.Color
	g.OnPick = func(c mofu.Color) {
		picked = c
	}

	g.mu.Lock()
	g.OnPick(hex("ff0000"))
	g.mu.Unlock()

	if picked.R != 255 {
		t.Errorf("expected R=255, got %d", picked.R)
	}
}

func TestChatInputCallbacks(t *testing.T) {
	g := NewRealChatInput("chat")

	var submitted string
	g.OnSubmit = func(text string) mofu.Cmd {
		submitted = text
		return nil
	}

	g.mu.Lock()
	g.OnSubmit("Hello world")
	g.mu.Unlock()

	if submitted != "Hello world" {
		t.Errorf("expected 'Hello world', got %s", submitted)
	}
}

func TestResourceMonitorMulti(t *testing.T) {
	g := NewRealResourceMonitor("res")
	g.Set("CPU", 50, 100, "%", hex("89b4fa"))
	g.Set("Memory", 8, 16, "GB", hex("a6e3a1"))
	g.Set("Disk", 200, 500, "GB", hex("fab387"))
	g.Set("Network", 45, 100, "Mbps", hex("f38ba8"))

	g.mu.RLock()
	n := len(g.Resources)
	g.mu.RUnlock()
	if n != 4 {
		t.Errorf("expected 4 resources, got %d", n)
	}

	val, ok := g.Get("Network")
	if !ok || val != 45 {
		t.Errorf("expected Network=45, got %f", val)
	}

	g.Set("CPU", 80, 100, "%", hex("fab387"))
	val, _ = g.Get("CPU")
	if val != 80 {
		t.Errorf("expected updated CPU=80, got %f", val)
	}
}
