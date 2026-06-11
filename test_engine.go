package mofu

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Deterministic Test Engine — replay + snapshot UI testing
// ---------------------------------------------------------------------------

// TestEvent is a recorded event for replay testing.
type TestEvent struct {
	Timestamp time.Time
	Type      string
	Data      any
	Frame     int64
}

// TestRecorder records events for deterministic replay.
type TestRecorder struct {
	mu       sync.Mutex
	events   []TestEvent
	recording bool
	frame    int64
}

// NewTestRecorder creates a new test event recorder.
func NewTestRecorder() *TestRecorder {
	return &TestRecorder{}
}

// Start begins recording events.
func (tr *TestRecorder) Start() {
	tr.mu.Lock()
	tr.recording = true
	tr.events = nil
	tr.frame = 0
	tr.mu.Unlock()
}

// Stop stops recording and returns the recorded events.
func (tr *TestRecorder) Stop() []TestEvent {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	tr.recording = false
	out := make([]TestEvent, len(tr.events))
	copy(out, tr.events)
	return out
}

// Record records an event. Only works while recording.
func (tr *TestRecorder) Record(eventType string, data any) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	if !tr.recording {
		return
	}
	tr.events = append(tr.events, TestEvent{
		Timestamp: time.Now(),
		Type:      eventType,
		Data:      data,
		Frame:     tr.frame,
	})
}

// NextFrame advances the frame counter.
func (tr *TestRecorder) NextFrame() {
	tr.mu.Lock()
	tr.frame++
	tr.mu.Unlock()
}

// Events returns a copy of recorded events.
func (tr *TestRecorder) Events() []TestEvent {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	out := make([]TestEvent, len(tr.events))
	copy(out, tr.events)
	return out
}

// Save writes recorded events to a file.
func (tr *TestRecorder) Save(path string) error {
	events := tr.Events()
	data, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// Load reads events from a file.
func LoadTestEvents(path string) ([]TestEvent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var events []TestEvent
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, err
	}
	return events, nil
}

// ---------------------------------------------------------------------------
// TestPlayer — replay events deterministically
// ---------------------------------------------------------------------------

// TestPlayer replays recorded events.
type TestPlayer struct {
	events  []TestEvent
	index   int
	handler func(TestEvent)
}

// NewTestPlayer creates a player for the given events.
func NewTestPlayer(events []TestEvent) *TestPlayer {
	return &TestPlayer{events: events}
}

// OnEvent registers a handler for replayed events.
func (tp *TestPlayer) OnEvent(fn func(TestEvent)) {
	tp.handler = fn
}

// Step replays the next event. Returns false when no more events.
func (tp *TestPlayer) Step() bool {
	if tp.index >= len(tp.events) {
		return false
	}
	event := tp.events[tp.index]
	tp.index++
	if tp.handler != nil {
		tp.handler(event)
	}
	return true
}

// StepFrame replays all events for the next frame.
// Returns the frame number and false when done.
func (tp *TestPlayer) StepFrame() (int64, bool) {
	if tp.index >= len(tp.events) {
		return 0, false
	}

	frame := tp.events[tp.index].Frame
	for tp.index < len(tp.events) && tp.events[tp.index].Frame == frame {
		if tp.handler != nil {
			tp.handler(tp.events[tp.index])
		}
		tp.index++
	}

	return frame, true
}

// Reset rewinds to the beginning.
func (tp *TestPlayer) Reset() {
	tp.index = 0
}

// Remaining returns the number of events left.
func (tp *TestPlayer) Remaining() int {
	return len(tp.events) - tp.index
}

// ---------------------------------------------------------------------------
// UI Snapshot Testing
// ---------------------------------------------------------------------------

// UISnapshot captures the rendered output of a UI for comparison.
type UISnapshot struct {
	Name      string
	Width     int
	Height    int
	Cells     [][]CellSnapshot
	Timestamp time.Time
	Metadata  map[string]string
}

// CellSnapshot is a single cell in a snapshot.
type CellSnapshot struct {
	Char rune
	Fg   uint32
	Bg   uint32
}

// CaptureSnapshot creates a snapshot from a scene buffer.
func CaptureSnapshot(name string, cells [][]CellSnapshot, width, height int) UISnapshot {
	// Deep copy cells
	cp := make([][]CellSnapshot, len(cells))
	for i, row := range cells {
		cp[i] = make([]CellSnapshot, len(row))
		copy(cp[i], row)
	}

	return UISnapshot{
		Name:      name,
		Width:     width,
		Height:    height,
		Cells:     cp,
		Timestamp: time.Now(),
		Metadata:  make(map[string]string),
	}
}

// SaveSnapshot writes a snapshot to a JSON file.
func SaveSnapshot(path string, snap UISnapshot) error {
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// LoadSnapshot reads a snapshot from a JSON file.
func LoadSnapshot(path string) (UISnapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return UISnapshot{}, err
	}
	var snap UISnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return UISnapshot{}, err
	}
	return snap, nil
}

// CompareSnapshots compares two snapshots and returns differences.
func CompareSnapshots(expected, actual UISnapshot) []SnapshotDiff {
	var diffs []SnapshotDiff

	if expected.Width != actual.Width || expected.Height != actual.Height {
		diffs = append(diffs, SnapshotDiff{
			Type:    "size",
			Message: fmt.Sprintf("size mismatch: %dx%d vs %dx%d", expected.Width, expected.Height, actual.Width, actual.Height),
		})
		return diffs
	}

	for y := 0; y < expected.Height && y < len(expected.Cells) && y < len(actual.Cells); y++ {
		for x := 0; x < expected.Width && x < len(expected.Cells[y]) && x < len(actual.Cells[y]); x++ {
			e := expected.Cells[y][x]
			a := actual.Cells[y][x]
			if e.Char != a.Char || e.Fg != a.Fg || e.Bg != a.Bg {
				diffs = append(diffs, SnapshotDiff{
					Type:     "cell",
					X:        x,
					Y:        y,
					Expected: e,
					Actual:   a,
					Message:  fmt.Sprintf("cell [%d,%d]: expected '%c' fg=%d bg=%d, got '%c' fg=%d bg=%d", x, y, e.Char, e.Fg, e.Bg, a.Char, a.Fg, a.Bg),
				})
			}
		}
	}

	return diffs
}

// SnapshotDiff describes a difference between two snapshots.
type SnapshotDiff struct {
	Type     string
	X, Y     int
	Expected CellSnapshot
	Actual   CellSnapshot
	Message  string
}

// ---------------------------------------------------------------------------
// Deterministic Frame Stepping
// ---------------------------------------------------------------------------

// FrameStepper advances frames deterministically for testing.
type FrameStepper struct {
	mu       sync.Mutex
	frame    int64
	delta    time.Duration
	onStep   func(frame int64, delta time.Duration)
}

// NewFrameStepper creates a frame stepper with a fixed delta.
func NewFrameStepper(delta time.Duration) *FrameStepper {
	return &FrameStepper{delta: delta}
}

// OnStep registers a callback for each frame step.
func (fs *FrameStepper) OnStep(fn func(frame int64, delta time.Duration)) {
	fs.mu.Lock()
	fs.onStep = fn
	fs.mu.Unlock()
}

// Step advances one frame.
func (fs *FrameStepper) Step() int64 {
	fs.mu.Lock()
	fs.frame++
	if fs.onStep != nil {
		fs.onStep(fs.frame, fs.delta)
	}
	frame := fs.frame
	fs.mu.Unlock()
	return frame
}

// StepN advances N frames.
func (fs *FrameStepper) StepN(n int) int64 {
	for i := 0; i < n; i++ {
		fs.Step()
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return fs.frame
}

// Frame returns the current frame number.
func (fs *FrameStepper) Frame() int64 {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return fs.frame
}

// Delta returns the frame delta.
func (fs *FrameStepper) Delta() time.Duration {
	return fs.delta
}

// Reset rewinds to frame 0.
func (fs *FrameStepper) Reset() {
	fs.mu.Lock()
	fs.frame = 0
	fs.mu.Unlock()
}

// ---------------------------------------------------------------------------
// Mock Terminal — headless testing
// ---------------------------------------------------------------------------

// MockTerminal simulates a terminal for headless testing.
type MockTerminal struct {
	mu       sync.Mutex
	width    int
	height   int
	output   []byte
	input    []byte
	cursor   struct{ X, Y int }
	rawMode  bool
}

// NewMockTerminal creates a mock terminal with the given dimensions.
func NewMockTerminal(width, height int) *MockTerminal {
	return &MockTerminal{
		width:  width,
		height: height,
	}
}

// Width returns the terminal width.
func (mt *MockTerminal) Width() int { return mt.width }

// Height returns the terminal height.
func (mt *MockTerminal) Height() int { return mt.height }

// Write appends to the output buffer.
func (mt *MockTerminal) Write(p []byte) (int, error) {
	mt.mu.Lock()
	mt.output = append(mt.output, p...)
	mt.mu.Unlock()
	return len(p), nil
}

// Read returns from the input buffer.
func (mt *MockTerminal) Read(p []byte) (int, error) {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	n := copy(p, mt.input)
	mt.input = mt.input[n:]
	return n, nil
}

// FeedInput adds data to the input buffer.
func (mt *MockTerminal) FeedInput(data []byte) {
	mt.mu.Lock()
	mt.input = append(mt.input, data...)
	mt.mu.Unlock()
}

// Output returns the accumulated output.
func (mt *MockTerminal) Output() []byte {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	out := make([]byte, len(mt.output))
	copy(out, mt.output)
	return out
}

// ClearOutput clears the output buffer.
func (mt *MockTerminal) ClearOutput() {
	mt.mu.Lock()
	mt.output = nil
	mt.mu.Unlock()
}

// Resize simulates a terminal resize.
func (mt *MockTerminal) Resize(width, height int) {
	mt.mu.Lock()
	mt.width = width
	mt.height = height
	mt.mu.Unlock()
}

// OutputString returns output as a string.
func (mt *MockTerminal) OutputString() string {
	return string(mt.Output())
}
