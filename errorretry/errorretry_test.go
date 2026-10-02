package errorretry

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
)

func key(t tui.KeyType) tui.Key { return tui.Key{Type: t} }
func rk(r rune) tui.Key         { return tui.Key{Type: tui.KeyRunes, Text: string([]rune{r})} }

// #509: View shows the given error message.
func TestViewShowsMessage(t *testing.T) {
	m := New("connection failed", 3)
	view := m.View()
	if !strings.Contains(view, "connection failed") {
		t.Errorf("View() = %q, want it to contain the error message", view)
	}
}

// #510: Enter or 'r', while retries remain, increments retryCount and
// returns a Cmd delivering RetryMsg.
func TestRetryKeysIncrementAndEmitRetryMsg(t *testing.T) {
	tests := []struct {
		name string
		key  tui.Key
	}{
		{"Enter", key(tui.KeyEnter)},
		{"lowercase r", rk('r')},
		{"uppercase R", rk('R')},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New("boom", 3)
			next, cmd := m.Update(tt.key)
			if next.RetryCount() != 1 {
				t.Errorf("retryCount = %d, want 1", next.RetryCount())
			}
			if cmd == nil {
				t.Fatal("expected a Cmd, got nil")
			}
			msg := cmd()
			if _, ok := msg.(RetryMsg); !ok {
				t.Errorf("Cmd delivered %#v, want RetryMsg{}", msg)
			}
		})
	}
}

func TestRetryIncrementsAcrossMultiplePresses(t *testing.T) {
	m := New("boom", 3)
	for i := 1; i <= 3; i++ {
		next, cmd := m.Update(key(tui.KeyEnter))
		m = next
		if m.RetryCount() != i {
			t.Fatalf("after %d presses, retryCount = %d, want %d", i, m.RetryCount(), i)
		}
		if cmd == nil {
			t.Fatalf("press %d: expected a Cmd, got nil", i)
		}
		if _, ok := cmd().(RetryMsg); !ok {
			t.Fatalf("press %d: expected RetryMsg", i)
		}
	}
}

// #511: once retryCount reaches maxRetries, further Enter/'r' presses are
// a no-op (retryCount doesn't increment, no RetryMsg), while Esc still
// dismisses.
func TestRetryDisabledOnceExhausted(t *testing.T) {
	m := New("boom", 2)
	m, _ = m.Update(key(tui.KeyEnter))
	m, _ = m.Update(key(tui.KeyEnter))
	if m.RetryCount() != 2 {
		t.Fatalf("retryCount = %d, want 2", m.RetryCount())
	}
	if !m.Exhausted() {
		t.Fatal("expected Exhausted() to be true once retryCount == MaxRetries")
	}

	for _, k := range []tui.Key{key(tui.KeyEnter), rk('r')} {
		next, cmd := m.Update(k)
		if next.RetryCount() != 2 {
			t.Errorf("retryCount after exhausted press = %d, want unchanged 2", next.RetryCount())
		}
		if cmd != nil {
			t.Errorf("expected no Cmd once exhausted, got one that delivers %#v", cmd())
		}
	}
}

func TestEscStillDismissesOnceExhausted(t *testing.T) {
	m := New("boom", 1)
	m, _ = m.Update(key(tui.KeyEnter))
	if !m.Exhausted() {
		t.Fatal("expected exhausted after 1 retry with MaxRetries=1")
	}

	_, cmd := m.Update(key(tui.KeyEsc))
	if cmd == nil {
		t.Fatal("expected Esc to still emit a Cmd once exhausted")
	}
	if _, ok := cmd().(DismissedMsg); !ok {
		t.Errorf("Cmd delivered %#v, want DismissedMsg{}", cmd())
	}
}

// #512: Esc at any retryCount returns a Cmd delivering DismissedMsg.
func TestEscDismissesAtAnyRetryCount(t *testing.T) {
	for retryCount := 0; retryCount <= 3; retryCount++ {
		m := New("boom", 3)
		for i := 0; i < retryCount; i++ {
			m, _ = m.Update(key(tui.KeyEnter))
		}

		_, cmd := m.Update(key(tui.KeyEsc))
		if cmd == nil {
			t.Fatalf("retryCount=%d: expected a Cmd from Esc, got nil", retryCount)
		}
		if _, ok := cmd().(DismissedMsg); !ok {
			t.Errorf("retryCount=%d: Cmd delivered %#v, want DismissedMsg{}", retryCount, cmd())
		}
	}
}

// #513: View visibly reflects the retry-exhausted state once retryCount
// reaches maxRetries, distinct from the normal retries-remaining state.
func TestViewDiffersOnceExhausted(t *testing.T) {
	m := New("boom", 2)
	before := m.View()

	m, _ = m.Update(key(tui.KeyEnter))
	m, _ = m.Update(key(tui.KeyEnter))
	after := m.View()

	if before == after {
		t.Fatal("View() should differ once retries are exhausted, but it didn't change")
	}
	if strings.Contains(after, "press Enter or r to retry") {
		t.Errorf("exhausted View() still contains the normal retry hint: %q", after)
	}
	if !strings.Contains(after, "no retries left") {
		t.Errorf("exhausted View() = %q, want it to indicate no retries left", after)
	}
}

func TestOtherMsgIsNoOp(t *testing.T) {
	m := New("boom", 3)
	next, cmd := m.Update("not a key")
	if cmd != nil {
		t.Error("non-Key Msg should not produce a Cmd")
	}
	if next.RetryCount() != m.RetryCount() {
		t.Error("non-Key Msg should not change retryCount")
	}
}
