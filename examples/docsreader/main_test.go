package main

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/input"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/markdownview"
)

func update(m model, msg tui.Msg) (model, tui.Cmd) {
	next, cmd := m.Update(msg)
	return next.(model), cmd
}

func send(m model, msg tui.Msg) model {
	m, _ = update(m, msg)
	return m
}

// run sends msg and feeds back the message its Cmd makes (a LinkMsg), once.
func run(m model, msg tui.Msg) model {
	m, cmd := update(m, msg)
	if cmd != nil {
		if out := cmd(); out != nil {
			if _, isLink := out.(markdownview.LinkMsg); isLink {
				m = send(m, out)
			}
		}
	}
	return m
}

func key(m model, t tui.KeyType) model { return run(m, tui.Key{Type: t}) }

func runes(m model, s string) model {
	for _, r := range s {
		m = run(m, tui.Key{Type: tui.KeyRunes, Text: string(r)})
	}
	return m
}

func ready() model { return send(newModel(sampleDocs()), tui.ResizeMsg{Width: 100, Height: 28}) }

func text(m model) string { return ansi.StripANSI(m.View()) }

func isQuit(cmd tui.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tui.QuitMsg)
	return ok
}

func rect(t testing.TB, m model, id string) layout.Rect {
	t.Helper()
	r, ok := layout.RectOf(m.screen(), m.size(), id)
	if !ok {
		t.Fatalf("no %q on screen:\n%s", id, text(m))
	}
	return r
}

func press(m model, x, y int) model {
	return run(m, tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
}

func at(m model) string { return m.pages[m.cur].path }

func TestOpensTheFrontPageWithTheSidebar(t *testing.T) {
	m := ready()
	out := text(m)
	for _, want := range []string{"docsreader", "Parcel docs", "Quick start", "Table", "Forms", "FAQ", "Testing", "What is in here", "6 pages · README.md"} {
		if !strings.Contains(out, want) {
			t.Errorf("screen lacks %q:\n%s", want, out)
		}
	}
	if at(m) != "README.md" {
		t.Fatalf("on %s", at(m))
	}
}

func TestFollowingALinkOpensThePage(t *testing.T) {
	m := key(key(ready(), tui.KeyTab), tui.KeyEnter)
	if at(m) != "quickstart.md" || !strings.Contains(text(m), "Send a file") {
		t.Fatalf("on %s:\n%s", at(m), text(m))
	}
	if len(m.back) != 1 || m.back[0].page != 0 {
		t.Fatalf("back = %+v", m.back)
	}
}

func TestBackAndForwardAcrossPages(t *testing.T) {
	m := key(key(ready(), tui.KeyTab), tui.KeyEnter) // quickstart
	m = m.goTo(m.pageIndex("faq.md"), "")
	if at(m) != "faq.md" {
		t.Fatal("did not reach faq")
	}
	m = key(m, tui.KeyBackspace)
	if at(m) != "quickstart.md" || len(m.fwd) != 1 {
		t.Fatalf("back: on %s, fwd %d", at(m), len(m.fwd))
	}
	m = key(m, tui.KeyBackspace)
	if at(m) != "README.md" {
		t.Fatalf("back again: on %s", at(m))
	}
	if m = key(m, tui.KeyBackspace); !strings.Contains(m.msg, "Nothing to go back to") {
		t.Fatalf("msg = %q", m.msg)
	}
	m = run(m, tui.Key{Type: tui.KeyRight, Mod: input.ModAlt})
	if at(m) != "quickstart.md" {
		t.Fatalf("forward: on %s", at(m))
	}
	m = run(m, tui.Key{Type: tui.KeyRight, Mod: input.ModAlt})
	if at(m) != "faq.md" || len(m.fwd) != 0 {
		t.Fatalf("on %s fwd %d", at(m), len(m.fwd))
	}
	if m = run(m, tui.Key{Type: tui.KeyRight, Mod: input.ModAlt}); !strings.Contains(m.msg, "Nothing to go forward to") {
		t.Fatalf("msg %q", m.msg)
	}
	m = run(m, tui.Key{Type: tui.KeyLeft, Mod: input.ModAlt})
	if at(m) != "quickstart.md" {
		t.Fatalf("Alt+Left: on %s", at(m))
	}
}

func TestANewVisitForgetsForward(t *testing.T) {
	m := m2(t)
	m = key(m, tui.KeyBackspace)
	m = m.goTo(m.pageIndex("testing.md"), "")
	if len(m.fwd) != 0 {
		t.Fatalf("fwd = %+v", m.fwd)
	}
}

func m2(t testing.TB) model {
	m := ready().goTo(2, "")
	return m.goTo(3, "")
}

func TestLinkWithAnAnchorScrollsToTheHeading(t *testing.T) {
	m := send(ready(), markdownview.LinkMsg{URL: "faq.md#troubleshooting"})
	if at(m) != "faq.md" {
		t.Fatalf("on %s", at(m))
	}
	if m.reader.YOffset() == 0 && !strings.Contains(text(m), "Troubleshooting") {
		t.Fatalf("\n%s", text(m))
	}
	m = send(ready(), markdownview.LinkMsg{URL: "faq.md#nonesuch"})
	if !strings.Contains(m.msg, "No heading #nonesuch") {
		t.Fatalf("msg = %q", m.msg)
	}
}

func TestLinkToAMissingPage(t *testing.T) {
	m := send(ready(), markdownview.LinkMsg{URL: "gone.md"})
	if at(m) != "README.md" || !strings.Contains(m.msg, "No such page: gone.md") {
		t.Fatalf("on %s msg %q", at(m), m.msg)
	}
}

func TestExternalLinkIsCopiedNotOpened(t *testing.T) {
	m, cmd := update(ready(), markdownview.LinkMsg{URL: "https://example.com/parcel"})
	if cmd == nil || !strings.Contains(m.msg, "Link copied: https://example.com/parcel") || at(m) != "README.md" {
		t.Fatalf("cmd=%v msg=%q", cmd != nil, m.msg)
	}
}

func TestRelativeLinksResolveFromTheirPage(t *testing.T) {
	m := ready().goTo(ready().pageIndex("widgets/table.md"), "")
	m = send(m, markdownview.LinkMsg{URL: "forms.md"})
	if at(m) != "widgets/forms.md" {
		t.Fatalf("on %s", at(m))
	}
	m = send(m, markdownview.LinkMsg{URL: "../README.md"})
	if at(m) != "README.md" {
		t.Fatalf("on %s", at(m))
	}
}

func TestSidebarListsTheOpenPagesHeadings(t *testing.T) {
	m := ready()
	rows := m.sideRows()
	var heads []string
	for _, r := range rows {
		if r.page < 0 {
			heads = append(heads, r.text)
		}
	}
	if len(heads) != 1 || heads[0] != "What is in here" {
		t.Fatalf("headings = %v", heads)
	}
	m = m.goTo(m.pageIndex("quickstart.md"), "")
	heads = nil
	for _, r := range m.sideRows() {
		if r.page < 0 {
			heads = append(heads, r.text)
		}
	}
	if strings.Join(heads, ",") != "Install,Send a file,Next steps" {
		t.Fatalf("headings = %v", heads)
	}
}

func TestSidebarClickOpensPagesAndHeadings(t *testing.T) {
	m := ready()
	r := rect(t, m, "side")
	var faq int
	for i, row := range m.sideRows() {
		if row.page == m.pageIndex("faq.md") {
			faq = i
		}
	}
	m = press(m, r.X+3, r.Y+faq)
	if at(m) != "faq.md" {
		t.Fatalf("clicked faq, on %s", at(m))
	}
	r = rect(t, m, "side")
	var trouble int
	for i, row := range m.sideRows() {
		if row.text == "Troubleshooting" {
			trouble = i
		}
	}
	before := len(m.back)
	m = press(m, r.X+5, r.Y+trouble)
	if len(m.back) != before+1 || at(m) != "faq.md" {
		t.Fatalf("heading click: back %d on %s", len(m.back), at(m))
	}
}

func TestSidebarKeys(t *testing.T) {
	m := runes(ready(), "s")
	if !m.side {
		t.Fatal("s did not move into the sidebar")
	}
	m = key(key(m, tui.KeyDown), tui.KeyDown) // past "What is in here"
	m = key(m, tui.KeyEnter)
	if at(m) != "faq.md" {
		t.Fatalf("on %s, want faq.md (the page after the front page's one heading)", at(m))
	}
	m = key(m, tui.KeyEsc)
	if m.side {
		t.Fatal("Esc did not leave the sidebar")
	}
	m = runes(runes(m, "s"), "q")
	if m.side {
		t.Fatal("q in the sidebar should only leave it")
	}
}

func TestSearchFindsAcrossPagesAndOpensTheHit(t *testing.T) {
	m := runes(ready(), "/")
	if !m.searching {
		t.Fatal("/ did not open the search")
	}
	m = runes(m, "firewall")
	if len(m.hits) != 1 || !strings.Contains(text(m), "FAQ › Troubleshooting") {
		t.Fatalf("hits=%d\n%s", len(m.hits), text(m))
	}
	m = key(m, tui.KeyEnter)
	if at(m) != "faq.md" || m.searching {
		t.Fatalf("on %s searching=%v", at(m), m.searching)
	}
	if len(m.back) != 1 {
		t.Fatal("opening a hit did not record where it came from")
	}
}

func TestSearchNavigationAndNoMatch(t *testing.T) {
	m := runes(runes(ready(), "/"), "table")
	n := len(m.hits)
	if n < 2 {
		t.Fatalf("hits = %d", n)
	}
	m = key(key(m, tui.KeyDown), tui.KeyDown)
	if m.hitCur != 2 {
		t.Fatalf("hitCur = %d", m.hitCur)
	}
	for i := 0; i < 50; i++ {
		m = key(m, tui.KeyDown)
	}
	if m.hitCur != n-1 {
		t.Fatalf("hitCur = %d past the end", m.hitCur)
	}
	m = runes(m, "zzzqq")
	if !strings.Contains(text(m), "No page contains that.") {
		t.Fatalf("\n%s", text(m))
	}
	if _, cmd := update(m, tui.Key{Type: tui.KeyRunes, Text: "q"}); isQuit(cmd) {
		t.Fatal("q quit while typing a search")
	}
	if m = key(m, tui.KeyEsc); m.searching {
		t.Fatal("Esc did not close the search")
	}
}

func TestSearchResultClick(t *testing.T) {
	m := runes(runes(ready(), "/"), "firewall")
	r := rect(t, m, "hits")
	m = press(m, r.X+4, r.Y)
	if at(m) != "faq.md" || m.searching {
		t.Fatalf("on %s searching=%v", at(m), m.searching)
	}
}

func TestBackForwardButtons(t *testing.T) {
	m := ready().goTo(1, "")
	if !strings.Contains(text(m), "[‹]") {
		t.Fatal("no back button")
	}
	r := rect(t, m, "back")
	m = press(m, r.X+1, r.Y)
	if at(m) != "README.md" {
		t.Fatalf("on %s", at(m))
	}
	r = rect(t, m, "fwd")
	m = press(m, r.X+1, r.Y)
	if at(m) == "README.md" {
		t.Fatal("the forward button did nothing")
	}
	m = send(m, tui.MouseEvent{X: r.X, Y: r.Y, Action: tui.MouseActionMotion})
	if m.hover != "" && m.hover != "fwd" {
		t.Fatalf("hover = %q", m.hover)
	}
}

func TestReaderScrollKeys(t *testing.T) {
	long := "# Long\n\n" + strings.Repeat("paragraph of text\n\n", 100)
	m := send(newModel(fstest.MapFS{"README.md": {Data: []byte(long)}}), tui.ResizeMsg{Width: 100, Height: 20})
	m = runes(m, " ")
	if m.reader.YOffset() == 0 {
		t.Fatal("space did not scroll")
	}
	y := m.reader.YOffset()
	m = runes(m, "j")
	if m.reader.YOffset() != y+1 {
		t.Fatalf("j: %d -> %d", y, m.reader.YOffset())
	}
	m = runes(runes(m, "k"), "g")
	if m.reader.YOffset() != 0 {
		t.Fatalf("g: offset %d", m.reader.YOffset())
	}
	if m = runes(m, "G"); m.reader.YOffset() == 0 {
		t.Fatal("G did not go to the end")
	}
	m = send(m, tui.MouseEvent{X: 50, Y: 5, Button: tui.MouseButtonWheelUp, Action: tui.MouseActionPress})
}

func TestEmptyFolder(t *testing.T) {
	m := send(newModel(fstest.MapFS{"notes.txt": {Data: []byte("x")}}), tui.ResizeMsg{Width: 80, Height: 20})
	if !strings.Contains(text(m), "No pages") {
		t.Fatalf("\n%s", text(m))
	}
	m = runes(runes(runes(m, "/"), "s"), "x")
	if _, cmd := update(m, tui.Key{Type: tui.KeyRunes, Text: "q"}); !isQuit(cmd) {
		t.Fatal("q did not quit an empty reader")
	}
	m = press(m, 3, 3)
}

func TestNarrowTerminalHidesTheSidebar(t *testing.T) {
	m := send(ready(), tui.ResizeMsg{Width: 60, Height: 20})
	if strings.Contains(text(m), "│") && strings.Contains(text(m), "Table") {
		t.Fatalf("sidebar at 60 columns:\n%s", text(m))
	}
	if m = runes(m, "s"); m.side {
		t.Fatal("the sidebar took the keys while hidden")
	}
}

func TestHelpAndQuit(t *testing.T) {
	m := runes(ready(), "?")
	if !strings.Contains(text(m), "search every page") {
		t.Fatalf("\n%s", text(m))
	}
	if m = key(m, tui.KeyEsc); m.help {
		t.Fatal("Esc did not close help")
	}
	for _, k := range []tui.Key{{Type: tui.KeyCtrlC}, {Type: tui.KeyRunes, Text: "q"}} {
		if _, cmd := update(ready(), k); !isQuit(cmd) {
			t.Errorf("%v did not quit", k)
		}
	}
}

func TestTinyTerminalsStayInside(t *testing.T) {
	m := runes(runes(ready(), "/"), "a")
	for _, sz := range [][2]int{{40, 10}, {20, 6}, {10, 4}, {300, 80}, {69, 20}, {70, 20}} {
		m = send(m, tui.ResizeMsg{Width: sz[0], Height: sz[1]})
		for i, line := range strings.Split(m.View(), "\n") {
			if w := ansi.Width(line); w > sz[0] {
				t.Fatalf("%dx%d: line %d is %d wide: %q", sz[0], sz[1], i, w, line)
			}
		}
	}
}
