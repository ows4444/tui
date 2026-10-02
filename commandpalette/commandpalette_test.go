package commandpalette

import (
	"reflect"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

func key(t tui.KeyType) tui.Key { return tui.Key{Type: t} }

func typeString(m *Model, s string) {
	for _, r := range s {
		next, _ := m.Update(tui.Key{Type: tui.KeyRunes, Text: string([]rune{r})})
		*m = next
	}
}

func cmds(names ...string) []Command {
	out := make([]Command, len(names))
	for i, n := range names {
		out[i] = Command{Name: n}
	}
	return out
}

func TestNewFocusesInput(t *testing.T) {
	m := New(Command{Name: "Open File"})
	if !m.Input.Focused() {
		t.Error("New() should focus Input so typing works immediately")
	}
}

func TestInitReturnsBlinkCmd(t *testing.T) {
	m := New(Command{Name: "Open File"})
	if cmd := m.Init(); cmd == nil {
		t.Error("Init() should return a non-nil Cmd (starts the cursor blink)")
	}
}

func TestFilteredEmptyQuery(t *testing.T) {
	m := New(cmds("Open File", "Close File")...)
	if got := m.filtered(); got != nil {
		t.Errorf("filtered() with empty query = %v, want nil", got)
	}
}

// #538: fuzzy subsequence matching — every rune of the query appears in
// order within the command name, not necessarily contiguously.
func TestFilteredFuzzySubsequenceMatch(t *testing.T) {
	m := New(cmds("CommandPalette", "Copy Path", "Close File")...)
	typeString(&m, "cp")
	got := m.filtered()
	want := []Command{{Name: "CommandPalette"}, {Name: "Copy Path"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("filtered() = %v, want %v", got, want)
	}
}

func TestFilteredFuzzyMatchNonContiguous(t *testing.T) {
	m := New(cmds("Save As...", "Save", "Quit")...)
	typeString(&m, "sa")
	got := m.filtered()
	want := []Command{{Name: "Save As..."}, {Name: "Save"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("filtered() = %v, want %v", got, want)
	}
}

func TestFilteredFuzzyMatchCaseInsensitive(t *testing.T) {
	m := New(cmds("CommandPalette")...)
	typeString(&m, "CP")
	got := m.filtered()
	want := []Command{{Name: "CommandPalette"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("filtered() = %v, want %v", got, want)
	}
}

func TestFilteredNoMatches(t *testing.T) {
	m := New(cmds("Open File", "Close File")...)
	typeString(&m, "zzz")
	if got := m.filtered(); got != nil {
		t.Errorf("filtered() with no matches = %v, want nil", got)
	}
}

func TestFilteredRequiresInOrder(t *testing.T) {
	// "pc" is not a subsequence of "CommandPalette" (P comes before the
	// second C, not after), so it should not match.
	m := New(cmds("CommandPalette")...)
	typeString(&m, "pc")
	if got := m.filtered(); got != nil {
		t.Errorf("filtered() = %v, want nil (out-of-order query should not match)", got)
	}
}

func TestIsOpen(t *testing.T) {
	m := New(cmds("Open File", "Close File")...)
	if m.IsOpen() {
		t.Error("IsOpen() with an empty query should be false")
	}
	typeString(&m, "o")
	if !m.IsOpen() {
		t.Error("IsOpen() with matches should be true")
	}
	next, _ := m.Update(key(tui.KeyEsc))
	if next.IsOpen() {
		t.Error("IsOpen() after Esc should be false")
	}
}

func TestFocusAndBlur(t *testing.T) {
	m := New(Command{Name: "Open File"})
	m.Blur()
	if m.Input.Focused() {
		t.Error("Blur() should un-focus Input")
	}
	m.Focus()
	if !m.Input.Focused() {
		t.Error("Focus() should focus Input")
	}
}

func TestNonKeyMsgForwardedToInput(t *testing.T) {
	m := New(cmds("Open File", "Close File")...)
	next, _ := m.Update(tui.PasteEvent{Text: "hi"})
	m = next
	if m.Input.Value() != "hi" {
		t.Errorf("a non-Key Msg should be forwarded to Input; Value() = %q, want %q", m.Input.Value(), "hi")
	}
}

// #539: Up/Down move the highlight among filtered results while the
// dropdown is open, the same shape as autocomplete.Model's handling.
func TestHighlightNavigationClamps(t *testing.T) {
	m := New(cmds("apple", "apricot", "avocado")...)
	typeString(&m, "a") // 3 matches, highlight starts at 0

	next, _ := m.Update(key(tui.KeyDown))
	m = next
	if m.highlight != 1 {
		t.Fatalf("after Down: highlight = %d, want 1", m.highlight)
	}
	next, _ = m.Update(key(tui.KeyDown))
	m = next
	next, _ = m.Update(key(tui.KeyDown)) // one past the end
	m = next
	if m.highlight != 2 {
		t.Fatalf("Down past the end: highlight = %d, want 2 (clamped)", m.highlight)
	}

	next, _ = m.Update(key(tui.KeyUp))
	m = next
	if m.highlight != 1 {
		t.Fatalf("after Up: highlight = %d, want 1", m.highlight)
	}
	m.highlight = 0
	next, _ = m.Update(key(tui.KeyUp)) // one past the start
	m = next
	if m.highlight != 0 {
		t.Fatalf("Up past the start: highlight = %d, want 0 (clamped)", m.highlight)
	}
}

func TestUpDownAreNoOpsWhenDropdownClosed(t *testing.T) {
	m := New(Command{Name: "apple"}) // empty query, dropdown closed
	next, cmd := m.Update(key(tui.KeyDown))
	if cmd != nil {
		t.Error("Down with a closed dropdown should not emit a Cmd")
	}
	if next.highlight != 0 {
		t.Error("Down with a closed dropdown should not move the highlight")
	}
}

// #540: Enter on the highlighted filtered command emits SelectedMsg
// without filling the input value.
func TestEnterEmitsSelectedMsgWithoutFillingInput(t *testing.T) {
	m := New(cmds("Open File", "Open Recent")...)
	typeString(&m, "op") // highlight 0 -> "Open File"

	next, cmd := m.Update(key(tui.KeyEnter))
	m = next
	if m.Input.Value() != "op" {
		t.Errorf("Value() = %q, want %q (Enter should not fill the input)", m.Input.Value(), "op")
	}
	if cmd == nil {
		t.Fatal("Enter should return a non-nil Cmd")
	}
	msg, ok := cmd().(SelectedMsg)
	if !ok || msg.Command.Name != "Open File" {
		t.Errorf("Cmd produced %#v, want SelectedMsg{Command: {Name: %q}}", cmd(), "Open File")
	}
	if got := m.filtered(); got != nil {
		t.Errorf("filtered() right after select = %v, want nil (dropdown closed)", got)
	}
}

func TestEnterSelectsHighlightedCommand(t *testing.T) {
	m := New(cmds("Open File", "Open Recent")...)
	typeString(&m, "op")
	next, _ := m.Update(key(tui.KeyDown)) // highlight "Open Recent"
	m = next

	next, cmd := m.Update(key(tui.KeyEnter))
	m = next
	msg := cmd().(SelectedMsg)
	if msg.Command.Name != "Open Recent" {
		t.Errorf("SelectedMsg.Command.Name = %q, want %q", msg.Command.Name, "Open Recent")
	}
	_ = next
}

func TestDropdownReopensAfterTypingPostSelect(t *testing.T) {
	m := New(cmds("Open File", "Open Recent", "Close File")...)
	typeString(&m, "op")
	next, _ := m.Update(key(tui.KeyEnter)) // selects "Open File"
	m = next
	if got := m.filtered(); got != nil {
		t.Fatalf("filtered() right after select = %v, want nil", got)
	}

	typeString(&m, "e") // Value becomes "ope"
	got := m.filtered()
	want := []Command{{Name: "Open File"}, {Name: "Open Recent"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("filtered() after typing post-select = %v, want %v", got, want)
	}
}

// #541: Esc closes the dropdown, matching autocomplete.Model's convention.
func TestEscClosesDropdownWithoutChangingValue(t *testing.T) {
	m := New(cmds("Open File", "Open Recent")...)
	typeString(&m, "op")

	next, cmd := m.Update(key(tui.KeyEsc))
	m = next
	if cmd != nil {
		t.Error("Esc should not emit a Cmd")
	}
	if m.Input.Value() != "op" {
		t.Errorf("Esc should not change the input value, got %q", m.Input.Value())
	}
	if got := m.filtered(); got != nil {
		t.Errorf("filtered() after Esc = %v, want nil (dropdown closed)", got)
	}
}

func TestEscIsNoOpWhenDropdownAlreadyClosed(t *testing.T) {
	m := New(Command{Name: "Open File"}) // empty query, dropdown already closed
	_, cmd := m.Update(key(tui.KeyEsc))
	if cmd != nil {
		t.Error("Esc with nothing to close should not emit a Cmd")
	}
}

func TestViewWithNoDropdownMatchesInputView(t *testing.T) {
	m := New(cmds("Open File", "Close File")...)
	if got, want := m.View(), m.Input.View(); got != want {
		t.Errorf("View() = %q, want %q (should equal Input.View() with no commands)", got, want)
	}
}

func TestViewWithDropdown(t *testing.T) {
	m := New(cmds("Open File", "Open Recent")...)
	m.Theme = theme.DarkTheme()
	typeString(&m, "op")

	dim := ansi.NewStyle().Faint()
	highlight := theme.DarkTheme().ResolvedStates().Selected.Bold()
	want := m.Input.View() + "\n" + highlight.Render("> Open File") + "\n" + dim.Render("  Open Recent")

	got := m.View()
	if got != want {
		t.Errorf("View() = %q, want %q", got, want)
	}
}

func TestViewIncludesDescription(t *testing.T) {
	m := New(Command{Name: "Open File", Description: "Open a file from disk"})
	typeString(&m, "op")

	dim := ansi.NewStyle().Faint()
	highlight := m.Theme.ResolvedStates().Selected.Bold()
	want := m.Input.View() + "\n" + highlight.Render("> Open File — Open a file from disk")
	_ = dim

	got := m.View()
	if got != want {
		t.Errorf("View() = %q, want %q", got, want)
	}
}
