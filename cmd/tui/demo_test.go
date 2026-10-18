package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/tuitest"
)

func screenText(s *tuitest.Session) string { return strings.Join(s.Screen(), "\n") }

func TestEveryEntryIsComplete(t *testing.T) {
	seen := map[string]bool{}
	entries := demoEntries()
	if len(entries) < 20 {
		t.Errorf("only %d entries", len(entries))
	}
	for _, e := range entries {
		if e.name == "" || e.group == "" || e.summary == "" || len(e.code) == 0 || e.build == nil {
			t.Errorf("entry %+v is incomplete", e.name)
		}
		if seen[strings.ToLower(e.name)] {
			t.Errorf("%q appears twice", e.name)
		}
		seen[strings.ToLower(e.name)] = true
		if len(e.code) > 6 {
			t.Errorf("%s: %d lines of code do not fit the code box", e.name, len(e.code))
		}
		for _, l := range e.code {
			if ansi.Width(l) > 56 {
				t.Errorf("%s: code line %q is wider than the code box", e.name, l)
			}
		}
	}
}

func TestEveryWidgetBuildsDrawsAndSurvivesKeys(t *testing.T) {
	keys := []tui.Key{
		{Type: tui.KeyDown}, {Type: tui.KeyUp}, {Type: tui.KeyLeft}, {Type: tui.KeyRight}, {Type: tui.KeyEnter},
		{Type: tui.KeySpace}, {Type: tui.KeyRunes, Text: "a"}, {Type: tui.KeyRunes, Text: "5"}, {Type: tui.KeyBackspace},
		{Type: tui.KeyPgDown}, {Type: tui.KeyPgUp}, {Type: tui.KeyHome}, {Type: tui.KeyEnd}, {Type: tui.KeyEsc},
	}
	for _, e := range demoEntries() {
		for _, th := range []theme.Theme{theme.DarkTheme(), theme.DarkTheme().ASCII(), theme.LightTheme()} {
			w, _ := e.build().withTheme(th).started()
			w, _ = w.focus()
			if strings.TrimSpace(ansi.StripANSI(w.View())) == "" {
				t.Errorf("%s draws nothing", e.name)
			}
			for _, k := range keys {
				next, _ := w.Update(k)
				var ok bool
				if w, ok = next.(demoWidget); !ok {
					t.Fatalf("%s: Update returned %T, not a demo widget", e.name, next)
				}
				for _, line := range strings.Split(ansi.StripANSI(w.View()), "\n") {
					if ansi.Width(line) > 70 {
						t.Errorf("%s: a line is %d wide after %v: %q", e.name, ansi.Width(line), k, line)
						break
					}
				}
			}
			w = w.blur()
			_ = w.View()
		}
	}
}

func TestTheFirstScreenShowsTheListTheWidgetAndItsCode(t *testing.T) {
	s := tuitest.New(newDemoApp(demoEntries(), ""), 100, 28)
	defer s.Close()
	got := screenText(s)
	for _, want := range []string{"Widgets 2", "> Button", "[ Save ]", "Code", "button.New", "Ctrl+T theme (dark)", "Enter try it"} {
		if !strings.Contains(got, want) {
			t.Errorf("the first screen lacks %q:\n%s", want, got)
		}
	}
	if !strings.ContainsAny(got, "┬├┤┴") {
		t.Errorf("the panes do not share their borders:\n%s", got)
	}
}

func TestTypingNarrowsTheListAndEscClears(t *testing.T) {
	s := tuitest.New(newDemoApp(demoEntries(), ""), 100, 28)
	defer s.Close()
	s.Keys("t", "o", "g")
	got := screenText(s)
	if !strings.Contains(got, "/ tog") || !strings.Contains(got, "> Toggle") || strings.Contains(got, "Checkbox") {
		t.Errorf("filtering by tog:\n%s", got)
	}
	s.Keys("backspace", "backspace", "backspace")
	if !strings.Contains(screenText(s), "type to filter") {
		t.Error("backspace did not empty the filter")
	}
	s.Keys("x", "y", "z")
	if !strings.Contains(screenText(s), "nothing matches") {
		t.Errorf("no match is not said:\n%s", screenText(s))
	}
	s.Keys("esc")
	if s.Done() {
		t.Error("Esc quit while a filter was set; it should clear the filter first")
	}
	if !strings.Contains(screenText(s), "type to filter") {
		t.Error("Esc did not clear the filter")
	}
}

func TestEnterGivesTheWidgetTheKeysAndTabTakesThemBack(t *testing.T) {
	s := tuitest.New(newDemoApp(demoEntries(), "Text input"), 100, 28)
	defer s.Close()
	if !strings.Contains(screenText(s), "> Text input") {
		t.Fatalf("did not start on Text input:\n%s", screenText(s))
	}
	s.Keys("enter")
	if !strings.Contains(screenText(s), "the widget has the keys") {
		t.Errorf("the footer does not say the widget has the keys:\n%s", screenText(s))
	}
	s.Keys("A", "d", "a")
	if !strings.Contains(screenText(s), "Name: Ada") {
		t.Errorf("typing did not reach the widget:\n%s", screenText(s))
	}
	if strings.Contains(screenText(s), "/ Ada") {
		t.Error("typing leaked into the list filter")
	}
	s.Keys("tab")
	if !strings.Contains(screenText(s), "up/down choose") {
		t.Errorf("Tab did not return to the list:\n%s", screenText(s))
	}
	s.Keys("down", "down")
	if strings.Contains(screenText(s), "Name: Ada") {
		t.Error("moving to another widget kept the old one")
	}
}

func TestWhatTheWidgetSendsIsShown(t *testing.T) {
	s := tuitest.New(newDemoApp(demoEntries(), "Button"), 100, 28)
	defer s.Close()
	s.Keys("enter", "enter")
	if !s.WaitForText("sent: PressedMsg", 3*time.Second) {
		t.Errorf("the message the button sent is not shown:\n%s", screenText(s))
	}
}

func TestCtrlTChangesTheThemeAndCtrlCQuits(t *testing.T) {
	s := tuitest.New(newDemoApp(demoEntries(), ""), 100, 28)
	defer s.Close()
	before := screenText(s)
	s.Keys("ctrl+t")
	after := screenText(s)
	if before == after || strings.Contains(after, "theme (dark)") {
		t.Errorf("Ctrl+T did not change the theme name:\n%s", after)
	}
	s.Keys("ctrl+c")
	if !s.Done() {
		t.Error("Ctrl+C did not quit")
	}
	s2 := tuitest.New(newDemoApp(demoEntries(), ""), 100, 28)
	defer s2.Close()
	s2.Keys("esc")
	if !s2.Done() {
		t.Error("Esc with no filter did not quit")
	}
}

func TestTheLayoutFitsEverySize(t *testing.T) {
	for _, sz := range [][2]int{{60, 14}, {80, 24}, {100, 40}, {120, 30}, {200, 60}} {
		s := tuitest.New(newDemoApp(demoEntries(), "Text area"), sz[0], sz[1])
		rows := s.Screen()
		if len(rows) != sz[1] {
			t.Errorf("%dx%d: %d rows", sz[0], sz[1], len(rows))
		}
		for _, r := range rows {
			if ansi.Width(r) > sz[0] {
				t.Errorf("%dx%d: a row is %d wide: %q", sz[0], sz[1], ansi.Width(r), r)
			}
		}
		s.Keys("enter", "down", "down", "x", "tab")
		s.Close()
	}
	small := tuitest.New(newDemoApp(demoEntries(), ""), 40, 10)
	defer small.Close()
	if got := strings.Join(strings.Fields(screenText(small)), " "); !strings.Contains(got, "needs a terminal of at least 60x14") {
		t.Errorf("a small terminal is not told:\n%s", screenText(small))
	}
}

func TestTheListScrollsToTheCursor(t *testing.T) {
	s := tuitest.New(newDemoApp(demoEntries(), ""), 100, 14)
	defer s.Close()
	s.Keys("end")
	got := screenText(s)
	entries := demoEntries()
	last := entries[len(entries)-1].name
	if !strings.Contains(got, "> "+last) || strings.Contains(got, "  Button\n") {
		t.Errorf("End did not scroll to the last widget:\n%s", got)
	}
	s.Keys("home")
	if !strings.Contains(screenText(s), "> Button") {
		t.Error("Home did not scroll back")
	}
}

func TestDemoCommand(t *testing.T) {
	var out bytes.Buffer
	if err := run([]string{"demo", "-list"}, &out); err != nil {
		t.Fatal(err)
	}
	for _, e := range demoEntries() {
		if !strings.Contains(out.String(), e.name) {
			t.Errorf("-list lacks %q", e.name)
		}
	}
	if err := run([]string{"demo", "nosuchwidget"}, &out); err == nil || !strings.Contains(err.Error(), "no widget called") {
		t.Errorf("an unknown widget: %v", err)
	}
	if err := run([]string{"demo", "a", "b"}, &out); err == nil {
		t.Error("two widget names were accepted")
	}
	if err := run([]string{"demo", "-nope"}, &out); err == nil {
		t.Error("an unknown flag was accepted")
	}
}

func TestEveryEntryDrawsACleanFrame(t *testing.T) {
	for _, e := range demoEntries() {
		s := tuitest.New(newDemoApp(demoEntries(), e.name), 100, 28)
		for _, keys := range [][]string{nil, {"enter"}, {"right", "down", "enter", "x"}} {
			s.Keys(keys...)
			rows := s.Screen()
			if len(rows) != 28 {
				t.Errorf("%s: %d rows", e.name, len(rows))
			}
			for _, r := range rows {
				if ansi.Width(r) > 100 || strings.ContainsAny(r, "\t\x1b") {
					t.Errorf("%s: a bad row after %v: %q", e.name, keys, r)
					break
				}
			}
			// The right border of the panes stays in one column: every row of the box region ends the same way.
			if got := rows[1]; !strings.HasSuffix(strings.TrimRight(got, " "), "│") {
				t.Errorf("%s: the first row of the panes does not end in a border: %q", e.name, got)
			}
			for i := 1; i < 26; i++ {
				if end := strings.TrimRight(rows[i], " "); !strings.HasSuffix(end, "│") && !strings.HasSuffix(end, "┤") {
					t.Errorf("%s: row %d lost its right border: %q", e.name, i, rows[i])
					break
				}
			}
		}
		s.Close()
	}
}

func TestTheSpinnerAnimatesAndTheThemeReachesTheWidget(t *testing.T) {
	s := tuitest.New(newDemoApp(demoEntries(), "Spinner"), 100, 28)
	defer s.Close()
	first := screenText(s)
	if !strings.Contains(first, "Loading") {
		t.Fatalf("no spinner:\n%s", first)
	}
	deadline := time.Now().Add(3 * time.Second)
	for screenText(s) == first {
		if time.Now().After(deadline) {
			t.Fatal("the spinner never moved: its Start command did not run")
		}
		time.Sleep(20 * time.Millisecond)
	}

	app := newDemoApp(demoEntries(), "Button")
	before := app.live.View()
	for i := 0; i < len(app.themes) && app.live.View() == before; i++ {
		app.Update(tui.Key{Type: tui.KeyCtrl, Code: 't'})
	}
	if app.live.View() == before {
		t.Error("changing the theme did not restyle the widget")
	}
}

func TestTheAvatarEntryChangesNameAndExpression(t *testing.T) {
	s := tuitest.New(newDemoApp(demoEntries(), "Avatar"), 100, 28)
	defer s.Close()
	first := screenText(s)
	if !strings.Contains(first, "ada") && !strings.Contains(first, "happy") && !strings.Contains(first, "round") && !strings.Contains(first, "organic") {
		t.Logf("first screen:\n%s", first)
	}
	if !strings.Contains(first, "avatar.New") {
		t.Fatalf("the Avatar entry is not shown:\n%s", first)
	}
	if got := avatarKeys(0, tui.Key{Type: tui.KeyRight}); got != 1 {
		t.Errorf("Right moved to name %d", got)
	}
	if got := avatarKeys(0, tui.Key{Type: tui.KeyLeft}); got != len(avatarNames)-1 {
		t.Errorf("Left from the first name went to %d", got)
	}
	if got := avatarKeys(5, tui.Key{Type: tui.KeyRunes, Text: "e"}); got != 105 {
		t.Errorf("E gave state %d, want 105", got)
	}
}
