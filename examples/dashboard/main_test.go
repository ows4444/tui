package main

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
)

func isQuit(cmd tui.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tui.QuitMsg)
	return ok
}

func runes(r rune) tui.Key { return tui.Key{Type: tui.KeyRunes, Text: string([]rune{r})} }

func TestViewFitsAnEightyByTwentyFourTerminal(t *testing.T) {
	out := initialModel().View()
	plain := ansi.StripANSI(out)
	for _, want := range []string{"Service Dashboard", "API", "Database", "CPU", "refresh", "quit"} {
		if !strings.Contains(plain, want) {
			t.Errorf("View() missing %q", want)
		}
	}
	lines := strings.Split(out, "\n")
	if len(lines) > 24 {
		t.Errorf("View() is %d rows, want at most 24", len(lines))
	}
	for i, l := range lines {
		if w := ansi.Width(l); w > 80 {
			t.Errorf("row %d is %d wide, want at most 80", i, w)
		}
	}
}

func TestInitStartsTheAnimations(t *testing.T) {
	if initialModel().Init() == nil {
		t.Error("Init() returned no Cmd, want the spinner and loading bar ticks")
	}
}

func TestQuitKeys(t *testing.T) {
	for name, k := range map[string]tui.Key{
		"ctrl+c": {Type: tui.KeyCtrlC},
		"esc":    {Type: tui.KeyEsc},
		"q":      runes('q'),
	} {
		if _, cmd := initialModel().Update(k); !isQuit(cmd) {
			t.Errorf("%s did not quit", name)
		}
	}
	if _, cmd := initialModel().Update(tui.Key{Type: tui.KeyRunes}); cmd != nil {
		t.Error("an empty rune key returned a Cmd")
	}
}

func TestRefreshReplacesSamplesAndShowsAToast(t *testing.T) {
	m := initialModel()
	oldLast, oldDisk := m.cpu[len(m.cpu)-1], m.disk
	next, cmd := m.Update(runes('r'))
	got := next.(model)
	if cmd == nil {
		t.Error("refresh returned no Cmd for the toast")
	}
	if len(got.cpu) != 20 || len(got.mem) != 20 {
		t.Errorf("sample windows are %d/%d long, want 20", len(got.cpu), len(got.mem))
	}
	if got.cpu[len(got.cpu)-1] == oldLast || got.disk == oldDisk {
		t.Error("refresh did not draw new samples")
	}
	if !strings.Contains(ansi.StripANSI(got.View()), "Refreshed!") {
		t.Error("View() after refresh missing the toast")
	}
}

func TestAboutDialogIsModal(t *testing.T) {
	m := initialModel()
	next, _ := m.Update(runes('d'))
	m = next.(model)
	if !m.about.Open() || !strings.Contains(ansi.StripANSI(m.View()), "About") {
		t.Fatal("d did not open the About dialog")
	}
	// While open, q must not quit.
	next, cmd := m.Update(runes('q'))
	if isQuit(cmd) {
		t.Error("q quit while the dialog was open")
	}
	m = next.(model)
	// Ctrl+C still force-quits.
	if _, cmd := m.Update(tui.Key{Type: tui.KeyCtrlC}); !isQuit(cmd) {
		t.Error("ctrl+c did not quit while the dialog was open")
	}
	// Esc closes it.
	next, _ = m.Update(tui.Key{Type: tui.KeyEsc})
	if next.(model).about.Open() {
		t.Error("Esc did not close the dialog")
	}
}

func TestNonKeyMessagesReachTheAnimatedWidgets(t *testing.T) {
	// Unknown messages are forwarded to spinner, loadingbar and toast;
	// none of them may panic or quit.
	next, cmd := initialModel().Update(tui.ResizeMsg{Width: 80, Height: 24})
	if next == nil || isQuit(cmd) {
		t.Error("a ResizeMsg broke the model")
	}
}
