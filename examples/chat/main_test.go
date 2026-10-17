package main

import (
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/tuitest"
)

func update(m model, msg tui.Msg) (model, tui.Cmd) {
	next, cmd := m.Update(msg)
	return next.(model), cmd
}

func send(m model, msg tui.Msg) model {
	m, _ = update(m, msg)
	return m
}

func typed(m model, s string) model {
	for _, r := range s {
		m = send(m, tui.Key{Type: tui.KeyRunes, Text: string(r)})
	}
	return m
}

func ready(w, h int) model {
	return send(newModel(true), tui.ResizeMsg{Width: w, Height: h})
}

func click(t testing.TB, m model, id string) model {
	t.Helper()
	r, ok := layout.RectOf(m.screen(), m.size(), id)
	if !ok {
		t.Fatalf("no target %q on screen", id)
	}
	return send(m, tui.MouseEvent{X: r.X + r.W/2, Y: r.Y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
}

func isQuit(cmd tui.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tui.QuitMsg)
	return ok
}

func text(m model) string { return ansi.StripANSI(m.View()) }

func TestStartsInGeneralWithItsHistory(t *testing.T) {
	m := ready(80, 24)
	for _, want := range []string{"#general", "Morning! The release branch is cut", "mira", "#design       2", "#random       1"} {
		if !strings.Contains(text(m), want) {
			t.Errorf("screen lacks %q:\n%s", want, text(m))
		}
	}
}

func TestClickChannelSwitchesAndReadsIt(t *testing.T) {
	m := click(t, ready(80, 24), "chan-1")
	if m.cur != 1 || m.chans[1].unread != 0 {
		t.Fatalf("cur=%d unread=%d, want channel 1 read", m.cur, m.chans[1].unread)
	}
	if out := text(m); !strings.Contains(out, "New mock for the settings page") || strings.Contains(out, "#design       2") {
		t.Fatalf("channel not shown or still unread:\n%s", out)
	}
}

func TestCtrlNAndPWrap(t *testing.T) {
	m := ready(80, 24)
	m = send(m, tui.Key{Type: tui.KeyCtrl, Code: 'p'})
	if m.cur != 2 {
		t.Fatalf("Ctrl+P from the first channel went to %d, want 2", m.cur)
	}
	m = send(m, tui.Key{Type: tui.KeyCtrl, Code: 'n'})
	if m.cur != 0 {
		t.Fatalf("Ctrl+N from the last channel went to %d, want 0", m.cur)
	}
}

func TestSendAppendsAndAnswers(t *testing.T) {
	m := typed(ready(80, 24), "  hi there  ")
	m = send(m, tui.Key{Type: tui.KeyEnter})
	msgs := m.chans[0].msgs
	if len(msgs) != 6 || msgs[4].from != "you" || msgs[4].text != "hi there" {
		t.Fatalf("messages after sending: %+v", msgs)
	}
	if msgs[5].from != "mira" {
		t.Fatalf("no reply from mira: %+v", msgs[5])
	}
	if m.input.Value() != "" {
		t.Fatal("the input kept its text after Enter")
	}
}

func TestBlankMessageIsNotSent(t *testing.T) {
	m := typed(ready(80, 24), "   ")
	m = send(m, tui.Key{Type: tui.KeyEnter})
	if len(m.chans[0].msgs) != 4 {
		t.Fatal("a blank message was sent")
	}
}

func TestRepliesAreDeterministic(t *testing.T) {
	m := ready(80, 24)
	for i := 0; i < 2; i++ {
		m = typed(m, "ping")
		m = send(m, tui.Key{Type: tui.KeyEnter})
	}
	got := m.chans[0].msgs
	if got[5].text != m.chans[0].replies[0] || got[7].text != m.chans[0].replies[1] {
		t.Fatalf("replies %q, %q are not the first two in order", got[5].text, got[7].text)
	}
}

func TestReplyStreamsInWithATypingIndicator(t *testing.T) {
	m := send(newModel(false), tui.ResizeMsg{Width: 80, Height: 24})
	m = typed(m, "hello")
	m, cmd := update(m, tui.Key{Type: tui.KeyEnter})
	if cmd == nil || m.live == nil || m.live.started {
		t.Fatal("sending did not start a typing delay")
	}
	if out := text(m); !strings.Contains(out, "mira is typing…") {
		t.Fatalf("no typing indicator:\n%s", out)
	}
	m = send(m, typingMsg{m.live.seq})
	m = send(m, streamMsg{m.live.seq})
	if m.live == nil || m.live.shown != perTick {
		t.Fatalf("one tick should reveal %d characters, got %+v", perTick, m.live)
	}
	if out := text(m); !strings.Contains(out, "Nice▌") {
		t.Fatalf("no streaming cursor:\n%s", out)
	}
	for m.live != nil {
		m = send(m, streamMsg{m.live.seq})
	}
	if last := m.chans[0].msgs[len(m.chans[0].msgs)-1]; last.from != "mira" || last.text != m.chans[0].replies[0] {
		t.Fatalf("finished reply = %+v", last)
	}
}

func TestStaleTicksAreIgnored(t *testing.T) {
	m := send(newModel(false), tui.ResizeMsg{Width: 80, Height: 24})
	m = typed(m, "x")
	m = send(m, tui.Key{Type: tui.KeyEnter})
	seq := m.live.seq
	m = send(m, typingMsg{seq + 7})
	m = send(m, streamMsg{seq + 7})
	if m.live.started || m.live.shown != 0 {
		t.Fatalf("a tick from another reply moved this one: %+v", m.live)
	}
}

func TestReplyInAnotherChannelCountsAsUnread(t *testing.T) {
	m := send(newModel(false), tui.ResizeMsg{Width: 80, Height: 24})
	m = typed(m, "one")
	m = send(m, tui.Key{Type: tui.KeyEnter})
	m = send(m, tui.Key{Type: tui.KeyCtrl, Code: 'n'}) // leave while mira is typing
	m = send(m, typingMsg{m.live.seq})
	for m.live != nil {
		m = send(m, streamMsg{m.live.seq})
	}
	if m.chans[0].unread != 1 {
		t.Fatalf("#general unread = %d, want 1", m.chans[0].unread)
	}
}

func TestWheelScrollsAndJumpReturns(t *testing.T) {
	m := ready(80, 8) // five message rows, history is longer
	if m.maxOff() == 0 {
		t.Fatal("the sample history fits in five rows; the test needs it to scroll")
	}
	m = send(m, tui.MouseEvent{Button: tui.MouseButtonWheelUp, Action: tui.MouseActionPress})
	if m.off != 3 {
		t.Fatalf("off = %d after one wheel step, want 3", m.off)
	}
	if !strings.Contains(text(m), "newer messages") {
		t.Fatalf("no jump button while scrolled up:\n%s", text(m))
	}
	m = click(t, m, "jump")
	if m.off != 0 {
		t.Fatalf("off = %d after the jump button, want 0", m.off)
	}
	m = send(m, tui.MouseEvent{Button: tui.MouseButtonWheelDown, Action: tui.MouseActionPress})
	if m.off != 0 {
		t.Fatalf("scrolled below the newest message: off = %d", m.off)
	}
}

func TestScrollKeys(t *testing.T) {
	m := ready(80, 7) // four message rows, so a page is three
	m = send(m, tui.Key{Type: tui.KeyPgUp})
	if m.off != 3 {
		t.Fatalf("PgUp: off = %d, want 3", m.off)
	}
	m = send(m, tui.Key{Type: tui.KeyUp})
	m = send(m, tui.Key{Type: tui.KeyDown})
	m = send(m, tui.Key{Type: tui.KeyCtrl, Code: 'l'})
	if m.off != 0 {
		t.Fatalf("Ctrl+L left off = %d", m.off)
	}
}

func TestHelpOpensWithQuestionMarkOnlyWhenInputIsEmpty(t *testing.T) {
	m := typed(ready(80, 24), "?")
	if !m.help {
		t.Fatal("? on an empty input did not open help")
	}
	m = send(m, tui.Key{Type: tui.KeyEsc})
	if m.help {
		t.Fatal("Esc did not close help")
	}
	m = typed(m, "why")
	m = typed(m, "?")
	if m.help || m.input.Value() != "why?" {
		t.Fatalf("? inside a message opened help (input %q)", m.input.Value())
	}
}

func TestHelpButton(t *testing.T) {
	m := click(t, ready(80, 24), "help")
	if out := text(m); !strings.Contains(out, "Ctrl+N / Ctrl+P") {
		t.Fatalf("help not shown:\n%s", out)
	}
	m = click(t, m, "help")
	if m.help {
		t.Fatal("the help button did not toggle help off")
	}
}

func TestEscClearsTheInput(t *testing.T) {
	m := typed(ready(80, 24), "draft")
	m = send(m, tui.Key{Type: tui.KeyEsc})
	if m.input.Value() != "" {
		t.Fatal("Esc left the draft")
	}
}

func TestNarrowTerminalFoldsTheListAndClickingTheTitleChangesChannel(t *testing.T) {
	m := ready(40, 10)
	if strings.Contains(text(m), "#design") {
		t.Fatalf("the channel list shows at 40 columns:\n%s", text(m))
	}
	m = click(t, m, "next")
	if m.cur != 1 {
		t.Fatalf("cur = %d after clicking the title, want 1", m.cur)
	}
}

func TestHoverIsTracked(t *testing.T) {
	m := ready(80, 24)
	r, _ := layout.RectOf(m.screen(), m.size(), "chan-2")
	m = send(m, tui.MouseEvent{X: r.X + 1, Y: r.Y, Action: tui.MouseActionMotion})
	if m.hover != "chan-2" {
		t.Fatalf("hover = %q, want chan-2", m.hover)
	}
}

func TestClickInputFocusesIt(t *testing.T) {
	m := ready(80, 24)
	m.input.Blur()
	m = click(t, m, "input")
	if !m.input.Focused() {
		t.Fatal("clicking the input did not focus it")
	}
}

func TestLongWordsAndTinyTerminalsStayInside(t *testing.T) {
	m := typed(ready(40, 10), strings.Repeat("x", 300))
	m = send(m, tui.Key{Type: tui.KeyEnter})
	for _, size := range [][2]int{{40, 10}, {20, 6}, {10, 4}} {
		m = send(m, tui.ResizeMsg{Width: size[0], Height: size[1]})
		for i, line := range strings.Split(m.View(), "\n") {
			if w := ansi.Width(line); w > size[0] {
				t.Fatalf("%dx%d: line %d is %d wide: %q", size[0], size[1], i, w, line)
			}
		}
	}
}

func TestMessageLengthIsCapped(t *testing.T) {
	m := ready(80, 24)
	m.input.SetValue(strings.Repeat("a", 5000))
	m = send(m, tui.Key{Type: tui.KeyEnter})
	if got := len([]rune(m.chans[0].msgs[4].text)); got != maxLine {
		t.Fatalf("stored %d characters, want %d", got, maxLine)
	}
}

func TestQuit(t *testing.T) {
	for _, k := range []tui.Key{{Type: tui.KeyCtrlC}, {Type: tui.KeyCtrl, Code: 'q'}} {
		if _, cmd := update(ready(80, 24), k); !isQuit(cmd) {
			t.Errorf("%v did not quit", k)
		}
	}
	if _, cmd := update(typed(ready(80, 24), "q"), tui.Key{Type: tui.KeyRunes, Text: "q"}); isQuit(cmd) {
		t.Error("typing q quit the program")
	}
}

func TestRunsInAProgram(t *testing.T) {
	s := tuitest.New(newModel(false), 80, 24)
	defer s.Close()
	s.Keys("hi", "enter")
	if !s.WaitForText("mira is typing", 2*time.Second) {
		t.Fatalf("no typing indicator:\n%s", strings.Join(s.Screen(), "\n"))
	}
	if !s.WaitForText("pass that on to the rest of the team.", 5*time.Second) {
		t.Fatalf("reply never finished:\n%s", strings.Join(s.Screen(), "\n"))
	}
	s.ClickText("#design")
	if !s.WaitForText("New mock for the settings page", time.Second) {
		t.Fatal("clicking #design did not open it")
	}
	s.Keys("ctrl+q")
}
