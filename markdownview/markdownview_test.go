package markdownview

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/input"
	"github.com/ows4444/tui/layout"
)

const doc = "# Guide\n\nIntro with a [link to setup](#setup) and an [outside one](https://example.com/x).\n\n" +
	"## Setup\n\nStep one.\n\n```\n# not a heading\n[nor](this)\n```\n\n" +
	"Second title\n------------\n\nText.\n\n## Setup\n\nAgain, see [top](#guide).\n"

func lines(n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteString("filler line\n\n")
	}
	return b.String()
}

func long() Model {
	return New("# Top\n\n[jump](#end)\n\n"+lines(30)+"## End\n\nlast words\n\n"+lines(12), 40, 8)
}

func TestHeadingsAnchorsAndLevels(t *testing.T) {
	m := New(doc, 60, 20)
	var got []string
	for _, h := range m.Headings() {
		got = append(got, h.Anchor)
	}
	if strings.Join(got, ",") != "guide,setup,second-title,setup-1" {
		t.Fatalf("anchors %v", got)
	}
	hs := m.Headings()
	if hs[0].Level != 1 || hs[1].Level != 2 || hs[2].Level != 2 || hs[2].Text != "Second title" {
		t.Fatalf("headings %+v", hs)
	}
	for i := 1; i < len(hs); i++ {
		if hs[i].Line <= hs[i-1].Line {
			t.Fatalf("heading rows do not increase: %+v", hs)
		}
	}
}

func TestLinksSkipCodeAndImages(t *testing.T) {
	m := New(doc+"\n![pic](a.png)\n", 60, 30)
	ls := m.Links()
	if len(ls) != 3 || ls[0].URL != "#setup" || ls[1].URL != "https://example.com/x" || ls[2].URL != "#guide" {
		t.Fatalf("links %+v", ls)
	}
	if ls[0].Text != "link to setup" {
		t.Fatalf("text %q", ls[0].Text)
	}
}

func TestTabWalksLinksAndFollowGoesToAnAnchor(t *testing.T) {
	m := long()
	m = press(m, tui.Key{Type: tui.KeyTab})
	l, ok := m.Active()
	if !ok || l.URL != "#end" {
		t.Fatalf("active %+v", l)
	}
	if v := ansi.StripANSI(m.View()); !strings.Contains(v, "jump -> #end") {
		t.Fatalf("no status row:\n%s", v)
	}
	m = press(m, tui.Key{Type: tui.KeyEnter})
	end := m.Headings()[1].Line
	if m.YOffset() != end {
		t.Fatalf("offset %d; want the End heading at %d", m.YOffset(), end)
	}
	m = press(m, tui.Key{Type: tui.KeyBackspace})
	if m.YOffset() != 0 {
		t.Fatalf("Back went to %d", m.YOffset())
	}
	m = press(m, tui.Key{Type: tui.KeyRight, Mod: input.ModAlt})
	if m.YOffset() != end {
		t.Fatalf("Forward went to %d", m.YOffset())
	}
}

func TestOutsideLinkIsReported(t *testing.T) {
	m := New(doc, 60, 20)
	m = press(m, tui.Key{Type: tui.KeyTab})
	m = press(m, tui.Key{Type: tui.KeyTab}) // the outside link
	_, cmd := m.Update(tui.Key{Type: tui.KeyEnter})
	if got, ok := cmd().(LinkMsg); !ok || got.URL != "https://example.com/x" || got.Text != "outside one" {
		t.Fatalf("%+v", got)
	}
}

func TestShiftTabWrapsBackwards(t *testing.T) {
	m := New(doc, 60, 20)
	m = press(m, tui.Key{Type: tui.KeyTab, Mod: input.ModShift})
	if l, _ := m.Active(); l.URL != "#guide" {
		t.Fatalf("Shift+Tab from nothing should reach the last link, got %+v", l)
	}
}

func TestHeadingKeysAndHistory(t *testing.T) {
	m := long()
	m = press(m, tui.Key{Type: tui.KeyRunes, Text: "]"})
	if m.YOffset() != m.Headings()[1].Line {
		t.Fatalf("] went to %d", m.YOffset())
	}
	m = press(m, tui.Key{Type: tui.KeyRunes, Text: "["})
	if m.YOffset() != 0 {
		t.Fatalf("[ went to %d", m.YOffset())
	}
	m = press(m, tui.Key{Type: tui.KeyBackspace}, tui.Key{Type: tui.KeyBackspace}, tui.Key{Type: tui.KeyBackspace})
	if m.YOffset() != 0 {
		t.Fatalf("Back past the start moved: %d", m.YOffset())
	}
}

func TestGoToAndUnknownAnchor(t *testing.T) {
	m := long()
	if !m.GoTo("#end") || m.YOffset() != m.Headings()[1].Line {
		t.Fatal("GoTo did not scroll")
	}
	if m.GoTo("nope") {
		t.Fatal("GoTo found an anchor that is not there")
	}
}

func TestScrollKeysStillWork(t *testing.T) {
	m := long()
	m = press(m, tui.Key{Type: tui.KeyDown}, tui.Key{Type: tui.KeyDown})
	if m.YOffset() != 2 {
		t.Fatalf("Down scrolled to %d", m.YOffset())
	}
}

func TestSetSizeRendersAgainAndSourceResets(t *testing.T) {
	m := long()
	m.GoTo("end")
	m.SetSize(20, 6)
	if m.Width() != 20 || m.Height() != 6 {
		t.Fatal("size")
	}
	if m.YOffset() != m.Headings()[1].Line {
		t.Fatalf("the heading at the top moved: %d vs %d", m.YOffset(), m.Headings()[1].Line)
	}
	m.SetSource("plain text, no links")
	if m.YOffset() != 0 || len(m.Links()) != 0 || len(m.Headings()) != 0 {
		t.Fatal("SetSource kept old state")
	}
	if strings.Contains(m.View(), "\n") && strings.Count(m.View(), "\n") > 5 {
		t.Fatal("a document without links should not reserve a status row")
	}
	if m.Linearize() == "" || len(m.Bindings()) < 7 {
		t.Fatal("Linearize or Bindings")
	}
}

func TestSlugAndEmptyDocument(t *testing.T) {
	if got := slug("Hello, World! 2: éa_b-c"); got != "hello-world-2-éa_b-c" {
		t.Fatalf("slug %q", got)
	}
	m := New("", 20, 5)
	if _, ok := m.Active(); ok {
		t.Fatal("empty document has an active link")
	}
	press(m, tui.Key{Type: tui.KeyTab}, tui.Key{Type: tui.KeyEnter})
}

func press(m Model, keys ...tui.Key) Model {
	for _, k := range keys {
		m, _ = m.Update(k)
	}
	return m
}

func TestMouseIsOffByDefaultAndWheelScrollsWhenOn(t *testing.T) {
	m := long()
	wheel := tui.MouseEvent{X: 1, Y: 1, Button: tui.MouseButtonWheelDown, Action: tui.MouseActionPress}
	if m2, _ := m.Update(wheel); m2.YOffset() != 0 {
		t.Fatal("the wheel scrolled with Mouse off")
	}
	m.Mouse = true
	m.Bounds.W, m.Bounds.H = 40, 8
	if m2, _ := m.Update(wheel); m2.YOffset() == 0 {
		t.Fatal("the wheel did not scroll with Mouse on")
	}
}

func TestLayoutNodeFillsAndKeepsTheStatusRow(t *testing.T) {
	m := long()
	m = press(m, tui.Key{Type: tui.KeyTab})
	out := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 40, H: 6}))
	rows := strings.Split(out, "\n")
	if len(rows) != 6 || !strings.Contains(rows[5], "jump -> #end") {
		t.Fatalf("layout node:\n%s", out)
	}
	if rows := strings.Split(ansi.StripANSI(New("plain", 20, 4).LayoutNode().Render(layout.Size{W: 20, H: 4})), "\n"); len(rows) != 4 {
		t.Fatal("a document without links should fill the whole height")
	}
}
