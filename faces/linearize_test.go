package faces

import (
	"testing"

	"github.com/ows4444/tui"
)

var _ tui.Linearizer = Model{}

func TestLinearizeNamesTheFaceAndItsAnimation(t *testing.T) {
	m := New()
	f := m.Face()
	want := "Face: " + f.Name + ", " + f.Anim
	if got := m.Linearize(); got != want {
		t.Errorf("Linearize = %q, want %q", got, want)
	}
	m2 := m
	m2.frame = 3 // animation frames must not change the text
	if m2.Linearize() != m.Linearize() {
		t.Error("Linearize should not depend on the animation frame")
	}
}
