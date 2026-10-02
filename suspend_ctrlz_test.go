package tui

import (
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ows4444/tui/ansi"
)

// lockedBuf is an output writer safe to read while the loop writes.
type lockedBuf struct {
	mu sync.Mutex
	sb strings.Builder
}

func (b *lockedBuf) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.sb.Write(p)
}
func (b *lockedBuf) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.sb.String()
}

// keyLog records every Key it is given, draws "v<n>" (n = messages seen) and
// quits on 'q'.
type keyLog struct {
	mu   *sync.Mutex
	keys *[]Key
	n    int
}

func (keyLog) Init() Cmd { return nil }
func (m keyLog) Update(msg Msg) (Model, Cmd) {
	m.n++
	k, ok := msg.(Key)
	if r, isRel := msg.(KeyReleaseMsg); isRel {
		k, ok = r.Key, true // #40: releases arrive as KeyReleaseMsg
	}
	if !ok {
		return m, nil
	}
	m.mu.Lock()
	*m.keys = append(*m.keys, k)
	m.mu.Unlock()
	if k.Type == KeyRunes && k.Text == "q" {
		return m, Quit()
	}
	return m, nil
}
func (m keyLog) View() string { return "v" + string(rune('0'+m.n)) }

func (m keyLog) snapshot() []Key {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Key(nil), *m.keys...)
}

func waitCtrlZ(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for condition")
		}
		time.Sleep(time.Millisecond)
	}
}

// Default off: Ctrl+Z is an ordinary key and nothing is stopped.
func TestCtrlZIsOrdinaryKeyByDefault(t *testing.T) {
	pr, pw := mustPipe(t)
	t.Cleanup(func() { pr.Close(); pw.Close() })
	var stops atomic.Int32
	m := keyLog{mu: &sync.Mutex{}, keys: new([]Key)}
	p := NewProgram(m, WithInput(pr), WithOutput(&lockedBuf{}))
	p.stopProcess = func() error { stops.Add(1); return nil }
	ch := startSendLoop(t, p)
	pw.WriteString("\x1aq")
	if res := <-ch; res.err != nil {
		t.Fatal(res.err)
	}
	got := m.snapshot()
	if len(got) != 2 || got[0].Type != KeyCtrl || got[0].Code != 'z' {
		t.Fatalf("keys = %+v, want Ctrl+Z then q", got)
	}
	if stops.Load() != 0 {
		t.Errorf("process stopped %d times with the option off", stops.Load())
	}
}

// With the option: the terminal is restored before the stop, re-entered
// (kitty flags included) after it, the reader is stopped during it and runs
// again, the whole frame is repainted, and Update never sees the Ctrl+Z.
func TestCtrlZStopsRestoresAndRedraws(t *testing.T) {
	pr, pw := mustPipe(t)
	t.Cleanup(func() { pr.Close(); pw.Close() })
	out := &lockedBuf{}
	m := keyLog{mu: &sync.Mutex{}, keys: new([]Key)}
	p := NewProgram(m, WithInput(pr), WithOutput(out), WithSuspendOnCtrlZ(true),
		WithKeyboard(KeyboardDisambiguate|KeyboardReportEvents), WithFocusReporting(true))

	var mu sync.Mutex
	var atStop string
	var readerStopped bool
	var stops atomic.Int32
	p.stopProcess = func() error {
		mu.Lock()
		atStop = out.String()
		select {
		case <-p.rdDone:
			readerStopped = true
		default:
		}
		mu.Unlock()
		stops.Add(1)
		return nil
	}
	ch := startSendLoop(t, p)
	waitCtrlZ(t, func() bool { return strings.Contains(out.String(), "v1") })
	pw.WriteString("\x1a")
	waitCtrlZ(t, func() bool { return stops.Load() == 1 })
	waitCtrlZ(t, func() bool {
		return strings.Count(out.String(), ansi.KittyKeyboardEnableFlags(3)) == 1 // startSendLoop skips Run, so only the resume writes it
	})
	pw.WriteString("aq")
	if res := <-ch; res.err != nil {
		t.Fatal(res.err)
	}

	mu.Lock()
	defer mu.Unlock()
	if !readerStopped {
		t.Error("reader still running when the process stopped")
	}
	for _, s := range []string{ansi.KittyKeyboardDisable, ansi.CursorShow, ansi.AltScreenDisable} {
		if !strings.Contains(atStop, s) {
			t.Errorf("terminal not restored before the stop: missing %q", s)
		}
	}
	if !strings.HasSuffix(atStop, ansi.AltScreenDisable) {
		t.Errorf("modes still on at the stop: %q", atStop)
	}
	after := strings.TrimPrefix(out.String(), atStop)
	for _, s := range []string{ansi.AltScreenEnable, ansi.CursorHide, ansi.FocusReportingEnable, ansi.ClearScreen, "v1"} {
		if !strings.Contains(after, s) {
			t.Errorf("after resume missing %q in %q", s, after)
		}
	}
	got := m.snapshot()
	if len(got) != 2 || got[0].Type != KeyRunes || got[1].Type != KeyRunes {
		t.Errorf("Update saw %+v, want only a then q (Ctrl+Z consumed, input works after resume)", got)
	}
}

// A kitty-encoded Ctrl+Z press suspends too; its release is not a suspend.
func TestCtrlZKittyPressSuspendsReleaseDoesNot(t *testing.T) {
	pr, pw := mustPipe(t)
	t.Cleanup(func() { pr.Close(); pw.Close() })
	m := keyLog{mu: &sync.Mutex{}, keys: new([]Key)}
	var stops atomic.Int32
	p := NewProgram(m, WithInput(pr), WithOutput(&lockedBuf{}), WithSuspendOnCtrlZ(true),
		WithKeyboard(KeyboardDisambiguate|KeyboardReportEvents))
	p.stopProcess = func() error { stops.Add(1); return nil }
	ch := startSendLoop(t, p)
	pw.WriteString("\x1b[122;5u\x1b[122;5:3uq")
	if res := <-ch; res.err != nil {
		t.Fatal(res.err)
	}
	if stops.Load() != 1 {
		t.Errorf("stops = %d, want 1", stops.Load())
	}
	got := m.snapshot()
	if len(got) != 2 || got[0].Action != KeyRelease {
		t.Errorf("Update saw %+v, want the release then q", got)
	}
}
