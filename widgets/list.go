package widgets

import (
	"strings"

	"github.com/xanstomper/mofu"
)

type ListItem struct {
	Title    string
	Subtitle string
	Data     any
}

type ListNode struct {
	mofu.BaseNode
	Items         []ListItem
	Selected      int
	Offset        int
	Title         string
	ItemRenderer  func(item ListItem, index int, selected bool) string
	OnSelect      func(item ListItem) mofu.Cmd
	selectedStyle mofu.Style
}

func NewList(items []ListItem) *ListNode {
	l := &ListNode{Items: items, selectedStyle: mofu.Style{Attrs: mofu.AttrBold}}
	l.clamp()
	return l
}

func (l *ListNode) SetItems(items []ListItem) {
	l.Items = items
	l.clamp()
	l.SetDirty()
}

func (l *ListNode) SetSelected(index int) {
	if l.Selected == index {
		return
	}
	l.Selected = index
	l.clamp()
	l.SetDirty()
}

func (l *ListNode) SetOffset(offset int) {
	l.Offset = offset
	l.clamp()
	l.SetDirty()
}

func (l *ListNode) SetSelectedStyle(style mofu.Style) {
	l.selectedStyle = style
	l.SetDirty()
}

func (l *ListNode) SelectedItem() *ListItem {
	if l.Selected >= 0 && l.Selected < len(l.Items) {
		return &l.Items[l.Selected]
	}
	return nil
}

func (l *ListNode) clamp() {
	if len(l.Items) == 0 {
		l.Selected = -1
		l.Offset = 0
		return
	}
	if l.Selected < 0 {
		l.Selected = 0
	}
	if l.Selected >= len(l.Items) {
		l.Selected = len(l.Items) - 1
	}
	if l.Offset < 0 {
		l.Offset = 0
	}
	if l.Offset > l.Selected {
		l.Offset = l.Selected
	}
	if l.Offset+l.visibleHeight() <= l.Selected {
		l.Offset = l.Selected - l.visibleHeight() + 1
	}
	if l.Offset < 0 {
		l.Offset = 0
	}
}

func (l *ListNode) visibleHeight() int {
	h := l.Bounds().Height
	if h <= 0 {
		return 1
	}
	if l.Title != "" && h > 1 {
		return h - 1
	}
	return h
}

func (l *ListNode) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	if r.Width <= 0 || r.Height <= 0 {
		return
	}
	l.clamp()

	sy := r.Y
	if l.Title != "" {
		ctx.Renderer.WriteString(mofu.Truncate(l.Title, r.Width, true), r.X, sy, l.Style().Foreground, l.Style().Background, l.Style().Attrs)
		sy++
	}
	if sy >= r.Y+r.Height {
		return
	}

	height := r.Y + r.Height - sy
	start := l.Offset
	if start < 0 {
		start = 0
	}
	if start > len(l.Items)-height {
		start = len(l.Items) - height
	}
	if start < 0 {
		start = 0
	}
	end := start + height
	if end > len(l.Items) {
		end = len(l.Items)
	}

	base := *l.Style()
	selected := l.selectedStyle
	for i := start; i < end; i++ {
		item := l.Items[i]
		line := l.renderItem(item, i, i == l.Selected, r.Width)
		style := base
		if i == l.Selected {
			style = selected
			if style.Foreground == mofu.ColorTransparent {
				style.Foreground = base.Foreground
			}
			if style.Background == mofu.ColorTransparent {
				style.Background = base.Background
			}
		}
		ctx.Renderer.WriteString(line, r.X, sy, style.Foreground, style.Background, style.Attrs)
		sy++
	}
}

func (l *ListNode) renderItem(item ListItem, index int, selected bool, width int) string {
	if l.ItemRenderer != nil {
		return mofu.Truncate(l.ItemRenderer(item, index, selected), width, true)
	}
	var b strings.Builder
	if selected {
		b.WriteString("› ")
	} else {
		b.WriteString("  ")
	}
	b.WriteString(item.Title)
	if item.Subtitle != "" {
		b.WriteString("  (")
		b.WriteString(item.Subtitle)
		b.WriteByte(')')
	}
	return mofu.Truncate(b.String(), width, true)
}

func (l *ListNode) HandleEvent(event mofu.Event) mofu.Cmd {
	if event.Type != mofu.EventKeyPress {
		return nil
	}
	ke, ok := event.Data.(mofu.KeyEvent)
	if !ok {
		return nil
	}

	switch ke.Key {
	case mofu.KeyDown:
		if l.Selected < len(l.Items)-1 {
			l.Selected++
			l.clamp()
			l.SetDirty()
		}
		return nil
	case mofu.KeyUp:
		if l.Selected > 0 {
			l.Selected--
			l.clamp()
			l.SetDirty()
		}
		return nil
	case mofu.KeyPgDn:
		l.Selected += l.visibleHeight()
		l.clamp()
		l.SetDirty()
		return nil
	case mofu.KeyPgUp:
		l.Selected -= l.visibleHeight()
		l.clamp()
		l.SetDirty()
		return nil
	case mofu.KeyHome:
		l.Selected = 0
		l.clamp()
		l.SetDirty()
		return nil
	case mofu.KeyEnd:
		l.Selected = len(l.Items) - 1
		l.clamp()
		l.SetDirty()
		return nil
	}

	for _, b := range ke.Runes {
		switch b {
		case 'j':
			if l.Selected < len(l.Items)-1 {
				l.Selected++
				l.clamp()
				l.SetDirty()
			}
		case 'k':
			if l.Selected > 0 {
				l.Selected--
				l.clamp()
				l.SetDirty()
			}
		case '\r', '\n':
			if l.OnSelect != nil && l.Selected >= 0 && l.Selected < len(l.Items) {
				return l.OnSelect(l.Items[l.Selected])
			}
		}
	}
	return nil
}

func (l *ListNode) Mount() mofu.Cmd { return nil }
func (l *ListNode) Unmount()        {}
