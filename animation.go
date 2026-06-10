package mofu

import (
	"math"
	"time"
)

// EasingFunc defines an easing curve.
type EasingFunc func(t float64) float64

// Standard easing functions
var (
	EaseLinear = func(t float64) float64 { return t }

	EaseInQuad    = func(t float64) float64 { return t * t }
	EaseOutQuad   = func(t float64) float64 { return t * (2 - t) }
	EaseInOutQuad = func(t float64) float64 {
		if t < 0.5 {
			return 2 * t * t
		}
		return -1 + (4-2*t)*t
	}

	EaseInCubic  = func(t float64) float64 { return t * t * t }
	EaseOutCubic = func(t float64) float64 {
		t--
		return t*t*t + 1
	}
	EaseInOutCubic = func(t float64) float64 {
		if t < 0.5 {
			return 4 * t * t * t
		}
		t = 2*t - 2
		return t*t*t/2 + 1
	}

	EaseInElastic = func(t float64) float64 {
		if t == 0 || t == 1 {
			return t
		}
		return -math.Pow(2, 10*(t-1)) * math.Sin((t-1.1)*5*math.Pi)
	}
	EaseOutElastic = func(t float64) float64 {
		if t == 0 || t == 1 {
			return t
		}
		return math.Pow(2, -10*t)*math.Sin((t-0.1)*5*math.Pi) + 1
	}

	EaseInBounce = func(t float64) float64 {
		return 1 - EaseOutBounce(1-t)
	}
	EaseOutBounce = func(t float64) float64 {
		if t < 1/2.75 {
			return 7.5625 * t * t
		} else if t < 2/2.75 {
			t -= 1.5 / 2.75
			return 7.5625*t*t + 0.75
		} else if t < 2.5/2.75 {
			t -= 2.25 / 2.75
			return 7.5625*t*t + 0.9375
		}
		t -= 2.625 / 2.75
		return 7.5625*t*t + 0.984375
	}

	EaseOutBack = func(t float64) float64 {
		t--
		return t*t*(2.70158*t+1.70158) + 1
	}
	EaseInBack = func(t float64) float64 {
		return t * t * (2.70158*t - 1.70158)
	}

	EaseSpring = func(t float64) float64 {
		return math.Pow(2, -10*t) * math.Sin((t-0.075)*2*math.Pi/0.3)
	}
)

// Animation represents a single animated value.
type Animation struct {
	From     float64
	To       float64
	Duration time.Duration
	Easing   EasingFunc
	OnUpdate func(value float64)
	OnDone   func()
	start    time.Time
	running  bool
}

// Animator manages multiple animations.
type Animator struct {
	animations []*Animation
	ticker     *time.Ticker
	running    bool
}

// NewAnimator creates a new animator.
func NewAnimator() *Animator {
	return &Animator{}
}

// Start begins an animation.
func (a *Animator) Start(anim *Animation) {
	anim.start = time.Now()
	anim.running = true
	a.animations = append(a.animations, anim)

	if !a.running {
		a.running = true
		a.ticker = time.NewTicker(16 * time.Millisecond) // ~60fps
		go a.runLoop()
	}
}

func (a *Animator) runLoop() {
	for range a.ticker.C {
		if !a.running {
			return
		}
		now := time.Now()
		allDone := true
		for _, anim := range a.animations {
			if !anim.running {
				continue
			}
			elapsed := now.Sub(anim.start)
			t := float64(elapsed) / float64(anim.Duration)
			if t >= 1.0 {
				t = 1.0
				anim.running = false
				if anim.OnUpdate != nil {
					anim.OnUpdate(anim.To)
				}
				if anim.OnDone != nil {
					anim.OnDone()
				}
				continue
			}
			allDone = false
			val := anim.From + (anim.To-anim.From)*anim.Easing(t)
			if anim.OnUpdate != nil {
				anim.OnUpdate(val)
			}
		}
		if allDone {
			a.running = false
			a.ticker.Stop()
		}
	}
}

// FadeIn creates a fade-in animation from 0 to 1 over the given duration.
func FadeIn(duration time.Duration, onUpdate func(value float64)) *Animation {
	return &Animation{
		From:     0,
		To:       1,
		Duration: duration,
		Easing:   EaseInOutCubic,
		OnUpdate: onUpdate,
	}
}

// FadeOut creates a fade-out animation from 1 to 0.
func FadeOut(duration time.Duration, onUpdate func(value float64), onDone func()) *Animation {
	return &Animation{
		From:     1,
		To:       0,
		Duration: duration,
		Easing:   EaseInCubic,
		OnUpdate: onUpdate,
		OnDone:   onDone,
	}
}

// SlideIn creates a slide-in animation.
func SlideIn(distance float64, duration time.Duration, onUpdate func(value float64)) *Animation {
	return &Animation{
		From:     distance,
		To:       0,
		Duration: duration,
		Easing:   EaseOutCubic,
		OnUpdate: onUpdate,
	}
}
