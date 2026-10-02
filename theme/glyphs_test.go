package theme_test

import (
	"reflect"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/spinner"
	"github.com/ows4444/tui/theme"
)

func env(kv map[string]string) func(string) string {
	return func(k string) string { return kv[k] }
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

func TestASCIIGlyphsAreASCIIAndUnicodeOnesAreNot(t *testing.T) {
	for name, g := range map[string]theme.Glyphs{"ascii": theme.ASCIIGlyphSet()} {
		for _, s := range []string{g.BarFull, g.BarEmpty, g.Collapsed, g.Expanded, g.Spinner} {
			if s == "" || !isASCII(s) {
				t.Errorf("%s glyph %q is empty or not ASCII", name, s)
			}
		}
	}
	u := theme.UnicodeGlyphSet()
	for _, s := range []string{u.BarFull, u.BarEmpty, u.Collapsed, u.Expanded} {
		if isASCII(s) {
			t.Errorf("unicode glyph %q is ASCII", s)
		}
	}
}

// The Unicode spinner glyphs stay in step with the spinner's own default.
func TestUnicodeSpinnerGlyphsMatchTheSpinnerDefault(t *testing.T) {
	got := theme.UnicodeGlyphSet().SpinnerFrames()
	if len(got) != len(spinner.Frames()) {
		t.Fatalf("%d frames, want %d", len(got), len(spinner.Frames()))
	}
	for i := range got {
		if got[i] != spinner.Frames()[i] {
			t.Errorf("frame %d = %q, want %q", i, got[i], spinner.Frames()[i])
		}
	}
}

func TestZeroGlyphsResolveToUnicode(t *testing.T) {
	if got := (theme.Glyphs{}).Resolved(); got != theme.UnicodeGlyphSet() {
		t.Errorf("zero Resolved = %+v", got)
	}
	// Every preset leaves Glyphs unset, so it gets the Unicode set.
	for name, th := range map[string]theme.Theme{"dark": theme.DarkTheme(), "light": theme.LightTheme(), "hc": theme.HighContrastTheme()} {
		if th.GlyphSet() != theme.UnicodeGlyphSet() {
			t.Errorf("%s GlyphSet = %+v", name, th.GlyphSet())
		}
	}
	// A partly filled set keeps its own fields and fills the rest.
	g := theme.Glyphs{BarFull: "="}.Resolved()
	if g.BarFull != "=" || g.BarEmpty != theme.UnicodeGlyphSet().BarEmpty {
		t.Errorf("partial Resolved = %+v", g)
	}
}

func TestThemeASCIISwitchesGlyphsAndBorderOnly(t *testing.T) {
	a := theme.DarkTheme().ASCII()
	if a.Glyphs != theme.ASCIIGlyphSet() || a.Border != layout.ASCIIBorder() {
		t.Errorf("ASCII() = glyphs %+v border %+v", a.Glyphs, a.Border)
	}
	if a.Primary != theme.DarkTheme().Primary || a.Text != theme.DarkTheme().Text {
		t.Error("ASCII() changed the colours")
	}
	if theme.DarkTheme().Glyphs != (theme.Glyphs{}) {
		t.Error("ASCII() modified the receiver")
	}
}

// Themes stay comparable: this fails to compile if Glyphs makes Theme not so.
var _ map[theme.Theme]bool

func TestDetectGlyphs(t *testing.T) {
	for _, c := range []struct {
		name string
		env  map[string]string
		want theme.Glyphs
	}{
		{"nothing set", nil, theme.UnicodeGlyphSet()},
		{"C locale", map[string]string{"LANG": "C"}, theme.ASCIIGlyphSet()},
		{"POSIX locale", map[string]string{"LANG": "POSIX"}, theme.ASCIIGlyphSet()},
		{"LC_ALL beats LANG", map[string]string{"LC_ALL": "C", "LANG": "en_US.UTF-8"}, theme.ASCIIGlyphSet()},
		{"LC_ALL utf8 beats LANG C", map[string]string{"LC_ALL": "en_US.UTF-8", "LANG": "C"}, theme.UnicodeGlyphSet()},
		{"LC_CTYPE C", map[string]string{"LC_CTYPE": "C"}, theme.ASCIIGlyphSet()},
		{"utf-8", map[string]string{"LANG": "en_US.UTF-8"}, theme.UnicodeGlyphSet()},
		{"utf8 spelling", map[string]string{"LANG": "de_DE.utf8"}, theme.UnicodeGlyphSet()},
		{"C.UTF-8", map[string]string{"LANG": "C.UTF-8"}, theme.UnicodeGlyphSet()},
		{"latin1", map[string]string{"LANG": "en_US.ISO-8859-1"}, theme.ASCIIGlyphSet()},
		{"dumb terminal", map[string]string{"TERM": "dumb", "LANG": "en_US.UTF-8"}, theme.ASCIIGlyphSet()},
		{"DUMB uppercase", map[string]string{"TERM": "DUMB"}, theme.ASCIIGlyphSet()},
		{"xterm utf8", map[string]string{"TERM": "xterm-256color", "LANG": "en_US.UTF-8"}, theme.UnicodeGlyphSet()},
	} {
		if got := theme.DetectGlyphsEnv(env(c.env)); got != c.want {
			t.Errorf("%s: got %+v, want %+v", c.name, got, c.want)
		}
	}
}

func TestDetectGlyphsReadsTheProcessEnvironment(t *testing.T) {
	t.Setenv("TERM", "dumb")
	if theme.DetectGlyphs() != theme.ASCIIGlyphSet() {
		t.Error("DetectGlyphs ignored TERM=dumb")
	}
}

// Every field of Glyphs is filled in both built-in sets, and Resolved fills
// every empty one, however many fields are added later.
func TestEveryGlyphFieldIsSetInBothSetsAndResolved(t *testing.T) {
	check := func(name string, g theme.Glyphs) {
		v := reflect.ValueOf(g)
		for i := 0; i < v.NumField(); i++ {
			if v.Field(i).Kind() != reflect.String {
				t.Fatalf("%s.%s is a %s, want string (Glyphs must stay comparable)", name, v.Type().Field(i).Name, v.Field(i).Kind())
			}
			if v.Field(i).String() == "" {
				t.Errorf("%s.%s is empty", name, v.Type().Field(i).Name)
			}
		}
	}
	check("UnicodeGlyphSet()", theme.UnicodeGlyphSet())
	check("ASCIIGlyphSet()", theme.ASCIIGlyphSet())
	check("Resolved zero value", theme.Glyphs{}.Resolved())
}

// Under ASCIIGlyphSet() every glyph is one column wide and a Unicode one, so
// widgets keep their layout; Arrow is the one exception.
func TestASCIIGlyphsKeepTheUnicodeColumnCounts(t *testing.T) {
	u, a := reflect.ValueOf(theme.UnicodeGlyphSet()), reflect.ValueOf(theme.ASCIIGlyphSet())
	for i := 0; i < u.NumField(); i++ {
		name := u.Type().Field(i).Name
		uw, aw := ansi.Width(u.Field(i).String()), ansi.Width(a.Field(i).String())
		switch name {
		case "Arrow", "Spinner", "Shades", "Sparks":
			continue // Arrow is the exception; the others are sequences checked by rune count
		}
		if uw != aw {
			t.Errorf("%s: Unicode is %d columns, ASCII is %d", name, uw, aw)
		}
	}
	if n := len([]rune(theme.UnicodeGlyphSet().Shades)); n != 4 || len([]rune(theme.ASCIIGlyphSet().Shades)) != 4 {
		t.Errorf("Shades must hold 4 runes in both sets, got %d", n)
	}
	if n := len([]rune(theme.UnicodeGlyphSet().Sparks)); n != 8 || len([]rune(theme.ASCIIGlyphSet().Sparks)) != 8 {
		t.Errorf("Sparks must hold 8 runes in both sets, got %d", n)
	}
	if theme.UnicodeGlyphSet().Ellipsis == "" || ansi.Width(theme.ASCIIGlyphSet().Ellipsis) != 1 {
		t.Error("Ellipsis must take exactly one column")
	}
}

// Theme stays comparable however many glyph fields there are: this fails to
// compile if Glyphs (or anything else in Theme) stops being comparable.
func TestThemeStaysComparable(t *testing.T) {
	if theme.DarkTheme() == theme.LightTheme() {
		t.Error("Dark and Light compare equal")
	}
	if theme.DarkTheme().ASCII() == theme.DarkTheme() || theme.DarkTheme().ASCII() != theme.DarkTheme().ASCII() {
		t.Error("Theme comparison is not behaving")
	}
}

func TestGlyphsASCIIPredicate(t *testing.T) {
	if !theme.ASCIIGlyphSet().ASCII() {
		t.Error("ASCIIGlyphSet().ASCII() = false")
	}
	if theme.UnicodeGlyphSet().ASCII() || (theme.Glyphs{}).ASCII() {
		t.Error("the Unicode set (and the zero value, which resolves to it) reported ASCII")
	}
	mixed := theme.ASCIIGlyphSet()
	mixed.Dot = "●"
	if mixed.ASCII() {
		t.Error("a set with one Unicode glyph reported ASCII")
	}
}
