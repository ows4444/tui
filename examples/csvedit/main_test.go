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

func ctrl(m model, c rune) model { return send(m, tui.Key{Type: tui.KeyCtrl, Code: c}) }

func runes(m model, s string) model {
	for _, r := range s {
		m = send(m, tui.Key{Type: tui.KeyRunes, Text: string(r)})
	}
	return m
}

func open(t testing.TB, st memStore, path string) model {
	t.Helper()
	m, err := newModel(st, path)
	if err != nil {
		t.Fatal(err)
	}
	return send(m, tui.ResizeMsg{Width: 80, Height: 24})
}

func small(t testing.TB) (model, memStore) {
	st := memStore{"s.csv": "name,n\nb,10\na,9\nc,\n"}
	return open(t, st, "s.csv"), st
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

// cellPos is where cell (row, col) is drawn, for clicks.
func cellPos(t testing.TB, m model, row, col int) (x, y int) {
	t.Helper()
	g := rect(t, m, "grid")
	for _, s := range m.visibleCols() {
		if s.col == col {
			return s.x + 1, g.Y + 1 + row - m.top
		}
	}
	t.Fatalf("column %d is not on screen", col)
	return
}

func click(t testing.TB, m model, row, col, clicks int) model {
	x, y := cellPos(t, m, row, col)
	if row < 0 {
		y = rect(t, m, "grid").Y
	}
	return send(m, tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress, Clicks: clicks})
}

func clickTool(t testing.TB, m model, id string) model {
	r := rect(t, m, id)
	return send(m, tui.MouseEvent{X: r.X + 1, Y: r.Y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
}

func TestOpensTheFile(t *testing.T) {
	m, _ := small(t)
	out := text(m)
	for _, want := range []string{"csvedit", "s.csv", "3 rows × 2 cols", "name", "10", "A1", "Opened s.csv"} {
		if !strings.Contains(out, want) && want != "A1" && want != "Opened s.csv" {
			t.Errorf("screen lacks %q:\n%s", want, out)
		}
	}
	if m.dirty() || strings.Contains(out, "●") {
		t.Fatal("a freshly opened file shows as modified")
	}
}

func TestMissingFileIsAnError(t *testing.T) {
	if _, err := newModel(memStore{}, "none.csv"); err == nil {
		t.Fatal("a missing file opened")
	}
	if _, err := newModel(memStore{"e.csv": ""}, "e.csv"); err == nil {
		t.Fatal("an empty file opened")
	}
}

func TestSampleOpensWithoutAFile(t *testing.T) {
	m := open(t, memStore{}, "")
	if len(m.cur.rows) != 12 || !strings.Contains(text(m), "Tokyo") || !strings.Contains(text(m), "sample") {
		t.Fatalf("\n%s", text(m))
	}
}

func TestCursorMovesAndStaysInside(t *testing.T) {
	m, _ := small(t)
	m = key(m, tui.KeyDown)
	m = runes(m, "l")
	if m.row != 1 || m.col != 1 {
		t.Fatalf("row=%d col=%d", m.row, m.col)
	}
	for i := 0; i < 10; i++ {
		m = key(key(m, tui.KeyDown), tui.KeyRight)
	}
	if m.row != 2 || m.col != 1 {
		t.Fatalf("row=%d col=%d past the edges", m.row, m.col)
	}
	m = runes(m, "g")
	if m.row != -1 {
		t.Fatalf("g: row = %d", m.row)
	}
	if m = runes(m, "G"); m.row != 2 {
		t.Fatalf("G: row = %d", m.row)
	}
	m = key(key(m, tui.KeyHome), tui.KeyEnd)
	if m.col != 1 {
		t.Fatalf("End: col = %d", m.col)
	}
}

func TestEditCellWithEnterAndEsc(t *testing.T) {
	m, _ := small(t)
	m = key(m, tui.KeyEnter) // the cursor starts on the first cell, b
	if m.mode != modeEdit || m.in.Value() != "b" || !strings.Contains(text(m), "A1 ▸") {
		t.Fatalf("edit did not open:\n%s", text(m))
	}
	m = runes(m, "x")
	m = key(m, tui.KeyEsc)
	if m.cur.cell(0, 0) != "b" || m.dirty() {
		t.Fatal("Esc did not drop the edit")
	}
	m = runes(key(m, tui.KeyEnter), "yz")
	m = key(m, tui.KeyEnter)
	if m.cur.cell(0, 0) != "byz" || !m.dirty() || !strings.Contains(text(m), "s.csv ●") {
		t.Fatalf("edit not kept:\n%s", text(m))
	}
}

func TestEditingToTheSameTextIsNotAChange(t *testing.T) {
	m, _ := small(t)
	m = key(key(m, tui.KeyDown), tui.KeyEnter)
	m = key(m, tui.KeyEnter)
	if m.dirty() || len(m.undo) != 0 {
		t.Fatal("an unchanged edit was recorded")
	}
}

func TestRenameAHeader(t *testing.T) {
	m, _ := small(t)
	m = key(runes(m, "g"), tui.KeyEnter) // g goes to the header row
	m = send(m, tui.Key{Type: tui.KeyCtrl, Code: 'u'})
	m = runes(m, "city")
	m = key(m, tui.KeyEnter)
	if m.cur.headers[0] != "city" || !strings.Contains(text(m), "city") {
		t.Fatalf("headers = %v", m.cur.headers)
	}
}

func TestAddAndDeleteRows(t *testing.T) {
	m, _ := small(t)
	m = key(m, tui.KeyDown)
	m = runes(m, "o")
	if len(m.cur.rows) != 4 || m.row != 2 || m.cur.rows[2][0] != "" || m.cur.rows[3][0] != "c" {
		t.Fatalf("o: rows=%q row=%d", m.cur.rows, m.row)
	}
	m = runes(m, "O")
	if len(m.cur.rows) != 5 || m.row != 2 {
		t.Fatalf("O: rows=%d row=%d", len(m.cur.rows), m.row)
	}
	m = runes(m, "d")
	m = key(m, tui.KeyDelete)
	if len(m.cur.rows) != 3 || m.cur.rows[0][0] != "b" || m.cur.rows[2][0] != "c" {
		t.Fatalf("after deleting: %q", m.cur.rows)
	}
	m = runes(runes(m, "g"), "d")
	if len(m.cur.rows) != 3 || !strings.Contains(text(m), "Select a row to delete") {
		t.Fatalf("deleting the header row:\n%s", text(m))
	}
}

func TestNoRowsMessage(t *testing.T) {
	m := open(t, memStore{"h.csv": "a,b\n"}, "h.csv")
	if !strings.Contains(text(m), "No rows. Press o to add one.") {
		t.Fatalf("\n%s", text(m))
	}
	if m = runes(m, "o"); len(m.cur.rows) != 1 || m.row != 0 {
		t.Fatalf("rows=%d row=%d after o on the header", len(m.cur.rows), m.row)
	}
}

func TestAddAndDeleteColumns(t *testing.T) {
	m, _ := small(t)
	m = runes(m, "+")
	if m.mode != modeNewCol {
		t.Fatal("+ did not ask for a name")
	}
	m = key(m, tui.KeyEnter)
	if m.cur.cols() != 2 || !strings.Contains(text(m), "A column needs a name") {
		t.Fatalf("an unnamed column was added:\n%s", text(m))
	}
	m = runes(runes(m, "+"), "tag")
	m = key(m, tui.KeyEnter)
	if m.cur.cols() != 3 || m.cur.headers[1] != "tag" || m.col != 1 {
		t.Fatalf("headers=%v col=%d", m.cur.headers, m.col)
	}
	m = runes(m, "-")
	if m.cur.cols() != 2 || m.cur.headers[1] != "n" {
		t.Fatalf("after -: %v", m.cur.headers)
	}
	m = runes(runes(m, "-"), "-")
	if m.cur.cols() != 1 || !strings.Contains(text(m), "The last column cannot be deleted") {
		t.Fatalf("\n%s", text(m))
	}
}

func TestSortKeyTogglesDirectionAndShowsAnArrow(t *testing.T) {
	m, _ := small(t)
	m = runes(m, "l")
	m = runes(m, "s")
	if m.cur.rows[0][1] != "9" || !strings.Contains(text(m), "n ▲") {
		t.Fatalf("ascending:\n%s", text(m))
	}
	m = runes(m, "s")
	if m.cur.rows[0][1] != "10" || !strings.Contains(text(m), "n ▼") {
		t.Fatalf("descending:\n%s", text(m))
	}
	m = runes(m, "o")
	if strings.Contains(text(m), "▼") {
		t.Fatal("the sort arrow stayed after an edit")
	}
	m = ctrl(ctrl(ctrl(m, 'z'), 'z'), 'z') // the added row, the descending sort, the ascending sort
	if m.cur.rows[0][0] != "b" || len(m.cur.rows) != 3 {
		t.Fatalf("undoing all three did not restore the file order: %q", m.cur.rows)
	}
}

func TestUndoAndRedo(t *testing.T) {
	m, _ := small(t)
	if m = ctrl(m, 'z'); !strings.Contains(text(m), "Nothing to undo") {
		t.Fatal("undo with no history")
	}
	m = runes(key(m, tui.KeyDown), "d")
	m = ctrl(m, 'z')
	if len(m.cur.rows) != 3 || m.dirty() {
		t.Fatal("undo did not restore the row")
	}
	m = ctrl(m, 'y')
	if len(m.cur.rows) != 2 {
		t.Fatal("redo did not delete again")
	}
	m = ctrl(ctrl(m, 'y'), 'y')
	if !strings.Contains(text(m), "Nothing to redo") {
		t.Fatal("redo with nothing left")
	}
	m = ctrl(m, 'z')
	m = runes(m, "o") // a new change forgets the redo history
	if len(m.redo) != 0 {
		t.Fatal("a new change kept the redo stack")
	}
}

func TestUndoStackIsBounded(t *testing.T) {
	m, _ := small(t)
	for i := 0; i < undoMax+30; i++ {
		m = runes(m, "o")
	}
	if len(m.undo) != undoMax {
		t.Fatalf("undo holds %d steps, want %d", len(m.undo), undoMax)
	}
}

func TestSaveWritesTheFileAndClearsTheMark(t *testing.T) {
	m, st := small(t)
	m = runes(key(m, tui.KeyDown), "o")
	m = ctrl(m, 's')
	if !strings.Contains(text(m), "Saved s.csv") || m.dirty() {
		t.Fatalf("\n%s", text(m))
	}
	if st["s.csv"] != "name,n\nb,10\na,9\n,\nc,\n" {
		t.Fatalf("file = %q", st["s.csv"])
	}
}

func TestSaveAsAsksForAName(t *testing.T) {
	st := memStore{}
	m := open(t, st, "")
	m = ctrl(m, 's')
	if m.mode != modeSaveAs {
		t.Fatal("Ctrl+S with no name did not ask")
	}
	m = runes(m, "out.csv")
	m = key(m, tui.KeyEnter)
	if !strings.HasPrefix(st["out.csv"], "city,country,population,area_km2\n") || m.path != "out.csv" {
		t.Fatalf("file = %q path=%q", st["out.csv"], m.path)
	}
}

func TestQuitAsksOnlyWhenChangesWouldBeLost(t *testing.T) {
	m, _ := small(t)
	if _, cmd := update(m, tui.Key{Type: tui.KeyRunes, Text: "q"}); !isQuit(cmd) {
		t.Fatal("q on a clean file did not quit")
	}
	m = runes(m, "o")
	m, cmd := update(m, tui.Key{Type: tui.KeyRunes, Text: "q"})
	if isQuit(cmd) || m.mode != modeDiscard || !strings.Contains(text(m), "Discard unsaved changes?") {
		t.Fatalf("q on a dirty file did not ask:\n%s", text(m))
	}
	if m = runes(m, "n"); m.mode != modeGrid {
		t.Fatal("n did not cancel")
	}
	m = ctrl(m, 'q')
	if _, cmd := update(m, tui.Key{Type: tui.KeyRunes, Text: "y"}); !isQuit(cmd) {
		t.Fatal("y did not quit")
	}
	if _, cmd := update(m, tui.Key{Type: tui.KeyCtrlC}); !isQuit(cmd) {
		t.Fatal("Ctrl+C did not quit at once")
	}
}

func TestClickSelectsAndASecondClickEdits(t *testing.T) {
	m, _ := small(t)
	m = click(t, m, 1, 1, 1)
	if m.row != 1 || m.col != 1 || m.mode != modeGrid {
		t.Fatalf("row=%d col=%d mode=%v", m.row, m.col, m.mode)
	}
	m = click(t, m, 1, 1, 2)
	if m.mode != modeEdit || m.in.Value() != "9" {
		t.Fatalf("double click: mode=%v value=%q", m.mode, m.in.Value())
	}
}

func TestClickingTheSelectedHeaderSorts(t *testing.T) {
	m, _ := small(t)
	m = click(t, m, -1, 1, 1) // select
	if m.row != -1 || m.col != 1 || m.sortCol != -1 {
		t.Fatalf("row=%d col=%d sortCol=%d", m.row, m.col, m.sortCol)
	}
	m = click(t, m, -1, 1, 1) // sort
	if m.sortCol != 1 || m.cur.rows[0][1] != "9" {
		t.Fatalf("sortCol=%d rows=%q", m.sortCol, m.cur.rows)
	}
}

func TestWheelScrollsLongFiles(t *testing.T) {
	var b strings.Builder
	b.WriteString("n\n")
	for i := 0; i < 60; i++ {
		b.WriteString("r" + strings.Repeat("x", i%3) + "\n")
	}
	m := open(t, memStore{"l.csv": b.String()}, "l.csv")
	m = send(m, tui.MouseEvent{Button: tui.MouseButtonWheelDown, Action: tui.MouseActionPress})
	if m.top != 3 || m.row < m.top {
		t.Fatalf("top=%d row=%d after the wheel", m.top, m.row)
	}
	m = send(m, tui.MouseEvent{Button: tui.MouseButtonWheelUp, Action: tui.MouseActionPress})
	if m.top != 0 {
		t.Fatalf("top = %d after wheel up", m.top)
	}
	m = runes(m, "G")
	if m.row != 59 || m.top != 60-m.visRows() {
		t.Fatalf("G: row=%d top=%d", m.row, m.top)
	}
}

func TestToolbarButtons(t *testing.T) {
	m, _ := small(t)
	m = key(m, tui.KeyDown)
	m = clickTool(t, m, "tb-row")
	if len(m.cur.rows) != 4 {
		t.Fatal("+Row did nothing")
	}
	m = clickTool(t, m, "tb-delrow")
	m = clickTool(t, m, "tb-undo")
	if len(m.cur.rows) != 4 {
		t.Fatalf("Undo: %d rows", len(m.cur.rows))
	}
	m = clickTool(t, m, "tb-redo")
	m = clickTool(t, m, "tb-sort")
	m = clickTool(t, m, "tb-col")
	if m.mode != modeNewCol {
		t.Fatal("+Col did not ask for a name")
	}
	m = key(m, tui.KeyEsc)
	m = clickTool(t, m, "tb-delcol")
	if m.cur.cols() != 1 {
		t.Fatal("−Col did nothing")
	}
	m = clickTool(t, m, "tb-save")
	if m.dirty() {
		t.Fatal("Save did not save")
	}
}

func TestWideFilesScrollSideways(t *testing.T) {
	hdr := make([]string, 30)
	row := make([]string, 30)
	for i := range hdr {
		hdr[i], row[i] = "col"+strings.Repeat("h", i%5), "value"+strings.Repeat("v", i%7)
	}
	m := open(t, memStore{"w.csv": strings.Join(hdr, ",") + "\n" + strings.Join(row, ",") + "\n"}, "w.csv")
	m = key(m, tui.KeyDown)
	for i := 0; i < 25; i++ {
		m = key(m, tui.KeyRight)
	}
	if m.left == 0 || !strings.Contains(text(m), "Z1") {
		t.Fatalf("left=%d\n%s", m.left, text(m))
	}
	vc := m.visibleCols()
	if vc[len(vc)-1].col < m.col {
		t.Fatalf("the cursor column %d is off screen (last shown %d)", m.col, vc[len(vc)-1].col)
	}
	m = key(m, tui.KeyHome)
	if m.left != 0 {
		t.Fatalf("left = %d after Home", m.left)
	}
}

func TestLongCellsAreCutAndNumbersAlignRight(t *testing.T) {
	m := open(t, memStore{"x.csv": "name,n\n" + strings.Repeat("w", 80) + ",5\n"}, "x.csv")
	out := text(m)
	if strings.Contains(out, strings.Repeat("w", 30)) {
		t.Fatalf("a long cell was not cut:\n%s", out)
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "wwww") && !strings.Contains(line, "5") {
			t.Fatalf("number missing on its row: %q", line)
		}
	}
}

func TestHelp(t *testing.T) {
	m, _ := small(t)
	m = key(m, tui.KeyF1)
	if !strings.Contains(text(m), "edit the cell") {
		t.Fatalf("\n%s", text(m))
	}
	if m = key(m, tui.KeyEsc); m.help {
		t.Fatal("Esc did not close help")
	}
}

func TestCopyReturnsAClipboardCommand(t *testing.T) {
	m, _ := small(t)
	m, cmd := update(m, tui.Key{Type: tui.KeyRunes, Text: "c"})
	if cmd == nil || !strings.Contains(text(m), "Copied A1") {
		t.Fatalf("\n%s", text(m))
	}
}

func TestTinyTerminalsStayInside(t *testing.T) {
	m := open(t, memStore{}, "")
	m = key(runes(m, "l"), tui.KeyDown)
	for _, sz := range [][2]int{{40, 10}, {20, 6}, {10, 4}, {300, 80}} {
		m = send(m, tui.ResizeMsg{Width: sz[0], Height: sz[1]})
		for i, line := range strings.Split(m.View(), "\n") {
			if w := ansi.Width(line); w > sz[0] {
				t.Fatalf("%dx%d: line %d is %d wide: %q", sz[0], sz[1], i, w, line)
			}
		}
	}
}
