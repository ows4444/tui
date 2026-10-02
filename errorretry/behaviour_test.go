package errorretry

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/theme"
)

func TestSetThemeReplacesTheme(t *testing.T) {
	m := New("boom", 1).SetTheme(theme.LightTheme())
	if m.Theme != theme.LightTheme() {
		t.Fatalf("Theme = %+v, want theme.Light", m.Theme)
	}
}

func TestNonKeyAndEmptyRuneMsgsAreInert(t *testing.T) {
	m := New("boom", 2)
	if got, cmd := m.Update(struct{}{}); cmd != nil || got.RetryCount() != 0 {
		t.Error("non-key Msg had an effect")
	}
	if got, cmd := m.Update(tui.Key{Type: tui.KeyRunes}); cmd != nil || got.RetryCount() != 0 {
		t.Error("KeyRunes with no runes had an effect")
	}
	if got, cmd := m.Update(tui.Key{Type: tui.KeyRunes, Text: "x"}); cmd != nil || got.RetryCount() != 0 {
		t.Error("an unrelated rune had an effect")
	}
}

func TestRetryKeysStopAtMaxRetriesButEscStillDismisses(t *testing.T) {
	m := New("boom", 2)
	m, cmd := m.Update(tui.Key{Type: tui.KeyRunes, Text: "R"})
	if _, ok := cmd().(RetryMsg); !ok {
		t.Fatal("capital R did not retry")
	}
	m, cmd = m.Update(tui.Key{Type: tui.KeyEnter})
	if _, ok := cmd().(RetryMsg); !ok || !m.Exhausted() {
		t.Fatal("Enter did not use the last retry")
	}
	if _, cmd = m.Update(tui.Key{Type: tui.KeyEnter}); cmd != nil {
		t.Error("Enter retried after exhaustion")
	}
	if _, cmd = m.Update(tui.Key{Type: tui.KeyEsc}); cmd == nil {
		t.Fatal("Esc did not dismiss")
	} else if _, ok := cmd().(DismissedMsg); !ok {
		t.Error("Esc did not deliver DismissedMsg")
	}
}
