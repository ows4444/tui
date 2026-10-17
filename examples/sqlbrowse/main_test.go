package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/input"
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

func ctrl(m model, c rune) model { return send(m, tui.Key{Type: tui.KeyCtrl, Code: c}) }

func runes(m model, s string) model {
	for _, r := range s {
		m = send(m, tui.Key{Type: tui.KeyRunes, Text: string(r)})
	}
	return m
}

func ready() model { return send(freshModel(), tui.ResizeMsg{Width: 110, Height: 28}) }

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

func press(m model, x, y int) (model, tui.Cmd) {
	return update(m, tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
}

func click(t testing.TB, m model, id string) (model, tui.Cmd) {
	r := rect(t, m, id)
	return press(m, r.X+1, r.Y)
}

// query types q into the box and runs it.
func query(m model, q string) model {
	m = ctrl(m, 'l')
	m = runes(m, q)
	return key(m, tui.KeyEnter)
}

func TestOpensWithTablesAndAFirstResult(t *testing.T) {
	m := ready()
	out := text(m)
	for _, w := range []string{"sqlbrowse", "read-only", "Tables", "customers 30", "orders 80", "products 15", "order_items", "SQL ▸ SELECT * FROM customers LIMIT 20",
		"✓ 20 rows × 6 columns", "Ada Lovelace", "Run", "History", "Export", "id int", "vip bool"} {
		if !strings.Contains(out, w) {
			t.Errorf("screen lacks %q:\n%s", w, out)
		}
	}
}

func TestRunningAQueryShowsItsResult(t *testing.T) {
	m := query(ready(), "SELECT status, COUNT(*) AS n FROM orders GROUP BY status ORDER BY n DESC")
	out := text(m)
	if m.err != nil || m.res == nil || !strings.Contains(out, "status") || !strings.Contains(out, "paid") || !strings.Contains(out, "✓ ") {
		t.Fatalf("err=%v\n%s", m.err, out)
	}
	if m.res.cols[1] != "n" {
		t.Fatalf("cols = %v", m.res.cols)
	}
}

func TestAnErrorIsShownWithItsColumn(t *testing.T) {
	m := query(ready(), "SELECT nope FROM customers")
	out := text(m)
	if m.err == nil || !strings.Contains(out, "✗ There is no column called nope (at column 8)") {
		t.Fatalf("\n%s", out)
	}
	if m.res != nil {
		t.Fatal("a failed query kept the old result on screen")
	}
	m = query(m, "DROP TABLE customers")
	if !strings.Contains(text(m), "only runs SELECT") {
		t.Fatalf("\n%s", text(m))
	}
	if len(m.db.tables) != 4 {
		t.Fatal("a table was dropped")
	}
	m = query(m, "SELECT 'é' FROM customers WHERE id @ 1")
	if !strings.Contains(text(m), "(at column 36)") {
		t.Fatalf("the column is counted in bytes, not characters:\n%s", text(m))
	}
}

func TestEmptyResultIsExplained(t *testing.T) {
	m := query(ready(), "SELECT * FROM customers WHERE id > 1000")
	if !strings.Contains(text(m), "(no rows)") || !strings.Contains(text(m), "0 rows × 6 columns") {
		t.Fatalf("\n%s", text(m))
	}
}

func TestBlankQuery(t *testing.T) {
	m := ctrl(ready(), 'l')
	m = key(m, tui.KeyEnter)
	if m.msg != "Type a query first" {
		t.Fatalf("msg = %q", m.msg)
	}
}

func TestNullsAreShownAndNumbersAlignRight(t *testing.T) {
	m := query(ready(), "SELECT id, status FROM orders WHERE status IS NULL")
	if !strings.Contains(text(m), "NULL") {
		t.Fatalf("\n%s", text(m))
	}
	m = query(m, "SELECT id, total FROM orders ORDER BY id LIMIT 3")
	lines := strings.Split(text(m), "\n")
	headEnd, valEnd := -1, -1
	for _, l := range lines {
		l = strings.TrimRight(l[strings.Index(l, "│")+len("│"):], " ")
		switch {
		case strings.HasSuffix(l, "total"):
			headEnd = len([]rune(l))
		case strings.HasPrefix(strings.TrimSpace(l), "2 "):
			valEnd = len([]rune(l))
		}
	}
	if headEnd < 0 || headEnd != valEnd+1-1 && headEnd != valEnd {
		t.Fatalf("a number does not end under its heading (%d vs %d):\n%s", headEnd, valEnd, text(m))
	}
}

func TestHistoryKeepsDistinctQueriesAndRecallsWithUpDown(t *testing.T) {
	m := ready()
	m = query(m, "SELECT 1 FROM customers LIMIT 1")
	m = query(m, "SELECT 2 FROM customers LIMIT 1")
	m = query(m, "SELECT 2 FROM customers LIMIT 1") // again: not stored twice
	if got := len(m.history); got != 3 {
		t.Fatalf("history = %q", m.history)
	}
	m = key(m, tui.KeyUp)
	if m.in.Value() != "SELECT 2 FROM customers LIMIT 1" {
		t.Fatalf("Up recalled %q", m.in.Value())
	}
	m = key(m, tui.KeyUp)
	if m.in.Value() != "SELECT 1 FROM customers LIMIT 1" {
		t.Fatalf("Up again recalled %q", m.in.Value())
	}
}

func TestHistoryListNewestFirstAndChoose(t *testing.T) {
	m := query(query(ready(), "SELECT 1 FROM customers LIMIT 1"), "SELECT 2 FROM customers LIMIT 1")
	m = ctrl(m, 'r')
	if m.pick != pickHistory || !strings.Contains(text(m), "History, newest first") {
		t.Fatalf("\n%s", text(m))
	}
	items := m.items()
	if items[0] != "SELECT 2 FROM customers LIMIT 1" || items[2] != "SELECT * FROM customers LIMIT 20" {
		t.Fatalf("items = %q", items)
	}
	m = key(key(m, tui.KeyDown), tui.KeyEnter)
	if m.pick != noPick || m.in.Value() != "SELECT 1 FROM customers LIMIT 1" || m.focus != onQuery {
		t.Fatalf("pick=%v value=%q", m.pick, m.in.Value())
	}
	if m = ctrl(m, 'r'); key(m, tui.KeyEsc).pick != noPick {
		t.Fatal("Esc did not close the list")
	}
}

func TestHistoryIsBounded(t *testing.T) {
	m := ready()
	for i := 0; i < maxHistory+20; i++ {
		m = m.run("SELECT " + strings.Repeat("1", i%7+1) + " FROM customers LIMIT " + string(rune('0'+i%10)) + strings.Repeat(" ", i%3) + "--" + strings.Repeat("x", i))
	}
	if len(m.history) > maxHistory {
		t.Fatalf("%d queries kept", len(m.history))
	}
}

func TestTablesListOpensATable(t *testing.T) {
	m := ctrl(ready(), 't')
	if m.pick != pickTables || !strings.Contains(text(m), "orders  (80 rows)") {
		t.Fatalf("\n%s", text(m))
	}
	m = key(key(m, tui.KeyDown), tui.KeyEnter)
	if m.in.Value() != "SELECT * FROM orders LIMIT 100" || m.res == nil || len(m.res.rows) != 80 || m.tCur != 1 {
		t.Fatalf("value=%q rows=%v tCur=%d", m.in.Value(), m.res != nil, m.tCur)
	}
}

func TestSidebarClickOpensATableAndShowsItsColumns(t *testing.T) {
	m := ready()
	m, _ = click(t, m, "tbl-2")
	if m.in.Value() != "SELECT * FROM products LIMIT 100" || !strings.Contains(text(m), "price real") || m.focus != onTables {
		t.Fatalf("value=%q focus=%v\n%s", m.in.Value(), m.focus, text(m))
	}
	m = key(m, tui.KeyUp)
	m = key(m, tui.KeyEnter)
	if m.tCur != 1 || !strings.Contains(m.in.Value(), "orders") {
		t.Fatalf("tCur=%d value=%q", m.tCur, m.in.Value())
	}
	m = runes(m, "j")
	m = runes(m, "k")
	if m.tCur != 1 {
		t.Fatalf("tCur = %d", m.tCur)
	}
}

func TestTabCyclesPanesAndNarrowSkipsTheTables(t *testing.T) {
	m := ready()
	for _, want := range []focus{onGrid, onTables, onQuery} {
		if m = key(m, tui.KeyTab); m.focus != want {
			t.Fatalf("Tab went to %v, want %v", m.focus, want)
		}
	}
	m = send(m, tui.Key{Type: tui.KeyTab, Mod: input.ModShift})
	if m.focus != onTables {
		t.Fatalf("Shift+Tab went to %v", m.focus)
	}
	n := send(freshModel(), tui.ResizeMsg{Width: 60, Height: 20})
	if strings.Contains(text(n), "Tables\n") && strings.Contains(text(n), "customers 30") {
		t.Fatalf("sidebar at 60 columns:\n%s", text(n))
	}
	for _, want := range []focus{onGrid, onQuery} {
		if n = key(n, tui.KeyTab); n.focus != want {
			t.Fatalf("narrow Tab went to %v, want %v", n.focus, want)
		}
	}
	if !strings.Contains(text(n), "Tables") {
		t.Fatal("no Tables button when the sidebar is hidden")
	}
}

func TestGridNavigationAndCopy(t *testing.T) {
	m := key(ready(), tui.KeyTab)
	m = key(key(m, tui.KeyDown), tui.KeyRight)
	if m.gr != 1 || m.gc != 1 {
		t.Fatalf("gr=%d gc=%d", m.gr, m.gc)
	}
	m, cmd := update(m, tui.Key{Type: tui.KeyRunes, Text: "y"})
	if cmd == nil || !strings.Contains(m.msg, `Copied "Alan Turing"`) {
		t.Fatalf("msg = %q", m.msg)
	}
	m = runes(runes(m, "G"), "g")
	if m.gr != 0 {
		t.Fatalf("gr = %d", m.gr)
	}
	m = runes(m, "G")
	if m.gr != 19 {
		t.Fatalf("G: gr = %d", m.gr)
	}
	m = key(m, tui.KeyEnd)
	if m.gc != 5 {
		t.Fatalf("End: gc = %d", m.gc)
	}
	m = key(m, tui.KeyHome)
	if m.gc != 0 {
		t.Fatalf("Home: gc = %d", m.gc)
	}
	for i := 0; i < 50; i++ {
		m = key(m, tui.KeyDown)
	}
	if m.gr != 19 {
		t.Fatalf("gr = %d past the end", m.gr)
	}
	if m = key(m, tui.KeyEnter); m.focus != onQuery {
		t.Fatal("Enter in the grid should go back to the query")
	}
}

func TestWideResultsScrollSideways(t *testing.T) {
	m := send(freshModel(), tui.ResizeMsg{Width: 60, Height: 20})
	m = query(m, "SELECT * FROM customers")
	m = key(m, tui.KeyTab)
	for i := 0; i < 5; i++ {
		m = key(m, tui.KeyRight)
	}
	if m.gleft == 0 || !strings.Contains(text(m), "vip") {
		t.Fatalf("gleft=%d\n%s", m.gleft, text(m))
	}
	spans := m.spans()
	if spans[len(spans)-1].col < m.gc {
		t.Fatalf("the selected column %d is off screen", m.gc)
	}
}

func TestClickAndWheelInTheGrid(t *testing.T) {
	m := ready()
	g := rect(t, m, "grid")
	sp := m.spans()
	m, _ = press(m, g.X+sp[1].x+2, g.Y+3)
	if m.gr != 2 || m.gc != 1 || m.focus != onGrid {
		t.Fatalf("gr=%d gc=%d focus=%v", m.gr, m.gc, m.focus)
	}
	m = send(m, tui.MouseEvent{X: g.X + 5, Y: g.Y + 3, Button: tui.MouseButtonWheelDown, Action: tui.MouseActionPress})
	if m.gr != 5 {
		t.Fatalf("gr = %d after the wheel", m.gr)
	}
	if m, _ = press(m, g.X+sp[0].x, g.Y); m.gr != 5 {
		t.Fatal("clicking the heading moved the cursor")
	}
}

func TestExportCopiesCSVAndWritesOnlyWithOut(t *testing.T) {
	m := query(ready(), "SELECT id, name, vip FROM customers WHERE id <= 2")
	m2, cmd := update(m, tui.Key{Type: tui.KeyCtrl, Code: 'e'})
	if cmd == nil || !strings.Contains(m2.msg, "Copied 2 rows as CSV") {
		t.Fatalf("msg = %q", m2.msg)
	}
	csv := m.res.csvOf()
	if !strings.HasPrefix(csv, "id,name,vip\n1,Ada Lovelace,") || strings.Count(csv, "\n") != 3 {
		t.Fatalf("csv = %q", csv)
	}
	path := filepath.Join(t.TempDir(), "r.csv")
	m.out = path
	_, cmd = update(m, tui.Key{Type: tui.KeyCtrl, Code: 'e'})
	if cmd == nil {
		t.Fatal("no command")
	}
	writeFile(path, csv)()
	if b, err := os.ReadFile(path); err != nil || string(b) != csv {
		t.Fatalf("file = %q err %v", b, err)
	}
	e := query(ready(), "SELECT nope FROM customers")
	if e, cmd = update(e, tui.Key{Type: tui.KeyCtrl, Code: 'e'}); cmd != nil || !strings.Contains(e.msg, "Run a query that works first") {
		t.Fatalf("msg = %q", e.msg)
	}
}

func TestCSVQuotesAndNulls(t *testing.T) {
	r := &result{cols: []string{"a", "b,c"}, rows: [][]value{{"x,y", nil}, {"say \"hi\"", 2.5}, {"line\nbreak", true}}}
	want := "a,\"b,c\"\n\"x,y\",\n\"say \"\"hi\"\"\",2.5\n\"line\nbreak\",true\n"
	if got := r.csvOf(); got != want {
		t.Fatalf("csv = %q, want %q", got, want)
	}
}

func TestButtonsRunAndOpenLists(t *testing.T) {
	m := runes(ctrl(ready(), 'l'), "SELECT COUNT(*) FROM products")
	m, _ = click(t, m, "run")
	if m.res == nil || format(m.res.rows[0][0]) != "15" {
		t.Fatalf("Run did not run: %v %v", m.res, m.err)
	}
	m, _ = click(t, m, "hist")
	if m.pick != pickHistory {
		t.Fatal("History button")
	}
	r := rect(t, m, "ovl-0")
	m, _ = press(m, r.X+2, r.Y)
	if m.pick != noPick || m.in.Value() != "SELECT COUNT(*) FROM products" {
		t.Fatalf("choosing from the list: pick=%v value=%q", m.pick, m.in.Value())
	}
	m, _ = click(t, m, "hist")
	m, _ = press(m, 2, 2) // a click outside the list closes it
	if m.pick != noPick {
		t.Fatal("a click elsewhere should close the list")
	}
	m, cmd := click(t, m, "export")
	if cmd == nil {
		t.Fatal("Export button gave no command")
	}
	m, _ = click(t, m, "query")
	if m.focus != onQuery {
		t.Fatal("clicking the query did not focus it")
	}
}

func TestF5RunsFromAnywhere(t *testing.T) {
	m := key(ctrl(ready(), 'l'), tui.KeyTab)
	m.in.SetValue("SELECT COUNT(*) FROM orders")
	m = key(m, tui.KeyF5)
	if m.res == nil || format(m.res.rows[0][0]) != "80" {
		t.Fatalf("F5: %v %v", m.res, m.err)
	}
}

func TestQAndSlashBehaviour(t *testing.T) {
	m := runes(ready(), "q")
	if m.in.Value() != "SELECT * FROM customers LIMIT 20q" {
		t.Fatalf("q was not typed: %q", m.in.Value())
	}
	g := key(ready(), tui.KeyTab)
	if _, cmd := update(g, tui.Key{Type: tui.KeyRunes, Text: "q"}); !isQuit(cmd) {
		t.Fatal("q in the result did not quit")
	}
	if g = runes(g, "i"); g.focus != onQuery {
		t.Fatal("i should return to the query")
	}
}

func TestLongRunsAreTimed(t *testing.T) {
	m := ready()
	if !strings.Contains(text(m), "in 0s") {
		t.Fatalf("a still clock should show 0s:\n%s", text(m))
	}
}

func TestResultIsCappedAtAScreenfulOfThousands(t *testing.T) {
	big := &database{tables: []*table{{name: "n", cols: []string{"i"}}}}
	for i := 0; i < maxShown+50; i++ {
		big.tables[0].rows = append(big.tables[0].rows, []value{float64(i)})
	}
	m := send(newModel(big, "", stillClock), tui.ResizeMsg{Width: 100, Height: 24})
	m = query(m, "SELECT * FROM n")
	if len(m.res.rows) != maxShown || !m.truncated || !strings.Contains(text(m), "first 10000 shown") {
		t.Fatalf("rows=%d truncated=%v", len(m.res.rows), m.truncated)
	}
}

func TestHelpAndQuit(t *testing.T) {
	m := key(ready(), tui.KeyF1)
	if !strings.Contains(text(m), "recall earlier queries") {
		t.Fatalf("\n%s", text(m))
	}
	if m = key(m, tui.KeyEsc); m.help {
		t.Fatal("Esc did not close help")
	}
	if _, cmd := update(ready(), tui.Key{Type: tui.KeyCtrlC}); !isQuit(cmd) {
		t.Fatal("Ctrl+C did not quit")
	}
}

func TestTinyTerminalsStayInside(t *testing.T) {
	m := query(ready(), "SELECT * FROM orders")
	m = ctrl(m, 't')
	for _, sz := range [][2]int{{40, 10}, {20, 6}, {10, 4}, {300, 80}, {72, 24}, {71, 24}, {3, 3}} {
		m = send(m, tui.ResizeMsg{Width: sz[0], Height: sz[1]})
		for _, st := range []int{0, 1, 2} {
			m.pick = pick(st)
			for i, line := range strings.Split(m.View(), "\n") {
				if w := ansi.Width(line); w > sz[0] {
					t.Fatalf("%dx%d pick %d: line %d is %d wide: %q", sz[0], sz[1], st, i, w, line)
				}
			}
		}
	}
}
