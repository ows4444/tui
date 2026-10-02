package clipboard

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/ows4444/tui"
)

// captureWrite returns a Write func that records every string passed to it,
// so tests can assert on the OSC52 bytes without touching a real terminal.
func captureWrite(writes *[]string) func(string) (int, error) {
	return func(s string) (int, error) {
		*writes = append(*writes, s)
		return len(s), nil
	}
}

func wantOSC52(text string) string {
	return "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(text)) + "\x07"
}

// TestActivationWritesOSC52 proves criterion #530: Enter or Space on a
// Clipboard Model writes an OSC52 clipboard-set escape sequence
// (base64-encoded per the OSC52 spec) for Text to the configured writer.
func TestActivationWritesOSC52(t *testing.T) {
	tests := []struct {
		name string
		key  tui.Key
	}{
		{"Enter", tui.Key{Type: tui.KeyEnter}},
		{"Space", tui.Key{Type: tui.KeySpace}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var writes []string
			m := New("hello world", "Copy")
			m.Write = captureWrite(&writes)

			m.Update(tt.key)

			if len(writes) != 1 {
				t.Fatalf("Write called %d times, want 1", len(writes))
			}
			if want := wantOSC52("hello world"); writes[0] != want {
				t.Errorf("Write got %q, want %q", writes[0], want)
			}
		})
	}
}

// TestActivationShowsCopiedConfirmation proves criterion #531: activation
// shows a "Copied!"-style confirmation in View in place of the normal label.
func TestActivationShowsCopiedConfirmation(t *testing.T) {
	var writes []string
	m := New("x", "Copy to clipboard")
	m.Write = captureWrite(&writes)

	if got := m.View(); got != "Copy to clipboard" {
		t.Fatalf("View() before activation = %q, want Label", got)
	}

	next, _ := m.Update(tui.Key{Type: tui.KeyEnter})
	if !next.Copied() {
		t.Fatal("Copied() should be true right after activation")
	}
	if got := next.View(); got == "Copy to clipboard" {
		t.Errorf("View() after activation = %q, want a distinct 'Copied!' confirmation", got)
	}
}

// TestTimeoutRevertsToLabel proves criterion #532: once the confirmation
// timeout elapses, View reverts to Label automatically, reusing toast's
// exact id-disambiguation pattern so repeated activations don't race.
func TestTimeoutRevertsToLabel(t *testing.T) {
	var writes []string
	m := New("x", "Copy")
	m.Write = captureWrite(&writes)
	m.Timeout = time.Millisecond

	next, cmd := m.Update(tui.Key{Type: tui.KeyEnter})
	if cmd == nil {
		t.Fatal("activation should return a non-nil Cmd")
	}
	msg, ok := tui.RunCmd(context.Background(), cmd).(dismissMsg)
	if !ok {
		t.Fatalf("activation's Cmd produced %T, want dismissMsg", tui.RunCmd(context.Background(), cmd))
	}
	if msg.id != next.id {
		t.Errorf("dismissMsg.id = %d, want %d (current generation)", msg.id, next.id)
	}

	next, _ = next.Update(msg)
	if next.Copied() {
		t.Error("a matching dismissMsg should clear the confirmation")
	}
	if got := next.View(); got != "Copy" {
		t.Errorf("View() after timeout = %q, want Label restored", got)
	}
}

// TestStaleDismissIgnoredAfterReactivation is the regression test for the
// id-disambiguation trick itself: if the clipboard is activated again before
// a previous timer fires, the earlier timer's dismissMsg must not clear the
// confirmation the second activation started.
func TestStaleDismissIgnoredAfterReactivation(t *testing.T) {
	var writes []string
	m := New("x", "Copy")
	m.Write = captureWrite(&writes)
	m.Timeout = time.Millisecond

	first, firstCmd := m.Update(tui.Key{Type: tui.KeyEnter})
	firstMsg := tui.RunCmd(context.Background(), firstCmd).(dismissMsg) // generation 1

	second, secondCmd := first.Update(tui.Key{Type: tui.KeyEnter}) // generation 2, before gen-1 timer fires
	if !second.Copied() {
		t.Fatal("confirmation should still be shown after re-activation")
	}

	afterStale, cmd := second.Update(firstMsg) // stale generation-1 dismiss arrives
	if !afterStale.Copied() {
		t.Fatal("a stale dismissMsg from a previous activation should not clear the confirmation")
	}
	if cmd != nil {
		t.Error("a stale dismissMsg should not return a Cmd")
	}

	secondMsg := tui.RunCmd(context.Background(), secondCmd).(dismissMsg) // generation 2
	final, _ := afterStale.Update(secondMsg)
	if final.Copied() {
		t.Error("the current generation's dismissMsg should clear the confirmation")
	}
}

// TestEmptyTextIsNoOp proves criterion #533: activating with an empty Text
// writes nothing, shows no confirmation, and doesn't panic.
func TestEmptyTextIsNoOp(t *testing.T) {
	tests := []tui.Key{
		{Type: tui.KeyEnter},
		{Type: tui.KeySpace},
	}
	for _, key := range tests {
		var writes []string
		m := New("", "Copy")
		m.Write = captureWrite(&writes)

		next, cmd := m.Update(key)

		if len(writes) != 0 {
			t.Errorf("Write called with empty Text, writes = %v", writes)
		}
		if next.Copied() {
			t.Error("Copied() should stay false when Text is empty")
		}
		if cmd != nil {
			t.Error("activating with empty Text should not return a Cmd")
		}
		if got := next.View(); got != "Copy" {
			t.Errorf("View() = %q, want Label unchanged", got)
		}
	}
}

// TestNonActivatingKeyIsNoOp checks that keys other than Enter/Space don't
// trigger a copy.
func TestNonActivatingKeyIsNoOp(t *testing.T) {
	var writes []string
	m := New("x", "Copy")
	m.Write = captureWrite(&writes)

	next, cmd := m.Update(tui.Key{Type: tui.KeyEsc})
	if len(writes) != 0 || next.Copied() || cmd != nil {
		t.Error("a non-activating key should be a no-op")
	}
}

// TestNonKeyMsgIsNoOp checks that an unrelated Msg type doesn't panic or
// activate the copy.
func TestNonKeyMsgIsNoOp(t *testing.T) {
	var writes []string
	m := New("x", "Copy")
	m.Write = captureWrite(&writes)

	next, cmd := m.Update(tui.ResizeMsg{Width: 10, Height: 10})
	if len(writes) != 0 || next.Copied() || cmd != nil {
		t.Error("an unrelated Msg should be a no-op")
	}
}

// TestNewDefaults checks New's documented defaults.
func TestNewDefaults(t *testing.T) {
	m := New("x", "Copy")
	if m.Timeout != 2000*time.Millisecond {
		t.Errorf("New().Timeout = %v, want 2000ms", m.Timeout)
	}
	if m.Write == nil {
		t.Error("New().Write should default to a non-nil writer")
	}
}
