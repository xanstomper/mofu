package ascii

import (
	"strings"
	"testing"
)

func TestVec2(t *testing.T) {
	v := Vec2{3, 4}
	if v.Len() != 5 {
		t.Fatalf("Len = %v, want 5", v.Len())
	}

	n := v.Normalize()
	if n.Len() < 0.99 || n.Len() > 1.01 {
		t.Fatalf("Normalize Len = %v, want ~1", n.Len())
	}

	a := Vec2{1, 2}.Add(Vec2{3, 4})
	if a.X != 4 || a.Y != 6 {
		t.Fatalf("Add = %v, want {4 6}", a)
	}

	s := Vec2{2, 3}.Scale(2)
	if s.X != 4 || s.Y != 6 {
		t.Fatalf("Scale = %v, want {4 6}", s)
	}
}

func TestParticle(t *testing.T) {
	p := Particle{Life: 10, MaxLife: 10}
	if !p.Alive() {
		t.Fatal("new particle should be alive")
	}
	if p.Progress() != 0 {
		t.Fatalf("Progress = %v, want 0", p.Progress())
	}
	p.Life = 0
	if p.Alive() {
		t.Fatal("dead particle should not be alive")
	}
}

func TestEmitterSpawn(t *testing.T) {
	e := NewEmitter(10, 10, DefaultParticleConfig())
	e.Spawn()
	if e.Count() < 1 {
		t.Fatalf("count after spawn = %d, want >= 1", e.Count())
	}
}

func TestEmitterUpdate(t *testing.T) {
	e := NewEmitter(10, 10, ParticleConfig{
		Velocity: Vec2{0, -1},
		Lifetime: 5,
		Char:     '*',
		Rate:     1,
	})
	e.Spawn()
	e.Update()

	particles := e.Particles()
	if len(particles) != 1 {
		t.Fatalf("particles = %d, want 1", len(particles))
	}
	if particles[0].Pos.Y >= 10 {
		t.Fatalf("particle should have moved up, Y = %v", particles[0].Pos.Y)
	}
}

func TestEmitterLifetime(t *testing.T) {
	e := NewEmitter(10, 10, ParticleConfig{
		Velocity: Vec2{0, 0},
		Lifetime: 3,
		Char:     '*',
		Rate:     1,
	})
	e.Spawn()

	for i := 0; i < 4; i++ {
		e.Update()
	}

	if e.Count() != 0 {
		t.Fatalf("count after lifetime = %d, want 0", e.Count())
	}
}

func TestEmitterSetActive(t *testing.T) {
	e := NewEmitter(10, 10, DefaultParticleConfig())
	e.SetActive(false)
	e.Spawn()
	if e.Count() != 0 {
		t.Fatal("inactive emitter should not spawn")
	}
}

func TestSceneRender(t *testing.T) {
	scene := NewScene(20, 10)
	scene.AddEmitter(NewEmitter(10, 5, ParticleConfig{
		Velocity: Vec2{0, 0},
		Lifetime: 100,
		Char:     'X',
		Rate:     1,
	}))
	scene.Tick()

	canvas := scene.Render()
	if len(canvas) != 10 {
		t.Fatalf("canvas height = %d, want 10", len(canvas))
	}
	if len(canvas[0]) != 20 {
		t.Fatalf("canvas width = %d, want 20", len(canvas[0]))
	}
}

func TestSceneString(t *testing.T) {
	scene := NewScene(10, 3)
	scene.AddEmitter(NewEmitter(5, 1, ParticleConfig{
		Velocity: Vec2{0, 0},
		Lifetime: 100,
		Char:     'X',
		Rate:     1,
	}))
	scene.Tick()

	s := scene.String()
	if !strings.Contains(s, "X") {
		t.Fatalf("output should contain X: %q", s)
	}
}

func TestSceneShapes(t *testing.T) {
	scene := NewScene(20, 10)

	scene.AddShape(&Line{X1: 0, Y1: 0, X2: 10, Y2: 5, Char: '-', Fg: 0xFF0000})
	scene.AddShape(&Circle{CX: 10, CY: 5, Radius: 3, Char: 'O', Fg: 0x00FF00})
	scene.AddShape(&Rect{X: 1, Y: 1, W: 5, H: 3, Char: '#', Fg: 0x0000FF, Filled: false})
	scene.AddShape(&Text{X: 0, Y: 9, Text: "Hello", Fg: 0xFFFFFF})

	canvas := scene.Render()
	if len(canvas) != 10 {
		t.Fatalf("canvas height = %d", len(canvas))
	}
}

func TestSceneSetCell(t *testing.T) {
	scene := NewScene(10, 10)
	// SetCell works on the current canvas (before Render clears it)
	scene.SetCell(5, 5, '@', 0xFF0000, 0)
	// Add a shape that persists across renders
	scene.AddShape(&Text{X: 5, Y: 5, Text: "@", Fg: 0xFF0000})
	canvas := scene.Render()
	if canvas[5][5].Char != '@' {
		t.Fatalf("cell = %c, want @", canvas[5][5].Char)
	}
}

func TestPresetEmitters(t *testing.T) {
	rain := NewRainEmitter(80)
	if rain == nil {
		t.Fatal("rain emitter is nil")
	}

	fire := NewFireEmitter(10, 10)
	if fire == nil {
		t.Fatal("fire emitter is nil")
	}

	sparkle := NewSparkleEmitter(10, 10)
	if sparkle == nil {
		t.Fatal("sparkle emitter is nil")
	}

	matrix := NewMatrixEmitter(80)
	if matrix == nil {
		t.Fatal("matrix emitter is nil")
	}
}

func TestRectFilled(t *testing.T) {
	scene := NewScene(10, 10)
	scene.AddShape(&Rect{X: 1, Y: 1, W: 3, H: 3, Char: '#', Filled: true})
	canvas := scene.Render()

	// All cells inside should be filled
	for dy := 1; dy <= 3; dy++ {
		for dx := 1; dx <= 3; dx++ {
			if canvas[dy][dx].Char != '#' {
				t.Fatalf("cell [%d][%d] = %c, want #", dy, dx, canvas[dy][dx].Char)
			}
		}
	}
}

func TestCircleDraw(t *testing.T) {
	scene := NewScene(20, 20)
	scene.AddShape(&Circle{CX: 10, CY: 10, Radius: 5, Char: 'O'})
	canvas := scene.Render()

	// Circle should have drawn something
	hasChar := false
	for _, row := range canvas {
		for _, cell := range row {
			if cell.Char == 'O' {
				hasChar = true
			}
		}
	}
	if !hasChar {
		t.Fatal("circle should have drawn at least one cell")
	}
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func BenchmarkEmitterTick(b *testing.B) {
	e := NewEmitter(40, 12, ParticleConfig{
		Velocity: Vec2{0, -1},
		Lifetime: 30,
		Char:     '*',
		Rate:     5,
	})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.Spawn()
		e.Update()
	}
}

func BenchmarkSceneRender(b *testing.B) {
	scene := NewScene(80, 24)
	scene.AddEmitter(NewEmitter(40, 12, ParticleConfig{
		Velocity: Vec2{0, -1},
		Lifetime: 30,
		Char:     '*',
		Rate:     10,
	}))
	for i := 0; i < 30; i++ {
		scene.Tick()
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scene.Render()
	}
}
