package main

import (
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/input"
	"github.com/ows4444/tui/tuitest"
)

// send applies each msg and then the message its Cmd delivers, as a Program
// would, and returns the model that results. A batch cannot be taken apart
// outside package tui, so mouse events, which come back as one, go through
// tuitest in the tests below. The address field's blink Cmd waits on a
// timer, so it is not run here.
func send(m model, msgs ...tui.Msg) model {
	for _, msg := range msgs {
		next, cmd := m.Update(msg)
		m = next.(model)
		if cmd != nil && m.page() != pageAddress {
			if out := cmd(); out != nil {
				next, _ = m.Update(out)
				m = next.(model)
			}
		}
	}
	return m
}

func key(t tui.KeyType) tui.Key { return tui.Key{Type: t} }

func char(s string) tui.Key { return tui.Key{Type: tui.KeyRunes, Text: s} }

var (
	tab    = key(tui.KeyTab)
	pgDown = key(tui.KeyPgDown)
	pgUp   = key(tui.KeyPgUp)
)

func plain(m model) string { return ansi.StripANSI(m.View()) }

// at returns the gallery on page p with the keys on the page.
func at(p int) model {
	m := initialModel()
	for m.page() < p {
		m = send(m, pgDown)
	}
	if m.onPager && m.hasBody() {
		m = send(m, tab)
	}
	return m
}

func TestPgDownAndPgUpWalkThePagesAndStopAtTheEnds(t *testing.T) {
	m := initialModel()
	if m.page() != pageCode || !m.code.Focused() {
		t.Fatal("the gallery does not start on the code page with its field focused")
	}
	m = send(m, pgUp)
	if m.page() != pageCode {
		t.Fatal("PgUp left the first page")
	}
	for p := 2; p <= pageCount; p++ {
		m = send(m, pgDown)
		if m.page() != p || !strings.Contains(plain(m), " Gallery / "+names[p]) {
			t.Fatalf("page %d: at %d\n%s", p, m.page(), plain(m))
		}
	}
	m = send(m, pgDown)
	if m.page() != pageCount {
		t.Fatal("PgDown left the last page")
	}
}

func TestTabMovesBetweenThePageAndThePager(t *testing.T) {
	m := send(initialModel(), tab)
	if !m.pager.Focused() || m.code.Focused() || !strings.Contains(plain(m), "<1>") {
		t.Fatalf("tab did not focus the pager:\n%s", plain(m))
	}
	m = send(m, key(tui.KeyRight), key(tui.KeyRight)) // the pager's own keys
	if m.page() != pageColumns || !m.pager.Focused() {
		t.Fatalf("page %d", m.page())
	}
	m = send(m, tui.Key{Type: tui.KeyTab, Mod: input.ModShift})
	if !m.cols.Focused() || m.pager.Focused() {
		t.Fatal("shift+tab did not go back to the page")
	}
}

// A page that only draws leaves the keys on the pager.
func TestAPageWithNothingToFocusKeepsThePagerFocused(t *testing.T) {
	m := at(pageHistory)
	if !m.pager.Focused() || !strings.Contains(plain(m), "Build passed") {
		t.Fatalf("history:\n%s", plain(m))
	}
	m = send(m, tab)
	if !m.pager.Focused() {
		t.Fatal("tab took focus off the pager on a page with nothing to focus")
	}
	m = send(m, key(tui.KeyRight))
	if m.page() != pageEmpty || !strings.Contains(plain(m), "No results") {
		t.Fatalf("empty:\n%s", plain(m))
	}
}

func TestTheCodePageReadsTheCodeBack(t *testing.T) {
	m := send(at(pageCode), char("4"), char("8"))
	if !strings.Contains(plain(m), "Code so far: 48") {
		t.Fatalf("status:\n%s", plain(m))
	}
	m = send(m, char("2"), char("9"), char("1"), char("3"))
	if !strings.Contains(plain(m), "Complete: 482913") || !strings.Contains(plain(m), "[4][8][2] — [9][1]") {
		t.Fatalf("status:\n%s", plain(m))
	}
}

func TestTheAddressPageJoinsTheEnds(t *testing.T) {
	m := send(at(pageAddress), char("g"), char("o"))
	if !m.site.Focused() || !strings.Contains(plain(m), "Address: https://go.com") {
		t.Fatalf("status:\n%s", plain(m))
	}
}

func TestTheColumnsPageMovesAColumnAcross(t *testing.T) {
	m := send(at(pageColumns), key(tui.KeyDown), key(tui.KeyEnter))
	if !strings.Contains(plain(m), "Size is now shown.") || !strings.Contains(plain(m), "Shown (1)") {
		t.Fatalf("after Enter:\n%s", plain(m))
	}
	m = send(m, key(tui.KeyRight), key(tui.KeyEnter))
	if !strings.Contains(plain(m), "Size is now hidden.") {
		t.Fatalf("after moving back:\n%s", plain(m))
	}
}

func TestTheTipsPageLoops(t *testing.T) {
	m := send(at(pageTips), key(tui.KeyLeft))
	if !strings.Contains(plain(m), "Tip 3 of 3.") || !strings.Contains(plain(m), "Press q to quit") {
		t.Fatalf("after Left on the first tip:\n%s", plain(m))
	}
}

// The keys reach all three overlays: the button's hint shows with focus,
// Enter on the name opens its card, and Enter on the button asks over a
// dimmed frame.
func TestTheOverlaysPageByKeys(t *testing.T) {
	m := at(pageOverlays)
	if !strings.Contains(plain(m), "Asks before it deletes") || !strings.Contains(plain(m), "< Delete >") {
		t.Fatalf("no hint under the focused button:\n%s", plain(m))
	}
	m = send(m, key(tui.KeyRight))
	if strings.Contains(plain(m), "Asks before") || !strings.Contains(plain(m), "<@ada>") {
		t.Fatalf("after Right:\n%s", plain(m))
	}
	m = send(m, key(tui.KeyEnter))
	if !m.card.Open() || !strings.Contains(plain(m), "Wrote the first program,") {
		t.Fatalf("no card:\n%s", plain(m))
	}
	m = send(m, key(tui.KeyEsc))
	if m.card.Open() {
		t.Fatal("Esc did not close the card")
	}
	m = send(m, key(tui.KeyLeft), key(tui.KeyEnter))
	if !m.ask.Open() || !m.dim.Open() || !strings.Contains(plain(m), "Delete the file?") {
		t.Fatalf("no dialog:\n%s", plain(m))
	}
	if plainBehind := plain(m); !strings.Contains(plainBehind, " Gallery / Overlays") {
		t.Fatalf("the dimmed frame lost its text:\n%s", plainBehind)
	}
	m = send(m, pgUp) // a dialog takes every key
	if m.page() != pageOverlays || !m.ask.Open() {
		t.Fatal("a key reached the gallery behind the dialog")
	}
	m = send(m, key(tui.KeyEsc))
	if m.ask.Open() || m.dim.Open() || !strings.Contains(plain(m), "Nothing was deleted.") {
		t.Fatalf("after Esc:\n%s", plain(m))
	}
}

func TestEscQuitsWhenNothingIsOpen(t *testing.T) {
	for _, k := range []tui.Key{key(tui.KeyEsc), {Type: tui.KeyRunes, Text: "c", Mod: input.ModCtrl}} {
		if _, cmd := initialModel().Update(k); cmd == nil {
			t.Errorf("%v did not quit", k)
		} else if _, ok := cmd().(tui.QuitMsg); !ok {
			t.Errorf("%v returned a Cmd that is not Quit", k)
		}
	}
}

func TestEveryPageFitsANarrowAndAShortTerminal(t *testing.T) {
	for _, sz := range [][2]int{{30, 8}, {40, 10}, {80, 24}} {
		for p := 1; p <= pageCount; p++ {
			m := send(at(p), tui.ResizeMsg{Width: sz[0], Height: sz[1]})
			lines := strings.Split(plain(m), "\n")
			if sz[1] >= 10 && len(lines) > sz[1] {
				t.Errorf("page %d at %v is %d rows", p, sz, len(lines))
			}
			for i, l := range lines {
				if w := ansi.Width(l); w > sz[0] {
					t.Errorf("page %d at %v: line %d is %d cells wide: %q", p, sz, i, w, l)
				}
			}
		}
	}
	if (model{}).contentWidth() != 60 {
		t.Error("contentWidth before the first resize is not 60")
	}
}

// session runs the program at 80x24 reporting every pointer movement, as
// main does.
func session(t *testing.T) *tuitest.Session {
	t.Helper()
	s := tuitest.New(initialModel(), 80, 24, tui.WithMouse(tui.MouseAllMotion))
	t.Cleanup(s.Close)
	return s
}

func waitFor(t *testing.T, s *tuitest.Session, text string) {
	t.Helper()
	if !s.WaitForText(text, 3*time.Second) {
		t.Fatalf("the screen never showed %q:\n%s", text, strings.Join(s.Screen(), "\n"))
	}
}

func hover(s *tuitest.Session, x, y int) {
	s.Send(tui.MouseEvent{X: x, Y: y, Action: tui.MouseActionMotion, Button: tui.MouseButtonNone})
}

// At 80x24 the breadcrumb is row 0, the page starts on row 2 and the pager
// is row 12: " 1  2  3  4  5  6  7 ", so page n's number is at column 3n-2.
func TestTheMouseWalksTheGallery(t *testing.T) {
	s := session(t)
	s.Click(7, 12) // the pager: page 3
	waitFor(t, s, "Gallery / Columns")
	s.Click(2, 4) // the second column of the hidden list: Size
	waitFor(t, s, "Size is now shown.")
	s.Click(10, 12) // page 4
	waitFor(t, s, "Press ? for help")
	s.Click(12, 2) // the right half of the slide
	waitFor(t, s, "Tip 2 of 3.")
	s.Click(2, 0) // the breadcrumb's first place
	waitFor(t, s, "Gallery / Code")
}

// " < Delete >   by @ada ": the button is columns 2..11 of row 2 and the name
// columns 18..23.
func TestThePointerOpensTheOverlays(t *testing.T) {
	s := session(t)
	s.Click(19, 12) // page 7
	waitFor(t, s, "Gallery / Overlays")
	hover(s, 5, 2)
	waitFor(t, s, "Asks before it deletes")
	hover(s, 20, 2)
	waitFor(t, s, "Ada Lovelace")
	hover(s, 22, 5) // onto the card: it stays
	hover(s, 60, 9) // off both: it closes
	// Messages are handled in order, so once the hint shows again the card's
	// closing has been drawn too.
	hover(s, 5, 2)
	waitFor(t, s, "Asks before it deletes")
	if screen := strings.Join(s.Screen(), "\n"); strings.Contains(screen, "Ada Lovelace") {
		t.Fatalf("the card stayed after the pointer left:\n%s", screen)
	}
	s.Click(5, 2) // the button
	waitFor(t, s, "Delete the file?")
	s.Click(70, 20) // outside the dialog
	waitFor(t, s, "Nothing was deleted.")
}
