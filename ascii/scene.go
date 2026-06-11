// Package ascii provides a procedural ASCII scene engine for MOFU.
//
// The scene engine provides particle systems, procedural shapes, and a
// canvas widget for custom ASCII art rendering.
//
// Usage:
//
//	scene := ascii.NewScene(80, 24)
//	emitter := ascii.NewEmitter(40, 12, ascii.ParticleConfig{
//	    Velocity: ascii.Vec2{X: 0, Y: -1},
//	    Lifetime: 60,
//	    Char:     '*',
//	})
//	scene.AddEmitter(emitter)
//	scene.Tick() // advance one frame
//	output := scene.Render() // get the ASCII output
package ascii

import (
	"math"
	"math/rand"
	"sync"
)

// ---------------------------------------------------------------------------
// Vec2 — 2D vector for positions and velocities
// ---------------------------------------------------------------------------

// Vec2 is a 2D vector.
type Vec2 struct {
	X, Y float64
}

// Add returns v + other.
func (v Vec2) Add(other Vec2) Vec2 { return Vec2{v.X + other.X, v.Y + other.Y} }

// Scale returns v * s.
func (v Vec2) Scale(s float64) Vec2 { return Vec2{v.X * s, v.Y * s} }

// Len returns the length of v.
func (v Vec2) Len() float64 { return math.Sqrt(v.X*v.X + v.Y*v.Y) }

// Normalize returns v scaled to unit length.
func (v Vec2) Normalize() Vec2 {
	l := v.Len()
	if l == 0 {
		return Vec2{}
	}
	return Vec2{v.X / l, v.Y / l}
}

// ---------------------------------------------------------------------------
// Particle
// ---------------------------------------------------------------------------

// Particle represents a single particle in the scene.
type Particle struct {
	Pos      Vec2
	Vel      Vec2
	Life     int    // remaining life in ticks
	MaxLife  int    // initial life
	Char     rune   // display character
	Fg       uint32 // foreground color
	Bg       uint32 // background color
	Age      int    // ticks since spawn
}

// Alive reports whether the particle is still alive.
func (p *Particle) Alive() bool { return p.Life > 0 }

// Progress returns the particle's lifetime progress (0.0 = just born, 1.0 = dying).
func (p *Particle) Progress() float64 {
	if p.MaxLife == 0 {
		return 1
	}
	return float64(p.Age) / float64(p.MaxLife)
}

// ---------------------------------------------------------------------------
// ParticleConfig — emitter configuration
// ---------------------------------------------------------------------------

// ParticleConfig configures particle spawning.
type ParticleConfig struct {
	Velocity    Vec2     // base velocity
	VelSpread   Vec2     // random velocity spread
	Lifetime    int      // particle lifetime in ticks
	LifeSpread  int      // lifetime spread
	Chars       []rune   // characters to pick from
	Char        rune     // single character (if Chars is empty)
	Fg          uint32   // foreground color
	Bg          uint32   // background color
	Rate        int      // particles per tick
	Gravity     Vec2     // gravity applied each tick
	FadeIn      int      // fade-in ticks
	FadeOut     int      // fade-out ticks
}

// DefaultParticleConfig returns a default config.
func DefaultParticleConfig() ParticleConfig {
	return ParticleConfig{
		Velocity:   Vec2{0, -1},
		Lifetime:   30,
		Char:       '*',
		Fg:         0xFFFFFF,
		Rate:       1,
		FadeOut:    10,
	}
}

// ---------------------------------------------------------------------------
// Emitter — particle emitter
// ---------------------------------------------------------------------------

// Emitter spawns and manages particles.
type Emitter struct {
	mu        sync.Mutex
	X, Y      float64
	config    ParticleConfig
	particles []Particle
	rng       *rand.Rand
	active    bool
}

// NewEmitter creates a new particle emitter at the given position.
func NewEmitter(x, y float64, config ParticleConfig) *Emitter {
	if config.Chars == nil && config.Char != 0 {
		config.Chars = []rune{config.Char}
	}
	if len(config.Chars) == 0 {
		config.Chars = []rune{'*'}
	}
	return &Emitter{
		X:      x,
		Y:      y,
		config: config,
		rng:    rand.New(rand.NewSource(42)),
		active: true,
	}
}

// SetActive enables or disables the emitter.
func (e *Emitter) SetActive(active bool) {
	e.mu.Lock()
	e.active = active
	e.mu.Unlock()
}

// Spawn creates new particles based on the emission rate.
func (e *Emitter) Spawn() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.active {
		return
	}

	for i := 0; i < e.config.Rate; i++ {
		vel := Vec2{
			X: e.config.Velocity.X + (e.rng.Float64()*2-1)*e.config.VelSpread.X,
			Y: e.config.Velocity.Y + (e.rng.Float64()*2-1)*e.config.VelSpread.Y,
		}
		life := e.config.Lifetime + int((e.rng.Float64()*2-1)*float64(e.config.LifeSpread))
		if life < 1 {
			life = 1
		}

		ch := e.config.Chars[e.rng.Intn(len(e.config.Chars))]

		e.particles = append(e.particles, Particle{
			Pos:     Vec2{X: e.X, Y: e.Y},
			Vel:     vel,
			Life:    life,
			MaxLife: life,
			Char:    ch,
			Fg:      e.config.Fg,
			Bg:      e.config.Bg,
		})
	}
}

// Update advances all particles by one tick.
func (e *Emitter) Update() {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Apply gravity and velocity
	for i := range e.particles {
		p := &e.particles[i]
		p.Vel = p.Vel.Add(e.config.Gravity)
		p.Pos = p.Pos.Add(p.Vel)
		p.Life--
		p.Age++
	}

	// Remove dead particles
	alive := e.particles[:0]
	for _, p := range e.particles {
		if p.Life > 0 {
			alive = append(alive, p)
		}
	}
	e.particles = alive
}

// Particles returns a copy of the current particles.
func (e *Emitter) Particles() []Particle {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]Particle, len(e.particles))
	copy(out, e.particles)
	return out
}

// Count returns the number of active particles.
func (e *Emitter) Count() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.particles)
}

// ---------------------------------------------------------------------------
// Scene — the ASCII scene container
// ---------------------------------------------------------------------------

// Scene manages emitters and renders to an ASCII canvas.
type Scene struct {
	mu       sync.Mutex
	width    int
	height   int
	canvas   [][]Cell
	emitters []*Emitter
	shapes   []Shape
	bg       rune
}

// Cell is a single character cell in the scene.
type Cell struct {
	Char rune
	Fg   uint32
	Bg   uint32
}

// NewScene creates a new ASCII scene.
func NewScene(width, height int) *Scene {
	s := &Scene{
		width:  width,
		height: height,
		bg:     ' ',
	}
	s.clearCanvas()
	return s
}

func (s *Scene) clearCanvas() {
	s.canvas = make([][]Cell, s.height)
	for y := range s.canvas {
		s.canvas[y] = make([]Cell, s.width)
		for x := range s.canvas[y] {
			s.canvas[y][x] = Cell{Char: s.bg}
		}
	}
}

// AddEmitter adds a particle emitter to the scene.
func (s *Scene) AddEmitter(e *Emitter) {
	s.mu.Lock()
	s.emitters = append(s.emitters, e)
	s.mu.Unlock()
}

// AddShape adds a procedural shape to the scene.
func (s *Scene) AddShape(shape Shape) {
	s.mu.Lock()
	s.shapes = append(s.shapes, shape)
	s.mu.Unlock()
}

// Tick advances the scene by one frame.
func (s *Scene) Tick() {
	s.mu.Lock()
	emitters := make([]*Emitter, len(s.emitters))
	copy(emitters, s.emitters)
	s.mu.Unlock()

	for _, e := range emitters {
		e.Spawn()
		e.Update()
	}
}

// Render draws the scene to the canvas and returns it as a 2D cell grid.
func (s *Scene) Render() [][]Cell {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.clearCanvas()

	// Draw shapes
	for _, shape := range s.shapes {
		shape.Draw(s)
	}

	// Draw particles
	for _, e := range s.emitters {
		for _, p := range e.Particles() {
			x := int(math.Round(p.Pos.X))
			y := int(math.Round(p.Pos.Y))
			if x >= 0 && x < s.width && y >= 0 && y < s.height {
				s.canvas[y][x] = Cell{Char: p.Char, Fg: p.Fg, Bg: p.Bg}
			}
		}
	}

	return s.canvas
}

// String renders the scene as a string.
func (s *Scene) String() string {
	canvas := s.Render()
	result := make([]byte, 0, s.width*s.height+s.height)
	for y, row := range canvas {
		for _, cell := range row {
			if cell.Char == 0 {
				result = append(result, ' ')
			} else {
				result = append(result, string(cell.Char)...)
			}
		}
		if y < s.height-1 {
			result = append(result, '\n')
		}
	}
	return string(result)
}

// SetCell sets a cell directly on the canvas.
func (s *Scene) SetCell(x, y int, ch rune, fg, bg uint32) {
	if x >= 0 && x < s.width && y >= 0 && y < s.height {
		s.canvas[y][x] = Cell{Char: ch, Fg: fg, Bg: bg}
	}
}

// Width returns the scene width.
func (s *Scene) Width() int { return s.width }

// Height returns the scene height.
func (s *Scene) Height() int { return s.height }

// ---------------------------------------------------------------------------
// Shape — procedural shapes
// ---------------------------------------------------------------------------

// Shape can be drawn on a scene.
type Shape interface {
	Draw(s *Scene)
}

// Line draws a line between two points.
type Line struct {
	X1, Y1, X2, Y2 float64
	Char            rune
	Fg              uint32
}

func (l *Line) Draw(s *Scene) {
	dx := l.X2 - l.X1
	dy := l.Y2 - l.Y1
	steps := math.Max(math.Abs(dx), math.Abs(dy))
	if steps == 0 {
		s.SetCell(int(l.X1), int(l.Y1), l.Char, l.Fg, 0)
		return
	}
	for i := 0; i <= int(steps); i++ {
		t := float64(i) / steps
		x := int(math.Round(l.X1 + dx*t))
		y := int(math.Round(l.Y1 + dy*t))
		s.SetCell(x, y, l.Char, l.Fg, 0)
	}
}

// Circle draws a circle.
type Circle struct {
	CX, CY float64
	Radius float64
	Char   rune
	Fg     uint32
}

func (c *Circle) Draw(s *Scene) {
	for angle := 0.0; angle < 2*math.Pi; angle += 0.1 {
		x := int(math.Round(c.CX + c.Radius*math.Cos(angle)))
		y := int(math.Round(c.CY + c.Radius*math.Sin(angle)))
		s.SetCell(x, y, c.Char, c.Fg, 0)
	}
}

// Rect draws a rectangle.
type Rect struct {
	X, Y, W, H int
	Char        rune
	Fg          uint32
	Filled      bool
}

func (r *Rect) Draw(s *Scene) {
	for dy := 0; dy < r.H; dy++ {
		for dx := 0; dx < r.W; dx++ {
			if r.Filled || dy == 0 || dy == r.H-1 || dx == 0 || dx == r.W-1 {
				s.SetCell(r.X+dx, r.Y+dy, r.Char, r.Fg, 0)
			}
		}
	}
}

// Text draws text at a position.
type Text struct {
	X, Y int
	Text string
	Fg   uint32
}

func (t *Text) Draw(s *Scene) {
	x := t.X
	for _, ch := range t.Text {
		s.SetCell(x, t.Y, ch, t.Fg, 0)
		x++
	}
}

// ---------------------------------------------------------------------------
// Preset emitters
// ---------------------------------------------------------------------------

// NewRainEmitter creates a rain effect emitter.
func NewRainEmitter(width int) *Emitter {
	return NewEmitter(0, 0, ParticleConfig{
		Velocity:   Vec2{0, 1},
		VelSpread:  Vec2{0.5, 0.3},
		Lifetime:   20,
		LifeSpread: 5,
		Chars:      []rune{'|', ':', '.'},
		Fg:         0x6699CC,
		Rate:       3,
	})
}

// NewFireEmitter creates a fire effect emitter.
func NewFireEmitter(x, y float64) *Emitter {
	return NewEmitter(x, y, ParticleConfig{
		Velocity:   Vec2{0, -1.5},
		VelSpread:  Vec2{0.8, 0.5},
		Lifetime:   15,
		LifeSpread: 5,
		Chars:      []rune{'.', '*', '#', '@'},
		Fg:         0xFF6600,
		Rate:       2,
		FadeOut:    5,
	})
}

// NewSparkleEmitter creates a sparkle effect emitter.
func NewSparkleEmitter(x, y float64) *Emitter {
	return NewEmitter(x, y, ParticleConfig{
		Velocity:   Vec2{0, 0},
		VelSpread:  Vec2{2, 2},
		Lifetime:   10,
		LifeSpread: 5,
		Chars:      []rune{'.', '+', '*'},
		Fg:         0xFFFF00,
		Rate:       1,
		FadeIn:     3,
		FadeOut:    3,
	})
}

// NewMatrixEmitter creates a Matrix-style rain effect.
func NewMatrixEmitter(width int) *Emitter {
	chars := make([]rune, 26)
	for i := range chars {
		chars[i] = rune('A' + i)
	}
	return NewEmitter(0, 0, ParticleConfig{
		Velocity:   Vec2{0, 1.5},
		VelSpread:  Vec2{0, 0.5},
		Lifetime:   25,
		LifeSpread: 10,
		Chars:      chars,
		Fg:         0x00FF00,
		Rate:       2,
	})
}
