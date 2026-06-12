package mofu

import (
	"fmt"
	"sync"
	"time"
)

type Binding struct {
	Key      Key
	Runes    string
	Alt      bool
	Ctrl     bool
	Shift    bool
	Disabled bool
	Help     HelpText
}

type HelpText struct {
	Key  string
	Desc string
}

func NewBinding(key Key, help HelpText) *Binding {
	return &Binding{Key: key, Help: help}
}

func (b *Binding) SetEnabled(v bool) { b.Disabled = v }
func (b *Binding) Enabled() bool     { return !b.Disabled }

func (b *Binding) Matches(e Event) bool {
	if b.Disabled {
		return false
	}
	if e.Type != EventKeyPress {
		return false
	}
	ke, ok := e.Data.(KeyEvent)
	if !ok {
		return false
	}
	if b.Runes != "" {
		r := string(ke.Runes)
		return r == b.Runes && !ke.Ctrl && !ke.Alt
	}
	return ke.Key == b.Key && ke.Ctrl == b.Ctrl && ke.Alt == b.Alt && ke.Shift == b.Shift
}

type KeyMap struct {
	mu       sync.RWMutex
	bindings map[string]*Binding
	order    []string
}

func NewKeyMap() *KeyMap {
	return &KeyMap{
		bindings: make(map[string]*Binding),
	}
}

func (km *KeyMap) Set(name string, b *Binding) {
	km.mu.Lock()
	defer km.mu.Unlock()
	if _, exists := km.bindings[name]; !exists {
		km.order = append(km.order, name)
	}
	km.bindings[name] = b
}

func (km *KeyMap) Get(name string) *Binding {
	km.mu.RLock()
	defer km.mu.RUnlock()
	return km.bindings[name]
}

func (km *KeyMap) ShortHelp() []Binding {
	km.mu.RLock()
	defer km.mu.RUnlock()
	var out []Binding
	for _, name := range km.order {
		b := km.bindings[name]
		if b != nil && !b.Disabled {
			out = append(out, *b)
			if len(out) >= 3 {
				break
			}
		}
	}
	return out
}

func (km *KeyMap) FullHelp() [][]Binding {
	km.mu.RLock()
	defer km.mu.RUnlock()
	var out []Binding
	for _, name := range km.order {
		b := km.bindings[name]
		if b != nil && !b.Disabled {
			out = append(out, *b)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return [][]Binding{out}
}

func (km *KeyMap) Help() string {
	km.mu.RLock()
	defer km.mu.RUnlock()
	var s string
	for _, name := range km.order {
		b := km.bindings[name]
		if b != nil && !b.Disabled {
			s += fmt.Sprintf(" %s %s", b.Help.Key, b.Help.Desc)
		}
	}
	return s
}

func (km *KeyMap) Matches(e Event) (string, bool) {
	km.mu.RLock()
	defer km.mu.RUnlock()
	for _, name := range km.order {
		b := km.bindings[name]
		if b != nil && b.Matches(e) {
			return name, true
		}
	}
	return "", false
}

func (km *KeyMap) Bindings() []*Binding {
	km.mu.RLock()
	defer km.mu.RUnlock()
	var out []*Binding
	for _, name := range km.order {
		out = append(out, km.bindings[name])
	}
	return out
}

type HelpView struct {
	ShowAll    bool
	ShortHelp  func() []Binding
	FullHelp   func() [][]Binding
	Styles     HelpStyles
}

type HelpStyles struct {
	ShortKey   Style
	ShortDesc  Style
	FullKey    Style
	FullDesc   Style
	Separator  Style
}

func DefaultHelpStyles() HelpStyles {
	return HelpStyles{
		ShortKey:  DefaultStyle().Fg(Hex("89b4fa")).WithAttrs(AttrBold),
		ShortDesc: DefaultStyle().Fg(Hex("6c7086")),
		FullKey:   DefaultStyle().Fg(Hex("89b4fa")).WithAttrs(AttrBold),
		FullDesc:  DefaultStyle().Fg(Hex("6c7086")),
		Separator: DefaultStyle().Fg(Hex("45475a")),
	}
}

func NewHelpView(km *KeyMap) *HelpView {
	return &HelpView{
		ShortHelp: func() []Binding { return km.ShortHelp() },
		FullHelp:  func() [][]Binding { return km.FullHelp() },
		Styles:    DefaultHelpStyles(),
	}
}

func (hv *HelpView) Render(w, h int) string {
	if hv.ShowAll {
		return hv.renderFull(w, h)
	}
	return hv.renderShort(w)
}

func (hv *HelpView) renderShort(w int) string {
	bindings := hv.ShortHelp()
	if len(bindings) == 0 {
		return ""
	}
	var out string
	for i, b := range bindings {
		if i > 0 {
			out += hv.Styles.Separator.Apply(" · ")
		}
		out += hv.Styles.ShortKey.Apply(" "+b.Help.Key) + " " + hv.Styles.ShortDesc.Apply(b.Help.Desc)
	}
	return out
}

func (hv *HelpView) renderFull(w, h int) string {
	groups := hv.FullHelp()
	if len(groups) == 0 {
		return ""
	}
	var out string
	for _, group := range groups {
		for _, b := range group {
			out += hv.Styles.FullKey.Apply(" "+b.Help.Key) + "  " + hv.Styles.FullDesc.Apply(b.Help.Desc) + "\n"
		}
	}
	return out
}

type KeyPressTracker struct {
	mu        sync.Mutex
	keys      []KeyEvent
	startTime time.Time
	threshold time.Duration
	callback  func(string)
}

func NewKeyPressTracker(threshold time.Duration) *KeyPressTracker {
	return &KeyPressTracker{
		startTime: time.Now(),
		threshold: threshold,
	}
}

func (t *KeyPressTracker) OnComplete(fn func(string)) {
	t.mu.Lock()
	t.callback = fn
	t.mu.Unlock()
}

func (t *KeyPressTracker) Feed(e KeyEvent) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	if now.Sub(t.startTime) > t.threshold {
		t.keys = t.keys[:0]
	}
	t.startTime = now

	if len(e.Runes) > 0 {
		t.keys = append(t.keys, e)
	}

	if e.Key == KeyEnter && len(t.keys) > 0 {
		var s string
		for _, k := range t.keys {
			s += string(k.Runes)
		}
		t.keys = t.keys[:0]
		if t.callback != nil {
			t.callback(s)
		}
	}
}

func (t *KeyPressTracker) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.keys = t.keys[:0]
	t.startTime = time.Now()
}

func (t *KeyPressTracker) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	var s string
	for _, k := range t.keys {
		s += string(k.Runes)
	}
	return s
}
