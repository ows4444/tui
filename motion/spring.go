package motion

import (
	"math"
	"time"
)

// defaultEpsilon is the settle tolerance a Spring uses when Epsilon is 0.
const defaultEpsilon = 1e-3

// Spring moves Value toward Target as a damped harmonic oscillator of unit
// mass: Stiffness is the spring constant, Damping the friction. Drive it
// from a tick: call Step with the time since the last tick and render
// Value. Step solves the motion in closed form, so the result does not
// depend on the step size and never blows up on a long frame.
//
// Damping of 2*sqrt(Stiffness) is critical damping: the fastest approach
// that does not overshoot. Less oscillates around Target, more is sluggish.
// Changing Target mid-flight (SetTarget) keeps Value and Velocity, so the
// motion stays continuous.
//
// A Spring honours reduced motion: with Motion set to Reduced it jumps to
// Target at once, with no intermediate frames. The zero Spring has no
// stiffness and also jumps to Target.
type Spring struct {
	Stiffness, Damping float64
	// Epsilon is how close to Target, in both distance and speed, counts as
	// settled. 0 means 0.001.
	Epsilon float64
	Motion  Preference

	Value, Velocity float64
	target          float64
}

// NewSpring returns a critically damped Spring at rest at value, with the
// given stiffness (larger is faster).
func NewSpring(value, stiffness float64) *Spring {
	return &Spring{
		Stiffness: stiffness,
		Damping:   2 * math.Sqrt(stiffness),
		Value:     value,
		target:    value,
	}
}

// Target reports the value the Spring is moving toward.
func (s *Spring) Target() float64 { return s.target }

// SetTarget retargets the Spring. Value and Velocity are kept, so a change
// mid-flight continues from where the Spring is, at the speed it has. Under
// reduced motion it jumps to the new target instead.
func (s *Spring) SetTarget(target float64) {
	s.target = target
	if s.Motion.Reduced() {
		s.settle()
	}
}

func (s *Spring) settle() { s.Value, s.Velocity = s.target, 0 }

// Done reports whether the Spring has settled on Target: within Epsilon in
// both distance and speed, or always under reduced motion.
func (s *Spring) Done() bool {
	if s.Motion.Reduced() {
		return true
	}
	eps := s.Epsilon
	if eps <= 0 {
		eps = defaultEpsilon
	}
	return math.Abs(s.Value-s.target) <= eps && math.Abs(s.Velocity) <= eps
}

// Step advances the Spring by dt and returns the new Value. Once settled it
// snaps to Target exactly and stays there. A non-positive dt changes
// nothing. Under reduced motion, or without positive Stiffness, it jumps to
// Target.
func (s *Spring) Step(dt time.Duration) float64 {
	if s.Motion.Reduced() || !(s.Stiffness > 0) {
		s.settle()
		return s.Value
	}
	if s.Done() {
		s.settle()
		return s.Value
	}
	if dt <= 0 {
		return s.Value
	}
	t := dt.Seconds()
	x, v := s.Value-s.target, s.Velocity
	w0 := math.Sqrt(s.Stiffness)
	zeta := s.Damping / (2 * w0)
	switch {
	case math.Abs(zeta-1) < 1e-9: // critically damped
		e := math.Exp(-w0 * t)
		b := v + w0*x
		x, v = e*(x+b*t), e*(v-b*w0*t)
	case zeta < 1: // underdamped: oscillates
		a := zeta * w0
		wd := w0 * math.Sqrt(1-zeta*zeta)
		e := math.Exp(-a * t)
		cos, sin := math.Cos(wd*t), math.Sin(wd*t)
		b := (v + a*x) / wd
		x, v = e*(x*cos+b*sin), e*((b*wd-a*x)*cos-(x*wd+a*b)*sin)
	default: // overdamped: two decaying modes
		q := w0 * math.Sqrt(zeta*zeta-1)
		r1, r2 := -zeta*w0+q, -zeta*w0-q
		c2 := (v - r1*x) / (r2 - r1)
		c1 := x - c2
		e1, e2 := math.Exp(r1*t), math.Exp(r2*t)
		x, v = c1*e1+c2*e2, c1*r1*e1+c2*r2*e2
	}
	s.Value, s.Velocity = s.target+x, v
	if s.Done() {
		s.settle()
	}
	return s.Value
}

// Transition is an interruptible move to a target: a critically damped
// Spring sized by duration rather than stiffness. Retarget it with To at any
// time; it carries on from the current Value and Velocity, so an interrupted
// transition never jumps or stops dead. Under reduced motion To jumps to the
// target. The zero Transition is not usable; build one with NewTransition.
type Transition struct {
	spring Spring
}

// NewTransition returns a Transition at rest at value that settles on a
// target in about duration. A non-positive duration jumps at once. motion is
// the reduced-motion preference.
func NewTransition(value float64, duration time.Duration, motion Preference) *Transition {
	tr := &Transition{spring: Spring{Value: value, target: value, Motion: motion}}
	if duration > 0 {
		// A critically damped spring is within 1% of its target after
		// about 6.6/omega seconds.
		w := 6.6 / duration.Seconds()
		tr.spring.Stiffness = w * w
		tr.spring.Damping = 2 * w
	}
	return tr
}

// To retargets the Transition from its current Value and Velocity.
func (tr *Transition) To(target float64) { tr.spring.SetTarget(target) }

// Step advances the Transition by dt and returns the new Value.
func (tr *Transition) Step(dt time.Duration) float64 { return tr.spring.Step(dt) }

// Value reports the current value.
func (tr *Transition) Value() float64 { return tr.spring.Value }

// Target reports the value the Transition is moving toward.
func (tr *Transition) Target() float64 { return tr.spring.target }

// Done reports whether the Transition has settled on its target.
func (tr *Transition) Done() bool { return tr.spring.Done() }
