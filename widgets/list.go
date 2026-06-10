package widgets

import (
	"fmt"
	"strings"

	"github.com/anomalyco/mofu"
)

// ListItem is a single item in a List.
type ListItem struct {
	Title    string
	Subtitle string
	Data     any
}

// List is a selectable list component.
type List struct {
	items    []ListItem
	selected int
	offset   int
	height   int
	title    string
	OnSelect func(item ListItem) mofu.Cmd
}

// NewList creates a new list component.
func NewList(items []ListItem) *List {
	return &List{
		items:  items,
		height: 10,
	}
}

// SetTitle sets the list title.
func (l *List) SetTitle(t string) *List {
	l.title = t
	return l
}

// SetHeight sets the visible height.
func (l *List) SetHeight(h int) *List {
	l.height = h
	return l
}

// Selected returns the selected item.
func (l *List) Selected() *ListItem {
	if l.selected >= 0 && l.selected < len(l.items) {
		return &l.items[l.selected]
	}
	return nil
}

// SelectedIndex returns the selected index.
func (l *List) SelectedIndex() int { return l.selected }

func (l *List) Render() string {
	var result strings.Builder
	if l.title != "" {
		result.WriteString(l.title)
		result.WriteByte('\n')
	}

	start := l.offset
	end := start + l.height
	if end > len(l.items) {
		end = len(l.items)
	}

	for i := start; i < end; i++ {
		item := l.items[i]
		if i == l.selected {
			result.WriteString(" → ")
		} else {
			result.WriteString("   ")
		}
		result.WriteString(item.Title)
		if item.Subtitle != "" {
			result.WriteString(fmt.Sprintf("  (%s)", item.Subtitle))
		}
		if i < end-1 {
			result.WriteByte('\n')
		}
	}
	return result.String()
}

func (l *List) HandleEvent(msg mofu.Msg) mofu.Cmd {
	switch msg := msg.(type) {
	case mofu.KeyPressMsg:
		for _, b := range msg.Runes {
			switch {
			case b == 'j' || msg.Key == mofu.KeyDown:
				if l.selected < len(l.items)-1 {
					l.selected++
					if l.selected >= l.offset+l.height {
						l.offset++
					}
				}
			case b == 'k' || msg.Key == mofu.KeyUp:
				if l.selected > 0 {
					l.selected--
					if l.selected < l.offset {
						l.offset--
					}
				}
			case b == '\r', b == '\n':
				if l.OnSelect != nil && l.selected >= 0 && l.selected < len(l.items) {
					return l.OnSelect(l.items[l.selected])
				}
			}
		}
	}
	return nil
}

func (l *List) Mount() mofu.Cmd { return nil }
func (l *List) Unmount()        {}
