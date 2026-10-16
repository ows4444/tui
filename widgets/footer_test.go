package widgets

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/keymap"
)

func TestHintsFromBindingsSkipsDisabledAndUndescribed(t *testing.T) {
	off := keymap.Binding{Keys: []string{"x"}, Desc: "hidden"}
	off.SetEnabled(false)
	got := HintsFromBindings([]keymap.Binding{
		{Keys: []string{"enter", "ctrl+m"}, Desc: "submit"},
		{Keys: []string{"q"}},
		{Desc: "no keys"},
		off,
		{Keys: []string{"esc"}, Desc: "cancel"},
	})
	want := []Hint{{"enter", "submit"}, {"esc", "cancel"}}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestFooterFitsAndDropsFromTheEnd(t *testing.T) {
	hints := []Hint{{"enter", "submit"}, {"esc", "cancel"}, {"?", "help"}}
	all := ansi.StripANSI(Footer(0, hints...))
	if all != "[enter] submit  [esc] cancel  [?] help" {
		t.Errorf("unbounded = %q", all)
	}
	if ansi.StripANSI(Footer(100, hints...)) != all {
		t.Error("a wide footer changed")
	}
	narrow := Footer(31, hints...)
	if w := ansi.Width(narrow); w > 31 {
		t.Errorf("width %d over 31: %q", w, ansi.StripANSI(narrow))
	}
	if got := ansi.StripANSI(narrow); got != "[enter] submit  [esc] cancel  …" {
		t.Errorf("narrow = %q", got)
	}
	// At 30 the ellipsis would not fit after two hints, so one is given up for it.
	if got := ansi.StripANSI(Footer(30, hints...)); got != "[enter] submit  …" {
		t.Errorf("width 30 = %q", got)
	}
	tiny := Footer(6, hints...)
	if ansi.Width(tiny) > 6 || !strings.HasPrefix(ansi.StripANSI(tiny), "[ent") {
		t.Errorf("tiny = %q", ansi.StripANSI(tiny))
	}
	if Footer(10) != "" {
		t.Error("no hints should give an empty footer")
	}
}
