package mofu

import (
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Easing function tests
// ---------------------------------------------------------------------------

func TestEasingFunctions(t *testing.T) {
	cases := []struct {
		name   string
		fn     EasingFn
		input  float64
		expect float64
	}{
		{"EaseLinear", EaseLinear, 0.5, 0.5},
		{"EaseInQuad", EaseInQuad, 0.5, 0.25},
		{"EaseOutQuad", EaseOutQuad, 0.5, 0.75},
		{"EaseInOutQuad", EaseInOutQuad, 0.5, 0.5},
		{"EaseInCubic", EaseInCubic, 0.5, 0.125},
		{"EaseOutCubic", EaseOutCubic, 0.5, 0.875},
		{"EaseInOutCubic", EaseInOutCubic, 0.5, 0.5},
	}
	for _, c := range cases {
		got := c.fn(c.input)
		if got < c.expect-0.01 || got > c.expect+0.01 {
			t.Errorf("%s(%.1f) = %.3f, want ~%.3f", c.name, c.input, got, c.expect)
		}
	}
}

func TestEasingBoundaries(t *testing.T) {
	fns := []struct {
		name string
		fn   EasingFn
	}{
		{"EaseInQuad", EaseInQuad},
		{"EaseOutQuad", EaseOutQuad},
		{"EaseInOutQuad", EaseInOutQuad},
		{"EaseInCubic", EaseInCubic},
		{"EaseOutCubic", EaseOutCubic},
		{"EaseInExpo", EaseInExpo},
		{"EaseOutExpo", EaseOutExpo},
		{"EaseOutBounce", EaseOutBounce},
		{"EaseInBounce", EaseInBounce},
		{"EaseOutElastic", EaseOutElastic},
		{"EaseInElastic", EaseInElastic},
		{"EaseOutBack", EaseOutBack},
		{"EaseInBack", EaseInBack},
	}
	for _, f := range fns {
		at0 := f.fn(0)
		at1 := f.fn(1)
		if at0 < -0.01 || at0 > 0.01 {
			t.Errorf("%s(0) = %.3f, want ~0", f.name, at0)
		}
		if at1 < 0.99 || at1 > 1.01 {
			t.Errorf("%s(1) = %.3f, want ~1", f.name, at1)
		}
	}
}

// ---------------------------------------------------------------------------
// AnimationSpec tests
// ---------------------------------------------------------------------------

func TestDefaultAnimationSpec(t *testing.T) {
	spec := DefaultAnimationSpec()
	if spec.Duration != 300*time.Millisecond {
		t.Fatalf("duration = %v, want 300ms", spec.Duration)
	}
	if spec.Easing == nil {
		t.Fatal("easing should not be nil")
	}
}

func TestQuickSpec(t *testing.T) {
	spec := QuickSpec(500*time.Millisecond, EaseOutBounce)
	if spec.Duration != 500*time.Millisecond {
		t.Fatalf("duration = %v, want 500ms", spec.Duration)
	}
	spec2 := QuickSpec(100*time.Millisecond, nil)
	if spec2.Easing == nil {
		t.Fatal("nil easing should be defaulted")
	}
}

// ---------------------------------------------------------------------------
// Animation tests
// ---------------------------------------------------------------------------

func TestAnimationBasic(t *testing.T) {
	spec := QuickSpec(100*time.Millisecond, EaseLinear)
	anim := NewAnimation(spec, 0, 100)

	v := anim.Update(0)
	if v != 0 {
		t.Fatalf("initial value = %v, want 0", v)
	}

	v = anim.Update(50 * time.Millisecond)
	if v < 49 || v > 51 {
		t.Fatalf("halfway value = %v, want ~50", v)
	}

	v = anim.Update(50 * time.Millisecond)
	if v < 99 || v > 101 {
		t.Fatalf("final value = %v, want ~100", v)
	}
	if !anim.Done() {
		t.Fatal("animation should be done")
	}
}

func TestAnimationDelay(t *testing.T) {
	spec := AnimationSpec{
		Duration: 100 * time.Millisecond,
		Delay:    50 * time.Millisecond,
		Easing:   EaseLinear,
	}
	anim := NewAnimation(spec, 0, 100)

	v := anim.Update(25 * time.Millisecond)
	if v != 0 {
		t.Fatalf("during delay: value = %v, want 0", v)
	}

	v = anim.Update(50 * time.Millisecond)
	if v < 24 || v > 26 {
		t.Fatalf("after delay: value = %v, want ~25", v)
	}
}

func TestAnimationOnChange(t *testing.T) {
	spec := QuickSpec(100*time.Millisecond, EaseLinear)
	anim := NewAnimation(spec, 0, 100)

	var values []float64
	anim.OnChange(func(v float64) {
		values = append(values, v)
	})

	anim.Update(50 * time.Millisecond)
	anim.Update(50 * time.Millisecond)

	if len(values) != 2 {
		t.Fatalf("OnChange called %d times, want 2", len(values))
	}
}

func TestAnimationRepeatForever(t *testing.T) {
	spec := AnimationSpec{
		Duration: 100 * time.Millisecond,
		Easing:   EaseLinear,
		Repeat:   AnimRepeatForever,
	}
	anim := NewAnimation(spec, 0, 100)

	anim.Update(100 * time.Millisecond)
	if anim.Done() {
		t.Fatal("forever animation should not be done")
	}
	anim.Update(100 * time.Millisecond)
	if anim.Done() {
		t.Fatal("forever animation should not be done after 2nd cycle")
	}
}

func TestAnimationRepeatN(t *testing.T) {
	spec := AnimationSpec{
		Duration: 100 * time.Millisecond,
		Easing:   EaseLinear,
		Repeat:   AnimRepeatN,
		RepeatN:  3,
	}
	anim := NewAnimation(spec, 0, 100)

	for i := 0; i < 3; i++ {
		anim.Update(100 * time.Millisecond)
	}
	if !anim.Done() {
		t.Fatal("animation with RepeatN=3 should be done after 3 cycles")
	}
}

func TestAnimationAlternate(t *testing.T) {
	spec := AnimationSpec{
		Duration: 100 * time.Millisecond,
		Easing:   EaseLinear,
		Repeat:   AnimRepeatForever,
		AnimDir:  AnimAlternate,
	}
	anim := NewAnimation(spec, 0, 100)

	v1 := anim.Update(100 * time.Millisecond)
	if v1 < 99 {
		t.Fatalf("forward: value = %v, want ~100", v1)
	}

	v2 := anim.Update(50 * time.Millisecond)
	if v2 > 52 {
		t.Fatalf("reverse: value = %v, want ~50", v2)
	}
}

func TestAnimationReset(t *testing.T) {
	spec := QuickSpec(100*time.Millisecond, EaseLinear)
	anim := NewAnimation(spec, 0, 100)

	anim.Update(100 * time.Millisecond)
	if !anim.Done() {
		t.Fatal("should be done")
	}

	anim.Reset()
	if anim.Done() {
		t.Fatal("should not be done after reset")
	}
}

// ---------------------------------------------------------------------------
// AnimationGroup tests
// ---------------------------------------------------------------------------

func TestParallelGroup(t *testing.T) {
	spec := QuickSpec(100*time.Millisecond, EaseLinear)
	a1 := NewAnimation(spec, 0, 100)
	a2 := NewAnimation(spec, 0, 200)

	group := Parallel(a1, a2)
	group.Update(50 * time.Millisecond)

	if a1.Done() || a2.Done() {
		t.Fatal("neither should be done at 50ms")
	}

	group.Update(50 * time.Millisecond)
	if !group.Done() {
		t.Fatal("both should be done at 100ms")
	}
}

func TestSequenceGroup(t *testing.T) {
	spec := QuickSpec(100*time.Millisecond, EaseLinear)
	a1 := NewAnimation(spec, 0, 100)
	a2 := NewAnimation(spec, 0, 100)

	seq := AnimSequence(a1, a2)

	seq.Update(100 * time.Millisecond)
	if !a1.Done() {
		t.Fatal("first should be done")
	}
	if a2.Done() {
		t.Fatal("second should not be done yet")
	}

	seq.Update(100 * time.Millisecond)
	if !seq.Done() {
		t.Fatal("sequence should be done")
	}
}

func TestStagger(t *testing.T) {
	spec := QuickSpec(100*time.Millisecond, EaseLinear)
	anims := Stagger(spec, 50*time.Millisecond, []StaggerFromTo{
		{0, 100}, {0, 200}, {0, 300},
	})

	if len(anims) != 3 {
		t.Fatalf("stagger created %d anims, want 3", len(anims))
	}

	anims[0].Update(100 * time.Millisecond)
	if !anims[0].Done() {
		t.Fatal("first should be done")
	}
}

// ---------------------------------------------------------------------------
// AnimTransition tests
// ---------------------------------------------------------------------------

func TestDefaultAnimTransition(t *testing.T) {
	tr := DefaultAnimTransition()
	if tr.Enter.Duration == 0 {
		t.Fatal("enter duration should not be 0")
	}
	if tr.Exit.Duration == 0 {
		t.Fatal("exit duration should not be 0")
	}
}

func TestSlideAnimTransition(t *testing.T) {
	tr := SlideAnimTransition()
	anim := tr.Animation(AnimTransitionEnter, 0, 1)
	if anim == nil {
		t.Fatal("animation should not be nil")
	}
}

func TestAnimTransitionPhases(t *testing.T) {
	tr := DefaultAnimTransition()
	enter := tr.Animation(AnimTransitionEnter, 0, 1)
	exit := tr.Animation(AnimTransitionExit, 1, 0)
	update := tr.Animation(AnimTransitionUpdate, 0.5, 0.8)

	if enter == nil || exit == nil || update == nil {
		t.Fatal("all transition animations should be created")
	}
}

// ---------------------------------------------------------------------------
// Helper tests
// ---------------------------------------------------------------------------

func TestAnimateValue(t *testing.T) {
	animator := NewAnimator()
	var completed bool
	AnimateValue(animator, 0, 100, 100*time.Millisecond, EaseLinear, func(v float64) {
		completed = true
	})
	animator.Update(50)
	// Check current value via Animator
	v, ok := animator.CurrentValue(0)
	if !ok {
		t.Fatal("CurrentValue should return true for active tween")
	}
	if v < 49 || v > 52 {
		t.Fatalf("AnimateValue: value = %v, want ~50", v)
	}
	// Complete the tween
	animator.Update(50)
	if !completed {
		t.Fatal("completion callback should have fired")
	}
}

func TestAnimatePosition(t *testing.T) {
	s := AnimatePosition(0, 100)
	if s.Target != 100 {
		t.Fatalf("target = %v, want 100", s.Target)
	}
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func BenchmarkEasingFunctions(b *testing.B) {
	fns := []struct {
		name string
		fn   EasingFn
	}{
		{"Linear", EaseLinear},
		{"InOutQuad", EaseInOutQuad},
		{"OutBounce", EaseOutBounce},
		{"OutElastic", EaseOutElastic},
	}
	for _, f := range fns {
		b.Run(f.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				f.fn(float64(i%100) / 100)
			}
		})
	}
}

func BenchmarkAnimationUpdate(b *testing.B) {
	spec := QuickSpec(1000*time.Millisecond, EaseInOutQuad)
	anim := NewAnimation(spec, 0, 100)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		anim.Update(time.Millisecond)
	}
}
