package gadgets

import (
	"fmt"
	"strings"
	"sync"

	"github.com/xanstomper/mofu"
)

// ---------------------------------------------------------------------------
// More Real Functional Gadgets
// ---------------------------------------------------------------------------

// RealTreeTable — Hierarchical data with expand/collapse
type RealTreeTable struct {
	Base
	Nodes    []*RealTreeNode
	Selected int
	Offset   int
	mu       sync.RWMutex
}

type RealTreeNode struct {
	ID       string
	Label    string
	Data     any
	Children []*RealTreeNode
	Expanded bool
	Level    int
}

func NewRealTreeTable(id string) *RealTreeTable {
	return &RealTreeTable{Base: *NewBase(id)}
}

func (g *RealTreeTable) AddNode(node *RealTreeNode) {
	g.mu.Lock()
	g.Nodes = append(g.Nodes, node)
	g.mu.Unlock()
}

func (g *RealTreeTable) SetNodes(nodes []*RealTreeNode) {
	g.mu.Lock()
	g.Nodes = nodes
	g.mu.Unlock()
}

func (g *RealTreeTable) GetSelected() *RealTreeNode {
	g.mu.RLock()
	defer g.mu.RUnlock()
	flat := g.flattenNodes()
	if g.Selected >= 0 && g.Selected < len(flat) {
		return flat[g.Selected]
	}
	return nil
}

func (g *RealTreeTable) GetFlatNodes() []*RealTreeNode {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.flattenNodes()
}

func (g *RealTreeTable) Expand(id string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, node := range g.Nodes {
		if found := g.findNode(node, id); found != nil {
			found.Expanded = true
			return
		}
	}
}

func (g *RealTreeTable) Collapse(id string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, node := range g.Nodes {
		if found := g.findNode(node, id); found != nil {
			found.Expanded = false
			return
		}
	}
}

func (g *RealTreeTable) Toggle(id string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, node := range g.Nodes {
		if found := g.findNode(node, id); found != nil {
			found.Expanded = !found.Expanded
			return
		}
	}
}

func (g *RealTreeTable) findNode(node *RealTreeNode, id string) *RealTreeNode {
	if node.ID == id {
		return node
	}
	for _, child := range node.Children {
		if found := g.findNode(child, id); found != nil {
			return found
		}
	}
	return nil
}

func (g *RealTreeTable) flattenNodes() []*RealTreeNode {
	var result []*RealTreeNode
	var flatten func(nodes []*RealTreeNode, level int)
	flatten = func(nodes []*RealTreeNode, level int) {
		for _, node := range nodes {
			node.Level = level
			result = append(result, node)
			if node.Expanded {
				flatten(node.Children, level+1)
			}
		}
	}
	flatten(g.Nodes, 0)
	return result
}

func (g *RealTreeTable) Render(state StateView) []RenderNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var nodes []RenderNode
	flat := g.flattenNodes()

	for i, node := range flat {
		indent := strings.Repeat("  ", node.Level)
		icon := "├─"
		if len(node.Children) == 0 {
			icon = "└─"
		}
		if node.Expanded {
			icon = "▼ "
		} else if len(node.Children) > 0 {
			icon = "▶ "
		}

		text := indent + icon + " " + node.Label
		style := mofu.DefaultStyle()
		if i == g.Selected {
			style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))
		}

		nodes = append(nodes, RenderNode{Type: "text", Content: text, Style: style})
	}
	return nodes
}

func (g *RealTreeTable) OnEvent(e Event) {
	if e.Type == "keypress" {
		if ke, ok := e.Payload.(mofu.KeyEvent); ok {
			g.mu.Lock()
			defer g.mu.Unlock()
			flat := g.flattenNodes()
			switch ke.Key {
			case mofu.KeyDown:
				if g.Selected < len(flat)-1 {
					g.Selected++
				}
			case mofu.KeyUp:
				if g.Selected > 0 {
					g.Selected--
				}
			case mofu.KeyRight:
				if g.Selected < len(flat) && len(flat[g.Selected].Children) > 0 {
					flat[g.Selected].Expanded = true
				}
			case mofu.KeyLeft:
				if g.Selected < len(flat) && flat[g.Selected].Expanded {
					flat[g.Selected].Expanded = false
				}
			}
		}
	}
}

// ---------------------------------------------------------------------------
// RealForm — Schema-driven form with validation
// ---------------------------------------------------------------------------

type RealForm struct {
	Base
	fields    []*FormField
	values    map[string]any
	errors    map[string]error
	focus     int
	onChange  func(name string, value any)
	onSubmit  func(values map[string]any)
	mu        sync.RWMutex
}

type FormFieldDef struct {
	Name      string
	Label     string
	Type      string
	Required  bool
	Validator func(any) error
}

func NewRealForm(id string) *RealForm {
	return &RealForm{
		Base:   *NewBase(id),
		fields: make([]*FormField, 0),
		values: make(map[string]any),
		errors: make(map[string]error),
	}
}

func (g *RealForm) AddField(field *FormField) {
	g.mu.Lock()
	g.fields = append(g.fields, field)
	g.values[field.Name] = field.Value
	g.mu.Unlock()
}

func (g *RealForm) SetValue(name string, value any) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.values[name] = value
	g.validateField(name)

	if g.onChange != nil {
		g.onChange(name, value)
	}
}

func (g *RealForm) GetValue(name string) any {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.values[name]
}

func (g *RealForm) GetError(name string) error {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.errors[name]
}

func (g *RealForm) IsValid() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.errors) == 0
}

func (g *RealForm) Submit() bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.validateAll()
	if len(g.errors) > 0 {
		return false
	}

	if g.onSubmit != nil {
		g.onSubmit(g.values)
	}
	return true
}

func (g *RealForm) validateField(name string) {
	for _, field := range g.fields {
		if field.Name == name {
			if field.Validator != nil {
				if err := field.Validator(g.values[name]); err != nil {
					g.errors[name] = err
					return
				}
			}
			delete(g.errors, name)
			return
		}
	}
}

func (g *RealForm) validateAll() {
	g.errors = make(map[string]error)
	for _, field := range g.fields {
		if field.Required {
			val := g.values[field.Name]
			if val == nil || val == "" {
				g.errors[field.Name] = fmt.Errorf("%s is required", field.Label)
			}
		}
		if field.Validator != nil {
			if err := field.Validator(g.values[field.Name]); err != nil {
				g.errors[field.Name] = err
			}
		}
	}
}

func (g *RealForm) Render(state StateView) []RenderNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var nodes []RenderNode
	for i, field := range g.fields {
		style := mofu.DefaultStyle()
		if i == g.focus {
			style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))
		}

		value := fmt.Sprintf("%v", g.values[field.Name])
		if value == "" {
			value = field.Placeholder
		}

		text := fmt.Sprintf("%-20s %s", field.Label+":", value)
		nodes = append(nodes, RenderNode{Type: "text", Content: text, Style: style})

		if err, exists := g.errors[field.Name]; exists {
			nodes = append(nodes, RenderNode{
				Type:    "text",
				Content: "  ✗ " + err.Error(),
				Style:   mofu.DefaultStyle().Fg(mofu.Hex("f38ba8")),
			})
		}
	}
	return nodes
}

func (g *RealForm) OnEvent(e Event) {
	if e.Type == "keypress" {
		if ke, ok := e.Payload.(mofu.KeyEvent); ok {
			switch ke.Key {
			case mofu.KeyTab:
				g.mu.Lock()
				g.focus = (g.focus + 1) % len(g.fields)
				g.mu.Unlock()
			case mofu.KeyEnter:
				g.Submit()
			}
		}
	}
}

// ---------------------------------------------------------------------------
// RealKanban — Kanban board with columns and cards
// ---------------------------------------------------------------------------

type RealKanban struct {
	Base
	Columns  []*KanbanColumn
	ColIndex int
	CardIndex int
	mu       sync.RWMutex
}

type KanbanColumn struct {
	Name  string
	Cards []*KanbanCard
}

type KanbanCard struct {
	ID    string
	Title string
	Color string
}

func NewRealKanban(id string) *RealKanban {
	return &RealKanban{
		Base: *NewBase(id),
		Columns: []*KanbanColumn{
			{Name: "To Do"},
			{Name: "In Progress"},
			{Name: "Done"},
		},
	}
}

func (g *RealKanban) AddCard(colIndex int, card *KanbanCard) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if colIndex >= 0 && colIndex < len(g.Columns) {
		g.Columns[colIndex].Cards = append(g.Columns[colIndex].Cards, card)
	}
}

func (g *RealKanban) MoveCard(fromCol, toCol, cardIndex int) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if fromCol < 0 || fromCol >= len(g.Columns) || toCol < 0 || toCol >= len(g.Columns) {
		return
	}
	from := g.Columns[fromCol]
	to := g.Columns[toCol]

	if cardIndex < 0 || cardIndex >= len(from.Cards) {
		return
	}

	card := from.Cards[cardIndex]
	from.Cards = append(from.Cards[:cardIndex], from.Cards[cardIndex+1:]...)
	to.Cards = append(to.Cards, card)
}

func (g *RealKanban) DeleteCard(colIndex, cardIndex int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if colIndex >= 0 && colIndex < len(g.Columns) {
		col := g.Columns[colIndex]
		if cardIndex >= 0 && cardIndex < len(col.Cards) {
			col.Cards = append(col.Cards[:cardIndex], col.Cards[cardIndex+1:]...)
		}
	}
}

func (g *RealKanban) GetSelectedCard() *KanbanCard {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.ColIndex >= 0 && g.ColIndex < len(g.Columns) {
		col := g.Columns[g.ColIndex]
		if g.CardIndex >= 0 && g.CardIndex < len(col.Cards) {
			return col.Cards[g.CardIndex]
		}
	}
	return nil
}

func (g *RealKanban) GetColumn(index int) *KanbanColumn {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if index >= 0 && index < len(g.Columns) {
		return g.Columns[index]
	}
	return nil
}

func (g *RealKanban) Render(state StateView) []RenderNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var nodes []RenderNode
	colWidth := 20

	for ci, col := range g.Columns {
		// Column header
		style := mofu.DefaultStyle().Fg(mofu.Hex("89b4fa")).WithAttrs(mofu.AttrBold)
		header := fmt.Sprintf(" %s (%d) ", col.Name, len(col.Cards))
		nodes = append(nodes, RenderNode{Type: "text", Content: header, Style: style})

		// Cards
		for i, card := range col.Cards {
			cardStyle := mofu.DefaultStyle()
			if ci == g.ColIndex && i == g.CardIndex {
				cardStyle = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))
			}
			title := card.Title
			if len(title) > colWidth-4 {
				title = title[:colWidth-7] + "..."
			}
			nodes = append(nodes, RenderNode{Type: "text", Content: "  " + title, Style: cardStyle})
		}

		_ = colWidth
	}
	return nodes
}

func (g *RealKanban) OnEvent(e Event) {
	if e.Type == "keypress" {
		if ke, ok := e.Payload.(mofu.KeyEvent); ok {
			g.mu.Lock()
			defer g.mu.Unlock()

			switch ke.Key {
			case mofu.KeyLeft:
				if g.ColIndex > 0 {
					g.ColIndex--
					g.CardIndex = 0
				}
			case mofu.KeyRight:
				if g.ColIndex < len(g.Columns)-1 {
					g.ColIndex++
					g.CardIndex = 0
				}
			case mofu.KeyUp:
				if g.CardIndex > 0 {
					g.CardIndex--
				}
			case mofu.KeyDown:
				if g.ColIndex < len(g.Columns) {
					maxCards := len(g.Columns[g.ColIndex].Cards) - 1
					if g.CardIndex < maxCards {
						g.CardIndex++
					}
				}
			}
		}
	}
}
