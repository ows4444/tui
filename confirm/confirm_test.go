package confirm

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/theme"
)

func key(t tui.KeyType) tui.Key { return tui.Key{Type: t} }
func rk(r rune) tui.Key         { return tui.Key{Type: tui.KeyRunes, Text: string([]rune{r})} }

func TestNewDefaultsToYesHighlighted(t *testing.T) {
	m := New("Proceed?")
	if !m.Highlighted() {
		t.Error("New() should highlight Yes by default")
	}
}

func TestLeftRightTabToggleHighlight(t *testing.T) {
	m := New("Proceed?")
	for _, k := range []tui.KeyType{tui.KeyLeft, tui.KeyRight, tui.KeyTab} {
		before := m.Highlighted()
		next, cmd := m.Update(key(k))
		m = next
		if cmd != nil {
			t.Errorf("%v should not emit a Cmd, just toggle highlight", k)
		}
		if m.Highlighted() == before {
			t.Errorf("%v should toggle Highlighted() (was %v, still %v)", k, before, m.Highlighted())
		}
	}
}

func TestEnterConfirmsHighlighted(t *testing.T) {
	m := New("Proceed?") // Yes highlighted
	_, cmd := m.Update(key(tui.KeyEnter))
	msg, ok := cmd().(ConfirmedMsg)
	if !ok || !msg.Yes {
		t.Errorf("Enter with Yes highlighted should confirm Yes, got %+v", msg)
	}

	next, _ := m.Update(key(tui.KeyLeft)) // toggle to No
	m = next
	_, cmd = m.Update(key(tui.KeyEnter))
	msg, ok = cmd().(ConfirmedMsg)
	if !ok || msg.Yes {
		t.Errorf("Enter with No highlighted should confirm No, got %+v", msg)
	}
}

func TestYNKeysConfirmDirectlyRegardlessOfHighlight(t *testing.T) {
	m := New("Proceed?") // Yes highlighted
	_, cmd := m.Update(rk('n'))
	msg, ok := cmd().(ConfirmedMsg)
	if !ok || msg.Yes {
		t.Errorf("'n' should confirm No even though Yes is highlighted, got %+v", msg)
	}

	_, cmd = m.Update(rk('Y'))
	msg, ok = cmd().(ConfirmedMsg)
	if !ok || !msg.Yes {
		t.Errorf("'Y' should confirm Yes, got %+v", msg)
	}
}

func TestUnrelatedRuneIsNoOp(t *testing.T) {
	m := New("Proceed?")
	next, cmd := m.Update(rk('x'))
	if cmd != nil {
		t.Error("an unrelated rune should not emit a Cmd")
	}
	if next.Highlighted() != m.Highlighted() {
		t.Error("an unrelated rune should not change the highlight")
	}
}

func TestNonKeyMsgIgnored(t *testing.T) {
	m := New("Proceed?")
	next, cmd := m.Update(tui.ResizeMsg{Width: 10, Height: 10})
	if cmd != nil || next.Highlighted() != m.Highlighted() {
		t.Error("a non-Key Msg should be a no-op")
	}
}

func TestView(t *testing.T) {
	m := New("Proceed?")
	m.Theme = theme.DarkTheme()
	highlight := theme.DarkTheme().ResolvedStates().Selected.Bold()

	got := m.View()
	want := "Proceed?  " + highlight.Render("[Yes]") + "   No "
	if got != want {
		t.Errorf("View() = %q, want %q", got, want)
	}

	next, _ := m.Update(key(tui.KeyTab))
	m = next
	got = m.View()
	want = "Proceed?  " + " Yes " + "  " + highlight.Render("[No]")
	if got != want {
		t.Errorf("View() after Tab = %q, want %q", got, want)
	}
}

func TestCustomLabels(t *testing.T) {
	m := New("Delete file?")
	m.YesLabel, m.NoLabel = "Delete", "Cancel"
	got := m.View()
	if got[:len("Delete file?")] != "Delete file?" {
		t.Fatalf("View() = %q, want it to start with the prompt", got)
	}
	// Just confirm the custom labels show up somewhere in the rendered
	// output; exact spacing is already covered by TestView.
	if !contains(got, "Delete") || !contains(got, "Cancel") {
		t.Errorf("View() = %q, want it to contain custom labels %q and %q", got, "Delete", "Cancel")
	}
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
