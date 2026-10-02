package faces

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

// referenceView is View as it was before an ASCII form existed, kept so the
// default glyphs can be checked against it.
func referenceView(m Model) string {
	f, frames := m.Face(), m.frames()
	body := ansi.NewStyle().Foreground(m.Theme.Primary).Render(frames[m.frame%len(frames)])
	if !m.ShowLabel {
		return body
	}
	label := fmt.Sprintf("%s · %s · %d/%d", f.Name, f.Anim, m.frame%len(frames)+1, len(frames))
	w, _ := m.Size.Cells()
	pad := (w - ansi.Width(label)) / 2
	if pad < 0 {
		pad = 0
	}
	return body + "\n" + ansi.NewStyle().Faint().Render(strings.Repeat(" ", pad)+label)
}

// eachFrame calls fn for every face, at both sizes, for every frame of its
// loop, with and without the label.
func eachFrame(fn func(name string, m Model)) {
	for i := 0; i < Count(); i++ {
		for _, sz := range []Size{Small, Large} {
			for _, label := range []bool{false, true} {
				m := New()
				m.Set(i)
				m.Size, m.ShowLabel = sz, label
				for f := 0; f < len(m.frames()); f++ {
					m.frame = f
					fn(fmt.Sprintf("%s size=%v label=%v frame=%d", m.Face().Name, sz, label, f), m)
				}
			}
		}
	}
}

// Under the default glyphs every catalog entry draws exactly as before.
func TestDefaultGlyphsDrawExactlyAsBefore(t *testing.T) {
	n := 0
	eachFrame(func(name string, m Model) {
		if got, want := m.View(), referenceView(m); got != want {
			t.Fatalf("%s: View differs from the reference:\n got %q\nwant %q", name, got, want)
		}
		n++
	})
	if n < 500 {
		t.Errorf("only %d frames were checked", n)
	}
}

// Under ASCII glyphs no frame draws a non-ASCII character, in View or in the
// layout node.
func TestASCIIGlyphsDrawOnlySevenBit(t *testing.T) {
	eachFrame(func(name string, m Model) {
		m.Theme = theme.DarkTheme().ASCII()
		for what, out := range map[string]string{
			"View":       m.View(),
			"LayoutNode": m.LayoutNode().Render(layout.Size{W: 12, H: 6}),
		} {
			for _, r := range ansi.StripANSI(out) {
				if r >= 0x80 {
					t.Fatalf("%s: %s draws %q (U+%04X)", name, what, r, r)
				}
			}
		}
	})
}

// The ASCII frame has the same rows and the same width per row as the braille
// one, so centring and layout do not change.
func TestASCIIFramesKeepTheirShape(t *testing.T) {
	eachFrame(func(name string, m Model) {
		uni := strings.Split(ansi.StripANSI(m.View()), "\n")
		m.Theme = theme.DarkTheme().ASCII()
		asc := strings.Split(ansi.StripANSI(m.View()), "\n")
		if len(uni) != len(asc) {
			t.Fatalf("%s: %d rows in ASCII, %d in braille", name, len(asc), len(uni))
		}
		for i := range uni {
			// The label row differs in its separator width only through the
			// dot glyph, which takes one column in both sets.
			if ansi.Width(uni[i]) != ansi.Width(asc[i]) {
				t.Fatalf("%s: row %d is %d wide in ASCII, %d in braille", name, i, ansi.Width(asc[i]), ansi.Width(uni[i]))
			}
		}
	})
}

// More dots in a cell never give a lighter shade.
func TestASCIIShadeNeverGetsLighterWithMoreDots(t *testing.T) {
	shades := theme.ASCIIGlyphSet().Shades
	rank := func(pattern rune) int {
		out := asciiFrame(string(0x2800+pattern), shades)
		if out == " " {
			return -1
		}
		return strings.Index(shades, out)
	}
	dots := func(p rune) int {
		n := 0
		for ; p != 0; p >>= 1 {
			n += int(p & 1)
		}
		return n
	}
	best := map[int]int{} // dot count -> shade rank
	for p := rune(0); p < 256; p++ {
		n, r := dots(p), rank(p)
		if prev, ok := best[n]; ok && prev != r {
			t.Fatalf("patterns with %d dots get different shades (%d and %d)", n, prev, r)
		}
		best[n] = r
	}
	for n := 1; n <= 8; n++ {
		if best[n] < best[n-1] {
			t.Errorf("%d dots draw a lighter shade (rank %d) than %d dots (rank %d)", n, best[n], n-1, best[n-1])
		}
	}
	if best[0] != -1 {
		t.Errorf("an empty cell is %d, want a space", best[0])
	}
	if best[8] != len([]rune(shades))-1 {
		t.Errorf("a full cell should take the darkest shade, got rank %d", best[8])
	}
}

// The ASCII frame still resembles its face: the head is solid where the braille
// one is, so the two have blanks in the same places.
func TestASCIIBlanksMatchTheBrailleBlanks(t *testing.T) {
	eachFrame(func(name string, m Model) {
		uni := ansi.StripANSI(m.View())
		m.Theme = theme.DarkTheme().ASCII()
		asc := ansi.StripANSI(m.View())
		ur, ar := []rune(uni), []rune(asc)
		if len(ur) != len(ar) {
			t.Fatalf("%s: lengths %d and %d", name, len(ar), len(ur))
		}
		for i := range ur {
			if (ur[i] == ' ') != (ar[i] == ' ') && ur[i] >= 0x2800 && ur[i] <= 0x28FF && ur[i] != 0x2800 {
				t.Fatalf("%s: position %d is %q in braille but %q in ASCII", name, i, ur[i], ar[i])
			}
		}
	})
}
