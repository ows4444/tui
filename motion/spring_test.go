package motion

import (
	"math"
	"testing"
	"time"
)

const frame = 16 * time.Millisecond

// TestSpringConvergesWithoutOvershoot proves criterion #82: a critically
// damped Spring stepped at a fixed dt reaches its target, reports Done, and
// never passes it.
func TestSpringConvergesWithoutOvershoot(t *testing.T) {
	s := NewSpring(0, 200)
	s.SetTarget(1)
	for i := 0; i < 1000 && !s.Done(); i++ {
		if v := s.Step(frame); v > 1 {
			t.Fatalf("step %d overshot: %v", i, v)
		}
	}
	if !s.Done() || s.Value != 1 || s.Velocity != 0 {
		t.Fatalf("not settled: value=%v velocity=%v done=%v", s.Value, s.Velocity, s.Done())
	}
	if s.Step(frame) != 1 {
		t.Error("settled Spring moved")
	}
}

// TestSpringUnderdampedOvershootsThenSettles supports criterion #82: an
// underdamped Spring does oscillate past the target, and still converges.
func TestSpringUnderdampedOvershootsThenSettles(t *testing.T) {
	s := &Spring{Stiffness: 200, Damping: 5}
	s.SetTarget(1)
	peak := 0.0
	for i := 0; i < 5000 && !s.Done(); i++ {
		peak = math.Max(peak, s.Step(frame))
	}
	if peak <= 1 || !s.Done() || s.Value != 1 {
		t.Fatalf("peak=%v value=%v done=%v", peak, s.Value, s.Done())
	}
}

// TestSpringOverdampedConverges supports criterion #82 for the third damping
// regime.
func TestSpringOverdampedConverges(t *testing.T) {
	s := &Spring{Stiffness: 100, Damping: 50}
	s.SetTarget(1)
	for i := 0; i < 5000 && !s.Done(); i++ {
		if v := s.Step(frame); v > 1 {
			t.Fatalf("overshot: %v", v)
		}
	}
	if !s.Done() || s.Value != 1 {
		t.Fatalf("value=%v done=%v", s.Value, s.Done())
	}
}

// TestSpringStepSizeIndependent supports criterion #82: the closed-form step
// gives the same value however time is divided.
func TestSpringStepSizeIndependent(t *testing.T) {
	a, b := NewSpring(0, 100), NewSpring(0, 100)
	a.SetTarget(1)
	b.SetTarget(1)
	a.Step(100 * time.Millisecond)
	for i := 0; i < 10; i++ {
		b.Step(10 * time.Millisecond)
	}
	if math.Abs(a.Value-b.Value) > 1e-9 {
		t.Errorf("one step %v, ten steps %v", a.Value, b.Value)
	}
}

// TestTransitionRetargetIsContinuous proves criterion #83: changing the
// target mid-flight keeps Value and Velocity, so the next frame carries on
// from the current value with no jump.
func TestTransitionRetargetIsContinuous(t *testing.T) {
	tr := NewTransition(0, 300*time.Millisecond, Normal)
	tr.To(10)
	for i := 0; i < 8; i++ {
		tr.Step(frame)
	}
	before, vel := tr.Value(), tr.spring.Velocity
	if before <= 0 || before >= 10 || vel <= 0 {
		t.Fatalf("not mid-flight: value=%v velocity=%v", before, vel)
	}
	tr.To(-5)
	if tr.Value() != before || tr.spring.Velocity != vel || tr.Target() != -5 {
		t.Fatalf("retarget changed state: value=%v velocity=%v", tr.Value(), tr.spring.Velocity)
	}
	// One tiny step must move by about velocity*dt, not jump toward -5.
	dt := time.Millisecond
	got := tr.Step(dt)
	if math.Abs(got-before) > math.Abs(vel)*dt.Seconds()*2+1e-3 {
		t.Errorf("jump after retarget: %v -> %v", before, got)
	}
	for i := 0; i < 1000 && !tr.Done(); i++ {
		tr.Step(frame)
	}
	if !tr.Done() || tr.Value() != -5 {
		t.Errorf("did not settle on new target: %v", tr.Value())
	}
}

// TestReducedJumpsToTarget proves criterion #86: under Reduced, a Spring and
// a Transition reach the target with no intermediate frames.
func TestReducedJumpsToTarget(t *testing.T) {
	s := NewSpring(0, 100)
	s.Motion = Reduced
	s.SetTarget(1)
	if s.Value != 1 || !s.Done() || s.Step(frame) != 1 {
		t.Errorf("reduced Spring: value=%v", s.Value)
	}
	tr := NewTransition(0, time.Second, Reduced)
	tr.To(7)
	if tr.Value() != 7 || !tr.Done() || tr.Step(frame) != 7 {
		t.Errorf("reduced Transition: value=%v", tr.Value())
	}
	// Even if the preference flips after a target was set, Step must not
	// produce an intermediate value.
	s2 := NewSpring(0, 100)
	s2.SetTarget(1)
	s2.Motion = Reduced
	if s2.Step(frame) != 1 {
		t.Error("Step under Reduced was not a jump")
	}
}
