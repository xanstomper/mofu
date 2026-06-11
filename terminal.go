package mofu

import (
	"os"
	"strings"
)

// CapabilityProfile detects and stores terminal capabilities.
// Use DetectCapabilities to probe the current terminal.
type CapabilityProfile struct {
	TrueColor   bool
	ANSI256     bool
	ANSI16      bool
	Mouse       bool
	MouseSGR    bool
	BracketedPaste bool
	Unicode     bool
	AltScreen   bool
	SyncOutput  bool // CSI 2026
	KittyKeyboard bool
	Width       int
	Height      int
	Terminal    string // e.g. "xterm-256color", "wezterm", "iterm2"
}

// DetectCapabilities probes the terminal environment and returns a capability profile.
// It checks environment variables and falls back to defaults.
func DetectCapabilities() CapabilityProfile {
	prof := CapabilityProfile{}

	// Check NO_COLOR
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		prof.TrueColor = false
		prof.ANSI256 = false
		prof.ANSI16 = true
		return prof
	}

	// Check COLORTERM
	colorterm := os.Getenv("COLORTERM")
	if strings.Contains(colorterm, "truecolor") || strings.Contains(colorterm, "24bit") {
		prof.TrueColor = true
		prof.ANSI256 = true
		prof.ANSI16 = true
	} else {
		// Check TERM for 256-color support
		term := os.Getenv("TERM")
		if strings.Contains(term, "256") {
			prof.ANSI256 = true
			prof.ANSI16 = true
		} else if term != "" {
			prof.ANSI16 = true
		}
	}

	// Detect terminal type
	termProgram := os.Getenv("TERM_PROGRAM")
	termProgVersion := os.Getenv("TERM_PROGRAM_VERSION")
	switch strings.ToLower(termProgram) {
	case "wezterm":
		prof.TrueColor = true
		prof.Mouse = true
		prof.MouseSGR = true
		prof.BracketedPaste = true
		prof.AltScreen = true
		prof.SyncOutput = true
		prof.KittyKeyboard = true
		prof.Terminal = "wezterm"
		_ = termProgVersion
	case "iterm.app":
		prof.TrueColor = true
		prof.Mouse = true
		prof.MouseSGR = true
		prof.BracketedPaste = true
		prof.AltScreen = true
		prof.SyncOutput = true
		prof.Terminal = "iterm2"
	case "mintty":
		prof.TrueColor = true
		prof.Mouse = true
		prof.AltScreen = true
		prof.Terminal = "mintty"
	case "windows-terminal":
		prof.TrueColor = true
		prof.Mouse = true
		prof.MouseSGR = true
		prof.BracketedPaste = true
		prof.AltScreen = true
		prof.SyncOutput = true
		prof.Terminal = "windows-terminal"
	default:
		// Generic: assume basic ANSI support
		prof.Mouse = true
		prof.AltScreen = true
		prof.Terminal = "unknown"
	}

	// Check TERM for width hints
	term := os.Getenv("TERM")
	if strings.Contains(term, "256") || strings.Contains(term, "color") {
		prof.ANSI256 = true
	}

	// Unicode support is assumed on modern terminals
	prof.Unicode = true

	return prof
}

// ColorDepth returns the maximum color depth supported.
func (p CapabilityProfile) ColorDepth() int {
	if p.TrueColor {
		return 24
	}
	if p.ANSI256 {
		return 8
	}
	if p.ANSI16 {
		return 4
	}
	return 1
}

// SupportsSyncOutput reports whether the terminal supports CSI 2026.
func (p CapabilityProfile) SupportsSyncOutput() bool {
	return p.SyncOutput
}

// SupportsSGRMouse reports whether the terminal supports SGR extended mouse.
func (p CapabilityProfile) SupportsSGRMouse() bool {
	return p.MouseSGR
}
