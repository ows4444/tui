package main

import (
	"context"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/faces"
)

// The default (small) sprite's size in cells; wallCell and the click
// tables below are written for it.
var smallW, smallH = faces.Small.Cells()

// spriteTop is the row the sprite starts on: the title line and a blank line.
// The click regions come from the layout; the tests below name rows in terms
// of it.
const spriteTop = 2

// rowOf returns row r of a frame.
func rowOf(frame string, r int) string { return strings.Split(frame, "\n")[r] }

// differingRow returns the first row where two frames differ, and that row
// of the second: something that is only on screen while it is showing.
func differingRow(a, b string) string {
	for r := range strings.Split(a, "\n") {
		if rowOf(a, r) != rowOf(b, r) {
			return rowOf(b, r)
		}
	}
	return ""
}

func key(r rune) tui.Key { return tui.Key{Type: tui.KeyRunes, Text: string([]rune{r})} }

func send(m model, msg tui.Msg) (model, tui.Cmd) {
	next, cmd := m.Update(msg)
	return next.(model), cmd
}

func press(x, y int) tui.MouseEvent {
	return tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress}
}

// wallCell returns a click position inside the wall cell at (row, col).
func wallCell(row, col int) (x, y int) {
	return col*(smallW+wallGap) + 3, spriteTop + row*(smallH+2) + 2
}

func wallModel(t *testing.T) model {
	t.Helper()
	m, _ := send(initialModel(), key('w'))
	return m
}

func TestFacesRestUntilClicked(t *testing.T) {
	m := initialModel()
	if m.Init() != nil {
		t.Error("Init should not start any animation")
	}
	if m.player.Running() {
		t.Error("the player should rest before a click")
	}
	if got := ansi.StripANSI(m.View()); !strings.Contains(got, rowOf(faces.Get(0).Frames(faces.Small)[0], 1)) {
		t.Errorf("view should show the first frame at rest:\n%s", got)
	}
}

func TestClickingTheSpritePlaysOneLoopThenRests(t *testing.T) {
	m := initialModel()
	m.player.Set(faces.Count() - 1) // fastest-loop face is not needed; any works
	m, cmd := send(m, press(5, spriteTop+3))
	if cmd == nil || !m.player.Running() {
		t.Fatal("clicking the sprite should start the animation")
	}
	frames := 1
	for cmd != nil {
		m, cmd = send(m, tui.RunCmd(context.Background(), cmd))
		frames++
		if frames > 20 {
			t.Fatal("the single loop never ended")
		}
	}
	if want := len(m.player.Face().Frames(m.player.Size)); frames != want+0 && frames != want+1 {
		t.Errorf("played %d steps for a %d-frame face", frames, want)
	}
	if m.player.Running() || m.player.Frame() != 0 {
		t.Errorf("after the loop: running %v frame %d; want resting on frame 0", m.player.Running(), m.player.Frame())
	}
}

func TestClicksOutsideTheSpriteOrWithOtherButtonsDoNothing(t *testing.T) {
	m := initialModel()
	for _, ev := range []tui.MouseEvent{
		press(smallW, spriteTop),   // just right of the sprite
		press(-1, spriteTop),       // off the left edge
		press(3, spriteTop-1),      // the blank line above
		press(3, spriteTop+smallH), // the label line below
		{X: 3, Y: spriteTop + 1, Button: tui.MouseButtonRight, Action: tui.MouseActionPress},
		{X: 3, Y: spriteTop + 1, Button: tui.MouseButtonLeft, Action: tui.MouseActionRelease},
		{X: 3, Y: spriteTop + 1, Button: tui.MouseButtonWheelUp, Action: tui.MouseActionPress},
	} {
		if next, cmd := send(m, ev); cmd != nil || next.player.Running() {
			t.Errorf("%+v should not play anything", ev)
		}
	}
}

func TestClickingAWallFacePlaysThatFaceOnly(t *testing.T) {
	m := wallModel(t)
	x, y := wallCell(1, 2) // second row, third column: face 1*4+2
	m, cmd := send(m, press(x, y))
	i := m.page*m.perPage() + 1*m.cols() + 2
	if cmd == nil {
		t.Fatal("clicking a wall face should start a ticker")
	}
	if len(m.plays) != 1 || m.plays[i].frame != 0 {
		t.Fatalf("plays = %v, want only face %d at frame 0", m.plays, i)
	}
	msg, ok := tui.RunCmd(context.Background(), cmd).(playMsg)
	if !ok || msg.face != i {
		t.Fatalf("ticker produced %#v, want a playMsg for face %d", msg, i)
	}
	m, cmd = send(m, msg)
	if m.plays[i].frame != 1 || cmd == nil {
		t.Errorf("first tick: frame %d reschedule %v; want 1 and true", m.plays[i].frame, cmd != nil)
	}
	f := faces.Get(i).Frames(faces.Small)
	if row := differingRow(f[0], f[1]); row == "" || !strings.Contains(m.wallView(), row) {
		t.Error("the wall should draw the playing face's second frame")
	}
}

func TestWallPlayEndsAndFaceReturnsToRest(t *testing.T) {
	m := wallModel(t)
	x, y := wallCell(0, 0)
	m, _ = send(m, press(x, y))
	n := len(faces.Get(0).Frames(faces.Small))
	var cmd tui.Cmd
	for i := 0; i < n; i++ {
		m, cmd = send(m, playMsg{face: 0, gen: m.plays[0].gen})
		if i < n-1 && (cmd == nil || m.plays[0].frame != i+1) {
			t.Fatalf("tick %d: frame %d, reschedule %v", i+1, m.plays[0].frame, cmd != nil)
		}
	}
	if _, playing := m.plays[0]; playing || cmd != nil {
		t.Errorf("after the last frame face 0 should rest and stop ticking (playing=%v cmd=%v)", playing, cmd != nil)
	}
}

func TestClickingAgainRestartsAndDropsTheOldTicker(t *testing.T) {
	m := wallModel(t)
	x, y := wallCell(0, 1)
	m, _ = send(m, press(x, y))
	old := playMsg{face: 1, gen: m.plays[1].gen}
	m, _ = send(m, old)
	m, _ = send(m, old.withGen(m.plays[1].gen))
	if m.plays[1].frame != 2 {
		t.Fatalf("setup: frame %d, want 2", m.plays[1].frame)
	}

	m, _ = send(m, press(x, y)) // click again mid-loop
	if m.plays[1].frame != 0 {
		t.Errorf("a second click should restart at frame 0, got %d", m.plays[1].frame)
	}
	if next, cmd := send(m, old); next.plays[1].frame != 0 || cmd != nil {
		t.Errorf("the first click's ticker advanced the restarted face (frame %d)", next.plays[1].frame)
	}
}

func (p playMsg) withGen(g int) playMsg { p.gen = g; return p }

func TestWallClicksBetweenCellsAndOnLabelsMiss(t *testing.T) {
	m := wallModel(t)
	for _, c := range [][2]int{
		{smallW, spriteTop + 2},                           // the gap between columns
		{smallW + 1, spriteTop + 2},                       // still the gap
		{3, spriteTop + smallH},                           // the label line
		{3, spriteTop + smallH + 1},                       // the blank line between rows
		{3, spriteTop - 1},                                // above the wall
		{maxWallCols*(smallW+wallGap) + 3, spriteTop + 2}, // right of the last column
		{3, spriteTop + wallRows*(smallH+2) + 1},          // below the last row
	} {
		if next, cmd := send(m, press(c[0], c[1])); cmd != nil || len(next.plays) != 0 {
			t.Errorf("click at %v should miss every face", c)
		}
	}
}

func TestClickPastTheLastFaceOnAPartialPageMisses(t *testing.T) {
	m := wallModel(t)
	m.page = m.pages() - 1 // 50 faces / 8 per page: this page holds two
	x, y := wallCell(1, 3)
	if next, cmd := send(m, press(x, y)); cmd != nil || len(next.plays) != 0 {
		t.Error("a click on an empty cell should do nothing")
	}
	x, y = wallCell(0, 1)
	if next, cmd := send(m, press(x, y)); cmd == nil || len(next.plays) != 1 {
		t.Error("a click on the page's last face should play it")
	}
}

func TestSpaceLoopsInSingleViewAndPlaysThePageOnTheWall(t *testing.T) {
	m := initialModel()
	m, cmd := send(m, tui.Key{Type: tui.KeySpace})
	if !m.player.Running() || cmd == nil {
		t.Fatal("space should start looping")
	}
	m, cmd = send(m, tui.Key{Type: tui.KeySpace})
	if m.player.Running() || cmd != nil {
		t.Error("space again should stop the loop")
	}

	m = wallModel(t)
	m, cmd = send(m, tui.Key{Type: tui.KeySpace})
	if cmd == nil || len(m.plays) != m.perPage() {
		t.Errorf("space on the wall should play all %d faces, got %d", m.perPage(), len(m.plays))
	}
}

func TestReducedMotionPlaysNothing(t *testing.T) {
	t.Setenv("NO_ANIMATION", "1")
	m := initialModel()
	if _, cmd := send(m, press(3, spriteTop+1)); cmd != nil {
		t.Error("a click should not animate under NO_ANIMATION")
	}
	if _, cmd := send(m, tui.Key{Type: tui.KeySpace}); cmd != nil {
		t.Error("space should not animate under NO_ANIMATION")
	}
	w := wallModel(t)
	x, y := wallCell(0, 0)
	if next, cmd := send(w, press(x, y)); cmd != nil || len(next.plays) != 0 {
		t.Error("a wall click should not animate under NO_ANIMATION")
	}
}

func TestArrowsAndVimKeysStepThroughFaces(t *testing.T) {
	m := initialModel()
	m, _ = send(m, tui.Key{Type: tui.KeyRight})
	m, _ = send(m, key('l'))
	if m.player.Index() != 2 {
		t.Errorf("after two steps right index = %d, want 2", m.player.Index())
	}
	m, _ = send(m, key('h'))
	m, _ = send(m, tui.Key{Type: tui.KeyLeft})
	m, _ = send(m, tui.Key{Type: tui.KeyLeft})
	if m.player.Index() != faces.Count()-1 {
		t.Errorf("stepping left past the start should wrap, got %d", m.player.Index())
	}
}

func TestSteppingToAnotherFaceStopsThePreviousAnimation(t *testing.T) {
	m := initialModel()
	m, _ = send(m, press(3, spriteTop+1))
	m, _ = send(m, tui.Key{Type: tui.KeyRight})
	if m.player.Running() || m.player.Frame() != 0 {
		t.Error("a newly selected face should rest until it is clicked")
	}
}

func TestQuitKeys(t *testing.T) {
	for _, k := range []tui.Key{key('q'), {Type: tui.KeyEsc}, {Type: tui.KeyCtrlC}} {
		_, cmd := send(initialModel(), k)
		if cmd == nil {
			t.Errorf("%v should quit", k)
			continue
		}
		if _, ok := tui.RunCmd(context.Background(), cmd).(tui.QuitMsg); !ok {
			t.Errorf("%v produced %T, want QuitMsg", k, tui.RunCmd(context.Background(), cmd))
		}
	}
}

func TestWallOpensOnThePageOfTheCurrentFaceAndPagesWrap(t *testing.T) {
	m := initialModel()
	m.player.Set(m.perPage() + 3) // second page
	m, _ = send(m, key('w'))
	if !m.wall || m.page != 1 {
		t.Fatalf("wall=%v page=%d, want wall on page 1", m.wall, m.page)
	}
	for i := 0; i < m.pages(); i++ {
		m, _ = send(m, tui.Key{Type: tui.KeyRight})
	}
	if m.page != 1 {
		t.Errorf("a full lap of pages should return to page 1, got %d", m.page)
	}
	m, _ = send(m, tui.Key{Type: tui.KeyLeft})
	m, _ = send(m, tui.Key{Type: tui.KeyLeft})
	if m.page != m.pages()-1 {
		t.Errorf("left from the first page should wrap to the last, got %d", m.page)
	}
	m, _ = send(m, key('w'))
	if m.wall || m.player.Index() != m.page*m.perPage() {
		t.Errorf("leaving the wall should land on the page's first face (index %d)", m.player.Index())
	}
}

func TestWallColumnsFollowTheTerminalWidth(t *testing.T) {
	m := initialModel()
	for _, c := range []struct{ width, cols int }{{0, 4}, {200, 4}, {54, 4}, {53, 3}, {40, 3}, {27, 2}, {25, 1}, {12, 1}, {5, 1}} {
		m, _ = send(m, tui.ResizeMsg{Width: c.width, Height: 30})
		if m.cols() != c.cols {
			t.Errorf("width %d -> %d columns, want %d", c.width, m.cols(), c.cols)
		}
	}
}

func TestResizeKeepsTheSameFacesInViewAndClearsPlays(t *testing.T) {
	m := wallModel(t)
	m, _ = send(m, tui.Key{Type: tui.KeyRight}) // faces 8..15
	first := m.page * m.perPage()
	x, y := wallCell(0, 0)
	m, _ = send(m, press(x, y))
	m, _ = send(m, tui.ResizeMsg{Width: 36, Height: 30}) // 2 columns: 4 faces a page
	if got := m.page * m.perPage(); got > first || first-got >= m.perPage() {
		t.Errorf("after narrowing, page starts at face %d; face %d should still be on it", got, first)
	}
	if len(m.plays) != 0 {
		t.Error("a resize should clear in-flight plays (their cells moved)")
	}
	m, _ = send(m, tui.ResizeMsg{Width: 17, Height: 30})
	if m.page >= m.pages() {
		t.Errorf("page %d out of range (%d pages)", m.page, m.pages())
	}
}

func TestViewsShowTheFaceAndTheWall(t *testing.T) {
	m := initialModel()
	single := ansi.StripANSI(m.View())
	if !strings.Contains(single, faces.Get(0).Name) || !strings.Contains(single, "#1 of 50") || !strings.Contains(single, "click") {
		t.Errorf("single view missing the face name, position or click hint:\n%s", single)
	}

	m, _ = send(m, key('w'))
	wall := ansi.StripANSI(m.View())
	for i := 0; i < m.perPage(); i++ {
		if !strings.Contains(wall, faces.Get(i).Name) {
			t.Errorf("wall page 1 missing %s", faces.Get(i).Name)
		}
	}
	if !strings.Contains(wall, "wall 1/7") {
		t.Errorf("wall view missing page indicator:\n%s", wall)
	}
	for _, line := range strings.Split(wall, "\n") {
		if strings.Contains(line, "quit") {
			continue // the help line is text, not part of the grid
		}
		// The layout pads every row to the widest one (the help line), so measure
		// the content: trailing padding is not part of the wall.
		if w := ansi.Width(strings.TrimRight(line, " ")); w > maxWallCols*(smallW+wallGap) {
			t.Errorf("wall line is %d columns wide, over the %d budget: %q", w, maxWallCols*(smallW+wallGap), line)
		}
	}
}

// --- size ---

func TestSKeyTogglesSmallAndLargeAndKeepsTheSameFacesInView(t *testing.T) {
	m := wallModel(t)
	m, _ = send(m, tui.Key{Type: tui.KeyRight}) // page 1: faces 8..15
	first := m.page * m.perPage()
	m, _ = send(m, tui.ResizeMsg{Width: 200, Height: 60})

	m, _ = send(m, key('s'))
	if m.player.Size != faces.Large {
		t.Fatalf("size = %d, want Large", m.player.Size)
	}
	if got := m.page * m.perPage(); got > first || first-got >= m.perPage() {
		t.Errorf("after switching, page starts at face %d but face %d should still be on it", got, first)
	}
	m, _ = send(m, key('s'))
	if m.player.Size != faces.Small {
		t.Errorf("s again should return to Small, got %d", m.player.Size)
	}
}

func TestSwitchingSizeStopsAnyAnimation(t *testing.T) {
	m := initialModel()
	m, _ = send(m, press(3, spriteTop+1))
	m, _ = send(m, key('s'))
	if m.player.Running() {
		t.Error("the player should rest after a size change")
	}
	m = wallModel(t)
	x, y := wallCell(0, 0)
	m, _ = send(m, press(x, y))
	m, _ = send(m, key('s'))
	if len(m.plays) != 0 {
		t.Error("wall animations should stop after a size change")
	}
}

func TestLargeFacesHaveTheirOwnClickAreasAndWallColumns(t *testing.T) {
	m := initialModel()
	m, _ = send(m, key('s'))
	w, h := faces.Large.Cells()
	if w <= smallW || h <= smallH {
		t.Fatalf("large cells %dx%d should exceed small %dx%d", w, h, smallW, smallH)
	}
	// Inside the large sprite but outside the small one: only the large hit-test accepts it.
	if _, cmd := send(m, press(w-2, spriteTop+h-2)); cmd == nil {
		t.Error("a click on the large sprite's far corner should play it")
	}
	if _, cmd := send(m, press(w, spriteTop)); cmd != nil {
		t.Error("a click just past the large sprite should miss")
	}

	m, _ = send(m, tui.ResizeMsg{Width: 74, Height: 40})
	if got, want := m.cols(), (74+wallGap)/(w+wallGap); got != want {
		t.Errorf("74 columns holds %d large faces, want %d", got, want)
	}
	m, _ = send(m, key('w'))
	x, y := wallCell(0, 0)
	if next, _ := send(m, press(x, y)); len(next.plays) != 1 {
		t.Error("a click on the first large wall face should play it")
	}
	lastRow := spriteTop + (h+2)*wallRows - 1 // the blank line below the last row
	if next, cmd := send(m, press(x, lastRow+1)); cmd != nil || len(next.plays) != 0 {
		t.Error("a click below the last large row should miss")
	}
}

func TestLargeHitGeometryMatchesTheRenderedLayout(t *testing.T) {
	m, _ := send(initialModel(), key('s'))
	_, h := faces.Large.Cells()
	lines := strings.Split(ansi.StripANSI(m.View()), "\n")
	if !strings.HasPrefix(lines[spriteTop+h], faces.Get(0).Name) {
		t.Errorf("label should sit on line %d under the large sprite, line is %q", spriteTop+h, lines[spriteTop+h])
	}
}
