package gadgets

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/xanstomper/mofu"
)

// =========================================================================
// BATCH 5: Product-Building Gadgets (replaced games/toys)
// =========================================================================

type RealPipelineRunner struct {
	Base
	Stages    []PipeStage
	Current   int
	Running   bool
	StartedAt time.Time
	mu        sync.RWMutex
	OnStage   func(idx int, stage PipeStage)
}

type PipeStage struct {
	Name    string
	Status  string
	Output  string
	Started time.Time
	Ended   time.Time
}

func NewRealPipelineRunner(id string) *RealPipelineRunner {
	return &RealPipelineRunner{Base: *NewBase(id)}
}

func (g *RealPipelineRunner) AddStage(name string) {
	g.mu.Lock()
	g.Stages = append(g.Stages, PipeStage{Name: name, Status: "pending"})
	g.mu.Unlock()
}

func (g *RealPipelineRunner) Start() {
	g.mu.Lock()
	g.Running = true
	g.Current = 0
	g.StartedAt = time.Now()
	for i := range g.Stages {
		g.Stages[i].Status = "pending"
		g.Stages[i].Output = ""
	}
	g.mu.Unlock()
}

func (g *RealPipelineRunner) AdvanceStage(output string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.Current < len(g.Stages) {
		g.Stages[g.Current].Status = "done"
		g.Stages[g.Current].Output = output
		g.Stages[g.Current].Ended = time.Now()
		g.Current++
		if g.Current < len(g.Stages) {
			g.Stages[g.Current].Status = "running"
			g.Stages[g.Current].Started = time.Now()
		} else {
			g.Running = false
		}
	}
}

func (g *RealPipelineRunner) FailStage(output string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.Current < len(g.Stages) {
		g.Stages[g.Current].Status = "failed"
		g.Stages[g.Current].Output = output
		g.Stages[g.Current].Ended = time.Now()
		g.Running = false
	}
}

func (g *RealPipelineRunner) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	doneCount := 0
	for _, s := range g.Stages {
		if s.Status == "done" {
			doneCount++
		}
	}
	ctx.Renderer.WriteString(fmt.Sprintf(" Pipeline %d/%d", doneCount, len(g.Stages)), r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	for _, stage := range g.Stages {
		if y >= r.Y+r.Height-2 {
			break
		}

		icon := "○"
		color := mofu.Hex("585b70")
		switch stage.Status {
		case "done":
			icon = "✓"
			color = mofu.Hex("a6e3a1")
		case "running":
			icon = "●"
			color = mofu.Hex("f9e2af")
		case "failed":
			icon = "✗"
			color = mofu.Hex("f38ba8")
		}

		elapsed := ""
		if !stage.Started.IsZero() {
			end := stage.Ended
			if end.IsZero() {
				end = time.Now()
			}
			elapsed = fmt.Sprintf(" (%s)", end.Sub(stage.Started).Round(time.Millisecond))
		}

		line := fmt.Sprintf(" %s %s%s", icon, stage.Name, elapsed)
		if len(line) > r.Width-2 {
			line = line[:r.Width-2]
		}
		ctx.Renderer.WriteString(line, r.X, y, color, mofu.ColorBlack, 0)
		y++

		if stage.Output != "" && y < r.Y+r.Height-1 {
			output := "   " + stage.Output
			if len(output) > r.Width-2 {
				output = output[:r.Width-5] + "..."
			}
			ctx.Renderer.WriteString(output, r.X, y, mofu.Hex("6c7086"), mofu.ColorBlack, 0)
			y++
		}
	}

	if !g.StartedAt.IsZero() && g.Running {
		elapsed := time.Since(g.StartedAt).Round(time.Second)
		ctx.Renderer.WriteString(fmt.Sprintf(" Running for %s", elapsed), r.X, r.Y+r.Height-1, mofu.Hex("f9e2af"), mofu.ColorBlack, 0)
	}
}

func (g *RealPipelineRunner) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)
	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()
	case len(ke.Runes) > 0 && ke.Runes[0] == 's':
		g.Start()
		if len(g.Stages) > 0 {
			g.mu.Lock()
			g.Stages[0].Status = "running"
			g.Stages[0].Started = time.Now()
			g.mu.Unlock()
		}
	case len(ke.Runes) > 0 && ke.Runes[0] == 'n':
		g.AdvanceStage("done")
	}
	return nil
}

type RealDBSchema struct {
	Base
	Tables   []DBTable
	Selected int
	mu       sync.RWMutex
}

type DBTable struct {
	Name    string
	Columns []DBColumn
}

type DBColumn struct {
	Name     string
	Type     string
	Nullable bool
	Primary  bool
}

func NewRealDBSchema(id string) *RealDBSchema {
	return &RealDBSchema{
		Base: *NewBase(id),
		Tables: []DBTable{
			{Name: "users", Columns: []DBColumn{
				{Name: "id", Type: "serial", Primary: true},
				{Name: "email", Type: "varchar(255)", Nullable: false},
				{Name: "name", Type: "varchar(100)"},
				{Name: "created_at", Type: "timestamp"},
			}},
			{Name: "posts", Columns: []DBColumn{
				{Name: "id", Type: "serial", Primary: true},
				{Name: "user_id", Type: "integer", Nullable: false},
				{Name: "title", Type: "varchar(255)"},
				{Name: "body", Type: "text"},
				{Name: "published", Type: "boolean", Nullable: false},
			}},
			{Name: "comments", Columns: []DBColumn{
				{Name: "id", Type: "serial", Primary: true},
				{Name: "post_id", Type: "integer", Nullable: false},
				{Name: "user_id", Type: "integer", Nullable: false},
				{Name: "body", Type: "text"},
			}},
		},
	}
}

func (g *RealDBSchema) AddTable(t DBTable) {
	g.mu.Lock()
	g.Tables = append(g.Tables, t)
	g.mu.Unlock()
}

func (g *RealDBSchema) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(fmt.Sprintf(" DB Schema (%d tables)", len(g.Tables)), r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	for i, table := range g.Tables {
		if y >= r.Y+r.Height-3 {
			break
		}

		style := mofu.DefaultStyle()
		if i == g.Selected {
			ctx.Renderer.WriteString(strings.Repeat(" ", r.Width-2), r.X+1, y, mofu.ColorWhite, mofu.Hex("2a2a3e"), 0)
			style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))
		}

		ctx.Renderer.WriteString(fmt.Sprintf(" %s (%d cols)", table.Name, len(table.Columns)), r.X, y, style.Foreground, style.Background, style.Attrs)
		y++

		if i == g.Selected {
			for _, col := range table.Columns {
				if y >= r.Y+r.Height-2 {
					break
				}
				attrs := ""
				if col.Primary {
					attrs = " PK"
				}
				if col.Nullable {
					attrs += " NULL"
				}
				ctx.Renderer.WriteString(fmt.Sprintf("   %-20s %-20s%s", col.Name, col.Type, attrs), r.X, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
				y++
			}
		}
	}
}

func (g *RealDBSchema) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	g.mu.Lock()
	defer g.mu.Unlock()

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()
	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		if g.Selected < len(g.Tables)-1 {
			g.Selected++
		}
	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if g.Selected > 0 {
			g.Selected--
		}
	}
	return nil
}

type RealFeatureFlags struct {
	Base
	Flags     []FeatureFlag
	Selected  int
	mu        sync.RWMutex
	OnToggle  func(key string, enabled bool)
}

type FeatureFlag struct {
	Key         string
	Enabled     bool
	Description string
	Env         string
	UpdatedAt   time.Time
}

func NewRealFeatureFlags(id string) *RealFeatureFlags {
	return &RealFeatureFlags{
		Base: *NewBase(id),
		Flags: []FeatureFlag{
			{Key: "dark_mode", Enabled: true, Description: "Enable dark mode UI", Env: "all", UpdatedAt: time.Now().Add(-24 * time.Hour)},
			{Key: "new_checkout", Enabled: false, Description: "New checkout flow", Env: "staging", UpdatedAt: time.Now().Add(-2 * time.Hour)},
			{Key: "ai_suggestions", Enabled: true, Description: "AI-powered suggestions", Env: "production", UpdatedAt: time.Now().Add(-1 * time.Hour)},
			{Key: "beta_api", Enabled: false, Description: "Beta API v2 endpoints", Env: "development", UpdatedAt: time.Now().Add(-48 * time.Hour)},
			{Key: "experimental_cache", Enabled: true, Description: "New caching layer", Env: "production", UpdatedAt: time.Now().Add(-12 * time.Hour)},
		},
	}
}

func (g *RealFeatureFlags) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	enabled := 0
	for _, f := range g.Flags {
		if f.Enabled {
			enabled++
		}
	}
	ctx.Renderer.WriteString(fmt.Sprintf(" Feature Flags (%d/%d enabled)", enabled, len(g.Flags)), r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	for i, flag := range g.Flags {
		if y >= r.Y+r.Height-2 {
			break
		}

		icon := "○"
		color := mofu.Hex("585b70")
		if flag.Enabled {
			icon = "●"
			color = mofu.Hex("a6e3a1")
		}

		if i == g.Selected {
			ctx.Renderer.WriteString(strings.Repeat(" ", r.Width-2), r.X+1, y, mofu.ColorWhite, mofu.Hex("2a2a3e"), 0)
			color = mofu.Hex("ff69b4")
		}

		key := flag.Key
		if len(key) > 20 {
			key = key[:17] + "..."
		}
		desc := flag.Description
		if len(desc) > r.Width-35 {
			desc = desc[:r.Width-38] + "..."
		}

		line := fmt.Sprintf(" %s %-20s %-30s [%s]", icon, key, desc, flag.Env)
		if len(line) > r.Width-2 {
			line = line[:r.Width-2]
		}
		ctx.Renderer.WriteString(line, r.X, y, color, mofu.ColorBlack, 0)
		y++
	}
}

func (g *RealFeatureFlags) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	g.mu.Lock()
	defer g.mu.Unlock()

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()
	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		if g.Selected < len(g.Flags)-1 {
			g.Selected++
		}
	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if g.Selected > 0 {
			g.Selected--
		}
	case ke.Key == mofu.KeySpace:
		if g.Selected < len(g.Flags) {
			g.Flags[g.Selected].Enabled = !g.Flags[g.Selected].Enabled
			g.Flags[g.Selected].UpdatedAt = time.Now()
			if g.OnToggle != nil {
				g.OnToggle(g.Flags[g.Selected].Key, g.Flags[g.Selected].Enabled)
			}
		}
	}
	return nil
}

type RealIncidentTracker struct {
	Base
	Incidents []Incident
	Selected  int
	mu        sync.RWMutex
	OnResolve func(id string)
}

type Incident struct {
	ID       string
	Title    string
	Severity string
	Status   string
	Service  string
	Created  time.Time
}

func NewRealIncidentTracker(id string) *RealIncidentTracker {
	now := time.Now()
	return &RealIncidentTracker{
		Base: *NewBase(id),
		Incidents: []Incident{
			{ID: "INC-001", Title: "Database connection pool exhausted", Severity: "critical", Status: "investigating", Service: "api-gateway", Created: now.Add(-30 * time.Minute)},
			{ID: "INC-002", Title: "Elevated error rate on /checkout", Severity: "high", Status: "identified", Service: "payment-service", Created: now.Add(-2 * time.Hour)},
			{ID: "INC-003", Title: "Slow query performance", Severity: "medium", Status: "monitoring", Service: "user-service", Created: now.Add(-6 * time.Hour)},
			{ID: "INC-004", Title: "Certificate expiring in 7 days", Severity: "low", Status: "resolved", Service: "cdn", Created: now.Add(-24 * time.Hour)},
		},
	}
}

func (g *RealIncidentTracker) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	open := 0
	for _, inc := range g.Incidents {
		if inc.Status != "resolved" {
			open++
		}
	}
	ctx.Renderer.WriteString(fmt.Sprintf(" Incidents (%d open)", open), r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	for i, inc := range g.Incidents {
		if y >= r.Y+r.Height-2 {
			break
		}

		severityColor := mofu.Hex("585b70")
		switch inc.Severity {
		case "critical":
			severityColor = mofu.Hex("f38ba8")
		case "high":
			severityColor = mofu.Hex("fab387")
		case "medium":
			severityColor = mofu.Hex("f9e2af")
		case "low":
			severityColor = mofu.Hex("89b4fa")
		}

		if i == g.Selected {
			ctx.Renderer.WriteString(strings.Repeat(" ", r.Width-2), r.X+1, y, mofu.ColorWhite, mofu.Hex("2a2a3e"), 0)
		}

		title := inc.Title
		if len(title) > r.Width-30 {
			title = title[:r.Width-33] + "..."
		}

		line := fmt.Sprintf(" %-8s [%-8s] %-10s %s", inc.ID, inc.Severity, inc.Status, title)
		if len(line) > r.Width-2 {
			line = line[:r.Width-2]
		}
		ctx.Renderer.WriteString(line, r.X, y, severityColor, mofu.ColorBlack, 0)
		y++
	}
}

func (g *RealIncidentTracker) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	g.mu.Lock()
	defer g.mu.Unlock()

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()
	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		if g.Selected < len(g.Incidents)-1 {
			g.Selected++
		}
	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if g.Selected > 0 {
			g.Selected--
		}
	case len(ke.Runes) > 0 && ke.Runes[0] == 'r':
		if g.Selected < len(g.Incidents) {
			g.Incidents[g.Selected].Status = "resolved"
		}
	}
	return nil
}

type RealDeploymentTracker struct {
	Base
	Deploys   []Deploy
	Selected  int
	mu        sync.RWMutex
}

type Deploy struct {
	Version   string
	Service   string
	Status    string
	Env       string
	Commit    string
	Author    string
	StartedAt time.Time
}

func NewRealDeploymentTracker(id string) *RealDeploymentTracker {
	now := time.Now()
	return &RealDeploymentTracker{
		Base: *NewBase(id),
		Deploys: []Deploy{
			{Version: "v1.2.3", Service: "api-gateway", Status: "live", Env: "production", Commit: "a1b2c3d", Author: "alice", StartedAt: now.Add(-1 * time.Hour)},
			{Version: "v1.2.4", Service: "api-gateway", Status: "deploying", Env: "staging", Commit: "e4f5g6h", Author: "bob", StartedAt: now.Add(-5 * time.Minute)},
			{Version: "v2.0.0", Service: "payment-service", Status: "pending", Env: "staging", Commit: "i7j8k9l", Author: "charlie", StartedAt: now.Add(-2 * time.Hour)},
			{Version: "v1.1.0", Service: "user-service", Status: "live", Env: "production", Commit: "m0n1o2p", Author: "alice", StartedAt: now.Add(-48 * time.Hour)},
		},
	}
}

func (g *RealDeploymentTracker) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(" Deployments", r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	for i, d := range g.Deploys {
		if y >= r.Y+r.Height-2 {
			break
		}

		statusIcon := "○"
		statusColor := mofu.Hex("585b70")
		switch d.Status {
		case "live":
			statusIcon = "✓"
			statusColor = mofu.Hex("a6e3a1")
		case "deploying":
			statusIcon = "●"
			statusColor = mofu.Hex("f9e2af")
		case "failed":
			statusIcon = "✗"
			statusColor = mofu.Hex("f38ba8")
		case "pending":
			statusIcon = "○"
			statusColor = mofu.Hex("89b4fa")
		}

		if i == g.Selected {
			ctx.Renderer.WriteString(strings.Repeat(" ", r.Width-2), r.X+1, y, mofu.ColorWhite, mofu.Hex("2a2a3e"), 0)
		}

		line := fmt.Sprintf(" %s %-10s %-16s %-12s %-10s %s", statusIcon, d.Version, d.Service, d.Env, d.Commit, d.Author)
		if len(line) > r.Width-2 {
			line = line[:r.Width-2]
		}
		ctx.Renderer.WriteString(line, r.X, y, statusColor, mofu.ColorBlack, 0)
		y++
	}
}

func (g *RealDeploymentTracker) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	g.mu.Lock()
	defer g.mu.Unlock()

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()
	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		if g.Selected < len(g.Deploys)-1 {
			g.Selected++
		}
	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if g.Selected > 0 {
			g.Selected--
		}
	}
	return nil
}

type RealAuditLog struct {
	Base
	Entries []AuditEntry
	Filter  string
	mu      sync.RWMutex
}

type AuditEntry struct {
	Timestamp time.Time
	User      string
	Action    string
	Resource  string
	Details   string
}

func NewRealAuditLog(id string) *RealAuditLog {
	now := time.Now()
	return &RealAuditLog{
		Base: *NewBase(id),
		Entries: []AuditEntry{
			{Timestamp: now.Add(-5 * time.Minute), User: "alice", Action: "deploy", Resource: "api-gateway", Details: "v1.2.3 → production"},
			{Timestamp: now.Add(-15 * time.Minute), User: "bob", Action: "update", Resource: "feature-flag:dark_mode", Details: "enabled=true"},
			{Timestamp: now.Add(-30 * time.Minute), User: "charlie", Action: "create", Resource: "incident:INC-001", Details: "database pool exhausted"},
			{Timestamp: now.Add(-1 * time.Hour), User: "alice", Action: "rotate", Resource: "secret:db-password", Details: "rotation completed"},
			{Timestamp: now.Add(-2 * time.Hour), User: "bob", Action: "delete", Resource: "user:test-account", Details: "cleanup test data"},
		},
	}
}

func (g *RealAuditLog) AddEntry(e AuditEntry) {
	g.mu.Lock()
	g.Entries = append([]AuditEntry{e}, g.Entries...)
	g.mu.Unlock()
}

func (g *RealAuditLog) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(fmt.Sprintf(" Audit Log (%d entries)", len(g.Entries)), r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	for _, entry := range g.Entries {
		if y >= r.Y+r.Height-2 {
			break
		}

		ts := entry.Timestamp.Format("15:04:05")

		actionColor := mofu.Hex("cdd6f4")
		switch entry.Action {
		case "create", "deploy":
			actionColor = mofu.Hex("a6e3a1")
		case "update":
			actionColor = mofu.Hex("f9e2af")
		case "delete":
			actionColor = mofu.Hex("f38ba8")
		case "rotate":
			actionColor = mofu.Hex("89b4fa")
		}

		line := fmt.Sprintf(" %s %-10s %-8s %-25s", ts, entry.User, entry.Action, entry.Resource)
		if len(line) > r.Width-2 {
			line = line[:r.Width-2]
		}
		ctx.Renderer.WriteString(line, r.X, y, actionColor, mofu.ColorBlack, 0)
		y++
	}
}

func (g *RealAuditLog) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)
	if ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q') {
		return mofu.QuitCmd()
	}
	return nil
}

type RealServiceHealth struct {
	Base
	Services []ServiceStatus
	mu       sync.RWMutex
}

type ServiceStatus struct {
	Name      string
	Status    string
	Uptime    string
	Latency   string
	ErrorRate string
}

func NewRealServiceHealth(id string) *RealServiceHealth {
	return &RealServiceHealth{
		Base: *NewBase(id),
		Services: []ServiceStatus{
			{Name: "api-gateway", Status: "healthy", Uptime: "99.98%", Latency: "12ms", ErrorRate: "0.02%"},
			{Name: "payment-service", Status: "degraded", Uptime: "99.50%", Latency: "245ms", ErrorRate: "2.30%"},
			{Name: "user-service", Status: "healthy", Uptime: "99.99%", Latency: "8ms", ErrorRate: "0.01%"},
			{Name: "notification-service", Status: "healthy", Uptime: "99.95%", Latency: "45ms", ErrorRate: "0.05%"},
			{Name: "search-service", Status: "unhealthy", Uptime: "95.00%", Latency: "timeout", ErrorRate: "15.00%"},
		},
	}
}

func (g *RealServiceHealth) SetStatus(name, status string) {
	g.mu.Lock()
	for i, s := range g.Services {
		if s.Name == name {
			g.Services[i].Status = status
			break
		}
	}
	g.mu.Unlock()
}

func (g *RealServiceHealth) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	healthy := 0
	for _, s := range g.Services {
		if s.Status == "healthy" {
			healthy++
		}
	}
	titleColor := mofu.Hex("a6e3a1")
	if healthy < len(g.Services) {
		titleColor = mofu.Hex("fab387")
	}
	if healthy == 0 {
		titleColor = mofu.Hex("f38ba8")
	}

	ctx.Renderer.WriteString(fmt.Sprintf(" Service Health %d/%d", healthy, len(g.Services)), r.X, y, titleColor, mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	header := fmt.Sprintf(" %-24s %-10s %-8s %-10s %-8s", "Service", "Status", "Uptime", "Latency", "Errors")
	ctx.Renderer.WriteString(header, r.X, y, mofu.Hex("89b4fa"), mofu.ColorBlack, mofu.AttrBold)
	y++

	for _, svc := range g.Services {
		if y >= r.Y+r.Height-1 {
			break
		}

		statusIcon := "●"
		color := mofu.Hex("a6e3a1")
		switch svc.Status {
		case "healthy":
			statusIcon = "✓"
		case "degraded":
			statusIcon = "⚠"
			color = mofu.Hex("fab387")
		case "unhealthy":
			statusIcon = "✗"
			color = mofu.Hex("f38ba8")
		}

		line := fmt.Sprintf(" %-24s %s %-7s %-8s %-10s %-8s", svc.Name, statusIcon, svc.Status, svc.Uptime, svc.Latency, svc.ErrorRate)
		if len(line) > r.Width-2 {
			line = line[:r.Width-2]
		}
		ctx.Renderer.WriteString(line, r.X, y, color, mofu.ColorBlack, 0)
		y++
	}
}

func (g *RealServiceHealth) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)
	if ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q') {
		return mofu.QuitCmd()
	}
	return nil
}

type RealLogAggregator struct {
	Base
	Sources  []LogSource
	Filter   string
	Level    string
	mu       sync.RWMutex
}

type LogSource struct {
	Name   string
	Color  mofu.Color
	Count  int
}

type LogEntry struct {
	Source    string
	Level     string
	Message   string
	Timestamp time.Time
}

func NewRealLogAggregator(id string) *RealLogAggregator {
	return &RealLogAggregator{
		Base: *NewBase(id),
		Sources: []LogSource{
			{Name: "api-gateway", Color: mofu.Hex("89b4fa")},
			{Name: "payment-svc", Color: mofu.Hex("a6e3a1")},
			{Name: "user-svc", Color: mofu.Hex("f9e2af")},
			{Name: "worker", Color: mofu.Hex("cba6f7")},
		},
	}
}

func (g *RealLogAggregator) SetSourceCount(name string, count int) {
	g.mu.Lock()
	for i, s := range g.Sources {
		if s.Name == name {
			g.Sources[i].Count = count
			break
		}
	}
	g.mu.Unlock()
}

func (g *RealLogAggregator) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	total := 0
	for _, s := range g.Sources {
		total += s.Count
	}

	ctx.Renderer.WriteString(fmt.Sprintf(" Log Aggregator (%d sources, %d entries)", len(g.Sources), total), r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	for _, src := range g.Sources {
		if y >= r.Y+r.Height-2 {
			break
		}

		barW := r.Width - 35
		filled := 0
		if total > 0 {
			filled = src.Count * barW / total
		}
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barW-filled)

		ctx.Renderer.WriteString(fmt.Sprintf(" %-16s %s %d", src.Name, bar, src.Count), r.X, y, src.Color, mofu.ColorBlack, 0)
		y++
	}
}

func (g *RealLogAggregator) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)
	if ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q') {
		return mofu.QuitCmd()
	}
	return nil
}
