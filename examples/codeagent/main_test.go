package main

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

func update(m model, msg tui.Msg) (model, tui.Cmd) {
	next, cmd := m.Update(msg)
	return next.(model), cmd
}

func key(m model, t tui.KeyType) (model, tui.Cmd) { return update(m, tui.Key{Type: t}) }

func runes(m model, s string) model {
	for _, r := range s {
		m, _ = update(m, tui.Key{Type: tui.KeyRunes, Text: string(r)})
	}
	return m
}

func ready() model {
	m := newModel()
	m.manual, m.still = true, false
	m, _ = update(m, tui.ResizeMsg{Width: 100, Height: 30})
	return m
}

// ask types a prompt and sends it.
func ask(m model, text string) model {
	m, _ = key(runes(m, text), tui.KeyEnter)
	return m
}

// run ticks until the agent waits for approval or finishes.
func run(m model) model {
	for n := 0; m.busy() && n < 5000; n++ {
		m, _ = update(m, tickMsg{})
	}
	return m
}

func text(m model) string { return ansi.StripANSI(m.View()) }

func rect(t testing.TB, m model, id string) layout.Rect {
	t.Helper()
	r, ok := layout.RectOf(m.screen(), m.size(), id)
	if !ok {
		t.Fatalf("no %q on screen:\n%s", id, text(m))
	}
	return r
}

func click(t testing.TB, m model, id string) model {
	r := rect(t, m, id)
	m, _ = update(m, tui.MouseEvent{X: r.X + 1, Y: r.Y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress, Clicks: 1})
	return m
}

func toolNamed(m model, name string) (block, bool) {
	for _, b := range m.blocks {
		if b.kind == bTool && b.t.name == name {
			return b, true
		}
	}
	return block{}, false
}

func TestFirstScreenHasAHintAndAnEmptyPrompt(t *testing.T) {
	out := text(ready())
	for _, want := range []string{"codeagent", "Ask for something", "❯ ", "Ask the agent", "ready", "0 tokens", "scripted-1", "[enter] send"} {
		if !strings.Contains(out, want) {
			t.Errorf("screen lacks %q:\n%s", want, out)
		}
	}
}

func TestPromptEditing(t *testing.T) {
	m := runes(ready(), "helo wrld")
	m, _ = key(m, tui.KeyLeft)
	m, _ = key(m, tui.KeyLeft)
	m, _ = key(m, tui.KeyLeft)
	m = runes(m, "o")
	m, _ = key(m, tui.KeyHome)
	m = runes(m, ">")
	m, _ = key(m, tui.KeyEnd)
	m, _ = key(m, tui.KeyBackspace)
	if string(m.prompt) != ">helo worl" {
		t.Fatalf("Backspace at the end: %q", string(m.prompt))
	}
	m, _ = update(m, tui.Key{Type: tui.KeyCtrl, Code: 'w'})
	if string(m.prompt) != ">helo " {
		t.Fatalf("Ctrl+W: %q", string(m.prompt))
	}
	m, _ = key(m, tui.KeyHome)
	m, _ = key(m, tui.KeyDelete)
	if string(m.prompt) != "helo " {
		t.Fatalf("Delete: %q", string(m.prompt))
	}
	m, _ = update(m, tui.Key{Type: tui.KeyCtrl, Code: 'u'})
	if len(m.prompt) != 0 {
		t.Fatal("Ctrl+U did not clear")
	}
}

func TestFixRunReadsAnEditsAfterApprovalAndTestsPass(t *testing.T) {
	m := run(ask(ready(), "fix the failing test"))
	if m.phase != waiting {
		t.Fatalf("phase %v", m.phase)
	}
	read, _ := toolNamed(m, "read_file")
	if read.st != done || !read.auto {
		t.Fatalf("read_file %+v: a low-risk call should run without asking", read)
	}
	out := text(m)
	for _, want := range []string{"edit_file layout/wrap.go", "medium risk · waiting for you", "y Approve", "n Deny", "a Always allow edit_file", "d Diff", "-\t\tif i"[:3], "waiting for your approval"} {
		if want == "-\t\tif i"[:3] {
			want = "-"
		}
		if !strings.Contains(out, want) {
			t.Errorf("screen lacks %q:\n%s", want, out)
		}
	}
	if _, ran := toolNamed(m, "run_tests"); ran {
		t.Fatal("the agent ran on past a call that needs approval")
	}
	m = runes(m, "y")
	m = run(m)
	if m.phase != waiting {
		t.Fatalf("the test run should ask too: phase %v", m.phase)
	}
	m, _ = key(m, tui.KeyEnter)
	m = run(m)
	tests, _ := toolNamed(m, "run_tests")
	if m.phase != idle || tests.st != done || !m.w.edited || !strings.Contains(text(m), "layout tests pass now") {
		t.Fatalf("phase %v tests %v edited %v:\n%s", m.phase, tests.st, m.w.edited, text(m))
	}
	if !strings.Contains(text(m), "ok      github.com/ows4444/tui/layout") {
		t.Fatalf("\n%s", text(m))
	}
}

func TestDenyingStopsThePlanAndChangesNothing(t *testing.T) {
	m := run(ask(ready(), "fix the failing test"))
	m = runes(m, "n")
	edit, _ := toolNamed(m, "edit_file")
	if edit.st != denied || m.phase != idle || m.w.edited {
		t.Fatalf("status %v phase %v", edit.st, m.phase)
	}
	if out := text(m); !strings.Contains(out, "✗ denied") || !strings.Contains(out, "I won't run edit_file") {
		t.Fatalf("\n%s", out)
	}
	if _, ran := toolNamed(m, "run_tests"); ran {
		t.Fatal("the plan went on after a denial")
	}
}

func TestEscDeniesAWaitingCall(t *testing.T) {
	m := run(ask(ready(), "delete the build directory"))
	m, _ = key(m, tui.KeyEsc)
	b, _ := toolNamed(m, "run_command")
	if b.st != denied || m.w.cleaned {
		t.Fatalf("status %v", b.st)
	}
}

func TestHighRiskCallsHaveNoAlwaysButton(t *testing.T) {
	m := run(ask(ready(), "delete the build directory"))
	out := text(m)
	if !strings.Contains(out, "high risk") || strings.Contains(out, "Always allow") || strings.Contains(out, "d Diff") {
		t.Fatalf("\n%s", out)
	}
	m = runes(m, "a")
	m = run(m)
	if !m.w.cleaned || len(m.always) != 0 || !strings.Contains(text(m), "High-risk calls always ask") {
		t.Fatalf("cleaned %v always %v", m.w.cleaned, m.always)
	}
}

func TestAlwaysAllowSkipsTheQuestionNextTime(t *testing.T) {
	m := run(ask(ready(), "fix the failing test"))
	m = runes(m, "a")
	m = run(m)
	if !m.always["edit_file"] || !strings.Contains(text(m), "edit_file runs without asking") {
		t.Fatalf("always %v", m.always)
	}
	m = runes(m, "y") // the test run
	m = run(m)
	m = ask(m, "fix the bug")
	m = run(m)
	if m.phase != waiting || m.blocks[pendingOrLast(m)].t.name != "run_tests" {
		t.Fatalf("the second run should skip the edit question and ask about the tests: phase %v", m.phase)
	}
	edits := 0
	for _, b := range m.blocks {
		if b.kind == bTool && b.t.name == "edit_file" && b.auto {
			edits++
		}
	}
	if edits != 1 {
		t.Fatalf("auto-run edits: %d", edits)
	}
}

func pendingOrLast(m model) int {
	if i := m.pendingIdx(); i >= 0 {
		return i
	}
	return len(m.blocks) - 1
}

func TestFailingTestsWhenTheEditWasNotApplied(t *testing.T) {
	m := ask(ready(), "hello there")
	m = run(m)
	if m.phase != idle || !strings.Contains(text(m), "scripted demo") {
		t.Fatalf("an unknown request should get the fallback:\n%s", text(m))
	}
}

func TestAnswerKeysTypeWhenThePromptHasText(t *testing.T) {
	m := run(ask(ready(), "fix the failing test"))
	m = runes(m, "hi n")
	if string(m.prompt) != "hi n" || m.phase != waiting || m.pendingIdx() < 0 {
		t.Fatalf("once the prompt has text, y n a d are just letters: %q phase %v", string(m.prompt), m.phase)
	}
	m, _ = key(m, tui.KeyEnter)
	if m.phase != waiting || !strings.Contains(m.msg, "busy") && m.pendingIdx() < 0 {
		t.Fatalf("Enter with text must not approve: phase %v msg %q", m.phase, m.msg)
	}
	edit, _ := toolNamed(m, "edit_file")
	if edit.st != pending {
		t.Fatalf("status %v", edit.st)
	}
}

func TestPasteGoesToThePromptAndNeverApproves(t *testing.T) {
	m := run(ask(ready(), "fix the failing test"))
	m, _ = update(m, tui.PasteEvent{Text: "y\nyes\ny"})
	edit, _ := toolNamed(m, "edit_file")
	if edit.st != pending || string(m.prompt) != "y yes y" || !strings.Contains(m.msg, "Pasted 3 lines") {
		t.Fatalf("status %v prompt %q msg %q", edit.st, string(m.prompt), m.msg)
	}
}

func TestMouseButtonsApproveDenyAndShowTheDiff(t *testing.T) {
	m := run(ask(ready(), "fix the failing test"))
	m = click(t, m, "btn-diff")
	edit, _ := toolNamed(m, "edit_file")
	if !edit.open || strings.Contains(text(m), "more lines (d shows all)") {
		t.Fatal("the diff did not open")
	}
	m = click(t, m, "btn-approve")
	m = run(m)
	if m.w.edited != true || m.phase != waiting {
		t.Fatalf("edited %v phase %v", m.w.edited, m.phase)
	}
	m = click(t, m, "btn-deny")
	tests, _ := toolNamed(m, "run_tests")
	if tests.st != denied {
		t.Fatalf("status %v", tests.st)
	}
	m2 := run(ask(ready(), "fix the failing test"))
	m2 = click(t, m2, "btn-always")
	if !m2.always["edit_file"] {
		t.Fatal("the Always button did nothing")
	}
}

func TestClickingACardOpensItsOutput(t *testing.T) {
	m := run(ask(ready(), "fix the failing test"))
	if strings.Contains(text(m), "t.Fatalf(\"got %d rows") {
		t.Fatal("output was open already")
	}
	idx := 0
	for i, b := range m.blocks {
		if b.kind == bTool && b.t.name == "read_file" {
			idx = i
		}
	}
	m = click(t, m, "card-"+itoa(idx))
	if !strings.Contains(text(m), "t.Fatalf") {
		t.Fatalf("\n%s", text(m))
	}
}

func TestStreamingRevealsTheReplyBitByBit(t *testing.T) {
	m := ask(ready(), "explain the layout package")
	for i := 0; i < thinkTicks; i++ {
		m, _ = update(m, tickMsg{})
	}
	if m.phase != saying {
		t.Fatalf("phase %v", m.phase)
	}
	m, _ = update(m, tickMsg{})
	first := text(m)
	if !strings.Contains(first, "The▌") || strings.Contains(first, "arranges nodes") {
		t.Fatalf("\n%s", first)
	}
	m = run(m)
	if !strings.Contains(text(m), "Named nodes let a program ask") || strings.Contains(text(m), "▌") || m.phase != idle {
		t.Fatalf("\n%s", text(m))
	}
}

func TestEscStopsAReplyAndKeepsWhatWasShown(t *testing.T) {
	m := ask(ready(), "explain the layout package")
	for i := 0; i < thinkTicks+5; i++ {
		m, _ = update(m, tickMsg{})
	}
	m, _ = key(m, tui.KeyEsc)
	if m.phase != idle || !strings.Contains(text(m), "· Stopped") || strings.Contains(text(m), "Named nodes") {
		t.Fatalf("\n%s", text(m))
	}
	if _, cmd := update(m, tickMsg{}); cmd != nil {
		t.Fatal("ticks went on after the stop")
	}
}

func TestCtrlCStopsThenQuits(t *testing.T) {
	m := ask(ready(), "explain the layout package")
	m, cmd := key(m, tui.KeyCtrlC)
	if cmd != nil || m.phase != idle {
		t.Fatal("first Ctrl+C should stop")
	}
	if _, cmd := key(m, tui.KeyCtrlC); cmd == nil {
		t.Fatal("second Ctrl+C should quit")
	}
	if _, cmd := update(ready(), tui.Key{Type: tui.KeyCtrl, Code: 'd'}); cmd == nil {
		t.Fatal("Ctrl+D should quit")
	}
	if _, cmd := key(runes(ready(), "/quit"), tui.KeyEnter); cmd == nil {
		t.Fatal("/quit should quit")
	}
}

func TestBusyAgentRefusesANewPrompt(t *testing.T) {
	m := ask(ready(), "explain the layout package")
	m = ask(m, "something else")
	if !strings.Contains(m.msg, "busy") || len(m.hist) != 1 {
		t.Fatalf("msg %q hist %v", m.msg, m.hist)
	}
}

func TestHistoryWalksPromptsAndKeepsTheDraft(t *testing.T) {
	m := run(ask(ready(), "explain the layout package"))
	m = run(ask(m, "what is a grid"))
	m = runes(m, "draft")
	m, _ = key(m, tui.KeyUp)
	if string(m.prompt) != "what is a grid" {
		t.Fatalf("%q", string(m.prompt))
	}
	m, _ = key(m, tui.KeyUp)
	if string(m.prompt) != "explain the layout package" {
		t.Fatalf("%q", string(m.prompt))
	}
	m, _ = key(m, tui.KeyUp)
	if string(m.prompt) != "explain the layout package" {
		t.Fatal("went past the first")
	}
	m, _ = key(m, tui.KeyDown)
	m, _ = key(m, tui.KeyDown)
	if string(m.prompt) != "draft" {
		t.Fatalf("the draft came back as %q", string(m.prompt))
	}
}

func TestSlashCommands(t *testing.T) {
	m := ask(ready(), "/allow")
	if !strings.Contains(text(m), "Nothing is always allowed") {
		t.Fatalf("\n%s", text(m))
	}
	m = ask(m, "/history")
	if !strings.Contains(text(m), "Earlier prompts: /allow") {
		t.Fatalf("\n%s", text(m))
	}
	m = ask(m, "/nope")
	if !strings.Contains(text(m), "Unknown command /nope") {
		t.Fatalf("\n%s", text(m))
	}
	m = ask(m, "/reset")
	if !strings.Contains(text(m), "put back") {
		t.Fatalf("\n%s", text(m))
	}
	m = ask(m, "/clear")
	if len(m.blocks) != 0 {
		t.Fatalf("%d blocks after /clear", len(m.blocks))
	}
	m = ask(m, "/help")
	if !m.help || !strings.Contains(text(m), "approve, deny, always allow") {
		t.Fatalf("\n%s", text(m))
	}
	if m, _ = key(m, tui.KeyEsc); m.help {
		t.Fatal("a key should close help")
	}
}

func TestScrollingTheTranscript(t *testing.T) {
	m, _ := update(ready(), tui.ResizeMsg{Width: 100, Height: 12})
	m = run(ask(m, "explain the layout package"))
	m = run(ask(m, "fix the failing test"))
	if !strings.Contains(text(m), "edit_file") {
		t.Fatalf("the newest lines should show:\n%s", text(m))
	}
	m, _ = key(m, tui.KeyPgUp)
	if m.off == 0 || !strings.Contains(text(m), "lines below") {
		t.Fatalf("off %d:\n%s", m.off, text(m))
	}
	up := m.off
	m, _ = update(m, tui.MouseEvent{Button: tui.MouseButtonWheelUp, Action: tui.MouseActionPress})
	if m.off <= up {
		t.Fatal("the wheel did not scroll")
	}
	m, _ = key(m, tui.KeyPgDown)
	m, _ = key(m, tui.KeyPgDown)
	m, _ = key(m, tui.KeyPgDown)
	if m.off != 0 {
		t.Fatalf("off %d", m.off)
	}
	m = ask(m, "x")
	if m.off != 0 {
		t.Fatal("new output should follow")
	}
}

func TestReducedMotionShowsTheWholeRunAtOnce(t *testing.T) {
	m := ready()
	m.still = true
	m, cmd := key(runes(m, "fix the failing test"), tui.KeyEnter)
	if cmd != nil || m.phase != waiting {
		t.Fatalf("phase %v cmd %v", m.phase, cmd)
	}
	if !strings.Contains(text(m), "The test expects five rows but Wrap produces six") || strings.Contains(text(m), "▌") {
		t.Fatalf("\n%s", text(m))
	}
	m, _ = key(m, tui.KeyEnter)
	if m.phase != waiting {
		t.Fatalf("phase %v", m.phase)
	}
	m, _ = key(m, tui.KeyEnter)
	if m.phase != idle || !strings.Contains(text(m), "tests pass now") {
		t.Fatalf("\n%s", text(m))
	}
}

func TestSmallWindowsKeepThePromptAndHints(t *testing.T) {
	m, _ := update(ready(), tui.ResizeMsg{Width: 40, Height: 10})
	m = run(ask(m, "fix the failing test"))
	lines := strings.Split(text(m), "\n")
	if len(lines) != 10 || !strings.Contains(lines[8], "❯") && !strings.Contains(lines[8], "?") || !strings.Contains(lines[9], "[enter]") {
		t.Fatalf("\n%s", text(m))
	}
}
