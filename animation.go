package mofu

import (
	"math"
	"sync"
)

// ---------------------------------------------------------------------------
// Vec2 — 2D point for layout and animation
// ---------------------------------------------------------------------------

// Vec2 is a 2D point used by layout, scroll, and animation systems.
type Vec2 struct {
	X, Y float64
}

// Vec2XY creates a Vec2 from x, y coordinates.
func Vec2XY(x, y float64) Vec2 { return Vec2{X: x, Y: y} }

// ---------------------------------------------------------------------------
// Easing Functions
// ---------------------------------------------------------------------------
// Animator (Anthology Ch.6 §6.1)
// runtime.go expects: type Animator struct, func NewAnimator() *Animator,
// method func (a *Animator) Update(deltaMs uint64)
// ---------------------------------------------------------------------------

// Animator manages active tween/spring animations for a Program.
type Animator struct {
	mu       sync.Mutex
	nextID   uint64
	tweens   map[uint64]*TweenEntry
	springs  map[uint64]*SpringEntry
	finished []uint64 // IDs removed this tick (for reuse safety)
}

// TweenEntry holds state for a single tween.
type TweenEntry struct {
	From, To   float64
	DurationMs uint64
	ElapsedMs  uint64
	Easing     EasingFn
	Apply      func(v float64) // callback to apply value each frame
}

// SpringEntry holds state for a single spring.
type SpringEntry struct {
	Spring *Spring
}

// NewAnimator returns an empty Animator.
func NewAnimator() *Animator {
	return &Animator{
		tweens:  make(map[uint64]*TweenEntry),
		springs: make(map[uint64]*SpringEntry),
	}
}

// Update advances all animations by deltaMs milliseconds.
// For completed tweens, Apply is called with the final value before removal.
// Returns IDs of animations that finished this tick.
func (a *Animator) Update(deltaMs uint64) []uint64 {
	a.mu.Lock()
	defer a.mu.Unlock()

	if deltaMs == 0 {
		return nil
	}

	var done []uint64

	for id, tw := range a.tweens {
		tw.ElapsedMs += deltaMs
		if tw.ElapsedMs >= tw.DurationMs {
			if tw.Apply != nil {
				tw.Apply(tw.To)
			}
			delete(a.tweens, id)
			done = append(done, id)
		}
	}
	for id, se := range a.springs {
		se.Spring.Advance(deltaMs)
		if se.Spring.IsAtRest() {
			delete(a.springs, id)
			done = append(done, id)
		}
	}
	if len(done) > 0 {
		a.finished = append(a.finished[:0], done...)
	}
	return done
}

// AddTween registers a tween and returns its ID.
// Caller supplies Apply(cb) to receive each frame's value.
func (a *Animator) AddTween(from, to float64, durationMs uint64, easing EasingFn, apply func(v float64)) uint64 {
	a.mu.Lock()
	defer a.mu.Unlock()

	if easing == nil {
		easing = EaseLinear
	}
	id := a.nextID
	a.nextID++

	a.tweens[id] = &TweenEntry{
		From:       from,
		To:         to,
		DurationMs: durationMs,
		ElapsedMs:  0,
		Easing:     easing,
		Apply:      apply,
	}
	return id
}

// AddSpring registers a spring animation and returns its ID.
func (a *Animator) AddSpring(s *Spring) uint64 {
	a.mu.Lock()
	defer a.mu.Unlock()

	id := a.nextID
	a.nextID++

	a.springs[id] = &SpringEntry{Spring: s}
	return id
}

// Remove cancels an animation by ID.
func (a *Animator) Remove(id uint64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.tweens, id)
	delete(a.springs, id)
}

// CurrentValue returns the current interpolated value for a tween by ID.
// Returns (0, false) if not found or not a tween.
func (a *Animator) CurrentValue(id uint64) (float64, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	tw, ok := a.tweens[id]
	if !ok {
		return 0, false
	}
	prog := float64(tw.ElapsedMs) / float64(tw.DurationMs)
	if prog > 1 {
		prog = 1
	}
	if prog < 0 {
		prog = 0
	}
	return tw.From + (tw.To-tw.From)*tw.Easing(prog), true
}

// ---------------------------------------------------------------------------
// Easing Functions
// ---------------------------------------------------------------------------

// EasingFn maps normalised progress t∈[0,1] to [0,1].
type EasingFn func(t float64) float64

func EaseLinear(t float64) float64 { return t }

func EaseInOutBack(t float64) float64 {
	const s = 1.70158 * 1.525
	if t < 0.5 {
		return (math.Pow(2*t, 2) * ((s+1)*2*t - s)) / 2
	}
	return (math.Pow(2*t-2, 2)*((s+1)*(t*2-2)+s)+2) / 2
}

func EaseOutQuint(t float64) float64 {
	t--
	return 1 + t*t*t*t*t
}

// ---------------------------------------------------------------------------
// Spring — damped spring physics for smooth motion
// ---------------------------------------------------------------------------

// Spring provides damped-spring interpolation for a single float64 value.
type Spring struct {
	Current   float64
	Target    float64
	Velocity  float64
	Stiffness float64
	Damping   float64
	Mass      float64
}

// NewSpring creates a spring anchored at current.
func NewSpring(current float64) *Spring {
	return &Spring{
		Current:   current,
		Target:    current,
		Stiffness: 120,
		Damping:   14,
		Mass:      1,
	}
}

// SetTarget changes the spring's resting value.
func (s *Spring) SetTarget(t float64) { s.Target = t }

// Advance advances the spring simulation by deltaMs (milliseconds).
func (s *Spring) Advance(deltaMs uint64) {
	dt := float64(deltaMs) / 1000
	if dt <= 0 {
		return
	}

	displacement := s.Current - s.Target
	springForce := -s.Stiffness * displacement
	dampingForce := -s.Damping * s.Velocity
	acc := (springForce + dampingForce) / s.Mass

	s.Velocity += acc * dt
	s.Current += s.Velocity * dt
}

// IsAtRest reports whether the spring has settled near its target.
func (s *Spring) IsAtRest() bool {
	return math.Abs(s.Velocity) < 0.001 && math.Abs(s.Current-s.Target) < 0.001
}
