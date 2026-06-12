package mofu

import (
	"fmt"
	"strings"
	"sync"
)

type PaginatorType int

const (
	PaginatorDots PaginatorType = iota
	PaginatorArabic
	PaginatorAlphabet
)

type Paginator struct {
	mu        sync.Mutex
	Page      int
	PerPage   int
	Total     int
	Type      PaginatorType
	styles    PaginatorStyles
	focused   bool
	inactive  Style
	active    Style
}

type PaginatorStyles struct {
	Active   Style
	Inactive Style
}

func DefaultPaginatorStyles() PaginatorStyles {
	return PaginatorStyles{
		Active:   DefaultStyle().Fg(Hex("89b4fa")).WithAttrs(AttrBold),
		Inactive: DefaultStyle().Fg(Hex("585b70")),
	}
}

func NewPaginator() Paginator {
	return Paginator{
		PerPage: 10,
		Type:    PaginatorDots,
		styles:  DefaultPaginatorStyles(),
	}
}

func (p *Paginator) SetTotal(total int) {
	p.mu.Lock()
	p.Total = total
	p.mu.Unlock()
}

func (p *Paginator) PageActive() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.PerPage > 0 && p.Total > p.PerPage
}

func (p *Paginator) PrevPage() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.Page > 0 {
		p.Page--
	}
}

func (p *Paginator) NextPage() {
	p.mu.Lock()
	defer p.mu.Unlock()
	maxPage := p.Total/p.PerPage
	if p.Total%p.PerPage == 0 {
		maxPage--
	}
	if p.Page < maxPage {
		p.Page++
	}
}

func (p *Paginator) GotoPage(page int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if page >= 0 {
		maxPage := p.Total / p.PerPage
		if page > maxPage {
			page = maxPage
		}
		p.Page = page
	}
}

func (p *Paginator) Render() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.PageActive() {
		return ""
	}

	switch p.Type {
	case PaginatorArabic:
		return p.renderArabic()
	case PaginatorAlphabet:
		return p.renderAlphabet()
	default:
		return p.renderDots()
	}
}

func (p *Paginator) renderDots() string {
	total := p.Total / p.PerPage
	if p.Total%p.PerPage > 0 {
		total++
	}
	var out string
	for i := 0; i <= total; i++ {
		if i == p.Page {
			out += p.styles.Active.Apply(" ● ")
		} else {
			out += p.styles.Inactive.Apply(" ○ ")
		}
	}
	return out
}

func (p *Paginator) renderArabic() string {
	total := p.Total / p.PerPage
	if p.Total%p.PerPage > 0 {
		total++
	}
	return fmt.Sprintf("%s / %s",
		p.styles.Active.Apply(fmt.Sprintf("%d", p.Page+1)),
		p.styles.Inactive.Apply(fmt.Sprintf("%d", total+1)),
	)
}

func (p *Paginator) renderAlphabet() string {
	total := p.Total / p.PerPage
	if p.Total%p.PerPage > 0 {
		total++
	}
	var out string
	for i := 0; i <= total; i++ {
		letter := string(rune('A' + i))
		if i == p.Page {
			out += p.styles.Active.Apply(" " + letter + " ")
		} else {
			out += p.styles.Inactive.Apply(" " + letter + " ")
		}
	}
	return out
}

type HelpComponent struct {
	mu              sync.Mutex
	ShowAll         bool
	ShortSeparator  string
	FullSeparator   string
	Ellipsis        string
	Styles          HelpStyles
	width           int
}

func NewHelpComponent() *HelpComponent {
	return &HelpComponent{
		ShortSeparator: " • ",
		FullSeparator:  "    ",
		Ellipsis:       "…",
		Styles:         DefaultHelpStyles(),
	}
}

func (h *HelpComponent) SetWidth(w int) {
	h.mu.Lock()
	h.width = w
	h.mu.Unlock()
}

func (h *HelpComponent) Width() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.width
}

func (h *HelpComponent) View(km *KeyMap) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.ShowAll {
		return h.fullHelpView(km)
	}
	return h.shortHelpView(km)
}

func (h *HelpComponent) shortHelpView(km *KeyMap) string {
	bindings := km.ShortHelp()
	if len(bindings) == 0 {
		return ""
	}
	var parts []string
	for _, b := range bindings {
		if !b.Enabled() {
			continue
		}
		parts = append(parts, h.Styles.ShortKey.Apply(" "+b.Help.Key)+" "+h.Styles.ShortDesc.Apply(b.Help.Desc))
	}
		result := strings.Join(parts, h.Styles.Separator.Apply(h.ShortSeparator))

		if h.width > 0 && len(result) > h.width {
		return result[:h.width-1] + h.Styles.ShortDesc.Apply(h.Ellipsis)
	}
	return result
}

func (h *HelpComponent) fullHelpView(km *KeyMap) string {
	groups := km.FullHelp()
	if len(groups) == 0 {
		return ""
	}
	var out string
	for i, group := range groups {
		var parts []string
		for _, b := range group {
			if !b.Enabled() {
				continue
			}
			parts = append(parts, h.Styles.FullKey.Apply(" "+b.Help.Key)+"  "+h.Styles.FullDesc.Apply(b.Help.Desc))
		}
		out += strings.Join(parts, h.Styles.Separator.Apply(h.FullSeparator))
		if i < len(groups)-1 {
			out += "\n"
		}
	}
	return out
}

type PaginatorKeyMap struct {
	NextPage *Binding
	PrevPage *Binding
}

func DefaultPaginatorKeyMap() PaginatorKeyMap {
	return PaginatorKeyMap{
		NextPage: NewBinding(KeyRight, HelpText{Key: "→", Desc: "next page"}),
		PrevPage: NewBinding(KeyLeft, HelpText{Key: "←", Desc: "prev page"}),
	}
}
