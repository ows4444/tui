package main

import (
	"strconv"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

func update(m model, msg tui.Msg) (model, tui.Cmd) {
	next, cmd := m.Update(msg)
	return next.(model), cmd
}

func send(m model, msg tui.Msg) model {
	m, _ = update(m, msg)
	return m
}

func key(m model, t tui.KeyType) model { return send(m, tui.Key{Type: t}) }

func runes(m model, s string) model {
	for _, r := range s {
		m = send(m, tui.Key{Type: tui.KeyRunes, Text: string(r)})
	}
	return m
}

// ready is a reduced-motion model at 100x24, so a queue settles at once.
func ready() model { return send(newModel(true), tui.ResizeMsg{Width: 100, Height: 24}) }

// animated keeps the queue running on ticks.
func animated() model { return send(newModel(false), tui.ResizeMsg{Width: 100, Height: 24}) }

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
		t.Fatalf("no target %q on screen:\n%s", id, text(m))
	}
	return r
}

func press(m model, x, y, clicks int) model {
	return send(m, tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress, Clicks: clicks})
}

func click(t testing.TB, m model, id string) model {
	r := rect(t, m, id)
	return press(m, r.X+r.W/2, r.Y, 1)
}

// goTo selects the package called name, widening nothing: it filters.
func goTo(t testing.TB, m model, name string) model {
	t.Helper()
	for i, e := range m.visible() {
		if m.pkgs[e.idx].name == name {
			m.cur = i
			return m
		}
	}
	t.Fatalf("%s is not in the list", name)
	return m
}

func pkgNamed(m model, name string) pkg { return m.pkgs[index(m.pkgs, name)] }

func TestStartsOnTheFirstPage(t *testing.T) {
	m := ready()
	out := text(m)
	for _, want := range []string{"pkgs", "All 31", "Installed 12", "Updates 4", "Press / to filter", "argon", "Queue empty", "Install", "Size  180 KB", "Needs", "textwrap"} {
		if !strings.Contains(out, want) {
			t.Errorf("screen lacks %q:\n%s", want, out)
		}
	}
	if m.pages() != 2 || m.page() != 1 {
		t.Fatalf("pages=%d page=%d with %d rows a page", m.pages(), m.page(), m.perPage())
	}
}

func TestMovingAcrossThePageBoundaryChangesPage(t *testing.T) {
	m := ready()
	for i := 0; i < m.perPage(); i++ {
		m = key(m, tui.KeyDown)
	}
	if m.page() != 2 {
		t.Fatalf("page = %d after moving past the first page", m.page())
	}
	m = key(m, tui.KeyPgUp)
	if m.cur != 0 || m.page() != 1 {
		t.Fatalf("PgUp: cur=%d page=%d", m.cur, m.page())
	}
	m = key(m, tui.KeyEnd)
	if m.cur != 30 || m.page() != 2 {
		t.Fatalf("End: cur=%d page=%d", m.cur, m.page())
	}
	for i := 0; i < 5; i++ {
		m = key(m, tui.KeyDown)
	}
	if m.cur != 30 {
		t.Fatalf("moved past the last package: %d", m.cur)
	}
	if m = runes(m, "g"); m.cur != 0 {
		t.Fatal("g did not go to the top")
	}
}

func TestPagerClickChangesPage(t *testing.T) {
	m := ready()
	r := rect(t, m, "pager")
	// The pager reads "‹ 1 2 ›": click the 2.
	line := strings.Split(text(m), "\n")[r.Y]
	x := strings.Index(line[r.X:], "2") + r.X
	m = press(m, x, r.Y, 1)
	if m.page() != 2 || m.cur != m.perPage() {
		t.Fatalf("page=%d cur=%d after clicking page 2", m.page(), m.cur)
	}
}

func TestFilterRanksAndHighlights(t *testing.T) {
	m := runes(runes(ready(), "/"), "lg")
	v := m.visible()
	if len(v) == 0 || m.pkgs[v[0].idx].name != "logr" {
		t.Fatalf("best match for lg = %v", v)
	}
	if !strings.Contains(text(m), "match") {
		t.Fatalf("no match count:\n%s", text(m))
	}
	m = key(m, tui.KeyEnter)
	if m.typing || m.query != "lg" || !strings.Contains(text(m), "/lg") {
		t.Fatalf("Enter did not keep the filter:\n%s", text(m))
	}
	m = key(m, tui.KeyEsc)
	if m.query != "" || len(m.visible()) != 31 {
		t.Fatal("Esc did not drop the filter")
	}
}

func TestFilterSearchesDescriptionsToo(t *testing.T) {
	m := runes(runes(ready(), "/"), "backoff")
	v := m.visible()
	if len(v) != 1 || m.pkgs[v[0].idx].name != "retry" {
		t.Fatalf("matches for backoff = %v", v)
	}
	if len(v[0].pos) != 0 {
		t.Fatalf("a description match highlighted the name: %v", v[0].pos)
	}
}

func TestFilterNoMatchIsExplained(t *testing.T) {
	m := runes(runes(ready(), "/"), "zzzzq")
	if !strings.Contains(text(m), "No package matches.") {
		t.Fatalf("\n%s", text(m))
	}
	if _, cmd := update(m, tui.Key{Type: tui.KeyRunes, Text: "q"}); isQuit(cmd) {
		t.Fatal("q quit while typing a filter")
	}
	m = key(m, tui.KeyEsc)
	if m.typing {
		t.Fatal("Esc did not leave the filter box")
	}
}

func TestTabsFilterByStatus(t *testing.T) {
	m := click(t, ready(), "tab-2")
	if m.tab != tabUpdates || len(m.visible()) != 4 {
		t.Fatalf("Updates tab shows %d", len(m.visible()))
	}
	m = key(m, tui.KeyTab)
	if m.tab != tabAll {
		t.Fatalf("Tab went to %v", m.tab)
	}
	m = click(t, m, "tab-1")
	if len(m.visible()) != 12 {
		t.Fatalf("Installed tab shows %d", len(m.visible()))
	}
}

func TestInstallWithDependenciesQueuesThemFirst(t *testing.T) {
	m := animated()
	m = goTo(t, m, "gitlite")
	m, cmd := update(m, tui.Key{Type: tui.KeyEnter})
	if cmd == nil || len(m.queue) != 2 || m.pkgs[m.queue[0].idx].name != "zlibx" {
		t.Fatalf("queue = %+v", m.queue)
	}
	if !strings.Contains(m.msg, "Installing gitlite with zlibx") {
		t.Fatalf("msg = %q", m.msg)
	}
	out := text(m)
	if !strings.Contains(out, "Installing zlibx") || !strings.Contains(out, "+1 waiting") || !strings.Contains(out, "[Cancel]") {
		t.Fatalf("queue line:\n%s", out)
	}
}

func TestQueueRunsToCompletionOnTicks(t *testing.T) {
	m := goTo(t, animated(), "glob")
	m = key(m, tui.KeyEnter)
	if pkgNamed(m, "glob").installed != "" {
		t.Fatal("installed before any time passed")
	}
	for i := 0; i < 200 && len(m.queue) > 0; i++ {
		m = send(m, tickMsg{m.seq})
	}
	if pkgNamed(m, "glob").installed != "1.0.0" || len(m.queue) != 0 || m.running {
		t.Fatalf("not finished: %+v", m.queue)
	}
	if !strings.Contains(text(m), "Installed glob 1.0.0") {
		t.Fatalf("\n%s", text(m))
	}
}

func TestProgressShowsAPercentage(t *testing.T) {
	m := goTo(t, animated(), "gitlite")
	m = key(m, tui.KeyEnter)
	m = send(m, tickMsg{m.seq})
	if !strings.Contains(text(m), "  12%") && !strings.Contains(text(m), " 15%") && !strings.Contains(text(m), "%") {
		t.Fatalf("no progress shown:\n%s", text(m))
	}
}

func TestCancelEmptiesTheQueueAndIgnoresLateTicks(t *testing.T) {
	m := goTo(t, animated(), "gitlite")
	m = key(m, tui.KeyEnter)
	seq := m.seq
	m = click(t, m, "cancel")
	if len(m.queue) != 0 || m.running || m.msg != "Queue cancelled" {
		t.Fatalf("queue=%v running=%v msg=%q", m.queue, m.running, m.msg)
	}
	m = send(m, tickMsg{seq})
	if pkgNamed(m, "zlibx").installed != "" || len(m.queue) != 0 {
		t.Fatal("a stale tick finished a cancelled job")
	}
	if m = runes(m, "c"); m.msg != "The queue is empty" {
		t.Fatalf("msg = %q", m.msg)
	}
}

func TestReducedMotionInstallsAtOnce(t *testing.T) {
	m := goTo(t, ready(), "gitlite")
	m = key(m, tui.KeyEnter)
	if pkgNamed(m, "gitlite").installed == "" || pkgNamed(m, "zlibx").installed == "" || len(m.queue) != 0 {
		t.Fatal("gitlite and zlibx were not installed")
	}
}

func TestUpdateAndUpToDateAndQueuedMessages(t *testing.T) {
	m := goTo(t, ready(), "clockwork")
	m = key(m, tui.KeyEnter)
	if pkgNamed(m, "clockwork").installed != "1.4.1" || !strings.Contains(m.msg, "Updated clockwork to 1.4.1") {
		t.Fatalf("msg = %q", m.msg)
	}
	m = key(m, tui.KeyEnter)
	if m.msg != "clockwork is up to date" {
		t.Fatalf("msg = %q", m.msg)
	}
	a := goTo(t, animated(), "glob")
	a = key(key(a, tui.KeyEnter), tui.KeyEnter)
	if a.msg != "glob is already in the queue" || len(a.queue) != 1 {
		t.Fatalf("msg=%q queue=%d", a.msg, len(a.queue))
	}
}

func TestUpdateAll(t *testing.T) {
	m := runes(ready(), "U")
	if !strings.HasPrefix(m.msg, "Updated ") {
		t.Fatalf("msg = %q", m.msg)
	}
	if m.counts()[2] != 0 {
		t.Fatalf("%d updates left", m.counts()[2])
	}
	if m = runes(m, "U"); m.msg != "Everything is up to date" {
		t.Fatalf("msg = %q", m.msg)
	}
}

func TestRemoveRespectsDependents(t *testing.T) {
	m := goTo(t, ready(), "textwrap")
	m = runes(m, "x")
	if !strings.Contains(m.msg, "Cannot remove textwrap: argon needs it") || pkgNamed(m, "textwrap").installed == "" {
		t.Fatalf("msg = %q", m.msg)
	}
	m = goTo(t, ready(), "csvkit")
	m = runes(m, "x")
	if pkgNamed(m, "csvkit").installed != "" || m.msg != "Removed csvkit" {
		t.Fatalf("msg = %q", m.msg)
	}
	if m = runes(m, "x"); m.msg != "csvkit is not installed" {
		t.Fatalf("msg = %q", m.msg)
	}
}

func TestInstalledTabShrinksAfterARemoval(t *testing.T) {
	m := click(t, ready(), "tab-1")
	m = goTo(t, m, "csvkit")
	m = runes(m, "x")
	if len(m.visible()) != 11 {
		t.Fatalf("%d installed after a removal", len(m.visible()))
	}
	if m.cur >= len(m.visible()) {
		t.Fatal("the cursor is past the end of the list")
	}
}

func TestClickRowAndDoubleClick(t *testing.T) {
	m := ready()
	r := rect(t, m, "row-3")
	m = press(m, r.X+3, r.Y, 1)
	if m.cur != 3 {
		t.Fatalf("cur = %d after clicking row 3", m.cur)
	}
	m = goTo(t, m, "glob")
	r = rect(t, m, "row-"+strconv.Itoa(m.cur%m.perPage()))
	m = press(m, r.X+3, r.Y, 2)
	if pkgNamed(m, "glob").installed == "" {
		t.Fatal("a double click did not install")
	}
}

func TestDetailButtons(t *testing.T) {
	m := goTo(t, ready(), "dotenv")
	if !strings.Contains(text(m), "Update to 1.1.0") {
		t.Fatalf("\n%s", text(m))
	}
	m = click(t, m, "act-main")
	if pkgNamed(m, "dotenv").installed != "1.1.0" {
		t.Fatal("the button did not update")
	}
	m = click(t, m, "act-remove")
	if pkgNamed(m, "dotenv").installed != "" {
		t.Fatal("the Remove button did not remove")
	}
}

func TestWheelMoves(t *testing.T) {
	m := send(ready(), tui.MouseEvent{Button: tui.MouseButtonWheelDown, Action: tui.MouseActionPress})
	if m.cur != 1 {
		t.Fatalf("cur = %d", m.cur)
	}
	m = send(send(m, tui.MouseEvent{Button: tui.MouseButtonWheelUp, Action: tui.MouseActionPress}), tui.MouseEvent{Button: tui.MouseButtonWheelUp, Action: tui.MouseActionPress})
	if m.cur != 0 {
		t.Fatalf("cur = %d", m.cur)
	}
}

func TestNarrowTerminalHidesTheDetail(t *testing.T) {
	m := send(ready(), tui.ResizeMsg{Width: 50, Height: 16})
	if strings.Contains(text(m), "Size  ") {
		t.Fatalf("detail shown at 50 columns:\n%s", text(m))
	}
	if m.perPage() != 11 {
		t.Fatalf("perPage = %d", m.perPage())
	}
}

func TestHelpAndQuit(t *testing.T) {
	m := key(ready(), tui.KeyF1)
	if !strings.Contains(text(m), "install or update, with what it needs") {
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
	m := runes(runes(animated(), "/"), "o")
	m = key(m, tui.KeyEnter)
	m = key(m, tui.KeyEnter)
	for _, sz := range [][2]int{{40, 10}, {20, 6}, {10, 4}, {300, 80}} {
		m = send(m, tui.ResizeMsg{Width: sz[0], Height: sz[1]})
		for i, line := range strings.Split(m.View(), "\n") {
			if w := ansi.Width(line); w > sz[0] {
				t.Fatalf("%dx%d: line %d is %d wide: %q", sz[0], sz[1], i, w, line)
			}
		}
	}
}
