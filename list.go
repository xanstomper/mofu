package mofu

import (
	"fmt"
	"strings"
	"sync"
)

type ListItem interface {
	FilterValue() string
}

type ListItemDelegate interface {
	Height() int
	Spacing() int
	Render(w int, item ListItem, index int, selected bool) string
}

type List struct {
	mu            sync.Mutex
	items         []ListItem
	filteredItems []ListItem
	selected      int
	cursor        int
	width         int
	height        int
	delegate      ListItemDelegate
	keyMap        *KeyMap
	filter        string
	filtering     bool
	title         string
	styles        ListStyles
	onSelect      func(int, ListItem)
}

type ListStyles struct {
	Title       Style
	ItemSelected Style
	Item        Style
	FilterMatch Style
	FilterPrompt Style
	StatusBar   Style
	Pagination   Style
}

func DefaultListStyles() ListStyles {
	return ListStyles{
		Title:        DefaultStyle().Fg(Hex("89b4fa")).WithAttrs(AttrBold),
		ItemSelected: DefaultStyle().Fg(Hex("1e1e2e")).Bg(Hex("89b4fa")),
		Item:         DefaultStyle().Fg(Hex("cdd6f4")),
		FilterMatch:  DefaultStyle().Fg(Hex("a6e3a1")).WithAttrs(AttrBold),
		FilterPrompt: DefaultStyle().Fg(Hex("f5c2e7")),
		StatusBar:    DefaultStyle().Fg(Hex("6c7086")),
		Pagination:   DefaultStyle().Fg(Hex("6c7086")),
	}
}

type defaultDelegate struct{}

func (d defaultDelegate) Height() int  { return 1 }
func (d defaultDelegate) Spacing() int { return 0 }
func (d defaultDelegate) Render(w int, item ListItem, index int, selected bool) string {
	label := fmt.Sprintf("  %s", item.FilterValue())
	if len(label) > w {
		label = label[:w]
	}
	if selected {
		return DefaultStyle().Fg(Hex("1e1e2e")).Bg(Hex("89b4fa")).Apply(label)
	}
	return DefaultStyle().Fg(Hex("cdd6f4")).Apply(label)
}

func NewList(items []ListItem) *List {
	l := &List{
		items:    items,
		delegate: &defaultDelegate{},
		keyMap:   NewKeyMap(),
		styles:   DefaultListStyles(),
	}
	l.filteredItems = items
	l.keyMap.Set("up", NewBinding(KeyUp, HelpText{Key: "↑", Desc: "up"}))
	l.keyMap.Set("down", NewBinding(KeyDown, HelpText{Key: "↓", Desc: "down"}))
	l.keyMap.Set("enter", NewBinding(KeyEnter, HelpText{Key: "enter", Desc: "select"}))
	l.keyMap.Set("filter", NewBinding(KeyCtrlF, HelpText{Key: "/", Desc: "filter"}))
	return l
}

func (l *List) SetDelegate(d ListItemDelegate) { l.mu.Lock(); l.delegate = d; l.mu.Unlock() }
func (l *List) SetSize(w, h int)               { l.mu.Lock(); l.width = w; l.height = h; l.mu.Unlock() }
func (l *List) Title(t string)                  { l.mu.Lock(); l.title = t; l.mu.Unlock() }
func (l *List) OnSelect(fn func(int, ListItem)) { l.mu.Lock(); l.onSelect = fn; l.mu.Unlock() }

func (l *List) SetItems(items []ListItem) {
	l.mu.Lock()
	l.items = items
	l.filteredItems = items
	l.selected = 0
	l.cursor = 0
	l.mu.Unlock()
}

func (l *List) SelectedItem() (ListItem, int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.selected >= 0 && l.selected < len(l.filteredItems) {
		return l.filteredItems[l.selected], l.selected
	}
	return nil, -1
}

func (l *List) FilterState() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.filtering {
		return l.filter
	}
	return ""
}

func (l *List) SetFilter(f string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.filter = f
	if f == "" {
		l.filteredItems = l.items
	} else {
		l.filteredItems = nil
		for _, item := range l.items {
			if strings.Contains(strings.ToLower(item.FilterValue()), strings.ToLower(f)) {
				l.filteredItems = append(l.filteredItems, item)
			}
		}
	}
	l.selected = 0
	l.cursor = 0
}

func (l *List) HandleEvent(e Event) {
	if e.Type != EventKeyPress {
		return
	}
	ke, ok := e.Data.(KeyEvent)
	if !ok {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.filtering {
		switch ke.Key {
		case KeyEsc:
			l.filtering = false
			l.filter = ""
			l.filteredItems = l.items
			l.selected = 0
			l.cursor = 0
			return
		case KeyEnter:
			l.filtering = false
			return
		case KeyBack:
			if len(l.filter) > 0 {
				l.filter = l.filter[:len(l.filter)-1]
				l.SetFilter(l.filter)
			}
			return
		default:
			if len(ke.Runes) > 0 && !ke.Ctrl && !ke.Alt {
				l.filter += string(ke.Runes)
				l.SetFilter(l.filter)
			}
			return
		}
	}

	switch ke.Key {
	case KeyUp:
		if l.selected > 0 {
			l.selected--
			if l.cursor > 0 {
				l.cursor--
			}
		}
	case KeyDown:
		if l.selected < len(l.filteredItems)-1 {
			l.selected++
			if l.cursor < l.height-1 {
				l.cursor++
			}
		}
	case KeyEnter:
		if l.onSelect != nil && l.selected >= 0 && l.selected < len(l.filteredItems) {
			l.onSelect(l.selected, l.filteredItems[l.selected])
		}
	case KeyCtrlF:
		l.filtering = true
		l.filter = ""
	}
}

func (l *List) Render() string {
	l.mu.Lock()
	defer l.mu.Unlock()

	var out strings.Builder

	if l.title != "" {
		out.WriteString(l.styles.Title.Apply(" "+l.title) + "\n")
	}

	if l.filtering {
		out.WriteString(l.styles.FilterPrompt.Apply(" Filter: "+l.filter) + "\n")
	}

	start := l.selected - l.cursor
	if start < 0 {
		start = 0
	}
	end := start + l.height
	if end > len(l.filteredItems) {
		end = len(l.filteredItems)
	}

	vis := l.delegate.Height()
	for i := start; i < end; i++ {
		selected := i == l.selected
		line := l.delegate.Render(l.width, l.filteredItems[i], i, selected)
		out.WriteString(line)
		for j := 1; j < vis; j++ {
			out.WriteString("\n")
		}
		if l.delegate.Spacing() > 0 {
			for j := 0; j < l.delegate.Spacing(); j++ {
				out.WriteString("\n")
			}
		}
	}

	if len(l.filteredItems) == 0 {
		out.WriteString(l.styles.StatusBar.Apply(" No items"))
	}

	paginate := fmt.Sprintf(" %d/%d", l.selected+1, len(l.filteredItems))
	out.WriteString("\n" + l.styles.Pagination.Apply(paginate))

	return out.String()
}
