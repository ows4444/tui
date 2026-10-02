package widgets

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

var updateBigTextGolden = flag.Bool("update", false, "rewrite the widgets golden files")

var bigTextFontNames = map[Font]string{
	FontBlock: "block", FontSimple: "simple", FontShade: "shade", FontSlim: "slim",
}

// When BigText renders text with FontSimple, FontShade and FontSlim, each
// output differs from FontBlock and from each other.
func TestBigTextFontsAreDistinct(t *testing.T) {
	for _, text := range []string{"A", "Hi 2"} {
		seen := map[string]Font{}
		for _, f := range []Font{FontBlock, FontSimple, FontShade, FontSlim} {
			got := BigText(text, f, theme.DarkTheme())
			if prev, dup := seen[got]; dup {
				t.Errorf("BigText(%q): font %s renders identically to %s", text, bigTextFontNames[f], bigTextFontNames[prev])
			}
			seen[got] = f
		}
	}
}

// Each font has a golden for 'A'.
func TestBigTextGoldens(t *testing.T) {
	for _, f := range []Font{FontSimple, FontShade, FontSlim} {
		name := filepath.Join("testdata", "bigtext_"+bigTextFontNames[f]+"_A.golden")
		got := ansi.StripANSI(BigText("A", f, theme.DarkTheme())) + "\n"
		if *updateBigTextGolden {
			if err := os.MkdirAll("testdata", 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(name, []byte(got), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		want, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("%v (go test ./widgets -update to create)", err)
		}
		if string(want) != got {
			t.Errorf("%s differs (go test ./widgets -update to regenerate):\ngot:\n%s\nwant:\n%s", name, got, want)
		}
	}
}

// Every glyph of a font has the font's height and one display width per row.
func TestBigTextGlyphGeometry(t *testing.T) {
	wantHeight := map[Font]int{FontBlock: 5, FontSimple: 5, FontShade: 5, FontSlim: 3}
	for f, h := range wantHeight {
		tab := bigTextFonts[f]
		if len(tab) != len(bigTextGlyphs) {
			t.Errorf("%s has %d glyphs, want %d", bigTextFontNames[f], len(tab), len(bigTextGlyphs))
		}
		for r, g := range tab {
			if len(g) != h {
				t.Errorf("%s %q has %d rows, want %d", bigTextFontNames[f], r, len(g), h)
				continue
			}
			for i, row := range g {
				if ansi.Width(row) != ansi.Width(g[0]) {
					t.Errorf("%s %q row %d width %d, want %d", bigTextFontNames[f], r, i, ansi.Width(row), ansi.Width(g[0]))
				}
			}
		}
		rows := strings.Split(BigText("Hi 2 A!", f, theme.DarkTheme()), "\n")
		if len(rows) != h {
			t.Errorf("%s rendered %d rows, want %d", bigTextFontNames[f], len(rows), h)
		}
		for i, row := range rows {
			if ansi.Width(row) != ansi.Width(rows[0]) {
				t.Errorf("%s row %d width %d, want %d", bigTextFontNames[f], i, ansi.Width(row), ansi.Width(rows[0]))
			}
		}
	}
}

// Under an ASCII theme no font emits a byte >= 0x80, and FontSimple is ASCII
// under every theme.
func TestBigTextASCIIDegrades(t *testing.T) {
	text := "ABCDEFGHIJKLMNOPQRSTUVWXYZ 0123456789"
	for f := range bigTextFontNames {
		got := ansi.StripANSI(BigText(text, f, theme.DarkTheme().ASCII()))
		for i := 0; i < len(got); i++ {
			if got[i] >= 0x80 {
				t.Fatalf("%s: non-ASCII byte in ASCII theme output:\n%s", bigTextFontNames[f], got)
			}
		}
		g := ansi.StripANSI(BigTextGradient(text, f, theme.DarkTheme().ASCII(), ansi.RGB{}, ansi.RGB{R: 255}))
		if g != got {
			t.Errorf("%s: gradient ASCII output differs", bigTextFontNames[f])
		}
	}
	got := BigText(text, FontSimple, theme.DarkTheme())
	for i := 0; i < len(got); i++ {
		if got[i] >= 0x80 {
			t.Fatal("FontSimple emitted non-ASCII under the Unicode theme")
		}
	}
}

// The gradient changes only styling, and the first and last columns take the
// endpoint colours.
func TestBigTextGradient(t *testing.T) {
	from, to := ansi.RGB{R: 255}, ansi.RGB{B: 255}
	for _, f := range []Font{FontBlock, FontSimple, FontShade, FontSlim} {
		plain := BigText("Hi 2", f, theme.DarkTheme())
		grad := BigTextGradient("Hi 2", f, theme.DarkTheme(), from, to)
		if ansi.StripANSI(grad) != ansi.StripANSI(plain) {
			t.Errorf("%s: gradient changed the text", bigTextFontNames[f])
		}
		rows := strings.Split(grad, "\n")
		plainRows := strings.Split(ansi.StripANSI(plain), "\n")
		for y, row := range rows {
			cells := []rune(plainRows[y])
			first := ansi.NewStyle().Foreground(from).Render(string(cells[0]))
			last := ansi.NewStyle().Foreground(to).Render(string(cells[len(cells)-1]))
			if !strings.HasPrefix(row, first) || !strings.HasSuffix(row, last) {
				t.Errorf("%s row %d: endpoints not %q ... %q in %q", bigTextFontNames[f], y, first, last, row)
			}
			if ansi.Width(row) != ansi.Width(plainRows[y]) {
				t.Errorf("%s row %d: width changed", bigTextFontNames[f], y)
			}
		}
	}
	if BigTextGradient("", FontBlock, theme.DarkTheme(), from, to) != "" {
		t.Error("empty text should render empty")
	}
	if got := BigTextGradient(" ", FontSlim, theme.DarkTheme(), from, to); ansi.StripANSI(got) != "   \n   \n   " {
		t.Errorf("single glyph gradient = %q", got)
	}
}
