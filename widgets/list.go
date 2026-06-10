package widgets

import (
	"fmt"
	"strings"

	"github.com/anomalyco/mofu"
)

type ListItem struct {
	Title    string
	Subtitle string
	Data     any
}

type ListNode struct {
	mofu.BaseNode
	Items    []ListItem
	Selected int
	Offset   int
	Title    string
	OnSelect func(item ListItem) mofu.Cmd
}

func NewList(items []ListItem) *ListNode {
	return &ListNode{Items: items}
}

func (l *ListNode) SelectedItem() *ListItem {
	if l.Selected >= 0 && l.Selected < len(l.Items) {
		return &l.Items[l.Selected]
	}
	return nil
}

func (l *ListNode) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	var result strings.Builder
	if l.Title != "" {
		result.WriteString(l.Title)
		result.WriteByte('\n')
	}
	start := l.Offset
	height := r.Height
	if l.Title != "" {
		height--
	}
	end := start + height
	if end > len(l.Items) {
		end = len(l.Items)
	}
	for i := start; i < end; i++ {
		item := l.Items[i]
		if i == l.Selected {
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
	ctx.Renderer.WriteStyledString(result.String(), r.X, r.Y, *l.Style())
}

func (l *ListNode) HandleEvent(event mofu.Event) mofu.Cmd {
	if event.Type != mofu.EventKeyPress {
		return nil
	}
	ke, ok := event.Data.(mofu.KeyEvent)
	if !ok {
		return nil
	}
	for _, b := range ke.Runes {
		switch {
		case b == 'j' || ke.Key == mofu.KeyDown:
			if l.Selected < len(l.Items)-1 {
				l.Selected++
				if l.Selected >= l.Offset+(l.Bounds().Height) {
					l.Offset++
				}
			}
		case b == 'k' || ke.Key == mofu.KeyUp:
			if l.Selected > 0 {
				l.Selected--
				if l.Selected < l.Offset {
					l.Offset--
				}
			}
		case b == '\r', b == '\n':
			if l.OnSelect != nil && l.Selected >= 0 && l.Selected < len(l.Items) {
				return l.OnSelect(l.Items[l.Selected])
			}
		}
	}
	return nil
}

func (l *ListNode) Mount() mofu.Cmd { return nil }
func (l *ListNode) Unmount()        {}
