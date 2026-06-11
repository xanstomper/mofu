package mofu

import "time"

type MemoryStats struct {
	HeapAlloc   uint64
	HeapInuse   uint64
	HeapIdle    uint64
	HeapObjects uint64
	StackInuse  uint64
	NumGC       uint32
}

func ReadMemoryStats() MemoryStats {
	return MemoryStats{}
}

type FrameProfiler struct{}

func NewFrameProfiler(targetFPS int) *FrameProfiler {
	return &FrameProfiler{}
}

func (f *FrameProfiler) Snapshot() FrameProfileSnapshot { return FrameProfileSnapshot{} }
func (f *FrameProfiler) FPS() float64                    { return 0 }
func (f *FrameProfiler) FrameTimeP95() time.Duration { return 0 }

type FrameProfileSnapshot struct{}

type ColorScale struct{}

func (c ColorScale) At(t float64) Color { return Color{} }

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

type BrailleChar rune

type brailleDots [8]bool

func (b *brailleDots) encode() rune { return 0 }

func brailleDecode(ch rune) *brailleDots { return &brailleDots{} }
