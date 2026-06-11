package mofu

import "time"

// FrameStats holds performance metrics for the current frame.
type FrameStats struct {
	FrameCount int64
	RenderTime time.Duration
	DirtyCells int
	TotalCells int
	FPS        float64
}
