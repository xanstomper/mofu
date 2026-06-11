package render

import (
	"strings"
	"testing"
)

func TestFrameBufferSetGet(t *testing.T) {
	fb := NewFrameBuffer(10, 5)
	if !fb.Set(3, 2, 'A', 0xFF0000, 0, 1) {
		t.Fatal("Set in bounds returned false")
	}
	c := fb.Get(3, 2)
	if c == nil || c.Char != 'A' || c.Fg != 0xFF0000 || c.Attrs != 1 {
		t.Fatalf("Get returned wrong cell: %+v", c)
	}
	if fb.Set(-1, 0, 'X', 0, 0, 0) || fb.Set(10, 0, 'X', 0, 0, 0) || fb.Set(0, 5, 'X', 0, 0, 0) {
		t.Fatal("Set out of bounds returned true")
	}
	if fb.Get(-1, 0) != nil || fb.Get(10, 0) != nil {
		t.Fatal("Get out of bounds returned non-nil")
	}
}

func TestFrameBufferWideChar(t *testing.T) {
	fb := NewFrameBuffer(10, 1)
	fb.Set(2, 0, '世', 0, 0, 0)
	c := fb.Get(2, 0)
	if c.Width != 2 {
		t.Fatalf("wide char width = %d, want 2", c.Width)
	}
	cont := fb.Get(3, 0)
	if cont.Char != 0 || cont.Width != 0 {
		t.Fatalf("continuation cell not marked: %+v", cont)
	}
}

func TestFrameBufferClear(t *testing.T) {
	fb := NewFrameBuffer(4, 4)
	fb.Set(1, 1, 'Z', 1, 2, 3)
	fb.Clear()
	c := fb.Get(1, 1)
	if c.Char != ' ' || c.Fg != 0 || !c.Dirty {
		t.Fatalf("Clear did not reset cell: %+v", c)
	}
}

func TestDiffRendererFlushEmitsChanges(t *testing.T) {
	dr := NewDiffRenderer(20, 5)
	dr.Front().Set(0, 0, 'H', 0, 0, 0)
	dr.Front().Set(1, 0, 'i', 0, 0, 0)
	dr.MarkRect(0, 0, 20, 1)
	out := dr.Flush()
	if out == "" {
		t.Fatal("Flush returned empty output for changed cells")
	}
	if !strings.HasPrefix(out, syncStart) || !strings.HasSuffix(out, syncEnd) {
		t.Fatal("output not wrapped in Synchronized Output sequences")
	}
	if !strings.Contains(out, "H") || !strings.Contains(out, "i") {
		t.Fatalf("output missing cell content: %q", out)
	}
}

func TestDiffRendererNoDirtyNoOutput(t *testing.T) {
	dr := NewDiffRenderer(10, 3)
	if out := dr.Flush(); out != "" {
		t.Fatalf("Flush without dirty rects produced output: %q", out)
	}
}

func TestDiffRendererResize(t *testing.T) {
	dr := NewDiffRenderer(10, 5)
	dr.Resize(40, 20)
	if dr.Width() != 40 || dr.Height() != 20 {
		t.Fatalf("Resize: got %dx%d, want 40x20", dr.Width(), dr.Height())
	}
	if !dr.Front().Set(39, 19, 'x', 0, 0, 0) {
		t.Fatal("Set at new bounds failed after Resize")
	}
}

func TestConsolidateRectsMergesOverlapping(t *testing.T) {
	dr := NewDiffRenderer(80, 24)
	dr.MarkRect(0, 0, 10, 2)
	dr.MarkRect(5, 1, 10, 2)
	rects := dr.consolidateRects()
	if len(rects) != 1 {
		t.Fatalf("got %d rects, want 1 merged", len(rects))
	}
	r := rects[0]
	if r.X != 0 || r.Y != 0 || r.Width != 15 || r.Height != 3 {
		t.Fatalf("merged rect wrong: %+v", r)
	}
}

func TestCompileSGR(t *testing.T) {
	cases := []struct {
		fg, bg uint32
		attrs  uint16
		want   string
	}{
		{0, 0, 0, ""},
		{1, 0, 0, "\x1b[31m"},
		{9, 0, 0, "\x1b[91m"},
		{100, 0, 0, "\x1b[38;5;100m"},
		{0xFF0000, 0, 0, "\x1b[38;2;255;0;0m"},
		{0, 4, 0, "\x1b[44m"},
		{0, 0, 1, "\x1b[1m"},
		{1, 4, 1, "\x1b[31;44;1m"},
	}
	for _, c := range cases {
		got := compileSGR(c.fg, c.bg, c.attrs)
		if got != c.want {
			t.Errorf("compileSGR(%d,%d,%d) = %q, want %q", c.fg, c.bg, c.attrs, got, c.want)
		}
	}
}

func TestCachedSGRStable(t *testing.T) {
	a := cachedSGR(0xFF8800, 0, 1)
	b := cachedSGR(0xFF8800, 0, 1)
	if a != b || a != compileSGR(0xFF8800, 0, 1) {
		t.Fatal("cachedSGR inconsistent with compileSGR")
	}
}

func TestItoa(t *testing.T) {
	for _, n := range []int{0, 1, 9, 10, 99, 1234} {
		if got := itoa(n); got != intToString(n) {
			t.Errorf("itoa(%d) = %q", n, got)
		}
	}
}

func intToString(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

// ---------------------------------------------------------------------------
// Benchmarks (C10)
// ---------------------------------------------------------------------------

func BenchmarkFrameBufferSet(b *testing.B) {
	fb := NewFrameBuffer(120, 40)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fb.Set(i%120, (i/120)%40, 'x', 7, 0, 0)
	}
}

func BenchmarkDiffFlushFullScreen(b *testing.B) {
	dr := NewDiffRenderer(120, 40)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch := rune('a' + i%26)
		for y := 0; y < 40; y++ {
			for x := 0; x < 120; x++ {
				dr.Front().Set(x, y, ch, 7, 0, 0)
			}
		}
		dr.MarkRect(0, 0, 120, 40)
		_ = dr.Flush()
	}
}

func BenchmarkDiffFlushSingleCell(b *testing.B) {
	dr := NewDiffRenderer(120, 40)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dr.Front().Set(60, 20, rune('a'+i%26), 7, 0, 0)
		dr.MarkRect(60, 20, 1, 1)
		_ = dr.Flush()
	}
}

func BenchmarkCachedSGR(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cachedSGR(uint32(i%16), 0, 1)
	}
}
