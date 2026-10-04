package avatar_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/avatar"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

var names = []string{"alain00", "tove", "kasper", "mdawais", "user-1", "user-12", "user-21", "Team Rocket 3", "7", ""}

// View is a block of exactly Height rows, each exactly Width cells wide, at
// every size and with every background.
func TestViewIsWidthByHeightCells(t *testing.T) {
	for _, bg := range []avatar.Background{avatar.BackgroundNone, avatar.BackgroundSquircle, avatar.BackgroundCircle, avatar.BackgroundSquare} {
		for _, size := range [][2]int{{1, 1}, {2, 1}, {4, 2}, {8, 4}, {12, 6}, {7, 9}, {30, 3}} {
			for _, name := range names {
				m := avatar.New(name)
				m.Width, m.Height, m.Background = size[0], size[1], bg
				rows := strings.Split(m.View(), "\n")
				if len(rows) != size[1] {
					t.Fatalf("%q %dx%d bg %d: %d rows", name, size[0], size[1], bg, len(rows))
				}
				for i, r := range rows {
					if w := ansi.Width(r); w != size[0] {
						t.Errorf("%q %dx%d bg %d: row %d is %d cells wide", name, size[0], size[1], bg, i, w)
					}
				}
			}
		}
	}
}

func TestViewIsEmptyWithoutASize(t *testing.T) {
	for _, size := range [][2]int{{0, 4}, {8, 0}, {-1, 4}, {8, -2}} {
		m := avatar.New("alain")
		m.Width, m.Height = size[0], size[1]
		if v := m.View(); v != "" {
			t.Errorf("%dx%d: View = %q", size[0], size[1], v)
		}
		if l := m.Linearize(); l != "" {
			t.Errorf("%dx%d: Linearize = %q", size[0], size[1], l)
		}
	}
}

// The same name draws the same avatar, case and surrounding space aside;
// another name draws another.
func TestViewDependsOnlyOnTheNormalizedName(t *testing.T) {
	a := avatar.New("alain@example.com").View()
	if b := avatar.New("alain@example.com").View(); a != b {
		t.Error("the same name rendered twice differs")
	}
	if b := avatar.New("  Alain@Example.COM ").View(); a != b {
		t.Error("case and surrounding space changed the avatar")
	}
	if b := avatar.New("alaim@example.com").View(); a == b {
		t.Error("a different name drew the same avatar")
	}
	raw := avatar.New("Alain@Example.COM")
	raw.Raw = true
	if raw.View() == a {
		t.Error("Raw did not keep the name's case")
	}
}

// With colour stripped the body is ink and both eyes are holes in it: the
// plain text of the default avatar, the eyes at the two notches.
func TestViewWithoutColourIsTheBodyWithEyeHoles(t *testing.T) {
	m := avatar.New("alain00")
	m.Width, m.Height = 12, 6
	got := ansi.StripANSI(m.View())
	want := strings.Join([]string{
		"  \u2597\u2584\u2584\u2584\u2584\u2584\u2584\u2596  ",
		"\u2597\u259f\u2588\u259b\u259c\u2588\u2588\u2588\u2588\u2588\u2599\u2596",
		"\u2590\u2588\u2588\u258c \u2588\u2588\u258c \u259c\u2588\u258c",
		"\u2590\u2588\u2588\u2588\u2584\u259f\u2588\u2588\u2584\u259f\u2588\u258c",
		"\u259d\u259c\u2588\u2588\u2588\u2588\u2588\u2588\u2588\u2588\u259b\u2598",
		"  \u259d\u2580\u2580\u2580\u2580\u2580\u2580\u2598  ",
	}, "\n")
	if got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

// The colours in the View are the three Colors reports, and nothing else.
func TestViewUsesOnlyTheAvatarsColours(t *testing.T) {
	m := avatar.New("alain00")
	m.Background = avatar.BackgroundCircle
	body, eyes, bg := m.Colors()
	seq := func(c ansi.RGB) string { return ansi.NewStyle().Foreground(c).Render("x") }
	view := m.View()
	for name, c := range map[string]ansi.RGB{"body": body, "eyes": eyes, "background": bg} {
		code := strings.TrimSuffix(strings.TrimPrefix(strings.SplitN(seq(c), "m", 2)[0], "\x1b[38;2;"), "m")
		if !strings.Contains(view, code) {
			t.Errorf("the %s colour %v is not in the View", name, c)
		}
	}
	if body == eyes || body == bg {
		t.Errorf("colours are not distinct: %v %v %v", body, eyes, bg)
	}
	if got := avatar.New("alain00"); strings.Contains(got.View(), "48;2;237;245;249") {
		t.Error("the background colour is drawn with BackgroundNone")
	}
}

// Every eye is visible at every size: the View always has a cell in the eye
// colour, even where an eye is narrower than a pixel.
func TestEyesAreAlwaysDrawn(t *testing.T) {
	for _, name := range names {
		for _, size := range [][2]int{{2, 1}, {4, 2}, {6, 3}, {8, 4}, {16, 8}} {
			m := avatar.New(name)
			m.Width, m.Height = size[0], size[1]
			_, eyes, _ := m.Colors()
			code := strings.TrimPrefix(strings.SplitN(ansi.NewStyle().Foreground(eyes).Render("x"), "m", 2)[0], "\x1b[38;")
			if !strings.Contains(m.View(), code) {
				t.Errorf("%q at %dx%d has no eye-coloured cell", name, size[0], size[1])
			}
		}
	}
}

// Under an ASCII glyph set the avatar is drawn with ASCII only and keeps its
// size.
func TestViewUnderASCIIGlyphsIsASCII(t *testing.T) {
	for _, name := range names {
		m := avatar.New(name).SetTheme(theme.DarkTheme().ASCII())
		m.Background = avatar.BackgroundSquircle
		plain := ansi.StripANSI(m.View())
		for _, r := range plain {
			if r >= utf8.RuneSelf {
				t.Fatalf("%q: non-ASCII rune %q in\n%s", name, r, plain)
			}
		}
		if rows := strings.Split(plain, "\n"); len(rows) != avatar.DefaultHeight || len(rows[0]) != avatar.DefaultWidth {
			t.Errorf("%q: ASCII view is not %dx%d:\n%s", name, avatar.DefaultWidth, avatar.DefaultHeight, plain)
		}
		if strings.TrimSpace(plain) == "" {
			t.Errorf("%q: ASCII view is blank", name)
		}
	}
}

func TestShapeIsOneOfTheTen(t *testing.T) {
	known := map[string]bool{"round": true, "organic": true, "boxy": true, "capsule": true, "nub": true,
		"cloud": true, "droplet": true, "hexagon": true, "sun": true, "triangle": true}
	seen := map[string]bool{}
	for i := 0; i < 400; i++ {
		s := avatar.New("user-" + strings.Repeat("x", i%7) + string(rune('a'+i%26)) + string(rune('0'+i%10))).Shape()
		if !known[s] {
			t.Fatalf("unknown shape %q", s)
		}
		seen[s] = true
	}
	if len(seen) != len(known) {
		t.Errorf("400 names drew only %d of the 10 shapes: %v", len(seen), seen)
	}
}

func TestSVGIsOneFigurePerBackground(t *testing.T) {
	m := avatar.New("alain")
	plain := m.SVG()
	if !strings.HasPrefix(plain, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><g fill="#`) || !strings.HasSuffix(plain, "</g></svg>") {
		t.Errorf("unexpected markup: %s", plain)
	}
	seen := map[string]bool{plain: true}
	for _, bg := range []avatar.Background{avatar.BackgroundSquircle, avatar.BackgroundCircle, avatar.BackgroundSquare} {
		m.Background = bg
		got := m.SVG()
		if seen[got] {
			t.Errorf("background %d repeats another's markup", bg)
		}
		seen[got] = true
		if !strings.HasSuffix(got, strings.TrimPrefix(plain, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100">`)) {
			t.Errorf("background %d changed the figure", bg)
		}
	}
}

func TestLinearizeNamesTheAvatar(t *testing.T) {
	for name, want := range map[string]string{
		"alain":             "Avatar: alain",
		"  Team Rocket  ":   "Avatar: Team Rocket",
		"":                  "Avatar",
		"   ":               "Avatar",
		"evil\x1b[31mname":  "Avatar: evilname",
		"line\nbreak\x07ok": "",
	} {
		got := avatar.New(name).Linearize()
		if name == "line\nbreak\x07ok" {
			if strings.ContainsAny(got, "\x07\x1b") || !strings.HasPrefix(got, "Avatar: line") {
				t.Errorf("Linearize(%q) = %q", name, got)
			}
			continue
		}
		if got != want {
			t.Errorf("Linearize(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestLayoutNodeDrawsAtTheAllottedSize(t *testing.T) {
	m := avatar.New("alain")
	n := m.LayoutNode()
	if got := n.Measure(layout.Unconstrained()); got != (layout.Size{W: avatar.DefaultWidth, H: avatar.DefaultHeight}) {
		t.Errorf("Measure = %+v", got)
	}
	if got := layout.Draw(n, layout.Unconstrained()); got != m.View() {
		t.Error("an unconstrained node does not draw the View")
	}
	big := m
	big.Width, big.Height = 12, 6
	if got := n.Render(layout.Size{W: 12, H: 6}); got != big.View() {
		t.Error("Render does not draw at the allotted size")
	}
	if got := n.Render(layout.Size{W: 0, H: 6}); got != "" {
		t.Errorf("Render at zero width = %q", got)
	}
	m.Width, m.Height = -3, -1
	if got := m.LayoutNode().Measure(layout.Unconstrained()); got != (layout.Size{}) {
		t.Errorf("Measure of a negative size = %+v", got)
	}
}

// rgb formats c as the parameters of an SGR colour, "r;g;b".
func rgb(c ansi.RGB) string {
	return strings.TrimSuffix(strings.SplitN(strings.SplitN(ansi.NewStyle().Foreground(c).Render("x"), "38;2;", 2)[1], "x", 2)[0], "m")
}

// containsCode reports whether view sets the SGR parameters code.
func containsCode(view, code string) bool { return strings.Contains(view, code) }

// A name written with combining marks is the avatar of its precomposed
// spelling; Raw keeps the two apart.
func TestCombiningMarksAreNormalized(t *testing.T) {
	for decomposed, composed := range map[string]string{
		"café":        "café",
		"Zoë":         "zoë",
		"한":          "한",
		"  Ångström": "ångström",
	} {
		a, b := avatar.New(decomposed), avatar.New(composed)
		if a.View() != b.View() || a.SVG() != b.SVG() || a.Shape() != b.Shape() {
			t.Errorf("%+q and %+q are different avatars", decomposed, composed)
		}
	}
	a, b := avatar.New("café"), avatar.New("café")
	a.Raw, b.Raw = true, true
	if a.SVG() == b.SVG() {
		t.Error("Raw normalized the name")
	}
}
