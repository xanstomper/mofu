package mofu

import (
	"strings"
	"sync"
)

type EchoMode int

const (
	EchoNormal EchoMode = iota
	EchoPassword
	EchoNone
)

type ValidateFunc func(string) error

type TextInputKeyMap struct {
	CharacterForward        *Binding
	CharacterBackward       *Binding
	WordForward             *Binding
	WordBackward            *Binding
	DeleteWordBackward      *Binding
	DeleteWordForward       *Binding
	DeleteAfterCursor       *Binding
	DefaultTextBeforeCursor *Binding
	DeleteCharacterBackward *Binding
	DeleteCharacterForward  *Binding
	LineStart               *Binding
	LineEnd                 *Binding
	Paste                   *Binding
	AcceptSuggestion        *Binding
	NextSuggestion          *Binding
	PrevSuggestion          *Binding
}

func DefaultTextInputKeyMap() TextInputKeyMap {
	return TextInputKeyMap{
		CharacterForward:        NewBinding(KeyRight, HelpText{Key: "→", Desc: "forward"}),
		CharacterBackward:       NewBinding(KeyLeft, HelpText{Key: "←", Desc: "backward"}),
		WordForward:             NewBinding(KeyNone, HelpText{Key: "alt+→", Desc: "word forward"}),
		WordBackward:            NewBinding(KeyNone, HelpText{Key: "alt+←", Desc: "word backward"}),
		DeleteWordBackward:      NewBinding(KeyNone, HelpText{Key: "alt+⌫", Desc: "delete word"}),
		DeleteAfterCursor:       NewBinding(KeyCtrlK, HelpText{Key: "ctrl+k", Desc: "delete line"}),
		DefaultTextBeforeCursor: NewBinding(KeyCtrlU, HelpText{Key: "ctrl+u", Desc: "delete before"}),
		DeleteCharacterBackward: NewBinding(KeyBack, HelpText{Key: "⌫", Desc: "delete backward"}),
		DeleteCharacterForward:  NewBinding(KeyDelete, HelpText{Key: "del", Desc: "delete forward"}),
		LineStart:               NewBinding(KeyHome, HelpText{Key: "home", Desc: "start"}),
		LineEnd:                 NewBinding(KeyEnd, HelpText{Key: "end", Desc: "end"}),
		Paste:                   NewBinding(KeyCtrlV, HelpText{Key: "ctrl+v", Desc: "paste"}),
		AcceptSuggestion:        NewBinding(KeyTab, HelpText{Key: "tab", Desc: "accept"}),
		NextSuggestion:          NewBinding(KeyDown, HelpText{Key: "↓", Desc: "next"}),
		PrevSuggestion:          NewBinding(KeyUp, HelpText{Key: "↑", Desc: "prev"}),
	}
}

type TextInput struct {
	mu              sync.Mutex
	Err             error
	Prompt          string
	Placeholder     string
	EchoMode        EchoMode
	EchoCharacter   rune
	CharLimit       int
	styles          TextInputStyles
	width           int
	KeyMap          TextInputKeyMap
	value           []rune
	focus           bool
	pos             int
	offset          int
	offsetRight     int
	Validate        ValidateFunc
	ShowSuggestions bool
	suggestions     [][]rune
	matchedSugs     [][]rune
	sugIndex        int
	cursor          VirtualCursor
	onChange        func(string)
}

type VirtualCursor struct {
	X         int
	Mode      CursorMode
	Blink     bool
	BlinkOn   bool
	CharUnder string
	Style     Style
	TextStyle Style
}

type CursorMode int

const (
	CursorBlinkMode CursorMode = iota
	CursorStatic
	CursorHide
)

type TextInputStyles struct {
	Base           Style
	Focused        Style
	Placeholder    Style
	Cursor         Style
	TextInput      Style
	CursorLine     Style
	Prompt         Style
	SuggestionMatch Style
}

func DefaultTextInputStyles() TextInputStyles {
	return TextInputStyles{
		Base:            DefaultStyle().Fg(Hex("cdd6f4")),
		Focused:         DefaultStyle().Fg(Hex("cdd6f4")),
		Placeholder:     DefaultStyle().Fg(Hex("585b70")),
		Cursor:          DefaultStyle().Fg(Hex("f5c2e7")),
		TextInput:       DefaultStyle().Fg(Hex("cdd6f4")),
		CursorLine:      DefaultStyle().Fg(Hex("cdd6f4")),
		Prompt:          DefaultStyle().Fg(Hex("89b4fa")),
		SuggestionMatch: DefaultStyle().Fg(Hex("a6e3a1")).WithAttrs(AttrBold),
	}
}

func NewTextInput() TextInput {
	return TextInput{
		Prompt:          "> ",
		EchoCharacter:   '*',
		CharLimit:       0,
		styles:          DefaultTextInputStyles(),
		KeyMap:          DefaultTextInputKeyMap(),
		suggestions:     [][]rune{},
		cursor:          VirtualCursor{Mode: CursorBlinkMode, Blink: true},
	}
}

func (t *TextInput) SetWidth(w int)   { t.mu.Lock(); t.width = w; t.mu.Unlock() }
func (t *TextInput) Width() int       { t.mu.Lock(); defer t.mu.Unlock(); return t.width }
func (t *TextInput) Focus()           { t.mu.Lock(); t.focus = true; t.mu.Unlock() }
func (t *TextInput) Blur()            { t.mu.Lock(); t.focus = false; t.mu.Unlock() }
func (t *TextInput) Focused() bool    { t.mu.Lock(); defer t.mu.Unlock(); return t.focus }
func (t *TextInput) Value() string    { t.mu.Lock(); defer t.mu.Unlock(); return string(t.value) }
func (t *TextInput) Position() int    { t.mu.Lock(); defer t.mu.Unlock(); return t.pos }
func (t *TextInput) Len() int         { t.mu.Lock(); defer t.mu.Unlock(); return len(t.value) }

func (t *TextInput) SetStyles(s TextInputStyles) {
	t.mu.Lock()
	t.styles = s
	t.mu.Unlock()
}

func (t *TextInput) SetSuggestions(s []string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.suggestions = make([][]rune, len(s))
	for i, v := range s {
		t.suggestions[i] = []rune(v)
	}
}

func (t *TextInput) SetValue(s string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	runes := []rune(s)
	if t.Validate != nil {
		t.Err = t.Validate(s)
	}
	t.value = runes
	t.pos = len(runes)
	if t.onChange != nil {
		t.onChange(s)
	}
}

func (t *TextInput) OnChange(fn func(string)) {
	t.mu.Lock()
	t.onChange = fn
	t.mu.Unlock()
}

func (t *TextInput) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.value = t.value[:0]
	t.pos = 0
	t.offset = 0
	t.Err = nil
}

func (t *TextInput) HandleEvent(e Event) {
	if !t.focus || e.Type != EventKeyPress {
		return
	}
	ke, ok := e.Data.(KeyEvent)
	if !ok {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	switch {
	case ke.Key == KeyRight || (ke.Ctrl && string(ke.Runes) == "f"):
		if t.pos < len(t.value) {
			t.pos++
		}
	case ke.Key == KeyLeft || (ke.Ctrl && string(ke.Runes) == "b"):
		if t.pos > 0 {
			t.pos--
		}
	case ke.Key == KeyHome || (ke.Ctrl && string(ke.Runes) == "a"):
		t.pos = 0
	case ke.Key == KeyEnd || (ke.Ctrl && string(ke.Runes) == "e"):
		t.pos = len(t.value)
	case ke.Key == KeyBack || (ke.Ctrl && string(ke.Runes) == "h"):
		if t.pos > 0 {
			t.value = append(t.value[:t.pos-1], t.value[t.pos:]...)
			t.pos--
		}
	case ke.Key == KeyDelete || (ke.Ctrl && string(ke.Runes) == "d"):
		if t.pos < len(t.value) {
			t.value = append(t.value[:t.pos], t.value[t.pos+1:]...)
		}
	case ke.Ctrl && string(ke.Runes) == "k":
		t.value = t.value[:t.pos]
	case ke.Ctrl && string(ke.Runes) == "u":
		t.value = t.value[t.pos:]
		t.pos = 0
	case ke.Key == KeyTab:
		if t.ShowSuggestions && len(t.matchedSugs) > 0 {
			if t.sugIndex < len(t.matchedSugs) {
				t.value = t.matchedSugs[t.sugIndex]
				t.pos = len(t.value)
			}
		}
	default:
		if len(ke.Runes) > 0 && !ke.Ctrl && !ke.Alt {
			if t.CharLimit > 0 && len(t.value) >= t.CharLimit {
				return
			}
			insert := []rune(string(ke.Runes))
			t.value = append(t.value[:t.pos], append(insert, t.value[t.pos:]...)...)
			t.pos += len(insert)
		}
	}

	if t.Validate != nil {
		t.Err = t.Validate(string(t.value))
	}
	t.updateSuggestions()
	if t.onChange != nil {
		t.onChange(string(t.value))
	}
}

func (t *TextInput) updateSuggestions() {
	if !t.ShowSuggestions || len(t.suggestions) == 0 {
		t.matchedSugs = nil
		return
	}
	query := strings.ToLower(string(t.value))
	if query == "" {
		t.matchedSugs = t.suggestions
		return
	}
	t.matchedSugs = nil
	for _, sug := range t.suggestions {
		if strings.Contains(strings.ToLower(string(sug)), query) {
			t.matchedSugs = append(t.matchedSugs, sug)
		}
	}
	t.sugIndex = 0
}

func (t *TextInput) renderCursor(ch string) string {
	if t.cursor.Mode == CursorHide || !t.focus {
		return t.styles.TextInput.Apply(ch)
	}
	return t.styles.Cursor.Apply(ch)
}

func (t *TextInput) Render() string {
	t.mu.Lock()
	defer t.mu.Unlock()

	st := t.styles.Base
	if t.focus {
		st = t.styles.Focused
	}

	prompt := t.styles.Prompt.Apply(t.Prompt)

	display := string(t.value)
	if t.width > 0 {
		visWidth := t.width
		if t.pos > t.offset+visWidth {
			t.offset = t.pos - visWidth + 1
		}
		if t.pos < t.offset {
			t.offset = t.pos
		}
		end := t.offset + visWidth
		if end > len(display) {
			end = len(display)
		}
		display = display[t.offset:end]
	}

	if len(display) == 0 && !t.focus && t.Placeholder != "" {
		return prompt + t.styles.Placeholder.Apply(t.Placeholder)
	}

	if t.EchoMode == EchoPassword {
		echoed := make([]rune, len(t.value))
		for i := range echoed {
			echoed[i] = t.EchoCharacter
		}
		display = string(echoed)
	} else if t.EchoMode == EchoNone {
		display = strings.Repeat(string(t.EchoCharacter), len(t.value))
	}

	posInDisplay := t.pos - t.offset
	if posInDisplay < 0 {
		posInDisplay = 0
	}
	if posInDisplay > len(display) {
		posInDisplay = len(display)
	}

	var before, cursor, after string
	if t.focus && posInDisplay < len(display) {
		before = st.Apply(display[:posInDisplay])
		cursor = t.renderCursor(string(display[posInDisplay]))
		after = st.Apply(display[posInDisplay+1:])
	} else if t.focus && posInDisplay == len(display) {
		before = st.Apply(display)
		cursor = t.renderCursor(" ")
		after = ""
	} else {
		before = st.Apply(display)
	}

	return prompt + before + cursor + after
}
