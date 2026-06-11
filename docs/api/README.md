# MOFU API Reference

## Core Types

### Program

```go
// Program is the main application container.
type Program struct { ... }

// New creates a new MOFU Program.
func New(root Node, opts ...Option) *Program

// Run starts the application.
func (p *Program) Run() error

// Send sends a message to the program.
func (p *Program) Send(msg Msg)

// Kill stops the program.
func (p *Program) Kill()
```

### Options

```go
func WithTheme(theme *Theme) Option
func WithSize(w, h int) Option
func WithFPS(fps int) Option
func WithInput(r io.Reader) Option
func WithOutputWriter(w io.Writer) Option
func WithoutRenderer() Option
func WithoutSignalHandler() Option
```

### Node Interface

```go
type Node interface {
    Render(ctx *RenderContext)
    HandleEvent(event Event) Cmd
    Mount() Cmd
    Unmount()
    Children() []Node
    AddChild(child Node)
    RemoveChild(child Node)
    SetDirty()
    Dirty() bool
    Bounds() Rect
    SetBounds(Rect)
    Style() *Style
}
```

### Minimal

```go
// Minimal provides default Node implementations.
type Minimal struct { ... }

// Embed Minimal in your struct for quick starts.
type MyApp struct {
    mofu.Minimal
    // your state
}
```

## Event System

### Event

```go
type Event struct {
    Type   EventType
    Data   Msg
    Time   time.Time
    Source string
}

type EventType int

const (
    EventKeyPress EventType = iota
    EventMouse
    EventResize
    EventData
    EventAnimation
    EventSystem
    EventCustom
)
```

### KeyEvent

```go
type KeyEvent struct {
    Runes            []byte
    Key              Key
    Alt, Ctrl, Shift bool
}
```

### Key Constants

```go
const (
    KeyNone Key = iota
    KeyUp, KeyDown, KeyRight, KeyLeft
    KeyEnter, KeyEsc, KeyTab, KeySpace, KeyBack
    KeyHome, KeyEnd, KeyPgUp, KeyPgDn
    KeyInsert, KeyDelete
    KeyF1 through KeyF12
    KeyShiftTab
    KeyCtrlA through KeyCtrlZ
)
```

### MouseEvent

```go
type MouseEvent struct {
    X, Y   int
    Button MouseButton
    Action MouseAction
}

type MouseButton int
const (
    MouseLeft, MouseRight, MouseMiddle
    MouseWheelUp, MouseWheelDown, MouseNone
)

type MouseAction int
const (
    MousePress, MouseRelease, MouseDrag, MouseMove
)
```

## Commands

```go
type Cmd func() Msg

func QuitCmd() Cmd
func SendCmd(msg Msg) Cmd
func Batch(cmds ...Cmd) Cmd
func Sequence(cmds ...Cmd) Cmd
func Tick(delay time.Duration, fn func() Msg) Cmd
```

## Messages

```go
type Msg any

type QuitMsg struct{}
type InterruptMsg struct{}
type WindowSizeMsg struct{ Width, Height int }
type BatchMsg []Cmd
type SequenceMsg []Cmd
```

## Rendering

### RenderContext

```go
type RenderContext struct {
    Renderer *Renderer
    Theme    *Theme
    Frame    int64
    Delta    time.Duration
    Bounds   Rect
}
```

### Renderer Methods

```go
func (r *Renderer) WriteString(text string, x, y int, fg, bg Color, attrs AttrMask)
func (r *Renderer) WriteStyledString(text string, x, y int, style Style)
func (r *Renderer) Clear()
func (r *Renderer) Flush() string
func (r *Renderer) Resize(w, h int)
```

## Styling

### Style

```go
type Style struct {
    Foreground Color
    Background Color
    Attrs      AttrMask
    Border     BorderStyle
    Margin     Spacing
    Padding    Spacing
    Width, Height int
    Align      Align
    Gap        int
    Grow       float64
    Direction  Direction
}
```

### Colors

```go
func RGB(r, g, b uint8) Color
func Hex(hex string) Color
func ANSI(code uint8) Color

// Built-in colors
var ColorWhite, ColorBlack, ColorRed, ColorGreen, ColorBlue, ...
```

### Attributes

```go
const (
    AttrBold, AttrDim, AttrItalic, AttrUnderline
    AttrSlowBlink, AttrRapidBlink, AttrReverse
    AttrHidden, AttrStrikethrough, AttrDoubleUnderline, AttrOverline
)
```

### Borders

```go
var BorderNone, BorderNormal, BorderRounded, BorderThick, BorderDouble
```

## Themes

### Theme

```go
type Theme struct {
    Name       string
    Colors     ThemeColors
    Semantic   SemanticColors
    Typography Typography
    Spacing    SpacingScale
    Border     BorderStyle
    Widgets    WidgetThemes
}
```

### ThemeManager

```go
func NewThemeManager(initial *Theme) *ThemeManager
func (tm *ThemeManager) Current() *Theme
func (tm *ThemeManager) Register(name string, theme *Theme)
func (tm *ThemeManager) Apply(name string) bool
func (tm *ThemeManager) OnChange(fn func(old, new *Theme))
```

## Gadgets

### Gadget Interface

```go
type Gadget interface {
    ID() string
    Init(ctx GadgetContext) error
    Bind(binder Binder)
    Render(state StateView) []RenderNode
    OnEvent(e Event)
    OnTick(dt int64)
    Dispose() error
}
```

### Layout Contract

```go
type LayoutContract interface {
    MinSize() (w, h int)
    MaxSize() (w, h int)
    Flex() float64
    Priority() int
    AspectRatio() float64
    OverflowBehavior() OverflowMode
}
```

### Animation Hook

```go
type AnimationHook interface {
    OnEnter(ctx AnimContext) AnimationSpec
    OnExit(ctx AnimContext) AnimationSpec
    OnStateChange(delta StateDelta) AnimationSpec
    OnLayoutChange(layout LayoutChange) AnimationSpec
}
```

## Widgets

### Input

```go
func NewInput() *InputNode
func (i *InputNode) Focus()
func (i *InputNode) Blur()
func (i *InputNode) SetValue(value string)
func (i *InputNode) InsertRune(r rune)
func (i *InputNode) DeleteBefore()
func (i *InputNode) DeleteAfter()
```

### List

```go
func NewList(items []ListItem) *ListNode
func (l *ListNode) SetItems(items []ListItem)
func (l *ListNode) SetSelected(index int)
func (l *ListNode) SelectedItem() *ListItem
```

### Table

```go
func NewTable(columns []TableColumn, rows [][]string) *Table
func (t *Table) Focus()
func (t *Table) Blur()
```

### Button

```go
func NewButton(label string, onPress func() mofu.Cmd) *Button
func (b *Button) Focus()
func (b *Button) Blur()
```

### And more...

See `widgets/` directory for all 15 widgets.
