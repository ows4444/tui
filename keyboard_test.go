package tui

import (
	"strings"
	"sync"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/input"
)

func TestWithKeyboardPushesFlagsAndPopsInReverse(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{}, WithOutput(out), WithBracketedPaste(true),
		WithKeyboard(KeyboardDisambiguate|KeyboardReportEvents), WithFocusReporting(true))
	p.enterModes()
	got := string(read())
	want := ansi.BracketedPasteEnable + "\x1b[>3u" + ansi.FocusReportingEnable
	if !strings.Contains(got, want) {
		t.Fatalf("enter = %q, want it to contain %q", got, want)
	}
	before := len(read())
	p.leaveModes() // what quit, panic (restoreTerminal) and Suspend all call
	left := string(read()[before:])
	if !strings.HasPrefix(left, ansi.FocusReportingDisable+ansi.KittyKeyboardDisable+ansi.BracketedPasteDisable) {
		t.Errorf("leave = %q", left)
	}
	// Suspend re-enters with the same flags.
	before = len(read())
	p.enterModes()
	if again := string(read()[before:]); !strings.Contains(again, "\x1b[>3u") {
		t.Errorf("re-enter = %q", again)
	}
}

func TestWithKeyboardMasksAndCombines(t *testing.T) {
	p := NewProgram(staticModel{}, WithKeyboard(0xff))
	if p.kittyFlags() != 15 {
		t.Errorf("flags = %d, want 15", p.kittyFlags())
	}
	p = NewProgram(staticModel{}, WithKittyKeyboard(true), WithKeyboard(KeyboardReportEvents))
	if p.kittyFlags() != 3 {
		t.Errorf("combined = %d, want 3", p.kittyFlags())
	}
	out, read := captureOutput(t)
	p = NewProgram(staticModel{}, WithOutput(out), WithKittyKeyboard(true))
	p.enterModes()
	if !strings.Contains(string(read()), ansi.KittyKeyboardEnable) {
		t.Error("WithKittyKeyboard output changed")
	}
	out2, read2 := captureOutput(t)
	p = NewProgram(staticModel{}, WithOutput(out2))
	p.enterModes()
	p.leaveModes()
	if got := string(read2()); strings.Contains(got, "\x1b[>") || strings.Contains(got, ansi.KittyKeyboardDisable) {
		t.Errorf("no keyboard sequence expected by default: %q", got)
	}
}

type actRec struct {
	mu   *sync.Mutex
	keys *[]Key
}

func (actRec) Init() Cmd { return nil }
func (m actRec) Update(msg Msg) (Model, Cmd) {
	k, ok := msg.(Key)
	switch r := msg.(type) { // #40: non-press keys arrive as their own Msgs
	case KeyReleaseMsg:
		k, ok = r.Key, true
	case KeyRepeatMsg:
		k, ok = r.Key, true
	}
	if ok {
		m.mu.Lock()
		*m.keys = append(*m.keys, k)
		m.mu.Unlock()
		if k.Type == KeyRunes && k.Code == 'q' {
			return m, Quit()
		}
	}
	return m, nil
}
func (actRec) View() string { return "x" }

func runKeyActions(t *testing.T, in string, opts ...ProgramOption) []Key {
	t.Helper()
	var mu sync.Mutex
	var keys []Key
	opts = append(opts, WithOutput(&strings.Builder{}), WithInput(strings.NewReader(in)))
	if _, err := NewProgram(actRec{&mu, &keys}, opts...).Run(); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	return append([]Key(nil), keys...)
}

// criterion #64 end to end: Update sees release/repeat only when asked.
func TestWithKeyboardDeliversReleaseAndRepeatOnlyWhenAsked(t *testing.T) {
	in := "\x1b[97;1:2u\x1b[97;1:3uq"
	on := runKeyActions(t, in, WithKeyboard(KeyboardDisambiguate|KeyboardReportEvents))
	if len(on) != 3 || on[0].Action != KeyRepeat || on[1].Action != KeyRelease || on[1].Type != KeyRunes {
		t.Fatalf("with report-events: %+v", on)
	}
	for _, opts := range [][]ProgramOption{nil, {WithKittyKeyboard(true)}, {WithKeyboard(KeyboardDisambiguate)}} {
		off := runKeyActions(t, in, opts...)
		for _, k := range off {
			if k.Action != KeyPress {
				t.Errorf("opts %d: Update saw %+v", len(opts), k)
			}
		}
		if len(off) < 2 || off[1].Type != input.KeyUnknown {
			t.Errorf("legacy release = %+v", off)
		}
	}
}
