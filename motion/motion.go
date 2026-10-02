// Package motion provides a small, caller-side reduced-motion preference.
//
// Two designs were considered for reduced-motion support in this repo
// (spec #18):
//
//  1. Thread an explicit `reduced bool` through every animated widget's
//     Start()/constructor (spinner.Model, streamtext.Model, skeleton.Model),
//     consistent with how theme.Theme is threaded explicitly everywhere in
//     this codebase.
//  2. A small standalone `motion` package exposing a Preference value that
//     the caller reads once, deciding whether to call Start() at all (or to
//     call streamtext's Skip() for instant reveal) — without touching any
//     existing animated widget's signature.
//
// Decision #11 selected design (2): spinner.Model and skeleton.Model already
// render a valid static single frame even when Start() is never called
// (their View() works pre-Start), and streamtext.Model already has Skip()
// (which jumps shown to total() for an instant reveal). A caller checking a
// motion.Preference before deciding Start() vs. Skip()/no-op fully satisfies
// reduced motion with zero changes to spinner.go, streamtext.go, or
// skeleton.go. See decision #11 for the full rationale.
//
// The package also has easing functions (Ease, from Linear to OutBack) and
// Tween, which interpolates a value over time and skips straight to the end
// value under a Reduced preference.
package motion

import "os"

// Preference reports whether the caller should prefer reduced motion (e.g.
// skip animation and render a static frame) or normal motion (animate as
// usual).
type Preference int

const (
	// Normal indicates animations should run as usual.
	Normal Preference = iota
	// Reduced indicates the caller should avoid animation: e.g. never call
	// an animated widget's Start(), or call streamtext.Model.Skip() for an
	// instant reveal instead of animating.
	Reduced
)

// reducedMotionEnvVar is the environment variable this package checks to
// detect a reduced-motion preference. Following the informal NO_COLOR-style
// convention used by some CLIs, setting it to any non-empty value requests
// reduced motion.
const reducedMotionEnvVar = "NO_ANIMATION"

// standardReducedMotionEnvVar is the second variable Detect honours,
// REDUCE_MOTION, named after the platform "reduce motion" accessibility
// setting. No variable is standardised across terminals; either one being
// non-empty requests reduced motion.
const standardReducedMotionEnvVar = "REDUCE_MOTION"

// Reduced reports whether p represents a reduced-motion preference.
func (p Preference) Reduced() bool {
	return p == Reduced
}

// Detect returns Reduced if the NO_ANIMATION or REDUCE_MOTION environment
// variable is set to a non-empty value, and Normal otherwise.
func Detect() Preference {
	if os.Getenv(reducedMotionEnvVar) != "" || os.Getenv(standardReducedMotionEnvVar) != "" {
		return Reduced
	}
	return Normal
}
