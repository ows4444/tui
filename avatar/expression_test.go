package avatar_test

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/avatar"
)

var expressions = []avatar.Expression{
	avatar.ExpressionNone, avatar.ExpressionHappy, avatar.ExpressionSad, avatar.ExpressionMad,
	avatar.ExpressionSurprised, avatar.ExpressionWink, avatar.ExpressionSleepy, avatar.ExpressionThinking,
	avatar.ExpressionSmug, avatar.ExpressionUnsure, avatar.ExpressionScared, avatar.ExpressionLove,
	avatar.ExpressionShy, avatar.ExpressionSick,
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
	want := []string{"none", "happy", "sad", "mad", "surprised", "wink", "sleepy", "thinking",
		"smug", "unsure", "scared", "love", "shy", "sick"}
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

// SVG draws the pose, and an expression still blinks and looks.
func TestExpressionComposesWithLookAndBlink(t *testing.T) {
	m := avatar.New("alain00")
	m.Width, m.Height = 24, 12
	rest := m.SVG()
	m.Expression = avatar.ExpressionSurprised
	if m.SVG() == rest {
		t.Error("an expression did not change the SVG")
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

// The seven expressions that came first draw what they drew when the figures
// were frozen for v1.0.0, eyebrows and mouths included: these are digests of
// their Views at 16x8, taken then.
func TestTheFirstSevenExpressionsAreFrozen(t *testing.T) {
	byName := map[string]avatar.Expression{}
	for _, e := range expressions {
		byName[e.String()] = e
	}
	for _, pin := range [][3]string{
		{"alain00", "happy", "7f2949e4a4c7"},
		{"alain00", "sad", "54927602ad1a"},
		{"alain00", "mad", "6fb02132dc82"},
		{"alain00", "surprised", "35cfa6a518fe"},
		{"alain00", "wink", "7c9474949385"},
		{"alain00", "sleepy", "ee2c00c22328"},
		{"alain00", "thinking", "f9a1a4fdf0f3"},
		{"kasper", "happy", "d34febacce81"},
		{"kasper", "sad", "4ffab559c884"},
		{"kasper", "mad", "da7eee6a01b9"},
		{"kasper", "surprised", "72126475e43d"},
		{"kasper", "wink", "f5079d372916"},
		{"kasper", "sleepy", "74666d50a2a5"},
		{"kasper", "thinking", "f2dc179ee655"},
	} {
		m := avatar.New(pin[0])
		m.Width, m.Height, m.Expression = 16, 8, byName[pin[1]]
		sum := sha256.Sum256([]byte(m.View()))
		if got := fmt.Sprintf("%x", sum[:6]); got != pin[2] {
			t.Errorf("%s %s: View digest %s, was %s", pin[0], pin[1], got, pin[2])
		}
	}
}

// The three poses that tint besides mad each colour the body their own way.
func TestTheNewTintedExpressions(t *testing.T) {
	rest := avatar.New("ada")
	body, _, _ := rest.Colors()
	seen := map[ansi.RGB]avatar.Expression{body: avatar.ExpressionNone}
	for _, e := range []avatar.Expression{avatar.ExpressionMad, avatar.ExpressionLove, avatar.ExpressionShy, avatar.ExpressionSick} {
		m := avatar.New("ada")
		m.Expression = e
		b, _, _ := m.Colors()
		if prev, dup := seen[b]; dup {
			t.Errorf("%v and %v colour the body alike", prev, e)
		}
		seen[b] = e
	}
	for _, e := range []avatar.Expression{avatar.ExpressionSmug, avatar.ExpressionUnsure, avatar.ExpressionScared} {
		m := avatar.New("ada")
		m.Expression = e
		if b, _, _ := m.Colors(); b != body {
			t.Errorf("%v tints the body", e)
		}
	}
}
