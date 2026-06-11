package gadgets

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"

	"github.com/xanstomper/mofu"
)

// =========================================================================
// BATCH 5: Games, Puzzles & Utility Gadgets (10 gadgets)
// =========================================================================

type RealMazeGenerator struct {
	Base
	Width   int
	Height  int
	Grid    [][]bool
	Visited [][]bool
	Paths   map[[2][2]int]bool
	mu      sync.RWMutex
}

func NewRealMazeGenerator(id string, w, h int) *RealMazeGenerator {
	g := &RealMazeGenerator{Base: *NewBase(id), Width: w, Height: h}
	g.Generate()
	return g
}

func (g *RealMazeGenerator) Generate() {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.Grid = make([][]bool, g.Height)
	g.Visited = make([][]bool, g.Height)
	for y := range g.Grid {
		g.Grid[y] = make([]bool, g.Width)
		g.Visited[y] = make([]bool, g.Width)
	}

	type cell struct{ x, y int }
	stack := []cell{{0, 0}}
	g.Visited[0][0] = true
	g.Grid[0][0] = true

	dirs := []struct{ dx, dy int }{{0, 1}, {1, 0}, {0, -1}, {-1, 0}}

	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		neighbors := []cell{}
		for _, d := range dirs {
			nx, ny := cur.x+d.dx*2, cur.y+d.dy*2
			if nx >= 0 && nx < g.Width && ny >= 0 && ny < g.Height && !g.Visited[ny][nx] {
				neighbors = append(neighbors, cell{nx, ny})
			}
		}
		if len(neighbors) == 0 {
			stack = stack[:len(stack)-1]
			continue
		}
		next := neighbors[rand.Intn(len(neighbors))]
		g.Grid[cur.y+(next.y-cur.y)/2][cur.x+(next.x-cur.x)/2] = true
		g.Grid[next.y][next.x] = true
		g.Visited[next.y][next.x] = true
		stack = append(stack, next)
	}
}

func (g *RealMazeGenerator) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(" Maze Generator", r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++

	for gy := 0; gy < g.Height && y < r.Y+r.Height-1; gy++ {
		line := ""
		for gx := 0; gx < g.Width && len(line) < r.Width-2; gx++ {
			if g.Grid[gy][gx] {
				line += "  "
			} else {
				line += "██"
			}
		}
		ctx.Renderer.WriteString(" "+line, r.X, y, mofu.Hex("89b4fa"), mofu.ColorBlack, 0)
		y++
	}
}

func (g *RealMazeGenerator) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)
	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()
	case len(ke.Runes) > 0 && ke.Runes[0] == 'r':
		g.Generate()
	}
	return nil
}

type RealSnakeGame struct {
	Base
	Body      [][2]int
	Food      [2]int
	Score     int
	HighScore int
	Dir       int
	GameOver  bool
	Width     int
	Height    int
	mu        sync.RWMutex
}

func NewRealSnakeGame(id string, w, h int) *RealSnakeGame {
	g := &RealSnakeGame{Base: *NewBase(id), Width: w, Height: h, Dir: 1}
	g.Reset()
	return g
}

func (g *RealSnakeGame) Reset() {
	g.Body = [][2]int{{g.Width / 2, g.Height / 2}, {g.Width / 2 - 1, g.Height / 2}, {g.Width / 2 - 2, g.Height / 2}}
	g.SpawnFood()
	g.Score = 0
	g.GameOver = false
	g.Dir = 1
}

func (g *RealSnakeGame) SpawnFood() {
	for {
		g.Food = [2]int{rand.Intn(g.Width), rand.Intn(g.Height)}
		hit := false
		for _, s := range g.Body {
			if s == g.Food {
				hit = true
				break
			}
		}
		if !hit {
			return
		}
	}
}

func (g *RealSnakeGame) Move() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.GameOver {
		return
	}

	head := g.Body[0]
	switch g.Dir {
	case 0:
		head = [2]int{head[0], head[1] - 1}
	case 1:
		head = [2]int{head[0] + 1, head[1]}
	case 2:
		head = [2]int{head[0], head[1] + 1}
	case 3:
		head = [2]int{head[0] - 1, head[1]}
	}

	if head[0] < 0 || head[0] >= g.Width || head[1] < 0 || head[1] >= g.Height {
		g.GameOver = true
		if g.Score > g.HighScore {
			g.HighScore = g.Score
		}
		return
	}

	for _, s := range g.Body {
		if s == head {
			g.GameOver = true
			if g.Score > g.HighScore {
				g.HighScore = g.Score
			}
			return
		}
	}

	g.Body = append([][2]int{head}, g.Body...)

	if head == g.Food {
		g.Score += 10
		g.SpawnFood()
	} else {
		g.Body = g.Body[:len(g.Body)-1]
	}
}

func (g *RealSnakeGame) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(fmt.Sprintf(" Snake  Score: %d  High: %d", g.Score, g.HighScore), r.X, y, mofu.Hex("a6e3a1"), mofu.ColorBlack, mofu.AttrBold)
	y++

	for gy := 0; gy < g.Height && y < r.Y+r.Height-2; gy++ {
		line := ""
		for gx := 0; gx < g.Width; gx++ {
			pos := [2]int{gx, gy}
			if g.Body[0] == pos {
				line += "██"
			} else if len(g.Body) > 1 {
				isBody := false
				for _, s := range g.Body[1:] {
					if s == pos {
						isBody = true
						break
					}
				}
				if isBody {
					line += "▓▓"
				} else if g.Food == pos {
					line += "◆◆"
				} else {
					line += "  "
				}
			} else if g.Food == pos {
				line += "◆◆"
			} else {
				line += "  "
			}
		}
		ctx.Renderer.WriteString(" "+line, r.X, y, mofu.Hex("89b4fa"), mofu.ColorBlack, 0)
		y++
	}

	if g.GameOver {
		msg := " GAME OVER - Press r to restart "
		x := r.X + (r.Width-len(msg))/2
		y2 := r.Y + r.Height/2
		ctx.Renderer.WriteString(msg, x, y2, mofu.Hex("f38ba8"), mofu.ColorBlack, mofu.AttrBold)
	}
}

func (g *RealSnakeGame) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	g.mu.Lock()
	defer g.mu.Unlock()

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()
	case (ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k')) && g.Dir != 2:
		g.Dir = 0
	case (ke.Key == mofu.KeyRight || (len(ke.Runes) > 0 && ke.Runes[0] == 'l')) && g.Dir != 3:
		g.Dir = 1
	case (ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j')) && g.Dir != 0:
		g.Dir = 2
	case (ke.Key == mofu.KeyLeft || (len(ke.Runes) > 0 && ke.Runes[0] == 'h')) && g.Dir != 1:
		g.Dir = 3
	case len(ke.Runes) > 0 && ke.Runes[0] == 'r':
		g.Body = [][2]int{{g.Width / 2, g.Height / 2}, {g.Width / 2 - 1, g.Height / 2}, {g.Width / 2 - 2, g.Height / 2}}
		g.SpawnFood()
		g.Score = 0
		g.GameOver = false
		g.Dir = 1
	}
	return nil
}

type RealSimonSays struct {
	Base
	Sequence    []int
	PlayerInput []int
	Level       int
	Score       int
	Showing     bool
	ShowIndex   int
	GameOver    bool
	Colors      []string
	mu          sync.RWMutex
}

func NewRealSimonSays(id string) *RealSimonSays {
	return &RealSimonSays{
		Base:   *NewBase(id),
		Colors: []string{"RED", "BLUE", "GREEN", "YELLOW"},
	}
}

func (g *RealSimonSays) StartGame() {
	g.mu.Lock()
	g.Sequence = []int{rand.Intn(4)}
	g.PlayerInput = nil
	g.Level = 1
	g.Score = 0
	g.GameOver = false
	g.mu.Unlock()
}

func (g *RealSimonSays) NextLevel() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.Sequence = append(g.Sequence, rand.Intn(4))
	g.Level++
	g.PlayerInput = nil
}

func (g *RealSimonSays) Input(color int) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.GameOver || len(g.PlayerInput) >= len(g.Sequence) {
		return false
	}

	g.PlayerInput = append(g.PlayerInput, color)

	for i := range g.PlayerInput {
		if g.PlayerInput[i] != g.Sequence[i] {
			g.GameOver = true
			if g.Score > 0 {
				g.Score--
			}
			return false
		}
	}

	if len(g.PlayerInput) == len(g.Sequence) {
		g.Score += g.Level * 10
		return true
	}
	return false
}

func (g *RealSimonSays) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(" Simon Says", r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++

	ctx.Renderer.WriteString(fmt.Sprintf(" Level: %d  Score: %d", g.Level, g.Score), r.X, y, mofu.Hex("f9e2af"), mofu.ColorBlack, 0)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	colors := []mofu.Color{mofu.Hex("f38ba8"), mofu.Hex("89b4fa"), mofu.Hex("a6e3a1"), mofu.Hex("f9e2af")}
	names := []string{"1:RED", "2:BLUE", "3:GREEN", "4:YELLOW"}

	for i := 0; i < 4; i++ {
		if y >= r.Y+r.Height-2 {
			break
		}
		ctx.Renderer.WriteString(fmt.Sprintf("  [%s] %s", strings.Repeat("█", 8), names[i]), r.X, y, colors[i], mofu.ColorBlack, 0)
		y++
	}

	y++
	if g.GameOver {
		ctx.Renderer.WriteString(fmt.Sprintf(" GAME OVER! Score: %d", g.Score), r.X, y, mofu.Hex("f38ba8"), mofu.ColorBlack, mofu.AttrBold)
	} else if g.Showing {
		ctx.Renderer.WriteString(fmt.Sprintf(" Watch sequence (%d/%d)...", g.ShowIndex+1, len(g.Sequence)), r.X, y, mofu.Hex("89b4fa"), mofu.ColorBlack, 0)
	} else {
		ctx.Renderer.WriteString(fmt.Sprintf(" Your turn! (%d/%d)", len(g.PlayerInput)+1, len(g.Sequence)), r.X, y, mofu.Hex("a6e3a1"), mofu.ColorBlack, 0)
	}
	y++
	ctx.Renderer.WriteString(" 1-4:input s:start q:quit", r.X, y, mofu.Hex("585b70"), mofu.ColorBlack, 0)
}

func (g *RealSimonSays) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()
	case len(ke.Runes) > 0 && ke.Runes[0] == 's':
		g.StartGame()
	case len(ke.Runes) > 0 && ke.Runes[0] >= '1' && ke.Runes[0] <= '4':
		color := int(ke.Runes[0] - '1')
		if g.Input(color) {
			g.NextLevel()
		}
	}
	return nil
}

type RealDiceRoller struct {
	Base
	Faces    int
	Count    int
	Results  []int
	History  map[int]int
	Total    int
	mu       sync.RWMutex
}

func NewRealDiceRoller(id string, faces, count int) *RealDiceRoller {
	return &RealDiceRoller{
		Base:    *NewBase(id),
		Faces:   faces,
		Count:   count,
		History: make(map[int]int),
	}
}

func (g *RealDiceRoller) Roll() int {
	g.mu.Lock()
	defer g.mu.Unlock()

	total := 0
	g.Results = nil
	for i := 0; i < g.Count; i++ {
		val := rand.Intn(g.Faces) + 1
		g.Results = append(g.Results, val)
		g.History[val]++
		total += val
	}
	g.Total += total
	return total
}

func (g *RealDiceRoller) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(fmt.Sprintf(" Dice Roller (%dd%d)", g.Count, g.Faces), r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	if len(g.Results) > 0 {
		line := " Results: "
		for _, r := range g.Results {
			line += fmt.Sprintf("[%d] ", r)
		}
		ctx.Renderer.WriteString(line, r.X, y, mofu.Hex("f9e2af"), mofu.ColorBlack, mofu.AttrBold)
		y++

		sum := 0
		for _, r := range g.Results {
			sum += r
		}
		ctx.Renderer.WriteString(fmt.Sprintf(" Total: %d", sum), r.X, y, mofu.Hex("a6e3a1"), mofu.ColorBlack, 0)
		y++
	}

	y++
	ctx.Renderer.WriteString(" Distribution:", r.X, y, mofu.Hex("89b4fa"), mofu.ColorBlack, mofu.AttrBold)
	y++

	totalRolls := 0
	for _, v := range g.History {
		totalRolls += v
	}

	if totalRolls > 0 {
		for face := 1; face <= g.Faces; face++ {
			if y >= r.Y+r.Height-2 {
				break
			}
			count := g.History[face]
			pct := float64(count) / float64(totalRolls) * 100
			barW := r.Width - 20
			filled := int(pct / 100 * float64(barW))
			bar := strings.Repeat("█", filled) + strings.Repeat("░", barW-filled)
			ctx.Renderer.WriteString(fmt.Sprintf("  %2d: %s %d (%.1f%%)", face, bar, count, pct), r.X, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
			y++
		}
	}

	ctx.Renderer.WriteString(fmt.Sprintf(" Total rolls: %d", totalRolls), r.X, r.Y+r.Height-2, mofu.Hex("585b70"), mofu.ColorBlack, 0)
	ctx.Renderer.WriteString(" r:roll +/-:count q:quit", r.X, r.Y+r.Height-1, mofu.Hex("585b70"), mofu.Hex("1e1e2e"), 0)
}

func (g *RealDiceRoller) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()
	case len(ke.Runes) > 0 && ke.Runes[0] == 'r':
		g.Roll()
	case len(ke.Runes) > 0 && ke.Runes[0] == '+':
		g.mu.Lock()
		g.Count++
		g.mu.Unlock()
	case len(ke.Runes) > 0 && ke.Runes[0] == '-':
		g.mu.Lock()
		if g.Count > 1 {
			g.Count--
		}
		g.mu.Unlock()
	}
	return nil
}

type RealContactBook struct {
	Base
	Contacts []Contact
	Selected int
	Filter   string
	Editing  bool
	mu       sync.RWMutex
	OnSelect func(idx int, c Contact)
}

type Contact struct {
	Name  string
	Phone string
	Email string
	Group string
}

func NewRealContactBook(id string) *RealContactBook {
	return &RealContactBook{
		Base: *NewBase(id),
		Contacts: []Contact{
			{Name: "Alice Johnson", Phone: "555-0101", Email: "alice@example.com", Group: "Work"},
			{Name: "Bob Smith", Phone: "555-0102", Email: "bob@example.com", Group: "Friends"},
			{Name: "Charlie Brown", Phone: "555-0103", Email: "charlie@example.com", Group: "Family"},
			{Name: "Diana Prince", Phone: "555-0104", Email: "diana@example.com", Group: "Work"},
			{Name: "Eve Wilson", Phone: "555-0105", Email: "eve@example.com", Group: "Friends"},
		},
	}
}

func (g *RealContactBook) filtered() []Contact {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.Filter == "" {
		return g.Contacts
	}
	var result []Contact
	for _, c := range g.Contacts {
		if strings.Contains(strings.ToLower(c.Name), strings.ToLower(g.Filter)) ||
			strings.Contains(strings.ToLower(c.Group), strings.ToLower(g.Filter)) {
			result = append(result, c)
		}
	}
	return result
}

func (g *RealContactBook) AddContact(c Contact) {
	g.mu.Lock()
	g.Contacts = append(g.Contacts, c)
	g.mu.Unlock()
}

func (g *RealContactBook) DeleteContact(idx int) {
	g.mu.Lock()
	if idx >= 0 && idx < len(g.Contacts) {
		g.Contacts = append(g.Contacts[:idx], g.Contacts[idx+1:]...)
	}
	g.mu.Unlock()
}

func (g *RealContactBook) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(fmt.Sprintf(" Contacts (%d)", len(g.Contacts)), r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++

	if g.Filter != "" {
		ctx.Renderer.WriteString(fmt.Sprintf(" Filter: %s", g.Filter), r.X, y, mofu.Hex("89b4fa"), mofu.ColorBlack, 0)
		y++
	}

	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	contacts := g.filtered()
	for i, c := range contacts {
		if y >= r.Y+r.Height-2 {
			break
		}

		style := mofu.DefaultStyle()
		if i == g.Selected {
			ctx.Renderer.WriteString(strings.Repeat(" ", r.Width-2), r.X+1, y, mofu.ColorWhite, mofu.Hex("2a2a3e"), 0)
			style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))
		}

		line := fmt.Sprintf(" %-20s %-15s %-20s", c.Name, c.Phone, c.Group)
		if len(line) > r.Width-2 {
			line = line[:r.Width-2]
		}
		ctx.Renderer.WriteString(line, r.X, y, style.Foreground, style.Background, style.Attrs)
		y++
	}

	ctx.Renderer.WriteString(" j/k:navigate /:search n:new d:delete q:quit", r.X, r.Y+r.Height-1, mofu.Hex("585b70"), mofu.Hex("1e1e2e"), 0)
}

func (g *RealContactBook) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	contacts := g.filtered()

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()
	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		if g.Selected < len(contacts)-1 {
			g.Selected++
		}
	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if g.Selected > 0 {
			g.Selected--
		}
	case len(ke.Runes) > 0 && ke.Runes[0] == '/':
		g.Filter = ""
		g.Selected = 0
	case ke.Key == mofu.KeyBack && len(g.Filter) > 0:
		g.Filter = g.Filter[:len(g.Filter)-1]
		g.Selected = 0
	default:
		if len(ke.Runes) > 0 && ke.Runes[0] >= 'a' && ke.Runes[0] <= 'z' {
			g.Filter += string(ke.Runes)
			g.Selected = 0
		}
	}
	return nil
}

type RealShoppingList struct {
	Base
	Items      []ShoppingItem
	Selected   int
	Categories []string
	mu         sync.RWMutex
	OnComplete func(idx int)
}

type ShoppingItem struct {
	Name     string
	Quantity int
	Category string
	Done     bool
}

func NewRealShoppingList(id string) *RealShoppingList {
	return &RealShoppingList{
		Base: *NewBase(id),
		Categories: []string{"Produce", "Dairy", "Meat", "Bakery", "Frozen", "Other"},
		Items: []ShoppingItem{
			{Name: "Apples", Quantity: 6, Category: "Produce"},
			{Name: "Milk", Quantity: 2, Category: "Dairy"},
			{Name: "Bread", Quantity: 1, Category: "Bakery"},
			{Name: "Chicken", Quantity: 1, Category: "Meat"},
			{Name: "Eggs", Quantity: 12, Category: "Dairy"},
			{Name: "Bananas", Quantity: 1, Category: "Produce"},
			{Name: "Ice Cream", Quantity: 1, Category: "Frozen"},
			{Name: "Rice", Quantity: 2, Category: "Other"},
		},
	}
}

func (g *RealShoppingList) Add(name string, qty int, category string) {
	g.mu.Lock()
	g.Items = append(g.Items, ShoppingItem{Name: name, Quantity: qty, Category: category})
	g.mu.Unlock()
}

func (g *RealShoppingList) ToggleDone(idx int) {
	g.mu.Lock()
	if idx >= 0 && idx < len(g.Items) {
		g.Items[idx].Done = !g.Items[idx].Done
	}
	g.mu.Unlock()
}

func (g *RealShoppingList) RemoveDone() {
	g.mu.Lock()
	var remaining []ShoppingItem
	for _, item := range g.Items {
		if !item.Done {
			remaining = append(remaining, item)
		}
	}
	g.Items = remaining
	g.mu.Unlock()
}

func (g *RealShoppingList) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	done := 0
	for _, item := range g.Items {
		if item.Done {
			done++
		}
	}
	ctx.Renderer.WriteString(fmt.Sprintf(" Shopping List (%d/%d done)", done, len(g.Items)), r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	for i, item := range g.Items {
		if y >= r.Y+r.Height-2 {
			break
		}

		icon := "○"
		if item.Done {
			icon = "●"
		}

		line := fmt.Sprintf(" %s %-20s x%-3d %-10s", icon, item.Name, item.Quantity, item.Category)
		if len(line) > r.Width-2 {
			line = line[:r.Width-2]
		}

		color := mofu.Hex("cdd6f4")
		if item.Done {
			color = mofu.Hex("585b70")
		}

		if i == g.Selected {
			ctx.Renderer.WriteString(strings.Repeat(" ", r.Width-2), r.X+1, y, mofu.ColorWhite, mofu.Hex("2a2a3e"), 0)
			color = mofu.Hex("ff69b4")
		}
		ctx.Renderer.WriteString(line, r.X, y, color, mofu.ColorBlack, 0)
		y++
	}

	ctx.Renderer.WriteString(" j/k:navigate space:toggle d:remove done a:add q:quit", r.X, r.Y+r.Height-1, mofu.Hex("585b70"), mofu.Hex("1e1e2e"), 0)
}

func (g *RealShoppingList) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()
	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		if g.Selected < len(g.Items)-1 {
			g.Selected++
		}
	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if g.Selected > 0 {
			g.Selected--
		}
	case ke.Key == mofu.KeySpace:
		g.ToggleDone(g.Selected)
	case len(ke.Runes) > 0 && ke.Runes[0] == 'd':
		g.RemoveDone()
	}
	return nil
}

type RealRecipeBook struct {
	Base
	Recipes   []Recipe
	Selected  int
	mu        sync.RWMutex
	OnSelect  func(idx int, r Recipe)
}

type Recipe struct {
	Name        string
	Category    string
	Time        int
	Difficulty  string
	Ingredients []string
	Steps       []string
}

func NewRealRecipeBook(id string) *RealRecipeBook {
	return &RealRecipeBook{
		Base: *NewBase(id),
		Recipes: []Recipe{
			{
				Name: "Pasta Carbonara", Category: "Italian", Time: 20, Difficulty: "Easy",
				Ingredients: []string{"Spaghetti", "Eggs", "Parmesan", "Pancetta", "Pepper"},
				Steps: []string{"Boil pasta", "Cook pancetta", "Mix eggs and cheese", "Combine"},
			},
			{
				Name: "Chicken Stir Fry", Category: "Asian", Time: 15, Difficulty: "Easy",
				Ingredients: []string{"Chicken", "Soy sauce", "Vegetables", "Rice"},
				Steps: []string{"Slice chicken", "Stir fry vegetables", "Add sauce", "Serve over rice"},
			},
			{
				Name: "Beef Stew", Category: "American", Time: 120, Difficulty: "Medium",
				Ingredients: []string{"Beef", "Potatoes", "Carrots", "Onions", "Broth"},
				Steps: []string{"Brown beef", "Add vegetables", "Simmer 2 hours"},
			},
		},
	}
}

func (g *RealRecipeBook) AddRecipe(r Recipe) {
	g.mu.Lock()
	g.Recipes = append(g.Recipes, r)
	g.mu.Unlock()
}

func (g *RealRecipeBook) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	ctx.Renderer.WriteString(" Recipe Book", r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	for i, recipe := range g.Recipes {
		if y >= r.Y+r.Height-3 {
			break
		}

		style := mofu.DefaultStyle()
		if i == g.Selected {
			ctx.Renderer.WriteString(strings.Repeat(" ", r.Width-2), r.X+1, y, mofu.ColorWhite, mofu.Hex("2a2a3e"), 0)
			style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))
		}

		line := fmt.Sprintf(" %-20s %-10s %3dm  %s", recipe.Name, recipe.Category, recipe.Time, recipe.Difficulty)
		if len(line) > r.Width-2 {
			line = line[:r.Width-2]
		}
		ctx.Renderer.WriteString(line, r.X, y, style.Foreground, style.Background, style.Attrs)
		y++
	}

	if g.Selected < len(g.Recipes) {
		y++
		recipe := g.Recipes[g.Selected]
		ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
		y++

		ctx.Renderer.WriteString(fmt.Sprintf(" %s (%s)", recipe.Name, recipe.Category), r.X, y, mofu.Hex("89b4fa"), mofu.ColorBlack, mofu.AttrBold)
		y++

		if y < r.Y+r.Height-1 {
			ctx.Renderer.WriteString(" Ingredients:", r.X, y, mofu.Hex("a6e3a1"), mofu.ColorBlack, 0)
			y++
		}
		for _, ing := range recipe.Ingredients {
			if y >= r.Y+r.Height-1 {
				break
			}
			ctx.Renderer.WriteString("  • "+ing, r.X, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
			y++
		}
	}

	ctx.Renderer.WriteString(" j/k:navigate q:quit", r.X, r.Y+r.Height-1, mofu.Hex("585b70"), mofu.Hex("1e1e2e"), 0)
}

func (g *RealRecipeBook) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()
	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		if g.Selected < len(g.Recipes)-1 {
			g.Selected++
		}
	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if g.Selected > 0 {
			g.Selected--
		}
	}
	return nil
}

type RealFitnessTracker struct {
	Base
	Workouts []Workout
	TotalMin int
	TotalCal int
	mu       sync.RWMutex
}

type Workout struct {
	Name     string
	Duration int
	Calories int
	Date     string
}

func NewRealFitnessTracker(id string) *RealFitnessTracker {
	return &RealFitnessTracker{
		Base: *NewBase(id),
		Workouts: []Workout{
			{Name: "Running", Duration: 30, Calories: 300, Date: "2026-06-10"},
			{Name: "Weight Training", Duration: 45, Calories: 250, Date: "2026-06-09"},
			{Name: "Yoga", Duration: 60, Calories: 150, Date: "2026-06-08"},
			{Name: "Swimming", Duration: 40, Calories: 400, Date: "2026-06-07"},
			{Name: "Cycling", Duration: 50, Calories: 350, Date: "2026-06-06"},
		},
	}
}

func (g *RealFitnessTracker) AddWorkout(w Workout) {
	g.mu.Lock()
	g.Workouts = append([]Workout{w}, g.Workouts...)
	g.TotalMin += w.Duration
	g.TotalCal += w.Calories
	g.mu.Unlock()
}

func (g *RealFitnessTracker) Render(ctx *mofu.RenderContext) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	r := ctx.Bounds
	y := r.Y

	totalMin := 0
	totalCal := 0
	for _, w := range g.Workouts {
		totalMin += w.Duration
		totalCal += w.Calories
	}

	ctx.Renderer.WriteString(" Fitness Tracker", r.X, y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	ctx.Renderer.WriteString(fmt.Sprintf("  Total: %d min | %d cal | %d workouts", totalMin, totalCal, len(g.Workouts)), r.X, y, mofu.Hex("a6e3a1"), mofu.ColorBlack, 0)
	y++
	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	header := fmt.Sprintf(" %-20s %-6s %-6s %s", "Workout", "Min", "Cal", "Date")
	ctx.Renderer.WriteString(header, r.X, y, mofu.Hex("89b4fa"), mofu.ColorBlack, mofu.AttrBold)
	y++

	for i, w := range g.Workouts {
		if y >= r.Y+r.Height-2 {
			break
		}
		line := fmt.Sprintf(" %-20s %-6d %-6d %s", w.Name, w.Duration, w.Calories, w.Date)
		if len(line) > r.Width-2 {
			line = line[:r.Width-2]
		}
		if i%2 == 0 {
			ctx.Renderer.WriteString(strings.Repeat(" ", r.Width-2), r.X+1, y, mofu.ColorWhite, mofu.Hex("1e1e2e"), 0)
		}
		ctx.Renderer.WriteString(line, r.X, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
		y++
	}
}

func (g *RealFitnessTracker) HandleEvent(e mofu.Event) mofu.Cmd {
	if e.Type != mofu.EventKeyPress {
		return nil
	}
	ke := e.Data.(mofu.KeyEvent)
	if ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q') {
		return mofu.QuitCmd()
	}
	return nil
}
