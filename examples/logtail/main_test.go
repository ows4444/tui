package main

import (
	"fmt"
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

func click(t testing.TB, m model, id string) model {
	r := rect(t, m, id)
	return send(m, tui.MouseEvent{X: r.X + 1, Y: r.Y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
}

// logModel is a 100x14 model holding the given lines.
func logModel(lines ...string) model {
	m := send(newModel("t.log"), tui.ResizeMsg{Width: 100, Height: 14})
	return m.ingest(lines)
}

var mixed = []string{
	"2026-10-18T09:30:00Z TRACE tick 1",
	"2026-10-18T09:30:01Z DEBUG cache miss key=a",
	"2026-10-18T09:30:02Z INFO started",
	"2026-10-18T09:30:03Z WARN slow request GET /api",
	"2026-10-18T09:30:04Z ERROR disk full",
	`{"time":"2026-10-18T09:30:05Z","level":"error","msg":"upstream timeout","service":"payments"}`,
	`{"time":"2026-10-18T09:30:06Z","level":"info","msg":"request","path":"/api/orders","status":200}`,
}

func numbered(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("2026-10-18T09:30:00Z INFO line %d", i+1)
	}
	return out
}

func TestShowsLinesWithLevelsAndCounts(t *testing.T) {
	m := logModel(mixed...)
	out := text(m)
	for _, w := range []string{"logtail", "☑1 TRACE 1", "☑3 INFO 2", "☑4 WARN 1", "☑5 ERROR 2", "ERROR disk full"[:5], "disk full", "upstream timeout  service=payments", "t.log · 7 of 7 lines", "☑ follow"} {
		if !strings.Contains(out, w) {
			t.Errorf("screen lacks %q:\n%s", w, out)
		}
	}
}

func TestLevelTogglesHideAndShow(t *testing.T) {
	m := runes(logModel(mixed...), "34") // hide INFO and WARN
	out := text(m)
	if strings.Contains(out, "started") || strings.Contains(out, "slow request") || !strings.Contains(out, "disk full") || !strings.Contains(out, "5 of 7") && !strings.Contains(out, "of 7 lines") {
		t.Fatalf("\n%s", out)
	}
	if !strings.Contains(out, "☐3 INFO 2") {
		t.Fatalf("hidden level should read as unchecked:\n%s", out)
	}
	m = click(t, m, "lvl-2")
	if !m.levels[lInfo] || !strings.Contains(text(m), "started") {
		t.Fatal("clicking the chip did not bring INFO back")
	}
	m = runes(runes(m, "1"), "2")
	for l := lTrace; l < numLevels; l++ { // hide whatever is still showing
		if m.levels[l] {
			m = m.toggleLevel(l)
		}
	}
	if !strings.Contains(text(m), "Every line is hidden") {
		t.Fatalf("\n%s", text(m))
	}
	if m = runes(m, "9"); m.levels != [numLevels]bool{} {
		t.Fatal("a digit past the levels changed something")
	}
}

func TestEntriesWithoutALevelCountAsInfo(t *testing.T) {
	m := runes(logModel("plain words here", "2026-10-18T09:30:02Z ERROR boom"), "3")
	out := text(m)
	if strings.Contains(out, "plain words") || !strings.Contains(out, "boom") {
		t.Fatalf("\n%s", out)
	}
}

func TestRegexFilterHighlightsAndEscClears(t *testing.T) {
	m := runes(logModel(mixed...), "/")
	if !m.typing {
		t.Fatal("/ did not open the filter")
	}
	for _, r := range `slow|timeout` {
		m = send(m, tui.Key{Type: tui.KeyRunes, Text: string(r)})
	}
	m = key(m, tui.KeyEnter)
	out := text(m)
	if len(m.vis) != 2 || !strings.Contains(out, "slow request") || !strings.Contains(out, "upstream timeout") || strings.Contains(out, "started") {
		t.Fatalf("vis=%d\n%s", len(m.vis), out)
	}
	if !strings.Contains(out, "[/slow|timeout]") {
		t.Fatalf("the filter is not shown in the bar:\n%s", out)
	}
	if m = key(m, tui.KeyEsc); m.re != nil || len(m.vis) != 7 {
		t.Fatalf("Esc did not clear the filter: vis=%d", len(m.vis))
	}
}

func TestFilterIsCaseInsensitiveUntilToggled(t *testing.T) {
	m := logModel(mixed...).setFilter("DISK")
	if len(m.vis) != 1 {
		t.Fatalf("vis = %d", len(m.vis))
	}
	m = runes(m, "i")
	if m.nocase || len(m.vis) != 0 {
		t.Fatalf("nocase=%v vis=%d after i", m.nocase, len(m.vis))
	}
	m = m.setFilter("disk")
	if len(m.vis) != 1 {
		t.Fatalf("vis = %d", len(m.vis))
	}
}

func TestBadRegexIsExplainedAndKeepsTheOldFilter(t *testing.T) {
	m := logModel(mixed...).setFilter("disk")
	m = m.setFilter("(unclosed")
	if m.query != "disk" || len(m.vis) != 1 || !strings.Contains(m.msg, "Not a valid regular expression") {
		t.Fatalf("query=%q vis=%d msg=%q", m.query, len(m.vis), m.msg)
	}
	m = runes(m, "/")
	if m = key(m, tui.KeyEsc); m.typing {
		t.Fatal("Esc did not close the prompt")
	}
	if _, cmd := update(runes(m, "/"), tui.Key{Type: tui.KeyRunes, Text: "q"}); isQuit(cmd) {
		t.Fatal("q quit while typing a filter")
	}
}

func TestFilterAppliesToLinesThatArriveLater(t *testing.T) {
	m := logModel(mixed...).setFilter("disk")
	m = m.ingest([]string{"2026-10-18T09:31:00Z ERROR disk almost full", "2026-10-18T09:31:01Z INFO unrelated"})
	if len(m.vis) != 2 || len(m.ents) != 9 {
		t.Fatalf("vis=%d ents=%d", len(m.vis), len(m.ents))
	}
}

func TestPauseHoldsNewLinesAndResumeShowsThem(t *testing.T) {
	m := logModel(numbered(3)...)
	m = runes(m, " ")
	if !m.paused || !strings.Contains(text(m), "resume 0 held") {
		t.Fatalf("\n%s", text(m))
	}
	m = m.receive(numbered(5))
	if len(m.ents) != 3 || len(m.pending) != 5 || !strings.Contains(text(m), "resume 5 held") || !strings.Contains(text(m), "PAUSED") {
		t.Fatalf("ents=%d pending=%d\n%s", len(m.ents), len(m.pending), text(m))
	}
	m = click(t, m, "pause")
	if m.paused || len(m.ents) != 8 || len(m.pending) != 0 {
		t.Fatalf("paused=%v ents=%d pending=%d", m.paused, len(m.ents), len(m.pending))
	}
}

func TestHeldLinesAreBounded(t *testing.T) {
	m := runes(logModel(), "p")
	m.pending = make([]string, maxPending-2)
	m = m.receive(numbered(10))
	if len(m.pending) != maxPending || m.lost != 8 || !strings.Contains(text(m), "8 lost") {
		t.Fatalf("pending=%d lost=%d", len(m.pending), m.lost)
	}
}

func TestFollowStaysAtTheEndUntilYouScrollUp(t *testing.T) {
	m := logModel(numbered(50)...)
	if !m.follow || m.cur != 49 || !strings.Contains(text(m), "line 50") {
		t.Fatalf("cur=%d follow=%v", m.cur, m.follow)
	}
	m = m.ingest(numbered(5))
	if m.cur != 54 || !strings.Contains(text(m), "line 5") {
		t.Fatalf("cur = %d: did not follow", m.cur)
	}
	m = key(key(m, tui.KeyUp), tui.KeyUp)
	if m.follow {
		t.Fatal("moving up should stop following")
	}
	at := m.cur
	m = m.ingest(numbered(5))
	if m.cur != at {
		t.Fatalf("cur moved from %d to %d while reading older lines", at, m.cur)
	}
	m = runes(m, "G")
	if !m.follow || m.cur != len(m.vis)-1 {
		t.Fatalf("G: follow=%v cur=%d", m.follow, m.cur)
	}
	m = runes(m, "f")
	if m.follow || !strings.Contains(text(m), "☐ follow") {
		t.Fatal("f did not turn following off")
	}
	m = click(t, m, "follow")
	if !m.follow {
		t.Fatal("clicking Follow did not turn it on")
	}
}

func TestMovingDownOntoTheLastLineFollowsAgain(t *testing.T) {
	m := logModel(numbered(30)...)
	m = key(key(m, tui.KeyUp), tui.KeyUp)
	m = key(m, tui.KeyDown)
	if m.follow {
		t.Fatal("followed too early")
	}
	if m = key(m, tui.KeyDown); !m.follow {
		t.Fatal("reaching the last line did not resume following")
	}
}

func TestOpenAndCloseALine(t *testing.T) {
	m := logModel(mixed...)
	m = runes(m, "g")
	for i := 0; i < 5; i++ {
		m = key(m, tui.KeyDown) // to the JSON "upstream timeout" line
	}
	m = key(m, tui.KeyEnter)
	id := m.ents[m.vis[m.cur]].id
	if !m.open[id] {
		t.Fatalf("Enter did not open line %d", id)
	}
	out := text(m)
	if !strings.Contains(out, "service  payments") || !strings.Contains(out, "▾") {
		t.Fatalf("no detail:\n%s", out)
	}
	if m = key(m, tui.KeyEnter); m.open[id] || strings.Contains(text(m), "service  payments") {
		t.Fatal("Enter did not close it")
	}
	m = key(m, tui.KeyEnter)
	if m = runes(m, "E"); len(m.open) != 0 {
		t.Fatal("E did not close every line")
	}
}

func TestClickSelectsThenOpens(t *testing.T) {
	m := logModel(mixed...)
	r := rect(t, m, "log")
	m = send(m, tui.MouseEvent{X: r.X + 20, Y: r.Y + 2, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if m.cur != 2 || m.follow {
		t.Fatalf("cur = %d follow = %v", m.cur, m.follow)
	}
	m = send(m, tui.MouseEvent{X: r.X + 20, Y: r.Y + 2, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if !m.open[m.ents[2].id] {
		t.Fatal("a second click did not open the line")
	}
	// A line below an open one is further down the screen.
	m = send(m, tui.MouseEvent{X: r.X + 20, Y: r.Y + 4 + len(m.ents[2].detail(94)), Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if m.cur != 4 {
		t.Fatalf("cur = %d after clicking below an open line", m.cur)
	}
}

func TestOpenLinesScrollIntoView(t *testing.T) {
	m := logModel(append(numbered(20), `{"level":"info","msg":"big","a":"1","b":"2","c":"3","d":"4","e":"5","f":"6"}`)...)
	m = key(m, tui.KeyEnter)
	if !strings.Contains(text(m), "f      6") {
		t.Fatalf("an opened last line is cut off:\n%s", text(m))
	}
}

func TestWheelScrolls(t *testing.T) {
	m := logModel(numbered(100)...)
	m = send(m, tui.MouseEvent{Button: tui.MouseButtonWheelUp, Action: tui.MouseActionPress})
	if m.cur != 96 || m.follow {
		t.Fatalf("cur = %d follow = %v", m.cur, m.follow)
	}
	m = send(m, tui.MouseEvent{Button: tui.MouseButtonWheelDown, Action: tui.MouseActionPress})
	if m.cur != 99 || !m.follow {
		t.Fatalf("cur = %d follow = %v", m.cur, m.follow)
	}
}

func TestOldLinesAreDroppedAtTheLimit(t *testing.T) {
	m := logModel(numbered(maxEntries)...)
	m = m.ingest(numbered(10))
	if len(m.ents) > maxEntries || m.dropped == 0 {
		t.Fatalf("ents=%d dropped=%d", len(m.ents), m.dropped)
	}
	if got := m.counts[lInfo]; got != len(m.ents) {
		t.Fatalf("counts %d disagree with %d entries", got, len(m.ents))
	}
	if !strings.Contains(text(m), "older dropped") {
		t.Fatalf("\n%s", text(m))
	}
	if len(m.vis) != len(m.ents) || m.cur != len(m.vis)-1 {
		t.Fatalf("vis=%d cur=%d", len(m.vis), m.cur)
	}
}

func TestClearAndCopy(t *testing.T) {
	m := logModel(mixed...)
	m2, cmd := update(m, tui.Key{Type: tui.KeyRunes, Text: "y"})
	if cmd == nil || !strings.Contains(text(m2), "Copied the line") {
		t.Fatalf("\n%s", text(m2))
	}
	m = runes(m, "C")
	if len(m.ents) != 0 || !strings.Contains(text(m), "Log cleared") {
		t.Fatalf("\n%s", text(m))
	}
	if _, cmd = update(m, tui.Key{Type: tui.KeyRunes, Text: "y"}); cmd != nil {
		t.Fatal("copied from an empty log")
	}
	if !strings.Contains(text(logModel()), "Waiting for log lines") {
		t.Fatal("no waiting message")
	}
}

func TestEndOfInputAndErrorsAreShown(t *testing.T) {
	m := send(logModel(mixed...), eofMsg{})
	if !strings.Contains(text(m), "end of input") {
		t.Fatalf("\n%s", text(m))
	}
	m = send(m, eofMsg{err: fmt.Errorf("read failed")})
	if !strings.Contains(text(m), "✗ read failed") {
		t.Fatalf("\n%s", text(m))
	}
}

func TestGeneratorFeedsTheModel(t *testing.T) {
	m := send(sampleModel(), tui.ResizeMsg{Width: 100, Height: 24})
	n := len(m.ents)
	if n != 40 {
		t.Fatalf("%d lines of history", n)
	}
	m, cmd := update(m, genMsg{m.seq})
	if len(m.ents) <= n || cmd == nil {
		t.Fatal("a tick did not add lines and ask for the next")
	}
	if m2, cmd := update(m, genMsg{m.seq + 1}); len(m2.ents) != len(m.ents) || cmd != nil {
		t.Fatal("a stale tick added lines")
	}
}

func TestHelpAndQuit(t *testing.T) {
	m := runes(logModel(mixed...), "?")
	if !strings.Contains(text(m), "pause and resume (new lines wait)") {
		t.Fatalf("\n%s", text(m))
	}
	if m = key(m, tui.KeyEsc); m.help {
		t.Fatal("Esc did not close help")
	}
	for _, k := range []tui.Key{{Type: tui.KeyCtrlC}, {Type: tui.KeyRunes, Text: "q"}} {
		if _, cmd := update(logModel(), k); !isQuit(cmd) {
			t.Errorf("%v did not quit", k)
		}
	}
}

func TestTinyTerminalsStayInside(t *testing.T) {
	m := logModel(mixed...)
	m = key(m, tui.KeyEnter)
	for _, sz := range [][2]int{{40, 10}, {20, 6}, {10, 4}, {300, 80}, {3, 3}} {
		m = send(m, tui.ResizeMsg{Width: sz[0], Height: sz[1]})
		for i, line := range strings.Split(m.View(), "\n") {
			if w := ansi.Width(line); w > sz[0] {
				t.Fatalf("%dx%d: line %d is %d wide: %q", sz[0], sz[1], i, w, line)
			}
		}
	}
}
