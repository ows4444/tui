package layout

import (
	"strings"
	"testing"
)

func TestWithOverflowModes(t *testing.T) {
	path := "/usr/local/bin/tool"
	for _, tt := range []struct {
		name string
		mode Overflow
		want string
	}{
		{"end", OverflowEnd, "/usr/loca~"},
		{"start", OverflowStart, "~/bin/tool"},
		{"middle", OverflowMiddle, "/usr/~tool"},
	} {
		got := Text(path, WithOverflow(tt.mode, "")).Render(Size{W: 10, H: 1})
		if got != tt.want {
			t.Errorf("%s: %q; want %q", tt.name, got, tt.want)
		}
	}
	if got := Text("short", WithOverflow(OverflowStart, "...")).Render(Size{W: 10, H: 1}); strings.TrimSpace(got) != "short" {
		t.Errorf("a line that fits was changed: %q", got)
	}
}

func TestWithJustify(t *testing.T) {
	n := Text("the quick brown fox jumps over the lazy dog", WithWrap(), WithJustify())
	got := strings.Split(n.Render(Size{W: 16, H: 3}), "\n")
	want := []string{"the  quick brown", "fox  jumps  over", "the lazy dog"}
	for i := range want {
		if strings.TrimRight(got[i], " ") != want[i] {
			t.Errorf("line %d = %q; want %q", i, got[i], want[i])
		}
	}
	if h := n.Measure(Loose(Size{W: 16, H: 20})).H; h != 3 {
		t.Errorf("Measure H = %d; justify must not change the height", h)
	}
	two := strings.Split(Text("aa bb cc\ndd ee", WithWrap(), WithJustify()).Render(Size{W: 12, H: 2}), "\n")
	if strings.TrimRight(two[0], " ") != "aa bb cc" || strings.TrimRight(two[1], " ") != "dd ee" {
		t.Errorf("the last line of a paragraph was justified: %q", two)
	}
}
