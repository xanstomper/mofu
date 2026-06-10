package mofu

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"sync/atomic"
	"time"
)

type OutputType int

const (
	OutputTerminal OutputType = iota
	OutputJSON
	OutputReplay
	OutputDebug
)

type OutputChannel interface {
	Write(scene *SceneBuffer) error
	Flush() error
	Close() error
	Type() OutputType
}

type TerminalOutput struct {
	output *os.File
	buf    bytes.Buffer
}

func NewTerminalOutput() *TerminalOutput {
	return &TerminalOutput{output: os.Stdout}
}

func (t *TerminalOutput) Write(scene *SceneBuffer) error {
	return nil
}

func (t *TerminalOutput) Flush() error {
	if t.buf.Len() > 0 {
		_, err := t.output.Write(t.buf.Bytes())
		t.buf.Reset()
		return err
	}
	return nil
}

func (t *TerminalOutput) Close() error     { return nil }
func (t *TerminalOutput) Type() OutputType { return OutputTerminal }
func (t *TerminalOutput) WriteRaw(data string) {
	t.buf.WriteString(data)
}

type jsonCell struct {
	X    int    `json:"x,omitempty"`
	Y    int    `json:"y,omitempty"`
	Char string `json:"char,omitempty"`
}

type jsonFrame struct {
	Frame int64      `json:"frame"`
	Time  time.Time  `json:"time"`
	Cells []jsonCell `json:"cells,omitempty"`
}

type JSONOutput struct {
	writer io.Writer
	enc    *json.Encoder
	count  int
}

func NewJSONOutput(w io.Writer) *JSONOutput {
	return &JSONOutput{writer: w, enc: json.NewEncoder(w)}
}

func (j *JSONOutput) Write(scene *SceneBuffer) error {
	var cells []jsonCell
	for y := 0; y < scene.Height; y++ {
		for x := 0; x < scene.Width; x++ {
			c := scene.Cells[y][x]
			if c.Char != ' ' {
				cells = append(cells, jsonCell{X: x, Y: y, Char: string(c.Char)})
			}
		}
	}
	j.count++
	return j.enc.Encode(jsonFrame{Frame: int64(j.count), Time: time.Now(), Cells: cells})
}

func (j *JSONOutput) Flush() error     { return nil }
func (j *JSONOutput) Close() error     { return nil }
func (j *JSONOutput) Type() OutputType { return OutputJSON }

type ReplayOutput struct {
	frames [][]byte
}

func NewReplayOutput() *ReplayOutput {
	return &ReplayOutput{}
}

func (r *ReplayOutput) Write(scene *SceneBuffer) error {
	var buf bytes.Buffer
	for y := 0; y < scene.Height; y++ {
		for x := 0; x < scene.Width; x++ {
			c := scene.Cells[y][x]
			buf.WriteRune(c.Char)
		}
		buf.WriteByte('\n')
	}
	r.frames = append(r.frames, buf.Bytes())
	return nil
}

func (r *ReplayOutput) Flush() error     { return nil }
func (r *ReplayOutput) Close() error     { return nil }
func (r *ReplayOutput) Type() OutputType { return OutputReplay }
func (r *ReplayOutput) Frames() [][]byte { return r.frames }

type FrameStats struct {
	FrameCount int64
	RenderTime time.Duration
	DirtyCells int
	TotalCells int
	FPS        float64
}

type DebugOutput struct {
	writer io.Writer
	stats  atomic.Value
}

func NewDebugOutput(w io.Writer) *DebugOutput {
	return &DebugOutput{writer: w}
}

func (d *DebugOutput) Write(scene *SceneBuffer) error {
	return nil
}

func (d *DebugOutput) Flush() error {
	s := d.stats.Load().(FrameStats)
	_, err := d.writer.Write([]byte(s.String()))
	return err
}

func (d *DebugOutput) Close() error             { return nil }
func (d *DebugOutput) Type() OutputType         { return OutputDebug }
func (d *DebugOutput) UpdateStats(s FrameStats) { d.stats.Store(s) }

func (fs FrameStats) String() string {
	return ""
}
