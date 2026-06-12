package mofu

import (
	"strings"
	"sync"
)

type Viewport struct {
	mu          sync.Mutex
	width       int
	height      int
	content     []string
	offsetY     int
	offsetX     int
	YOffset     int
	XOffset     int
	linePanning  int
	mouseWheelEnabled bool
	keyMap       *KeyMap
}

func NewViewport(w, h int) *Viewport {
	vp := &Viewport{
		width:  w,
		height: h,
		keyMap: NewKeyMap(),
	}
	vp.keyMap.Set("up", NewBinding(KeyUp, HelpText{Key: "↑", Desc: "scroll up"}))
	vp.keyMap.Set("down", NewBinding(KeyDown, HelpText{Key: "↓", Desc: "scroll down"}))
	vp.keyMap.Set("pgup", NewBinding(KeyPgUp, HelpText{Key: "pgup", Desc: "page up"}))
	vp.keyMap.Set("pgdown", NewBinding(KeyPgDn, HelpText{Key: "pgdn", Desc: "page down"}))
	vp.keyMap.Set("home", NewBinding(KeyHome, HelpText{Key: "home", Desc: "top"}))
	vp.keyMap.Set("end", NewBinding(KeyEnd, HelpText{Key: "end", Desc: "bottom"}))
	return vp
}

func (vp *Viewport) SetContent(content string) {
	vp.mu.Lock()
	vp.content = strings.Split(content, "\n")
	vp.mu.Unlock()
}

func (vp *Viewport) SetWidth(w int) {
	vp.mu.Lock()
	vp.width = w
	vp.mu.Unlock()
}

func (vp *Viewport) SetHeight(h int) {
	vp.mu.Lock()
	vp.height = h
	vp.mu.Unlock()
}

func (vp *Viewport) GotoTop() {
	vp.mu.Lock()
	vp.YOffset = 0
	vp.mu.Unlock()
}

func (vp *Viewport) GotoBottom() {
	vp.mu.Lock()
	maxOffset := len(vp.content) - vp.height
	if maxOffset < 0 {
		maxOffset = 0
	}
	vp.YOffset = maxOffset
	vp.mu.Unlock()
}

func (vp *Viewport) HalfViewUp() {
	vp.mu.Lock()
	vp.YOffset -= vp.height / 2
	if vp.YOffset < 0 {
		vp.YOffset = 0
	}
	vp.mu.Unlock()
}

func (vp *Viewport) HalfViewDown() {
	vp.mu.Lock()
	vp.YOffset += vp.height / 2
	maxOffset := len(vp.content) - vp.height
	if maxOffset < 0 {
		maxOffset = 0
	}
	if vp.YOffset > maxOffset {
		vp.YOffset = maxOffset
	}
	vp.mu.Unlock()
}

func (vp *Viewport) ScrollUp(n int) {
	vp.mu.Lock()
	vp.YOffset -= n
	if vp.YOffset < 0 {
		vp.YOffset = 0
	}
	vp.mu.Unlock()
}

func (vp *Viewport) ScrollDown(n int) {
	vp.mu.Lock()
	vp.YOffset += n
	maxOffset := len(vp.content) - vp.height
	if maxOffset < 0 {
		maxOffset = 0
	}
	if vp.YOffset > maxOffset {
		vp.YOffset = maxOffset
	}
	vp.mu.Unlock()
}

func (vp *Viewport) ScrollLeft(n int) {
	vp.mu.Lock()
	vp.XOffset -= n
	if vp.XOffset < 0 {
		vp.XOffset = 0
	}
	vp.mu.Unlock()
}

func (vp *Viewport) ScrollRight(n int) {
	vp.mu.Lock()
	vp.XOffset += n
	vp.mu.Unlock()
}

func (vp *Viewport) Update(msg KeyEvent) {
	vp.mu.Lock()
	defer vp.mu.Unlock()
	switch msg.Key {
	case KeyUp:
		vp.YOffset--
	case KeyDown:
		vp.YOffset++
	case KeyPgUp:
		vp.YOffset -= vp.height
	case KeyPgDn:
		vp.YOffset += vp.height
	case KeyHome:
		vp.YOffset = 0
	case KeyEnd:
		vp.YOffset = len(vp.content) - vp.height
	}
	if vp.YOffset < 0 {
		vp.YOffset = 0
	}
	maxOffset := len(vp.content) - vp.height
	if maxOffset < 0 {
		maxOffset = 0
	}
	if vp.YOffset > maxOffset {
		vp.YOffset = maxOffset
	}
}

func (vp *Viewport) AtBottom() bool {
	vp.mu.Lock()
	defer vp.mu.Unlock()
	maxOffset := len(vp.content) - vp.height
	if maxOffset < 0 {
		maxOffset = 0
	}
	return vp.YOffset >= maxOffset
}

func (vp *Viewport) ScrollPercentage() float64 {
	vp.mu.Lock()
	defer vp.mu.Unlock()
	total := len(vp.content)
	if total <= vp.height {
		return 100
	}
	return float64(vp.YOffset) / float64(total-vp.height) * 100
}

func (vp *Viewport) Render() string {
	vp.mu.Lock()
	defer vp.mu.Unlock()

	start := vp.YOffset
	if start < 0 {
		start = 0
	}
	end := start + vp.height
	if end > len(vp.content) {
		end = len(vp.content)
	}

	visible := vp.content[start:end]
	var out strings.Builder
	for i, line := range visible {
		if vp.XOffset > 0 && vp.XOffset < len(line) {
			line = line[vp.XOffset:]
		} else if vp.XOffset >= len(line) {
			line = ""
		}
		if len(line) > vp.width {
			line = line[:vp.width]
		}
		out.WriteString(line)
		if i < len(visible)-1 {
			out.WriteString("\n")
		}
	}

	remaining := vp.height - len(visible)
	for i := 0; i < remaining; i++ {
		out.WriteString("\n")
	}

	return out.String()
}
