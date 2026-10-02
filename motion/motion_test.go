package motion

import (
	"os"
	"testing"
)

// TestPreferenceReduced proves criterion #452: Preference.Reduced reports
// true for the Reduced constant and false for the Normal constant.
func TestPreferenceReduced(t *testing.T) {
	for _, tc := range []struct {
		name string
		pref Preference
		want bool
	}{
		{"Reduced", Reduced, true},
		{"Normal", Normal, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.pref.Reduced(); got != tc.want {
				t.Errorf("Preference(%v).Reduced() = %v, want %v", tc.pref, got, tc.want)
			}
		})
	}
}

// TestDetectReduceMotion proves criterion #85: Detect returns Reduced when
// either NO_ANIMATION or REDUCE_MOTION is non-empty, and Normal when both
// are empty.
func TestDetectReduceMotion(t *testing.T) {
	for _, tc := range []struct {
		name, noAnim, reduce string
		want                 Preference
	}{
		{"neither", "", "", Normal},
		{"REDUCE_MOTION only", "", "1", Reduced},
		{"NO_ANIMATION only", "1", "", Reduced},
		{"both", "1", "1", Reduced},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(reducedMotionEnvVar, tc.noAnim)
			t.Setenv(standardReducedMotionEnvVar, tc.reduce)
			if got := Detect(); got != tc.want {
				t.Errorf("Detect() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestDetect proves criterion #451: Detect returns Reduced when the
// recognized reduced-motion environment variable is set to a non-empty
// value, and Normal otherwise.
func TestDetect(t *testing.T) {
	for _, tc := range []struct {
		name   string
		envVal string
		unset  bool
		want   Preference
	}{
		{"unset", "", true, Normal},
		{"empty", "", false, Normal},
		{"set to 1", "1", false, Reduced},
		{"set to arbitrary non-empty value", "true", false, Reduced},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.unset {
				// t.Setenv marks the test as using env vars so it can't
				// run in parallel with others that also set it, and
				// ensures the original value is restored afterward.
				t.Setenv(reducedMotionEnvVar, "placeholder")
				if err := os.Unsetenv(reducedMotionEnvVar); err != nil {
					t.Fatalf("failed to unset %s: %v", reducedMotionEnvVar, err)
				}
			} else {
				t.Setenv(reducedMotionEnvVar, tc.envVal)
			}

			if got := Detect(); got != tc.want {
				t.Errorf("Detect() = %v, want %v", got, tc.want)
			}
		})
	}
}
