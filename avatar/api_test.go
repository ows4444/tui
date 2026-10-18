package avatar_test

import (
	"testing"

	"github.com/ows4444/tui/avatar"
)

// A hue of 0 is a pin like any other; a nil Hue is the name's own.
func TestHueZeroIsAPinAndNilIsNot(t *testing.T) {
	own := avatar.New("hue-zero")
	pinned := own
	pinned.Hue = deg(0)
	a, _, _ := own.Colors()
	b, _, _ := pinned.Colors()
	if a == b {
		t.Fatal("a hue pinned at 0 degrees left the name's own colour")
	}
	full := own
	full.Hue = deg(360)
	if c, _, _ := full.Colors(); c != b {
		t.Errorf("0 and 360 degrees are one hue, got %v and %v", b, c)
	}
}

// Shape names a silhouette and is never "auto", pinned or not.
func TestShapeIsASilhouetteNeverAuto(t *testing.T) {
	for _, name := range []string{"ada", "grace", "", "x"} {
		m := avatar.New(name)
		if s := m.Shape(); s < avatar.SilhouetteRound || s > avatar.SilhouetteTriangle {
			t.Errorf("%q: Shape() = %d", name, s)
		}
		m.Silhouette = avatar.SilhouetteSun
		if m.Shape() != avatar.SilhouetteSun {
			t.Errorf("%q pinned to sun draws %v", name, m.Shape())
		}
	}
}

// The mutators act on the Model they are called on: a copy that is thrown away
// keeps the change to itself.
func TestMutatorsChangeTheModelTheyAreCalledOn(t *testing.T) {
	kept := avatar.New("kept")
	rest := kept.View()
	copyOf := kept
	copyOf.SetExpression(avatar.ExpressionSurprised)
	copyOf.LookAt(5, 0)
	if kept.View() != rest {
		t.Error("changing a copy changed the kept model")
	}
	kept.SetExpression(avatar.ExpressionSurprised)
	if kept.Expression == avatar.ExpressionNone && kept.View() == rest {
		t.Error("SetExpression on the kept model changed nothing")
	}
}
