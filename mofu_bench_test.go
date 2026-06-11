package mofu

import (
	"testing"
)

// Benchmarks demonstrating MOFU's performance characteristics.

func BenchmarkColorHex2(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Hex("ff69b4")
	}
}

func BenchmarkColorRGB2(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RGB(255, 105, 180)
	}
}

func BenchmarkStyleFg2(b *testing.B) {
	s := DefaultStyle()
	c := Hex("ff69b4")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Fg(c)
	}
}

func BenchmarkStyleChain2(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DefaultStyle().
			Fg(Hex("ff69b4")).
			Bg(Hex("1e1e2e")).
			WithAttrs(AttrBold)
	}
}

func BenchmarkStateGraphSet2(b *testing.B) {
	g := NewStateGraph()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.Set("key", i)
	}
}

func BenchmarkStateGraphGet2(b *testing.B) {
	g := NewStateGraph()
	g.Set("key1", 42)
	g.Set("key2", "hello")
	g.Set("key3", 3.14)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.Get("key1")
	}
}

func BenchmarkSceneBufferSet2(b *testing.B) {
	r := NewSceneBuffer(120, 40)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Set(i%120, i%40, 'X', Hex("ff69b4"), ColorBlack, 0)
	}
}

func BenchmarkSceneBufferClear2(b *testing.B) {
	r := NewSceneBuffer(120, 40)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Clear()
	}
}

func BenchmarkANSICode2(b *testing.B) {
	c := Hex("ff69b4")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.foreground()
	}
}

func BenchmarkMinimal2(b *testing.B) {
	m := &Minimal{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.SetBounds(Rect{Width: 80, Height: 24})
		_ = m.Bounds()
		_ = m.Dirty()
	}
}
