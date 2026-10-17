package main

import (
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/tuitest"
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

func click(t testing.TB, m model, id string) (model, tui.Cmd) {
	r := rect(t, m, id)
	return update(m, tui.MouseEvent{X: r.X + 1, Y: r.Y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
}

// live runs the model in a real Program against the fast fixture.
func live(t *testing.T, city string) (*tuitest.Session, *fixture) {
	t.Helper()
	f := fastFixture()
	m := newModel(f, 0, city)
	m.now = f.now
	s := tuitest.New(m, 100, 28)
	t.Cleanup(s.Close)
	return s, f
}

func screen(s *tuitest.Session) string { return strings.Join(s.Screen(), "\n") }

func want(t *testing.T, s *tuitest.Session, text string) {
	t.Helper()
	if !s.WaitForText(text, 3*time.Second) {
		t.Fatalf("screen never showed %q:\n%s", text, screen(s))
	}
}

func TestStartsIdleWithAPrompt(t *testing.T) {
	m := send(newModel(fastFixture(), 0, ""), tui.ResizeMsg{Width: 100, Height: 28})
	out := text(m)
	for _, w := range []string{"weather", "Search ▸", "Recent", "none yet", "Search for a city above", "Try Berlin", "built-in fixture", "[°C]"} {
		if !strings.Contains(out, w) {
			t.Errorf("screen lacks %q:\n%s", w, out)
		}
	}
}

func TestSearchOpenAndShowSevenDays(t *testing.T) {
	s, _ := live(t, "")
	s.Keys("ber")
	want(t, s, "› Berlin, Germany")
	s.Keys("enter")
	want(t, s, "fetched 09:30:00 from built-in fixture")
	out := screen(s)
	for _, w := range []string{"Berlin, Germany", "Now ", "Wed 14 Oct", "Tue 20 Oct", "▲", "▼", "☂", "1 Berlin"} {
		if !strings.Contains(out, w) {
			t.Errorf("forecast lacks %q:\n%s", w, out)
		}
	}
}

func TestOpeningACityFromTheCommandLine(t *testing.T) {
	s, _ := live(t, "oslo")
	want(t, s, "Oslo, Norway")
	want(t, s, "fetched ")
}

func TestUnknownCityOnTheCommandLine(t *testing.T) {
	s, _ := live(t, "zzzz")
	want(t, s, "No city called zzzz")
}

func TestFlakyPlaceFailsThenRetrySucceeds(t *testing.T) {
	s, _ := live(t, "")
	s.Keys("reyk")
	want(t, s, "› Reykjavik, Iceland")
	s.Keys("enter")
	want(t, s, "Could not load Reykjavik, Iceland")
	out := screen(s)
	if !strings.Contains(out, "did not answer in time") || !strings.Contains(out, "Retry (r)") {
		t.Fatalf("\n%s", out)
	}
	s.Keys("tab", "r")
	want(t, s, "fetched ")
	if strings.Contains(screen(s), "Could not load") {
		t.Fatal("the error stayed after a good retry")
	}
}

func TestRetryButton(t *testing.T) {
	s, _ := live(t, "")
	s.Keys("reyk")
	want(t, s, "› Reykjavik, Iceland")
	s.Keys("enter")
	want(t, s, "Retry (r)")
	if !s.ClickText("Retry (r)") {
		t.Fatalf("no Retry button:\n%s", screen(s))
	}
	want(t, s, "fetched ")
}

func TestMissingPlaceIsExplained(t *testing.T) {
	s, _ := live(t, "")
	s.Keys("atlan")
	want(t, s, "› Atlantis, Nowhere")
	s.Keys("enter")
	want(t, s, "No forecast for that place (404)")
}

func TestEscCancelsASlowLoad(t *testing.T) {
	s, _ := live(t, "")
	s.Keys("slow")
	want(t, s, "› Slowville, Patience")
	s.Keys("enter")
	want(t, s, "Loading the forecast for Slowville, Patience")
	s.Keys("esc")
	want(t, s, "Cancelled loading Slowville")
	if strings.Contains(screen(s), "Loading the forecast") {
		t.Fatalf("still loading:\n%s", screen(s))
	}
}

func TestCancelKeepsThePreviousForecast(t *testing.T) {
	m := loadedModel()
	m, _ = m.load(cities[11]) // Slowville
	if m.state != loading || !strings.Contains(text(m), "Still showing Berlin") {
		t.Fatalf("\n%s", text(m))
	}
	m = key(m, tui.KeyEsc)
	if m.state != loaded || !m.have || !strings.Contains(text(m), "Berlin, Germany") || !strings.Contains(m.msg, "Cancelled") {
		t.Fatalf("state = %v\n%s", m.state, text(m))
	}
}

func TestStaleAnswersAreIgnored(t *testing.T) {
	f := fastFixture()
	m, _ := newModel(f, 0, "").load(cities[0])
	m, _ = m.load(cities[2]) // wanted Oslo now
	m = m.onLoaded(loadedMsg{p: cities[0], fc: f.build(cities[0])})
	if m.state != loading || m.have {
		t.Fatalf("an answer for Berlin was taken while waiting for Oslo: state %v have %v", m.state, m.have)
	}
	m = m.onLoaded(loadedMsg{p: cities[2], fc: f.build(cities[2])})
	if m.state != loaded || m.fc.Place.Name != "Oslo" {
		t.Fatalf("state %v place %s", m.state, m.fc.Place.Name)
	}
	m = m.onLoaded(loadedMsg{p: cities[2], err: context_error()})
	if m.state != loaded {
		t.Fatal("an answer after the load had finished changed the screen")
	}
}

func context_error() error { return errNotFound }

func TestCanceledTaskResultsAreDropped(t *testing.T) {
	m := loadedModel()
	m, _ = m.load(cities[1])
	m, _ = update(m, tui.TaskDoneMsg{Key: "forecast", Canceled: true, Msg: loadedMsg{p: cities[1], err: errNotFound}})
	if m.state != loading {
		t.Fatalf("a cancelled run ended the load: %v", m.state)
	}
	m, _ = update(m, tui.TaskDoneMsg{Key: "forecast", Msg: nil})
	if m.state != loading {
		t.Fatal("an empty result ended the load")
	}
}

func TestRecentsAreFreshestFirstUniqueAndBounded(t *testing.T) {
	f := fastFixture()
	m := newModel(f, 0, "")
	for _, c := range []int{0, 1, 2, 3, 4, 5, 1} {
		m, _ = m.load(cities[c])
		m = m.onLoaded(loadedMsg{p: cities[c], fc: f.build(cities[c])})
	}
	var names []string
	for _, p := range m.recents {
		names = append(names, p.Name)
	}
	if strings.Join(names, ",") != "Cairo,Nairobi,Lima,Tokyo,Oslo" {
		t.Fatalf("recents = %v", names)
	}
}

func TestRecentsReopenByNumberAndClick(t *testing.T) {
	m := loadedModel()
	m = send(m, tui.ResizeMsg{Width: 100, Height: 28})
	f := fastFixture()
	m, _ = m.load(cities[1])
	m = m.onLoaded(loadedMsg{p: cities[1], fc: f.build(cities[1])})
	m = key(m, tui.KeyTab) // to the days
	m, cmd := update(m, tui.Key{Type: tui.KeyRunes, Text: "2"})
	if cmd == nil || m.state != loading || m.target.Name != "Berlin" {
		t.Fatalf("state %v target %s", m.state, m.target.Name)
	}
	m = m.onLoaded(loadedMsg{p: cities[0], fc: f.build(cities[0])})
	m, cmd = click(t, m, "rec-1")
	if cmd == nil || m.target.Name != "Cairo" {
		t.Fatalf("clicking the second recent loaded %s", m.target.Name)
	}
	if _, cmd = update(m, tui.Key{Type: tui.KeyRunes, Text: "9"}); cmd != nil {
		t.Fatal("a number past the recents loaded something")
	}
}

func TestDaysAreSelectableAndDetailFollows(t *testing.T) {
	m := key(loadedModel(), tui.KeyTab)
	m = send(m, tui.ResizeMsg{Width: 100, Height: 28})
	m = key(key(m, tui.KeyRight), tui.KeyRight)
	if m.sel != 2 || !strings.Contains(text(m), "Fri 16 Oct:") {
		t.Fatalf("sel = %d\n%s", m.sel, text(m))
	}
	for i := 0; i < 20; i++ {
		m = key(m, tui.KeyRight)
	}
	if m.sel != 6 {
		t.Fatalf("sel = %d past the last day", m.sel)
	}
	for i := 0; i < 20; i++ {
		m = key(m, tui.KeyLeft)
	}
	if m.sel != 0 {
		t.Fatalf("sel = %d", m.sel)
	}
	m, _ = click(t, m, "day-4")
	if m.sel != 4 || m.typing {
		t.Fatalf("sel = %d typing = %v after clicking a card", m.sel, m.typing)
	}
}

func TestUnitsToggleChangesEveryNumber(t *testing.T) {
	m := send(loadedModel(), tui.ResizeMsg{Width: 100, Height: 28})
	if !strings.Contains(text(m), "°C") || strings.Contains(text(m), "°F") {
		t.Fatalf("\n%s", text(m))
	}
	m, _ = click(t, m, "units")
	out := text(m)
	if !strings.Contains(out, "°F") || strings.Contains(out, "°C") || !strings.Contains(out, "mph") || !strings.Contains(out, "[°F]") {
		t.Fatalf("\n%s", out)
	}
	m = runes(key(m, tui.KeyTab), "u")
	if m.units != metric {
		t.Fatal("u did not switch back")
	}
}

func TestCardsBecomeAListWhenThereIsNoRoom(t *testing.T) {
	m := send(loadedModel(), tui.ResizeMsg{Width: 100, Height: 28})
	if !strings.Contains(text(m), "╭") {
		t.Fatalf("no cards at 100x28:\n%s", text(m))
	}
	for _, sz := range [][2]int{{100, 14}, {40, 28}} {
		m = send(m, tui.ResizeMsg{Width: sz[0], Height: sz[1]})
		out := text(m)
		if strings.Contains(out, "╭") || !strings.Contains(out, "Tue 20") {
			t.Fatalf("%dx%d should list the days:\n%s", sz[0], sz[1], out)
		}
	}
	m = send(m, tui.ResizeMsg{Width: 40, Height: 28})
	m, _ = click(t, m, "day-3")
	if m.sel != 3 {
		t.Fatalf("clicking a listed day: sel = %d", m.sel)
	}
}

func TestCardsWrapOnNarrowTerminals(t *testing.T) {
	m := send(loadedModel(), tui.ResizeMsg{Width: 60, Height: 40})
	if rect(t, m, "day-0").Y == rect(t, m, "day-6").Y {
		t.Fatal("seven cards in one row at 60 columns")
	}
}

func TestResultsListAndNoMatch(t *testing.T) {
	m := send(newModel(fastFixture(), 0, ""), tui.ResizeMsg{Width: 100, Height: 28})
	m.in.SetValue("o")
	m = send(m, searchedMsg{q: "o", res: cities[:3]})
	if !m.showRes || len(m.results) != 3 || !strings.Contains(text(m), "› Berlin, Germany") {
		t.Fatalf("\n%s", text(m))
	}
	m = key(key(m, tui.KeyDown), tui.KeyDown)
	m = key(m, tui.KeyDown)
	if m.resCur != 2 {
		t.Fatalf("resCur = %d past the end", m.resCur)
	}
	m = key(m, tui.KeyEsc)
	if m.showRes {
		t.Fatal("Esc did not close the results")
	}
	m.in.SetValue("zz")
	m = send(m, searchedMsg{q: "zz"})
	if !strings.Contains(text(m), "No city matches “zz”") {
		t.Fatalf("\n%s", text(m))
	}
}

func TestOutOfDateSearchAnswerIsIgnored(t *testing.T) {
	m := newModel(fastFixture(), 0, "")
	m.in.SetValue("ber")
	m = send(m, searchedMsg{q: "be", res: cities[:2]})
	if m.showRes {
		t.Fatal("an answer for an older query was shown")
	}
	m = send(m, searchedMsg{q: "ber", err: errNotFound})
	if !strings.Contains(m.msg, "Search failed") || m.showRes {
		t.Fatalf("msg = %q", m.msg)
	}
}

func TestClickAResultOpensIt(t *testing.T) {
	m := send(newModel(fastFixture(), 0, ""), tui.ResizeMsg{Width: 100, Height: 28})
	m.in.SetValue("o")
	m = send(m, searchedMsg{q: "o", res: cities[:3]})
	m, cmd := click(t, m, "res-2")
	if cmd == nil || m.state != loading || m.target.Name != "Oslo" || m.in.Value() != "" || m.showRes {
		t.Fatalf("state %v target %s input %q", m.state, m.target.Name, m.in.Value())
	}
}

func TestDebounceSearchesOnlyAfterTypingPauses(t *testing.T) {
	f := &countingProvider{fixture: fastFixture()}
	m := newModel(f, 250*time.Millisecond, "")
	m.now = f.now
	s := tuitest.New(m, 100, 28)
	defer s.Close()
	s.Keys("b", "e", "r") // three keystrokes well inside one debounce
	want(t, s, "› Berlin, Germany")
	if n := f.searches(); n != 1 {
		t.Fatalf("%d searches reached the provider for three keystrokes, want 1", n)
	}
}

func TestHelpAndQuit(t *testing.T) {
	m := send(loadedModel(), tui.ResizeMsg{Width: 100, Height: 28})
	m = key(m, tui.KeyF1)
	if !strings.Contains(text(m), "reload, or retry after a failure") {
		t.Fatalf("\n%s", text(m))
	}
	if m = key(m, tui.KeyEsc); m.help {
		t.Fatal("Esc did not close help")
	}
	if _, cmd := update(m, tui.Key{Type: tui.KeyRunes, Text: "q"}); isQuit(cmd) {
		t.Fatal("q quit while the search box had the keys")
	}
	m = key(m, tui.KeyTab)
	for _, k := range []tui.Key{{Type: tui.KeyCtrlC}, {Type: tui.KeyRunes, Text: "q"}} {
		if _, cmd := update(m, k); !isQuit(cmd) {
			t.Errorf("%v did not quit", k)
		}
	}
}

func TestReloadWithNothingToReload(t *testing.T) {
	m := key(newModel(fastFixture(), 0, ""), tui.KeyTab)
	m = runes(m, "r")
	if m.msg != "Nothing to reload yet" || m.state != idle {
		t.Fatalf("msg = %q", m.msg)
	}
}

func TestTinyTerminalsStayInside(t *testing.T) {
	m := loadedModel()
	m, _ = m.load(cities[9])
	for _, st := range []state{idle, loading, failed, loaded} {
		m.state, m.err = st, "Something went wrong"
		for _, sz := range [][2]int{{40, 10}, {20, 6}, {10, 4}, {300, 80}, {3, 3}} {
			m = send(m, tui.ResizeMsg{Width: sz[0], Height: sz[1]})
			for i, line := range strings.Split(m.View(), "\n") {
				if w := ansi.Width(line); w > sz[0] {
					t.Fatalf("state %v %dx%d: line %d is %d wide: %q", st, sz[0], sz[1], i, w, line)
				}
			}
		}
	}
}
