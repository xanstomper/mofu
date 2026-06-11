package mofu

// ---------------------------------------------------------------------------
// Layout Intelligence — density-aware adaptive layouts
// ---------------------------------------------------------------------------

// AdaptiveDensity controls spacing and sizing across the UI.
type AdaptiveDensity int

const (
	AdaptiveCompact         AdaptiveDensity = iota // minimal spacing, max info density
	AdaptiveNormalDensity                          // balanced spacing
	AdaptiveComfortable                            // generous spacing, easy scanning
)

// SpacingScale returns the spacing multiplier for a density level.
func (d AdaptiveDensity) SpacingScale() float64 {
	switch d {
	case AdaptiveCompact:
		return 0.5
	case AdaptiveNormalDensity:
		return 1.0
	case AdaptiveComfortable:
		return 1.5
	default:
		return 1.0
	}
}

// RowHeight returns the row height in cells for a density level.
func (d AdaptiveDensity) RowHeight() int {
	switch d {
	case AdaptiveCompact:
		return 1
	case AdaptiveNormalDensity:
		return 2
	case AdaptiveComfortable:
		return 3
	default:
		return 2
	}
}

// Padding returns the padding in cells for a density level.
func (d AdaptiveDensity) Padding() int {
	switch d {
	case AdaptiveCompact:
		return 0
	case AdaptiveNormalDensity:
		return 1
	case AdaptiveComfortable:
		return 2
	default:
		return 1
	}
}

// Gap returns the gap between elements for a density level.
func (d AdaptiveDensity) Gap() int {
	switch d {
	case AdaptiveCompact:
		return 0
	case AdaptiveNormalDensity:
		return 1
	case AdaptiveComfortable:
		return 2
	default:
		return 1
	}
}

// ---------------------------------------------------------------------------
// Adaptive Layout — auto-adjusts based on terminal size
// ---------------------------------------------------------------------------

// AdaptiveConfig defines how a layout adapts to different terminal sizes.
type AdaptiveConfig struct {
	Compact  LayoutConfig
	Normal   LayoutConfig
	Expanded LayoutConfig
}

// LayoutConfig defines layout parameters for a specific size class.
type LayoutConfig struct {
	Columns int
	Sidebar bool
	Header  bool
	Footer  bool
	Density AdaptiveDensity
}

// ResolveAdaptive returns the layout config for the given terminal width.
func ResolveAdaptive(config AdaptiveConfig, width int) LayoutConfig {
	switch ClassifySize(width) {
	case SizeCompact:
		return config.Compact
	case SizeMedium:
		return config.Normal
	default:
		return config.Expanded
	}
}

// DefaultAdaptiveConfig returns sensible defaults for all size classes.
func DefaultAdaptiveConfig() AdaptiveConfig {
	return AdaptiveConfig{
		Compact: LayoutConfig{
			Columns: 1,
			Sidebar: false,
			Header:  true,
			Footer:  false,
			Density: AdaptiveCompact,
		},
		Normal: LayoutConfig{
			Columns: 2,
			Sidebar: true,
			Header:  true,
			Footer:  true,
			Density: AdaptiveNormalDensity,
		},
		Expanded: LayoutConfig{
			Columns: 3,
			Sidebar: true,
			Header:  true,
			Footer:  true,
			Density: AdaptiveComfortable,
		},
	}
}

// ---------------------------------------------------------------------------
// Content-aware sizing
// ---------------------------------------------------------------------------

// ContentSize hints the size needed for a content block.
type ContentSize struct {
	MinWidth    int
	MinHeight   int
	IdealWidth  int
	IdealHeight int
	MaxWidth    int
	MaxHeight   int
}

// FitContent calculates the best size for content within available space.
func FitContent(content ContentSize, available Rect) Rect {
	w := available.Width
	h := available.Height

	if content.MaxWidth > 0 && w > content.MaxWidth {
		w = content.MaxWidth
	}
	if content.MaxHeight > 0 && h > content.MaxHeight {
		h = content.MaxHeight
	}
	if w < content.MinWidth {
		w = content.MinWidth
	}
	if h < content.MinHeight {
		h = content.MinHeight
	}

	x := available.X + (available.Width-w)/2
	y := available.Y + (available.Height-h)/2

	return Rect{X: x, Y: y, Width: w, Height: h}
}

// ---------------------------------------------------------------------------
// Overflow handling
// ---------------------------------------------------------------------------

// OverflowStrategy controls what happens when content exceeds available space.
type OverflowStrategy int

const (
	OverflowClip      OverflowStrategy = iota // truncate content
	OverflowScroll                            // add scroll offset
	OverflowWrap                              // wrap text
	OverflowEllipsis                          // add "..." indicator
)

// HandleOverflow returns the visible portion of content based on strategy.
func HandleOverflow(content string, maxWidth int, strategy OverflowStrategy) string {
	if len(content) <= maxWidth {
		return content
	}

	switch strategy {
	case OverflowClip:
		return content[:maxWidth]
	case OverflowEllipsis:
		if maxWidth <= 3 {
			return content[:maxWidth]
		}
		return content[:maxWidth-3] + "..."
	case OverflowWrap:
		return content[:maxWidth]
	default:
		return content[:maxWidth]
	}
}

// ---------------------------------------------------------------------------
// Auto-collapse
// ---------------------------------------------------------------------------

// CollapseRule defines when to collapse a section.
type CollapseRule struct {
	MinWidth  int
	MinHeight int
	Priority  int
	OnCollapse func()
	OnExpand   func()
}

// ShouldCollapse reports whether a section should be collapsed.
func (r CollapseRule) ShouldCollapse(width, height int) bool {
	if r.MinWidth > 0 && width < r.MinWidth {
		return true
	}
	if r.MinHeight > 0 && height < r.MinHeight {
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// Density-aware style helpers
// ---------------------------------------------------------------------------

// DensityStyle returns a style adjusted for the given density level.
func DensityStyle(base Style, density AdaptiveDensity) Style {
	s := base
	s.Padding = Spacing{
		Top:    density.Padding(),
		Right:  density.Padding(),
		Bottom: density.Padding(),
		Left:   density.Padding(),
	}
	s.Gap = density.Gap()
	return s
}

// DensitySpacing returns the spacing for a density level.
func DensitySpacing(density AdaptiveDensity) Spacing {
	p := density.Padding()
	return Spacing{Top: p, Right: p, Bottom: p, Left: p}
}
