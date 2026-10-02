package tui

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/termio"
	"github.com/ows4444/tui/internal/vtscreen"
)

// --- fixtures ---

// staticModel draws a fixed view; tests that call p.render() directly
// (bypassing runLoop) swap p.model between renders to control what View()
// returns from one render to the next.
type staticModel struct{ view string }

func (m staticModel) Init() Cmd               { return nil }
func (m staticModel) Update(Msg) (Model, Cmd) { return m, nil }
func (m staticModel) View() string            { return m.view }

// captureOutput returns a temp file to use as a Program's output, and a
// function that reads back everything written to it so far.
func captureOutput(t *testing.T) (f *os.File, read func() []byte) {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "out")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	t.Cleanup(func() { f.Close() })
	return f, func() []byte {
		b, err := os.ReadFile(f.Name())
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		return b
	}
}

// visibleLines replays render output through the shared VT screen model
// (internal/vtscreen.Screen, the model behind the public tuitest package) on
// an 80x8 screen and returns the rows through the last non-empty one, padded
// to at least min rows.
func visibleLines(data []byte, min int) []string {
	scr := vtscreen.NewScreen(80, 8)
	scr.Write(data)
	lines := scr.Lines()
	n := len(lines)
	for n > min && lines[n-1] == "" {
		n--
	}
	return lines[:n]
}

// --- render(): relative addressing, criterion #458 ---

func TestRenderVisibleContentMatchesViewAcrossSequences(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{view: "a\nb\nc"}, WithOutput(out))

	// Initial render, from p.liveLines == 0.
	p.render()

	// A changed View(): one row's content changes.
	p.model = staticModel{view: "a\nbb\nc"}
	p.render()

	// A shrinking View(): fewer rows than before.
	p.model = staticModel{view: "a\nbb"}
	p.render()

	got := visibleLines(read(), 3)
	want := []string{"a", "bb", ""} // row 2 blanked, not removed
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("visible content = %#v, want %#v", got, want)
	}
}

func TestRenderUnchangedRowsSkipRewrite(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{view: "one\ntwo"}, WithOutput(out))
	p.render()

	before := len(read())
	p.render() // identical View(): no content bytes should be emitted
	after := read()

	added := string(after[before:])
	if strings.Contains(added, "one") || strings.Contains(added, "two") {
		t.Fatalf("re-render of an unchanged View() rewrote content: %q", added)
	}
	got := visibleLines(after, 2)
	want := []string{"one", "two"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("visible content after no-op render = %#v, want %#v", got, want)
	}
}

// TestRenderWrapsFrameInSynchronizedOutput proves acceptance
// criterion: render()'s write is bracketed in CSI ?2026h/l so the terminal
// paints the whole diffed frame atomically instead of tearing mid-repaint.
func TestRenderWrapsFrameInSynchronizedOutput(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{view: "one\ntwo"}, WithOutput(out))
	p.render()

	got := string(read())
	if !strings.HasPrefix(got, ansi.SyncOutputEnable) {
		t.Fatalf("render() output does not start with SyncOutputEnable: %q", got)
	}
	if !strings.HasSuffix(got, ansi.SyncOutputDisable) {
		t.Fatalf("render() output does not end with SyncOutputDisable: %q", got)
	}
}

// TestRepaintFreshWrapsClearAndRedrawInOneSynchronizedWrite proves that
// repaintFresh's clear-then-redraw is a single CSI ?2026-wrapped write, not
// two separate ones — two separate wrapped writes would let the terminal
// paint the clear before the redraw catches up, reintroducing the tearing
// mode 2026 is meant to prevent.
func TestRepaintFreshWrapsClearAndRedrawInOneSynchronizedWrite(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{view: "one\ntwo"}, WithOutput(out))
	p.altScreen = true
	p.repaintFresh()

	got := string(read())
	if !strings.HasPrefix(got, ansi.SyncOutputEnable) {
		t.Fatalf("repaintFresh() output does not start with SyncOutputEnable: %q", got)
	}
	if !strings.HasSuffix(got, ansi.SyncOutputDisable) {
		t.Fatalf("repaintFresh() output does not end with SyncOutputDisable: %q", got)
	}
	if strings.Count(got, ansi.SyncOutputEnable) != 1 || strings.Count(got, ansi.SyncOutputDisable) != 1 {
		t.Fatalf("repaintFresh() wrapped its clear and redraw in more than one synchronized-output span: %q", got)
	}
}

func TestRenderUsesRelativeNotAbsoluteAddressing(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{view: "a\nb"}, WithOutput(out))
	p.render()
	p.model = staticModel{view: "x\nb"}
	p.render()

	got := string(read())
	if strings.Contains(got, "\x1b[1;1H") || strings.Contains(got, "\x1b[2;1H") {
		t.Fatalf("render() used absolute CursorPosition addressing: %q", got)
	}
	if !strings.Contains(got, "\x1b[") || !strings.Contains(got, "A") {
		t.Fatalf("render() never emitted a relative CursorUp: %q", got)
	}
}

// --- Println / printMsg ---

// printModel lets a test control exactly which Cmd Init() returns, and
// records every Msg its Update() is handed (which must never include a
// printMsg — that's runLoop's job to intercept).
type printModel struct {
	initCmd   Cmd
	view      string
	received  []Msg
	quitAfter int
}

func (m printModel) Init() Cmd { return m.initCmd }
func (m printModel) Update(msg Msg) (Model, Cmd) {
	m.received = append(append([]Msg(nil), m.received...), msg)
	if len(m.received) >= m.quitAfter {
		return m, Quit()
	}
	return m, nil
}
func (m printModel) View() string { return m.view }

func TestPrintlnMessageNotForwardedToUpdate(t *testing.T) {
	pr, pw := mustPipe(t)
	defer pr.Close()
	defer pw.Close()
	out, read := captureOutput(t)

	p := NewProgram(printModel{view: "v", quitAfter: 2}, WithInput(pr), WithOutput(out), WithAltScreen(false))

	resultCh := make(chan runResult, 1)
	go func() {
		m, err := p.runLoop()
		resultCh <- runResult{m, err}
	}()

	// The seed ResizeMsg lands first (received[0]); a printMsg must never
	// show up in received at all; the Key below is received[1] and quits.
	p.msgs <- printMsg{text: "hello"}
	p.msgs <- Key{Type: KeyRunes, Text: "x", Code: 'x'}

	var res runResult
	select {
	case res = <-resultCh:
	case <-time.After(2 * time.Second):
		t.Fatal("runLoop did not return in time")
	}

	pm, ok := res.model.(printModel)
	if !ok {
		t.Fatalf("result model type = %T", res.model)
	}
	if len(pm.received) != 2 {
		t.Fatalf("Update saw %d msgs, want 2: %#v", len(pm.received), pm.received)
	}
	for _, msg := range pm.received {
		if _, ok := msg.(printMsg); ok {
			t.Fatalf("printMsg reached Update: %#v", pm.received)
		}
	}
	if !strings.Contains(string(read()), "hello") {
		t.Fatalf("output missing committed text: %q", read())
	}
}

func TestPrintlnTextPrecedesNextLiveFrameRedraw(t *testing.T) {
	pr, pw := mustPipe(t)
	defer pr.Close()
	defer pw.Close()
	out, read := captureOutput(t)

	p := NewProgram(printModel{view: "before", quitAfter: 2}, WithInput(pr), WithOutput(out), WithAltScreen(false))

	resultCh := make(chan runResult, 1)
	go func() {
		m, err := p.runLoop()
		resultCh <- runResult{m, err}
	}()

	p.msgs <- printMsg{text: "committed-line"}
	p.msgs <- Key{Type: KeyRunes, Text: "x", Code: 'x'}

	select {
	case <-resultCh:
	case <-time.After(2 * time.Second):
		t.Fatal("runLoop did not return in time")
	}

	got := string(read())
	printIdx := strings.Index(got, "committed-line")
	if printIdx < 0 {
		t.Fatalf("output missing committed text: %q", got)
	}
	// "before" is the frame's content both before and after the printMsg
	// is processed; what matters is that a repaint (the redraw render()
	// triggers right after the commit) follows the committed text in the
	// byte stream, not that "before" is unique — search after printIdx.
	if !strings.Contains(got[printIdx:], "before") {
		t.Fatalf("live-frame redraw did not follow the committed text: %q", got)
	}
}

func TestPrintlnThenLiveRegionStillMatchesView(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{view: "line1\nline2"}, WithOutput(out))
	p.render()

	p.printLines("committed", p.output)
	p.render()

	got := visibleLines(read(), 3)
	want := []string{"committed", "line1", "line2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("visible content after Println = %#v, want %#v", got, want)
	}
}

func TestPrintlnMultiLineTextEachLineGetsRealNewline(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{view: "v"}, WithOutput(out))
	p.render()

	p.printLines("first\nsecond\nthird", p.output)

	got := string(read())
	if strings.Count(got, "\r\n") < 3 {
		t.Fatalf("expected at least 3 real newlines for 3 committed lines, got %q", got)
	}
	for _, want := range []string{"first\r\n", "second\r\n", "third\r\n"} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q: %q", want, got)
		}
	}
}

func TestPrintlnBeforeAnyRenderDoesNotPanic(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{view: "v"}, WithOutput(out))

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("printLines before any render() panicked: %v", r)
		}
	}()
	p.printLines("first output", p.output)

	got := string(read())
	if !strings.Contains(got, "first output\r\n") {
		t.Fatalf("output missing committed text: %q", got)
	}
	if strings.Contains(got, "\x1b[") {
		t.Fatalf("printLines before any render() emitted cursor movement it shouldn't have: %q", got)
	}
}

// TestEprintlnDefaultsToStderr proves criterion #634: with WithErrOutput
// never called, Eprintln's destination is os.Stderr.
func TestEprintlnDefaultsToStderr(t *testing.T) {
	p := NewProgram(staticModel{view: "v"})
	if p.errOutput != os.Stderr {
		t.Errorf("errOutput = %v, want os.Stderr", p.errOutput)
	}
}

// TestEprintlnWritesToErrOutputNotOutput proves criteria #630 and #632: a
// stderr printMsg (what Eprintln's Cmd produces) is committed to the
// WithErrOutput destination, never to the ordinary WithOutput destination,
// and never reaches Update.
func TestEprintlnWritesToErrOutputNotOutput(t *testing.T) {
	pr, pw := mustPipe(t)
	defer pr.Close()
	defer pw.Close()
	out, readOut := captureOutput(t)
	errOut, readErr := captureOutput(t)

	p := NewProgram(printModel{view: "v", quitAfter: 2}, WithInput(pr), WithOutput(out), WithErrOutput(errOut), WithAltScreen(false))

	resultCh := make(chan runResult, 1)
	go func() {
		m, err := p.runLoop()
		resultCh <- runResult{m, err}
	}()

	p.msgs <- printMsg{text: "stderr-text", stderr: true}
	p.msgs <- Key{Type: KeyRunes, Text: "x", Code: 'x'}

	var res runResult
	select {
	case res = <-resultCh:
	case <-time.After(2 * time.Second):
		t.Fatal("runLoop did not return in time")
	}

	pm, ok := res.model.(printModel)
	if !ok {
		t.Fatalf("result model type = %T", res.model)
	}
	for _, msg := range pm.received {
		if _, ok := msg.(printMsg); ok {
			t.Fatalf("stderr printMsg reached Update: %#v", pm.received)
		}
	}

	if !strings.Contains(string(readErr()), "stderr-text") {
		t.Fatalf("errOutput missing committed text: %q", readErr())
	}
	if strings.Contains(string(readOut()), "stderr-text") {
		t.Fatalf("output should not receive Eprintln's text, got: %q", readOut())
	}
}

// TestEprintlnCursorMathAnchoredToLiveLinesAndResetsFrame proves criteria
// #631 and #633: an Eprintln commit uses the same CursorUp/EraseDown
// sequence printLines already computes from p.liveLines (the STDOUT live
// region), written to the stderr destination, and resets p.lastFrame so
// the following render() repaints fresh.
func TestEprintlnCursorMathAnchoredToLiveLinesAndResetsFrame(t *testing.T) {
	out, _ := captureOutput(t)
	errOut, readErr := captureOutput(t)
	p := NewProgram(staticModel{view: "line1\nline2"}, WithOutput(out), WithErrOutput(errOut))
	p.render() // establishes p.liveLines = 2

	p.printLines("committed", errOut)

	got := string(readErr())
	if !strings.Contains(got, ansi.CursorUp(1)) {
		t.Errorf("errOutput missing CursorUp(1) anchored to the stdout live region: %q", got)
	}
	if !strings.Contains(got, ansi.EraseDown) {
		t.Errorf("errOutput missing EraseDown: %q", got)
	}
	if !strings.Contains(got, "committed\r\n") {
		t.Errorf("errOutput missing the committed text: %q", got)
	}
	if p.lastFrame != nil {
		t.Error("p.lastFrame should be reset to nil after a commit, so the next render() repaints fresh")
	}
	if p.liveLines != 0 {
		t.Errorf("p.liveLines = %d, want 0 after a commit", p.liveLines)
	}
}

// suspendRecorderModel records every Msg it sees (including SuspendMsg)
// and quits only on a 'q' key, so a test can confirm the Program keeps
// running normally after a SuspendMsg carrying an error.
type suspendRecorderModel struct {
	received []Msg
}

func (m suspendRecorderModel) Init() Cmd { return nil }
func (m suspendRecorderModel) Update(msg Msg) (Model, Cmd) {
	m.received = append(append([]Msg(nil), m.received...), msg)
	if k, ok := msg.(Key); ok && k.Type == KeyRunes && k.Text == "q" {
		return m, Quit()
	}
	return m, nil
}
func (suspendRecorderModel) View() string { return "" }

// TestSuspendUndoesAndRedoesTerminalSetupInOrder proves criteria #636,
// #637 (termRestore nil is the no-real-terminal case; the ansi setup still
// runs) and #638: with every optional feature enabled, suspend writes the
// disable codes in the reverse of Run's enable order, calls fn, then
// writes the enable codes back in Run's original order.
func TestSuspendUndoesAndRedoesTerminalSetupInOrder(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{view: "v"}, WithOutput(out), WithAltScreen(true), WithMouse(MouseClick), WithBracketedPaste(true))
	p.render() // establishes p.liveLines > 0, for the reset check below

	var fnCalled bool
	err := p.suspend(func() error {
		fnCalled = true
		return nil
	})
	if err != nil {
		t.Fatalf("suspend returned error %v, want nil", err)
	}
	if !fnCalled {
		t.Fatal("suspend did not call fn")
	}

	got := string(read())
	mouseEnable, mouseDisable := mouseModeCodes(MouseClick)
	undoOrder := []string{
		ansi.BracketedPasteDisable,
		ansi.MouseSGRDisable + mouseDisable,
		ansi.CursorShow,
		ansi.AltScreenDisable,
	}
	redoOrder := []string{
		ansi.AltScreenEnable,
		ansi.CursorHide,
		mouseEnable + ansi.MouseSGREnable,
		ansi.BracketedPasteEnable,
	}

	last := -1
	for _, code := range append(append([]string{}, undoOrder...), redoOrder...) {
		idx := strings.Index(got, code)
		if idx < 0 {
			t.Fatalf("output missing %q entirely: %q", code, got)
		}
		if idx <= last {
			t.Fatalf("code %q at index %d is out of order (want after index %d): %q", code, idx, last, got)
		}
		last = idx
	}
}

// TestSuspendSkipsDisabledFeatures proves criterion #641: a feature Run
// never enabled is neither undone nor redone by suspend, while the
// unconditional cursor show/hide still happens.
func TestSuspendSkipsDisabledFeatures(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{view: "v"}, WithOutput(out), WithAltScreen(false), WithBracketedPaste(false))
	// mouseMode defaults to MouseOff; bracketed paste is on by default, so it
	// is turned off here.

	if err := p.suspend(func() error { return nil }); err != nil {
		t.Fatalf("suspend returned error %v, want nil", err)
	}

	got := string(read())
	for _, absent := range []string{ansi.AltScreenDisable, ansi.AltScreenEnable, ansi.MouseSGRDisable, ansi.MouseSGREnable, ansi.BracketedPasteDisable, ansi.BracketedPasteEnable} {
		if strings.Contains(got, absent) {
			t.Errorf("output should not contain %q for a disabled feature: %q", absent, got)
		}
	}
	if !strings.Contains(got, ansi.CursorShow) || !strings.Contains(got, ansi.CursorHide) {
		t.Errorf("output should still contain the unconditional CursorShow/CursorHide: %q", got)
	}
}

// TestSuspendResetsFrameState proves criterion #639.
func TestSuspendResetsFrameState(t *testing.T) {
	out, _ := captureOutput(t)
	p := NewProgram(staticModel{view: "v"}, WithOutput(out))
	p.render()
	if p.liveLines == 0 {
		t.Fatal("setup: render() should have set p.liveLines > 0")
	}

	if err := p.suspend(func() error { return nil }); err != nil {
		t.Fatalf("suspend returned error %v, want nil", err)
	}
	if p.lastFrame != nil {
		t.Error("p.lastFrame should be reset to nil after suspend")
	}
	if p.liveLines != 0 {
		t.Errorf("p.liveLines = %d, want 0 after suspend", p.liveLines)
	}
}

// TestSuspendViaRunLoopDeliversErrorWithoutQuitting proves criteria #640
// and #642: dispatched through runLoop (not called directly), a Suspend
// whose fn errors delivers that error to Update via SuspendMsg without
// ending the Program, and fd/state — Program fields the suspend handler
// reads rather than Run-local variables — are reachable with no real
// terminal involved (termRestore stays nil here, so the suspend handler skips
// the raw-mode calls and a Fake terminal sees none).
func TestSuspendViaRunLoopDeliversErrorWithoutQuitting(t *testing.T) {
	pr, pw := mustPipe(t)
	defer pr.Close()
	defer pw.Close()
	out, _ := captureOutput(t)

	ft := &termio.Fake{}
	p := NewProgram(suspendRecorderModel{}, WithInput(pr), WithOutput(out), WithTerminal(ft))

	resultCh := make(chan runResult, 1)
	go func() {
		m, err := p.runLoop()
		resultCh <- runResult{m, err}
	}()

	wantErr := errors.New("child exited nonzero")
	p.msgs <- suspendMsg{fn: func() error { return wantErr }}
	// Confirm the loop is still alive after the error by sending an
	// ordinary key, then quit explicitly.
	p.msgs <- Key{Type: KeyRunes, Text: "q", Code: 'q'}

	var res runResult
	select {
	case res = <-resultCh:
	case <-time.After(2 * time.Second):
		t.Fatal("runLoop did not return in time (Suspend's error may have wrongly terminated it, or SuspendMsg never arrived)")
	}
	if res.err != nil {
		t.Fatalf("runLoop returned error: %v", res.err)
	}

	rec, ok := res.model.(suspendRecorderModel)
	if !ok {
		t.Fatalf("result model type = %T", res.model)
	}

	var sawSuspendMsg bool
	for _, msg := range rec.received {
		if sm, ok := msg.(SuspendMsg); ok {
			sawSuspendMsg = true
			if sm.Err == nil || sm.Err.Error() != wantErr.Error() {
				t.Errorf("SuspendMsg.Err = %v, want %v", sm.Err, wantErr)
			}
		}
		if _, ok := msg.(suspendMsg); ok {
			t.Fatalf("the raw suspendMsg reached Update: %#v", rec.received)
		}
	}
	if !sawSuspendMsg {
		t.Fatalf("Update never received a SuspendMsg: %#v", rec.received)
	}
	if raw, unraw, _, _ := ft.Calls(); raw != 0 || unraw != 0 {
		t.Errorf("raw mode touched without Run having entered it: raw=%d unraw=%d", raw, unraw)
	}
}

// TestInlineRenderAndPrintlnStayBelowThePrompt guards the inline anchor:
// with a shell prompt above the live region, repainting and committing text
// must never touch that prompt line, nor drift the region up a row.
func TestInlineRenderAndPrintlnStayBelowThePrompt(t *testing.T) {
	f, read := captureOutput(t)
	f.WriteString("PROMPT$ run\r\n")
	p := &Program{output: f, altScreen: false, model: staticModel{"A\nB\nC"}}
	p.render()
	p.model = staticModel{"A\nB\nD"}
	p.render()
	p.printLines("hist", f)
	p.render()

	scr := vtscreen.NewScreen(20, 8)
	scr.Write(read())
	got := scr.Lines()[:6]
	want := []string{"PROMPT$ run", "hist", "A", "B", "D", ""}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("screen = %q, want %q", got, want)
	}
}
