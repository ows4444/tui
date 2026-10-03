package tui_test

import (
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/input"
	"github.com/ows4444/tui/textinput"
)

// seen is the input's value after the latest Update, for a test to poll.
type seen struct {
	mu    sync.Mutex
	value string
}

func (s *seen) get() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.value
}

// fieldModel is one focused text input.
type fieldModel struct {
	in   textinput.Model
	seen *seen
}

func newFieldModel() fieldModel {
	in := textinput.New()
	in.Focus()
	return fieldModel{in: in, seen: &seen{}}
}

func (m fieldModel) Init() tui.Cmd { return nil }
func (m fieldModel) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	var cmd tui.Cmd
	m.in, cmd = m.in.Update(msg)
	m.seen.mu.Lock()
	m.seen.value = m.in.Value()
	m.seen.mu.Unlock()
	return m, cmd
}
func (m fieldModel) View() string { return "value=[" + m.in.Value() + "]" }

// syncOut is a Program output a test can read while the Program writes.
type syncOut struct {
	mu sync.Mutex
	b  strings.Builder
}

func (s *syncOut) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncOut) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// session is one running Program with a text input.
type session struct {
	p    *tui.Program
	out  *syncOut
	seen *seen
	done chan tui.Model
	pw   *io.PipeWriter
}

func startSession(t *testing.T) *session {
	t.Helper()
	pr, pw := io.Pipe()
	m := newFieldModel()
	s := &session{out: &syncOut{}, seen: m.seen, done: make(chan tui.Model, 1), pw: pw}
	s.p = tui.NewProgram(m, tui.WithInput(pr), tui.WithOutput(s.out))
	go func() {
		m, _ := s.p.Run()
		s.done <- m
	}()
	t.Cleanup(func() { s.p.Quit(); pw.Close() })
	return s
}

func (s *session) keys(ks ...tui.Key) {
	for _, k := range ks {
		s.p.Send(k)
	}
}

// waitValue waits for the input to hold want.
func (s *session) waitValue(t *testing.T, want string) {
	t.Helper()
	for end := time.Now().Add(3 * time.Second); s.seen.get() != want; {
		if time.Now().After(end) {
			t.Fatalf("value = %q, want %q", s.seen.get(), want)
		}
		time.Sleep(time.Millisecond)
	}
}

// waitOutput waits for the Program to have written what.
func (s *session) waitOutput(t *testing.T, what string) {
	t.Helper()
	for end := time.Now().Add(3 * time.Second); !strings.Contains(s.out.String(), what); {
		if time.Now().After(end) {
			t.Fatalf("the Program never wrote %q", what)
		}
		time.Sleep(time.Millisecond)
	}
}

// value quits the Program and returns the input's final value.
func (s *session) value(t *testing.T) string {
	t.Helper()
	s.p.Quit()
	select {
	case m := <-s.done:
		return m.(fieldModel).in.Value()
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not return")
		return ""
	}
}

func typed(text string) []tui.Key {
	var ks []tui.Key
	for _, r := range text {
		ks = append(ks, tui.Key{Type: tui.KeyRunes, Text: string(r), Code: r})
	}
	return ks
}

var (
	selectToStart = tui.Key{Type: tui.KeyHome, Mod: input.ModShift}
	toEnd         = tui.Key{Type: tui.KeyEnd}
	copyKey       = tui.Key{Type: tui.KeyCtrlC}
	pasteKey      = tui.Key{Type: tui.KeyCtrl, Code: 'v'}
)

// captureStdout swaps os.Stdout for a pipe while f runs and returns what was
// written to it.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = old }()
	f()
	w.Close()
	b, _ := io.ReadAll(r)
	r.Close()
	return string(b)
}

// A copy in one Program goes to that Program's output, not to os.Stdout, and
// can be pasted back in the same Program but not in another one.
func TestCopyBufferBelongsToTheProgram(t *testing.T) {
	a, b := startSession(t), startSession(t)

	stdout := captureStdout(t, func() {
		a.keys(typed("hello")...)
		a.waitValue(t, "hello")
		a.keys(selectToStart, copyKey)
		a.waitOutput(t, ansi.OSC52Copy("hello")) // the copy reached A's own output
	})
	if stdout != "" {
		t.Errorf("the copy wrote %q to os.Stdout; the Program's output is elsewhere", stdout)
	}
	if strings.Contains(b.out.String(), "\x1b]52;") {
		t.Error("the copy in one Program wrote OSC 52 to another Program's output")
	}

	// The same Program pastes what it copied.
	a.keys(toEnd, pasteKey)
	a.waitValue(t, "hellohello")
	if got := a.value(t); got != "hellohello" {
		t.Errorf("paste in the copying Program: value = %q, want %q", got, "hellohello")
	}

	// Another Program does not.
	b.keys(typed("x")...)
	b.waitValue(t, "x")
	b.keys(pasteKey)
	time.Sleep(50 * time.Millisecond) // a paste, if one is coming, has arrived
	b.keys(typed("y")...)
	if got := b.value(t); got != "xy" {
		t.Errorf("paste in another Program: value = %q, want %q (nothing pasted)", got, "xy")
	}
}
