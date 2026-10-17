package main

import (
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

func ready(t testing.TB) model {
	t.Helper()
	m := newModel(mustParse(t, sampleJSON), "sample.json")
	return send(m, tui.ResizeMsg{Width: 100, Height: 30})
}

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

// rowY is the screen line of row i.
func rowY(t testing.TB, m model, i int) int { return rect(t, m, "tree").Y + i - m.top }

// find moves the cursor to the row whose node has path p.
func find(t testing.TB, m model, p string) model {
	t.Helper()
	for i, r := range m.rows() {
		if r.n.path() == p {
			m.cur = i
			return m.keep()
		}
	}
	t.Fatalf("no visible row with path %s", p)
	return m
}

func TestOpensWithTheTopLevelShown(t *testing.T) {
	out := text(ready(t))
	for _, want := range []string{"jsonx", "sample.json", "$", "status", `"ok"`, "shop", "{4 keys}", "orders", "[2 items]", "object · "} {
		if !strings.Contains(out, want) {
			t.Errorf("screen lacks %q:\n%s", want, out)
		}
	}
	if n := len(ready(t).rows()); n != 6 {
		t.Errorf("%d rows shown, want the root and its 5 members", n)
	}
}

func TestOpenAndCloseWithKeys(t *testing.T) {
	m := find(t, ready(t), "$.orders")
	m = key(m, tui.KeyRight)
	if !m.open[m.selected()] || !strings.Contains(text(m), "[0]") {
		t.Fatalf("Right did not open:\n%s", text(m))
	}
	m = key(m, tui.KeyRight) // steps in
	if m.selected().path() != "$.orders[0]" {
		t.Fatalf("Right on an open branch went to %s", m.selected().path())
	}
	m = key(m, tui.KeyLeft) // an element that is closed: to the parent
	if m.selected().path() != "$.orders" {
		t.Fatalf("Left on a closed branch went to %s", m.selected().path())
	}
	m = key(m, tui.KeyLeft)
	if m.open[m.selected()] || strings.Contains(text(m), "[0]") {
		t.Fatal("Left did not close")
	}
	m = runes(m, "l")
	m = key(m, tui.KeyEnter)
	if m.open[m.selected()] {
		t.Fatal("Enter did not toggle it closed")
	}
	m = runes(m, " ")
	if !m.open[m.selected()] {
		t.Fatal("Space did not toggle it open")
	}
}

func TestRootStaysOpen(t *testing.T) {
	m := ready(t)
	m.cur = 0
	m = key(key(m, tui.KeyLeft), tui.KeyEnter)
	if !m.open[m.root] {
		t.Fatal("the root was closed")
	}
	m = runes(m, "C")
	if !m.open[m.root] || len(m.rows()) != 1+4+0 && len(m.rows()) < 5 {
		t.Fatalf("C: %d rows", len(m.rows()))
	}
}

func TestExpandAllAndCollapseAll(t *testing.T) {
	m := runes(ready(t), "E")
	if len(m.rows()) != m.root.count() {
		t.Fatalf("E shows %d rows of %d", len(m.rows()), m.root.count())
	}
	m = runes(m, "C")
	if len(m.rows()) != len(m.root.children)+1 || m.cur != 0 {
		t.Fatalf("C shows %d rows, cur %d", len(m.rows()), m.cur)
	}
	m = find(t, m, "$.totals")
	m = runes(m, "*")
	if !strings.Contains(text(m), "by-status") || !strings.Contains(text(m), "unpaid") {
		t.Fatalf("* did not open everything below:\n%s", text(m))
	}
	if strings.Contains(text(m), "Ada Lovelace") {
		t.Fatal("* opened a sibling branch")
	}
}

func TestPathBarFollowsTheCursor(t *testing.T) {
	m := runes(ready(t), "E")
	m = find(t, m, "$.orders[0].customer.tags[1]")
	if !strings.Contains(text(m), "$.orders[0].customer.tags[1]") {
		t.Fatalf("path bar:\n%s", text(m))
	}
	m = find(t, m, "$.totals[\"by-status\"]")
	if !strings.Contains(text(m), `$.totals["by-status"]`) {
		t.Fatalf("quoted key path:\n%s", text(m))
	}
}

func TestPreviewShowsTheSelectedValue(t *testing.T) {
	m := find(t, ready(t), "$.shop")
	out := text(m)
	for _, want := range []string{"object · 4 values inside", `"name": "Analytical Engines"`, `"rating": 4.50`, `"closed_on": null`} {
		if !strings.Contains(out, want) {
			t.Errorf("preview lacks %q:\n%s", want, out)
		}
	}
	m = send(m, tui.ResizeMsg{Width: 60, Height: 20})
	if strings.Contains(text(m), "values inside") {
		t.Fatal("the preview shows at 60 columns")
	}
}

func TestSearchJumpsAndOpensAncestors(t *testing.T) {
	m := runes(ready(t), "/")
	if !m.typing {
		t.Fatal("/ did not open the search")
	}
	m = runes(m, "babbage")
	if m.selected().path() != "$.orders[1].customer.name" {
		t.Fatalf("selected %s after typing", m.selected().path())
	}
	if !strings.Contains(text(m), "1 of 1") {
		t.Fatalf("no match count:\n%s", text(m))
	}
	m = key(m, tui.KeyEnter)
	if m.typing || m.query != "babbage" {
		t.Fatal("Enter did not keep the search")
	}
}

func TestSearchNextPreviousAndWrap(t *testing.T) {
	m := runes(runes(ready(t), "/"), "name")
	m = key(m, tui.KeyEnter)
	var seen []string
	for i := 0; i < 5; i++ {
		seen = append(seen, m.selected().path())
		m = runes(m, "n")
	}
	want := "$.shop.name,$.orders[0].customer.name,$.orders[1].customer.name,$.shop.name,$.orders[0].customer.name"
	if got := strings.Join(seen, ","); got != want {
		t.Fatalf("n went %s, want %s", got, want)
	}
	if m = runes(m, "N"); m.selected().path() != "$.orders[0].customer.name" {
		t.Fatalf("N went to %s", m.selected().path())
	}
	m = runes(m, "N")
	if m = runes(m, "N"); m.selected().path() != "$.orders[1].customer.name" {
		t.Fatalf("N did not wrap to the last match: %s", m.selected().path())
	}
}

func TestSearchEscGoesBack(t *testing.T) {
	m := find(t, ready(t), "$.status")
	at := m.cur
	m = runes(runes(m, "/"), "babbage")
	m = key(m, tui.KeyEsc)
	if m.typing || m.query != "" || m.cur != at {
		t.Fatalf("Esc: typing=%v query=%q cur=%d want %d", m.typing, m.query, m.cur, at)
	}
	m = runes(runes(m, "/"), "nonesuch")
	m = key(m, tui.KeyEnter)
	if !strings.Contains(text(m), `No match for "nonesuch"`) {
		t.Fatalf("\n%s", text(m))
	}
	m = key(m, tui.KeyEsc)
	if m.query != "" {
		t.Fatal("Esc outside the search box did not drop the search")
	}
}

func TestMatchesAreMarked(t *testing.T) {
	m := runes(runes(ready(t), "/"), "ada")
	m = key(m, tui.KeyEnter)
	for _, line := range strings.Split(text(m), "\n") {
		line = ansi.Truncate(line, m.treeW()) // the tree, not the preview beside it
		if strings.Contains(line, "Ada Lovelace") && !strings.Contains(line, "●") {
			t.Fatalf("a matching row has no mark: %q", line)
		}
	}
}

func TestTypingQAndSlashInTheSearchBox(t *testing.T) {
	m := runes(ready(t), "/")
	if _, cmd := update(m, tui.Key{Type: tui.KeyRunes, Text: "q"}); isQuit(cmd) {
		t.Fatal("q quit while typing a search")
	}
}

func TestCopyCommands(t *testing.T) {
	m := find(t, ready(t), "$.status")
	m2, cmd := update(m, tui.Key{Type: tui.KeyRunes, Text: "p"})
	if cmd == nil || !strings.Contains(text(m2), "Copied $.status") {
		t.Fatalf("p: cmd=%v\n%s", cmd != nil, text(m2))
	}
	m2, cmd = update(m, tui.Key{Type: tui.KeyRunes, Text: "y"})
	if cmd == nil || !strings.Contains(text(m2), "Copied value") {
		t.Fatalf("y: cmd=%v\n%s", cmd != nil, text(m2))
	}
	m = find(t, m, "$.shop")
	if m2, _ = update(m, tui.Key{Type: tui.KeyRunes, Text: "y"}); !strings.Contains(text(m2), "Copied {4 keys}") {
		t.Fatalf("copying a branch:\n%s", text(m2))
	}
	for _, id := range []string{"copy-path", "copy-value"} {
		r := rect(t, m, id)
		if _, cmd := update(m, tui.MouseEvent{X: r.X + 1, Y: r.Y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress}); cmd == nil {
			t.Errorf("clicking %s gave no command", id)
		}
	}
}

func TestClickSelectsAndArrowToggles(t *testing.T) {
	m := ready(t)
	i := -1
	for k, r := range m.rows() {
		if r.n.path() == "$.orders" {
			i = k
		}
	}
	tr := rect(t, m, "tree")
	m = press(m, tr.X+10, rowY(t, m, i), 1)
	if m.selected().path() != "$.orders" || m.open[m.selected()] {
		t.Fatalf("a click on the text: selected %s open=%v", m.selected().path(), m.open[m.selected()])
	}
	m = press(m, tr.X+2, rowY(t, m, i), 1) // the arrow, one level in
	if !m.open[m.selected()] {
		t.Fatal("clicking the arrow did not open the branch")
	}
	m = press(m, tr.X+10, rowY(t, m, i), 2)
	if m.open[m.selected()] {
		t.Fatal("a double click did not close it")
	}
}

func TestWheelScrolls(t *testing.T) {
	m := send(runes(ready(t), "E"), tui.ResizeMsg{Width: 100, Height: 10})
	m = send(m, tui.MouseEvent{Button: tui.MouseButtonWheelDown, Action: tui.MouseActionPress})
	if m.top != 3 || m.cur < m.top {
		t.Fatalf("top=%d cur=%d after the wheel", m.top, m.cur)
	}
	m = send(m, tui.MouseEvent{Button: tui.MouseButtonWheelUp, Action: tui.MouseActionPress})
	if m.top != 0 {
		t.Fatalf("top = %d after wheel up", m.top)
	}
	m = runes(m, "G")
	if m.cur != len(m.rows())-1 || m.top != len(m.rows())-m.visRows() {
		t.Fatalf("G: cur=%d top=%d", m.cur, m.top)
	}
}

func TestMovementStaysInside(t *testing.T) {
	m := ready(t)
	for i := 0; i < 100; i++ {
		m = key(m, tui.KeyDown)
	}
	if m.cur != len(m.rows())-1 {
		t.Fatalf("cur = %d past the end", m.cur)
	}
	for i := 0; i < 100; i++ {
		m = key(m, tui.KeyUp)
	}
	if m.cur != 0 {
		t.Fatalf("cur = %d past the start", m.cur)
	}
	m = key(key(m, tui.KeyEnd), tui.KeyHome)
	if m.cur != 0 {
		t.Fatal("Home did not go to the top")
	}
}

func TestEmptyDocumentsAndScalars(t *testing.T) {
	for _, doc := range []string{`{}`, `[]`, `7`, `"x"`, `null`} {
		m := send(newModel(mustParse(t, doc), "d.json"), tui.ResizeMsg{Width: 80, Height: 24})
		m = key(key(runes(m, "l"), tui.KeyDown), tui.KeyEnter)
		if !strings.Contains(text(m), "$") {
			t.Errorf("%s: no path bar:\n%s", doc, text(m))
		}
	}
}

func TestHelpAndQuit(t *testing.T) {
	m := key(ready(t), tui.KeyF1)
	if !strings.Contains(text(m), "next and previous match") {
		t.Fatalf("\n%s", text(m))
	}
	if m = key(m, tui.KeyEsc); m.help {
		t.Fatal("Esc did not close help")
	}
	for _, k := range []tui.Key{{Type: tui.KeyCtrlC}, {Type: tui.KeyRunes, Text: "q"}} {
		if _, cmd := update(ready(t), k); !isQuit(cmd) {
			t.Errorf("%v did not quit", k)
		}
	}
}

func TestReadChoosesTheSource(t *testing.T) {
	data, src, err := read(nil, strings.NewReader("x"), false)
	if err != nil || src != "sample.json" || !strings.Contains(data, "orders") {
		t.Fatalf("no args, no pipe: %q %q %v", src, data[:10], err)
	}
	data, src, err = read(nil, strings.NewReader(`{"a":1}`), true)
	if err != nil || src != "stdin" || data != `{"a":1}` {
		t.Fatalf("piped: %q %q %v", src, data, err)
	}
	if _, src, _ = read([]string{"-"}, strings.NewReader("1"), false); src != "stdin" {
		t.Fatalf("-: %q", src)
	}
	if _, _, err = read([]string{"/no/such/file.json"}, strings.NewReader(""), false); err == nil {
		t.Fatal("a missing file did not fail")
	}
}

func TestTinyTerminalsStayInside(t *testing.T) {
	m := runes(runes(ready(t), "E"), "/")
	m = runes(m, "name")
	for _, sz := range [][2]int{{40, 10}, {20, 6}, {10, 4}, {300, 80}} {
		m = send(m, tui.ResizeMsg{Width: sz[0], Height: sz[1]})
		for i, line := range strings.Split(m.View(), "\n") {
			if w := ansi.Width(line); w > sz[0] {
				t.Fatalf("%dx%d: line %d is %d wide: %q", sz[0], sz[1], i, w, line)
			}
		}
	}
}
