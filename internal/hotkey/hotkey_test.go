package hotkey

import (
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/input"
)

func TestParse(t *testing.T) {
	cases := []struct {
		in   string
		want Label
	}{
		{"Save", Label{Text: "Save", Pos: -1}},
		{"&Save", Label{Text: "Save", Key: 's', Pos: 0}},
		{"Sa&ve", Label{Text: "Save", Key: 'v', Pos: 2}},
		{"&Éclair", Label{Text: "Éclair", Key: 'é', Pos: 0}},
		{"日本&語", Label{Text: "日本語", Key: '語', Pos: 6}},
		{"R&&D", Label{Text: "R&D", Pos: -1}},
		{"R&&D &Team", Label{Text: "R&D Team", Key: 't', Pos: 4}},
		{"&1st", Label{Text: "1st", Key: '1', Pos: 0}},
		{"a & b", Label{Text: "a & b", Pos: -1}},
		{"trailing&", Label{Text: "trailing&", Pos: -1}},
		{"&", Label{Text: "&", Pos: -1}},
		{"&&", Label{Text: "&", Pos: -1}},
		{"&A and &B", Label{Text: "A and &B", Key: 'a', Pos: 0}},
		{"& x", Label{Text: "& x", Pos: -1}},
		{"", Label{Pos: -1}},
	}
	for _, c := range cases {
		if got := Parse(c.in); got != c.want {
			t.Errorf("Parse(%q) = %+v, want %+v", c.in, got, c.want)
		}
	}
}

func key(text string, mod input.Mod) input.Key {
	return input.Key{Type: input.KeyRunes, Text: text, Mod: mod}
}

func TestHit(t *testing.T) {
	l := Parse("&Save")
	for name, tc := range map[string]struct {
		msg  any
		want bool
	}{
		"alt s":       {key("s", input.ModAlt), true},
		"alt S":       {key("S", input.ModAlt), true},
		"alt shift S": {key("S", input.ModAlt|input.ModShift), true},
		"s":           {key("s", 0), false},
		"ctrl s":      {key("s", input.ModCtrl), false},
		"alt ctrl s":  {key("s", input.ModAlt|input.ModCtrl), false},
		"alt x":       {key("x", input.ModAlt), false},
		"two runes":   {key("ss", input.ModAlt), false},
		"empty":       {key("", input.ModAlt), false},
		"release":     {input.Key{Type: input.KeyRunes, Text: "s", Mod: input.ModAlt, Action: input.KeyRelease}, false},
		"enter":       {input.Key{Type: input.KeyEnter, Mod: input.ModAlt}, false},
		"not a key":   {"s", false},
	} {
		if got := l.Hit(tc.msg); got != tc.want {
			t.Errorf("%s: Hit = %v, want %v", name, got, tc.want)
		}
	}
	if Parse("Save").Hit(key("s", input.ModAlt)) {
		t.Error("a label with no hot key was hit")
	}
	if !Parse("&é").Hit(key("É", input.ModAlt)) {
		t.Error("a non-ASCII hot key was not matched case-insensitively")
	}
}

func TestRenderUnderlinesOneRune(t *testing.T) {
	base := ansi.NewStyle().Foreground(ansi.Red)
	got := Render("Save", 0, base)
	if ansi.StripANSI(got) != "Save" {
		t.Fatalf("text changed: %q", ansi.StripANSI(got))
	}
	if want := base.Underline().Render("S"); !contains(got, want) {
		t.Errorf("S is not underlined in %q", got)
	}
	if got := Render("日本語", 3, base); ansi.StripANSI(got) != "日本語" || !contains(got, base.Underline().Render("本")) {
		t.Errorf("wide rune: %q", got)
	}
	for _, pos := range []int{-1, 4, 99} {
		if got := Render("Save", pos, base); got != base.Render("Save") {
			t.Errorf("pos %d: %q", pos, got)
		}
	}
}

func contains(s, sub string) bool { return len(sub) > 0 && indexOf(s, sub) >= 0 }

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestSpoken(t *testing.T) {
	if got := Parse("&save").Spoken(); got != "Alt+S" {
		t.Errorf("Spoken = %q", got)
	}
	if Parse("save").Spoken() != "" {
		t.Error("a label with no hot key is spoken")
	}
}
