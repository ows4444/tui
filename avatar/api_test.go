package avatar_test

import (
	"github.com/ows4444/tui"
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

// A Decorative avatar is silent, so a label beside it is not read twice.
func TestDecorativeAvatarLinearizesToNothing(t *testing.T) {
	m := avatar.New("ada")
	if got := m.Linearize(); got != "Avatar: ada" {
		t.Fatalf("Linearize = %q", got)
	}
	m.Decorative = true
	if got := m.Linearize(); got != "" {
		t.Errorf("a decorative avatar says %q", got)
	}
	if m.View() == "" {
		t.Error("Decorative hid the picture")
	}
}

// Motion over the avatar's Bounds starts the idle loop and turns the eyes
// toward the pointer; leaving stops both.
func TestMouseOverBoundsHoversAndLooks(t *testing.T) {
	m := avatar.New("mouse")
	m.X, m.Y = 10, 5
	if b := m.Bounds(); b.X != 10 || b.Y != 5 || b.W != m.Width || b.H != m.Height {
		t.Fatalf("Bounds = %+v", b)
	}
	move := func(x, y int) tui.MouseEvent { return tui.MouseEvent{X: x, Y: y, Action: tui.MouseActionMotion} }

	m, cmd := m.Update(move(10+m.Width-1, 5+m.Height/2))
	if !m.Idling() || cmd == nil {
		t.Error("the pointer arriving did not start the idle loop")
	}
	if m.LookX <= 0 {
		t.Errorf("the pointer is right of centre but LookX = %v", m.LookX)
	}
	m, cmd = m.Update(move(10+m.Width-2, 5+m.Height/2))
	if cmd != nil || !m.Idling() {
		t.Error("staying over the avatar changed the idle loop")
	}
	m, _ = m.Update(move(0, 0))
	if m.Idling() || m.LookX != 0 || m.LookY != 0 {
		t.Error("leaving did not stop the loop and centre the eyes")
	}
	before := m
	m, cmd = m.Update(tui.MouseEvent{X: 11, Y: 6, Action: tui.MouseActionRelease})
	if cmd != nil || m.Idling() != before.Idling() {
		t.Error("a release changed the avatar")
	}
}

// A left click on the avatar makes it pull a face; a click elsewhere does not.
func TestLeftClickOverBoundsReacts(t *testing.T) {
	m := avatar.New("click")
	m.X, m.Y = 3, 3
	press := func(x, y int, b tui.MouseButton) tui.MouseEvent {
		return tui.MouseEvent{X: x, Y: y, Button: b, Action: tui.MouseActionPress}
	}
	if out, _ := m.Update(press(0, 0, tui.MouseButtonLeft)); out.Reacting() {
		t.Error("a click outside reacted")
	}
	if out, _ := m.Update(press(4, 4, tui.MouseButtonRight)); out.Reacting() {
		t.Error("a right click reacted")
	}
	out, cmd := m.Update(press(4, 4, tui.MouseButtonLeft))
	if !out.Reacting() || cmd == nil {
		t.Error("a left click on the avatar did not react")
	}
}
