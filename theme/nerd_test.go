package theme_test

import (
	"reflect"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

func TestNerdGlyphsFieldsNonEmptyAndOneColumn(t *testing.T) {
	n := reflect.ValueOf(theme.NerdGlyphSet())
	u := reflect.ValueOf(theme.UnicodeGlyphSet())
	ty := n.Type()
	changed := 0
	for i := 0; i < n.NumField(); i++ {
		name := ty.Field(i).Name
		ns, us := n.Field(i).String(), u.Field(i).String()
		if ns == "" {
			t.Errorf("%s is empty", name)
			continue
		}
		if ns != us {
			changed++
		}
		if nw, uw := ansi.Width(ns), ansi.Width(us); nw != uw {
			t.Errorf("%s: nerd width %d, unicode width %d", name, nw, uw)
		}
	}
	if changed == 0 {
		t.Error("NerdGlyphSet() equals UnicodeGlyphSet(); expected private-use glyphs")
	}
	if theme.NerdGlyphSet().ASCII() {
		t.Error("NerdGlyphSet() reports ASCII")
	}
}

func TestDetectGlyphsNerdFont(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want theme.Glyphs
	}{
		{"nerd on", map[string]string{"TUI_NERD_FONT": "1"}, theme.NerdGlyphSet()},
		{"nerd on utf8", map[string]string{"TUI_NERD_FONT": "1", "LANG": "en_US.UTF-8"}, theme.NerdGlyphSet()},
		{"unset", map[string]string{}, theme.UnicodeGlyphSet()},
		{"not exactly 1", map[string]string{"TUI_NERD_FONT": "true"}, theme.UnicodeGlyphSet()},
		{"zero", map[string]string{"TUI_NERD_FONT": "0"}, theme.UnicodeGlyphSet()},
		{"dumb beats nerd", map[string]string{"TUI_NERD_FONT": "1", "TERM": "dumb"}, theme.ASCIIGlyphSet()},
		{"C locale beats nerd", map[string]string{"TUI_NERD_FONT": "1", "LC_ALL": "C"}, theme.ASCIIGlyphSet()},
		{"ascii unchanged", map[string]string{"LANG": "C"}, theme.ASCIIGlyphSet()},
	}
	for _, c := range cases {
		if got := theme.DetectGlyphsEnv(env(c.env)); got != c.want {
			t.Errorf("%s: got %+v, want %+v", c.name, got, c.want)
		}
	}
}
