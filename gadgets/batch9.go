package gadgets

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/xanstomper/mofu"
)

// =========================================================================
// BATCH 9: Dev Tools & System Gadgets (10 gadgets)
// =========================================================================

type RealAPIClient struct {
	Base
	BaseURL    string
	Headers    map[string]string
	History    []APIRequest
	Response   *APIResponse
	Selected   int
	mu         sync.RWMutex
	OnRequest  func(method, url string) *APIResponse
}

type APIRequest struct {
	Method   string
	URL      string
	Status   int
	Time     time.Duration
	Size     int
}

type APIResponse struct {
	StatusCode int
	Body       string
	Headers    map[string]string
	Time       time.Duration
}

func NewRealAPIClient(id, baseURL string) *RealAPIClient {
	return &RealAPIClient{
		Base:    *NewBase(id),
		BaseURL: baseURL,
		Headers: make(map[string]string),
	}
}

func (g *RealAPIClient) SetHeader(key, value string) {
	g.mu.Lock()
	g.Headers[key] = value
	g.mu.Unlock()
}

func (g *RealAPIClient) Request(method, path string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	start := time.Now()
	url := g.BaseURL + path

	var resp *APIResponse
	if g.OnRequest != nil {
		resp = g.OnRequest(method, url)
	}

	elapsed := time.Since(start)

	req := APIRequest{
		Method: method,
		URL:    url,
		Time:   elapsed,
	}
	if resp != nil {
		req.Status = resp.StatusCode
		req.Size = len(resp.Body)
		g.Response = resp
	}

	g.History = append(g.History, req)
	if len(g.History) > 50 {
		g.History = g.History[len(g.History)-50:]
	}
}

func (g *RealAPIClient) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(fmt.Sprintf(" API Client — %s", g.BaseURL), r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	if len(g.History) > 0 {
		ctx.Renderer.WriteString(" History:", r.X, y, mofu.Hex("89b4fa"), mofu.ColorBlack, mofu.AttrBold)
		y++

		start := 0
		if len(g.History) > r.Height/2-3 {
			start = len(g.History) - r.Height/2 + 3
		}

		for i := start; i < len(g.History); i++ {
			if y >= r.Y+r.Height/2 {
				break
			}
			req := g.History[i]
			statusColor := mofu.Hex("a6e3a1")
			if req.Status >= 400 {
				statusColor = mofu.Hex("f38ba8")
			} else if req.Status >= 300 {
				statusColor = mofu.Hex("fab387")
			}

			line := fmt.Sprintf("  %s %s %d %s", req.Method, req.URL, req.Status, req.Time.Round(time.Millisecond))
			if len(line) > r.Width-2 {
				line = line[:r.Width-5] + "..."
			}
			ctx.Renderer.WriteString(line, r.X, y, statusColor, mofu.ColorBlack, 0)
			y++
		}
	}

	if g.Response != nil {
		y++
		ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
		y++

		ctx.Renderer.WriteString(fmt.Sprintf(" Response: %d (%s)", g.Response.StatusCode, g.Response.Time.Round(time.Millisecond)), r.X, y, mofu.Hex("89b4fa"), mofu.ColorBlack, 0)
		y++

		bodyLines := strings.Split(g.Response.Body, "\n")
		for _, line := range bodyLines {
			if y >= r.Y+r.Height-1 {
				break
			}
			if len(line) > r.Width-2 {
				line = line[:r.Width-5] + "..."
			}
			ctx.Renderer.WriteString(line, r.X, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
			y++
		}
	}
}

func (g *RealAPIClient) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)
	if ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q') {
		return mofu.QuitCmd()
	}
	return nil
}

type RealProcessViewer struct {
	Base
	Processes []ProcEntry
	Selected  int
	SortBy    int
	Filter    string
	mu        sync.RWMutex
	OnKill    func(pid int)
}

type ProcEntry struct {
	PID    int
	Name   string
	CPU    float64
	Memory float64
	Status string
	User   string
}

func NewRealProcessViewer(id string) *RealProcessViewer {
	return &RealProcessViewer{
		Base: *NewBase(id),
		Processes: []ProcEntry{
			{PID: 1, Name: "systemd", CPU: 0.1, Memory: 2.3, Status: "S", User: "root"},
			{PID: 1234, Name: "node", CPU: 15.2, Memory: 8.5, Status: "S", User: "ben"},
			{PID: 2345, Name: "go", CPU: 45.8, Memory: 12.1, Status: "R", User: "ben"},
			{PID: 3456, Name: "docker", CPU: 5.3, Memory: 25.6, Status: "S", User: "root"},
			{PID: 4567, Name: "postgres", CPU: 2.1, Memory: 15.3, Status: "S", User: "postgres"},
			{PID: 5678, Name: "nginx", CPU: 0.5, Memory: 3.2, Status: "S", User: "www-data"},
			{PID: 6789, Name: "redis-server", CPU: 1.2, Memory: 4.8, Status: "S", User: "redis"},
		},
	}
}

func (g *RealProcessViewer) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(fmt.Sprintf(" Processes (%d)", len(g.Processes)), r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	header := fmt.Sprintf(" %-8s %-16s %7s %7s %4s %-10s", "PID", "NAME", "CPU%", "MEM%", "ST", "USER")
	ctx.Renderer.WriteString(header, r.X, y, mofu.Hex("89b4fa"), mofu.ColorBlack, mofu.AttrBold)
	y++

	for i, proc := range g.Processes {
		if y >= r.Y+r.Height-1 {
			break
		}

		style := mofu.DefaultStyle()
		if i == g.Selected {
			ctx.Renderer.WriteString(strings.Repeat(" ", r.Width-2), r.X+1, y, mofu.ColorWhite, mofu.Hex("2a2a3e"), 0)
			style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))
		}

		line := fmt.Sprintf(" %-8d %-16s %6.1f%% %6.1f%% %4s %-10s",
			proc.PID, proc.Name, proc.CPU, proc.Memory, proc.Status, proc.User)
		if len(line) > r.Width-2 {
			line = line[:r.Width-2]
		}

		color := style.Foreground
		if proc.CPU > 30 {
			color = mofu.Hex("f38ba8")
		} else if proc.CPU > 10 {
			color = mofu.Hex("fab387")
		}

		ctx.Renderer.WriteString(line, r.X, y, color, mofu.ColorBlack, style.Attrs)
		y++
	}
}

func (g *RealProcessViewer) HandleEvent(e mofu.Event) mofu.Cmd {
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
		if g.Selected < len(g.Processes)-1 {
			g.Selected++
		}
	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if g.Selected > 0 {
			g.Selected--
		}
	case (len(ke.Runes) > 0 && ke.Runes[0] == 'k') && ke.Shift:
		if g.Selected < len(g.Processes) && g.OnKill != nil {
			g.OnKill(g.Processes[g.Selected].PID)
			g.Processes = append(g.Processes[:g.Selected], g.Processes[g.Selected+1:]...)
			if g.Selected >= len(g.Processes) && g.Selected > 0 {
				g.Selected--
			}
		}
	}
	return nil
}

type RealPortScanner struct {
	Base
	Host      string
	Ports     []PortResult
	Scanning  bool
	mu        sync.RWMutex
}

type PortResult struct {
	Port   int
	Status string
	Service string
}

func NewRealPortScanner(id, host string) *RealPortScanner {
	return &RealPortScanner{Base: *NewBase(id), Host: host}
}

func (g *RealPortScanner) Scan(start, end int) {
	g.mu.Lock()
	g.Scanning = true
	g.Ports = nil
	g.mu.Unlock()

	services := map[int]string{
		22: "ssh", 80: "http", 443: "https", 3306: "mysql",
		5432: "postgres", 6379: "redis", 8080: "http-alt", 27017: "mongodb",
	}

	for port := start; port <= end; port++ {
		open := rand.Float64() > 0.7
		status := "closed"
		if open {
			status = "open"
		}
		svc := services[port]
		if svc == "" {
			svc = "unknown"
		}

		g.mu.Lock()
		g.Ports = append(g.Ports, PortResult{Port: port, Status: status, Service: svc})
		g.mu.Unlock()
	}

	g.mu.Lock()
	g.Scanning = false
	g.mu.Unlock()
}

func (g *RealPortScanner) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(fmt.Sprintf(" Port Scanner — %s", g.Host), r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	if g.Scanning {
		ctx.Renderer.WriteString(" Scanning...", r.X, y, mofu.Hex("f9e2af"), mofu.ColorBlack, 0)
		y++
	}

	for _, port := range g.Ports {
		if y >= r.Y+r.Height-2 {
			break
		}

		color := mofu.Hex("585b70")
		if port.Status == "open" {
			color = mofu.Hex("a6e3a1")
		}

		line := fmt.Sprintf("  %-8d %-8s %s", port.Port, port.Status, port.Service)
		if len(line) > r.Width-2 {
			line = line[:r.Width-2]
		}
		ctx.Renderer.WriteString(line, r.X, y, color, mofu.ColorBlack, 0)
		y++
	}
}

func (g *RealPortScanner) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()
	case len(ke.Runes) > 0 && ke.Runes[0] == 's':
		go g.Scan(1, 1024)
	}
	return nil
}

type RealGitBranches struct {
	Base
	Branches  []GitBranch
	Selected  int
	mu        sync.RWMutex
	OnCheckout func(branch string)
}

type GitBranch struct {
	Name   string
	Active bool
	Behind int
	Ahead  int
}

func NewRealGitBranches(id string) *RealGitBranches {
	return &RealGitBranches{
		Base: *NewBase(id),
		Branches: []GitBranch{
			{Name: "main", Active: true},
			{Name: "feature/auth", Behind: 2, Ahead: 5},
			{Name: "feature/api", Behind: 0, Ahead: 3},
			{Name: "fix/memory-leak", Behind: 1, Ahead: 1},
			{Name: "release/v1.0", Behind: 0, Ahead: 0},
		},
	}
}

func (g *RealGitBranches) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(" Git Branches", r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	for i, branch := range g.Branches {
		if y >= r.Y+r.Height-1 {
			break
		}

		icon := "  "
		if branch.Active {
			icon = "▸ "
		}

		name := branch.Name
		if len(name) > r.Width-20 {
			name = name[:r.Width-23] + "..."
		}

		sync := ""
		if branch.Behind > 0 || branch.Ahead > 0 {
			sync = fmt.Sprintf(" ↓%d ↑%d", branch.Behind, branch.Ahead)
		}

		style := mofu.DefaultStyle()
		if i == g.Selected {
			ctx.Renderer.WriteString(strings.Repeat(" ", r.Width-2), r.X+1, y, mofu.ColorWhite, mofu.Hex("2a2a3e"), 0)
			style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))
		} else if branch.Active {
			style = mofu.DefaultStyle().Fg(mofu.Hex("a6e3a1")).WithAttrs(mofu.AttrBold)
		}

		ctx.Renderer.WriteString(icon+name+sync, r.X, y, style.Foreground, style.Background, style.Attrs)
		y++
	}
}

func (g *RealGitBranches) HandleEvent(e mofu.Event) mofu.Cmd {
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
		if g.Selected < len(g.Branches)-1 {
			g.Selected++
		}
	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if g.Selected > 0 {
			g.Selected--
		}
	case ke.Key == mofu.KeyEnter:
		for i := range g.Branches {
			g.Branches[i].Active = false
		}
		if g.Selected < len(g.Branches) {
			g.Branches[g.Selected].Active = true
			if g.OnCheckout != nil {
				g.OnCheckout(g.Branches[g.Selected].Name)
			}
		}
	}
	return nil
}

type RealFileExplorer struct {
	Base
	Entries   []FileExplorerEntry
	Selected  int
	ShowHidden bool
	CurrentPath string
	mu        sync.RWMutex
	OnOpen    func(path string)
}

type FileExplorerEntry struct {
	Name     string
	Size     int64
	IsDir    bool
	Modified time.Time
	Mode     string
}

func NewRealFileExplorer(id, path string) *RealFileExplorer {
	return &RealFileExplorer{
		Base:        *NewBase(id),
		CurrentPath: path,
		Entries: []FileExplorerEntry{
			{Name: "..", IsDir: true},
			{Name: "src", IsDir: true, Modified: time.Now().Add(-2 * time.Hour)},
			{Name: "cmd", IsDir: true, Modified: time.Now().Add(-24 * time.Hour)},
			{Name: "gadgets", IsDir: true, Modified: time.Now().Add(-1 * time.Hour)},
			{Name: "README.md", Size: 15234, Modified: time.Now().Add(-3 * time.Hour), Mode: "-rw-r--r--"},
			{Name: "go.mod", Size: 523, Modified: time.Now().Add(-48 * time.Hour), Mode: "-rw-r--r--"},
			{Name: "go.sum", Size: 12345, Modified: time.Now().Add(-48 * time.Hour), Mode: "-rw-r--r--"},
			{Name: "main.go", Size: 3456, Modified: time.Now().Add(-30 * time.Minute), Mode: "-rwxr-xr-x"},
		},
	}
}

func (g *RealFileExplorer) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(fmt.Sprintf(" %s", g.CurrentPath), r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	header := fmt.Sprintf(" %-20s %10s  %-12s", "Name", "Size", "Modified")
	ctx.Renderer.WriteString(header, r.X, y, mofu.Hex("89b4fa"), mofu.ColorBlack, mofu.AttrBold)
	y++

	for i, entry := range g.Entries {
		if y >= r.Y+r.Height-1 {
			break
		}

		if !g.ShowHidden && strings.HasPrefix(entry.Name, ".") && entry.Name != ".." {
			continue
		}

		icon := "  "
		if entry.IsDir {
			icon = "📁"
		} else {
			icon = "  "
		}

		name := entry.Name
		if len(name) > 20 {
			name = name[:17] + "..."
		}

		sizeStr := ""
		if entry.IsDir {
			sizeStr = "<DIR>"
		} else {
			sizeStr = formatBytesHuman(entry.Size)
		}

		dateStr := entry.Modified.Format("Jan 02 15:04")

		line := fmt.Sprintf("%s %-18s %10s  %-12s", icon, name, sizeStr, dateStr)
		if len(line) > r.Width-2 {
			line = line[:r.Width-2]
		}

		style := mofu.DefaultStyle()
		if i == g.Selected {
			ctx.Renderer.WriteString(strings.Repeat(" ", r.Width-2), r.X+1, y, mofu.ColorWhite, mofu.Hex("2a2a3e"), 0)
			style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))
		} else if entry.IsDir {
			style = mofu.DefaultStyle().Fg(mofu.Hex("89b4fa"))
		}

		ctx.Renderer.WriteString(line, r.X, y, style.Foreground, style.Background, style.Attrs)
		y++
	}
}

func (g *RealFileExplorer) HandleEvent(e mofu.Event) mofu.Cmd {
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
		if g.Selected < len(g.Entries)-1 {
			g.Selected++
		}
	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if g.Selected > 0 {
			g.Selected--
		}
	case ke.Key == mofu.KeyEnter && g.OnOpen != nil:
		if g.Selected < len(g.Entries) {
			g.OnOpen(g.Entries[g.Selected].Name)
		}
	case len(ke.Runes) > 0 && ke.Runes[0] == 'h':
		g.ShowHidden = !g.ShowHidden
	}
	return nil
}

type RealDockerContainers struct {
	Base
	Containers []DockerContainer
	Selected   int
	mu         sync.RWMutex
	OnAction   func(id, action string)
}

type DockerContainer struct {
	ID     string
	Name   string
	Image  string
	Status string
	State  string
	Ports  string
}

func NewRealDockerContainers(id string) *RealDockerContainers {
	return &RealDockerContainers{
		Base: *NewBase(id),
		Containers: []DockerContainer{
			{ID: "a1b2c3", Name: "nginx-proxy", Image: "nginx:1.25", Status: "Up 3 days", State: "running", Ports: "80:80, 443:443"},
			{ID: "d4e5f6", Name: "postgres-db", Image: "postgres:16", Status: "Up 5 days", State: "running", Ports: "5432:5432"},
			{ID: "g7h8i9", Name: "redis-cache", Image: "redis:7", Status: "Up 2 days", State: "running", Ports: "6379:6379"},
			{ID: "j1k2l3", Name: "api-server", Image: "app:latest", Status: "Up 1 hour", State: "running", Ports: "8080:8080"},
			{ID: "m4n5o6", Name: "worker", Image: "app:latest", Status: "Exited (0) 2h ago", State: "exited", Ports: ""},
			{ID: "p7q8r9", Name: "test-runner", Image: "golang:1.22", Status: "Created", State: "created", Ports: ""},
		},
	}
}

func (g *RealDockerContainers) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(fmt.Sprintf(" Docker Containers (%d)", len(g.Containers)), r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	for i, c := range g.Containers {
		if y >= r.Y+r.Height-2 {
			break
		}

		stateIcon := "●"
		stateColor := mofu.Hex("a6e3a1")
		switch c.State {
		case "exited":
			stateIcon = "○"
			stateColor = mofu.Hex("585b70")
		case "created":
			stateIcon = "◐"
			stateColor = mofu.Hex("fab387")
		}

		if i == g.Selected {
			ctx.Renderer.WriteString(strings.Repeat(" ", r.Width-2), r.X+1, y, mofu.ColorWhite, mofu.Hex("2a2a3e"), 0)
		}

		line := fmt.Sprintf(" %s %-6s %-16s %-20s %s", stateIcon, c.ID, c.Name, c.Image, c.Status)
		if len(line) > r.Width-2 {
			line = line[:r.Width-2]
		}
		ctx.Renderer.WriteString(line, r.X, y, stateColor, mofu.ColorBlack, 0)
		y++

		if c.Ports != "" && y < r.Y+r.Height-1 {
			ctx.Renderer.WriteString(fmt.Sprintf("    Ports: %s", c.Ports), r.X, y, mofu.Hex("6c7086"), mofu.ColorBlack, 0)
			y++
		}
	}
}

func (g *RealDockerContainers) HandleEvent(e mofu.Event) mofu.Cmd {
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
		if g.Selected < len(g.Containers)-1 {
			g.Selected++
		}
	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if g.Selected > 0 {
			g.Selected--
		}
	case len(ke.Runes) > 0 && ke.Runes[0] == 's' && g.OnAction != nil:
		if g.Selected < len(g.Containers) {
			g.OnAction(g.Containers[g.Selected].ID, "stop")
		}
	case len(ke.Runes) > 0 && ke.Runes[0] == 'r' && g.OnAction != nil:
		if g.Selected < len(g.Containers) {
			g.OnAction(g.Containers[g.Selected].ID, "restart")
		}
	}
	return nil
}

type RealCronScheduler struct {
	Base
	Jobs   []CronJob
	Selected int
	mu     sync.RWMutex
	OnToggle func(idx int)
}

type CronJob struct {
	Schedule string
	Command  string
	Enabled  bool
	LastRun  time.Time
	NextRun  time.Time
}

func NewRealCronScheduler(id string) *RealCronScheduler {
	now := time.Now()
	return &RealCronScheduler{
		Base: *NewBase(id),
		Jobs: []CronJob{
			{Schedule: "*/5 * * * *", Command: "health-check", Enabled: true, LastRun: now.Add(-3 * time.Minute), NextRun: now.Add(2 * time.Minute)},
			{Schedule: "0 * * * *", Command: "sync-logs", Enabled: true, LastRun: now.Add(-30 * time.Minute), NextRun: now.Add(30 * time.Minute)},
			{Schedule: "0 0 * * *", Command: "backup-db", Enabled: true, LastRun: now.Add(-12 * time.Hour), NextRun: now.Add(12 * time.Hour)},
			{Schedule: "0 9 * * 1-5", Command: "report-gen", Enabled: true, LastRun: now.Add(-24 * time.Hour), NextRun: now.Add(12 * time.Hour)},
			{Schedule: "*/30 * * * *", Command: "cache-purge", Enabled: false, LastRun: now.Add(-72 * time.Hour), NextRun: time.Time{}},
			{Schedule: "0 2 * * *", Command: "cleanup-temp", Enabled: true, LastRun: now.Add(-22 * time.Hour), NextRun: now.Add(2 * time.Hour)},
		},
	}
}

func (g *RealCronScheduler) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(" Cron Scheduler", r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	header := fmt.Sprintf(" %-4s %-15s %-16s %-18s %-18s", "On", "Schedule", "Command", "Last Run", "Next Run")
	ctx.Renderer.WriteString(header, r.X, y, mofu.Hex("89b4fa"), mofu.ColorBlack, mofu.AttrBold)
	y++

	for i, job := range g.Jobs {
		if y >= r.Y+r.Height-1 {
			break
		}

		icon := "○"
		if job.Enabled {
			icon = "●"
		}

		color := mofu.Hex("cdd6f4")
		if !job.Enabled {
			color = mofu.Hex("585b70")
		}

		if i == g.Selected {
			ctx.Renderer.WriteString(strings.Repeat(" ", r.Width-2), r.X+1, y, mofu.ColorWhite, mofu.Hex("2a2a3e"), 0)
			color = mofu.Hex("ff69b4")
		}

		line := fmt.Sprintf(" %s %-14s %-16s %-18s %-18s",
			icon, job.Schedule, job.Command,
			job.LastRun.Format("Jan 2 15:04"),
			job.NextRun.Format("Jan 2 15:04"))
		if len(line) > r.Width-2 {
			line = line[:r.Width-2]
		}
		ctx.Renderer.WriteString(line, r.X, y, color, mofu.ColorBlack, 0)
		y++
	}
}

func (g *RealCronScheduler) HandleEvent(e mofu.Event) mofu.Cmd {
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
		if g.Selected < len(g.Jobs)-1 {
			g.Selected++
		}
	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if g.Selected > 0 {
			g.Selected--
		}
	case ke.Key == mofu.KeySpace:
		if g.Selected < len(g.Jobs) {
			g.Jobs[g.Selected].Enabled = !g.Jobs[g.Selected].Enabled
			if g.OnToggle != nil {
				g.OnToggle(g.Selected)
			}
		}
	}
	return nil
}

type RealEnvConfig struct {
	Base
	Vars     []EnvVar
	Selected int
	Filter   string
	mu       sync.RWMutex
}

type EnvVar struct {
	Key      string
	Value    string
	Modified bool
}

func NewRealEnvConfig(id string) *RealEnvConfig {
	return &RealEnvConfig{
		Base: *NewBase(id),
		Vars: []EnvVar{
			{Key: "GOOS", Value: "linux"},
			{Key: "GOARCH", Value: "amd64"},
			{Key: "CGO_ENABLED", Value: "0"},
			{Key: "GOPATH", Value: "/home/user/go"},
			{Key: "GOPROXY", Value: "https://proxy.golang.org,direct"},
			{Key: "GORACE", Value: "halt_on_error=1"},
			{Key: "GODEBUG", Value: "gctrace=1"},
		},
	}
}

func (g *RealEnvConfig) Set(key, value string) {
	g.mu.Lock()
	for i, v := range g.Vars {
		if v.Key == key {
			g.Vars[i].Value = value
			g.Vars[i].Modified = true
			g.mu.Unlock()
			return
		}
	}
	g.Vars = append(g.Vars, EnvVar{Key: key, Value: value, Modified: true})
	g.mu.Unlock()
}

func (g *RealEnvConfig) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(" Environment Config", r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	keyW := 20
	for i, v := range g.Vars {
		if y >= r.Y+r.Height-2 {
			break
		}

		if g.Filter != "" && !strings.Contains(strings.ToLower(v.Key), strings.ToLower(g.Filter)) {
			continue
		}

		style := mofu.DefaultStyle()
		if i == g.Selected {
			ctx.Renderer.WriteString(strings.Repeat(" ", r.Width-2), r.X+1, y, mofu.ColorWhite, mofu.Hex("2a2a3e"), 0)
			style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))
		}

		val := v.Value
		if len(val) > r.Width-keyW-8 {
			val = val[:r.Width-keyW-11] + "..."
		}

		line := fmt.Sprintf(" %-*s = %s", keyW, v.Key, val)
		if v.Modified {
			line += " *"
		}
		ctx.Renderer.WriteString(line, r.X, y, style.Foreground, style.Background, style.Attrs)
		y++
	}
}

func (g *RealEnvConfig) HandleEvent(e mofu.Event) mofu.Cmd {
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
		if g.Selected < len(g.Vars)-1 {
			g.Selected++
		}
	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if g.Selected > 0 {
			g.Selected--
		}
	}
	return nil
}
