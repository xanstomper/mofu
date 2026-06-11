package mofu

// ---------------------------------------------------------------------------
// Design Grammar — composable layout primitives
// ---------------------------------------------------------------------------
//
// These are the high-level building blocks for composing interfaces.
// The design engine translates intent into optimized layouts using these primitives.

// DensityProfile controls information density.
type DensityProfile int

const (
	DensityComfortable DensityProfile = iota
	DensityCompact
	DensityDense
	DensityMinimal
)

// SpacingForDensity returns the base spacing unit for a density profile.
func SpacingForDensity(d DensityProfile) int {
	return [...]int{4, 2, 1, 0}[d]
}

// LayoutIntent describes the high-level purpose of a layout.
type LayoutIntent int

const (
	IntentDashboard  LayoutIntent = iota // Multi-panel monitoring
	IntentWorkspace                      // Editor + sidebar
	IntentInspector                       // Detail view
	IntentList                            // Scrollable list
	IntentForm                            // Input form
	IntentModal                           // Overlay dialog
	IntentSplash                          // Boot/loading screen
)

// ---------------------------------------------------------------------------
// Design Primitives
// ---------------------------------------------------------------------------

// Container is a generic layout container with spacing and optional border.
type Container struct {
	BaseNode
	Title   string
	Density DensityProfile
}

// NewContainer creates a styled container.
func NewContainer(title string, children ...Node) *Container {
	c := &Container{Title: title, Density: DensityComfortable}
	c.children = children
	c.Style().Border = BorderRounded
	c.Style().Padding = SpacingTokenAll(SpacingS)
	return c
}

// Panel is a titled section with border and padding.
type Panel struct {
	BaseNode
	Title string
}

// NewPanel creates a titled panel.
func NewPanel(title string, children ...Node) *Panel {
	p := &Panel{Title: title}
	p.children = children
	p.Style().Border = BorderRounded
	p.Style().Padding = SpacingTokenAll(SpacingS)
	p.Style().Margin = Spacing{Bottom: 1}
	return p
}

// Section is a visual grouping without border.
type Section struct {
	BaseNode
	Title string
}

// NewSection creates a titled section.
func NewSection(title string, children ...Node) *Section {
	s := &Section{Title: title}
	s.children = children
	s.Style().Margin = Spacing{Bottom: 1}
	return s
}

// Header is a top-level title bar.
type Header struct {
	BaseNode
	Title    string
	Subtitle string
}

// NewHeader creates a header with title and optional subtitle.
func NewHeader(title, subtitle string) *Header {
	h := &Header{Title: title, Subtitle: subtitle}
	h.Style().Padding = Spacing{Bottom: 1}
	return h
}

// Footer is a bottom status bar.
type Footer struct {
	BaseNode
	Items []string
}

// NewFooter creates a footer status bar.
func NewFooter(items ...string) *Footer {
	f := &Footer{Items: items}
	f.Style().Padding = Spacing{Top: 1}
	return f
}

// StatusBar is a single-line status display.
type StatusBar struct {
	BaseNode
	Left   string
	Center string
	Right  string
}

// NewStatusBar creates a three-section status bar.
func NewStatusBar(left, center, right string) *StatusBar {
	return &StatusBar{Left: left, Center: center, Right: right}
}

// Sidebar is a fixed-width side panel.
type Sidebar struct {
	BaseNode
	Width int
}

// NewSidebar creates a sidebar with the given width.
func NewSidebar(width int, children ...Node) *Sidebar {
	s := &Sidebar{Width: width}
	s.children = children
	s.Style().Width = width
	s.Style().Border = BorderRounded
	return s
}

// Grid is a multi-column layout.
type Grid struct {
	BaseNode
	Columns int
	Gap     int
}

// NewSimpleGrid creates a simple grid with the given number of columns.
func NewSimpleGrid(columns int, children ...Node) *Grid {
	g := &Grid{Columns: columns, Gap: 1}
	g.children = children
	return g
}

// Overlay is a layered composition.
type DesignOverlay struct {
	BaseNode
}

// NewDesignOverlay creates an overlay composition.
func NewDesignOverlay(children ...Node) *DesignOverlay {
	o := &DesignOverlay{}
	o.children = children
	return o
}

// Divider is a horizontal or vertical separator.
type Divider struct {
	BaseNode
	Direction Direction
}

// NewHorizontalDivider creates a horizontal line separator.
func NewHorizontalDivider() *Divider {
	d := &Divider{Direction: DirectionRow}
	d.Style().Height = 1
	return d
}

// NewVerticalDivider creates a vertical line separator.
func NewVerticalDivider() *Divider {
	d := &Divider{Direction: DirectionColumn}
	d.Style().Width = 1
	return d
}

// Spacer is an empty flexible space.
type Spacer struct {
	BaseNode
}

// NewSpacer creates flexible space that grows to fill available area.
func NewSpacer() *Spacer {
	s := &Spacer{}
	s.Style().Grow = 1
	return s
}

// Gap is a fixed-size empty space.
type Gap struct {
	BaseNode
}

// NewGap creates a fixed-size gap.
func NewGap(size int) *Gap {
	g := &Gap{}
	g.Style().Width = size
	g.Style().Height = 1
	return g
}
