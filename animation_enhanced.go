package mofu

import (
	"math"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Easing Functions — standard Robert Penner easing curves
// ---------------------------------------------------------------------------

// EaseInQuad accelerates from zero velocity.
func EaseInQuad(t float64) float64 { return t * t }

// EaseOutQuad decelerates to zero velocity.
func EaseOutQuad(t float64) float64 { return t * (2 - t) }

// EaseInOutQuad acceleration until halfway, then deceleration.
func EaseInOutQuad(t float64) float64 {
	if t < 0.5 {
		return 2 * t * t
	}
	return -1 + (4-2*t)*t
}

// EaseInCubic accelerates from zero velocity.
func EaseInCubic(t float64) float64 { return t * t * t }

// EaseOutCubic decelerates to zero velocity.
func EaseOutCubic(t float64) float64 {
	t--
	return t*t*t + 1
}

// EaseInOutCubic acceleration until halfway, then deceleration.
func EaseInOutCubic(t float64) float64 {
	if t < 0.5 {
		return 4 * t * t * t
	}
	return (t-1)*(2*t-2)*(2*t-2) + 1
}

// EaseInExpo accelerates exponentially.
func EaseInExpo(t float64) float64 {
	if t == 0 {
		return 0
	}
	return math.Pow(2, 10*(t-1))
}

// EaseOutExpo decelerates exponentially.
func EaseOutExpo(t float64) float64 {
	if t == 1 {
		return 1
	}
	return 1 - math.Pow(2, -10*t)
}

// EaseInOutExpo acceleration until halfway, then deceleration.
func EaseInOutExpo(t float64) float64 {
	if t == 0 || t == 1 {
		return t
	}
	if t < 0.5 {
		return 0.5 * math.Pow(2, 20*t-10)
	}
	return 1 - 0.5*math.Pow(2, -20*t+10)
}

// EaseOutBounce decelerates with a bounce effect.
func EaseOutBounce(t float64) float64 {
	if t < 1/2.75 {
		return 7.5625 * t * t
	}
	if t < 2/2.75 {
		t -= 1.5 / 2.75
		return 7.5625*t*t + 0.75
	}
	if t < 2.5/2.75 {
		t -= 2.25 / 2.75
		return 7.5625*t*t + 0.9375
	}
	t -= 2.625 / 2.75
	return 7.5625*t*t + 0.984375
}

// EaseInBounce accelerates with a bounce effect.
func EaseInBounce(t float64) float64 { return 1 - EaseOutBounce(1-t) }

// EaseInOutBounce bounce effect in both directions.
func EaseInOutBounce(t float64) float64 {
	if t < 0.5 {
		return EaseInBounce(t*2) * 0.5
	}
	return EaseOutBounce(t*2-1)*0.5 + 0.5
}

// EaseOutElastic decelerates with an elastic snap.
func EaseOutElastic(t float64) float64 {
	if t == 0 || t == 1 {
		return t
	}
	return math.Pow(2, -10*t)*math.Sin((t-0.1)*5*math.Pi) + 1
}

// EaseInElastic accelerates with an elastic snap.
func EaseInElastic(t float64) float64 {
	if t == 0 || t == 1 {
		return t
	}
	return 1 - EaseOutElastic(1-t)
}

// EaseOutBack decelerates overshooting then settling.
func EaseOutBack(t float64) float64 {
	const s = 1.70158
	t--
	return t*t*((s+1)*t+s) + 1
}

// EaseInBack accelerates pulling back then shooting forward.
func EaseInBack(t float64) float64 {
	const s = 1.70158
	return t * t * ((s+1)*t - s)
}

// ---------------------------------------------------------------------------
// AnimationSpec — declarative animation configuration
// ---------------------------------------------------------------------------

// AnimRepeatMode controls how animations repeat.
type AnimRepeatMode int

const (
	AnimRepeatNone    AnimRepeatMode = iota // play once
	AnimRepeatForever                       // loop forever
	AnimRepeatN                             // loop N times
)

// AnimDirection controls the animation direction.
type AnimDirection int

const (
	AnimForward    AnimDirection = iota // 0 to 1
	AnimReverse                         // 1 to 0
	AnimAlternate                       // forward then reverse
)

// AnimationSpec is a declarative animation configuration.
type AnimationSpec struct {
	Duration  time.Duration
	Delay     time.Duration
	Easing    EasingFn
	Repeat    AnimRepeatMode
	RepeatN   int
	AnimDir   AnimDirection
}

// DefaultAnimationSpec returns a spec with sensible defaults.
func DefaultAnimationSpec() AnimationSpec {
	return AnimationSpec{
		Duration: 300 * time.Millisecond,
		Easing:   EaseInOutQuad,
	}
}

// QuickSpec creates a spec with just duration and easing.
func QuickSpec(duration time.Duration, easing EasingFn) AnimationSpec {
	if easing == nil {
		easing = EaseInOutQuad
	}
	return AnimationSpec{Duration: duration, Easing: easing}
}

// ---------------------------------------------------------------------------
// Animation — a running animation instance
// ---------------------------------------------------------------------------

// Animation represents a running animation that produces float64 values.
type Animation struct {
	mu       sync.Mutex
	spec     AnimationSpec
	from, to float64
	elapsed  time.Duration
	repeats  int
	reverse  bool
	done     bool
	onChange []func(float64)
}

// NewAnimation creates a new animation from a spec.
func NewAnimation(spec AnimationSpec, from, to float64) *Animation {
	return &Animation{
		spec: spec,
		from: from,
		to:   to,
	}
}

// OnChange registers a callback for each frame's value.
func (a *Animation) OnChange(fn func(float64)) {
	a.mu.Lock()
	a.onChange = append(a.onChange, fn)
	a.mu.Unlock()
}

// Update advances the animation by delta. Returns the current value.
func (a *Animation) Update(delta time.Duration) float64 {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.done {
		if a.reverse {
			return a.from
		}
		return a.to
	}

	a.elapsed += delta

	if a.elapsed < a.spec.Delay {
		return a.from
	}
	active := a.elapsed - a.spec.Delay

	progress := float64(active) / float64(a.spec.Duration)
	if progress > 1 {
		progress = 1
	}

	effectiveProgress := progress
	if a.reverse && a.spec.AnimDir == AnimAlternate {
		effectiveProgress = 1 - progress
	} else if a.spec.AnimDir == AnimReverse {
		effectiveProgress = 1 - progress
	}

	eased := effectiveProgress
	if a.spec.Easing != nil {
		eased = a.spec.Easing(effectiveProgress)
	}

	value := a.from + (a.to-a.from)*eased

	for _, fn := range a.onChange {
		fn(value)
	}

	if progress >= 1 {
		switch a.spec.Repeat {
		case AnimRepeatNone:
			a.done = true
		case AnimRepeatForever:
			a.elapsed = a.spec.Delay
			if a.spec.AnimDir == AnimAlternate {
				a.reverse = !a.reverse
			}
		case AnimRepeatN:
			a.repeats++
			if a.repeats >= a.spec.RepeatN {
				a.done = true
			} else {
				a.elapsed = a.spec.Delay
				if a.spec.AnimDir == AnimAlternate {
					a.reverse = !a.reverse
				}
			}
		}
	}

	return value
}

// Done reports whether the animation has completed.
func (a *Animation) Done() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.done
}

// Reset restarts the animation from the beginning.
func (a *Animation) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.elapsed = 0
	a.repeats = 0
	a.reverse = false
	a.done = false
}

// ---------------------------------------------------------------------------
// AnimationGroup — parallel composition
// ---------------------------------------------------------------------------

// AnimationGroup runs multiple animations simultaneously.
type AnimationGroup struct {
	mu         sync.Mutex
	animations []*Animation
}

// Parallel creates a group that runs all animations simultaneously.
func Parallel(anims ...*Animation) *AnimationGroup {
	return &AnimationGroup{animations: anims}
}

// Update advances all animations in the group.
func (g *AnimationGroup) Update(delta time.Duration) {
	g.mu.Lock()
	anims := make([]*Animation, len(g.animations))
	copy(anims, g.animations)
	g.mu.Unlock()

	for _, a := range anims {
		a.Update(delta)
	}
}

// Done reports whether all animations in the group are complete.
func (g *AnimationGroup) Done() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, a := range g.animations {
		if !a.Done() {
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// Sequence — sequential composition
// ---------------------------------------------------------------------------

// AnimSequenceGroup runs animations one after another.
type AnimSequenceGroup struct {
	mu         sync.Mutex
	animations []*Animation
	current    int
}

// AnimSequence creates a group that runs animations in order.
func AnimSequence(anims ...*Animation) *AnimSequenceGroup {
	return &AnimSequenceGroup{animations: anims}
}

// Update advances the current animation in the sequence.
func (g *AnimSequenceGroup) Update(delta time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.current >= len(g.animations) {
		return
	}

	g.animations[g.current].Update(delta)
	if g.animations[g.current].Done() {
		g.current++
	}
}

// Done reports whether all animations in the sequence are complete.
func (g *AnimSequenceGroup) Done() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.current >= len(g.animations)
}

// ---------------------------------------------------------------------------
// Stagger — staggered delay composition
// ---------------------------------------------------------------------------

// StaggerFromTo is a from/to value pair for stagger animations.
type StaggerFromTo struct {
	From, To float64
}

// Stagger creates animations with staggered delays.
func Stagger(spec AnimationSpec, staggerDelay time.Duration, fromTos []StaggerFromTo) []*Animation {
	anims := make([]*Animation, len(fromTos))
	for i, ft := range fromTos {
		s := spec
		s.Delay = time.Duration(i) * staggerDelay
		anims[i] = NewAnimation(s, ft.From, ft.To)
	}
	return anims
}

// ---------------------------------------------------------------------------
// AnimTransition — enter/exit animations for widgets
// ---------------------------------------------------------------------------

// AnimTransitionType identifies the transition phase.
type AnimTransitionType int

const (
	AnimTransitionEnter  AnimTransitionType = iota
	AnimTransitionExit
	AnimTransitionUpdate
)

// AnimTransition defines enter/exit animations for a widget.
type AnimTransition struct {
	Enter  AnimationSpec
	Exit   AnimationSpec
	Update AnimationSpec
}

// DefaultAnimTransition returns a transition with fade animations.
func DefaultAnimTransition() AnimTransition {
	return AnimTransition{
		Enter:  QuickSpec(200*time.Millisecond, EaseOutCubic),
		Exit:   QuickSpec(150*time.Millisecond, EaseInCubic),
		Update: QuickSpec(200*time.Millisecond, EaseInOutQuad),
	}
}

// SlideAnimTransition returns a transition with slide animations.
func SlideAnimTransition() AnimTransition {
	return AnimTransition{
		Enter:  QuickSpec(300*time.Millisecond, EaseOutBack),
		Exit:   QuickSpec(200*time.Millisecond, EaseInCubic),
		Update: QuickSpec(250*time.Millisecond, EaseInOutQuad),
	}
}

// Animation creates an Animation for the given transition phase.
func (t AnimTransition) Animation(typ AnimTransitionType, from, to float64) *Animation {
	switch typ {
	case AnimTransitionEnter:
		return NewAnimation(t.Enter, from, to)
	case AnimTransitionExit:
		return NewAnimation(t.Exit, from, to)
	default:
		return NewAnimation(t.Update, from, to)
	}
}

// ---------------------------------------------------------------------------
// Widget animation helpers
// ---------------------------------------------------------------------------

// AnimateValue creates a tween that updates a setter function each frame.
func AnimateValue(animator *Animator, from, to float64, duration time.Duration, easing EasingFn, setter func(float64)) uint64 {
	return animator.AddTween(from, to, uint64(duration.Milliseconds()), easing, setter)
}

// AnimateOpacity creates a fade animation (0-1 range).
func AnimateOpacity(animator *Animator, from, to float64, duration time.Duration, setter func(float64)) uint64 {
	return AnimateValue(animator, from, to, duration, EaseInOutQuad, setter)
}

// AnimatePosition creates a position animation with spring physics.
func AnimatePosition(current, target float64) *Spring {
	s := NewSpring(current)
	s.SetTarget(target)
	return s
}
