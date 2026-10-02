package autocomplete

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

func TestNewFocusesInput(t *testing.T) {
	m := New("apple")
	if !m.Input.Focused() {
		t.Error("New() should focus Input so typing works immediately")
	}
}

func TestInitReturnsBlinkCmd(t *testing.T) {
	m := New("apple")
	if cmd := m.Init(); cmd == nil {
		t.Error("Init() should return a non-nil Cmd (starts the cursor blink)")
	}
}

func TestFilteredEmptyQuery(t *testing.T) {
	m := New("apple", "banana")
	if got := m.filtered(); got != nil {
		t.Errorf("filtered() with empty query = %v, want nil", got)
	}
}

func TestFilteredPrefixMatchCaseInsensitive(t *testing.T) {
	m := New("Apple", "Apricot", "Banana")
	typeString(&m, "ap")
	got := m.filtered()
	want := []string{"Apple", "Apricot"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("filtered() = %v, want %v", got, want)
	}
}

func TestFilteredNoMatches(t *testing.T) {
	m := New("apple", "banana")
	typeString(&m, "zzz")
	if got := m.filtered(); got != nil {
		t.Errorf("filtered() with no matches = %v, want nil", got)
	}
}

func TestIsOpen(t *testing.T) {
	m := New("apple", "banana")
	if m.IsOpen() {
		t.Error("IsOpen() with an empty query should be false")
	}
	typeString(&m, "ap")
	if !m.IsOpen() {
		t.Error("IsOpen() with matches should be true")
	}
	next, _ := m.Update(key(tui.KeyEsc))
	if next.IsOpen() {
		t.Error("IsOpen() after Esc should be false")
	}
}

func TestFocusAndBlur(t *testing.T) {
	m := New("apple")
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
	m := New("apple", "banana")
	next, _ := m.Update(tui.PasteEvent{Text: "hi"})
	m = next
	if m.Input.Value() != "hi" {
		t.Errorf("a non-Key Msg should be forwarded to Input; Value() = %q, want %q", m.Input.Value(), "hi")
	}
}

func TestHighlightNavigationClamps(t *testing.T) {
	m := New("apple", "apricot", "avocado")
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
	m := New("apple") // empty query, dropdown closed
	next, cmd := m.Update(key(tui.KeyDown))
	if cmd != nil {
		t.Error("Down with a closed dropdown should not emit a Cmd")
	}
	if next.highlight != 0 {
		t.Error("Down with a closed dropdown should not move the highlight")
	}
}

func TestTabAcceptsHighlightedSuggestion(t *testing.T) {
	m := New("apple", "apricot")
	typeString(&m, "ap") // highlight 0 -> "apple"

	next, cmd := m.Update(key(tui.KeyTab))
	m = next
	if m.Input.Value() != "apple" {
		t.Errorf("Value() = %q, want %q", m.Input.Value(), "apple")
	}
	if cmd == nil {
		t.Fatal("Tab should return a non-nil Cmd")
	}
	msg, ok := cmd().(AcceptedMsg)
	if !ok || msg.Value != "apple" {
		t.Errorf("Cmd produced %#v, want AcceptedMsg{Value: %q}", cmd(), "apple")
	}
	if got := m.filtered(); got != nil {
		t.Errorf("filtered() right after accept = %v, want nil (dropdown closed)", got)
	}
}

func TestEnterAcceptsHighlightedSuggestion(t *testing.T) {
	m := New("apple", "apricot")
	typeString(&m, "ap")
	next, _ := m.Update(key(tui.KeyDown)) // highlight "apricot"
	m = next

	next, cmd := m.Update(key(tui.KeyEnter))
	m = next
	if m.Input.Value() != "apricot" {
		t.Errorf("Value() = %q, want %q", m.Input.Value(), "apricot")
	}
	msg := cmd().(AcceptedMsg)
	if msg.Value != "apricot" {
		t.Errorf("AcceptedMsg.Value = %q, want %q", msg.Value, "apricot")
	}
}

func TestDropdownReopensAfterTypingPostAccept(t *testing.T) {
	m := New("apple", "applet", "apricot")
	typeString(&m, "ap")
	next, _ := m.Update(key(tui.KeyTab)) // accepts "apple"
	m = next
	if got := m.filtered(); got != nil {
		t.Fatalf("filtered() right after accept = %v, want nil", got)
	}

	typeString(&m, "t") // Value becomes "applet"
	got := m.filtered()
	want := []string{"applet"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("filtered() after typing post-accept = %v, want %v", got, want)
	}
}

func TestEscClosesDropdownWithoutChangingValue(t *testing.T) {
	m := New("apple", "apricot")
	typeString(&m, "ap")

	next, cmd := m.Update(key(tui.KeyEsc))
	m = next
	if cmd != nil {
		t.Error("Esc should not emit a Cmd")
	}
	if m.Input.Value() != "ap" {
		t.Errorf("Esc should not change the input value, got %q", m.Input.Value())
	}
	if got := m.filtered(); got != nil {
		t.Errorf("filtered() after Esc = %v, want nil (dropdown closed)", got)
	}
}

func TestEscIsNoOpWhenDropdownAlreadyClosed(t *testing.T) {
	m := New("apple") // empty query, dropdown already closed
	_, cmd := m.Update(key(tui.KeyEsc))
	if cmd != nil {
		t.Error("Esc with nothing to close should not emit a Cmd")
	}
}

func TestViewWithNoDropdownMatchesInputView(t *testing.T) {
	m := New("apple", "banana")
	if got, want := m.View(), m.Input.View(); got != want {
		t.Errorf("View() = %q, want %q (should equal Input.View() with no suggestions)", got, want)
	}
}

func TestViewWithDropdown(t *testing.T) {
	m := New("apple", "apricot")
	m.Theme = theme.DarkTheme()
	typeString(&m, "ap")

	dim := ansi.NewStyle().Faint()
	highlight := theme.DarkTheme().ResolvedStates().Selected.Bold()
	want := m.Input.View() + "\n" + highlight.Render("> apple") + "\n" + dim.Render("  apricot")

	got := m.View()
	if got != want {
		t.Errorf("View() = %q, want %q", got, want)
	}
}
