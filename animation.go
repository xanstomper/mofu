package mofu

import (
	"math"
	"sync"
	"time"
)

type EasingFunc func(t float64) float64

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

	EaseInBounce  = func(t float64) float64 { return 1 - EaseOutBounce(1-t) }
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

type Tween struct {
	From, To   float64
	Duration   time.Duration
	Easing     EasingFunc
	OnUpdate   func(value float64)
	OnComplete func()
	Playing    bool
	Paused     bool
	elapsed    time.Duration
}

type Animator struct {
	tweens []*Tween
	mu     sync.Mutex
}

func NewAnimator() *Animator {
	return &Animator{}
}

func (a *Animator) Add(t *Tween) {
	a.mu.Lock()
	defer a.mu.Unlock()
	t.Playing = true
	t.elapsed = 0
	a.tweens = append(a.tweens, t)
}

func (a *Animator) Remove(t *Tween) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i, tw := range a.tweens {
		if tw == t {
			a.tweens = append(a.tweens[:i], a.tweens[i+1:]...)
			return
		}
	}
}

func (a *Animator) Clear() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.tweens = nil
}

func (a *Animator) Update(delta time.Duration) {
	a.mu.Lock()
	defer a.mu.Unlock()
	var remaining []*Tween
	for _, t := range a.tweens {
		if !t.Playing || t.Paused {
			remaining = append(remaining, t)
			continue
		}
		t.elapsed += delta
		if t.elapsed >= t.Duration {
			if t.OnUpdate != nil {
				t.OnUpdate(t.To)
			}
			if t.OnComplete != nil {
				t.OnComplete()
			}
			continue
		}
		frac := float64(t.elapsed) / float64(t.Duration)
		val := t.From + (t.To-t.From)*t.Easing(frac)
		if t.OnUpdate != nil {
			t.OnUpdate(val)
		}
		remaining = append(remaining, t)
	}
	a.tweens = remaining
}

func FadeIn(dur time.Duration, node Node) *Tween {
	s := node.Style()
	return &Tween{
		From:     0,
		To:       1,
		Duration: dur,
		Easing:   EaseInOutCubic,
		OnUpdate: func(val float64) { s.Opacity = val; node.SetDirty() },
	}
}

func FadeOut(dur time.Duration, node Node) *Tween {
	s := node.Style()
	return &Tween{
		From:     1,
		To:       0,
		Duration: dur,
		Easing:   EaseInCubic,
		OnUpdate: func(val float64) { s.Opacity = val; node.SetDirty() },
	}
}

func SlideInX(dur time.Duration, distance int, node Node) *Tween {
	s := node.Style()
	return &Tween{
		From:     float64(distance),
		To:       0,
		Duration: dur,
		Easing:   EaseOutCubic,
		OnUpdate: func(val float64) { s.OffsetX = int(val); node.SetDirty() },
	}
}

func SlideInY(dur time.Duration, distance int, node Node) *Tween {
	s := node.Style()
	return &Tween{
		From:     float64(distance),
		To:       0,
		Duration: dur,
		Easing:   EaseOutCubic,
		OnUpdate: func(val float64) { s.OffsetY = int(val); node.SetDirty() },
	}
}
