package mofu

import (
	"runtime"
	"sync"
)

// ---------------------------------------------------------------------------
// Accessibility (Anthology Ch.15)
// ---------------------------------------------------------------------------

// AriaRole maps ARIA-like roles to TUI widgets.
type AriaRole uint8

const (
	AriaAlert AriaRole = iota
	AriaAlertDialog
	AriaButton
	AriaCheckbox
	AriaDialog
	AriaGrid
	AriaLink
	AriaListbox
	AriaMenu
	AriaMenuItem
	AriaOption
	AriaProgressbar
	AriaRadio
	AriaSlider
	AriaSpinbutton
	AriaStatus
	AriaTab
	AriaTablist
	AriaTextbox
	AriaTimer
	AriaTooltip
)

// LiveRegion controls screen-reader announcement urgency.
type LiveRegion uint8

const (
	LiveOff LiveRegion = iota
	LivePolite
	LiveAssertive
)

// AriaAttributes stores accessibility metadata for a widget.
type AriaAttributes struct {
	Role        AriaRole
	Label       string
	Description string
	ValueNow    float64
	ValueMin    float64
	ValueMax    float64
	Checked     bool
	Disabled    bool
	Hidden      bool
	Live        LiveRegion
}

// FocusableAccessibleWidget is implemented by widgets that can expose ARIA metadata and receive focus.
type FocusableAccessibleWidget interface {
	AccessibleWidget
	Focus()
	Blur()
	IsFocused() bool
}

// AccessibleWidget is implemented by widgets that can expose ARIA metadata.
type AccessibleWidget interface {
	AccessibleAttributes() AriaAttributes
}

// FocusMode controls focus visibility.
type FocusMode uint8

const (
	FocusModeNormal FocusMode = iota
	FocusModeHighContrast
	FocusModeLargeText
)

// FocusIndicator describes how focus is rendered and announced.
type FocusIndicator struct {
	Mode        FocusMode
	BorderStyle BorderStyle
	Style       Style
	Announce    string
}

// A11yContext carries global accessibility preferences.
type A11yContext struct {
	mu                 sync.RWMutex
	HighContrast       bool
	ReduceMotion       bool
	LargeTextScale     float64
	ScreenReaderActive bool
	FocusIndicator     FocusIndicator
}

// NewA11yContext returns default accessibility settings.
func NewA11yContext() *A11yContext {
	return &A11yContext{
		LargeTextScale: 1,
		FocusIndicator: FocusIndicator{BorderStyle: BorderDouble, Style: DefaultStyle().WithAttrs(AttrReverse)},
	}
}

// SetHighContrast enables high contrast mode.
func (a *A11yContext) SetHighContrast(on bool) { a.mu.Lock(); a.HighContrast = on; a.mu.Unlock() }

// SetReduceMotion enables motion reduction.
func (a *A11yContext) SetReduceMotion(on bool) { a.mu.Lock(); a.ReduceMotion = on; a.mu.Unlock() }

// SetLargeTextScale sets the text scale factor.
func (a *A11yContext) SetLargeTextScale(v float64) {
	a.mu.Lock()
	if v < 1 {
		v = 1
	}
	a.LargeTextScale = v
	a.mu.Unlock()
}

// ScreenReaderAnnouncer queues announcements for platform TTS.
type ScreenReaderAnnouncer struct {
	mu           sync.Mutex
	queue        []string
	isSpeaking   bool
	SpeechRate   int
	lastAnnounce string
}

// Announce queues a polite announcement.
func (s *ScreenReaderAnnouncer) Announce(text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queue = append(s.queue, text)
	s.lastAnnounce = text
	s.processQueueLocked()
}

// AnnouncePriority queues an assertive announcement at the front.
func (s *ScreenReaderAnnouncer) AnnouncePriority(text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queue = append([]string{text}, s.queue...)
	s.lastAnnounce = text
	s.processQueueLocked()
}

// LastAnnouncement returns the latest queued announcement.
func (s *ScreenReaderAnnouncer) LastAnnouncement() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastAnnounce
}

func (s *ScreenReaderAnnouncer) processQueueLocked() {
	if s.isSpeaking || len(s.queue) == 0 {
		return
	}
	s.isSpeaking = true
	text := s.queue[0]
	s.queue = s.queue[1:]
	go s.speak(text)
}

func (s *ScreenReaderAnnouncer) speak(text string) {
	// Platform-neutral fallback: keep the text available; external bridge can consume LastAnnouncement.
	switch runtime.GOOS {
	case "windows":
		// Windows SAPI bridge can be added without changing this API.
	case "darwin":
		// say(1) bridge can be added without changing this API.
	case "linux":
		// espeak bridge can be added without changing this API.
	}
	s.mu.Lock()
	s.isSpeaking = false
	if len(s.queue) > 0 {
		next := s.queue[0]
		s.queue = s.queue[1:]
		s.mu.Unlock()
		s.speak(next)
		return
	}
	s.mu.Unlock()
}

// KeyboardNavigator implements roving-tabindex style keyboard focus.
type KeyboardNavigator struct {
	mu           sync.Mutex
	focusOrder   []FocusableAccessibleWidget
	currentIndex int
	Wrap         bool
}

// SetFocusOrder replaces the focus order.
func (k *KeyboardNavigator) SetFocusOrder(items []FocusableAccessibleWidget) {
	k.mu.Lock()
	k.focusOrder = append([]FocusableAccessibleWidget(nil), items...)
	if k.currentIndex >= len(k.focusOrder) {
		k.currentIndex = 0
	}
	k.mu.Unlock()
}

// FocusNext moves to the next focusable widget.
func (k *KeyboardNavigator) FocusNext() bool {
	k.mu.Lock()
	defer k.mu.Unlock()
	if len(k.focusOrder) == 0 {
		return false
	}
	if len(k.focusOrder) == 1 {
		k.focusOrder[0].Focus()
		return true
	}
	k.currentIndex = (k.currentIndex + 1) % len(k.focusOrder)
	if !k.Wrap && k.currentIndex == 0 {
		k.currentIndex = len(k.focusOrder) - 1
	}
	k.focusOrder[k.currentIndex].Focus()
	return true
}

// FocusPrevious moves to the previous focusable widget.
func (k *KeyboardNavigator) FocusPrevious() bool {
	k.mu.Lock()
	defer k.mu.Unlock()
	if len(k.focusOrder) == 0 {
		return false
	}
	if len(k.focusOrder) == 1 {
		k.focusOrder[0].Focus()
		return true
	}
	k.currentIndex--
	if k.currentIndex < 0 {
		if k.Wrap {
			k.currentIndex = len(k.focusOrder) - 1
		} else {
			k.currentIndex = 0
		}
	}
	k.focusOrder[k.currentIndex].Focus()
	return true
}
