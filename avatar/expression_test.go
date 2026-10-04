package avatar_test

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/avatar"
)

var expressions = []avatar.Expression{
	avatar.ExpressionNone, avatar.ExpressionHappy, avatar.ExpressionSad, avatar.ExpressionMad,
	avatar.ExpressionSurprised, avatar.ExpressionWink, avatar.ExpressionSleepy, avatar.ExpressionThinking,
}

// With no expression set the avatar is the figure at rest, and a value that
// is not an expression is treated as none.
func TestNoExpressionIsTheRestingFigure(t *testing.T) {
	for _, name := range names {
		rest := avatar.New(name)
		m := avatar.New(name)
		m.Expression = avatar.ExpressionNone
		if m.View() != rest.View() {
			t.Errorf("%q: ExpressionNone changed the View", name)
		}
		for _, bad := range []avatar.Expression{-1, 99} {
			m.Expression = bad
			if m.View() != rest.View() || m.Linearize() != rest.Linearize() || bad.String() != "none" {
				t.Errorf("%q: expression %d is not treated as none", name, bad)
			}
		}
	}
}

// Every expression draws its own picture: at 12x6 cells and larger, no two
// expressions of one name share a View, with or without colour.
func TestExpressionsAreDistinct(t *testing.T) {
	for _, size := range [][2]int{{12, 6}, {16, 8}, {24, 12}} {
		for _, name := range names {
			coloured, plain := map[string]avatar.Expression{}, map[string]avatar.Expression{}
			for _, e := range expressions {
				m := avatar.New(name)
				m.Width, m.Height, m.Expression = size[0], size[1], e
				v := m.View()
				if prev, dup := coloured[v]; dup {
					t.Errorf("%q at %dx%d: %v and %v draw the same View", name, size[0], size[1], prev, e)
				}
				coloured[v] = e
				p := ansi.StripANSI(v)
				if prev, dup := plain[p]; dup {
					t.Errorf("%q at %dx%d: %v and %v are the same without colour:\n%s", name, size[0], size[1], prev, e, p)
				}
				plain[p] = e
			}
		}
	}
}

func TestExpressionNames(t *testing.T) {
	want := []string{"none", "happy", "sad", "mad", "surprised", "wink", "sleepy", "thinking"}
	for i, e := range expressions {
		if e.String() != want[i] {
			t.Errorf("expression %d is %q, want %q", i, e, want[i])
		}
	}
}

// The expression is spoken after the name; the picture is not described.
func TestLinearizeNamesTheExpression(t *testing.T) {
	m := avatar.New("ada")
	m.Expression = avatar.ExpressionThinking
	if got := m.Linearize(); got != "Avatar: ada, thinking" {
		t.Errorf("Linearize = %q", got)
	}
	m.Name = ""
	if got := m.Linearize(); got != "Avatar, thinking" {
		t.Errorf("Linearize with no name = %q", got)
	}
}

// SVG stays the resting figure, and an expression still blinks and looks.
func TestExpressionComposesWithLookAndBlink(t *testing.T) {
	m := avatar.New("alain00")
	m.Width, m.Height = 24, 12
	rest := m.SVG()
	m.Expression = avatar.ExpressionSurprised
	if m.SVG() != rest {
		t.Error("an expression changed the SVG")
	}
	posed := m.View()
	m.LookAt(100, 0)
	if m.View() == posed {
		t.Error("a look did not move a posed avatar's eyes")
	}
	m.LookAt(0, 0)
	if cmd := m.Blink(); cmd == nil || m.View() == posed {
		t.Error("a blink did not change a posed avatar")
	}
	for _, e := range expressions {
		m := avatar.New("kasper")
		m.Expression = e
		for _, row := range strings.Split(m.View(), "\n") {
			if w := ansi.Width(row); w != avatar.DefaultWidth {
				t.Errorf("%v: a row is %d cells wide", e, w)
			}
		}
	}
}
