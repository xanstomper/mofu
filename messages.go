package mofu

import "strings"

type FocusMsg struct{}

type BlurMsg struct{}

type PasteStartMsg struct{}

type PasteEndMsg struct{}

type PasteMsg struct {
	Content string
}

func (p PasteMsg) String() string {
	return p.Content
}

type ColorProfileRequestMsg struct{}

type TerminalVersionMsg struct {
	Name string
}

func (t TerminalVersionMsg) String() string {
	return t.Name
}

type KeyboardEnhancementsMsg struct {
	Flags int
}

func (k KeyboardEnhancementsMsg) SupportsKeyDisambiguation() bool {
	return k.Flags > 0
}

type ModeReportMsg struct {
	Mode  int
	Value bool
}

type ModifierKey uint32

const (
	ModShift ModifierKey = 1 << iota
	ModAlt
	ModCtrl
	ModMeta
	ModHyper
	ModSuper
	ModCapsLock
	ModNumLock
	ModScrollLock
)

type ExecCommand interface {
	Run() error
	SetStdin(r interface{ Read([]byte) (int, error) })
	SetStdout(w interface{ Write([]byte) (int, error) })
	SetStderr(w interface{ Write([]byte) (int, error) })
}

type ExecCallback func(error) Msg

func RequestWindowSize() Msg {
	return WindowSizeMsg{}
}

type windowSizeMsg2 struct{}

func RequestTerminalVersion() Msg {
	return TerminalVersionMsg{}
}

type terminalVersionMsg struct{}

func RequestColorProfile() Msg {
	return ColorProfileRequestMsg{}
}

type CapabilityMsg struct {
	Name  string
	Value string
}

type RawMsg2 struct {
	Seq string
}

func (r RawMsg2) String() string {
	return r.Seq
}

type LogMsg struct {
	Text string
}

type PrintfMsg struct {
	Format string
	Args   []any
}

type PrintMsg struct {
	Text string
}

type printLineMsg struct {
	Body string
}

func compactCmds2(cmds []Cmd) Cmd {
	valid := make([]Cmd, 0, len(cmds))
	for _, c := range cmds {
		if c != nil {
			valid = append(valid, c)
		}
	}
	switch len(valid) {
	case 0:
		return nil
	case 1:
		return valid[0]
	default:
		return func() Msg {
			return BatchMsg(valid)
		}
	}
}

func SequenceCmds(cmds ...Cmd) Cmd {
	valid := make([]Cmd, 0, len(cmds))
	for _, c := range cmds {
		if c != nil {
			valid = append(valid, c)
		}
	}
	switch len(valid) {
	case 0:
		return nil
	case 1:
		return valid[0]
	default:
		return func() Msg {
			return SequenceMsg(valid)
		}
	}
}

func (ke KeyEvent) Modifiers() ModifierKey {
	var mod ModifierKey
	if ke.Ctrl {
		mod |= ModCtrl
	}
	if ke.Alt {
		mod |= ModAlt
	}
	if ke.Shift {
		mod |= ModShift
	}
	return mod
}

func (ke KeyEvent) HasMod(mod ModifierKey) bool {
	return ke.Modifiers()&mod != 0
}

func (ke KeyEvent) Rune() rune {
	if len(ke.Runes) > 0 {
		return rune(ke.Runes[0])
	}
	return 0
}

type ScrollUpMsg struct {
	X, Y int
}

type ScrollDownMsg struct {
	X, Y int
}

type MouseClickMsg struct {
	X, Y   int
	Button MouseButton
}

type MouseReleaseMsg struct {
	X, Y   int
	Button MouseButton
}

type MouseMotionMsg struct {
	X, Y   int
	Button MouseButton
}

func (m MouseClickMsg) String() string {
	return "click"
}

func (m MouseReleaseMsg) String() string {
	return "release"
}

func (m MouseMotionMsg) String() string {
	return "motion"
}

type KeyMsg interface {
	Key() Key
	String() string
}

type KeyPressMsg struct {
	ke KeyEvent
}

func (k KeyPressMsg) Key() Key {
	return k.ke.Key
}

func (k KeyPressMsg) String() string {
	return k.ke.String()
}

type KeyReleaseMsg struct {
	ke KeyEvent
}

func (k KeyReleaseMsg) Key() Key {
	return k.ke.Key
}

func (k KeyReleaseMsg) String() string {
	return k.ke.String()
}

type ExecProcessMsg struct {
	Cmd string
	Fn  ExecCallback
}

type TextInputSuggestionMsg struct {
	Suggestions []string
}

type ListFilterMatchesMsg struct {
	Matches []int
}

func (l ListFilterMatchesMsg) String() string {
	return strings.Repeat("x", len(l.Matches))
}

type TableKeyMap struct {
	LineUp       *Binding
	LineDown     *Binding
	PageUp       *Binding
	PageDown     *Binding
	HalfPageUp   *Binding
	HalfPageDown *Binding
	GotoTop      *Binding
	GotoBottom   *Binding
}

func DefaultTableKeyMap() TableKeyMap {
	return TableKeyMap{
		LineUp:       NewBinding(KeyUp, HelpText{Key: "↑", Desc: "up"}),
		LineDown:     NewBinding(KeyDown, HelpText{Key: "↓", Desc: "down"}),
		PageUp:       NewBinding(KeyPgUp, HelpText{Key: "pgup", Desc: "page up"}),
		PageDown:     NewBinding(KeyPgDn, HelpText{Key: "pgdn", Desc: "page down"}),
		HalfPageUp:   NewBinding(KeyCtrlU, HelpText{Key: "ctrl+u", Desc: "½ page up"}),
		HalfPageDown: NewBinding(KeyCtrlD, HelpText{Key: "ctrl+d", Desc: "½ page down"}),
		GotoTop:      NewBinding(KeyHome, HelpText{Key: "home", Desc: "go to start"}),
		GotoBottom:   NewBinding(KeyEnd, HelpText{Key: "end", Desc: "go to end"}),
	}
}

func (km TableKeyMap) ShortHelp() []Binding {
	return []Binding{*km.LineUp, *km.LineDown}
}

func (km TableKeyMap) FullHelp() [][]Binding {
	return [][]Binding{
		{*km.LineUp, *km.LineDown, *km.GotoTop, *km.GotoBottom},
		{*km.PageUp, *km.PageDown, *km.HalfPageUp, *km.HalfPageDown},
	}
}

type ViewportKeyMap struct {
	LineUp       *Binding
	LineDown     *Binding
	PageUp       *Binding
	PageDown     *Binding
	HalfPageUp   *Binding
	HalfPageDown *Binding
	GotoTop      *Binding
	GotoBottom   *Binding
}

func DefaultViewportKeyMap() ViewportKeyMap {
	return ViewportKeyMap{
		LineUp:       NewBinding(KeyUp, HelpText{Key: "↑", Desc: "up"}),
		LineDown:     NewBinding(KeyDown, HelpText{Key: "↓", Desc: "down"}),
		PageUp:       NewBinding(KeyPgUp, HelpText{Key: "pgup", Desc: "page up"}),
		PageDown:     NewBinding(KeyPgDn, HelpText{Key: "pgdn", Desc: "page down"}),
		HalfPageUp:   NewBinding(KeyCtrlU, HelpText{Key: "ctrl+u", Desc: "½ page up"}),
		HalfPageDown: NewBinding(KeyCtrlD, HelpText{Key: "ctrl+d", Desc: "½ page down"}),
		GotoTop:      NewBinding(KeyHome, HelpText{Key: "home", Desc: "go to start"}),
		GotoBottom:   NewBinding(KeyEnd, HelpText{Key: "end", Desc: "go to end"}),
	}
}

func (km ViewportKeyMap) ShortHelp() []Binding {
	return []Binding{*km.LineUp, *km.LineDown}
}

func (km ViewportKeyMap) FullHelp() [][]Binding {
	return [][]Binding{
		{*km.LineUp, *km.LineDown, *km.GotoTop, *km.GotoBottom},
		{*km.PageUp, *km.PageDown, *km.HalfPageUp, *km.HalfPageDown},
	}
}

type ListKeyMap struct {
	LineUp       *Binding
	LineDown     *Binding
	PageUp       *Binding
	PageDown     *Binding
	HalfPageUp   *Binding
	HalfPageDown *Binding
	GotoTop      *Binding
	GotoBottom   *Binding
	Filter       *Binding
	Accept       *Binding
	Cancel       *Binding
}

func DefaultListKeyMap() ListKeyMap {
	return ListKeyMap{
		LineUp:       NewBinding(KeyUp, HelpText{Key: "↑", Desc: "up"}),
		LineDown:     NewBinding(KeyDown, HelpText{Key: "↓", Desc: "down"}),
		PageUp:       NewBinding(KeyPgUp, HelpText{Key: "pgup", Desc: "page up"}),
		PageDown:     NewBinding(KeyPgDn, HelpText{Key: "pgdn", Desc: "page down"}),
		HalfPageUp:   NewBinding(KeyCtrlU, HelpText{Key: "ctrl+u", Desc: "½ page up"}),
		HalfPageDown: NewBinding(KeyCtrlD, HelpText{Key: "ctrl+d", Desc: "½ page down"}),
		GotoTop:      NewBinding(KeyHome, HelpText{Key: "home", Desc: "go to start"}),
		GotoBottom:   NewBinding(KeyEnd, HelpText{Key: "end", Desc: "go to end"}),
		Filter:       NewBinding(KeyCtrlF, HelpText{Key: "/", Desc: "filter"}),
		Accept:       NewBinding(KeyEnter, HelpText{Key: "enter", Desc: "accept"}),
		Cancel:       NewBinding(KeyEsc, HelpText{Key: "esc", Desc: "cancel"}),
	}
}

func (km ListKeyMap) ShortHelp() []Binding {
	return []Binding{*km.LineUp, *km.LineDown, *km.Accept}
}

func (km ListKeyMap) FullHelp() [][]Binding {
	return [][]Binding{
		{*km.LineUp, *km.LineDown, *km.GotoTop, *km.GotoBottom},
		{*km.PageUp, *km.PageDown, *km.HalfPageUp, *km.HalfPageDown},
		{*km.Filter, *km.Accept, *km.Cancel},
	}
}
