package appshell

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

// TestNew_DefaultsThemeToDark proves: New must not
// leave Theme at its zero value, since a zero theme.Theme renders with
// zero-value (invisible) colors.
func TestNew_DefaultsThemeToDark(t *testing.T) {
	m := New("My App", 20, 3)

	if m.Theme != theme.DarkTheme() {
		t.Fatalf("New's default Theme = %+v, want theme.DarkTheme()", m.Theme)
	}
}

// TestView_ComposesExistingWidgets proves criterion #546: View joins
// Header, Input.View, Content.View and (when present) a KeyHints footer via
// layout.JoinVertical, reusing those widgets' own rendering rather than
// reimplementing it.
func TestView_ComposesExistingWidgets(t *testing.T) {
	m := New("My App", 20, 3)
	m.Content.SetContent("hello\nworld")
	m.Hints = []widgets.Hint{{Key: "enter", Action: "submit"}, {Key: "q", Action: "quit"}}

	got := m.View()

	wantHeader := widgets.Header("My App", m.Theme)
	if !strings.Contains(got, wantHeader) {
		t.Errorf("View() = %q, want it to contain Header render %q", got, wantHeader)
	}
	if !strings.Contains(got, m.Input.View()) {
		t.Errorf("View() = %q, want it to contain Input.View() %q", got, m.Input.View())
	}
	if !strings.Contains(got, "hello") || !strings.Contains(got, "world") {
		t.Errorf("View() = %q, want it to contain Content's rendered lines", got)
	}
	wantHints := widgets.KeyHints(" ", m.Hints...)
	if !strings.Contains(got, wantHints) {
		t.Errorf("View() = %q, want it to contain KeyHints render %q", got, wantHints)
	}

	// Header should come before Input, which should come before Content,
	// which should come before the hints footer, matching the specified
	// join order.
	iHeader := strings.Index(got, wantHeader)
	iInput := strings.Index(got, m.Input.View())
	iContent := strings.Index(got, "hello")
	iHints := strings.Index(got, wantHints)
	if !(iHeader < iInput && iInput < iContent && iContent < iHints) {
		t.Errorf("View() blocks out of order: header=%d input=%d content=%d hints=%d", iHeader, iInput, iContent, iHints)
	}
}

// TestUpdate_EnterEmitsSubmitAndClears proves criterion #547: pressing
// Enter emits a SubmitMsg carrying the Input's value via a Cmd, and clears
// the Input.
func TestUpdate_EnterEmitsSubmitAndClears(t *testing.T) {
	m := New("App", 20, 3)
	m.Input.SetValue("hello world")

	updated, cmd := m.Update(tui.Key{Type: tui.KeyEnter})

	if cmd == nil {
		t.Fatal("Update() on Enter returned a nil Cmd, want one delivering SubmitMsg")
	}
	msg := cmd()
	submit, ok := msg.(SubmitMsg)
	if !ok {
		t.Fatalf("Cmd() = %#v (%T), want SubmitMsg", msg, msg)
	}
	if submit.Value != "hello world" {
		t.Errorf("SubmitMsg.Value = %q, want %q", submit.Value, "hello world")
	}
	if got := updated.Input.Value(); got != "" {
		t.Errorf("Input.Value() after Enter = %q, want empty (cleared)", got)
	}
}

// TestUpdate_OtherKeyForwardsToInput proves criterion #548: a non-Enter key
// is forwarded to Input.Update, updating its value normally.
func TestUpdate_OtherKeyForwardsToInput(t *testing.T) {
	m := New("App", 20, 3)

	m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: "hi"})

	if got := m.Input.Value(); got != "hi" {
		t.Errorf("Input.Value() after forwarded rune key = %q, want %q", got, "hi")
	}

	m, _ = m.Update(tui.Key{Type: tui.KeyBackspace})
	if got := m.Input.Value(); got != "h" {
		t.Errorf("Input.Value() after forwarded backspace = %q, want %q", got, "h")
	}
}

// TestView_NoHintsOmitsFooter proves criterion #549: with no Hints, the
// hints footer row is omitted entirely rather than rendered as a stray
// empty line.
func TestView_NoHintsOmitsFooter(t *testing.T) {
	withHints := New("App", 20, 3)
	withHints.Hints = []widgets.Hint{{Key: "q", Action: "quit"}}

	withoutHints := New("App", 20, 3)
	withoutHints.Hints = nil

	gotWith := withHints.View()
	gotWithout := withoutHints.View()

	linesWith := strings.Split(gotWith, "\n")
	linesWithout := strings.Split(gotWithout, "\n")

	if len(linesWithout) != len(linesWith)-1 {
		t.Errorf("View() line count without Hints = %d, with Hints = %d, want exactly one fewer line without Hints", len(linesWithout), len(linesWith))
	}

	if strings.Contains(gotWithout, "[q] quit") {
		t.Errorf("View() without Hints = %q, want it not to contain a rendered hint", gotWithout)
	}
}
