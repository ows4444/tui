package prompt

import (
	"errors"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/picker"
)

type region struct{ id string }

func regions() []Choice[region] {
	return []Choice[region]{
		{Label: "Frankfurt", Value: region{"eu-central"}, Group: "Europe"},
		{Label: "Dublin", Value: region{"eu-west"}, Group: "Europe"},
		{Label: "Oregon", Value: region{"us-west"}, Group: "Americas"},
		{Label: "Sydney", Value: region{"ap-se"}},
	}
}

func TestChooseReturnsTheValue(t *testing.T) {
	o, out := drive("\x1b[B\x1b[B\r") // Down, Down, Enter: the third
	got, err := Choose("Region", regions(), o)
	if err != nil || got.id != "us-west" {
		t.Fatalf("Choose = %+v, %v", got, err)
	}
	if s := ansi.StripANSI(out.String()); !strings.Contains(s, "Europe > Frankfurt") || !strings.Contains(s, "Sydney") {
		t.Errorf("groups not shown:\n%s", s)
	}
}

func TestChooseFiltersByGroupName(t *testing.T) {
	o, _ := drive("Amer\r")
	got, err := Choose("Region", regions(), o)
	if err != nil || got.id != "us-west" {
		t.Fatalf("typing the group should pick Oregon: %+v, %v", got, err)
	}
}

func TestStartMovesTheCursor(t *testing.T) {
	o, _ := drive("\r")
	o.Start = 3
	got, err := Choose("Region", regions(), o)
	if err != nil || got.id != "ap-se" {
		t.Fatalf("Start 3 gave %+v, %v", got, err)
	}
	o2, _ := drive("\r")
	o2.Start = 99
	if i, err := Select("x", []string{"a", "b"}, o2); err != nil || i != 0 {
		t.Fatalf("an out of range Start should be the first: %d %v", i, err)
	}
}

func TestPageSizeLimitsWhatIsDrawn(t *testing.T) {
	var items []string
	for _, n := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
		items = append(items, "item-"+n)
	}
	o, out := drive("\r")
	o.PageSize = 3
	o.NoFilter = true
	if _, err := Select("Pick", items, o); err != nil {
		t.Fatal(err)
	}
	s := ansi.StripANSI(out.String())
	if strings.Contains(s, "item-e") {
		t.Fatalf("PageSize 3 drew a fifth entry:\n%s", s)
	}
	// The window follows the cursor: after five Downs the sixth entry is in it.
	p := picker.NewStrings(items...)
	m := selectModel{label: "Pick", p: p, page: 3}
	for i := 0; i < 5; i++ {
		next, _ := m.Update(tui.Key{Type: tui.KeyDown})
		m = next.(selectModel)
	}
	v := ansi.StripANSI(m.View())
	if !strings.Contains(v, "item-f") || strings.Contains(v, "item-a") || strings.Count(v, "item-") != 3 {
		t.Fatalf("window after five Downs:\n%s", v)
	}
}

func TestChooseErrorsAndCancel(t *testing.T) {
	if _, err := Choose("x", []Choice[int](nil)); err == nil {
		t.Fatal("no choices should be an error")
	}
	o, _ := drive("\x1b")
	if _, err := Choose("x", regions(), o); !errors.Is(err, ErrCanceled) {
		t.Fatalf("Esc: %v", err)
	}
	o, _ = drive("\r")
	got, err := Choose("x", []Choice[string]{{Label: "\x1b]52;c;ZXZpbA==\x07evil", Value: "v"}}, o)
	if err != nil || got != "v" {
		t.Fatalf("%q %v", got, err)
	}
}

func TestPageOfKeepsTheQueryLine(t *testing.T) {
	view := "query\n1\n2\n3\n4\n5\n6"
	got := pageOf(view, 4, 3, true)
	lines := strings.Split(got, "\n")
	if len(lines) != 4 || lines[0] != "query" || !strings.Contains(got, "5") {
		t.Fatalf("%q", got)
	}
	if pageOf(view, 0, 0, true) != view || pageOf("a\nb", 0, 5, false) != "a\nb" {
		t.Fatal("no page, or a short list, must be unchanged")
	}
}
