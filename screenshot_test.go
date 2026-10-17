package tui

import (
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ows4444/tui/ansi"
)

// shotKeys is a goroutine-safe list of the keys a model saw.
type shotKeys struct {
	mu   sync.Mutex
	keys []string
}

func (l *shotKeys) add(k string) { l.mu.Lock(); l.keys = append(l.keys, k); l.mu.Unlock() }
func (l *shotKeys) all() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.keys...)
}

// shotModel draws a red word and a plain one, and records its keys.
type shotModel struct{ keys *shotKeys }

func (shotModel) Init() Cmd { return nil }
func (m shotModel) Update(msg Msg) (Model, Cmd) {
	if k, ok := msg.(Key); ok {
		m.keys.add(k.String())
	}
	return m, nil
}
func (shotModel) View() string {
	return ansi.NewStyle().Foreground(ansi.Red).Render("hello") + " world  \n\nlast"
}

func startShot(t *testing.T, opts ...ProgramOption) (*Program, *io.PipeWriter, *shotKeys, func()) {
	t.Helper()
	pr, pw := io.Pipe()
	keys := &shotKeys{}
	opts = append([]ProgramOption{WithInput(pr), WithOutput(&unsyncBuffer{}), WithColorProfile(ansi.TrueColor)}, opts...)
	p := NewProgram(shotModel{keys: keys}, opts...)
	errc := make(chan error, 1)
	go func() { _, err := p.Run(); errc <- err }()
	p.Send(ResizeMsg{Width: 30, Height: 6})
	time.Sleep(100 * time.Millisecond)
	return p, pw, keys, func() {
		p.Quit()
		<-errc
		pw.Close()
	}
}

// askModel asks for a Screenshot when it sees "s" and keeps the answer.
type askModel struct{ got *atomic.Pointer[Screenshot] }

type gotShot struct{ s Screenshot }

func (askModel) Init() Cmd { return nil }
func (m askModel) Update(msg Msg) (Model, Cmd) {
	switch v := msg.(type) {
	case Key:
		if v.String() == "s" {
			return m, ScreenshotCmd(func(s Screenshot) Msg { return gotShot{s} })
		}
	case gotShot:
		m.got.Store(&v.s)
	}
	return m, nil
}
func (askModel) View() string {
	return ansi.NewStyle().Foreground(ansi.Red).Render("hello") + " world  \n\nlast"
}

func TestScreenshotCmdDeliversTheScreen(t *testing.T) {
	pr, pw := io.Pipe()
	defer pw.Close()
	var got atomic.Pointer[Screenshot]
	p := NewProgram(askModel{got: &got}, WithInput(pr), WithOutput(&unsyncBuffer{}), WithColorProfile(ansi.TrueColor))
	errc := make(chan error, 1)
	go func() { _, err := p.Run(); errc <- err }()
	p.Send(ResizeMsg{Width: 30, Height: 6})
	time.Sleep(100 * time.Millisecond)
	_, _ = io.WriteString(pw, "s")
	deadline := time.Now().Add(3 * time.Second)
	for got.Load() == nil && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	s := got.Load()
	if s == nil {
		t.Fatal("no Screenshot arrived")
	}
	if s.Text != "hello world\n\nlast" {
		t.Fatalf("Text %q", s.Text)
	}
	if !strings.Contains(s.ANSI, "\x1b[31m") || !strings.HasPrefix(ansi.StripANSI(s.ANSI), "hello world") {
		t.Fatalf("ANSI lost the colour: %q", s.ANSI)
	}
	if s.Width != 30 || s.Height != 6 {
		t.Fatalf("size %dx%d", s.Width, s.Height)
	}
	p.Quit()
	<-errc
}

func TestScreenshotCmdNilWrap(t *testing.T) {
	if ScreenshotCmd(nil) != nil {
		t.Fatal("a nil wrap gave a Cmd")
	}
}

// nilWrapModel's wrap returns nil, which must not reach Update.
type nilWrapModel struct{ updates *atomic.Int32 }

func (nilWrapModel) Init() Cmd { return nil }
func (m nilWrapModel) Update(msg Msg) (Model, Cmd) {
	m.updates.Add(1)
	if k, ok := msg.(Key); ok && k.String() == "s" {
		return m, ScreenshotCmd(func(Screenshot) Msg { return nil })
	}
	return m, nil
}
func (nilWrapModel) View() string { return "x" }

func TestScreenshotCmdNilAnswerIsDropped(t *testing.T) {
	pr, pw := io.Pipe()
	defer pw.Close()
	var n atomic.Int32
	p := NewProgram(nilWrapModel{updates: &n}, WithInput(pr), WithOutput(&unsyncBuffer{}))
	errc := make(chan error, 1)
	go func() { _, err := p.Run(); errc <- err }()
	p.Send(ResizeMsg{Width: 10, Height: 2})
	time.Sleep(100 * time.Millisecond)
	before := n.Load()
	_, _ = io.WriteString(pw, "s")
	time.Sleep(200 * time.Millisecond)
	if n.Load() != before+1 { // the key itself, and nothing for the nil answer
		t.Fatalf("Update ran %d times for one key", n.Load()-before)
	}
	p.Quit()
	<-errc
}

func TestScreenshotKey(t *testing.T) {
	var got atomic.Pointer[Screenshot]
	p, pw, keys, stop := startShot(t, WithScreenshot("ctrl+p", func(s Screenshot) { got.Store(&s) }))
	defer stop()
	_ = p
	_, _ = io.WriteString(pw, "\x10") // ctrl+p
	deadline := time.Now().Add(3 * time.Second)
	for got.Load() == nil && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	s := got.Load()
	if s == nil || s.Text != "hello world\n\nlast" {
		t.Fatalf("screenshot %+v", s)
	}
	_, _ = io.WriteString(pw, "x")
	time.Sleep(100 * time.Millisecond)
	if got := keys.all(); len(got) != 1 || got[0] != "x" {
		t.Fatalf("the screenshot key reached Update, or x did not: %v", got)
	}
}
